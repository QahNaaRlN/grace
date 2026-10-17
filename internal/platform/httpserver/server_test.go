package httpserver

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QahNaaRlN/grace/internal/platform/config"
)

func TestAssistantEndpointIsUnavailableWithoutExplicitOptInAndKey(t *testing.T) {
	for _, cfg := range []config.Config{{AuthMode: "development"}, {AuthMode: "development", AssistantRoutingEnabled: true}} {
		server := New(cfg, nil)
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodPost, "/api/v1/assistant/query", bytes.NewBufferString(`{"utterance":"Покажи счета","language":"ru","referenceTime":"2026-10-05T12:00:00+05:00"}`))
		server.Handler.ServeHTTP(recorder, request)
		if recorder.Code != http.StatusServiceUnavailable || !bytes.Contains(recorder.Body.Bytes(), []byte(`"errorCode":"assistant_unavailable"`)) {
			t.Fatalf("unexpected response: %d %s", recorder.Code, recorder.Body.String())
		}
	}
}

func TestInstallmentPutPreflightOnlyAllowsConfiguredOrigin(t *testing.T) {
	server := New(config.Config{CORSOrigins: []string{"http://localhost:5174"}}, nil)
	for _, origin := range []string{"http://localhost:5174", "https://other.invalid"} {
		request := httptest.NewRequest(http.MethodOptions, "/api/v1/installment-plans/plan/schedule-drafts/draft", nil)
		request.Header.Set("Origin", origin)
		request.Header.Set("Access-Control-Request-Method", "PUT")
		request.Header.Set("Access-Control-Request-Headers", "Content-Type, Idempotency-Key, X-CSRF-Token")
		response := httptest.NewRecorder()
		server.Handler.ServeHTTP(response, request)
		allowed := response.Header().Get("Access-Control-Allow-Methods") == "PUT" && response.Header().Get("Access-Control-Allow-Origin") == origin
		if allowed != (origin == "http://localhost:5174") {
			t.Fatalf("unexpected CORS preflight for %s: %v", origin, response.Header())
		}
	}
}
