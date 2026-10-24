package pdfextract

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"github.com/QahNaaRlN/grace/internal/documentdata"
	"github.com/QahNaaRlN/grace/internal/statementdata"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

type Rendered struct {
	Document documentdata.Document   `json:"document"`
	Layout   *statementdata.Document `json:"layout"`
}
type Renderer interface {
	Render(context.Context, string) ([][]byte, error)
}

func (c *Client) Render(ctx context.Context, raw []byte) (Rendered, error) {
	var out Rendered
	if len(raw) > statementdata.MaxPDFBytes {
		return out, errors.New("pdf_size_limit")
	}
	req, e := http.NewRequestWithContext(ctx, "POST", "http://pdf-extractor/render", bytes.NewReader(raw))
	if e != nil {
		return out, e
	}
	req.Header.Set("Content-Type", "application/pdf")
	res, e := c.client.Do(req)
	if e != nil {
		return out, errors.New("extractor_unavailable")
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		var f struct {
			Code string `json:"code"`
		}
		_ = json.NewDecoder(io.LimitReader(res.Body, 1024)).Decode(&f)
		switch f.Code {
		case "document_limit", "unsupported_pdf", "pdf_extraction_timeout", "extraction_busy", "pdf_size_limit", "invalid_pdf":
			return out, errors.New(f.Code)
		}
		return out, errors.New("extractor_unavailable")
	}
	limited := &io.LimitedReader{R: res.Body, N: 14 << 20}
	dec := json.NewDecoder(limited)
	dec.DisallowUnknownFields()
	if dec.Decode(&out) != nil || limited.N <= 0 || dec.Decode(&struct{}{}) != io.EOF || out.Document.SourceSHA256 != documentdata.Hash(raw) || out.Document.Kind != "pdf" || out.Document.Validate() != nil {
		return Rendered{}, errors.New("invalid_pdf_layout")
	}
	if out.Layout != nil && out.Layout.ValidateVisualLayout() != nil {
		return Rendered{}, errors.New("invalid_pdf_layout")
	}
	return out, nil
}
func (s *Server) render(w http.ResponseWriter, r *http.Request) {
	fail := func(status int, code string) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(map[string]string{"code": code})
	}
	if r.Method != "POST" || r.Header.Get("Content-Type") != "application/pdf" {
		fail(415, "pdf_required")
		return
	}
	renderer, ok := s.runner.(Renderer)
	if !ok {
		fail(503, "extractor_unavailable")
		return
	}
	select {
	case s.busy <- struct{}{}:
		defer func() { <-s.busy }()
	default:
		fail(429, "extraction_busy")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()
	raw, e := io.ReadAll(http.MaxBytesReader(w, r.Body, statementdata.MaxPDFBytes))
	if e != nil {
		fail(413, "pdf_size_limit")
		return
	}
	if !bytes.HasPrefix(raw, []byte("%PDF-")) {
		fail(422, "invalid_pdf")
		return
	}
	dir, e := os.MkdirTemp("", "grace-render-")
	if e != nil {
		fail(503, "extractor_unavailable")
		return
	}
	defer os.RemoveAll(dir)
	path := dir + "/source.pdf"
	if os.WriteFile(path, raw, 0600) != nil {
		fail(503, "extractor_unavailable")
		return
	}
	pages, e := renderer.Render(ctx, path)
	if e != nil {
		if ctx.Err() != nil {
			fail(422, "pdf_extraction_timeout")
		} else if e.Error() == "document_limit" {
			fail(422, "document_limit")
		} else {
			fail(422, "unsupported_pdf")
		}
		return
	}
	out := Rendered{Document: documentdata.Document{Kind: "pdf", SourceSHA256: documentdata.Hash(raw), Pages: []documentdata.Page{}}}
	for i, b := range pages {
		p, e := documentdata.NewPage(i+1, b)
		if e != nil {
			fail(422, "invalid_pdf_layout")
			return
		}
		out.Document.Pages = append(out.Document.Pages, p)
	}
	text, e := s.runner.Run(ctx, path)
	if e != nil {
		fail(422, "unsupported_pdf")
		return
	}
	layout, e := statementdata.DecodeVisualLayout(bytes.NewReader(text))
	if e == nil {
		if len(layout.Spans) > 0 {
			out.Layout = &layout
		}
		parts := []string{}
		for _, span := range layout.Spans {
			parts = append(parts, span.Text)
		}
		out.Document.Text = strings.Join(parts, " ")
	} else if e.Error() != "pdf_text_required" {
		fail(422, e.Error())
		return
	}
	if out.Document.Validate() != nil {
		fail(422, "document_limit")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(out)
}
