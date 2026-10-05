package auth

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRS256KeyFilteringFailsClosed(t *testing.T) {
	for _, body := range []string{
		`not json`,
		`{"keys":[]}`,
		`{"keys":[{"alg":"ES256K","kty":"EC","crv":"secp256k1"}]}`,
		`{"keys":[{"alg":"RS256","kty":"RSA","use":"enc"}]}`,
		`{"keys":[{"alg":"RS256","kty":"RSA","key_ops":["encrypt"]}]}`,
		`{"keys":[{"alg":"RS256","kty":"RSA","key_ops":[]}]}`,
		strings.Repeat("x", (1<<20)+1),
	} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = io.WriteString(w, body) }))
		req, _ := http.NewRequest("GET", server.URL, nil)
		_, err := (rs256KeyTransport{base: http.DefaultTransport, url: server.URL}).RoundTrip(req)
		server.Close()
		if err == nil {
			t.Fatal("invalid or ineligible key set accepted")
		}
	}
}

func TestRS256KeyFilteringLeavesHTTPFailuresUntouched(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(503)
		_, _ = io.WriteString(w, "unavailable")
	}))
	defer server.Close()
	req, _ := http.NewRequest("GET", server.URL, nil)
	resp, err := (rs256KeyTransport{base: http.DefaultTransport, url: server.URL}).RoundTrip(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 503 {
		t.Fatal("provider failure hidden")
	}
}
