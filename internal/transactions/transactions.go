package transactions

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/QahNaaRlN/grace/internal/platform/web"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var kindDirection = map[string]string{
	"purchase":        "expense",
	"income_received": "income",
	"debt_payment":    "expense",
	"refund":          "income",
}

var validSources = map[string]bool{"manual": true, "bank_import": true, "receipt": true}

type Transaction struct {
	ID           string    `json:"id"`
	AccountID    string    `json:"accountId"`
	AccountName  string    `json:"accountName"`
	CategoryID   *string   `json:"categoryId"`
	CategoryName *string   `json:"categoryName"`
	AmountMinor  int64     `json:"amountMinor"`
	Currency     string    `json:"currency"`
	Direction    string    `json:"direction"`
	Kind         string    `json:"kind"`
	Source       string    `json:"source"`
	Merchant     string    `json:"merchant"`
	Description  string    `json:"description"`
	OccurredAt   time.Time `json:"occurredAt"`
	CreatedAt    time.Time `json:"createdAt"`
}

type CreateRequest struct {
	AccountID   string  `json:"accountId"`
	CategoryID  *string `json:"categoryId"`
	AmountMinor int64   `json:"amountMinor"`
	Direction   string  `json:"direction"`
	Kind        string  `json:"kind"`
	Source      string  `json:"source"`
	Merchant    string  `json:"merchant"`
	Description string  `json:"description"`
	OccurredAt  string  `json:"occurredAt"`
}

type Repository struct{ pool *pgxpool.Pool }

func NewRepository(pool *pgxpool.Pool) *Repository { return &Repository{pool: pool} }

func (repo *Repository) List(ctx context.Context, limit int) ([]Transaction, error) {
	rows, err := repo.pool.Query(ctx, `
		SELECT t.id::text, t.account_id::text, a.name, t.category_id::text, c.name,
		       t.amount_minor, t.currency, t.direction::text, t.kind::text, t.source::text,
		       t.merchant, t.description, t.occurred_at, t.created_at
		FROM transactions t
		JOIN accounts a ON a.id = t.account_id
		LEFT JOIN categories c ON c.id = t.category_id
		ORDER BY t.occurred_at DESC, t.created_at DESC
		LIMIT $1`, limit)
	if err != nil {
		return nil, fmt.Errorf("list transactions: %w", err)
	}
	defer rows.Close()
	items := make([]Transaction, 0)
	for rows.Next() {
		var item Transaction
		if err := rows.Scan(
			&item.ID, &item.AccountID, &item.AccountName, &item.CategoryID, &item.CategoryName,
			&item.AmountMinor, &item.Currency, &item.Direction, &item.Kind, &item.Source,
			&item.Merchant, &item.Description, &item.OccurredAt, &item.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan transaction: %w", err)
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (repo *Repository) Create(ctx context.Context, request CreateRequest, occurredAt time.Time) (Transaction, error) {
	tx, err := repo.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return Transaction{}, fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var accountName, currency string
	if err := tx.QueryRow(ctx, "SELECT name, currency FROM accounts WHERE id = $1 FOR UPDATE", request.AccountID).Scan(&accountName, &currency); err != nil {
		return Transaction{}, fmt.Errorf("lock account: %w", err)
	}

	var categoryName *string
	if request.CategoryID != nil {
		var name string
		if err := tx.QueryRow(ctx, "SELECT name FROM categories WHERE id = $1", *request.CategoryID).Scan(&name); err != nil {
			return Transaction{}, fmt.Errorf("load category: %w", err)
		}
		categoryName = &name
	}

	var item Transaction
	err = tx.QueryRow(ctx, `
		INSERT INTO transactions (
			account_id, category_id, amount_minor, currency, direction, kind, source,
			merchant, description, occurred_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id::text, account_id::text, category_id::text, amount_minor, currency,
		          direction::text, kind::text, source::text, merchant, description, occurred_at, created_at`,
		request.AccountID, request.CategoryID, request.AmountMinor, currency, request.Direction,
		request.Kind, request.Source, request.Merchant, request.Description, occurredAt,
	).Scan(
		&item.ID, &item.AccountID, &item.CategoryID, &item.AmountMinor, &item.Currency,
		&item.Direction, &item.Kind, &item.Source, &item.Merchant, &item.Description,
		&item.OccurredAt, &item.CreatedAt,
	)
	if err != nil {
		return Transaction{}, fmt.Errorf("insert transaction: %w", err)
	}
	item.AccountName = accountName
	item.CategoryName = categoryName

	delta := request.AmountMinor
	if request.Direction == "expense" {
		delta = -delta
	}
	if _, err := tx.Exec(ctx, "UPDATE accounts SET balance_minor = balance_minor + $1, updated_at = now() WHERE id = $2", delta, request.AccountID); err != nil {
		return Transaction{}, fmt.Errorf("update balance: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return Transaction{}, fmt.Errorf("commit transaction: %w", err)
	}
	return item, nil
}

type Handler struct{ repo *Repository }

func NewHandler(repo *Repository) *Handler { return &Handler{repo: repo} }

func (handler *Handler) List(w http.ResponseWriter, r *http.Request) {
	items, err := handler.repo.List(r.Context(), 100)
	if err != nil {
		web.WriteError(w, http.StatusInternalServerError, "internal_error", "Не удалось получить операции.")
		return
	}
	web.WriteJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (handler *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var request CreateRequest
	if !web.DecodeJSON(w, r, &request) {
		return
	}
	request.AccountID = strings.TrimSpace(request.AccountID)
	request.Direction = strings.TrimSpace(request.Direction)
	request.Kind = strings.TrimSpace(request.Kind)
	request.Source = strings.TrimSpace(request.Source)
	request.Merchant = strings.TrimSpace(request.Merchant)
	request.Description = strings.TrimSpace(request.Description)

	occurredAt, err := validateCreateRequest(request)
	if err != nil {
		web.WriteError(w, http.StatusUnprocessableEntity, "validation_error", err.Error())
		return
	}
	item, err := handler.repo.Create(r.Context(), request, occurredAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			web.WriteError(w, http.StatusUnprocessableEntity, "validation_error", "Указанный счёт не существует.")
			return
		}
		web.WriteError(w, http.StatusInternalServerError, "internal_error", "Не удалось сохранить операцию.")
		return
	}
	web.WriteJSON(w, http.StatusCreated, item)
}

func validateCreateRequest(request CreateRequest) (time.Time, error) {
	expectedDirection, knownKind := kindDirection[request.Kind]
	if request.AccountID == "" {
		return time.Time{}, fmt.Errorf("Укажите счёт.")
	}
	if request.AmountMinor <= 0 {
		return time.Time{}, fmt.Errorf("Сумма должна быть больше нуля.")
	}
	if !knownKind || request.Direction != expectedDirection {
		return time.Time{}, fmt.Errorf("Вид операции не соответствует направлению денег.")
	}
	if !validSources[request.Source] {
		return time.Time{}, fmt.Errorf("Неизвестный источник операции.")
	}
	if request.Kind == "purchase" && request.CategoryID == nil {
		return time.Time{}, fmt.Errorf("Для покупки требуется категория.")
	}
	if request.Kind == "purchase" && request.Merchant == "" && request.Description == "" {
		return time.Time{}, fmt.Errorf("Для покупки укажите получателя или описание.")
	}
	occurredAt, err := time.Parse(time.RFC3339, request.OccurredAt)
	if err != nil {
		return time.Time{}, fmt.Errorf("Дата операции должна быть в формате RFC 3339.")
	}
	return occurredAt, nil
}
