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
