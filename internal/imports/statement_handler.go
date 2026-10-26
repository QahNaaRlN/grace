package imports

import (
	"encoding/json"
	"io"
	"mime"
	"net/http"
	"strconv"
	"time"

	"github.com/QahNaaRlN/grace/internal/platform/web"
	"github.com/QahNaaRlN/grace/internal/statementdata"
	"github.com/go-chi/chi/v5"
)

func (h *Handler) statementRoutes(r chi.Router) {
	r.Post("/statements/pdf", h.UploadStatement)
	r.Post("/statements/images", h.UploadStatementImage)
	r.Get("/statements/{statementId}/pages/{page}", h.StatementPage)
	r.Post("/statements/{statementId}/visual-reviews", h.VisualReview)
	r.Post("/statements/{statementId}/plain-reviews", h.PlainReview)
	r.Get("/statements/{statementId}", h.Statement)
	r.Patch("/statements/{statementId}", h.EditStatement)
	r.Post("/statements/{statementId}/manual-drafts", h.ManualStatementDraft)
	r.Get("/statements/{statementId}/original", h.StatementOriginal)
	r.Delete("/statements/{statementId}/original", h.DeleteStatementOriginal)
	r.Get("/batches/{batchId}/statement/reconciliation", h.StatementReconciliation)
}
func (h *Handler) UploadStatement(w http.ResponseWriter, r *http.Request) {
	h.uploadStatementSource(w, r, false)
}
func (h *Handler) UploadStatementImage(w http.ResponseWriter, r *http.Request) {
	h.uploadStatementSource(w, r, true)
}
func (h *Handler) uploadStatementSource(w http.ResponseWriter, r *http.Request, image bool) {
	_ = http.NewResponseController(w).SetWriteDeadline(time.Now().Add(35 * time.Second))
	r.Body = http.MaxBytesReader(w, r.Body, 11<<20)
	if e := r.ParseMultipartForm(1 << 20); e != nil {
		web.WriteError(w, 413, "invalid_pdf_upload", "Выберите PDF до 10 MiB.")
		return
	}
	if r.MultipartForm != nil {
		defer r.MultipartForm.RemoveAll()
	}
	if len(r.MultipartForm.File) != 1 || len(r.MultipartForm.File["file"]) != 1 || (len(r.MultipartForm.Value) != 1 && !(len(r.MultipartForm.Value) == 2 && len(r.MultipartForm.Value["readingMode"]) == 1)) || len(r.MultipartForm.Value["accountId"]) != 1 {
		web.WriteError(w, 400, "invalid_pdf_upload", "Ожидаются один PDF и счёт.")
		return
	}
	file, header, e := r.FormFile("file")
	if e != nil {
		web.WriteError(w, 400, "pdf_required", "Выберите PDF.")
		return
	}
	defer file.Close()
	raw, e := io.ReadAll(io.LimitReader(file, statementdata.MaxPDFBytes+1))
	if e != nil || len(raw) > statementdata.MaxPDFBytes {
		web.WriteError(w, 413, "pdf_size_limit", "PDF не должен превышать 10 MiB.")
		return
	}
	upload := h.repo.UploadStatement
	if image {
		upload = h.repo.UploadStatementImage
	}
	detected := http.DetectContentType(raw)
	declared, _, _ := mime.ParseMediaType(header.Header.Get("Content-Type"))
	if declared != "" && declared != "application/octet-stream" && declared != detected {
		web.WriteError(w, 415, "invalid_request", "Тип файла не соответствует содержимому.")
		return
	}
	s, created, e := upload(r.Context(), raw, r.FormValue("accountId"), header.Filename, key(r), r.FormValue("readingMode"))
	result := StatementUploadResult{Statement: s, Document: StatementDocument{ID: s.DocumentID, FileName: s.FileName, SHA256: s.SHA256, SizeBytes: s.SizeBytes, ContentType: detected}, BatchID: s.BatchID}
	if e == nil && created {
		web.WriteJSON(w, 201, result)
		return
	}
	if asFault(e).Status == 429 {
		w.Header().Set("Retry-After", "2")
	}
	respond(w, result, e)
}
func (h *Handler) Statement(w http.ResponseWriter, r *http.Request) {
	id, ok := path(w, r, "statementId")
	if !ok {
		return
	}
	s, e := h.repo.Statement(r.Context(), id)
	w.Header().Set("Cache-Control", "no-store")
	respond(w, s, e)
}
func (h *Handler) EditStatement(w http.ResponseWriter, r *http.Request) {
	id, ok := path(w, r, "statementId")
	if !ok {
		return
	}
	var in StatementEdit
	if !decode(w, r, &in) {
		return
	}
	s, e := h.repo.EditStatement(r.Context(), id, key(r), in)
	respond(w, s, e)
}
func (h *Handler) DeleteStatementOriginal(w http.ResponseWriter, r *http.Request) {
	id, ok := path(w, r, "statementId")
	if !ok {
		return
	}
	var in StatementDelete
	if !decode(w, r, &in) {
		return
	}
	s, e := h.repo.DeleteStatementOriginal(r.Context(), id, key(r), in)
	respond(w, s, e)
}
func (h *Handler) StatementOriginal(w http.ResponseWriter, r *http.Request) {
	id, ok := path(w, r, "statementId")
	if !ok {
		return
	}
	raw, name, e := h.repo.StatementOriginal(r.Context(), id)
	if e != nil {
		respond(w, nil, e)
		return
	}
	w.Header().Set("Content-Type", http.DetectContentType(raw))
	w.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": name}))
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	_, _ = w.Write(raw)
}
func (h *Handler) ManualStatementDraft(w http.ResponseWriter, r *http.Request) {
	id, ok := path(w, r, "statementId")
	if !ok {
		return
	}
	// A proposal can cover up to 200k locators; it contains no PDF bytes. Ordinary
	// review edits retain their existing smaller request bound.
	r.Body = http.MaxBytesReader(w, r.Body, 16<<20)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	var in StatementManualDraft
	if e := dec.Decode(&in); e != nil {
		web.WriteError(w, 400, "invalid_request", "Некорректная схема ссылок на исходник.")
		return
	}
	if e := dec.Decode(&struct{}{}); e != io.EOF {
		web.WriteError(w, 400, "invalid_request", "Ожидается один объект.")
		return
	}
	s, e := h.repo.ManualStatementDraft(r.Context(), id, key(r), in)
	respond(w, s, e)
}
func (h *Handler) StatementReconciliation(w http.ResponseWriter, r *http.Request) {
	batch, ok := path(w, r, "batchId")
	if !ok {
		return
	}
	tx, e := h.repo.begin(r.Context(), false)
	if e != nil {
		respond(w, nil, e)
		return
	}
	defer tx.Rollback(r.Context())
	var id string
	if e = tx.QueryRow(r.Context(), "SELECT id::text FROM statement_sources WHERE batch_id=$1", batch).Scan(&id); e != nil {
		respond(w, nil, ErrNotFound)
		return
	}
	s, e := loadStatement(r.Context(), tx, id, false)
	if e != nil {
		respond(w, nil, e)
		return
	}
	applyStatementFlags(&s)
	if s.Review == nil {
		respond(w, nil, fault("statement_review_required", 422))
		return
	}
	respond(w, s.Review.Reconciliation, nil)
}

func (h *Handler) StatementPage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "private, no-store")
	id, ok := path(w, r, "statementId")
	if !ok {
		return
	}
	n, e := strconv.Atoi(chi.URLParam(r, "page"))
	if e != nil {
		respond(w, nil, fault("not_found", 404))
		return
	}
	page, e := h.repo.StatementPage(r.Context(), id, n)
	if e != nil {
		respond(w, nil, e)
		return
	}
	w.Header().Set("Cache-Control", "private, no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Type", page.ContentType)
	_, _ = w.Write(page.Data)
}
func (h *Handler) VisualReview(w http.ResponseWriter, r *http.Request) {
	id, ok := path(w, r, "statementId")
	if !ok {
		return
	}
	var in VisualReviewInput
	if !decode(w, r, &in) {
		return
	}
	out, e := h.repo.ReviewVisual(r.Context(), id, key(r), in)
	respond(w, out, e)
}
