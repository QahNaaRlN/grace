package imports

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/QahNaaRlN/grace/internal/importreview"
	"github.com/QahNaaRlN/grace/internal/platform/web"
	"github.com/go-chi/chi/v5"
)

type Handler struct{ repo *Repository }

func NewHandler(repo *Repository) *Handler { return &Handler{repo} }
func (h *Handler) Routes(r chi.Router) {
	r.Post("/batches", h.Upload)
	r.Post("/csv", h.UploadCSV)
	r.Get("/batches", h.Batches)
	r.Get("/batches/{batchId}", h.Batch)
	r.Get("/batches/{batchId}/month", h.Month)
	r.Get("/batches/{batchId}/records", h.Records)
	r.Get("/batches/{batchId}/records/{recordId}", h.Record)
	r.Patch("/batches/{batchId}/records/{recordId}", h.Edit)
	r.Post("/batches/{batchId}/previews", h.Preview)
	r.Post("/batches/{batchId}/confirmations", h.Confirm)
	r.Get("/batches/{batchId}/confirmations/{confirmationId}", h.Receipt)
}
func respond(w http.ResponseWriter, value any, err error) {
	if err != nil {
		f := asFault(err)
		web.WriteError(w, f.Status, f.Code, "Не удалось выполнить действие импорта. Проверьте данные и обновите черновик.")
		return
	}
	web.WriteJSON(w, 200, value)
}
func path(w http.ResponseWriter, r *http.Request, name string) (string, bool) {
	id := chi.URLParam(r, name)
	if !uuid.MatchString(id) {
		web.WriteError(w, 400, "invalid_reference", "Некорректный идентификатор.")
		return "", false
	}
	return id, true
}
func key(r *http.Request) string { return strings.TrimSpace(r.Header.Get("Idempotency-Key")) }
func pagination(w http.ResponseWriter, r *http.Request) (string, int, bool) {
	c := r.URL.Query().Get("cursor")
	limit := 100
	if l := r.URL.Query().Get("limit"); l != "" {
		n, e := strconv.Atoi(l)
		if e != nil || n < 1 || n > 100 {
			web.WriteError(w, 400, "invalid_pagination", "Размер страницы от 1 до 100.")
			return "", 0, false
		}
		limit = n
	}
	if c != "" && !uuid.MatchString(c) {
		web.WriteError(w, 400, "invalid_pagination", "Некорректный cursor.")
		return "", 0, false
	}
	return c, limit, true
}
func decode(w http.ResponseWriter, r *http.Request, v any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 256<<10)
	d := json.NewDecoder(r.Body)
	d.DisallowUnknownFields()
	if err := d.Decode(v); err != nil {
		web.WriteError(w, 400, "invalid_request", "Некорректный запрос.")
		return false
	}
	if err := d.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		web.WriteError(w, 400, "invalid_request", "Ожидается один JSON объект.")
		return false
	}
	return true
}
func (h *Handler) Upload(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, importreview.MaxBytes)
	raw, err := io.ReadAll(r.Body)
	if err != nil {
		web.WriteError(w, 413, "body_too_large", "Снимок не должен превышать 16 MiB.")
		return
	}
	b, err := h.repo.Upload(r.Context(), raw, key(r))
	if err == nil && b.created {
		web.WriteJSON(w, http.StatusCreated, b)
		return
	}
	respond(w, b, err)
}
func (h *Handler) Batches(w http.ResponseWriter, r *http.Request) {
	c, l, ok := pagination(w, r)
	if !ok {
		return
	}
	v, e := h.repo.Batches(r.Context(), c, l)
	respond(w, v, e)
}

func (h *Handler) UploadCSV(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, importreview.MaxBytes)
	raw, err := io.ReadAll(r.Body)
	if err != nil {
		web.WriteError(w, 413, "body_too_large", "Файл не должен превышать 16 MiB.")
		return
	}
	b, err := h.repo.UploadCSV(r.Context(), raw, r.URL.Query().Get("sourceId"), key(r))
	if err == nil && b.created {
		web.WriteJSON(w, http.StatusCreated, b)
		return
	}
	respond(w, b, err)
}

func (h *Handler) Month(w http.ResponseWriter, r *http.Request) {
	b, ok := path(w, r, "batchId")
	if !ok {
		return
	}
	v, err := h.repo.Month(r.Context(), b, r.URL.Query().Get("month"))
	respond(w, v, err)
}
func (h *Handler) Batch(w http.ResponseWriter, r *http.Request) {
	b, ok := path(w, r, "batchId")
	if !ok {
		return
	}
	v, e := h.repo.Batch(r.Context(), b)
	respond(w, v, e)
}
func (h *Handler) Records(w http.ResponseWriter, r *http.Request) {
	b, ok := path(w, r, "batchId")
	if !ok {
		return
	}
	c, l, ok := pagination(w, r)
	if !ok {
		return
	}
	v, e := h.repo.Records(r.Context(), b, c, r.URL.Query().Get("domain"), r.URL.Query().Get("status"), l)
	respond(w, v, e)
}
func (h *Handler) Record(w http.ResponseWriter, r *http.Request) {
	b, ok := path(w, r, "batchId")
	if !ok {
		return
	}
	id, ok := path(w, r, "recordId")
	if !ok {
		return
	}
	v, e := h.repo.Record(r.Context(), b, id)
	respond(w, v, e)
}
func (h *Handler) Edit(w http.ResponseWriter, r *http.Request) {
	b, ok := path(w, r, "batchId")
	if !ok {
		return
	}
	id, ok := path(w, r, "recordId")
	if !ok {
		return
	}
	var input Edit
	if !decode(w, r, &input) {
		return
	}
	v, e := h.repo.Edit(r.Context(), b, id, key(r), input)
	respond(w, v, e)
}
func (h *Handler) Preview(w http.ResponseWriter, r *http.Request) {
	b, ok := path(w, r, "batchId")
	if !ok {
		return
	}
	var input Selection
	if !decode(w, r, &input) {
		return
	}
	v, e := h.repo.Preview(r.Context(), b, key(r), input)
	respond(w, v, e)
}
func (h *Handler) Confirm(w http.ResponseWriter, r *http.Request) {
	b, ok := path(w, r, "batchId")
	if !ok {
		return
	}
	var input Confirmation
	if !decode(w, r, &input) {
		return
	}
	v, e := h.repo.Confirm(r.Context(), b, key(r), input)
	respond(w, v, e)
}
func (h *Handler) Receipt(w http.ResponseWriter, r *http.Request) {
	b, ok := path(w, r, "batchId")
	if !ok {
		return
	}
	id, ok := path(w, r, "confirmationId")
	if !ok {
		return
	}
	v, e := h.repo.Receipt(r.Context(), b, id)
	respond(w, v, e)
}
