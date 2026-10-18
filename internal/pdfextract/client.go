package pdfextract

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"path/filepath"
	"time"

	"github.com/QahNaaRlN/grace/internal/statementdata"
)

type Client struct{ client *http.Client }

func NewClient(socket string) (*Client, error) {
	if !filepath.IsAbs(socket) {
		return nil, errors.New("absolute_extractor_socket_required")
	}
	transport := &http.Transport{Proxy: nil, DisableKeepAlives: true, DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, "unix", socket)
	}}
	return &Client{&http.Client{Transport: transport, Timeout: 32 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}}, nil
}
func (c *Client) Extract(ctx context.Context, raw []byte) (statementdata.Document, error) {
	if len(raw) > statementdata.MaxPDFBytes {
		return statementdata.Document{}, errors.New("pdf_size_limit")
	}
	req, e := http.NewRequestWithContext(ctx, "POST", "http://pdf-extractor/extract", bytes.NewReader(raw))
	if e != nil {
		return statementdata.Document{}, e
	}
	req.Header.Set("Content-Type", "application/pdf")
	resp, e := c.client.Do(req)
	if e != nil {
		return statementdata.Document{}, errors.New("extractor_unavailable")
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		if resp.StatusCode == 429 {
			return statementdata.Document{}, errors.New("extraction_busy")
		}
		var failure struct {
			Code string `json:"code"`
		}
		_ = json.NewDecoder(io.LimitReader(resp.Body, 1024)).Decode(&failure)
		switch failure.Code {
		case "pdf_size_limit", "invalid_pdf", "unsupported_pdf", "pdf_extraction_timeout", "pdf_text_limit", "pdf_page_limit", "pdf_span_limit", "pdf_text_required", "invalid_pdf_layout":
			return statementdata.Document{}, errors.New(failure.Code)
		}
		return statementdata.Document{}, errors.New("extractor_unavailable")
	}
	// JSON includes locators and geometry, so its bound is larger than text alone.
	limited := &io.LimitedReader{R: resp.Body, N: 64<<20 + 1}
	dec := json.NewDecoder(limited)
	dec.DisallowUnknownFields()
	var d statementdata.Document
	if e = dec.Decode(&d); e != nil || limited.N <= 0 {
		return statementdata.Document{}, errors.New("invalid_pdf_layout")
	}
	if e = dec.Decode(&struct{}{}); e != io.EOF {
		return statementdata.Document{}, errors.New("invalid_pdf_layout")
	}
	return d, d.Validate()
}
