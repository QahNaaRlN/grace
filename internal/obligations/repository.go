package obligations

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/QahNaaRlN/grace/internal/platform/database"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrNotFound            = errors.New("obligation or scheduled payment not found")
	ErrInvalid             = errors.New("obligation settlement is invalid")
	ErrIdempotencyConflict = errors.New("idempotency key conflicts with another payload")
	ErrEventAlreadySettled = errors.New("financial event already has a settlement")
)

type Repository struct{ pool *database.TenantPool }

func NewRepository(pool *pgxpool.Pool) *Repository { return &Repository{pool: database.ForUsers(pool)} }

type queryer interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
}

func (repo *Repository) List(ctx context.Context, asOf string, limit int) ([]Obligation, error) {
	rows, err := repo.pool.Query(ctx, obligationQuery+`
		WHERE o.id IN (
			SELECT id FROM obligations ORDER BY created_at DESC, id LIMIT $1
		)
		ORDER BY o.created_at DESC, o.id, sp.position, s.created_at, s.id`, limit)
	if err != nil {
		return nil, fmt.Errorf("list obligations: %w", err)
	}
	defer rows.Close()
	return collectObligations(rows, asOf)
}

func (repo *Repository) Get(ctx context.Context, id, asOf string) (Obligation, error) {
	return loadObligation(ctx, repo.pool, id, asOf)
}

func (repo *Repository) Create(ctx context.Context, command CreateCommand) (Obligation, bool, error) {
	tx, err := repo.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return Obligation{}, false, fmt.Errorf("begin obligation: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, "SELECT pg_advisory_xact_lock(hashtextextended($1, 0))", "obligation:"+command.IdempotencyKey); err != nil {
		return Obligation{}, false, fmt.Errorf("lock obligation idempotency key: %w", err)
	}

	var existingID, existingHash string
	err = tx.QueryRow(ctx, `SELECT id::text, request_hash FROM obligations WHERE idempotency_key = $1`, command.IdempotencyKey).Scan(&existingID, &existingHash)
	if err == nil {
		if existingHash != command.RequestHash {
			return Obligation{}, false, ErrIdempotencyConflict
		}
		obligation, err := loadObligation(ctx, tx, existingID, command.AsOf)
		return obligation, false, err
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return Obligation{}, false, fmt.Errorf("find idempotent obligation: %w", err)
	}

	var obligationID string
	err = tx.QueryRow(ctx, `
		INSERT INTO obligations (name, kind, currency, total_minor, idempotency_key, request_hash)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id::text`,
		command.Name, command.Kind, command.Currency, command.TotalMinor,
		command.IdempotencyKey, command.RequestHash,
	).Scan(&obligationID)
	if err != nil {
		return Obligation{}, false, fmt.Errorf("insert obligation: %w", err)
	}
	for index, payment := range command.Payments {
		if _, err := tx.Exec(ctx, `
			INSERT INTO scheduled_payments (obligation_id, position, due_on, amount_minor)
			VALUES ($1, $2, $3, $4)`,
			obligationID, index+1, payment.DueOn, payment.AmountMinor,
		); err != nil {
			return Obligation{}, false, fmt.Errorf("insert scheduled payment: %w", err)
		}
	}

	obligation, err := loadObligation(ctx, tx, obligationID, command.AsOf)
	if err != nil {
		return Obligation{}, false, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Obligation{}, false, fmt.Errorf("commit obligation: %w", err)
	}
	return obligation, true, nil
}

func (repo *Repository) CreateSettlement(ctx context.Context, command SettlementCommand) (Settlement, bool, error) {
	tx, err := repo.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return Settlement{}, false, fmt.Errorf("begin settlement: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, "SELECT pg_advisory_xact_lock(hashtextextended($1, 0))", "obligation-settlement:"+command.IdempotencyKey); err != nil {
		return Settlement{}, false, fmt.Errorf("lock settlement idempotency key: %w", err)
	}

	var existingID, existingHash string
	err = tx.QueryRow(ctx, `
		SELECT id::text, request_hash
		FROM scheduled_payment_settlements WHERE idempotency_key = $1`,
		command.IdempotencyKey,
	).Scan(&existingID, &existingHash)
	if err == nil {
		if existingHash != command.RequestHash {
			return Settlement{}, false, ErrIdempotencyConflict
		}
		settlement, err := loadSettlement(ctx, tx, existingID)
		return settlement, false, err
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return Settlement{}, false, fmt.Errorf("find idempotent settlement: %w", err)
	}

	var paymentAmount int64
	var currency string
	err = tx.QueryRow(ctx, `
		SELECT sp.amount_minor, o.currency
		FROM scheduled_payments sp
		JOIN obligations o ON o.id = sp.obligation_id
		WHERE sp.id::text = $1 AND o.id::text = $2
		FOR UPDATE OF sp, o`, command.PaymentID, command.ObligationID,
	).Scan(&paymentAmount, &currency)
	if errors.Is(err, pgx.ErrNoRows) {
		return Settlement{}, false, ErrNotFound
	}
	if err != nil {
		return Settlement{}, false, fmt.Errorf("lock scheduled payment: %w", err)
	}

	rows, err := tx.Query(ctx, `
		SELECT e.kind::text, p.direction::text, p.amount_minor, p.currency
		FROM financial_events e
		JOIN postings p ON p.financial_event_id = e.id
		WHERE e.id::text = $1
		FOR UPDATE OF e, p`, command.FinancialEventID)
	if err != nil {
		return Settlement{}, false, fmt.Errorf("lock financial event: %w", err)
	}
	type eventPosting struct {
		kind, direction, currency string
		amount                    int64
	}
	postings := make([]eventPosting, 0, 2)
	for rows.Next() {
		var posting eventPosting
		if err := rows.Scan(&posting.kind, &posting.direction, &posting.amount, &posting.currency); err != nil {
			rows.Close()
			return Settlement{}, false, fmt.Errorf("scan financial event: %w", err)
		}
		postings = append(postings, posting)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return Settlement{}, false, fmt.Errorf("read financial event: %w", err)
	}
	rows.Close()
	if len(postings) != 1 || postings[0].kind != "debt_payment" || postings[0].direction != "debit" || postings[0].currency != currency {
		return Settlement{}, false, fmt.Errorf("%w: financial event must be one debit debt_payment in %s", ErrInvalid, currency)
	}

	var used bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS (
		SELECT 1 FROM scheduled_payment_settlements WHERE financial_event_id::text = $1
	)`, command.FinancialEventID).Scan(&used); err != nil {
		return Settlement{}, false, fmt.Errorf("check financial event settlement: %w", err)
	}
	if used {
		return Settlement{}, false, ErrEventAlreadySettled
	}

	var paidMinor int64
	if err := tx.QueryRow(ctx, `
		SELECT COALESCE(SUM(amount_minor), 0)
		FROM scheduled_payment_settlements WHERE scheduled_payment_id::text = $1`,
		command.PaymentID,
	).Scan(&paidMinor); err != nil {
		return Settlement{}, false, fmt.Errorf("sum scheduled payment settlements: %w", err)
	}
	if postings[0].amount > paymentAmount-paidMinor {
		return Settlement{}, false, fmt.Errorf("%w: debt payment exceeds scheduled payment remainder", ErrInvalid)
	}

	var settlementID string
	err = tx.QueryRow(ctx, `
		INSERT INTO scheduled_payment_settlements (
			scheduled_payment_id, financial_event_id, amount_minor,
			idempotency_key, request_hash
		) VALUES ($1, $2, $3, $4, $5)
		RETURNING id::text`, command.PaymentID, command.FinancialEventID,
		postings[0].amount, command.IdempotencyKey, command.RequestHash,
	).Scan(&settlementID)
	if err != nil {
		return Settlement{}, false, fmt.Errorf("insert settlement: %w", err)
	}
	settlement, err := loadSettlement(ctx, tx, settlementID)
	if err != nil {
		return Settlement{}, false, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Settlement{}, false, fmt.Errorf("commit settlement: %w", err)
	}
	return settlement, true, nil
}

const obligationQuery = `
	SELECT
		o.id::text, o.name, o.kind, o.currency, o.total_minor, o.created_at,
		sp.id::text, sp.position, sp.due_on::text, sp.amount_minor,
		s.id::text, s.financial_event_id::text, s.amount_minor, s.created_at
	FROM obligations o
	JOIN scheduled_payments sp ON sp.obligation_id = o.id
	LEFT JOIN scheduled_payment_settlements s ON s.scheduled_payment_id = sp.id
	`

func loadObligation(ctx context.Context, db queryer, id, asOf string) (Obligation, error) {
	rows, err := db.Query(ctx, obligationQuery+`
		WHERE o.id::text = $1
		ORDER BY sp.position, s.created_at, s.id`, id)
	if err != nil {
		return Obligation{}, fmt.Errorf("load obligation: %w", err)
	}
	defer rows.Close()
	items, err := collectObligations(rows, asOf)
	if err != nil {
		return Obligation{}, err
	}
	if len(items) != 1 {
		return Obligation{}, ErrNotFound
	}
	return items[0], nil
}

func collectObligations(rows pgx.Rows, asOf string) ([]Obligation, error) {
	items := make([]Obligation, 0)
	obligationIndex := make(map[string]int)
	paymentIndex := make(map[string]int)
	for rows.Next() {
		var obligation Obligation
		var payment ScheduledPayment
		var settlementID, eventID *string
		var settlementAmount *int64
		var settlementCreatedAt *time.Time
		if err := rows.Scan(
			&obligation.ID, &obligation.Name, &obligation.Kind, &obligation.Currency,
			&obligation.TotalMinor, &obligation.CreatedAt, &payment.ID,
			&payment.Position, &payment.DueOn, &payment.AmountMinor,
			&settlementID, &eventID, &settlementAmount, &settlementCreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan obligation: %w", err)
		}
		oi, exists := obligationIndex[obligation.ID]
		if !exists {
			obligation.Payments = make([]ScheduledPayment, 0)
			items = append(items, obligation)
			oi = len(items) - 1
			obligationIndex[obligation.ID] = oi
		}
		pi, exists := paymentIndex[payment.ID]
		if !exists {
			payment.Settlements = make([]Settlement, 0)
			items[oi].Payments = append(items[oi].Payments, payment)
			pi = len(items[oi].Payments) - 1
			paymentIndex[payment.ID] = pi
		}
		if settlementID != nil && eventID != nil && settlementAmount != nil && settlementCreatedAt != nil {
			items[oi].Payments[pi].Settlements = append(items[oi].Payments[pi].Settlements, Settlement{
				ID: *settlementID, ScheduledPaymentID: payment.ID,
				FinancialEventID: *eventID, AmountMinor: *settlementAmount,
				CreatedAt: *settlementCreatedAt,
			})
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for oi := range items {
		for pi := range items[oi].Payments {
			payment := &items[oi].Payments[pi]
			for _, settlement := range payment.Settlements {
				payment.PaidMinor += settlement.AmountMinor
			}
			payment.RemainingMinor = payment.AmountMinor - payment.PaidMinor
			payment.IsOverdue = payment.RemainingMinor > 0 && payment.DueOn < asOf
			switch {
			case payment.RemainingMinor == 0:
				payment.Status = "paid"
			case payment.PaidMinor > 0:
				payment.Status = "partially_paid"
			case payment.IsOverdue:
				payment.Status = "overdue"
			default:
				payment.Status = "planned"
			}
			items[oi].PaidMinor += payment.PaidMinor
		}
		items[oi].RemainingMinor = items[oi].TotalMinor - items[oi].PaidMinor
		items[oi].Status = "active"
		if items[oi].RemainingMinor == 0 {
			items[oi].Status = "completed"
		}
	}
	return items, nil
}

func loadSettlement(ctx context.Context, db queryer, id string) (Settlement, error) {
	rows, err := db.Query(ctx, `
		SELECT id::text, scheduled_payment_id::text, financial_event_id::text,
		       amount_minor, created_at
		FROM scheduled_payment_settlements WHERE id::text = $1`, id)
	if err != nil {
		return Settlement{}, fmt.Errorf("load settlement: %w", err)
	}
	defer rows.Close()
	if !rows.Next() {
		return Settlement{}, ErrNotFound
	}
	var settlement Settlement
	if err := rows.Scan(&settlement.ID, &settlement.ScheduledPaymentID,
		&settlement.FinancialEventID, &settlement.AmountMinor, &settlement.CreatedAt); err != nil {
		return Settlement{}, fmt.Errorf("scan settlement: %w", err)
	}
	return settlement, rows.Err()
}
