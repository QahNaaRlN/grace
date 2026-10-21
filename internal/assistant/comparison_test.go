package assistant

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/QahNaaRlN/grace/internal/statementextract"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func comparisonMetadataFixture(c statementextract.ComparisonCandidate, path string) map[string]any {
	p := c.Policy
	switch {
	case strings.HasSuffix(path, "/models"):
		return map[string]any{"data": []any{map[string]any{"id": p.Model, "canonical_slug": c.Canonical, "reasoning": map[string]any{"mandatory": false, "supported_efforts": []string{"none"}}}}}
	case strings.HasSuffix(path, "/endpoints/zdr"):
		return map[string]any{"data": []any{map[string]any{"model_id": p.Model, "tag": p.Provider, "status": 0}}}
	default:
		return map[string]any{"data": map[string]any{"endpoints": []any{map[string]any{"model_id": p.Model, "tag": p.Provider, "provider_name": p.ProviderName, "status": 0, "context_length": 116000, "max_completion_tokens": 16000, "supported_parameters": []string{"structured_outputs", "response_format", "max_tokens", "reasoning"}, "pricing": map[string]any{"prompt": fmt.Sprintf("%.12f", float64(p.InputMicrosPerMillion)/1e12), "completion": fmt.Sprintf("%.12f", float64(p.OutputMicrosPerMillion)/1e12)}}}}}
	}
}
func comparisonWire(c statementextract.ComparisonCandidate) map[string]any {
	return map[string]any{"id": "gen-SYNTHETIC-comparison", "model": c.Policy.Model, "provider": c.Policy.ProviderName, "openrouter_metadata": map[string]any{"requested": c.Policy.Model, "strategy": "direct", "attempt": 1, "is_byok": false, "pipeline": []any{}, "endpoints": map[string]any{"available": []any{map[string]any{"model": c.Canonical, "provider": c.Policy.ProviderName, "selected": true}}}}, "choices": []any{map[string]any{"finish_reason": "stop", "message": map[string]any{"content": "{}"}}}, "usage": map[string]any{"prompt_tokens": 100, "completion_tokens": 10, "cost": 0.001}}
}
func jsonResponse(v any) *http.Response {
	raw, _ := json.Marshal(v)
	return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(string(raw)))}
}
func comparisonTestReasoner(t *testing.T, c statementextract.ComparisonCandidate) *ComparisonReasoner {
	t.Helper()
	r, e := NewStatementComparisonReasoner("synthetic", c.ID)
	if e != nil {
		t.Fatal(e)
	}
	return r
}
func TestComparisonFreshMetadataAndRefusals(t *testing.T) {
	for _, c := range statementextract.ComparisonCandidates() {
		for _, mode := range []string{"valid", "alias", "canonical", "duplicate", "reasoning", "endpoint", "price", "status", "cap", "parameters", "zdr", "redirect", "oversize"} {
			t.Run(c.ID+"/"+mode, func(t *testing.T) {
				r := comparisonTestReasoner(t, c)
				gets, posts := 0, 0
				r.client = doerFunc(func(req *http.Request) (*http.Response, error) {
					gets++
					if req.Method != "GET" || req.Header.Get("Authorization") != "" {
						posts++
						t.Fatal("metadata boundary")
					}
					v := comparisonMetadataFixture(c, req.URL.Path)
					if strings.HasSuffix(req.URL.Path, "/models") {
						m := v["data"].([]any)[0].(map[string]any)
						switch mode {
						case "alias":
							m["id"] = "other"
						case "canonical":
							m["canonical_slug"] = "other"
						case "duplicate":
							v["data"] = append(v["data"].([]any), m)
						case "reasoning":
							m["reasoning"] = map[string]any{"mandatory": true, "supported_efforts": []string{"none"}}
						}
					} else if strings.HasSuffix(req.URL.Path, "/endpoints/zdr") {
						if mode == "zdr" {
							v["data"] = []any{}
						}
					} else {
						ep := v["data"].(map[string]any)["endpoints"].([]any)[0].(map[string]any)
						switch mode {
						case "endpoint":
							ep["tag"] = "other"
						case "price":
							ep["pricing"].(map[string]any)["prompt"] = "0.001"
						case "status":
							delete(ep, "status")
						case "cap":
							ep["context_length"] = 100000
						case "parameters":
							ep["supported_parameters"] = []string{"response_format"}
						}
					}
					if mode == "redirect" {
						return &http.Response{StatusCode: 302, Body: io.NopCloser(strings.NewReader(""))}, nil
					}
					if mode == "oversize" {
						return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(strings.Repeat(" ", (8<<20)+1)))}, nil
					}
					return jsonResponse(v), nil
				})
				p, e := r.CurrentPolicy(context.Background())
				if (e == nil) != (mode == "valid") || posts != 0 {
					t.Fatal("metadata refusal", p, e)
				}
				if mode == "valid" {
					before := gets
					proof, e := r.currentProof(context.Background())
					if e != nil || !proof.ValidAt(time.Now()) || gets != before {
						t.Fatal("proof cache")
					}
					proof.Candidate.Canonical = "mutated"
					if r.proof.Candidate.Canonical != c.Canonical {
						t.Fatal("cache aliasing")
					}
					r.proof.VerifiedAt = time.Now().Add(-11 * time.Minute)
					r.client = doerFunc(func(*http.Request) (*http.Response, error) { return nil, errors.New("private") })
					if _, e = r.CurrentPolicy(context.Background()); e == nil || r.proof != nil {
						t.Fatal("stale proof reused")
					}
				}
			})
		}
	}
	redirects := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if strings.HasSuffix(req.URL.Path, "/models") {
			http.Redirect(w, req, "/elsewhere", 302)
		} else {
			redirects++
		}
	}))
	defer server.Close()
	r := comparisonTestReasoner(t, statementextract.ComparisonCandidates()[0])
	r.base = server.URL + "/"
	if _, e := r.CurrentPolicy(context.Background()); e == nil || redirects != 0 {
		t.Fatal("redirect followed")
	}
}
func TestComparisonWireIdentityAndIsolation(t *testing.T) {
	for _, c := range statementextract.ComparisonCandidates() {
		for _, mode := range []string{"canonical", "alias", "top_canonical", "other", "provider", "display", "retry", "requested", "pipeline", "cost", "token", "content", "stale_proof", "future_proof", "wrong_proof", "product"} {
			t.Run(c.ID+"/"+mode, func(t *testing.T) {
				r := comparisonTestReasoner(t, c)
				r.client = doerFunc(func(req *http.Request) (*http.Response, error) {
					return jsonResponse(comparisonMetadataFixture(c, req.URL.Path)), nil
				})
				if _, e := r.CurrentPolicy(context.Background()); e != nil {
					t.Fatal(e)
				}
				wire := comparisonWire(c)
				meta := wire["openrouter_metadata"].(map[string]any)
				selected := meta["endpoints"].(map[string]any)["available"].([]any)[0].(map[string]any)
				switch mode {
				case "alias":
					selected["model"] = c.Policy.Model
				case "top_canonical":
					wire["model"] = c.Canonical
				case "other":
					selected["model"] = "PRIVATE-other"
				case "provider":
					selected["provider"] = "PRIVATE-other"
				case "display":
					wire["provider"] = "PRIVATE-other"
				case "retry":
					meta["attempt"] = 2
				case "requested":
					meta["requested"] = c.Canonical
				case "pipeline":
					meta["pipeline"] = []string{"PRIVATE"}
				case "cost":
					delete(wire["usage"].(map[string]any), "cost")
				case "token":
					wire["usage"].(map[string]any)["completion_tokens"] = 16001
				case "content":
					wire["choices"].([]any)[0].(map[string]any)["message"].(map[string]any)["content"] = "PRIVATE-not-json"
				}
				posts := 0
				r.provider.client = doerFunc(func(req *http.Request) (*http.Response, error) {
					posts++
					var in map[string]any
					if json.NewDecoder(req.Body).Decode(&in) != nil {
						t.Fatal("request")
					}
					route := in["provider"].(map[string]any)
					if req.Method != "POST" || in["model"] != c.Policy.Model || len(route["only"].([]any)) != 1 || route["only"].([]any)[0] != c.Policy.Provider || route["zdr"] != true || route["data_collection"] != "deny" || route["allow_fallbacks"] != false || route["require_parameters"] != true || in["max_tokens"] != float64(16000) || in["reasoning"].(map[string]any)["enabled"] != false {
						t.Fatal("wire policy")
					}
					return jsonResponse(wire), nil
				})
				proof := *r.proof
				switch mode {
				case "stale_proof":
					proof.VerifiedAt = time.Now().Add(-11 * time.Minute)
				case "future_proof":
					proof.VerifiedAt = time.Now().Add(time.Minute)
				case "wrong_proof":
					proof.Candidate.Canonical = "PRIVATE"
				case "product":
					r.provider.comparisonID = ""
				}
				p := c.Policy
				policy := ModelPolicy{Model: p.Model, Provider: p.Provider, ProviderName: p.ProviderName, InputMicrosPerMillion: p.InputMicrosPerMillion, OutputMicrosPerMillion: p.OutputMicrosPerMillion, VerifiedAt: time.Now().Add(-time.Second)}
				out, e := r.provider.Complete(context.Background(), LLMRequest{comparisonProof: &proof, Policy: policy, DisableReasoning: true, Messages: []LLMMessage{{Role: "system", Content: "Synthetic"}, {Role: "user", Content: "Synthetic"}}, Schema: json.RawMessage(`{}`), InputTokenUpperBound: 100000, MaxOutputTokens: 16000})
				good := mode == "canonical" || mode == "alias" || mode == "top_canonical"
				if (e == nil) != good {
					t.Fatal("guard", e)
				}
				noPost := mode == "stale_proof" || mode == "future_proof" || mode == "wrong_proof" || mode == "product"
				if noPost && posts != 0 || !noPost && posts != 1 {
					t.Fatal("post count", posts)
				}
				if good && (!out.BillingKnown || !out.ComparisonRoute.Valid(c.ID, time.Now().Add(-time.Minute), time.Now())) {
					t.Fatal("route proof", out)
				}
				safe, _ := json.Marshal(out.ScreeningDiagnostic)
				if strings.Contains(string(safe), "PRIVATE") {
					t.Fatal("diagnostic leak")
				}
				if c.Policy.Valid() {
					t.Fatal("product policy accepted")
				}
			})
		}
	}
}

func TestComparisonReasonerStrictJSONRetainsKnownBill(t *testing.T) {
	for _, c := range statementextract.ComparisonCandidates() {
		for _, content := range []string{`{}`, `{"PRIVATE":"echo"}`, `{} {}`, `not-json`} {
			t.Run(c.ID+content, func(t *testing.T) {
				r := comparisonTestReasoner(t, c)
				r.client = doerFunc(func(req *http.Request) (*http.Response, error) {
					return jsonResponse(comparisonMetadataFixture(c, req.URL.Path)), nil
				})
				r.provider.client = doerFunc(func(req *http.Request) (*http.Response, error) {
					wire := comparisonWire(c)
					wire["choices"].([]any)[0].(map[string]any)["message"].(map[string]any)["content"] = content
					return jsonResponse(wire), nil
				})
				chunk := statementextract.Chunk{Index: 1, Spans: []statementextract.RedactedSpan{{ID: "s1", Page: 1, Text: "RUB"}}}
				_, bill, e := r.Complete(context.Background(), c.Policy, chunk)
				if (e == nil) != (content == `{}`) || bill == nil || bill.CostMicros != 1000 || !bill.ComparisonRoute.Valid(c.ID, time.Now().Add(-time.Minute), time.Now()) {
					t.Fatal("decode/billing", bill, e)
				}
			})
		}
	}
}
