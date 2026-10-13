package financialevents

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// SPEC-0041: in-place corrections with an append-only history, and voiding
// that moves postings to voided_postings so no balance or report counts them.

var (
	ErrNotFound         = errors.New("financial event not found")
	ErrRevisionConflict = errors.New("financial event changed")
	ErrVoided           = errors.New("financial event is voided")
	ErrLinked           = errors.New("financial event is linked")
)

// LinkError names what prevents changing money or voiding.
type LinkError struct{ Links []string }

func (e *LinkError) Error() string { return "linked: " + strings.Join(e.Links, ", ") }
func (e *LinkError) Unwrap() error { return ErrLinked }

type Correction struct {
	Revision             int
	IdempotencyKey       string
	Merchant             *string
	Description          *string
	CategoryID           *string
	ClearCategory        bool
	OccurredAt           *time.Time
	OccurredDate         *string
	Kind                 *string
	AmountMinor          *int64
	AccountID            *string
	DestinationAccountID *string
}

type Change struct {
	Field  string `json:"field"`
	Before any    `json:"before"`
	After  any    `json:"after"`
}

type Revision struct {
	Revision  int       `json:"revision"`
	Changes   []Change  `json:"changes"`
	CreatedAt time.Time `json:"createdAt"`
}

type livePosting struct {
	id        string
	accountID string
	direction string
	amount    int64
	currency  string
}

type eventState struct {
	kind         string
	categoryID   *string
	merchant     string
	description  string
	occurredAt   *time.Time
	occurredDate *string
	revision     int
	voided       bool
	postings     []livePosting
}

// replayed reports an Idempotency-Key already used: for this event the
// earlier result stands; for another event it is a conflict.
func replayed(ctx context.Context, tx pgx.Tx, eventID, key string) (bool, error) {
	var owner string
	err := tx.QueryRow(ctx, `SELECT event_id::text FROM financial_event_revisions WHERE idempotency_key=$1`, key).Scan(&owner)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if owner != eventID {
		return false, fmt.Errorf("%w: Idempotency-Key used for another operation", ErrRevisionConflict)
	}
	return true, nil
}

func lockState(ctx context.Context, tx pgx.Tx, id string) (eventState, error) {
	var state eventState
	err := tx.QueryRow(ctx, `SELECT kind::text, category_id::text, merchant, description, occurred_at, occurred_date::text, revision, voided_at IS NOT NULL
		FROM financial_events WHERE id=$1 FOR UPDATE`, id).Scan(&state.kind, &state.categoryID, &state.merchant, &state.description,
		&state.occurredAt, &state.occurredDate, &state.revision, &state.voided)
	var dbErr *pgconn.PgError
	if errors.Is(err, pgx.ErrNoRows) || (errors.As(err, &dbErr) && dbErr.Code == "22P02") {
		return state, ErrNotFound
	}
	if err != nil {
		return state, err
	}
	rows, err := tx.Query(ctx, `SELECT id::text, account_id::text, direction::text, amount_minor, currency FROM postings
		WHERE financial_event_id=$1 ORDER BY direction DESC, created_at, id FOR UPDATE`, id)
	if err != nil {
		return state, err
	}
	defer rows.Close()
	for rows.Next() {
		var p livePosting
		if err := rows.Scan(&p.id, &p.accountID, &p.direction, &p.amount, &p.currency); err != nil {
			return state, err
		}
		state.postings = append(state.postings, p)
	}
	return state, rows.Err()
}

// links lists records whose meaning depends on this operation's money.
func links(ctx context.Context, tx pgx.Tx, id, kind string) ([]string, error) {
	var found []string
	checks := []struct{ label, query string }{
		{"чек", `SELECT EXISTS (SELECT 1 FROM receipts WHERE purchase_event_id=$1)`},
		{"оплата графика обязательства", `SELECT EXISTS (SELECT 1 FROM scheduled_payment_settlements WHERE financial_event_id=$1)`},
		{"подписка", `SELECT EXISTS (SELECT 1 FROM subscription_expenses WHERE financial_event_id=$1)`},
		{"возврат по этой покупке", `SELECT EXISTS (SELECT 1 FROM financial_events WHERE original_event_id=$1 AND voided_at IS NULL)`},
	}
	for _, check := range checks {
		var exists bool
		if err := tx.QueryRow(ctx, check.query, id).Scan(&exists); err != nil {
			return nil, err
		}
		if exists {
			found = append(found, check.label)
		}
	}
	if kind == "refund" {
		found = append(found, "исходная покупка возврата")
	}
	return found, nil
}

func (repo *Repository) Correct(ctx context.Context, id string, c Correction) (Event, error) {
	tx, err := repo.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return Event{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := LockOwner(ctx, tx); err != nil {
		return Event{}, err
	}
	state, err := lockState(ctx, tx, id)
	if err != nil {
		return Event{}, err
	}
	if done, err := replayed(ctx, tx, id, c.IdempotencyKey); err != nil || done {
		if err != nil {
			return Event{}, err
		}
		return loadEvent(ctx, tx, id)
	}
	if state.voided {
		return Event{}, ErrVoided
	}
	if c.Revision != state.revision {
		return Event{}, ErrRevisionConflict
	}
	changes, money, err := applyCorrection(ctx, tx, id, &state, c)
	if err != nil {
		return Event{}, err
	}
	if len(changes) == 0 {
		return Event{}, fmt.Errorf("%w: no changes", ErrInvalid)
	}
	if money {
		found, err := links(ctx, tx, id, state.kind)
		if err != nil {
			return Event{}, err
		}
		if len(found) > 0 {
			return Event{}, &LinkError{Links: found}
		}
	}
	payload, _ := json.Marshal(changes)
	if _, err := tx.Exec(ctx, `UPDATE financial_events SET revision=revision+1, edited_by_user=true WHERE id=$1`, id); err != nil {
		return Event{}, mapCorrectionError(err)
	}
	if _, err := tx.Exec(ctx, `INSERT INTO financial_event_revisions(owner_id,event_id,revision,changes,idempotency_key)
		VALUES(grace_owner_id(),$1,$2,$3,$4)`, id, state.revision+1, payload, c.IdempotencyKey); err != nil {
		return Event{}, mapCorrectionError(err)
	}
	event, err := loadEvent(ctx, tx, id)
	if err != nil {
		return Event{}, err
	}
	return event, tx.Commit(ctx)
}

// applyCorrection writes the requested fields and returns what changed and
// whether money (amount, account, kind) changed.
func applyCorrection(ctx context.Context, tx pgx.Tx, id string, state *eventState, c Correction) ([]Change, bool, error) {
	var changes []Change
	money := false
	kindChanged := false
	set := func(column string, value any) error {
		_, err := tx.Exec(ctx, `UPDATE financial_events SET `+column+`=$2 WHERE id=$1`, id, value)
		return mapCorrectionError(err)
	}
	if c.Merchant != nil && strings.TrimSpace(*c.Merchant) != state.merchant {
		value := strings.TrimSpace(*c.Merchant)
		if utf8.RuneCountInString(value) > 200 {
			return nil, false, fmt.Errorf("%w: merchant is too long", ErrInvalid)
		}
		changes = append(changes, Change{"merchant", state.merchant, value})
		if err := set("merchant", value); err != nil {
			return nil, false, err
		}
		state.merchant = value
	}
	if c.Description != nil && strings.TrimSpace(*c.Description) != state.description {
		value := strings.TrimSpace(*c.Description)
		if utf8.RuneCountInString(value) > 500 {
			return nil, false, fmt.Errorf("%w: description is too long", ErrInvalid)
		}
		changes = append(changes, Change{"description", state.description, value})
		if err := set("description", value); err != nil {
			return nil, false, err
		}
		state.description = value
	}
	if c.OccurredAt != nil || c.OccurredDate != nil {
		if c.OccurredAt != nil && c.OccurredDate != nil {
			return nil, false, fmt.Errorf("%w: either a moment or a date", ErrInvalid)
		}
		if c.OccurredDate != nil && !ValidCalendarDate(*c.OccurredDate) {
			return nil, false, fmt.Errorf("%w: invalid date", ErrInvalid)
		}
		before := dateValue(state.occurredAt, state.occurredDate)
		var at *time.Time
		var date *string
		if c.OccurredAt != nil {
			value := c.OccurredAt.UTC()
			at = &value
		} else {
			date = c.OccurredDate
		}
		if after := dateValue(at, date); after != before {
			changes = append(changes, Change{"occurred", before, after})
			if _, err := tx.Exec(ctx, `UPDATE financial_events SET occurred_at=$2, occurred_date=$3::date WHERE id=$1`, id, at, date); err != nil {
				return nil, false, mapCorrectionError(err)
			}
		}
	}
	if c.Kind != nil && *c.Kind != state.kind {
		convertible := []string{"purchase", "income_received", "debt_payment"}
		if !slices.Contains(convertible, state.kind) || !slices.Contains(convertible, *c.Kind) || len(state.postings) != 1 {
			return nil, false, fmt.Errorf("%w: only purchase, income and obligation payment convert between each other", ErrInvalid)
		}
		direction := "debit"
		if *c.Kind == "income_received" {
			direction = "credit"
		}
		changes = append(changes, Change{"kind", state.kind, *c.Kind})
		money = true
		if _, err := tx.Exec(ctx, `UPDATE financial_events SET kind=$2::financial_event_kind WHERE id=$1`, id, *c.Kind); err != nil {
			return nil, false, mapCorrectionError(err)
		}
		if _, err := tx.Exec(ctx, `UPDATE postings SET direction=$2::posting_direction WHERE id=$1`, state.postings[0].id, direction); err != nil {
			return nil, false, mapCorrectionError(err)
		}
		state.kind = *c.Kind
		kindChanged = true
		if state.kind != "purchase" && c.CategoryID == nil {
			c.ClearCategory = true
		}
	}
	// A purchase needs a category when it becomes one or its category changes;
	// imported uncategorized purchases may still be edited otherwise.
	if c.CategoryID != nil || c.ClearCategory || (kindChanged && state.kind == "purchase") {
		var next *string
		if !c.ClearCategory {
			next = c.CategoryID
		}
		if err := validateCategory(ctx, tx, next, state.kind, false); err != nil {
			return nil, false, err
		}
		if fmt.Sprint(deref(state.categoryID)) != fmt.Sprint(deref(next)) {
			changes = append(changes, Change{"category", state.categoryID, next})
			if err := set("category_id", next); err != nil {
				return nil, false, err
			}
		}
	}
	if c.AmountMinor != nil && *c.AmountMinor != state.postings[0].amount {
		if *c.AmountMinor <= 0 {
			return nil, false, fmt.Errorf("%w: amount must be positive", ErrInvalid)
		}
		changes = append(changes, Change{"amountMinor", state.postings[0].amount, *c.AmountMinor})
		money = true
		if _, err := tx.Exec(ctx, `UPDATE postings SET amount_minor=$2 WHERE financial_event_id=$1`, id, *c.AmountMinor); err != nil {
			return nil, false, mapCorrectionError(err)
		}
	}
	move := func(posting livePosting, account *string, field string) error {
		if account == nil || *account == posting.accountID {
			return nil
		}
		var currency string
		err := tx.QueryRow(ctx, `SELECT currency FROM accounts WHERE id=$1`, *account).Scan(&currency)
		if errors.Is(err, pgx.ErrNoRows) || (err == nil && currency != posting.currency) {
			return fmt.Errorf("%w: account must exist and use %s", ErrInvalid, posting.currency)
		}
		if err != nil {
			return mapCorrectionError(err)
		}
		changes = append(changes, Change{field, posting.accountID, *account})
		money = true
		_, err = tx.Exec(ctx, `UPDATE postings SET account_id=$2 WHERE id=$1`, posting.id, *account)
		return mapCorrectionError(err)
	}
	if state.kind == "internal_transfer" {
		var debit, credit livePosting
		for _, p := range state.postings {
			if p.direction == "debit" {
				debit = p
			} else {
				credit = p
			}
		}
		source, destination := debit.accountID, credit.accountID
		if c.AccountID != nil {
			source = *c.AccountID
		}
		if c.DestinationAccountID != nil {
			destination = *c.DestinationAccountID
		}
		if source == destination {
			return nil, false, fmt.Errorf("%w: transfer accounts must differ", ErrInvalid)
		}
		if err := move(debit, c.AccountID, "account"); err != nil {
			return nil, false, err
		}
		if err := move(credit, c.DestinationAccountID, "destinationAccount"); err != nil {
			return nil, false, err
		}
	} else {
		if c.DestinationAccountID != nil {
			return nil, false, fmt.Errorf("%w: only a transfer has a destination account", ErrInvalid)
		}
		if err := move(state.postings[0], c.AccountID, "account"); err != nil {
			return nil, false, err
		}
	}
	return changes, money, nil
}

func (repo *Repository) Void(ctx context.Context, id string, revision int, reason, key string) (Event, error) {
	reason = strings.TrimSpace(reason)
	if n := utf8.RuneCountInString(reason); n < 1 || n > 500 {
		return Event{}, fmt.Errorf("%w: reason is required", ErrInvalid)
	}
	tx, err := repo.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return Event{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := LockOwner(ctx, tx); err != nil {
		return Event{}, err
	}
	state, err := lockState(ctx, tx, id)
	if err != nil {
		return Event{}, err
	}
	if done, err := replayed(ctx, tx, id, key); err != nil || done {
		if err != nil {
			return Event{}, err
		}
		return loadEvent(ctx, tx, id)
	}
	if state.voided {
		return Event{}, ErrVoided
	}
	if revision != state.revision {
		return Event{}, ErrRevisionConflict
	}
	found, err := links(ctx, tx, id, "")
	if err != nil {
		return Event{}, err
	}
	if len(found) > 0 {
		return Event{}, &LinkError{Links: found}
	}
	payload, _ := json.Marshal([]Change{{"voided", nil, reason}})
	statements := []struct {
		query string
		args  []any
	}{
		{`INSERT INTO financial_event_revisions(owner_id,event_id,revision,changes,idempotency_key) VALUES(grace_owner_id(),$1,$2,$3,$4)`, []any{id, state.revision + 1, payload, key}},
		{`UPDATE financial_events SET revision=revision+1, voided_at=now(), void_reason=$2 WHERE id=$1`, []any{id, reason}},
		{`INSERT INTO voided_postings(id,owner_id,financial_event_id,account_id,direction,amount_minor,currency,created_at)
			SELECT id,owner_id,financial_event_id,account_id,direction,amount_minor,currency,created_at FROM postings WHERE financial_event_id=$1`, []any{id}},
		{`DELETE FROM postings WHERE financial_event_id=$1`, []any{id}},
	}
	for _, statement := range statements {
		if _, err := tx.Exec(ctx, statement.query, statement.args...); err != nil {
			return Event{}, mapCorrectionError(err)
		}
	}
	event, err := loadEvent(ctx, tx, id)
	if err != nil {
		return Event{}, err
	}
	return event, tx.Commit(ctx)
}

func (repo *Repository) History(ctx context.Context, id string) ([]Revision, error) {
	rows, err := repo.pool.Query(ctx, `SELECT revision, changes, created_at FROM financial_event_revisions
		WHERE event_id::text=$1 ORDER BY revision DESC`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]Revision, 0)
	for rows.Next() {
		var item Revision
		var payload []byte
		if err := rows.Scan(&item.Revision, &payload, &item.CreatedAt); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(payload, &item.Changes); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func dateValue(at *time.Time, date *string) string {
	if at != nil {
		return at.UTC().Format(time.RFC3339)
	}
	if date != nil {
		return *date
	}
	return ""
}

func deref(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func mapCorrectionError(err error) error {
	var dbErr *pgconn.PgError
	if errors.As(err, &dbErr) {
		switch dbErr.Code {
		case "23514", "23503", "22P02", "23502":
			return fmt.Errorf("%w: %s", ErrInvalid, dbErr.Message)
		case "23505":
			return fmt.Errorf("%w: Idempotency-Key already used", ErrRevisionConflict)
		}
	}
	return err
}
