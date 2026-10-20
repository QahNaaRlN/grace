package assistant

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/QahNaaRlN/grace/internal/statementextract"
)

// These are synthetic counterexamples, not a replay of the lost live envelope.
// They establish which guards can emit the observed unsuffixed failure without
// accepting a different model/provider or making any external request.
func TestScreeningFailureDiagnosisSeparatesEnvelopeRouteAndContent(t *testing.T) {
	const model = "google/gemini-3.1-flash-lite"
	const canonical = "google/gemini-3.1-flash-lite-20260507"
	type object = map[string]any
	base := `{"id":"synthetic-diagnosis","model":"google/gemini-3.1-flash-lite","provider":"Google","openrouter_metadata":{"requested":"google/gemini-3.1-flash-lite","strategy":"direct","attempt":1,"is_byok":false,"pipeline":[],"endpoints":{"available":[{"model":"google/gemini-3.1-flash-lite","provider":"Google","selected":true}]}},"choices":[{"finish_reason":"stop","message":{"content":"{}"}}],"usage":{"prompt_tokens":100,"completion_tokens":10,"cost":0.001}}`
	tests := []struct {
		name, detail string
		known        bool
		mutate       func(object)
	}{
		{"valid", "", true, func(o object) {}},
		{"canonical_top_model", "envelope", false, func(o object) { o["model"] = canonical }},
		{"metadata_missing", "generic", false, func(o object) { delete(o, "openrouter_metadata") }},
		{"canonical_requested", "generic", false, func(o object) { o["openrouter_metadata"].(object)["requested"] = canonical }},
		{"canonical_selected_model", "generic", false, func(o object) {
			o["openrouter_metadata"].(object)["endpoints"].(object)["available"].([]any)[0].(object)["model"] = canonical
		}},
		{"changed_selected_provider", "generic", false, func(o object) {
			o["openrouter_metadata"].(object)["endpoints"].(object)["available"].([]any)[0].(object)["provider"] = "Google AI Studio"
		}},
		{"changed_display_provider", "generic", false, func(o object) { o["provider"] = "Google AI Studio" }},
		{"retry_attempt", "generic", false, func(o object) { o["openrouter_metadata"].(object)["attempt"] = 2 }},
		{"pipeline", "generic", false, func(o object) { o["openrouter_metadata"].(object)["pipeline"] = []any{object{"type": "synthetic"}} }},
		{"byok", "generic", false, func(o object) { o["openrouter_metadata"].(object)["is_byok"] = true }},
		{"no_selected_route", "generic", false, func(o object) { o["openrouter_metadata"].(object)["endpoints"].(object)["available"] = []any{} }},
		{"missing_cost", "generic", false, func(o object) { delete(o["usage"].(object), "cost") }},
		{"length", "finish_length", true, func(o object) { o["choices"].([]any)[0].(object)["finish_reason"] = "length" }},
		{"invalid_content", "content", true, func(o object) { o["choices"].([]any)[0].(object)["message"].(object)["content"] = "not JSON" }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var envelope object
			if e := json.Unmarshal([]byte(base), &envelope); e != nil {
				t.Fatal(e)
			}
			test.mutate(envelope)
			calls := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { calls++; _ = json.NewEncoder(w).Encode(envelope) }))
			defer server.Close()
			p, e := NewStatementOpenRouterProvider("synthetic", server.Client())
			if e != nil {
				t.Fatal(e)
			}
			p.screeningDocument = true
			p.endpoint = server.URL
			pin := statementextract.ScreeningPolicy()
			policy := ModelPolicy{Model: model, Provider: pin.Provider, ProviderName: pin.ProviderName, InputMicrosPerMillion: pin.InputMicrosPerMillion, OutputMicrosPerMillion: pin.OutputMicrosPerMillion, VerifiedAt: time.Now().Add(-time.Minute)}
			out, e := p.Complete(context.Background(), LLMRequest{Policy: policy, DisableReasoning: true, Messages: []LLMMessage{{Role: "system", Content: "Synthetic"}, {Role: "user", Content: "Synthetic"}}, Schema: json.RawMessage(`{"type":"object"}`), InputTokenUpperBound: 100000, MaxOutputTokens: 16000})
			detail := ""
			if e != nil {
				detail = "generic"
				var protocol interface{ ProtocolReason() string }
				if errors.As(e, &protocol) {
					detail = protocol.ProtocolReason()
				}
			}
			if calls != 1 || detail != test.detail || out.BillingKnown != test.known {
				t.Fatalf("calls=%d detail=%q known=%v error=%v", calls, detail, out.BillingKnown, e)
			}
		})
	}
}
