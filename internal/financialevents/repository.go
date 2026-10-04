package financialevents

import (
	"context"
	"errors"
	"fmt"
	"sort"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrConflict = errors.New("idempotency key conflicts with another payload")
	ErrInvalid  = errors.New("financial event is invalid")
)

type Repository struct{ pool *pgxpool.Pool }

func NewRepository(pool *pgxpool.Pool) *Repository { return &Repository{pool: pool} }

type queryer interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
}

func (repo *Repository) List(ctx context.Context, limit int) ([]Event, error) {
	rows, err := repo.pool.Query(ctx, eventQuery+`
		WHERE e.id IN (
			SELECT id
			FROM financial_events
			ORDER BY occurred_at DESC, created_at DESC
			LIMIT $1
		)
		ORDER BY e.occurred_at DESC, e.created_at DESC, p.created_at, p.id`, limit)
	if err != nil {
		return nil, fmt.Errorf("list financial events: %w", err)
	}
	defer rows.Close()
	return collectEvents(rows)
}

func (repo *Repository) Create(ctx context.Context, command Command) (Event, bool, error) {
	tx, err := repo.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return Event{}, false, fmt.Errorf("begin financial event: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	lockKey := command.Kind + ":" + command.IdempotencyKey
	if _, err := tx.Exec(ctx, "SELECT pg_advisory_xact_lock(hashtextextended($1, 0))", lockKey); err != nil {
		return Event{}, false, fmt.Errorf("lock idempotency key: %w", err)
	}

	var existingID, existingHash string
	err = tx.QueryRow(ctx, `
		SELECT id::text, request_hash
		FROM financial_events
		WHERE kind = $1 AND idempotency_key = $2`,
		command.Kind, command.IdempotencyKey,
	).Scan(&existingID, &existingHash)
	if err == nil {
		if existingHash != command.RequestHash {
			return Event{}, false, ErrConflict
		}
		event, err := loadEvent(ctx, tx, existingID)
		return event, false, err
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return Event{}, false, fmt.Errorf("find idempotent event: %w", err)
	}

	categoryID, originalEventID, postings, err := resolveCommand(ctx, tx, command)
	if err != nil {
		return Event{}, false, err
	}

	var eventID string
	err = tx.QueryRow(ctx, `
		INSERT INTO financial_events (
			kind, source, category_id, original_event_id, merchant, description,
			occurred_at, idempotency_key, request_hash
		) VALUES ($1, 'manual', $2, $3, $4, $5, $6, $7, $8)
		RETURNING id::text`,
		command.Kind, categoryID, originalEventID, command.Merchant,
		command.Description, command.OccurredAt, command.IdempotencyKey,
		command.RequestHash,
	).Scan(&eventID)
	if err != nil {
		return Event{}, false, fmt.Errorf("insert financial event: %w", err)
	}

	for _, posting := range postings {
		_, err := tx.Exec(ctx, `
			INSERT INTO postings (
				financial_event_id, account_id, direction, amount_minor, currency
			) VALUES ($1, $2, $3, $4, $5)`,
			eventID, posting.AccountID, posting.Direction, posting.Amount, posting.Currency,
		)
		if err != nil {
			return Event{}, false, fmt.Errorf("insert posting: %w", err)
		}
	}

	event, err := loadEvent(ctx, tx, eventID)
	if err != nil {
		return Event{}, false, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Event{}, false, fmt.Errorf("commit financial event: %w", err)
	}
	return event, true, nil
}

type account struct {
	ID       string
	Name     string
	Currency string
}

func loadAccountsForUpdate(ctx context.Context, tx pgx.Tx, ids ...string) (map[string]account, error) {
	ordered := append([]string(nil), ids...)
	sort.Strings(ordered)
	result := make(map[string]account, len(ordered))
	for _, id := range ordered {
		if _, exists := result[id]; exists {
			continue
		}
		var item account
		err := tx.QueryRow(ctx, `
			SELECT id::text, name, currency
			FROM accounts WHERE id = $1 FOR UPDATE`, id,
		).Scan(&item.ID, &item.Name, &item.Currency)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%w: account %s does not exist", ErrInvalid, id)
		}
		if err != nil {
			return nil, fmt.Errorf("lock account: %w", err)
		}
		result[id] = item
	}
	return result, nil
}

func resolveCommand(ctx context.Context, tx pgx.Tx, command Command) (*string, *string, []Posting, error) {
	switch command.Kind {
	case "purchase", "income_received", "debt_payment":
		accounts, err := loadAccountsForUpdate(ctx, tx, command.AccountID)
		if err != nil {
			return nil, nil, nil, err
		}
		account := accounts[command.AccountID]
		if err := validateCategory(ctx, tx, command.CategoryID, command.Kind); err != nil {
			return nil, nil, nil, err
		}
		direction := "debit"
		if command.Kind == "income_received" {
			direction = "credit"
		}
		return command.CategoryID, nil, []Posting{{
			AccountID: account.ID, Account: account.Name, Direction: direction,
			Amount: command.AmountMinor, Currency: account.Currency,
		}}, nil

	case "internal_transfer":
		if command.AccountID == command.DestinationAccountID {
			return nil, nil, nil, fmt.Errorf("%w: transfer accounts must differ", ErrInvalid)
		}
		accounts, err := loadAccountsForUpdate(ctx, tx, command.AccountID, command.DestinationAccountID)
		if err != nil {
			return nil, nil, nil, err
		}
		source := accounts[command.AccountID]
		destination := accounts[command.DestinationAccountID]
		if source.Currency != destination.Currency {
			return nil, nil, nil, fmt.Errorf("%w: transfer currencies must match", ErrInvalid)
		}
		return nil, nil, []Posting{
			{AccountID: source.ID, Account: source.Name, Direction: "debit", Amount: command.AmountMinor, Currency: source.Currency},
			{AccountID: destination.ID, Account: destination.Name, Direction: "credit", Amount: command.AmountMinor, Currency: destination.Currency},
		}, nil

	case "refund":
		var categoryID *string
		var accountID, accountName, currency string
		var purchaseAmount int64
		err := tx.QueryRow(ctx, `
			SELECT e.category_id::text, p.account_id::text, a.name, p.currency, p.amount_minor
			FROM financial_events e
			JOIN postings p ON p.financial_event_id = e.id
			JOIN accounts a ON a.id = p.account_id
			WHERE e.id = $1 AND e.kind = 'purchase'
			FOR UPDATE OF e, a`, command.OriginalEventID,
		).Scan(&categoryID, &accountID, &accountName, &currency, &purchaseAmount)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil, nil, fmt.Errorf("%w: original purchase does not exist", ErrInvalid)
		}
		if err != nil {
			return nil, nil, nil, fmt.Errorf("load original purchase: %w", err)
		}
		var refunded int64
		if err := tx.QueryRow(ctx, `
			SELECT COALESCE(SUM(p.amount_minor), 0)
			FROM financial_events e
			JOIN postings p ON p.financial_event_id = e.id
			WHERE e.original_event_id = $1 AND e.kind = 'refund'`,
			command.OriginalEventID,
		).Scan(&refunded); err != nil {
			return nil, nil, nil, fmt.Errorf("sum refunds: %w", err)
		}
		if refunded+command.AmountMinor > purchaseAmount {
			return nil, nil, nil, fmt.Errorf("%w: refund exceeds purchase amount", ErrInvalid)
		}
		originalID := command.OriginalEventID
		return categoryID, &originalID, []Posting{{
			AccountID: accountID, Account: accountName, Direction: "credit",
			Amount: command.AmountMinor, Currency: currency,
		}}, nil
	default:
		return nil, nil, nil, fmt.Errorf("%w: unsupported financial event kind", ErrInvalid)
	}
}

func validateCategory(ctx context.Context, tx pgx.Tx, categoryID *string, kind string) error {
	if categoryID == nil {
		if kind == "purchase" {
			return fmt.Errorf("%w: purchase requires an expense category", ErrInvalid)
		}
		return nil
	}
	var categoryKind string
	if err := tx.QueryRow(ctx, "SELECT kind::text FROM categories WHERE id = $1", *categoryID).Scan(&categoryKind); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("%w: category does not exist", ErrInvalid)
		}
		return fmt.Errorf("load category: %w", err)
	}
	expected := "expense"
	if kind == "income_received" {
		expected = "income"
	}
	if categoryKind != expected {
		return fmt.Errorf("%w: category kind must be %s", ErrInvalid, expected)
	}
	return nil
}

const eventQuery = `
	SELECT
		e.id::text,
		e.kind::text,
		e.source::text,
		e.category_id::text,
		c.name,
		e.original_event_id::text,
		e.merchant,
		e.description,
		e.occurred_at,
		e.created_at,
		p.id::text,
		p.account_id::text,
		a.name,
		p.direction::text,
		p.amount_minor,
		p.currency
	FROM financial_events e
	LEFT JOIN categories c ON c.id = e.category_id
	JOIN postings p ON p.financial_event_id = e.id
	JOIN accounts a ON a.id = p.account_id
	`

func loadEvent(ctx context.Context, db queryer, eventID string) (Event, error) {
	rows, err := db.Query(ctx, eventQuery+" WHERE e.id = $1 ORDER BY p.created_at, p.id", eventID)
	if err != nil {
		return Event{}, fmt.Errorf("load financial event: %w", err)
	}
	defer rows.Close()
	events, err := collectEvents(rows)
	if err != nil {
		return Event{}, err
	}
	if len(events) != 1 {
		return Event{}, fmt.Errorf("load financial event: %w", pgx.ErrNoRows)
	}
	return events[0], nil
}

func collectEvents(rows pgx.Rows) ([]Event, error) {
	items := make([]Event, 0)
	index := make(map[string]int)
	for rows.Next() {
		var event Event
		var posting Posting
		if err := rows.Scan(
			&event.ID, &event.Kind, &event.Source, &event.CategoryID, &event.CategoryName,
			&event.OriginalEventID, &event.Merchant, &event.Description,
			&event.OccurredAt, &event.CreatedAt,
			&posting.ID, &posting.AccountID, &posting.Account, &posting.Direction,
			&posting.Amount, &posting.Currency,
		); err != nil {
			return nil, fmt.Errorf("scan financial event: %w", err)
		}
		position, exists := index[event.ID]
		if !exists {
			event.Postings = make([]Posting, 0, 2)
			items = append(items, event)
			position = len(items) - 1
			index[event.ID] = position
		}
		items[position].Postings = append(items[position].Postings, posting)
	}
	return items, rows.Err()
}
