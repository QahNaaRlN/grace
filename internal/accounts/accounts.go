package accounts

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
	"net/http"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/QahNaaRlN/grace/internal/financialevents"
	"github.com/QahNaaRlN/grace/internal/platform/database"
	"github.com/QahNaaRlN/grace/internal/platform/web"
	"github.com/jackc/pgx/v5/pgxpool"
)

var validTypes = map[string]bool{"debit": true, "credit": true, "installment": true, "cash": true}

type PaymentInstrument struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Bank  string `json:"bank"`
	Last4 string `json:"last4"`
}

type CardRequest struct {
	Name  string `json:"name"`
	Bank  string `json:"bank"`
	Last4 string `json:"last4"`
}

type Account struct {
	PaymentInstruments []PaymentInstrument `json:"paymentInstruments,omitempty"`
	ID                 string              `json:"id"`
	Name               string              `json:"name"`
	Type               string              `json:"type"`
	Currency           string              `json:"currency"`
	BalanceMinor       *int64              `json:"balanceMinor"`
	BalanceReconciled  bool                `json:"balanceReconciled"`
	CreatedAt          time.Time           `json:"createdAt"`
}

type CreateRequest struct {
	Card                *CardRequest `json:"card,omitempty"`
	Name                string       `json:"name"`
	Type                string       `json:"type"`
	Currency            string       `json:"currency"`
	OpeningBalanceMinor *int64       `json:"openingBalanceMinor"`
	BalanceReconciled   *bool        `json:"balanceReconciled,omitempty"`
}

func (request CreateRequest) reconciled() bool {
	return request.BalanceReconciled == nil || *request.BalanceReconciled
}

func (request *CreateRequest) UnmarshalJSON(data []byte) error {
	type plain CreateRequest
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}
	if _, ok := fields["openingBalanceMinor"]; !ok {
		return fmt.Errorf("openingBalanceMinor is required")
	}
	var parsed plain
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&parsed); err != nil {
		return err
	}
	*request = CreateRequest(parsed)
	return nil
}

type Repository struct{ pool *database.TenantPool }

func NewRepository(pool *pgxpool.Pool) *Repository { return &Repository{pool: database.ForUsers(pool)} }

func (repo *Repository) List(ctx context.Context) ([]Account, error) {
	rows, err := repo.pool.Query(ctx, `
		SELECT
			a.id::text,
			a.name,
			a.type::text,
			a.currency,
			CASE WHEN a.balance_reconciled THEN a.opening_balance_minor + COALESCE(SUM(
				CASE p.direction
					WHEN 'credit' THEN p.amount_minor
					ELSE -p.amount_minor
				END
			), 0) ELSE NULL END AS balance_minor,
			a.balance_reconciled, a.created_at, COALESCE((SELECT jsonb_agg(jsonb_build_object('id', i.id, 'name', i.name, 'bank', i.bank, 'last4', i.last4) ORDER BY i.created_at, i.id) FROM payment_instruments i WHERE i.account_id = a.id), '[]'::jsonb)
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
		if err := rows.Scan(&item.ID, &item.Name, &item.Type, &item.Currency, &item.BalanceMinor, &item.BalanceReconciled, &item.CreatedAt, &item.PaymentInstruments); err != nil {
			return nil, fmt.Errorf("scan account: %w", err)
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (repo *Repository) Create(ctx context.Context, request CreateRequest) (Account, error) {
	tx, err := repo.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return Account{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err = financialevents.LockOwner(ctx, tx); err != nil {
		return Account{}, err
	}
	var item Account
	err = tx.QueryRow(ctx, `INSERT INTO accounts (name, type, currency, opening_balance_minor, balance_reconciled)
 VALUES ($1, $2, $3, $4, $5) RETURNING id::text, name, type::text, currency, CASE WHEN balance_reconciled THEN opening_balance_minor ELSE NULL END, balance_reconciled, created_at`,
		request.Name, request.Type, request.Currency, request.OpeningBalanceMinor, request.reconciled()).Scan(&item.ID, &item.Name, &item.Type, &item.Currency, &item.BalanceMinor, &item.BalanceReconciled, &item.CreatedAt)
	if err != nil {
		return Account{}, fmt.Errorf("create account: %w", err)
	}
	if request.Card != nil {
		card := PaymentInstrument{Name: request.Card.Name, Bank: request.Card.Bank, Last4: request.Card.Last4}
		err = tx.QueryRow(ctx, `INSERT INTO payment_instruments (account_id,name,bank,last4) VALUES ($1,$2,$3,$4) RETURNING id::text`, item.ID, card.Name, card.Bank, card.Last4).Scan(&card.ID)
		if err != nil {
			return Account{}, fmt.Errorf("create card: %w", err)
		}
		item.PaymentInstruments = []PaymentInstrument{card}
	}
	if err = tx.Commit(ctx); err != nil {
		return Account{}, err
	}
	return item, nil
}

var lastFourPattern = regexp.MustCompile(`^[0-9]{4}$`)

func validateRequest(request *CreateRequest) bool {
	request.Name = strings.TrimSpace(request.Name)
	request.Type = strings.TrimSpace(request.Type)
	request.Currency = strings.ToUpper(strings.TrimSpace(request.Currency))
	if request.reconciled() && request.OpeningBalanceMinor == nil {
		return false
	}
	if request.Name == "" || utf8.RuneCountInString(request.Name) > 160 || !validTypes[request.Type] || !regexp.MustCompile(`^[A-Z]{3}$`).MatchString(request.Currency) {
		return false
	}
	if request.Card != nil {
		request.Card.Name = strings.TrimSpace(request.Card.Name)
		request.Card.Bank = strings.TrimSpace(request.Card.Bank)
		if request.Type == "cash" || request.Type == "installment" || request.Card.Name == "" || request.Card.Bank == "" || utf8.RuneCountInString(request.Card.Name) > 160 || utf8.RuneCountInString(request.Card.Bank) > 160 || !lastFourPattern.MatchString(request.Card.Last4) {
			return false
		}
	}
	return true
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
	if !validateRequest(&request) {
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
