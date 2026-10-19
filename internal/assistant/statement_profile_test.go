package assistant

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/QahNaaRlN/grace/internal/statementdata"
	"github.com/QahNaaRlN/grace/internal/statementextract"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func statementVerifierFixture(t *testing.T) *PolicyVerifier {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "" {
			t.Error("metadata received a key")
		}
		fmt.Fprint(w, `{"data":{"endpoints":[{"model_id":"google/gemini-2.5-flash","tag":"google-vertex/eu","provider_name":"Google","status":0,"context_length":1048576,"max_completion_tokens":65535,"supported_parameters":["reasoning","structured_outputs","response_format","max_tokens"],"pricing":{"prompt":"0.0000003","completion":"0.0000025"}}]}}`)
	}))
	t.Cleanup(server.Close)
	v := NewPolicyVerifier()
	v.endpoint = server.URL + "/"
	return v
}
func statementPolicyFixture() ModelPolicy {
	return ModelPolicy{Model: "google/gemini-2.5-flash", Provider: "google-vertex/eu", ProviderName: "Google", InputMicrosPerMillion: 300000, OutputMicrosPerMillion: 2500000, VerifiedAt: time.Now().Add(-time.Minute)}
}
func TestStatementProfileIsolatedFromDefaultAndBounds(t *testing.T) {
	policy := statementPolicyFixture()
	client := &http.Client{Timeout: 30 * time.Second}
	standard, _ := NewOpenRouterProvider("synthetic", client)
	statement, e := NewStatementOpenRouterProvider("synthetic", client)
	if e != nil {
		t.Fatal(e)
	}
	if client.Timeout != 30*time.Second || standard.StatementDocumentProfile() || !statement.StatementDocumentProfile() || statement.client.(*http.Client).Timeout != 120*time.Second {
		t.Fatal("shared transport changed")
	}
	if _, e := standard.reservation(policy, 100000, 16000); e == nil {
		t.Fatal("standard profile widened")
	}
	if n, e := statement.reservation(policy, 100000, 16000); e != nil || n != 70000 {
		t.Fatal(n, e)
	}
	for _, test := range []struct{ input, output int }{{100001, 16000}, {100000, 16001}, {0, 1}, {1, 0}} {
		if _, e := statement.reservation(policy, test.input, test.output); e == nil {
			t.Fatal("bounds widened", test)
		}
	}
	policy.Vision = true
	if _, e := statement.reservation(policy, 100000, 16000); e == nil {
		t.Fatal("vision allowed")
	}
}
func TestStatementMetadataRequiresExactCapabilitiesEvenFreshApproval(t *testing.T) {
	for _, tc := range []struct {
		name                string
		context, completion int
		tag                 string
		reasoning           bool
		want                bool
	}{
		{"valid", 116000, 16000, "google-vertex/eu", true, true}, {"context", 115999, 16000, "google-vertex/eu", true, false}, {"output", 116000, 15999, "google-vertex/eu", true, false}, {"route", 116000, 16000, "google-vertex/global", true, false}, {"reasoning", 116000, 16000, "google-vertex/eu", false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				params := []string{"structured_outputs", "response_format", "max_tokens"}
				if tc.reasoning {
					params = append(params, "reasoning")
				}
				p, _ := json.Marshal(params)
				fmt.Fprintf(w, `{"data":{"endpoints":[{"model_id":"google/gemini-2.5-flash","tag":%q,"provider_name":"Google","status":0,"context_length":%d,"max_completion_tokens":%d,"supported_parameters":%s,"pricing":{"prompt":"0.0000003","completion":"0.0000025"}}]}}`, tc.tag, tc.context, tc.completion, p)
			}))
			defer server.Close()
			verifier := NewPolicyVerifier()
			verifier.endpoint = server.URL + "/"
			_, e := verifier.CurrentStatement(context.Background(), statementPolicyFixture())
			if (e == nil) != tc.want || calls != 1 {
				t.Fatal(e, calls)
			}
			verifier.CurrentStatement(context.Background(), statementPolicyFixture())
			if calls != 1 {
				t.Fatal("metadata retried without bound")
			}
		})
	}
	if _, e := (*PolicyVerifier)(nil).CurrentStatement(context.Background(), statementPolicyFixture()); e == nil {
		t.Fatal("missing capability evidence accepted")
	}
}
func TestStatementEnvelopeOverrunDoesNotReachTransport(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { calls++ }))
	defer server.Close()
	p, _ := NewStatementOpenRouterProvider("synthetic", server.Client())
	p.endpoint = server.URL
	_, e := p.Complete(context.Background(), LLMRequest{Policy: statementPolicyFixture(), Messages: []LLMMessage{{Role: "system", Content: "Synthetic system"}, {Role: "user", Content: strings.Repeat("a", 100000)}}, Schema: json.RawMessage(`{}`), InputTokenUpperBound: 100000, MaxOutputTokens: 16000, DisableReasoning: true})
	if e == nil || calls != 0 {
		t.Fatal("oversize request transmitted", e, calls)
	}
}

func TestStatementWireProfileRoundTripAboveLegacyTokenBounds(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		raw, err := io.ReadAll(r.Body)
		if err != nil {
			t.Error(err)
		}
		if strings.Contains(string(raw), "p1-b1-l1-w") || strings.Contains(string(raw), `"aliases":`) {
			t.Error("canonical map escaped to provider")
		}
		var body struct {
			MaxTokens int `json:"max_tokens"`
			Reasoning struct {
				Enabled bool `json:"enabled"`
			} `json:"reasoning"`
			Provider struct {
				Only       []string `json:"only"`
				Fallback   bool     `json:"allow_fallbacks"`
				ZDR        bool     `json:"zdr"`
				Collection string   `json:"data_collection"`
				Require    bool     `json:"require_parameters"`
			} `json:"provider"`
		}
		if err = json.Unmarshal(raw, &body); err != nil || body.MaxTokens != 16000 || body.Reasoning.Enabled || len(body.Provider.Only) != 1 || body.Provider.Only[0] != "google-vertex/eu" || body.Provider.Fallback || !body.Provider.ZDR || body.Provider.Collection != "deny" || !body.Provider.Require {
			t.Errorf("wire profile mismatch: %s", raw)
		}
		content := `{"rows":[{"operationDate":["s0"],"bookingDate":[],"accountAmount":["s1"],"accountCurrency":["s2"],"transactionAmount":[],"transactionCurrency":[],"description":[],"direction":"","directionEvidence":[]}],"header":{"openingBookDate":[],"closingBookDate":[],"openingBookBalance":[],"closingBookBalance":[],"availableBalanceObserved":[],"debitTurnover":[],"creditTurnover":[],"currency":[],"labels":{"openingBookBalance":[],"closingBookBalance":[],"availableBalanceObserved":[],"debitTurnover":[],"creditTurnover":[]}},"service":[]}`
		json.NewEncoder(w).Encode(map[string]any{"id": "synthetic-wide-profile", "model": "google/gemini-2.5-flash", "provider": "Google", "openrouter_metadata": map[string]any{"requested": "google/gemini-2.5-flash", "strategy": "direct", "attempt": 1, "is_byok": false, "pipeline": []any{}, "endpoints": map[string]any{"available": []any{map[string]any{"model": "google/gemini-2.5-flash", "provider": "Google", "selected": true}}}}, "choices": []any{map[string]any{"finish_reason": "stop", "message": map[string]any{"content": content}}}, "usage": map[string]any{"prompt_tokens": 30000, "completion_tokens": 3000, "cost": json.Number("0.0165")}})
	}))
	defer server.Close()
	provider, e := NewStatementOpenRouterProvider("synthetic", server.Client())
	if e != nil {
		t.Fatal(e)
	}
	provider.endpoint = server.URL
	reasoner := &StatementReasoner{Verifier: statementVerifierFixture(t), Policy: statementPolicyFixture(), Provider: provider}
	policy, e := reasoner.CurrentPolicy(context.Background())
	if e != nil {
		t.Fatal(e)
	}
	document := statementdata.Document{Extractor: statementdata.ExtractorVersion, Pages: []statementdata.Page{{Number: 1, Width: 600, Height: 800}}}
	for i, text := range []string{"19.10.2026", "-12.34", "RUB"} {
		document.Spans = append(document.Spans, statementdata.Span{ID: fmt.Sprintf("p1-b1-l1-w%d", i+1), Page: 1, Block: 1, Line: 1, Word: i + 1, Box: statementdata.Box{XMin: float64(i * 100), YMin: 1, XMax: float64(i*100 + 50), YMax: 10}, Text: text})
	}
	plan, e := statementextract.BuildPlan(document, policy, nil)
	if e != nil {
		t.Fatal(e)
	}
	wire, bill, e := reasoner.Complete(context.Background(), policy, plan.Chunks[0])
	if e != nil || calls != 1 || bill == nil || bill.CostMicros != 16500 || bill.InputTokens != 30000 || bill.OutputTokens != 3000 {
		t.Fatal("new profile rejected verified bounded usage", e, calls, bill)
	}
	canonical, e := statementextract.CanonicalProposal(plan, wire)
	if e != nil || len(canonical.Rows) != 1 || canonical.Rows[0].OperationDate[0] != "p1-b1-l1-w1" {
		t.Fatal("wire map did not round trip", e)
	}
	review, e := statementdata.ValidateProposal(document, canonical, []int{1}, true, "")
	if e != nil || review.Rows[0].AccountAmount.AmountMinor != 1234 || review.Rows[0].AccountAmount.Currency != "RUB" {
		t.Fatal("canonical financial values changed", e)
	}
}

func TestStatementProfileRejectsImageContentAndEnabledReasoningBeforeRequest(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { calls++ }))
	defer server.Close()
	provider, _ := NewStatementOpenRouterProvider("synthetic", server.Client())
	provider.endpoint = server.URL
	for _, reasoningOff := range []bool{true, false} {
		content := any("Synthetic text")
		if reasoningOff {
			content = []map[string]any{{"type": "image_url", "image_url": map[string]string{"url": "data:image/png;base64,synthetic"}}}
		}
		_, e := provider.Complete(context.Background(), LLMRequest{DisableReasoning: reasoningOff, Policy: statementPolicyFixture(), Messages: []LLMMessage{{Role: "system", Content: "Synthetic system"}, {Role: "user", Content: content}}, Schema: json.RawMessage(`{}`), InputTokenUpperBound: 100000, MaxOutputTokens: 16000})
		if e == nil || calls != 0 {
			t.Fatal("statement profile accepted image/reasoning", e, calls)
		}
	}
}
