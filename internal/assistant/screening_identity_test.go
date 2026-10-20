package assistant

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/QahNaaRlN/grace/internal/statementextract"
)

func syntheticScreeningProof() *statementextract.ScreeningIdentityProof {
	pin := statementextract.ScreeningPolicy()
	return &statementextract.ScreeningIdentityProof{Alias: pin.Model, Canonical: statementextract.ScreeningCanonicalModel, Provider: pin.Provider, CatalogSHA256: strings.Repeat("a", 64), VerifiedAt: time.Now().UTC().Add(-time.Second)}
}

const identityCatalog = `{"data":[{"id":"google/gemini-3.1-flash-lite","canonical_slug":"google/gemini-3.1-flash-lite-20260507"}]}`

func TestScreeningCatalogProofAndRefresh(t *testing.T) {
	for _, mode := range []string{"valid", "missing", "duplicate", "changed", "invalid", "status", "redirect", "timeout", "oversize"} {
		t.Run(mode, func(t *testing.T) {
			gets, posts := 0, 0
			verifier := NewPolicyVerifier()
			verifier.client = doerFunc(func(r *http.Request) (*http.Response, error) {
				if r.Method != http.MethodGet {
					posts++
					t.Fatal("catalog used POST")
				}
				gets++
				if r.Header.Get("Authorization") != "" || r.URL.Path != "/api/v1/models" {
					t.Fatal("catalog request boundary")
				}
				if deadline, ok := r.Context().Deadline(); !ok || time.Until(deadline) > 5*time.Second {
					t.Fatal("catalog timeout missing")
				}
				raw, status := identityCatalog, 200
				switch mode {
				case "missing":
					raw = `{"data":[]}`
				case "duplicate":
					raw = strings.Replace(identityCatalog, "]}", `,{"id":"google/gemini-3.1-flash-lite","canonical_slug":"google/gemini-3.1-flash-lite-20260507"}]}`, 1)
				case "changed":
					raw = strings.ReplaceAll(identityCatalog, "20260507", "NEW")
				case "invalid":
					raw = "PRIVATE-invalid"
				case "status":
					status = 503
				case "redirect":
					status = 302
				case "timeout":
					return nil, context.DeadlineExceeded
				case "oversize":
					raw = strings.Repeat(" ", (8<<20)+1)
				}
				return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader(raw))}, nil
			})
			// Stale/future/zero proof никогда не служит обходом провалившегося refresh.
			for _, when := range []time.Time{time.Now().Add(-11 * time.Minute), time.Now().Add(time.Minute), {}} {
				verifier.screeningProof = syntheticScreeningProof()
				verifier.screeningProof.VerifiedAt = when
				proof, err := verifier.screeningIdentity(context.Background())
				if mode == "valid" {
					if err != nil || proof == nil || !proof.ValidAt(time.Now()) {
						t.Fatal("valid refresh", err)
					}
					before := gets
					cached, err := verifier.screeningIdentity(context.Background())
					if err != nil || gets != before || cached == proof {
						t.Fatal("cache/copy")
					}
					cached.Canonical = "other"
					if verifier.screeningProof.Canonical != "google/gemini-3.1-flash-lite-20260507" {
						t.Fatal("cache aliasing")
					}
				} else if err == nil || proof != nil || verifier.screeningProof != nil {
					t.Fatal("failed refresh accepted")
				}
			}
			if gets != 3 || posts != 0 {
				t.Fatal("refresh count", gets, posts)
			}
		})
	}
	// Настоящий HTTP redirect не достигает второго URL.
	redirects := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/models" {
			http.Redirect(w, r, "/elsewhere", 302)
			return
		}
		redirects++
		fmt.Fprint(w, identityCatalog)
	}))
	defer server.Close()
	v := NewPolicyVerifier()
	v.endpoint = server.URL + "/models/"
	if _, err := v.screeningIdentity(context.Background()); err == nil || redirects != 0 {
		t.Fatal("redirect followed")
	}
}

func TestScreeningProofRejectsBeforePOST(t *testing.T) {
	for _, mode := range []string{"nil", "stale", "future", "zero", "canonical", "provider", "digest", "alias", "policy"} {
		t.Run(mode, func(t *testing.T) {
			posts := 0
			provider, _ := NewStatementOpenRouterProvider("synthetic", doerFunc(func(*http.Request) (*http.Response, error) { posts++; return nil, errors.New("unexpected POST") }))
			provider.screeningDocument = true
			pin := statementextract.ScreeningPolicy()
			in := LLMRequest{screeningProof: syntheticScreeningProof(), Policy: ModelPolicy{Model: pin.Model, Provider: pin.Provider, ProviderName: pin.ProviderName, InputMicrosPerMillion: pin.InputMicrosPerMillion, OutputMicrosPerMillion: pin.OutputMicrosPerMillion, VerifiedAt: time.Now().Add(-time.Minute)}, DisableReasoning: true, Messages: []LLMMessage{{Role: "system", Content: "Synthetic"}, {Role: "user", Content: "Synthetic"}}, Schema: json.RawMessage(`{}`), InputTokenUpperBound: 100000, MaxOutputTokens: 16000}
			switch mode {
			case "nil":
				in.screeningProof = nil
			case "stale":
				in.screeningProof.VerifiedAt = time.Now().Add(-11 * time.Minute)
			case "future":
				in.screeningProof.VerifiedAt = time.Now().Add(time.Minute)
			case "zero":
				in.screeningProof.VerifiedAt = time.Time{}
			case "canonical":
				in.screeningProof.Canonical += "-NEW"
			case "provider":
				in.screeningProof.Provider = "google-vertex/global"
			case "digest":
				in.screeningProof.CatalogSHA256 = strings.Repeat("A", 64)
			case "alias":
				in.screeningProof.Alias += "-other"
			case "policy":
				in.Policy.Provider = "google-vertex/global"
			}
			out, err := provider.Complete(context.Background(), in)
			if err == nil || out.BillingKnown || posts != 0 || out.ScreeningDiagnostic == nil || out.ScreeningDiagnostic.Code != "request" {
				t.Fatal("invalid proof reached POST", err, posts)
			}
		})
	}
}

func TestScreeningCanonicalKeepsAllRouteAndContentGuards(t *testing.T) {
	type object = map[string]any
	base := `{"id":"gen-SYNTHETIC-identity","model":"google/gemini-3.1-flash-lite","provider":"Google","openrouter_metadata":{"requested":"google/gemini-3.1-flash-lite","strategy":"direct","attempt":1,"is_byok":false,"pipeline":[],"endpoints":{"available":[{"model":"google/gemini-3.1-flash-lite-20260507","provider":"Google","selected":true}]}},"choices":[{"finish_reason":"stop","message":{"content":"{}"}}],"usage":{"prompt_tokens":100,"completion_tokens":10,"cost":0.001}}`
	for _, mode := range []string{"canonical", "alias", "other", "empty", "multiple", "none", "provider", "display", "top", "requested", "retry", "byok", "pipeline", "cost", "token", "content"} {
		t.Run(mode, func(t *testing.T) {
			var wire object
			if json.Unmarshal([]byte(base), &wire) != nil {
				t.Fatal("fixture")
			}
			metadata := wire["openrouter_metadata"].(object)
			routes := metadata["endpoints"].(object)
			endpoints := routes["available"].([]any)
			endpoint := endpoints[0].(object)
			wantClass, wantCode, known := "pinned_canonical", "unknown", true
			switch mode {
			case "alias":
				endpoint["model"] = statementextract.ScreeningPolicy().Model
				wantClass = "alias"
			case "other":
				endpoint["model"] = "PRIVATE-unapproved-model"
				wantClass, wantCode, known = "other", "selected_model", false
			case "empty":
				delete(endpoint, "model")
				wantClass, wantCode, known = "empty", "selected_model", false
			case "multiple":
				routes["available"] = append(endpoints, endpoint)
				wantClass, wantCode, known = "multiple", "count", false
			case "none":
				endpoint["selected"] = false
				wantClass, wantCode, known = "not_observed", "count", false
			case "provider":
				endpoint["provider"] = "PRIVATE-provider"
				wantCode, known = "provider", false
			case "display":
				wire["provider"] = "PRIVATE-provider"
				wantCode, known = "display_provider", false
			case "top":
				wire["model"] = statementextract.ScreeningCanonicalModel
				wantCode, known = "envelope", false
			case "requested":
				metadata["requested"] = statementextract.ScreeningCanonicalModel
				wantCode, known = "requested", false
			case "retry":
				metadata["attempt"] = 2
				wantCode, known = "attempt", false
			case "byok":
				metadata["is_byok"] = true
				wantCode, known = "byok", false
			case "pipeline":
				metadata["pipeline"] = []any{object{"type": "PRIVATE"}}
				wantCode, known = "pipeline", false
			case "cost":
				delete(wire["usage"].(object), "cost")
				wantCode, known = "cost", false
			case "token":
				wire["usage"].(object)["completion_tokens"] = 16001
				wantCode = "token_bound"
			case "content":
				wire["choices"].([]any)[0].(object)["message"].(object)["content"] = "PRIVATE-invalid-json"
				wantCode = "content"
			}
			payload, _ := json.Marshal(wire)
			posts := 0
			p, _ := NewStatementOpenRouterProvider("synthetic", doerFunc(func(req *http.Request) (*http.Response, error) {
				posts++
				var request object
				if json.NewDecoder(req.Body).Decode(&request) != nil {
					t.Fatal("request")
				}
				privacy := request["provider"].(object)
				if req.Method != "POST" || privacy["allow_fallbacks"] != false || privacy["zdr"] != true || privacy["data_collection"] != "deny" || privacy["only"].([]any)[0] != "google-vertex/eu" || request["reasoning"].(object)["enabled"] != false || request["max_tokens"] != float64(16000) {
					t.Fatal("request profile changed")
				}
				return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(string(payload)))}, nil
			}))
			p.screeningDocument = true
			pin := statementextract.ScreeningPolicy()
			policy := ModelPolicy{Model: pin.Model, Provider: pin.Provider, ProviderName: pin.ProviderName, InputMicrosPerMillion: pin.InputMicrosPerMillion, OutputMicrosPerMillion: pin.OutputMicrosPerMillion, VerifiedAt: time.Now().Add(-time.Minute)}
			out, err := p.Complete(context.Background(), LLMRequest{screeningProof: syntheticScreeningProof(), Policy: policy, DisableReasoning: true, Messages: []LLMMessage{{Role: "system", Content: "Synthetic"}, {Role: "user", Content: "Synthetic"}}, Schema: json.RawMessage(`{}`), InputTokenUpperBound: 100000, MaxOutputTokens: 16000})
			if posts != 1 || out.BillingKnown != known || (err == nil) != (mode == "canonical" || mode == "alias") || out.ScreeningDiagnostic.Code != wantCode || out.ScreeningDiagnostic.Route.SelectedModelClass != wantClass {
				t.Fatal("route/content guard", err, out.ScreeningDiagnostic, known)
			}
			safe, _ := json.Marshal(out.ScreeningDiagnostic)
			if strings.Contains(string(safe), "PRIVATE") {
				t.Fatal("diagnostic leak")
			}
		})
	}
}

func TestScreeningProofConcurrentCacheCopies(t *testing.T) {
	verifier := NewPolicyVerifier()
	gets := 0
	verifier.client = doerFunc(func(*http.Request) (*http.Response, error) {
		gets++
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(identityCatalog))}, nil
	})
	var workers sync.WaitGroup
	for range 32 {
		workers.Add(1)
		go func() {
			defer workers.Done()
			proof, err := verifier.screeningIdentity(context.Background())
			if err != nil {
				t.Error(err)
				return
			}
			proof.Canonical = "local-copy"
		}()
	}
	workers.Wait()
	if gets != 1 || verifier.screeningProof.Canonical != statementextract.ScreeningCanonicalModel {
		t.Fatal("shared cache/proof mutated", gets)
	}
}
