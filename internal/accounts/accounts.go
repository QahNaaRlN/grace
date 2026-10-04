package accounts

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/QahNaaRlN/grace/internal/platform/web"
	"github.com/jackc/pgx/v5/pgxpool"
)

var validTypes = map[string]bool{"debit": true, "credit": true, "installment": true, "cash": true}

type Account struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Type         string    `json:"type"`
	Currency     string    `json:"currency"`
	BalanceMinor int64     `json:"balanceMinor"`
	CreatedAt    time.Time `json:"createdAt"`
}

type CreateRequest struct {
	Name                string `json:"name"`
	Type                string `json:"type"`
	Currency            string `json:"currency"`
	OpeningBalanceMinor int64  `json:"openingBalanceMinor"`
}

type Repository struct{ pool *pgxpool.Pool }

func NewRepository(pool *pgxpool.Pool) *Repository { return &Repository{pool: pool} }

func (repo *Repository) List(ctx context.Context) ([]Account, error) {
	rows, err := repo.pool.Query(ctx, `
		SELECT
			a.id::text,
			a.name,
			a.type::text,
			a.currency,
			a.opening_balance_minor + COALESCE(SUM(
				CASE p.direction
					WHEN 'credit' THEN p.amount_minor
					ELSE -p.amount_minor
				END
			), 0) AS balance_minor,
			a.created_at
		FROM accounts a
		LEFT JOIN postings p ON p.account_id = a.id
		GROUP BY a.id
		ORDER BY a.created_at, a.name`)
	if err != nil {
		return nil, fmt.Errorf("list accounts: %w", err)
	}
	defer rows.Close()

	items := make([]Account, 0)
	for rows.Next() {
		var item Account
		if err := rows.Scan(&item.ID, &item.Name, &item.Type, &item.Currency, &item.BalanceMinor, &item.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan account: %w", err)
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (repo *Repository) Create(ctx context.Context, request CreateRequest) (Account, error) {
	var item Account
	err := repo.pool.QueryRow(ctx, `
		INSERT INTO accounts (name, type, currency, opening_balance_minor)
		VALUES ($1, $2, $3, $4)
		RETURNING id::text, name, type::text, currency, opening_balance_minor, created_at`,
		request.Name, request.Type, request.Currency, request.OpeningBalanceMinor,
	).Scan(&item.ID, &item.Name, &item.Type, &item.Currency, &item.BalanceMinor, &item.CreatedAt)
	if err != nil {
		return Account{}, fmt.Errorf("create account: %w", err)
	}
	return item, nil
}

type Handler struct{ repo *Repository }

func NewHandler(repo *Repository) *Handler { return &Handler{repo: repo} }

func (handler *Handler) List(w http.ResponseWriter, r *http.Request) {
	items, err := handler.repo.List(r.Context())
	if err != nil {
		web.WriteError(w, http.StatusInternalServerError, "internal_error", "Не удалось получить счета.")
		return
	}
	web.WriteJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (handler *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var request CreateRequest
	if !web.DecodeJSON(w, r, &request) {
		return
	}
	request.Name = strings.TrimSpace(request.Name)
	request.Type = strings.TrimSpace(request.Type)
	request.Currency = strings.ToUpper(strings.TrimSpace(request.Currency))
	if request.Name == "" || !validTypes[request.Type] || len(request.Currency) != 3 {
		web.WriteError(w, http.StatusUnprocessableEntity, "validation_error", "Проверьте название, тип и валюту счёта.")
		return
	}
	item, err := handler.repo.Create(r.Context(), request)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return
		}
		web.WriteError(w, http.StatusInternalServerError, "internal_error", "Не удалось создать счёт.")
		return
	}
	web.WriteJSON(w, http.StatusCreated, item)
}
