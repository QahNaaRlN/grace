package assistant

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
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
	base := `{"id":"gen-SYNTHETIC-diagnosis","model":"google/gemini-3.1-flash-lite","provider":"Google","openrouter_metadata":{"requested":"google/gemini-3.1-flash-lite","strategy":"direct","attempt":1,"is_byok":false,"pipeline":[],"endpoints":{"available":[{"model":"google/gemini-3.1-flash-lite","provider":"Google","selected":true}]}},"choices":[{"finish_reason":"stop","message":{"content":"{}"}}],"usage":{"prompt_tokens":100,"completion_tokens":10,"cost":0.001}}`
	tests := []struct {
		name, detail string
		known        bool
		mutate       func(object)
	}{
		{"valid", "", true, func(o object) {}},
		{"malformed_id", "", true, func(o object) { o["id"] = "PRIVATE-invalid-id" }},
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

	tests = append(tests, []struct {
		name, detail string
		known        bool
		mutate       func(object)
	}{
		{"strategy", "generic", false, func(o object) { o["openrouter_metadata"].(object)["strategy"] = "fallback" }},
		{"negative_prompt", "usage", false, func(o object) { o["usage"].(object)["prompt_tokens"] = -1 }},
		{"negative_completion", "usage", false, func(o object) { o["usage"].(object)["completion_tokens"] = -1 }},
		{"zero_completion", "usage", true, func(o object) { o["usage"].(object)["completion_tokens"] = 0 }},
		{"token_bound", "token_bound", true, func(o object) { o["usage"].(object)["completion_tokens"] = 16001 }},
		{"choices", "choices", true, func(o object) { o["choices"] = []any{} }},
		{"tools", "tools", true, func(o object) {
			o["choices"].([]any)[0].(object)["message"].(object)["tool_calls"] = []any{object{"private": "never-log"}}
		}},
		{"finish_other", "finish_other", true, func(o object) { o["choices"].([]any)[0].(object)["finish_reason"] = "private-other-reason" }},
		{"duplicate_selected", "generic", false, func(o object) {
			a := o["openrouter_metadata"].(object)["endpoints"].(object)
			a["available"] = append(a["available"].([]any), a["available"].([]any)[0])
		}},
	}...)
	codes := map[string]string{"malformed_id": "unknown", "valid": "unknown", "canonical_top_model": "envelope", "metadata_missing": "metadata_missing", "canonical_requested": "requested", "canonical_selected_model": "selected_model", "changed_selected_provider": "provider", "changed_display_provider": "display_provider", "retry_attempt": "attempt", "pipeline": "pipeline", "byok": "byok", "no_selected_route": "count", "duplicate_selected": "count", "missing_cost": "cost", "length": "finish_length", "invalid_content": "content", "strategy": "strategy", "negative_prompt": "usage", "negative_completion": "usage", "zero_completion": "usage", "token_bound": "token_bound", "choices": "choices", "tools": "tools", "finish_other": "finish_other"}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			for _, screening := range []bool{false, true} {
				raw := base
				if !screening {
					raw = strings.ReplaceAll(raw, model, "google/gemini-2.5-flash")
				}
				var envelope object
				if e := json.Unmarshal([]byte(raw), &envelope); e != nil {
					t.Fatal(e)
				}
				test.mutate(envelope)
				calls := 0
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					calls++
					w.Header().Set("Content-Type", "application/json")
					w.Header().Set("Content-Type", "application/json")
					_ = json.NewEncoder(w).Encode(envelope)
				}))
				defer server.Close()
				p, e := NewStatementOpenRouterProvider("synthetic", server.Client())
				if e != nil {
					t.Fatal(e)
				}
				p.screeningDocument = screening
				p.endpoint = server.URL
				pin := statementextract.ScreeningPolicy()
				policy := ModelPolicy{Model: model, Provider: pin.Provider, ProviderName: pin.ProviderName, InputMicrosPerMillion: pin.InputMicrosPerMillion, OutputMicrosPerMillion: pin.OutputMicrosPerMillion, VerifiedAt: time.Now().Add(-time.Minute)}
				if !screening {
					policy.Model = "google/gemini-2.5-flash"
					policy.InputMicrosPerMillion = 300000
					policy.OutputMicrosPerMillion = 2500000
				}
				out, e := p.Complete(context.Background(), LLMRequest{Policy: policy, DisableReasoning: true, Messages: []LLMMessage{{Role: "system", Content: "Synthetic"}, {Role: "user", Content: "Synthetic"}}, Schema: json.RawMessage(`{"type":"object"}`), InputTokenUpperBound: 100000, MaxOutputTokens: 16000})
				detail := ""
				if e != nil {
					detail = "generic"
					var protocol interface{ ProtocolReason() string }
					if errors.As(e, &protocol) {
						detail = protocol.ProtocolReason()
					}
				}
				expectedID := "gen-SYNTHETIC-diagnosis"
				if test.name == "malformed_id" {
					expectedID = ""
				}
				if screening && (out.ScreeningDiagnostic == nil || out.ScreeningDiagnostic.Code != codes[test.name] || out.ScreeningDiagnostic.HTTPStatus != 200 || out.ScreeningDiagnostic.ContentType != "json" || out.ScreeningDiagnostic.CorrelationID != expectedID) {
					t.Fatalf("diagnostic mismatch: %+v", out.ScreeningDiagnostic)
				}
				if !screening && out.ScreeningDiagnostic != nil {
					t.Fatal("product diagnostics changed")
				}
				if e != nil {
					want := "provider_protocol"
					if test.name == "missing_cost" {
						want = "price_unknown"
					}
					if e.Error() != want {
						t.Fatal("legacy error changed", e)
					}
				}
				if calls != 1 || detail != test.detail || out.BillingKnown != test.known {
					t.Fatalf("calls=%d detail=%q known=%v error=%v", calls, detail, out.BillingKnown, e)
				}
			}
		})
	}
}
