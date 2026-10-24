package statementextract

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"regexp"
	"time"

	"github.com/QahNaaRlN/grace/internal/imports"
	"github.com/QahNaaRlN/grace/internal/platform/web"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

var identifier = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

type Handler struct{ service *Service }

func NewHandler(s *Service) *Handler { return &Handler{s} }
func (h *Handler) Routes(r chi.Router) {
	r.Post("/statements/{statementId}/extraction-previews", h.preview)
	r.Post("/statements/{statementId}/extraction-consents", h.consent)
	r.Post("/statements/{statementId}/extractions", h.extract)
	r.Get("/statements/{statementId}/extractions", h.attempts)
	r.Get("/statements/{statementId}/extractions/{attemptId}", h.attempt)
}
func decode(w http.ResponseWriter, r *http.Request, v any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 256<<10)
	d := json.NewDecoder(r.Body)
	d.DisallowUnknownFields()
	if e := d.Decode(v); e != nil {
		reply(w, nil, fault("invalid_request", 400))
		return false
	}
	if e := d.Decode(&struct{}{}); e != io.EOF {
		reply(w, nil, fault("invalid_request", 400))
		return false
	}
	return true
}
func path(w http.ResponseWriter, r *http.Request, name string) (string, bool) {
	id := chi.URLParam(r, name)
	if !identifier.MatchString(id) {
		reply(w, nil, fault("invalid_reference", 400))
		return "", false
	}
	return id, true
}
func reply(w http.ResponseWriter, v any, e error) {
	w.Header().Set("Cache-Control", "no-store")
	if e == nil {
		web.WriteJSON(w, 200, v)
		return
	}
	code, status := "internal_error", 500
	var f *Fault
	var importFault *imports.Fault
	var db *pgconn.PgError
	if errors.As(e, &f) {
		code, status = f.Code, f.Status
	} else if errors.As(e, &importFault) {
		code, status = importFault.Code, importFault.Status
	} else if errors.As(e, &db) {
		switch db.Message {
		case "budget_exhausted", "ai_usage_requires_review", "ai_concurrency_limit":
			code, status = db.Message, 422
		}
	} else {
		switch e.Error() {
		case "document_limit", "incomplete_reading", "source_binding_failed":
			code, status = e.Error(), 422
		case "policy_changed":
			code, status = e.Error(), 409
		case "ai_disabled", "price_unknown", "provider_unavailable", "no_eligible_model", "capability_unverified":
			code, status = e.Error(), 503
		}
	}
	if status == 429 {
		w.Header().Set("Retry-After", "2")
	}
	message := "Распознавание не завершено. Проверьте согласие, состояние выписки и доступность AI; ручная проверка остаётся доступна."
	if code == "statement_atomic_block_limit" || code == "statement_document_span_limit" {
		message = "Документ слишком велик для AI. Загрузите выписку за меньший период или используйте ручную подготовку строк."
	}
	web.WriteError(w, status, code, message)
}
func (h *Handler) preview(w http.ResponseWriter, r *http.Request) {
	id, ok := path(w, r, "statementId")
	if !ok {
		return
	}
	var in PreviewInput
	if !decode(w, r, &in) {
		return
	}
	out, e := h.service.Preview(r.Context(), id, in)
	reply(w, out, e)
}
func (h *Handler) consent(w http.ResponseWriter, r *http.Request) {
	id, ok := path(w, r, "statementId")
	if !ok {
		return
	}
	var in ConsentInput
	if !decode(w, r, &in) {
		return
	}
	if !identifier.MatchString(in.PreviewID) {
		reply(w, nil, fault("invalid_reference", 400))
		return
	}
	out, e := h.service.Consent(r.Context(), id, r.Header.Get("Idempotency-Key"), in)
	reply(w, out, e)
}
func (h *Handler) extract(w http.ResponseWriter, r *http.Request) {
	id, ok := path(w, r, "statementId")
	if !ok {
		return
	}
	var in ExtractionInput
	if !decode(w, r, &in) {
		return
	}
	if !identifier.MatchString(in.PreviewID) || !identifier.MatchString(in.ConsentID) {
		reply(w, nil, fault("invalid_reference", 400))
		return
	}
	if e := http.NewResponseController(w).SetWriteDeadline(time.Now().Add(215 * time.Second)); e != nil {
		reply(w, nil, fault("statement_deadline_unavailable", 503))
		return
	}
	out, e := h.service.Extract(r.Context(), id, r.Header.Get("Idempotency-Key"), in)
	reply(w, out, e)
}
func (h *Handler) attempt(w http.ResponseWriter, r *http.Request) {
	id, ok := path(w, r, "statementId")
	if !ok {
		return
	}
	attempt, ok := path(w, r, "attemptId")
	if !ok {
		return
	}
	out, e := h.service.Attempt(r.Context(), id, attempt)
	reply(w, out, e)
}

func (h *Handler) attempts(w http.ResponseWriter, r *http.Request) {
	id, ok := path(w, r, "statementId")
	if !ok {
		return
	}
	items, e := h.service.Attempts(r.Context(), id)
	reply(w, map[string]any{"items": items}, e)
}
