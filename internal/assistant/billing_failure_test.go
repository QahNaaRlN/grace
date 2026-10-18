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

type failedBillingMeter struct {
	settled *LLMResponse
	calls   int
}

func (m *failedBillingMeter) Reserve(context.Context, ModelPolicy, int, int, string) (string, error) {
	return "synthetic-reservation", nil
}
func (m *failedBillingMeter) Settle(_ context.Context, _ string, out *LLMResponse) error {
	m.calls++
	m.settled = out
	return nil
}

func TestRejectedOutputRetainsVerifiedBillingWithoutReturningContent(t *testing.T) {
	for _, tc := range []struct {
		name, choices string
		tokens        int
		reason        string
	}{
		{"truncated", `[{"finish_reason":"length","message":{"content":"private incomplete output"}}]`, 10, "finish_length"},
		{"invalid_json", `[{"finish_reason":"stop","message":{"content":"private invalid output"}}]`, 10, "content"},
		{"tools", `[{"finish_reason":"stop","message":{"content":"{}","tool_calls":[{}]}}]`, 10, "tools"},
		{"no_choices", `[]`, 10, "choices"},
		{"zero_output_usage", `[{"finish_reason":"stop","message":{"content":"{}"}}]`, 0, "usage"},
		{"token_overrun", `[{"finish_reason":"stop","message":{"content":"{}"}}]`, 101, "token_bound"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			envelope := map[string]any{"id": "synthetic-call", "model": "fixture/model-v1", "openrouter_metadata": map[string]any{"requested": "fixture/model-v1", "strategy": "direct", "attempt": 1, "is_byok": false, "endpoints": map[string]any{"available": []any{map[string]any{"provider": "fixture", "model": "fixture/model-v1", "selected": true}}}, "pipeline": []any{}}, "choices": json.RawMessage(tc.choices), "usage": map[string]any{"prompt_tokens": 100, "completion_tokens": tc.tokens, "cost": json.Number("0.0000500001")}}
			var calls int
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { calls++; json.NewEncoder(w).Encode(envelope) }))
			defer server.Close()
			provider, _ := NewOpenRouterProvider("synthetic-key", server.Client())
			provider.endpoint = server.URL
			meter := &failedBillingMeter{}
			out, err := MeteredCompletion(context.Background(), provider, meter, LLMRequest{Policy: policyFixture(), Messages: []LLMMessage{{Role: "user", Content: "Synthetic"}}, Schema: json.RawMessage(`{}`), InputTokenUpperBound: 1000, MaxOutputTokens: 100}, "analysis")
			var protocol *ProviderProtocolError
			if !errors.As(err, &protocol) || protocol.Reason != tc.reason || calls != 1 || meter.calls != 1 {
				t.Fatal(err, calls, meter.calls)
			}
			if len(out.Content) != 0 || out.ID != "" {
				t.Fatal("rejected completion escaped")
			}
			if meter.settled == nil || !meter.settled.BillingKnown || meter.settled.ID != "synthetic-call" || meter.settled.CostMicros != 51 || len(meter.settled.Content) != 0 {
				t.Fatal("verified bill lost or private output retained", meter.settled)
			}
			// Missing route or cost evidence must still freeze settlement, never become zero cost.
			delete(envelope, "openrouter_metadata")
			meter = &failedBillingMeter{}
			_, err = MeteredCompletion(context.Background(), provider, meter, LLMRequest{Policy: policyFixture(), Messages: []LLMMessage{{Role: "user", Content: "Synthetic"}}, Schema: json.RawMessage(`{}`), InputTokenUpperBound: 1000, MaxOutputTokens: 100}, "analysis")
			if err == nil || meter.calls != 1 || meter.settled != nil {
				t.Fatal("unverified bill settled", err, meter.settled)
			}
		})
	}
}

type failedStatementProvider struct{ known bool }

func (p failedStatementProvider) Complete(context.Context, LLMRequest) (LLMResponse, error) {
	return LLMResponse{BillingKnown: p.known, ID: "synthetic-failed", CostMicros: 17, PromptTokens: 100, CompletionTokens: 2000}, &ProviderProtocolError{Reason: "finish_length"}
}
func TestStatementReasonerRetainsOnlyVerifiedBillOnProviderError(t *testing.T) {
	for _, known := range []bool{true, false} {
		r := &StatementReasoner{Policy: ModelPolicy{Model: "google/gemini-2.5-flash", Provider: "google-vertex/eu", ProviderName: "Google", InputMicrosPerMillion: 300000, OutputMicrosPerMillion: 2500000, VerifiedAt: time.Now().Add(-time.Minute)}, Provider: failedStatementProvider{known}}
		policy, err := r.CurrentPolicy(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		proposal, billing, err := r.Complete(context.Background(), policy, statementextract.Chunk{Index: 1, Spans: []statementextract.RedactedSpan{{ID: "span", Page: 1, Text: "[text_span]"}}})
		if err == nil || len(proposal.Rows) != 0 || (billing != nil) != known {
			t.Fatal("provider failure became usable or lost bill", err, billing)
		}
		if known && (billing.GenerationID != "synthetic-failed" || billing.CostMicros != 17) {
			t.Fatal(billing)
		}
	}
}
