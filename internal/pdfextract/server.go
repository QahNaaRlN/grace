// Package pdfextract runs only in the isolated Poppler helper. The API uses a
// Unix-socket client and never executes PDF programs in its own container.
package pdfextract

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/QahNaaRlN/grace/internal/statementdata"
)

const Binary = "/usr/bin/pdftotext"

type Runner interface {
	Run(context.Context, string) ([]byte, error)
}
type Server struct {
	runner Runner
	busy   chan struct{}
}

func NewServer(r Runner) *Server { return &Server{runner: r, busy: make(chan struct{}, 1)} }
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	fail := func(status int, code string) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(map[string]string{"code": code})
	}
	if r.Method != "POST" || r.URL.Path != "/extract" {
		fail(404, "not_found")
		return
	}
	if r.Header.Get("Content-Type") != "application/pdf" {
		fail(415, "pdf_required")
		return
	}
	select {
	case s.busy <- struct{}{}:
		defer func() { <-s.busy }()
	default:
		w.Header().Set("Retry-After", "2")
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
	if len(raw) < 5 || !bytes.HasPrefix(raw, []byte("%PDF-")) {
		fail(422, "invalid_pdf")
		return
	}
	dir, e := os.MkdirTemp("", "grace-pdf-")
	if e != nil {
		fail(503, "extractor_unavailable")
		return
	}
	defer os.RemoveAll(dir)
	path := dir + "/source.pdf"
	if e = os.WriteFile(path, raw, 0600); e != nil {
		fail(503, "extractor_unavailable")
		return
	}
	output, e := s.runner.Run(ctx, path)
	if e != nil {
		if ctx.Err() != nil {
			fail(422, "pdf_extraction_timeout")
		} else {
			fail(422, "unsupported_pdf")
		}
		return
	}
	doc, e := statementdata.DecodeLayout(bytes.NewReader(output))
	if e != nil {
		fail(422, e.Error())
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(doc)
}

var ErrOutputLimit = errors.New("pdf_output_limit")

type limitedBuffer struct {
	buffer bytes.Buffer
	limit  int
}

func (b *limitedBuffer) Write(p []byte) (int, error) {
	if len(p) > b.limit-b.buffer.Len() {
		return 0, ErrOutputLimit
	}
	return b.buffer.Write(p)
}

func (b *limitedBuffer) Bytes() []byte { return b.buffer.Bytes() }
