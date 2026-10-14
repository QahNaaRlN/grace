package accounts

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

const maxStatedMinor = 1_000_000_000_000_000

// ReconciliationPreview is what Grace computes for a stated bank balance; it
// writes nothing (SPEC-0040).
type ReconciliationPreview struct {
	AsOf        time.Time `json:"asOf"`
	StatedMinor int64     `json:"statedMinor"`
	Currency    string    `json:"currency"`
	// ComputedMinor is the Grace balance at AsOf; for an unreconciled account
	// it is only the net of its operations, because the opening is unknown.
	ComputedMinor        int64  `json:"computedMinor"`
	DifferenceMinor      int64  `json:"differenceMinor"`
	Outcome              string `json:"outcome"`
	ProposedOpeningMinor *int64 `json:"proposedOpeningMinor"`
	PreviewHash          string `json:"previewHash"`
}

type Reconciliation struct {
	ID                string    `json:"id"`
	AccountID         string    `json:"accountId"`
	AsOf              time.Time `json:"asOf"`
	StatedMinor       int64     `json:"statedMinor"`
	ComputedMinor     int64     `json:"computedMinor"`
	DifferenceMinor   int64     `json:"differenceMinor"`
	Outcome           string    `json:"outcome"`
	AdjustmentEventID *string   `json:"adjustmentEventId"`
	CreatedAt         time.Time `json:"createdAt"`
}

type ReconcileRequest struct {
	AsOf        time.Time `json:"asOf"`
	StatedMinor int64     `json:"statedMinor"`
	PreviewHash string    `json:"previewHash"`
}

func validMoment(asOf time.Time, stated int64, now time.Time) bool {
	return !asOf.IsZero() && !asOf.After(now.Add(time.Minute)) && stated > -maxStatedMinor && stated < maxStatedMinor
}

func preview(ctx context.Context, tx pgx.Tx, account Account, asOf time.Time, stated int64) (ReconciliationPreview, error) {
	var opening *int64
	var net int64
	err := tx.QueryRow(ctx, `SELECT a.opening_balance_minor, `+fmt.Sprintf(netAt, "a.id", "$2::timestamptz")+` FROM accounts a WHERE a.id::text=$1`, account.ID, asOf).Scan(&opening, &net)
	if err != nil {
		return ReconciliationPreview{}, err
	}
	result := ReconciliationPreview{AsOf: asOf, StatedMinor: stated, Currency: account.Currency, ComputedMinor: net}
	if account.BalanceReconciled && opening != nil {
		result.ComputedMinor = *opening + net
	}
	result.DifferenceMinor = stated - result.ComputedMinor
	switch {
	case !account.BalanceReconciled:
		result.Outcome = "opening_set"
		value := stated - net
		result.ProposedOpeningMinor = &value
	case result.DifferenceMinor == 0:
		result.Outcome = "matched"
	default:
		result.Outcome = "adjusted"
	}
	sum := sha256.Sum256(fmt.Appendf(nil, "%s|%d|%d|%d|%d|%s", account.ID, asOf.UnixNano(), stated, result.ComputedMinor, account.Revision, result.Outcome))
	result.PreviewHash = hex.EncodeToString(sum[:])
	return result, nil
}

func (repo *Repository) PreviewReconciliation(ctx context.Context, accountID string, asOf time.Time, stated int64) (ReconciliationPreview, error) {
	if !validMoment(asOf, stated, time.Now()) {
		return ReconciliationPreview{}, ErrInvalid
	}
	tx, err := repo.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return ReconciliationPreview{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	account, err := loadAccount(ctx, tx, accountID)
	if err != nil {
		return ReconciliationPreview{}, err
	}
	return preview(ctx, tx, account, asOf, stated)
}

// Reconcile records a confirmed reconciliation. The preview hash proves the
// person saw exactly this computed balance; any change in between is stale.
func (repo *Repository) Reconcile(ctx context.Context, accountID, key string, request ReconcileRequest) (Reconciliation, Account, error) {
	if !validMoment(request.AsOf, request.StatedMinor, time.Now()) || key == "" || len(key) > 160 {
		return Reconciliation{}, Account{}, ErrInvalid
	}
	tx, err := repo.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return Reconciliation{}, Account{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	account, err := lockAccount(ctx, tx, accountID)
	if err != nil {
		return Reconciliation{}, Account{}, err
	}
	if stored, err := scanReconciliation(tx.QueryRow(ctx, reconciliationQuery+` WHERE idempotency_key=$1`, key)); err == nil {
		if stored.AccountID != accountID {
			return Reconciliation{}, Account{}, ErrInvalid
		}
		return stored, account, nil
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return Reconciliation{}, Account{}, err
	}
	result, err := preview(ctx, tx, account, request.AsOf, request.StatedMinor)
	if err != nil {
		return Reconciliation{}, Account{}, err
	}
	if result.PreviewHash != request.PreviewHash {
		return Reconciliation{}, Account{}, ErrStalePreview
	}
	var adjustment *string
	switch result.Outcome {
	case "opening_set":
		if _, err = tx.Exec(ctx, `UPDATE accounts SET opening_balance_minor=$2, balance_reconciled=true, revision=revision+1, updated_at=now() WHERE id::text=$1`, accountID, *result.ProposedOpeningMinor); err != nil {
			return Reconciliation{}, Account{}, err
		}
	case "adjusted":
		id, err := insertAdjustment(ctx, tx, account, result, key)
		if err != nil {
			return Reconciliation{}, Account{}, err
		}
		adjustment = &id
	}
	stored, err := scanReconciliation(tx.QueryRow(ctx, `INSERT INTO account_reconciliations
		(account_id, as_of, stated_minor, computed_minor, difference_minor, outcome, adjustment_event_id, idempotency_key)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8) RETURNING `+reconciliationColumns,
		accountID, result.AsOf, result.StatedMinor, result.ComputedMinor, result.DifferenceMinor, result.Outcome, adjustment, key))
	if err != nil {
		return Reconciliation{}, Account{}, fmt.Errorf("record reconciliation: %w", err)
	}
	updated, err := loadAccount(ctx, tx, accountID)
	if err != nil {
		return Reconciliation{}, Account{}, err
	}
	return stored, updated, tx.Commit(ctx)
}

// insertAdjustment books the difference as a balance_adjustment: it moves
// the account balance and is neither expense nor income in analytics.
func insertAdjustment(ctx context.Context, tx pgx.Tx, account Account, result ReconciliationPreview, key string) (string, error) {
	direction, amount := "credit", result.DifferenceMinor
	if amount < 0 {
		direction, amount = "debit", -amount
	}
	hash := sha256.Sum256([]byte("reconciliation|" + account.ID + "|" + key))
	var id string
	err := tx.QueryRow(ctx, `INSERT INTO financial_events (kind, source, merchant, description, occurred_at, idempotency_key, request_hash)
		VALUES ('balance_adjustment', 'manual', '', 'Корректировка остатка по сверке', $1, $2, $3) RETURNING id::text`,
		result.AsOf, "reconciliation:"+hex.EncodeToString(hash[:16]), hex.EncodeToString(hash[:])).Scan(&id)
	if err != nil {
		return "", fmt.Errorf("insert adjustment: %w", err)
	}
	if _, err = tx.Exec(ctx, `INSERT INTO postings (financial_event_id, account_id, direction, amount_minor, currency) VALUES ($1, $2, $3, $4, $5)`,
		id, account.ID, direction, amount, account.Currency); err != nil {
		return "", fmt.Errorf("insert adjustment posting: %w", err)
	}
	return id, nil
}

const reconciliationColumns = `id::text, account_id::text, as_of, stated_minor, computed_minor, difference_minor, outcome, adjustment_event_id::text, created_at`

var reconciliationQuery = `SELECT ` + reconciliationColumns + ` FROM account_reconciliations`

func scanReconciliation(row pgx.Row) (Reconciliation, error) {
	var item Reconciliation
	err := row.Scan(&item.ID, &item.AccountID, &item.AsOf, &item.StatedMinor, &item.ComputedMinor, &item.DifferenceMinor, &item.Outcome, &item.AdjustmentEventID, &item.CreatedAt)
	return item, err
}

func (repo *Repository) Reconciliations(ctx context.Context, accountID string) ([]Reconciliation, error) {
	rows, err := repo.pool.Query(ctx, reconciliationQuery+` WHERE account_id::text=$1 ORDER BY created_at DESC, id DESC`, accountID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]Reconciliation, 0)
	for rows.Next() {
		item, err := scanReconciliation(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}
