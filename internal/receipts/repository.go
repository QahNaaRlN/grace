package receipts

import (
	"context"
	"errors"
	"fmt"

	"github.com/QahNaaRlN/grace/internal/platform/database"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrNotFound            = errors.New("receipt not found")
	ErrInvalid             = errors.New("receipt is invalid")
	ErrIdempotencyConflict = errors.New("idempotency key conflicts with another payload")
	ErrPurchaseHasReceipt  = errors.New("purchase already has a receipt")
)

type Repository struct{ pool *database.TenantPool }

func HasPurchaseReceiptInTx(ctx context.Context, tx pgx.Tx, eventID string) (bool, error) {
	var exists bool
	err := tx.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM receipts WHERE purchase_event_id::text=$1)", eventID).Scan(&exists)
	return exists, err
}

func GetInTx(ctx context.Context, tx pgx.Tx, id string) (Receipt, error) {
	return loadReceipt(ctx, tx, id)
}

func NewRepository(pool *pgxpool.Pool) *Repository { return &Repository{pool: database.ForUsers(pool)} }

type queryer interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
}

func (repo *Repository) List(ctx context.Context, limit int) ([]Receipt, error) {
	rows, err := repo.pool.Query(ctx, receiptQuery+`
		WHERE r.id IN (
			SELECT id FROM receipts
			ORDER BY COALESCE(issued_at, issued_date::timestamp AT TIME ZONE 'Asia/Yekaterinburg') DESC, created_at DESC
			LIMIT $1
		)
		ORDER BY COALESCE(r.issued_at, r.issued_date::timestamp AT TIME ZONE 'Asia/Yekaterinburg') DESC, r.created_at DESC, i.position`, limit)
	if err != nil {
		return nil, fmt.Errorf("list receipts: %w", err)
	}
	defer rows.Close()
	return collectReceipts(rows)
}

func (repo *Repository) Get(ctx context.Context, id string) (Receipt, error) {
	return loadReceipt(ctx, repo.pool, id)
}

func (repo *Repository) Create(ctx context.Context, command Command) (Receipt, bool, error) {
	tx, err := repo.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return Receipt{}, false, fmt.Errorf("begin receipt: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	receipt, created, err := CreateInTx(ctx, tx, command)
	if err != nil {
		return Receipt{}, false, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Receipt{}, false, err
	}
	return receipt, created, nil
}

// CreateInTx allows a confirmed purchase and its receipt to share one atomic
// transaction. It never creates a financial event or another posting itself.
func CreateInTx(ctx context.Context, tx pgx.Tx, command Command) (Receipt, bool, error) {
	if err := ValidateCommand(command); err != nil {
		return Receipt{}, false, err
	}

	if _, err := tx.Exec(ctx,
		"SELECT pg_advisory_xact_lock(hashtextextended($1, 0))",
		"receipt:"+command.IdempotencyKey,
	); err != nil {
		return Receipt{}, false, fmt.Errorf("lock idempotency key: %w", err)
	}

	var existingID, existingHash string
	err := tx.QueryRow(ctx, `
		SELECT id::text, request_hash FROM receipts WHERE idempotency_key = $1`,
		command.IdempotencyKey,
	).Scan(&existingID, &existingHash)
	if err == nil {
		if existingHash != command.RequestHash {
			return Receipt{}, false, ErrIdempotencyConflict
		}
		receipt, err := loadReceipt(ctx, tx, existingID)
		return receipt, false, err
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return Receipt{}, false, fmt.Errorf("find idempotent receipt: %w", err)
	}

	var currency string
	var purchaseAmount int64
	err = tx.QueryRow(ctx, `
		SELECT p.currency, p.amount_minor
		FROM financial_events e
		JOIN postings p ON p.financial_event_id = e.id
		WHERE e.id::text = $1 AND e.kind = 'purchase' AND p.direction = 'debit'
		FOR UPDATE OF e`, command.PurchaseEventID,
	).Scan(&currency, &purchaseAmount)
	if errors.Is(err, pgx.ErrNoRows) {
		return Receipt{}, false, fmt.Errorf("%w: purchase event does not exist", ErrInvalid)
	}
	if err != nil {
		return Receipt{}, false, fmt.Errorf("lock purchase: %w", err)
	}
	if command.TotalMinor != purchaseAmount {
		return Receipt{}, false, fmt.Errorf("%w: receipt total must match purchase amount", ErrInvalid)
	}

	var anotherReceiptID string
	err = tx.QueryRow(ctx, `
		SELECT id::text FROM receipts WHERE purchase_event_id::text = $1`,
		command.PurchaseEventID,
	).Scan(&anotherReceiptID)
	if err == nil {
		return Receipt{}, false, ErrPurchaseHasReceipt
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return Receipt{}, false, fmt.Errorf("find purchase receipt: %w", err)
	}

	var receiptID string
	var issuedAt, issuedDate any
	if command.IssuedDate != "" {
		issuedDate = command.IssuedDate
	} else {
		issuedAt = command.IssuedAt
	}
	err = tx.QueryRow(ctx, `
		INSERT INTO receipts (
			purchase_event_id, seller, issued_at, currency, total_minor,
			fiscal_document_number, idempotency_key, request_hash, issued_date
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id::text`,
		command.PurchaseEventID, command.Seller, issuedAt, currency,
		command.TotalMinor, command.FiscalDocumentNumber, command.IdempotencyKey,
		command.RequestHash, issuedDate,
	).Scan(&receiptID)
	if err != nil {
		return Receipt{}, false, fmt.Errorf("insert receipt: %w", err)
	}

	for index, item := range command.Items {
		_, err := tx.Exec(ctx, `
			INSERT INTO receipt_items (
				receipt_id, position, name, quantity_milli,
				unit_price_minor, line_total_minor
				, brand, package, broad_group
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
			receiptID, index+1, item.Name, item.QuantityMilli,
			item.UnitPriceMinor, item.LineTotalMinor, item.Brand, item.Package, item.BroadGroup,
		)
		if err != nil {
			return Receipt{}, false, fmt.Errorf("insert receipt item: %w", err)
		}
	}

	receipt, err := loadReceipt(ctx, tx, receiptID)
	if err != nil {
		return Receipt{}, false, err
	}
	return receipt, true, nil
}

const receiptQuery = `
	SELECT
		r.id::text,
		r.purchase_event_id::text,
		r.seller,
		r.issued_at,
		r.issued_date::text,
		r.currency,
		r.total_minor,
		r.fiscal_document_number,
		r.created_at,
		i.id::text,
		i.position,
		i.name,
		i.quantity_milli,
		i.unit_price_minor,
		i.line_total_minor, i.brand, i.package, i.broad_group
	FROM receipts r
	JOIN receipt_items i ON i.receipt_id = r.id
	`

func loadReceipt(ctx context.Context, db queryer, id string) (Receipt, error) {
	rows, err := db.Query(ctx, receiptQuery+`
		WHERE r.id::text = $1
		ORDER BY i.position`, id)
	if err != nil {
		return Receipt{}, fmt.Errorf("load receipt: %w", err)
	}
	defer rows.Close()
	receipts, err := collectReceipts(rows)
	if err != nil {
		return Receipt{}, err
	}
	if len(receipts) != 1 {
		return Receipt{}, ErrNotFound
	}
	return receipts[0], nil
}

func collectReceipts(rows pgx.Rows) ([]Receipt, error) {
	items := make([]Receipt, 0)
	index := make(map[string]int)
	for rows.Next() {
		var receipt Receipt
		var item Item
		if err := rows.Scan(
			&receipt.ID, &receipt.PurchaseEventID, &receipt.Seller,
			&receipt.IssuedAt, &receipt.IssuedDate, &receipt.Currency, &receipt.TotalMinor,
			&receipt.FiscalDocumentNumber, &receipt.CreatedAt,
			&item.ID, &item.Position, &item.Name, &item.QuantityMilli,
			&item.UnitPriceMinor, &item.LineTotalMinor, &item.Brand, &item.Package, &item.BroadGroup,
		); err != nil {
			return nil, fmt.Errorf("scan receipt: %w", err)
		}
		position, exists := index[receipt.ID]
		receipt.DatePrecision = "instant"
		if receipt.IssuedDate != nil {
			receipt.DatePrecision = "date"
		}
		if !exists {
			receipt.Items = make([]Item, 0)
			items = append(items, receipt)
			position = len(items) - 1
			index[receipt.ID] = position
		}
		items[position].Items = append(items[position].Items, item)
	}
	return items, rows.Err()
}
