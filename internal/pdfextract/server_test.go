package pdfextract

import (
	"context"
	"io"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/QahNaaRlN/grace/internal/statementdata"
)

type fakeRunner func(context.Context, string) ([]byte, error)

func (f fakeRunner) Run(c context.Context, p string) ([]byte, error) { return f(c, p) }

const validXML = `<doc><page width="600" height="800"><block><line><word xMin="1" yMin="2" xMax="50" yMax="12">Synthetic</word></line></block></page></doc>`

func TestHelperPrivateFileAndCleanup(t *testing.T) {
	var path string
	server := NewServer(fakeRunner(func(_ context.Context, p string) ([]byte, error) {
		path = p
		info, e := os.Stat(p)
		if e != nil || info.Mode().Perm() != 0600 {
			t.Fatal(info, e)
		}
		raw, _ := os.ReadFile(p)
		if string(raw) != "%PDF-synthetic" {
			t.Fatal("wrong input")
		}
		return []byte(validXML), nil
	}))
	req := httptest.NewRequest("POST", "/extract", strings.NewReader("%PDF-synthetic"))
	req.Header.Set("Content-Type", "application/pdf")
	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)
	if w.Code != 200 || w.Header().Get("Cache-Control") != "no-store" {
		t.Fatal(w.Code, w.Body.String())
	}
	if _, e := os.Stat(filepath.Dir(path)); !os.IsNotExist(e) {
		t.Fatal("private files retained", e)
	}
}
func TestHelperBusyAndCancellation(t *testing.T) {
	started := make(chan struct{})
	var once sync.Once
	s := NewServer(fakeRunner(func(ctx context.Context, _ string) ([]byte, error) {
		once.Do(func() { close(started) })
		<-ctx.Done()
		return nil, ctx.Err()
	}))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan struct{})
	request := func() *httptest.ResponseRecorder {
		r := httptest.NewRequest("POST", "/extract", strings.NewReader("%PDF-synthetic")).WithContext(ctx)
		r.Header.Set("Content-Type", "application/pdf")
		w := httptest.NewRecorder()
		s.ServeHTTP(w, r)
		return w
	}
	go func() { defer close(done); request() }()
	<-started
	w := request()
	if w.Code != 429 || w.Header().Get("Retry-After") == "" {
		t.Fatal(w)
	}
	cancel()
	<-done
}
func TestHelperRejectsOversizeAndInvalid(t *testing.T) {
	called := false
	s := NewServer(fakeRunner(func(context.Context, string) ([]byte, error) { called = true; return nil, nil }))
	for _, raw := range []string{"not PDF", "%PDF-" + strings.Repeat("a", statementdata.MaxPDFBytes)} {
		r := httptest.NewRequest("POST", "/extract", strings.NewReader(raw))
		r.Header.Set("Content-Type", "application/pdf")
		w := httptest.NewRecorder()
		s.ServeHTTP(w, r)
		if w.Code == 200 {
			t.Fatal("accepted invalid PDF")
		}
	}
	if called {
		t.Fatal("executed parser for invalid input")
	}
	b := &limitedBuffer{limit: 2}
	if _, e := io.WriteString(b, "123"); e == nil {
		t.Fatal("unbounded parser output")
	}
}
