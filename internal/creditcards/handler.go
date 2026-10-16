package creditcards

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/QahNaaRlN/grace/internal/platform/database"
	"github.com/QahNaaRlN/grace/internal/platform/web"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// localDay is the calendar day of an operation in the application time zone.
const localDay = `COALESCE(e.occurred_date,(e.occurred_at AT TIME ZONE 'Asia/Yekaterinburg')::date)`

const documentQuota = 50 << 20

var errNotCredit = errors.New("account is not a credit card")

type TermsVersion struct {
	ID                  string  `json:"id"`
	EffectiveFrom       string  `json:"effectiveFrom"`
	LimitMinor          *int64  `json:"limitMinor"`
	PurchaseRateBP      *int    `json:"purchaseRateBasisPoints"`
	CashRateBP          *int    `json:"cashRateBasisPoints"`
	GraceModel          string  `json:"graceModel"`
	StatementDay        *int    `json:"statementDay"`
	PaymentDays         *int    `json:"paymentDays"`
	GraceDays           *int    `json:"graceDays"`
	MinPercentBP        *int    `json:"minPercentBasisPoints"`
	MinAmountMinor      *int64  `json:"minAmountMinor"`
	MinRoundingMinor    *int64  `json:"minRoundingMinor"`
	MinPercentIsCap     bool    `json:"minPercentIsCap"`
	MinIncludesInterest bool    `json:"minIncludesInterest"`
	DocumentID          *string `json:"documentId"`
	Note                string  `json:"note"`
	Revision            int     `json:"revision"`
}

type Snapshot struct {
	ID               string  `json:"id"`
	TakenOn          string  `json:"takenOn"`
	DebtMinor        int64   `json:"debtMinor"`
	AvailableMinor   *int64  `json:"availableMinor"`
	GraceAmountMinor *int64  `json:"graceAmountMinor"`
	GraceDueOn       *string `json:"graceDueOn"`
	MinimumMinor     *int64  `json:"minimumMinor"`
	MinimumDueOn     *string `json:"minimumDueOn"`
	Note             string  `json:"note"`
}

type Document struct {
	ID        string    `json:"id"`
	Filename  string    `json:"filename"`
	Size      int       `json:"size"`
	CreatedAt time.Time `json:"createdAt"`
}

type StatusView struct {
	On                  string   `json:"on"`
	Source              string   `json:"source"`
	DebtMinor           *int64   `json:"debtMinor"`
	AvailableMinor      *int64   `json:"availableMinor"`
	StatementClosedOn   *string  `json:"statementClosedOn"`
	NextStatementOn     *string  `json:"nextStatementOn"`
	GraceAmountMinor    *int64   `json:"graceAmountMinor"`
	GraceDueOn          *string  `json:"graceDueOn"`
	GraceEstimate       bool     `json:"graceEstimate"`
	GraceLost           bool     `json:"graceLost"`
	GraceAtRisk         bool     `json:"graceAtRisk"`
	MinimumMinor        *int64   `json:"minimumMinor"`
	MinimumDueOn        *string  `json:"minimumDueOn"`
	MinimumIsUpperBound bool     `json:"minimumIsUpperBound"`
	OverdueMinimumMinor int64    `json:"overdueMinimumMinor"`
	Unknown             []string `json:"unknown"`
}

type Comparison struct {
	TakenOn         string `json:"takenOn"`
	BankDebtMinor   int64  `json:"bankDebtMinor"`
	LedgerDebtMinor int64  `json:"ledgerDebtMinor"`
	DifferenceMinor int64  `json:"differenceMinor"`
}

type Card struct {
	AccountID      string         `json:"accountId"`
	Currency       string         `json:"currency"`
	Reminders      bool           `json:"reminders"`
	ActiveTermsID  *string        `json:"activeTermsId"`
	Terms          []TermsVersion `json:"terms"`
	Snapshots      []Snapshot     `json:"snapshots"`
	Documents      []Document     `json:"documents"`
	Status         StatusView     `json:"status"`
	BankComparison *Comparison    `json:"bankComparison"`
}

type Handler struct{ pool *database.TenantPool }

func NewHandler(pool *pgxpool.Pool) *Handler { return &Handler{database.ForUsers(pool)} }

func (h *Handler) Routes(r chi.Router) {
	r.Get("/accounts/{accountId}/credit-card", h.Get)
	r.Post("/accounts/{accountId}/credit-card/terms", h.CreateTerms)
	r.Patch("/accounts/{accountId}/credit-card/terms/{termsId}", h.EditTerms)
	r.Post("/accounts/{accountId}/credit-card/snapshots", h.CreateSnapshot)
	r.Put("/accounts/{accountId}/credit-card/settings", h.SaveSettings)
	r.Post("/accounts/{accountId}/credit-card/documents", h.UploadDocument)
	r.Get("/accounts/{accountId}/credit-card/documents/{documentId}", h.DownloadDocument)
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	on := Today()
	if value := r.URL.Query().Get("on"); value != "" {
		parsed, err := ParseDay(value)
		if err != nil {
			web.WriteError(w, 400, "invalid_request", "Дата должна быть в формате ГГГГ-ММ-ДД.")
			return
		}
		on = parsed
	}
	tx, err := h.pool.BeginTx(r.Context(), pgx.TxOptions{AccessMode: pgx.ReadOnly})
	if err != nil {
		failure(w, err)
		return
	}
	defer tx.Rollback(r.Context())
	card, err := Load(r.Context(), tx, chi.URLParam(r, "accountId"), on)
	if err != nil {
		failure(w, err)
		return
	}
	web.WriteJSON(w, 200, card)
}

// Load reads the card and computes its status at the end of day `on`.
func Load(ctx context.Context, tx pgx.Tx, accountID string, on time.Time) (Card, error) {
	card := Card{AccountID: accountID, Terms: []TermsVersion{}, Snapshots: []Snapshot{}, Documents: []Document{}}
	var kind string
	var opening *int64
	var reconciled bool
	err := tx.QueryRow(ctx, `SELECT a.type::text,a.currency,a.opening_balance_minor,a.balance_reconciled,COALESCE(s.reminders,false)
		FROM accounts a LEFT JOIN credit_card_settings s ON s.account_id=a.id WHERE a.id::text=$1`, accountID).
		Scan(&kind, &card.Currency, &opening, &reconciled, &card.Reminders)
	if err != nil {
		return card, err
	}
	if kind != "credit" {
		return card, errNotCredit
	}
	if card.Terms, err = loadTerms(ctx, tx, accountID); err != nil {
		return card, err
	}
	if card.Snapshots, err = loadSnapshots(ctx, tx, accountID); err != nil {
		return card, err
	}
	if card.Documents, err = loadDocuments(ctx, tx, accountID); err != nil {
		return card, err
	}
	terms, termsID := activeTerms(card.Terms, on)
	card.ActiveTermsID = termsID
	movements, first, err := loadMovements(ctx, tx, accountID, on)
	if err != nil {
		return card, err
	}
	var latest *Snapshot
	for i := range card.Snapshots {
		if taken, _ := ParseDay(card.Snapshots[i].TakenOn); !taken.After(on) {
			latest = &card.Snapshots[i]
			break
		}
	}
	view := StatusView{On: FormatDay(on), Source: "none", Unknown: []string{}}
	switch {
	case reconciled && opening != nil:
		start := on
		if first != nil {
			start = *first
		}
		ledger := Ledger{Start: start, OpeningDebtMinor: -*opening, Movements: movements}
		view = toView(Calculate(terms, ledger, on), on, "ledger")
		if latest != nil {
			taken, _ := ParseDay(latest.TakenOn)
			grace := debtAt(ledger, taken)
			card.BankComparison = &Comparison{latest.TakenOn, latest.DebtMinor, grace, latest.DebtMinor - grace}
		}
	case latest != nil:
		// Without a reconciled balance, the bank snapshot is the starting point.
		taken, _ := ParseDay(latest.TakenOn)
		after := []Movement{}
		for _, m := range movements {
			if m.On.After(taken) {
				after = append(after, m)
			}
		}
		ledger := Ledger{Start: taken.AddDate(0, 0, 1), OpeningDebtMinor: latest.DebtMinor, Movements: after}
		view = toView(Calculate(terms, ledger, on), on, "snapshot")
	default:
		status := Calculate(terms, Ledger{Start: on}, on)
		view = toView(status, on, "none")
		view.DebtMinor, view.AvailableMinor = nil, nil
		view.Unknown = append([]string{UnknownDebt}, view.Unknown...)
	}
	card.Status = view
	return card, nil
}

func toView(s Status, on time.Time, source string) StatusView {
	debt := s.DebtMinor
	return StatusView{
		On: FormatDay(on), Source: source, DebtMinor: &debt, AvailableMinor: s.AvailableMinor,
		StatementClosedOn: dayPtr(s.StatementClosedOn), NextStatementOn: dayPtr(s.NextStatementOn),
		GraceAmountMinor: s.GraceAmountMinor, GraceDueOn: dayPtr(s.GraceDueOn), GraceEstimate: s.GraceEstimate,
		GraceLost: s.GraceLost, GraceAtRisk: s.GraceAtRisk, MinimumMinor: s.MinimumMinor,
		MinimumDueOn: dayPtr(s.MinimumDueOn), MinimumIsUpperBound: s.MinimumIsUpperBound,
		OverdueMinimumMinor: s.OverdueMinimumMinor, Unknown: s.Unknown,
	}
}

func dayPtr(t *time.Time) *string {
	if t == nil {
		return nil
	}
	v := FormatDay(*t)
	return &v
}

// activeTerms picks the version with the latest start on or before `on`.
func activeTerms(versions []TermsVersion, on time.Time) (*Terms, *string) {
	for _, v := range versions { // newest first
		from, err := ParseDay(v.EffectiveFrom)
		if err != nil || from.After(on) {
			continue
		}
		id := v.ID
		return &Terms{
			EffectiveFrom: from, LimitMinor: v.LimitMinor, GraceModel: v.GraceModel,
			StatementDay: v.StatementDay, PaymentDays: v.PaymentDays, GraceDays: v.GraceDays,
			MinPercentBP: v.MinPercentBP, MinAmountMinor: v.MinAmountMinor, MinRoundingMinor: v.MinRoundingMinor,
			MinPercentIsCap: v.MinPercentIsCap, MinIncludesInterest: v.MinIncludesInterest,
		}, &id
	}
	return nil, nil
}

const termsColumns = `id::text,effective_from::text,limit_minor,purchase_rate_bp,cash_rate_bp,grace_model,statement_day,payment_days,grace_days,min_percent_bp,min_amount_minor,min_rounding_minor,min_percent_is_cap,min_includes_interest,document_id::text,note,revision`

func scanTerms(row pgx.Row) (TermsVersion, error) {
	var v TermsVersion
	err := row.Scan(&v.ID, &v.EffectiveFrom, &v.LimitMinor, &v.PurchaseRateBP, &v.CashRateBP, &v.GraceModel, &v.StatementDay, &v.PaymentDays, &v.GraceDays, &v.MinPercentBP, &v.MinAmountMinor, &v.MinRoundingMinor, &v.MinPercentIsCap, &v.MinIncludesInterest, &v.DocumentID, &v.Note, &v.Revision)
	return v, err
}

func loadTerms(ctx context.Context, tx pgx.Tx, accountID string) ([]TermsVersion, error) {
	rows, err := tx.Query(ctx, `SELECT `+termsColumns+` FROM credit_card_terms WHERE account_id::text=$1 ORDER BY effective_from DESC`, accountID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []TermsVersion{}
	for rows.Next() {
		v, err := scanTerms(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

func loadSnapshots(ctx context.Context, tx pgx.Tx, accountID string) ([]Snapshot, error) {
	rows, err := tx.Query(ctx, `SELECT id::text,taken_on::text,debt_minor,available_minor,grace_amount_minor,grace_due_on::text,minimum_minor,minimum_due_on::text,note
		FROM credit_card_snapshots WHERE account_id::text=$1 ORDER BY taken_on DESC,created_at DESC LIMIT 24`, accountID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Snapshot{}
	for rows.Next() {
		var s Snapshot
		if err := rows.Scan(&s.ID, &s.TakenOn, &s.DebtMinor, &s.AvailableMinor, &s.GraceAmountMinor, &s.GraceDueOn, &s.MinimumMinor, &s.MinimumDueOn, &s.Note); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func loadDocuments(ctx context.Context, tx pgx.Tx, accountID string) ([]Document, error) {
	rows, err := tx.Query(ctx, `SELECT DISTINCT d.id::text,d.filename,octet_length(d.content),d.created_at FROM documents d
		JOIN credit_card_terms t ON t.document_id=d.id WHERE t.account_id::text=$1 ORDER BY d.created_at DESC`, accountID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Document{}
	for rows.Next() {
		var d Document
		if err := rows.Scan(&d.ID, &d.Filename, &d.Size, &d.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

// loadMovements returns the account's debt changes up to `on` (spending positive)
// and the first day with an operation.
func loadMovements(ctx context.Context, tx pgx.Tx, accountID string, on time.Time) ([]Movement, *time.Time, error) {
	rows, err := tx.Query(ctx, `SELECT `+localDay+`::text,SUM(CASE p.direction WHEN 'debit' THEN p.amount_minor ELSE -p.amount_minor END)
		FROM postings p JOIN financial_events e ON e.id=p.financial_event_id
		WHERE p.account_id::text=$1 AND `+localDay+`<=$2::date GROUP BY 1 ORDER BY 1`, accountID, FormatDay(on))
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	out := []Movement{}
	var first *time.Time
	for rows.Next() {
		var dayText string
		var amount int64
		if err := rows.Scan(&dayText, &amount); err != nil {
			return nil, nil, err
		}
		d, err := ParseDay(dayText)
		if err != nil {
			return nil, nil, err
		}
		if first == nil {
			first = &d
		}
		out = append(out, Movement{d, amount})
	}
	return out, first, rows.Err()
}

func (h *Handler) CreateTerms(w http.ResponseWriter, r *http.Request) {
	key, ok := idempotencyKey(w, r)
	if !ok {
		return
	}
	var v TermsVersion
	if !web.DecodeJSON(w, r, &v) || !validTerms(w, v) {
		return
	}
	h.write(w, r, func(ctx context.Context, tx pgx.Tx, accountID string) (int, error) {
		var existing string
		err := tx.QueryRow(ctx, `SELECT id::text FROM credit_card_terms WHERE creation_key=$1`, key).Scan(&existing)
		if err == nil {
			return 200, nil
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return 0, err
		}
		_, err = tx.Exec(ctx, `INSERT INTO credit_card_terms(account_id,effective_from,limit_minor,purchase_rate_bp,cash_rate_bp,grace_model,statement_day,payment_days,grace_days,min_percent_bp,min_amount_minor,min_rounding_minor,min_percent_is_cap,min_includes_interest,document_id,note,creation_key)
			VALUES($1::uuid,$2::date,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15::uuid,$16,$17)`,
			accountID, v.EffectiveFrom, v.LimitMinor, v.PurchaseRateBP, v.CashRateBP, v.GraceModel, v.StatementDay, v.PaymentDays, v.GraceDays, v.MinPercentBP, v.MinAmountMinor, v.MinRoundingMinor, v.MinPercentIsCap, v.MinIncludesInterest, v.DocumentID, v.Note, key)
		return 201, err
	})
}

func (h *Handler) EditTerms(w http.ResponseWriter, r *http.Request) {
	var v TermsVersion
	if !web.DecodeJSON(w, r, &v) || !validTerms(w, v) {
		return
	}
	termsID := chi.URLParam(r, "termsId")
	h.write(w, r, func(ctx context.Context, tx pgx.Tx, accountID string) (int, error) {
		current, err := scanTerms(tx.QueryRow(ctx, `SELECT `+termsColumns+` FROM credit_card_terms WHERE id::text=$1 AND account_id::text=$2 FOR UPDATE`, termsID, accountID))
		if err != nil {
			return 0, err
		}
		if current.Revision != v.Revision {
			return 0, pgx.ErrNoRows
		}
		previous, err := json.Marshal(current)
		if err != nil {
			return 0, err
		}
		if _, err = tx.Exec(ctx, `INSERT INTO credit_card_term_revisions(terms_id,revision,previous) VALUES($1::uuid,$2,$3)`, termsID, current.Revision, previous); err != nil {
			return 0, err
		}
		_, err = tx.Exec(ctx, `UPDATE credit_card_terms SET effective_from=$2::date,limit_minor=$3,purchase_rate_bp=$4,cash_rate_bp=$5,grace_model=$6,statement_day=$7,payment_days=$8,grace_days=$9,min_percent_bp=$10,min_amount_minor=$11,min_rounding_minor=$12,min_percent_is_cap=$13,min_includes_interest=$14,document_id=$15::uuid,note=$16,revision=revision+1,updated_at=now() WHERE id::text=$1`,
			termsID, v.EffectiveFrom, v.LimitMinor, v.PurchaseRateBP, v.CashRateBP, v.GraceModel, v.StatementDay, v.PaymentDays, v.GraceDays, v.MinPercentBP, v.MinAmountMinor, v.MinRoundingMinor, v.MinPercentIsCap, v.MinIncludesInterest, v.DocumentID, v.Note)
		return 200, err
	})
}

func (h *Handler) CreateSnapshot(w http.ResponseWriter, r *http.Request) {
	key, ok := idempotencyKey(w, r)
	if !ok {
		return
	}
	var s Snapshot
	if !web.DecodeJSON(w, r, &s) {
		return
	}
	if !validDay(s.TakenOn) || (s.GraceDueOn != nil && !validDay(*s.GraceDueOn)) || (s.MinimumDueOn != nil && !validDay(*s.MinimumDueOn)) ||
		!validText(s.Note, 500) || negative(s.AvailableMinor) || negative(s.GraceAmountMinor) || negative(s.MinimumMinor) {
		invalid(w)
		return
	}
	h.write(w, r, func(ctx context.Context, tx pgx.Tx, accountID string) (int, error) {
		tag, err := tx.Exec(ctx, `INSERT INTO credit_card_snapshots(account_id,taken_on,debt_minor,available_minor,grace_amount_minor,grace_due_on,minimum_minor,minimum_due_on,note,creation_key)
			VALUES($1::uuid,$2::date,$3,$4,$5,$6::date,$7,$8::date,$9,$10) ON CONFLICT(owner_id,creation_key) DO NOTHING`,
			accountID, s.TakenOn, s.DebtMinor, s.AvailableMinor, s.GraceAmountMinor, s.GraceDueOn, s.MinimumMinor, s.MinimumDueOn, s.Note, key)
		if err != nil {
			return 0, err
		}
		if tag.RowsAffected() == 0 {
			return 200, nil
		}
		return 201, nil
	})
}

func (h *Handler) SaveSettings(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Reminders bool `json:"reminders"`
	}
	if !web.DecodeJSON(w, r, &body) {
		return
	}
	h.write(w, r, func(ctx context.Context, tx pgx.Tx, accountID string) (int, error) {
		_, err := tx.Exec(ctx, `INSERT INTO credit_card_settings(account_id,reminders) VALUES($1::uuid,$2) ON CONFLICT(account_id) DO UPDATE SET reminders=EXCLUDED.reminders`, accountID, body.Reminders)
		return 200, err
	})
}

func (h *Handler) UploadDocument(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 17<<20)
	if err := r.ParseMultipartForm(1 << 20); err != nil {
		invalid(w)
		return
	}
	if r.MultipartForm != nil {
		defer r.MultipartForm.RemoveAll()
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		invalid(w)
		return
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, (16<<20)+1))
	if err != nil || len(data) > 16<<20 || !bytes.HasPrefix(data, []byte("%PDF-")) {
		invalid(w)
		return
	}
	name := filepath.Base(strings.ReplaceAll(header.Filename, "\\", "/"))
	if name == "." || strings.TrimSpace(name) == "" || !validText(name, 180) {
		invalid(w)
		return
	}
	tx, err := h.pool.BeginTx(r.Context(), pgx.TxOptions{})
	if err != nil {
		failure(w, err)
		return
	}
	defer tx.Rollback(r.Context())
	if err = lockCard(r.Context(), tx, chi.URLParam(r, "accountId")); err != nil {
		failure(w, err)
		return
	}
	if _, err = tx.Exec(r.Context(), `SELECT pg_advisory_xact_lock(hashtextextended('documents:'||grace_owner_id()::text,0))`); err != nil {
		failure(w, err)
		return
	}
	var used int64
	if err = tx.QueryRow(r.Context(), `SELECT COALESCE(sum(octet_length(content)),0) FROM documents`).Scan(&used); err != nil {
		failure(w, err)
		return
	}
	if used+int64(len(data)) > documentQuota {
		web.WriteError(w, 422, "document_quota", "Лимит документов карт — 50 МиБ. Новый файл не сохранён.")
		return
	}
	digest := sha256.Sum256(data)
	d := Document{Filename: name, Size: len(data)}
	if err = tx.QueryRow(r.Context(), `INSERT INTO documents(kind,filename,sha256,content) VALUES('credit_card_terms',$1,$2,$3) RETURNING id::text,created_at`, name, hex.EncodeToString(digest[:]), data).Scan(&d.ID, &d.CreatedAt); err != nil {
		failure(w, err)
		return
	}
	if err = tx.Commit(r.Context()); err != nil {
		failure(w, err)
		return
	}
	web.WriteJSON(w, 201, d)
}

func (h *Handler) DownloadDocument(w http.ResponseWriter, r *http.Request) {
	var name string
	var data []byte
	// A document belongs to the card through one of its term versions; a fresh upload
	// is reachable until it is attached.
	err := h.pool.QueryRow(r.Context(), `SELECT d.filename,d.content FROM documents d WHERE d.id::text=$1 AND (
		EXISTS(SELECT 1 FROM credit_card_terms t WHERE t.document_id=d.id AND t.account_id::text=$2)
		OR NOT EXISTS(SELECT 1 FROM credit_card_terms t WHERE t.document_id=d.id))`, chi.URLParam(r, "documentId"), chi.URLParam(r, "accountId")).Scan(&name, &data)
	if err != nil {
		web.WriteError(w, 404, "not_found", "Документ недоступен.")
		return
	}
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": name}))
	w.Header().Set("Cache-Control", "private, no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	_, _ = w.Write(data)
}

// write runs a change for a credit card account and answers with the fresh card.
func (h *Handler) write(w http.ResponseWriter, r *http.Request, change func(context.Context, pgx.Tx, string) (int, error)) {
	ctx := r.Context()
	accountID := chi.URLParam(r, "accountId")
	tx, err := h.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		failure(w, err)
		return
	}
	defer tx.Rollback(ctx)
	if err = lockCard(ctx, tx, accountID); err != nil {
		failure(w, err)
		return
	}
	status, err := change(ctx, tx, accountID)
	if err != nil {
		failure(w, err)
		return
	}
	card, err := Load(ctx, tx, accountID, Today())
	if err != nil {
		failure(w, err)
		return
	}
	if err = tx.Commit(ctx); err != nil {
		failure(w, err)
		return
	}
	web.WriteJSON(w, status, card)
}

func lockCard(ctx context.Context, tx pgx.Tx, accountID string) error {
	var kind string
	if err := tx.QueryRow(ctx, `SELECT type::text FROM accounts WHERE id::text=$1 FOR UPDATE`, accountID).Scan(&kind); err != nil {
		return err
	}
	if kind != "credit" {
		return errNotCredit
	}
	return nil
}

func validTerms(w http.ResponseWriter, v TermsVersion) bool {
	ok := validDay(v.EffectiveFrom) && validText(v.Note, 500) &&
		(v.LimitMinor == nil || *v.LimitMinor > 0) && !negative(v.MinAmountMinor) &&
		(v.MinRoundingMinor == nil || *v.MinRoundingMinor >= 1)
	switch v.GraceModel {
	case GraceStatement:
		ok = ok && v.StatementDay != nil && v.PaymentDays != nil
	case GraceFromFirstPurchase:
		ok = ok && v.GraceDays != nil
	case GraceNone, GraceUnknown:
	default:
		ok = false
	}
	if !ok {
		web.WriteError(w, 422, "validation_error", "Проверьте дату начала, модель льготного периода и суммы условий.")
	}
	return ok
}

func idempotencyKey(w http.ResponseWriter, r *http.Request) (string, bool) {
	key := r.Header.Get("Idempotency-Key")
	if key == "" || utf8.RuneCountInString(key) > 160 {
		web.WriteError(w, 400, "idempotency_key_required", "Передайте заголовок Idempotency-Key до 160 символов.")
		return "", false
	}
	return key, true
}

func validDay(value string) bool {
	_, err := ParseDay(value)
	return err == nil
}
func validText(value string, max int) bool {
	return utf8.RuneCountInString(value) <= max && !strings.ContainsAny(value, "\x00")
}
func negative(v *int64) bool { return v != nil && *v < 0 }

func invalid(w http.ResponseWriter) {
	web.WriteError(w, 422, "validation_error", "Проверьте даты, суммы и файл.")
}

func failure(w http.ResponseWriter, err error) {
	var db *pgconn.PgError
	switch {
	case errors.Is(err, errNotCredit):
		web.WriteError(w, 422, "not_credit_card", "Условия карты доступны только для кредитного счёта.")
	case errors.Is(err, pgx.ErrNoRows):
		web.WriteError(w, 409, "conflict", "Запись недоступна или уже изменена. Обновите страницу.")
	case errors.As(err, &db) && db.Code == "23505":
		web.WriteError(w, 409, "conflict", "Версия условий с этой датой начала уже есть.")
	case errors.As(err, &db) && (db.Code == "23503" || db.Code == "22P02" || db.Code == "23514"):
		invalid(w)
	default:
		web.WriteError(w, 500, "internal_error", "Не удалось сохранить данные карты.")
	}
}

// Today is the current calendar day in the application time zone (UTC+5).
func Today() time.Time { return day(time.Now().UTC().Add(5 * time.Hour)) }
