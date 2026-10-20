package assistant

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/QahNaaRlN/grace/internal/statementextract"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestScreeningTransportIsExplicitAndPinned(t *testing.T) {
	posts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			if r.URL.Path == "/endpoints/zdr" {
				fmt.Fprint(w, `{"data":[{"model_id":"google/gemini-3.1-flash-lite","tag":"google-vertex/eu","status":0}]}`)
				return
			}
			fmt.Fprint(w, `{"data":{"endpoints":[{"model_id":"google/gemini-3.1-flash-lite","tag":"google-vertex/eu","provider_name":"Google","status":0,"context_length":1048576,"max_completion_tokens":65536,"supported_parameters":["structured_outputs","response_format","max_tokens","reasoning"],"pricing":{"prompt":"0.000000275","completion":"0.00000165"}}]}}`)
			return
		}
		posts++
		var request struct {
			Model    string `json:"model"`
			Tokens   int    `json:"max_tokens"`
			Provider struct {
				Only       []string `json:"only"`
				ZDR        bool     `json:"zdr"`
				Fallback   bool     `json:"allow_fallbacks"`
				Collection string   `json:"data_collection"`
			} `json:"provider"`
			Reasoning struct {
				Enabled bool `json:"enabled"`
			} `json:"reasoning"`
		}
		if json.NewDecoder(r.Body).Decode(&request) != nil || request.Model != "google/gemini-3.1-flash-lite" || request.Tokens != 16000 || len(request.Provider.Only) != 1 || request.Provider.Only[0] != "google-vertex/eu" || !request.Provider.ZDR || request.Provider.Fallback || request.Provider.Collection != "deny" || request.Reasoning.Enabled {
			t.Error("request profile", request)
		}
		// A malformed model answer must still retain authoritative known billing.
		fmt.Fprint(w, `{"id":"synthetic-screen","model":"google/gemini-3.1-flash-lite","provider":"Google","openrouter_metadata":{"requested":"google/gemini-3.1-flash-lite","strategy":"direct","attempt":1,"is_byok":false,"pipeline":[],"endpoints":{"available":[{"model":"google/gemini-3.1-flash-lite","provider":"Google","selected":true}]}},"choices":[{"finish_reason":"stop","message":{"content":"invalid JSON"}}],"usage":{"prompt_tokens":30000,"completion_tokens":3000,"cost":0.01}}`)
	}))
	defer server.Close()
	reasoner, e := NewStatementScreeningReasoner("synthetic")
	if e != nil {
		t.Fatal(e)
	}
	reasoner.Verifier.endpoint = server.URL + "/models/"
	reasoner.Verifier.client = server.Client()
	provider := reasoner.Provider.(*OpenRouterProvider)
	provider.endpoint = server.URL
	provider.client = server.Client()
	policy, e := reasoner.CurrentPolicy(context.Background())
	if e != nil {
		t.Fatal(e)
	}
	_, bill, e := reasoner.Complete(context.Background(), policy, statementextract.Chunk{Index: 1, Spans: []statementextract.RedactedSpan{{ID: "s0", Page: 1, Text: "-1.00", LexicalClass: statementextract.ClassNumericFragment}}})
	if e == nil || bill == nil || bill.CostMicros != 10000 || posts != 1 {
		t.Fatal("lost malformed billing", e, bill, posts)
	}
	standard, e := NewStatementOpenRouterProvider("synthetic", server.Client())
	if e != nil {
		t.Fatal(e)
	}
	if _, e = standard.reservation(reasoner.Policy, 100000, 16000); e == nil {
		t.Fatal("standard transport accepted candidate")
	}
	product := *reasoner
	product.screening = false
	if _, e = product.CurrentPolicy(context.Background()); e == nil {
		t.Fatal("standard reasoner accepted candidate")
	}
	// Stale/absent privacy membership fails before POST.
	reasoner.screeningVerified = time.Time{}
	reasoner.Verifier.endpoint = server.URL + "/missing/models/"
	if _, e = reasoner.CurrentPolicy(context.Background()); e == nil || posts != 1 {
		t.Fatal("unverified privacy reached inference")
	}
}
