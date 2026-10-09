package assistant

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestOpenRouterRejectsMissingOrChangedRouteEvidence(t *testing.T) {
	for _, kind := range []string{"missing", "fallback", "plugin", "provider", "multiple"} {
		t.Run(kind, func(t *testing.T) {
			selected := []map[string]any{{"provider": "fixture", "model": "fixture/model-v1", "selected": true}}
			metadata := map[string]any{"requested": "fixture/model-v1", "strategy": "direct", "attempt": 1, "is_byok": false, "endpoints": map[string]any{"available": selected}, "pipeline": []any{}}
			switch kind {
			case "fallback":
				metadata["attempt"] = 2
			case "plugin":
				metadata["pipeline"] = []any{map[string]any{"type": "server_tools"}}
			case "provider":
				selected[0]["provider"] = "other"
			case "multiple":
				metadata["endpoints"] = map[string]any{"available": append(selected, selected[0])}
			}
			response := map[string]any{"id": "synthetic-call", "model": "fixture/model-v1", "choices": []any{map[string]any{"finish_reason": "stop", "message": map[string]any{"content": "{}"}}}, "usage": map[string]any{"prompt_tokens": 100, "completion_tokens": 1, "cost": 0.0001}}
			if kind != "missing" {
				response["openrouter_metadata"] = metadata
			}
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { json.NewEncoder(w).Encode(response) }))
			defer server.Close()
			provider, _ := NewOpenRouterProvider("synthetic-key", server.Client())
			provider.endpoint = server.URL
			_, err := provider.Complete(context.Background(), LLMRequest{Policy: policyFixture(), Messages: []LLMMessage{{Role: "user", Content: "Synthetic"}}, Schema: json.RawMessage(`{}`), InputTokenUpperBound: 1000, MaxOutputTokens: 100})
			if err == nil {
				t.Fatal("unverified route accepted", kind)
			}
		})
	}
}
