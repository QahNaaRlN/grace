package main

import (
	"bytes"
	"encoding/json"
	"github.com/QahNaaRlN/grace/internal/assistant"
	"github.com/QahNaaRlN/grace/internal/documentdata"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func fixture(t *testing.T) bundle {
	t.Helper()
	im := image.NewRGBA(image.Rect(0, 0, 32, 32))
	for y := 0; y < 32; y++ {
		for x := 0; x < 32; x++ {
			im.Set(x, y, color.White)
		}
	}
	for y := 8; y < 24; y++ {
		for x := 8; x < 24; x++ {
			im.Set(x, y, color.Black)
		}
	}
	var raw bytes.Buffer
	_ = png.Encode(&raw, im)
	b := bundle{Source: strings.Repeat("a", 40), DocumentSHA256: strings.Repeat("b", 64), Candidate: assistant.GroupMembership{RequestFeeEvidence: assistant.RequestFeeRouterCeilingOnly, MaxImages: 5, MaxImageSide: 2048, Canonical: "google/gemini-3.1-flash-lite-20260507", Region: "eu", Policy: assistant.ModelPolicy{Model: "google/gemini-3.1-flash-lite", Provider: "google-vertex/eu", ProviderName: "Google", InputMicrosPerMillion: 275000, OutputMicrosPerMillion: 1650000, RequestFeeMicros: 1000, ImageFeeMicros: 1, ImageTokenUpperBound: 4200, Vision: true}}}
	for i := 1; i <= 5; i++ {
		b.Pages = append(b.Pages, documentdata.Page{Number: i, Width: 32, Height: 32, ContentType: "image/png", Data: raw.Bytes(), SHA256: digest(raw.Bytes())})
	}
	return b
}
func TestCompletePrivateInputAndNoGolden(t *testing.T) {
	old := sourceSHA
	sourceSHA = strings.Repeat("a", 40)
	defer func() { sourceSHA = old }()
	b := fixture(t)
	path := filepath.Join(t.TempDir(), "input.json")
	write := func() {
		raw, _ := json.Marshal(b)
		if e := os.WriteFile(path, raw, 0600); e != nil {
			t.Fatal(e)
		}
	}
	write()
	got, _, e := load(path)
	if e != nil || len(got.Pages) != 5 {
		t.Fatal(e)
	}
	r := request(b.Candidate.Policy, b.Pages)
	raw, _ := json.Marshal(r.Messages)
	if bytes.Contains(raw, []byte("96")) || bytes.Contains(raw, []byte("expected")) || len(r.Messages[0].Content.([]any)) != 11 || len(r.Schema) != 0 {
		t.Fatal("leak or missing pages")
	}
	for _, change := range []func(){func() { b.Pages = b.Pages[:4] }, func() { b.Source = strings.Repeat("c", 40) }, func() { b.Pages[0].Number = 2 }, func() { b.Pages[0].SHA256 = strings.Repeat("0", 64) }, func() { b.Candidate.Policy.RequestFeeMicros++ }} {
		b = fixture(t)
		change()
		write()
		if _, _, e = load(path); e == nil {
			t.Fatal("invalid input accepted")
		}
	}
	b = fixture(t)
	white := image.NewRGBA(image.Rect(0, 0, 32, 32))
	for y := 0; y < 32; y++ {
		for x := 0; x < 32; x++ {
			white.Set(x, y, color.White)
		}
	}
	var pngbuf bytes.Buffer
	_ = png.Encode(&pngbuf, white)
	b.Pages[0].Data = pngbuf.Bytes()
	b.Pages[0].SHA256 = digest(pngbuf.Bytes())
	write()
	if _, _, e = load(path); e == nil {
		t.Fatal("blank image accepted")
	}
}
func TestTransportRefusesUnknownHostAndExcessCalls(t *testing.T) {
	h := &observedHTTP{post: 1, get: 3}
	for _, url := range []string{"https://example.test/api/v1/models", "https://openrouter.ai/api/v1/models", "https://openrouter.ai/api/v1/generation?id=other"} {
		r, _ := http.NewRequest("GET", url, nil)
		if _, e := h.Do(r); e == nil {
			t.Fatal("unexpected call")
		}
	}
	r, _ := http.NewRequest("POST", "https://openrouter.ai/api/v1/chat/completions", nil)
	if _, e := h.Do(r); e == nil {
		t.Fatal("second POST")
	}
}
