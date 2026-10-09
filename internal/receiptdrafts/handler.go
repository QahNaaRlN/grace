package receiptdrafts

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/QahNaaRlN/grace/internal/platform/access"
	"github.com/QahNaaRlN/grace/internal/platform/web"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Handler struct {
	repo      *Repository
	store     *PhotoStore
	extractor Extractor
}

func NewHandler(p *pgxpool.Pool, store *PhotoStore, extractor Extractor) *Handler {
	return &Handler{NewRepository(p), store, extractor}
}
func (h *Handler) Routes(r chi.Router) {
	r.Post("/", h.Upload)
	r.Get("/{draftId}", h.Get)
	r.Patch("/{draftId}", h.Edit)
	r.Delete("/{draftId}", h.Cancel)
	r.Post("/{draftId}/extract", h.Extract)
	r.Post("/{draftId}/previews", h.Preview)
	r.Post("/{draftId}/confirmations", h.Confirm)
}
func draftError(w http.ResponseWriter, err error) {
	code := "receipt_unavailable"
	status := 422
	for _, v := range []string{"not_found", "invalid_image", "image_busy", "invalid_request", "stale_revision", "draft_not_editable", "image_expired", "extraction_limit", "extraction_incomplete", "invalid_amount", "invalid_quantity", "invalid_date", "decision_required", "totals_mismatch", "candidate_review_required", "invalid_account", "invalid_existing_purchase", "stale_preview", "idempotency_conflict", "ai_disabled", "consent_required", "provider_unavailable", "provider_protocol", "provider_timeout", "price_unknown", "budget_exhausted", "ai_usage_requires_review", "ai_concurrency_limit"} {
		if err.Error() == v || strings.Contains(err.Error(), ": "+v) {
			code = v
			break
		}
	}
	if code == "not_found" {
		status = 404
	}
	if code == "stale_revision" || code == "stale_preview" || code == "draft_not_editable" || code == "idempotency_conflict" {
		status = 409
	}
	if code == "receipt_unavailable" || code == "ai_disabled" {
		status = 503
	}
	web.WriteError(w, status, code, "Чек не подтверждён. Проверь поля, совпадения и согласие; при изменении данных получи новый предварительный просмотр.")
}
func (h *Handler) enabled() bool { return h.store != nil }
func (h *Handler) Upload(w http.ResponseWriter, r *http.Request) {
	if !h.enabled() {
		draftError(w, errors.New("ai_disabled"))
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, MaxPhotoBytes+(64<<10))
	if err := r.ParseMultipartForm(1 << 20); err != nil {
		draftError(w, errors.New("invalid_image"))
		return
	}
	defer r.MultipartForm.RemoveAll()
	if len(r.MultipartForm.File) != 1 || len(r.MultipartForm.File["photo"]) != 1 {
		draftError(w, errors.New("invalid_image"))
		return
	}
	f, _, err := r.FormFile("photo")
	if err != nil {
		draftError(w, errors.New("invalid_image"))
		return
	}
	defer f.Close()
	raw, err := io.ReadAll(io.LimitReader(f, MaxPhotoBytes+1))
	if err != nil {
		draftError(w, errors.New("invalid_image"))
		return
	}
	normalized, err := NormalizePhoto(raw)
	if err != nil {
		draftError(w, err)
		return
	}
	id, err := h.store.Save(normalized)
	if err != nil {
		draftError(w, err)
		return
	}
	hash := sha256.Sum256(normalized)
	d, created, err := h.repo.Create(r.Context(), id, hex.EncodeToString(hash[:]))
	if err != nil || !created {
		_ = h.store.Delete(id)
	}
	if err != nil {
		draftError(w, err)
		return
	}
	status := 200
	if created {
		status = 201
	}
	web.WriteJSON(w, status, d)
}
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	d, err := h.repo.Get(r.Context(), chi.URLParam(r, "draftId"))
	if err != nil {
		draftError(w, err)
		return
	}
	web.WriteJSON(w, 200, d)
}
func (h *Handler) Edit(w http.ResponseWriter, r *http.Request) {
	var in Edit
	if !web.DecodeJSON(w, r, &in) {
		return
	}
	d, err := h.repo.Edit(r.Context(), chi.URLParam(r, "draftId"), in)
	if err != nil {
		draftError(w, err)
		return
	}
	web.WriteJSON(w, 200, d)
}
func (h *Handler) Extract(w http.ResponseWriter, r *http.Request) {
	key := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	if key == "" || len(key) > 160 {
		draftError(w, errors.New("invalid_request"))
		return
	}
	if !h.enabled() || h.extractor == nil || !h.extractor.Available() {
		draftError(w, errors.New("ai_disabled"))
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 90*time.Second)
	defer cancel()
	_ = http.NewResponseController(w).SetWriteDeadline(time.Now().Add(95 * time.Second))
	d, run, err := h.repo.BeginExtraction(ctx, chi.URLParam(r, "draftId"), key)
	if err != nil {
		draftError(w, err)
		return
	}
	if !run {
		web.WriteJSON(w, 200, d)
		return
	}
	photo, err := h.store.Read(d.ImageID)
	var out Extracted
	if err == nil {
		out, err = h.extractor.Extract(ctx, photo)
	}
	var result *Extracted
	if err == nil {
		result = &out
	}
	d, finishErr := h.repo.FinishExtraction(ctx, d.ID, key, result)
	if finishErr != nil {
		draftError(w, finishErr)
		return
	}
	if err != nil {
		draftError(w, err)
		return
	}
	web.WriteJSON(w, 200, d)
}
func (h *Handler) Preview(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Revision int64 `json:"revision"`
	}
	if !web.DecodeJSON(w, r, &in) {
		return
	}
	p, err := h.repo.Preview(r.Context(), chi.URLParam(r, "draftId"), in.Revision)
	if err != nil {
		draftError(w, err)
		return
	}
	web.WriteJSON(w, 201, p)
}
func (h *Handler) Confirm(w http.ResponseWriter, r *http.Request) {
	var in Confirmation
	if !web.DecodeJSON(w, r, &in) {
		return
	}
	key := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	if key == "" || len(key) > 160 {
		draftError(w, errors.New("invalid_request"))
		return
	}
	d, err := h.repo.Confirm(r.Context(), chi.URLParam(r, "draftId"), key, in)
	if err != nil {
		draftError(w, err)
		return
	}
	h.cleanup(r.Context(), &d)
	web.WriteJSON(w, 200, d)
}
func (h *Handler) Cancel(w http.ResponseWriter, r *http.Request) {
	d, err := h.repo.Cancel(r.Context(), chi.URLParam(r, "draftId"))
	if err != nil {
		draftError(w, err)
		return
	}
	h.cleanup(r.Context(), &d)
	web.WriteJSON(w, 200, d)
}
func (h *Handler) cleanup(ctx context.Context, d *Draft) {
	if h.store == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancel()
	if h.repo.DeletePhoto(ctx, *d, h.store) == nil && d.PhotoStatus == "pending_delete" {
		d.PhotoStatus = "deleted"
	}
}

// RunCleanup reads only owner IDs under the operator connection; every draft
// operation subsequently executes under that owner's restricted RLS role.
func (h *Handler) RunCleanup(ctx context.Context, p *pgxpool.Pool) {
	if h.store == nil {
		return
	}
	sweep := func() {
		bounded, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()
		rows, err := p.Query(bounded, "SELECT DISTINCT owner_id::text FROM receipt_drafts WHERE photo_status<>'deleted'")
		if err != nil {
			return
		}
		var owners []string
		for rows.Next() {
			var id string
			if rows.Scan(&id) == nil {
				owners = append(owners, id)
			}
		}
		rows.Close()
		for _, id := range owners {
			_ = h.repo.Sweep(access.WithPrincipal(bounded, access.Principal{ID: id, Role: "member"}), h.store)
		}
	}
	sweep()
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			sweep()
		}
	}
}
