package loans

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/QahNaaRlN/grace/internal/platform/database"
	"github.com/QahNaaRlN/grace/internal/platform/web"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	errInvalid    = errors.New("invalid loan allocation")
	errLinked     = errors.New("financial event is already linked")
	errNotLinked  = errors.New("financial event is not linked to a loan")
	errIdempotent = errors.New("idempotency key conflicts with another payload")
	errFuture     = errors.New("snapshot date is in the future")
)

type Handler struct{ pool *database.TenantPool }

func NewHandler(pool *pgxpool.Pool) *Handler { return &Handler{database.ForUsers(pool)} }

func (h *Handler) Routes(r chi.Router) {
	r.Post("/loans/{obligationId}/prepayments", h.CreatePrepayment)
	r.Patch("/loans/{obligationId}/prepayments/{prepaymentId}", h.EditPrepayment)
	r.Post("/loans/{obligationId}/debt-snapshots", h.CreateSnapshot)
	r.Patch("/loans/{obligationId}/debt-snapshots/{snapshotId}", h.EditSnapshot)
	r.Get("/financial-events/{eventId}/loan-allocation", h.GetAllocation)
	r.Put("/financial-events/{eventId}/loan-allocation", h.SaveAllocation)
	r.Delete("/financial-events/{eventId}/loan-allocation", h.ResetAllocation)
}

// optionalParts is a split in a request; all three parts or none.
type optionalParts struct {
	PrincipalMinor *int64 `json:"principalMinor"`
	InterestMinor  *int64 `json:"interestMinor"`
	OtherMinor     *int64 `json:"otherMinor"`
}

func (p optionalParts) parts(amount int64) (Parts, bool) {
	if p.PrincipalMinor == nil && p.InterestMinor == nil && p.OtherMinor == nil {
		return Parts{PrincipalMinor: amount}, true
	}
	if !ValidParts(p.PrincipalMinor, p.InterestMinor, p.OtherMinor, amount) {
		return Parts{}, false
	}
	return Parts{*p.PrincipalMinor, *p.InterestMinor, *p.OtherMinor}, true
}

// CreatePrepayment links a debt payment to the loan without a schedule row; without
// parts the whole amount repays principal.
func (h *Handler) CreatePrepayment(w http.ResponseWriter, r *http.Request) {
	key, ok := idempotencyKey(w, r)
	if !ok {
		return
	}
	var body struct {
		FinancialEventID string `json:"financialEventId"`
		optionalParts
	}
	if !web.DecodeJSON(w, r, &body) {
		return
	}
	body.FinancialEventID = strings.TrimSpace(body.FinancialEventID)
	hash := hashOf(body)
	h.loanWrite(w, r, func(ctx context.Context, tx pgx.Tx, loanID, currency string) (int, error) {
		var existingHash string
		err := tx.QueryRow(ctx, `SELECT request_hash FROM loan_prepayments WHERE creation_key=$1`, key).Scan(&existingHash)
		if err == nil {
			if existingHash != hash {
				return 0, errIdempotent
			}
			return 200, nil
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return 0, err
		}
		amount, err := lockDebtPayment(ctx, tx, body.FinancialEventID, currency)
		if err != nil {
			return 0, err
		}
		var linked bool
		if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM scheduled_payment_settlements WHERE financial_event_id::text=$1)
			OR EXISTS(SELECT 1 FROM loan_prepayments WHERE financial_event_id::text=$1)`, body.FinancialEventID).Scan(&linked); err != nil {
			return 0, err
		}
		if linked {
			return 0, errLinked
		}
		parts, ok := body.parts(amount)
		if !ok {
			return 0, errInvalid
		}
		_, err = tx.Exec(ctx, `INSERT INTO loan_prepayments(obligation_id,financial_event_id,amount_minor,principal_minor,interest_minor,other_minor,creation_key,request_hash)
			VALUES($1::uuid,$2::uuid,$3,$4,$5,$6,$7,$8)`, loanID, body.FinancialEventID, amount, parts.PrincipalMinor, parts.InterestMinor, parts.OtherMinor, key, hash)
		if err != nil {
			return 0, err
		}
		return 201, recordRevision(ctx, tx, body.FinancialEventID, "prepayment", nil, parts)
	})
}

func (h *Handler) EditPrepayment(w http.ResponseWriter, r *http.Request) {
	var body struct {
		optionalParts
		Revision int `json:"revision"`
	}
	if !web.DecodeJSON(w, r, &body) {
		return
	}
	h.loanWrite(w, r, func(ctx context.Context, tx pgx.Tx, loanID, _ string) (int, error) {
		var eventID string
		var amount int64
		var current Parts
		var revision int
		err := tx.QueryRow(ctx, `SELECT financial_event_id::text,amount_minor,principal_minor,interest_minor,other_minor,revision
			FROM loan_prepayments WHERE id::text=$1 AND obligation_id::text=$2 FOR UPDATE`, chi.URLParam(r, "prepaymentId"), loanID).
			Scan(&eventID, &amount, &current.PrincipalMinor, &current.InterestMinor, &current.OtherMinor, &revision)
		if err != nil {
			return 0, err
		}
		if revision != body.Revision {
			return 0, pgx.ErrNoRows
		}
		parts, ok := body.parts(amount)
		if !ok {
			return 0, errInvalid
		}
		if _, err = tx.Exec(ctx, `UPDATE loan_prepayments SET principal_minor=$2,interest_minor=$3,other_minor=$4,revision=revision+1 WHERE id::text=$1`,
			chi.URLParam(r, "prepaymentId"), parts.PrincipalMinor, parts.InterestMinor, parts.OtherMinor); err != nil {
			return 0, err
		}
		return 200, recordRevision(ctx, tx, eventID, "prepayment", &current, parts)
	})
}

type snapshotBody struct {
	TakenOn              string `json:"takenOn"`
	PrincipalMinor       int64  `json:"principalMinor"`
	AccruedInterestMinor *int64 `json:"accruedInterestMinor"`
	PayoffMinor          *int64 `json:"payoffMinor"`
	Note                 string `json:"note"`
	Revision             int    `json:"revision"`
}

func (b snapshotBody) valid() error {
	on, err := ParseDay(b.TakenOn)
	switch {
	case err != nil || FormatDay(on) != b.TakenOn || b.PrincipalMinor < 0 || negative(b.AccruedInterestMinor) || negative(b.PayoffMinor) ||
		utf8.RuneCountInString(b.Note) > 500 || strings.ContainsRune(b.Note, 0):
		return errInvalid
	case on.After(Today()):
		return errFuture
	}
	return nil
}

func (h *Handler) CreateSnapshot(w http.ResponseWriter, r *http.Request) {
	key, ok := idempotencyKey(w, r)
	if !ok {
		return
	}
	var body snapshotBody
	if !web.DecodeJSON(w, r, &body) {
		return
	}
	body.Note = strings.TrimSpace(body.Note)
	if err := body.valid(); err != nil {
		failure(w, err)
		return
	}
	h.loanWrite(w, r, func(ctx context.Context, tx pgx.Tx, loanID, _ string) (int, error) {
		var exists bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM loan_debt_snapshots WHERE creation_key=$1)`, key).Scan(&exists); err != nil || exists {
			return 200, err
		}
		_, err := tx.Exec(ctx, `INSERT INTO loan_debt_snapshots(obligation_id,taken_on,principal_minor,accrued_interest_minor,payoff_minor,note,creation_key)
			VALUES($1::uuid,$2::date,$3,$4,$5,$6,$7)`, loanID, body.TakenOn, body.PrincipalMinor, body.AccruedInterestMinor, body.PayoffMinor, body.Note, key)
		return 201, err
	})
}

func (h *Handler) EditSnapshot(w http.ResponseWriter, r *http.Request) {
	var body snapshotBody
	if !web.DecodeJSON(w, r, &body) {
		return
	}
	body.Note = strings.TrimSpace(body.Note)
	if err := body.valid(); err != nil {
		failure(w, err)
		return
	}
	h.loanWrite(w, r, func(ctx context.Context, tx pgx.Tx, loanID, _ string) (int, error) {
		tag, err := tx.Exec(ctx, `UPDATE loan_debt_snapshots SET taken_on=$3::date,principal_minor=$4,accrued_interest_minor=$5,payoff_minor=$6,note=$7,revision=revision+1
			WHERE id::text=$1 AND obligation_id::text=$2 AND revision=$8`,
			chi.URLParam(r, "snapshotId"), loanID, body.TakenOn, body.PrincipalMinor, body.AccruedInterestMinor, body.PayoffMinor, body.Note, body.Revision)
		if err == nil && tag.RowsAffected() == 0 {
			err = pgx.ErrNoRows
		}
		return 200, err
	})
}

// loanWrite runs a change of a loan and answers with its fresh debt status.
func (h *Handler) loanWrite(w http.ResponseWriter, r *http.Request, change func(context.Context, pgx.Tx, string, string) (int, error)) {
	ctx := r.Context()
	loanID := chi.URLParam(r, "obligationId")
	tx, err := h.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		failure(w, err)
		return
	}
	defer tx.Rollback(ctx)
	currency, err := lockLoan(ctx, tx, loanID)
	if err != nil {
		failure(w, err)
		return
	}
	status, err := change(ctx, tx, loanID, currency)
	if err != nil {
		failure(w, err)
		return
	}
	debt, err := Debt(ctx, tx, loanID, Today())
	if err != nil {
		failure(w, err)
		return
	}
	if err = tx.Commit(ctx); err != nil {
		failure(w, err)
		return
	}
	web.WriteJSON(w, status, debt)
}

// LockLoan locks a loan obligation and returns its currency.
func lockLoan(ctx context.Context, tx pgx.Tx, id string) (string, error) {
	var kind, currency string
	if err := tx.QueryRow(ctx, `SELECT kind,currency FROM obligations WHERE id::text=$1 FOR UPDATE`, id).Scan(&kind, &currency); err != nil {
		return "", err
	}
	if kind != "loan" {
		return "", ErrNotLoan
	}
	return currency, nil
}

// lockDebtPayment locks an operation that must be one debit debt payment in the
// loan currency and returns its amount.
func lockDebtPayment(ctx context.Context, tx pgx.Tx, eventID, currency string) (int64, error) {
	rows, err := tx.Query(ctx, `SELECT e.kind::text,p.direction::text,p.amount_minor,p.currency FROM financial_events e
		JOIN postings p ON p.financial_event_id=e.id WHERE e.id::text=$1 AND e.voided_at IS NULL FOR UPDATE OF e,p`, eventID)
	if err != nil {
		return 0, err
	}
	defer rows.Close()
	n := 0
	var kind, direction, postingCurrency string
	var amount int64
	for rows.Next() {
		n++
		if err := rows.Scan(&kind, &direction, &amount, &postingCurrency); err != nil {
			return 0, err
		}
	}
	if err := rows.Err(); err != nil {
		return 0, err
	}
	if n != 1 || kind != "debt_payment" || direction != "debit" || postingCurrency != currency {
		return 0, errInvalid
	}
	return amount, nil
}

func recordRevision(ctx context.Context, tx pgx.Tx, eventID, part string, previous *Parts, current any) error {
	var before []byte
	if previous != nil {
		before, _ = json.Marshal(previous)
	}
	after, _ := json.Marshal(current)
	if current == nil {
		after = nil
	}
	_, err := tx.Exec(ctx, `INSERT INTO loan_allocation_revisions(financial_event_id,part,previous,current) VALUES($1::uuid,$2,$3,$4)`, eventID, part, before, after)
	return err
}

func idempotencyKey(w http.ResponseWriter, r *http.Request) (string, bool) {
	key := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	if key == "" || utf8.RuneCountInString(key) > 160 {
		web.WriteError(w, 400, "idempotency_key_required", "Передайте заголовок Idempotency-Key до 160 символов.")
		return "", false
	}
	return key, true
}

func hashOf(v any) string {
	data, _ := json.Marshal(v)
	digest := sha256.Sum256(data)
	return hex.EncodeToString(digest[:])
}

func negative(v *int64) bool { return v != nil && *v < 0 }

func failure(w http.ResponseWriter, err error) {
	var db *pgconn.PgError
	switch {
	case errors.Is(err, ErrNotLoan):
		web.WriteError(w, 422, "not_loan", "Тело и проценты доступны только для кредита.")
	case errors.Is(err, errInvalid):
		web.WriteError(w, 422, "validation_error", "Проверьте платёж и части: тело, проценты и прочее в сумме должны равняться платежу.")
	case errors.Is(err, errFuture):
		web.WriteError(w, 422, "validation_error", "Дата данных из банка не может быть позже сегодняшней.")
	case errors.Is(err, errLinked):
		web.WriteError(w, 409, "financial_event_linked", "Этот платёж уже связан с графиком или досрочным погашением.")
	case errors.Is(err, errNotLinked):
		web.WriteError(w, 404, "not_found", "Платёж не связан с графиком кредита.")
	case errors.Is(err, errIdempotent):
		web.WriteError(w, 409, "idempotency_conflict", "Этот Idempotency-Key уже использован с другим запросом.")
	case errors.Is(err, pgx.ErrNoRows):
		web.WriteError(w, 409, "conflict", "Запись недоступна или уже изменена. Обновите страницу.")
	case errors.As(err, &db) && db.Code == "23505":
		web.WriteError(w, 409, "conflict", "Данные из банка на эту дату уже есть — исправьте их.")
	case errors.As(err, &db) && (db.Code == "23503" || db.Code == "22P02" || db.Code == "23514"):
		web.WriteError(w, 422, "validation_error", "Проверьте даты и суммы.")
	default:
		web.WriteError(w, 500, "internal_error", "Не удалось сохранить данные кредита.")
	}
}
