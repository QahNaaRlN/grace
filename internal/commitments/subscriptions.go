package commitments

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"github.com/QahNaaRlN/grace/internal/financialevents"
	"github.com/QahNaaRlN/grace/internal/platform/database"
	"github.com/QahNaaRlN/grace/internal/platform/web"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"
)

type Subscription struct {
	ID                  string   `json:"id"`
	Name                string   `json:"name"`
	AmountMinor         int64    `json:"amountMinor"`
	Currency            string   `json:"currency"`
	Period              string   `json:"period"`
	StartsOn            string   `json:"startsOn"`
	EndsOn              *string  `json:"endsOn"`
	AccountID           *string  `json:"accountId"`
	PaymentInstrumentID *string  `json:"paymentInstrumentId"`
	CategoryID          *string  `json:"categoryId"`
	Paused              bool     `json:"paused"`
	Reminders           bool     `json:"reminders"`
	Revision            int      `json:"revision"`
	NextDueOn           *string  `json:"nextDueOn"`
	ExpenseIDs          []string `json:"expenseIds"`
}
type Handler struct{ pool *database.TenantPool }

func NewHandler(pool *pgxpool.Pool) *Handler { return &Handler{database.ForUsers(pool)} }
func (h *Handler) Routes(r chi.Router) {
	r.Get("/subscriptions", h.List)
	r.Post("/subscriptions", h.Save)
	r.Patch("/subscriptions/{subscriptionId}", h.Save)
	r.Post("/subscriptions/{subscriptionId}/expenses", h.LinkExpense)
	r.Get("/subscriptions/{subscriptionId}/expenses", h.ListExpenses)
	r.Get("/payment-reminders", h.ListReminders)
	r.Get("/loans/{obligationId}", h.GetLoan)
	r.Patch("/loans/{obligationId}", h.SaveLoan)
	r.Post("/loans/{obligationId}/documents", h.UploadDocument)
	r.Get("/loans/{obligationId}/documents/{documentId}", h.DownloadDocument)
	r.Post("/loans/{obligationId}/schedule-drafts", h.CopySchedule)
	r.Patch("/loans/{obligationId}/schedule-drafts/{draftId}", h.EditSchedule)
	r.Post("/loans/{obligationId}/schedule-drafts/{draftId}/apply", h.ApplySchedule)
	r.Post("/loans/{obligationId}/schedule-drafts/{draftId}/calculate-parts", h.CalculateParts)
}

func (h *Handler) ListExpenses(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tx, err := h.pool.BeginTx(ctx, pgx.TxOptions{AccessMode: pgx.ReadOnly})
	if err != nil {
		failure(w, err)
		return
	}
	defer tx.Rollback(ctx)
	var id string
	if err = tx.QueryRow(ctx, `SELECT id::text FROM subscriptions WHERE id::text=$1`, chi.URLParam(r, "subscriptionId")).Scan(&id); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			web.WriteError(w, 404, "not_found", "Подписка не найдена.")
			return
		}
		failure(w, err)
		return
	}
	rows, err := tx.Query(ctx, `SELECT financial_event_id::text FROM subscription_expenses WHERE subscription_id=$1 ORDER BY created_at DESC,financial_event_id LIMIT 100`, id)
	if err != nil {
		failure(w, err)
		return
	}
	ids := []string{}
	for rows.Next() {
		var eventID string
		if err = rows.Scan(&eventID); err != nil {
			rows.Close()
			failure(w, err)
			return
		}
		ids = append(ids, eventID)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		failure(w, err)
		return
	}
	items := []financialevents.Event{}
	for _, eventID := range ids {
		event, loadErr := financialevents.LoadInTx(ctx, tx, eventID)
		if loadErr != nil {
			failure(w, loadErr)
			return
		}
		items = append(items, event)
	}
	if err = tx.Commit(ctx); err != nil {
		failure(w, err)
		return
	}
	web.WriteJSON(w, 200, map[string]any{"items": items})
}

type ReminderStatus struct {
	ID         string `json:"id"`
	SourceKind string `json:"sourceKind"`
	SourceID   string `json:"sourceId"`
	DueOn      string `json:"dueOn"`
	LeadDays   int    `json:"leadDays"`
	Status     string `json:"status"`
}

func (h *Handler) ListReminders(w http.ResponseWriter, r *http.Request) {
	rows, err := h.pool.Query(r.Context(), `SELECT id::text,source_kind,source_id::text,due_on::text,lead_days,status FROM payment_reminders ORDER BY due_on DESC,id LIMIT 100`)
	if err != nil {
		failure(w, err)
		return
	}
	defer rows.Close()
	items := []ReminderStatus{}
	for rows.Next() {
		var item ReminderStatus
		if err = rows.Scan(&item.ID, &item.SourceKind, &item.SourceID, &item.DueOn, &item.LeadDays, &item.Status); err != nil {
			failure(w, err)
			return
		}
		items = append(items, item)
	}
	if err = rows.Err(); err != nil {
		failure(w, err)
		return
	}
	web.WriteJSON(w, 200, map[string]any{"items": items})
}

const subscriptionColumns = `id::text,name,amount_minor,currency,period,starts_on::text,ends_on::text,account_id::text,payment_instrument_id::text,paused,reminders,revision,category_id::text`

func scanSubscription(row pgx.Row) (Subscription, error) {
	var s Subscription
	err := row.Scan(&s.ID, &s.Name, &s.AmountMinor, &s.Currency, &s.Period, &s.StartsOn, &s.EndsOn, &s.AccountID, &s.PaymentInstrumentID, &s.Paused, &s.Reminders, &s.Revision, &s.CategoryID)
	end := ""
	if s.EndsOn != nil {
		end = *s.EndsOn
	}
	s.NextDueOn = NextDate(s.StartsOn, s.Period, end, today(), s.Paused)
	s.ExpenseIDs = []string{}
	return s, err
}
func today() string { return time.Now().UTC().Add(5 * time.Hour).Format("2006-01-02") }
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	tx, err := h.pool.BeginTx(r.Context(), pgx.TxOptions{AccessMode: pgx.ReadOnly})
	if err != nil {
		failure(w, err)
		return
	}
	defer tx.Rollback(r.Context())
	rows, err := tx.Query(r.Context(), `SELECT `+subscriptionColumns+` FROM subscriptions ORDER BY paused,name,id LIMIT 500`)
	if err != nil {
		failure(w, err)
		return
	}
	items := []Subscription{}
	for rows.Next() {
		s, e := scanSubscription(rows)
		if e != nil {
			rows.Close()
			failure(w, e)
			return
		}
		items = append(items, s)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		failure(w, err)
		return
	}
	for i := range items {
		links, e := tx.Query(r.Context(), `SELECT financial_event_id::text FROM subscription_expenses WHERE subscription_id=$1 ORDER BY created_at DESC`, items[i].ID)
		if e != nil {
			failure(w, e)
			return
		}
		for links.Next() {
			var id string
			if e = links.Scan(&id); e != nil {
				links.Close()
				failure(w, e)
				return
			}
			items[i].ExpenseIDs = append(items[i].ExpenseIDs, id)
		}
		e = links.Err()
		links.Close()
		if e != nil {
			failure(w, e)
			return
		}
	}
	web.WriteJSON(w, 200, map[string]any{"items": items})
}
func validDate(value string) bool { _, err := time.Parse("2006-01-02", value); return err == nil }
func validText(value string, max int) bool {
	return utf8.RuneCountInString(value) <= max && !strings.ContainsAny(value, "\r\n\x00")
}
func (h *Handler) Save(w http.ResponseWriter, r *http.Request) {
	var s Subscription
	if !web.DecodeJSON(w, r, &s) {
		return
	}
	s.Name = strings.TrimSpace(s.Name)
	if s.Name == "" || !validText(s.Name, 160) || s.AmountMinor < 1 || s.AmountMinor > 9007199254740991 || len(s.Currency) != 3 || strings.Trim(s.Currency, "ABCDEFGHIJKLMNOPQRSTUVWXYZ") != "" || !validDate(s.StartsOn) || (s.EndsOn != nil && (!validDate(*s.EndsOn) || *s.EndsOn < s.StartsOn)) || (s.Period != "weekly" && s.Period != "monthly" && s.Period != "yearly") {
		invalid(w)
		return
	}
	id := chi.URLParam(r, "subscriptionId")
	// SPEC-0047: a new subscription needs an expense category, like a purchase.
	// On update null keeps the stored one, so older subscriptions still pause.
	if s.CategoryID == nil && id == "" {
		invalid(w)
		return
	}
	creationKey := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	if id == "" && (creationKey == "" || len(creationKey) > 160) {
		web.WriteError(w, 400, "idempotency_required", "Нужен ключ повторной отправки.")
		return
	}
	tx, err := h.pool.BeginTx(r.Context(), pgx.TxOptions{})
	if err != nil {
		failure(w, err)
		return
	}
	defer tx.Rollback(r.Context())
	if s.CategoryID != nil {
		var categoryKind string
		if err = tx.QueryRow(r.Context(), `SELECT kind::text FROM categories WHERE id::text=$1`, *s.CategoryID).Scan(&categoryKind); err != nil || categoryKind != "expense" {
			if err != nil && !errors.Is(err, pgx.ErrNoRows) {
				failure(w, err)
				return
			}
			invalid(w)
			return
		}
	}
	if id == "" {
		canonical := s
		canonical.ID = ""
		canonical.Revision = 0
		canonical.NextDueOn = nil
		canonical.ExpenseIDs = nil
		raw, _ := json.Marshal(canonical)
		sum := sha256.Sum256(raw)
		hash := hex.EncodeToString(sum[:])
		if _, err = tx.Exec(r.Context(), `SELECT pg_advisory_xact_lock(hashtextextended('subscription:'||grace_owner_id()::text||':'||$1,0))`, creationKey); err != nil {
			failure(w, err)
			return
		}
		var existing, existingHash string
		err = tx.QueryRow(r.Context(), `SELECT id::text,request_hash FROM subscriptions WHERE creation_key=$1`, creationKey).Scan(&existing, &existingHash)
		if err == nil {
			if hash != existingHash {
				failure(w, pgx.ErrNoRows)
				return
			}
			s, err = scanSubscription(tx.QueryRow(r.Context(), `SELECT `+subscriptionColumns+` FROM subscriptions WHERE id::text=$1`, existing))
			if err != nil {
				failure(w, err)
				return
			}
			web.WriteJSON(w, 200, s)
			return
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			failure(w, err)
			return
		}
		s, err = scanSubscription(tx.QueryRow(r.Context(), `INSERT INTO subscriptions(name,amount_minor,currency,period,starts_on,ends_on,account_id,paused,reminders,payment_instrument_id,creation_key,request_hash,category_id) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13) RETURNING `+subscriptionColumns, s.Name, s.AmountMinor, s.Currency, s.Period, s.StartsOn, s.EndsOn, s.AccountID, s.Paused, s.Reminders, s.PaymentInstrumentID, creationKey, hash, s.CategoryID))
	} else {
		s, err = scanSubscription(tx.QueryRow(r.Context(), `UPDATE subscriptions SET name=$2,amount_minor=$3,currency=$4,period=$5,starts_on=$6,ends_on=$7,account_id=$8,paused=$9,reminders=$10,payment_instrument_id=$12,category_id=COALESCE($13::uuid,category_id),revision=revision+1 WHERE id::text=$1 AND revision=$11 RETURNING `+subscriptionColumns, id, s.Name, s.AmountMinor, s.Currency, s.Period, s.StartsOn, s.EndsOn, s.AccountID, s.Paused, s.Reminders, s.Revision, s.PaymentInstrumentID, s.CategoryID))
	}
	if err != nil {
		failure(w, err)
		return
	}
	if err = tx.Commit(r.Context()); err != nil {
		failure(w, err)
		return
	}
	status := 200
	if id == "" {
		status = 201
	}
	web.WriteJSON(w, status, s)
}
func (h *Handler) LinkExpense(w http.ResponseWriter, r *http.Request) {
	var input struct {
		FinancialEventID string `json:"financialEventId"`
	}
	if !web.DecodeJSON(w, r, &input) {
		return
	}
	tx, err := h.pool.BeginTx(r.Context(), pgx.TxOptions{})
	if err != nil {
		failure(w, err)
		return
	}
	defer tx.Rollback(r.Context())
	// Lock the actual purchase; the relation creates no financial event or posting.
	var currency string
	err = tx.QueryRow(r.Context(), `SELECT s.currency FROM subscriptions s JOIN financial_events e ON e.id::text=$2 JOIN postings p ON p.financial_event_id=e.id WHERE s.id::text=$1 AND e.kind='purchase' AND p.currency=s.currency AND p.direction='debit' FOR UPDATE OF e,s`, chi.URLParam(r, "subscriptionId"), input.FinancialEventID).Scan(&currency)
	if err != nil {
		failure(w, err)
		return
	}
	var id string
	err = tx.QueryRow(r.Context(), `SELECT financial_event_id::text FROM subscription_expenses WHERE financial_event_id::text=$1 AND subscription_id::text=$2`, input.FinancialEventID, chi.URLParam(r, "subscriptionId")).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		err = tx.QueryRow(r.Context(), `INSERT INTO subscription_expenses(subscription_id,financial_event_id) VALUES($1,$2) RETURNING financial_event_id::text`, chi.URLParam(r, "subscriptionId"), input.FinancialEventID).Scan(&id)
	}
	if err != nil {
		failure(w, err)
		return
	}
	if err = tx.Commit(r.Context()); err != nil {
		failure(w, err)
		return
	}
	web.WriteJSON(w, 200, map[string]string{"financialEventId": id})
}
func invalid(w http.ResponseWriter) {
	web.WriteError(w, 422, "validation_error", "Проверьте название, сумму, даты и период.")
}
func failure(w http.ResponseWriter, err error) {
	var db *pgconn.PgError
	if errors.Is(err, pgx.ErrNoRows) {
		web.WriteError(w, 409, "conflict", "Запись недоступна или уже изменена. Обновите страницу.")
		return
	}
	if errors.As(err, &db) && (db.Code == "23503" || db.Code == "22P02" || db.Code == "23514") {
		invalid(w)
		return
	}
	if errors.As(err, &db) && db.Code == "23505" {
		web.WriteError(w, 409, "conflict", "Эта запись уже связана.")
		return
	}
	web.WriteError(w, 500, "internal_error", "Не удалось сохранить данные.")
}
func lockLoan(ctx context.Context, tx pgx.Tx, id string) error {
	var found string
	return tx.QueryRow(ctx, `SELECT id::text FROM obligations WHERE id::text=$1 AND kind='loan' FOR UPDATE`, id).Scan(&found)
}
