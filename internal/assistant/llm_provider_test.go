package assistant

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func policyFixture() ModelPolicy {
	return ModelPolicy{Model: "fixture/model-v1", Provider: "fixture", InputMicrosPerMillion: 500000, OutputMicrosPerMillion: 1500000, VerifiedAt: time.Now().Add(-time.Minute)}
}
func TestOpenRouterPinsPrivacyRouteAndParsesExactCost(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.Header.Get("Authorization") != "Bearer synthetic-test-key" {
			t.Error("missing server authentication")
		}
		if r.Header.Get("X-OpenRouter-Metadata") != "enabled" {
			t.Error("missing route evidence opt-in")
		}
		var input map[string]json.RawMessage
		json.NewDecoder(r.Body).Decode(&input)
		var route struct {
			Only     []string `json:"only"`
			Fallback bool     `json:"allow_fallbacks"`
			ZDR      bool     `json:"zdr"`
			Policy   string   `json:"data_collection"`
			Require  bool     `json:"require_parameters"`
		}
		json.Unmarshal(input["provider"], &route)
		if len(route.Only) != 1 || route.Only[0] != "fixture" || route.Fallback || !route.ZDR || route.Policy != "deny" || !route.Require {
			t.Error("unsafe route", route)
		}
		if input["tools"] != nil || input["plugins"] != nil {
			t.Error("external tools leaked")
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"id":"fixture-call","model":"fixture/model-v1","openrouter_metadata":{"requested":"fixture/model-v1","strategy":"direct","attempt":1,"is_byok":false,"endpoints":{"available":[{"provider":"Fixture Display","model":"fixture/model-v1","selected":true}]},"pipeline":[]},"choices":[{"finish_reason":"stop","message":{"content":"{\"status\":\"ok\"}"}}],"usage":{"prompt_tokens":100,"completion_tokens":10,"cost":0.0000500001}}`))
	}))
	defer server.Close()
	p, _ := NewOpenRouterProvider("synthetic-test-key", server.Client())
	p.endpoint = server.URL
	policy := policyFixture()
	policy.ProviderName = "Fixture Display"
	out, err := p.Complete(context.Background(), LLMRequest{Policy: policy, Messages: []LLMMessage{{Role: "user", Content: "Synthetic input"}}, Schema: json.RawMessage(`{"type":"object"}`), InputTokenUpperBound: 1000, MaxOutputTokens: 100})
	if err != nil || out.CostMicros != 51 || out.PromptTokens != 100 || calls.Load() != 1 {
		t.Fatal(out, err, calls.Load())
	}
}
func TestOpenRouterNeverRetriesOrReflectsPrivateError(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(429)
		w.Write([]byte("synthetic-private-financial-text"))
	}))
	defer server.Close()
	p, _ := NewOpenRouterProvider("synthetic-test-key", server.Client())
	p.endpoint = server.URL
	_, err := p.Complete(context.Background(), LLMRequest{Policy: policyFixture(), Messages: []LLMMessage{{Role: "user", Content: "Synthetic"}}, Schema: json.RawMessage(`{}`), InputTokenUpperBound: 1000, MaxOutputTokens: 100})
	if err == nil || strings.Contains(err.Error(), "financial-text") || calls.Load() != 1 {
		t.Fatal(err, calls.Load())
	}
}
func TestOpenRouterRejectsMissingCostChangedProviderAndTruncation(t *testing.T) {
	fixtures := []string{
		`{"id":"call","model":"fixture/model-v1","provider":"fixture","choices":[{"finish_reason":"stop","message":{"content":"{}"}}],"usage":{"prompt_tokens":100,"completion_tokens":1}}`,
		`{"id":"call","model":"fixture/model-v1","provider":"other","choices":[{"finish_reason":"stop","message":{"content":"{}"}}],"usage":{"prompt_tokens":100,"completion_tokens":1,"cost":0.001}}`,
		`{"id":"call","model":"fixture/model-v1","provider":"fixture","choices":[{"finish_reason":"length","message":{"content":"{}"}}],"usage":{"prompt_tokens":100,"completion_tokens":1,"cost":0.001}}`,
	}
	for _, fixture := range fixtures {
		t.Run(fixture, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write([]byte(fixture)) }))
			defer server.Close()
			p, _ := NewOpenRouterProvider("synthetic-test-key", server.Client())
			p.endpoint = server.URL
			_, err := p.Complete(context.Background(), LLMRequest{Policy: policyFixture(), Messages: []LLMMessage{{Role: "user", Content: "Synthetic"}}, Schema: json.RawMessage(`{}`), InputTokenUpperBound: 1000, MaxOutputTokens: 100})
			if err == nil {
				t.Fatal("ambiguous response accepted")
			}
		})
	}
}

func TestOpenRouterDistinguishesRejectedRequestWithoutEchoOrRetry(t *testing.T) {
	for _, status := range []int{400, 422} {
		var calls int
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			calls++
			w.WriteHeader(status)
			w.Write([]byte(`{"error":{"message":"private-photo-text-and-key"}}`))
		}))
		p, _ := NewOpenRouterProvider("synthetic-test-key", server.Client())
		p.endpoint = server.URL
		_, err := p.Complete(context.Background(), LLMRequest{Policy: policyFixture(), Messages: []LLMMessage{{Role: "user", Content: "Synthetic"}}, Schema: json.RawMessage(`{}`), InputTokenUpperBound: 1000, MaxOutputTokens: 100})
		server.Close()
		if err == nil || err.Error() != "provider_request" || calls != 1 {
			t.Fatal(err, calls)
		}
	}
}
func TestCostMicrosNeverRoundsDownOrOverflows(t *testing.T) {
	for value, want := range map[string]int64{"0": 0, "0.0000001": 1, "1.1234567": 1123457, "1e-6": 1} {
		got, err := costMicros(value)
		if err != nil || got != want {
			t.Fatal(value, got, err)
		}
	}
	for _, value := range []string{"", "-1", "NaN", "999999999999999999999999", "1e999999", " 0.1"} {
		if _, err := costMicros(value); err == nil {
			t.Fatal(value)
		}
	}
	p := policyFixture()
	p.VerifiedAt = time.Now().Add(-25 * time.Hour)
	if p.Valid(time.Now()) {
		t.Fatal("stale price accepted")
	}
}
