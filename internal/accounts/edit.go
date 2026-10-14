package accounts

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/QahNaaRlN/grace/internal/financialevents"
	"github.com/jackc/pgx/v5"
)

var (
	ErrNotFound         = errors.New("account not found")
	ErrInvalid          = errors.New("invalid account change")
	ErrRevisionConflict = errors.New("account revision conflict")
	ErrHasOperations    = errors.New("account type is fixed once it has operations")
	ErrStalePreview     = errors.New("reconciliation preview is stale")
)

type LastReconciliation struct {
	AsOf            time.Time `json:"asOf"`
	Outcome         string    `json:"outcome"`
	StatedMinor     int64     `json:"statedMinor"`
	DifferenceMinor int64     `json:"differenceMinor"`
	CreatedAt       time.Time `json:"createdAt"`
	// Stale: operations up to the reconciled moment changed since, so the
	// Grace balance at that moment no longer equals the confirmed one.
	Stale bool `json:"stale"`
}

// netAt is the sum of an account's live postings up to a moment; date-only
// operations of that local day count (SPEC-0040, decision 4).
const netAt = `COALESCE((SELECT SUM(CASE p.direction WHEN 'credit' THEN p.amount_minor ELSE -p.amount_minor END)
	FROM postings p JOIN financial_events e ON e.id = p.financial_event_id
	WHERE p.account_id = %s AND (e.occurred_at <= %[2]s OR e.occurred_date <= (%[2]s AT TIME ZONE 'Asia/Yekaterinburg')::date)), 0)`

var accountQuery = `
	SELECT a.id::text, a.name, a.type::text, a.currency,
		CASE WHEN a.balance_reconciled THEN a.opening_balance_minor + COALESCE((SELECT SUM(CASE p.direction WHEN 'credit' THEN p.amount_minor ELSE -p.amount_minor END) FROM postings p WHERE p.account_id = a.id), 0) END,
		a.balance_reconciled, a.created_at, a.revision,
		COALESCE((SELECT jsonb_agg(jsonb_build_object('id', i.id, 'name', i.name, 'bank', i.bank, 'last4', i.last4, 'disabledAt', i.disabled_at) ORDER BY i.disabled_at NULLS FIRST, i.created_at, i.id) FROM payment_instruments i WHERE i.account_id = a.id), '[]'::jsonb),
		EXISTS (SELECT 1 FROM postings p WHERE p.account_id = a.id) OR EXISTS (SELECT 1 FROM voided_postings v WHERE v.account_id = a.id),
		r.as_of, r.outcome, r.stated_minor, r.difference_minor, r.created_at,
		r.as_of IS NOT NULL AND r.stated_minor <> COALESCE(a.opening_balance_minor, 0) + ` + fmt.Sprintf(netAt, "a.id", "r.as_of") + `,
		(SELECT max(c.created_at) FROM import_confirmations c
			JOIN import_provenance pv ON pv.confirmation_id = c.id
			JOIN postings p ON p.financial_event_id = pv.event_id
			WHERE p.account_id = a.id)
	FROM accounts a
	LEFT JOIN LATERAL (SELECT * FROM account_reconciliations r WHERE r.account_id = a.id ORDER BY r.created_at DESC, r.id DESC LIMIT 1) r ON true`

func scanAccount(row pgx.Row) (Account, error) {
	var item Account
	var asOf, createdAt *time.Time
	var outcome *string
	var stated, difference *int64
	var stale bool
	if err := row.Scan(&item.ID, &item.Name, &item.Type, &item.Currency, &item.BalanceMinor, &item.BalanceReconciled, &item.CreatedAt, &item.Revision,
		&item.PaymentInstruments, &item.HasOperations, &asOf, &outcome, &stated, &difference, &createdAt, &stale, &item.LastImportConfirmedAt); err != nil {
		return Account{}, fmt.Errorf("scan account: %w", err)
	}
	if asOf != nil {
		item.LastReconciliation = &LastReconciliation{AsOf: *asOf, Outcome: *outcome, StatedMinor: *stated, DifferenceMinor: *difference, CreatedAt: *createdAt, Stale: stale}
	}
	return item, nil
}

func loadAccount(ctx context.Context, tx pgx.Tx, id string) (Account, error) {
	item, err := scanAccount(tx.QueryRow(ctx, accountQuery+` WHERE a.id::text = $1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return Account{}, ErrNotFound
	}
	return item, err
}

// Update is a change of account details (SPEC-0040 AC1).
type Update struct {
	Revision int     `json:"revision"`
	Name     *string `json:"name"`
	Type     *string `json:"type"`
}

func (repo *Repository) Update(ctx context.Context, id string, change Update) (Account, error) {
	tx, err := repo.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return Account{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	current, err := lockAccount(ctx, tx, id)
	if err != nil {
		return Account{}, err
	}
	if change.Revision != current.Revision {
		return Account{}, ErrRevisionConflict
	}
	name, kind := current.Name, current.Type
	if change.Name != nil {
		name = strings.TrimSpace(*change.Name)
		if name == "" || utf8.RuneCountInString(name) > 160 {
			return Account{}, ErrInvalid
		}
	}
	if change.Type != nil && *change.Type != current.Type {
		kind = *change.Type
		if !validTypes[kind] {
			return Account{}, ErrInvalid
		}
		// debit↔credit changes what a balance means for every past operation.
		if current.HasOperations {
			return Account{}, ErrHasOperations
		}
		if !cardsAllowed(kind) && activeCards(current) > 0 {
			return Account{}, ErrInvalid
		}
	}
	if name == current.Name && kind == current.Type {
		return current, nil
	}
	if _, err = tx.Exec(ctx, `UPDATE accounts SET name=$2, type=$3::account_type, revision=revision+1, updated_at=now() WHERE id=$1`, id, name, kind); err != nil {
		return Account{}, fmt.Errorf("update account: %w", err)
	}
	item, err := loadAccount(ctx, tx, id)
	if err != nil {
		return Account{}, err
	}
	return item, tx.Commit(ctx)
}

func lockAccount(ctx context.Context, tx pgx.Tx, id string) (Account, error) {
	if err := financialevents.LockOwner(ctx, tx); err != nil {
		return Account{}, err
	}
	var locked string
	if err := tx.QueryRow(ctx, `SELECT id::text FROM accounts WHERE id::text=$1 FOR UPDATE`, id).Scan(&locked); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Account{}, ErrNotFound
		}
		return Account{}, err
	}
	return loadAccount(ctx, tx, id)
}

func cardsAllowed(kind string) bool { return kind == "debit" || kind == "credit" }

func activeCards(account Account) int {
	count := 0
	for _, card := range account.PaymentInstruments {
		if card.DisabledAt == nil {
			count++
		}
	}
	return count
}

// CardChange adds or edits a payment instrument; Disabled retires a card while
// its past operations keep pointing at it.
type CardChange struct {
	Name     *string `json:"name"`
	Bank     *string `json:"bank"`
	Last4    *string `json:"last4"`
	Disabled *bool   `json:"disabled"`
}

func cleanCard(name, bank, last4 string) (string, string, string, bool) {
	name, bank, last4 = strings.TrimSpace(name), strings.TrimSpace(bank), strings.TrimSpace(last4)
	ok := name != "" && bank != "" && utf8.RuneCountInString(name) <= 160 && utf8.RuneCountInString(bank) <= 160 && lastFourPattern.MatchString(last4)
	return name, bank, last4, ok
}

func (repo *Repository) AddCard(ctx context.Context, accountID string, change CardChange) (Account, error) {
	tx, err := repo.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return Account{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	current, err := lockAccount(ctx, tx, accountID)
	if err != nil {
		return Account{}, err
	}
	name, bank, last4, ok := cleanCard(deref(change.Name), deref(change.Bank), deref(change.Last4))
	if !ok || !cardsAllowed(current.Type) {
		return Account{}, ErrInvalid
	}
	if _, err = tx.Exec(ctx, `INSERT INTO payment_instruments (account_id, name, bank, last4) VALUES ($1, $2, $3, $4)`, accountID, name, bank, last4); err != nil {
		return Account{}, fmt.Errorf("add card: %w", err)
	}
	item, err := loadAccount(ctx, tx, accountID)
	if err != nil {
		return Account{}, err
	}
	return item, tx.Commit(ctx)
}

func (repo *Repository) UpdateCard(ctx context.Context, accountID, cardID string, change CardChange) (Account, error) {
	tx, err := repo.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return Account{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	current, err := lockAccount(ctx, tx, accountID)
	if err != nil {
		return Account{}, err
	}
	var card *PaymentInstrument
	for i := range current.PaymentInstruments {
		if current.PaymentInstruments[i].ID == cardID {
			card = &current.PaymentInstruments[i]
		}
	}
	if card == nil {
		return Account{}, ErrNotFound
	}
	name, bank, last4 := card.Name, card.Bank, card.Last4
	if change.Name != nil {
		name = *change.Name
	}
	if change.Bank != nil {
		bank = *change.Bank
	}
	if change.Last4 != nil {
		last4 = *change.Last4
	}
	name, bank, last4, ok := cleanCard(name, bank, last4)
	if !ok {
		return Account{}, ErrInvalid
	}
	disabled := card.DisabledAt != nil
	if change.Disabled != nil {
		disabled = *change.Disabled
	}
	if !disabled && !cardsAllowed(current.Type) {
		return Account{}, ErrInvalid
	}
	if _, err = tx.Exec(ctx, `UPDATE payment_instruments SET name=$3, bank=$4, last4=$5,
		disabled_at = CASE WHEN $6 THEN COALESCE(disabled_at, now()) END
		WHERE id::text=$1 AND account_id::text=$2`, cardID, accountID, name, bank, last4, disabled); err != nil {
		return Account{}, fmt.Errorf("update card: %w", err)
	}
	item, err := loadAccount(ctx, tx, accountID)
	if err != nil {
		return Account{}, err
	}
	return item, tx.Commit(ctx)
}

func deref(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
