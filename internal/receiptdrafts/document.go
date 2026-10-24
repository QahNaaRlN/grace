package receiptdrafts

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/QahNaaRlN/grace/internal/documentdata"
	"github.com/QahNaaRlN/grace/internal/pdfextract"
	"github.com/QahNaaRlN/grace/internal/platform/web"
	"github.com/go-chi/chi/v5"
	"io"
	"mime"
	"net/http"
	"strconv"
	"time"
)

type PDFRenderer interface {
	Render(context.Context, []byte) (pdfextract.Rendered, error)
}
type ReadingPlan struct {
	Revision          int64               `json:"revision"`
	SourceSHA256      string              `json:"sourceSHA256"`
	Pages             []documentdata.Page `json:"pages"`
	Model             string              `json:"model"`
	Provider          string              `json:"provider"`
	Profile           string              `json:"profile"`
	GroupRevision     string              `json:"groupRevision"`
	MaximumMicros     int64               `json:"maximumMicros"`
	ConsentPolicyHash string              `json:"consentPolicyHash"`
	Hash              string              `json:"hash"`
}
type DocumentExtractor interface {
	PlanDocument(context.Context, Draft, documentdata.Document) (ReadingPlan, error)
	ExtractDocument(context.Context, Draft, documentdata.Document, string) (Extracted, error)
}
type ReadingInput struct {
	Revision    int64  `json:"revision"`
	PreviewHash string `json:"previewHash"`
	Consent     bool   `json:"consent"`
}

func (h *Handler) SetPDFRenderer(r PDFRenderer) { h.renderer = r }
func (s *PhotoStore) SaveDocument(doc documentdata.Document) (string, error) {
	if doc.Validate() != nil {
		return "", errors.New("invalid_image")
	}
	b, e := json.Marshal(doc)
	if e != nil || len(b) > documentdata.MaxBundleBytes {
		return "", errors.New("document_limit")
	}
	return s.save(b, documentdata.MaxBundleBytes)
}
func (s *PhotoStore) ReadDocument(d Draft) (documentdata.Document, error) {
	b, e := s.read(d.ImageID, documentdata.MaxBundleBytes)
	if e != nil {
		return documentdata.Document{}, e
	}
	var doc documentdata.Document
	if len(b) > 0 && b[0] == '{' {
		if json.Unmarshal(b, &doc) != nil || doc.SourceSHA256 != d.ImageHash || doc.Validate() != nil {
			return doc, errors.New("invalid_image")
		}
		return doc, nil
	}
	if documentdata.Hash(b) != d.ImageHash {
		return doc, errors.New("invalid_image")
	}
	p, e := documentdata.NewPage(1, b)
	if e != nil {
		return doc, e
	}
	doc = documentdata.Document{Kind: "image", SourceSHA256: d.ImageHash, Pages: []documentdata.Page{p}}
	return doc, doc.Validate()
}
func (h *Handler) UploadDocument(w http.ResponseWriter, r *http.Request) {
	if !h.enabled() {
		draftError(w, errors.New("ai_disabled"))
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 11<<20)
	if r.ParseMultipartForm(1<<20) != nil {
		draftError(w, errors.New("document_limit"))
		return
	}
	defer r.MultipartForm.RemoveAll()
	if len(r.MultipartForm.File) != 1 || len(r.MultipartForm.File["file"]) != 1 {
		draftError(w, errors.New("invalid_request"))
		return
	}
	file, header, e := r.FormFile("file")
	if e != nil {
		draftError(w, errors.New("invalid_request"))
		return
	}
	defer file.Close()
	raw, e := io.ReadAll(io.LimitReader(file, (10<<20)+1))
	if e != nil || len(raw) > 10<<20 {
		draftError(w, errors.New("document_limit"))
		return
	}
	detected := http.DetectContentType(raw)
	declared, _, _ := mime.ParseMediaType(header.Header.Get("Content-Type"))
	if declared != "" && declared != "application/octet-stream" && declared != detected {
		draftError(w, errors.New("invalid_request"))
		return
	}
	var doc documentdata.Document
	if detected == "application/pdf" {
		if h.renderer == nil {
			draftError(w, errors.New("extractor_unavailable"))
			return
		}
		rendered, err := h.renderer.Render(r.Context(), raw)
		e = err
		doc = rendered.Document
		doc.Original = raw
	} else {
		doc, e = documentdata.Photo(raw)
	}
	if e != nil {
		draftError(w, e)
		return
	}
	id, e := h.store.SaveDocument(doc)
	if e != nil {
		draftError(w, e)
		return
	}
	d, created, e := h.repo.createDocument(r.Context(), id, doc.SourceSHA256, doc.Kind, doc.Manifest().Pages)
	if e != nil || !created {
		_ = h.store.Delete(id)
	}
	if e != nil {
		draftError(w, e)
		return
	}
	status := 200
	if created {
		status = 201
	}
	web.WriteJSON(w, status, d)
}
func (h *Handler) sourceDocument(ctx context.Context, id string) (Draft, documentdata.Document, error) {
	d, e := h.repo.Get(ctx, id)
	if e != nil {
		return d, documentdata.Document{}, e
	}
	if d.PhotoStatus != "stored" || !time.Now().Before(d.ExpiresAt) {
		return d, documentdata.Document{}, errors.New("image_expired")
	}
	doc, e := h.store.ReadDocument(d)
	return d, doc, e
}
func (h *Handler) ReadingPreview(w http.ResponseWriter, r *http.Request) {
	reader, ok := h.extractor.(DocumentExtractor)
	if !ok {
		draftError(w, errors.New("ai_disabled"))
		return
	}
	d, doc, e := h.sourceDocument(r.Context(), chi.URLParam(r, "draftId"))
	if e != nil {
		draftError(w, e)
		return
	}
	plan, e := reader.PlanDocument(r.Context(), d, doc)
	if e != nil {
		draftError(w, e)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	web.WriteJSON(w, 200, plan)
}
func (h *Handler) Page(w http.ResponseWriter, r *http.Request) {
	_, doc, e := h.sourceDocument(r.Context(), chi.URLParam(r, "draftId"))
	if e != nil {
		draftError(w, e)
		return
	}
	page, e := strconv.Atoi(chi.URLParam(r, "page"))
	if e != nil || page < 1 || page > len(doc.Pages) {
		web.WriteError(w, 404, "not_found", "Страница недоступна.")
		return
	}
	p := doc.Pages[page-1]
	w.Header().Set("Cache-Control", "private, no-store")
	w.Header().Set("Content-Type", p.ContentType)
	w.Header().Set("X-Content-Type-Options", "nosniff")
	_, _ = w.Write(p.Data)
}
func (h *Handler) extractDocument(w http.ResponseWriter, r *http.Request, key string, reader DocumentExtractor) {
	var in ReadingInput
	if !web.DecodeJSON(w, r, &in) {
		return
	}
	if !in.Consent || in.PreviewHash == "" {
		draftError(w, errors.New("consent_required"))
		return
	}
	id := chi.URLParam(r, "draftId")
	d, e := h.repo.Get(r.Context(), id)
	if e != nil {
		draftError(w, e)
		return
	}
	if d.ExtractionKey == key && d.Status != "extracting" {
		if d.ReadingConsent == nil || d.ReadingConsent.Hash != in.PreviewHash || d.ReadingConsent.Revision != in.Revision {
			draftError(w, errors.New("idempotency_conflict"))
			return
		}
		web.WriteJSON(w, 200, d)
		return
	}
	d, doc, e := h.sourceDocument(r.Context(), id)
	if e != nil {
		draftError(w, e)
		return
	}
	plan, e := reader.PlanDocument(r.Context(), d, doc)
	if e != nil {
		draftError(w, e)
		return
	}
	if plan.Hash != in.PreviewHash || d.Revision != in.Revision {
		draftError(w, errors.New("stale_revision"))
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 210*time.Second)
	defer cancel()
	_ = http.NewResponseController(w).SetWriteDeadline(time.Now().Add(215 * time.Second))
	d, run, e := h.repo.beginDocumentExtraction(ctx, id, key, &plan)
	if e != nil {
		draftError(w, e)
		return
	}
	if !run {
		web.WriteJSON(w, 200, d)
		return
	}
	out, e := reader.ExtractDocument(ctx, d, doc, plan.Hash)
	var result *Extracted
	if e == nil {
		result = &out
	}
	final, finishErr := h.repo.FinishExtraction(ctx, id, key, result)
	if finishErr != nil {
		draftError(w, finishErr)
		return
	}
	if e != nil {
		draftError(w, e)
		return
	}
	web.WriteJSON(w, 200, final)
}
