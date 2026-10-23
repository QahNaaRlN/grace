package assistant

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/QahNaaRlN/grace/internal/statementdata"
	"github.com/QahNaaRlN/grace/internal/statementextract"
	"github.com/QahNaaRlN/grace/internal/statementqualification"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
)

func TestClockActualHTTPAndExactPrice(t *testing.T) {
	for _, mode := range []string{"valid", "lower-price", "protocol", "unknown"} {
		t.Run(mode, func(t *testing.T) {
			raw, e := os.ReadFile("../statementextract/testdata/s119-cards-document.json")
			if e != nil {
				t.Fatal(e)
			}
			var d statementdata.Document
			if json.Unmarshal(raw, &d) != nil {
				t.Fatal("fixture")
			}
			var posts atomic.Int32
			c := statementextract.ComparisonCandidates()[0]
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				if req.Method == "GET" {
					if req.Header.Get("Authorization") != "" {
						t.Error("metadata key")
					}
					v := comparisonMetadataFixture(c, req.URL.Path)
					if mode == "lower-price" && strings.HasSuffix(req.URL.Path, "/endpoints") {
						v["data"].(map[string]any)["endpoints"].([]any)[0].(map[string]any)["pricing"].(map[string]any)["prompt"] = "0.0000001"
					}
					_ = json.NewEncoder(w).Encode(v)
					return
				}
				i := int(posts.Add(1)) - 1
				if i >= 6 {
					t.Error("retry")
					w.WriteHeader(500)
					return
				}
				q, e := statementextract.BuildClockRequest(d, statementqualification.ClockOrder()[i])
				if e != nil {
					t.Error(e)
					return
				}
				b, _ := io.ReadAll(req.Body)
				var body map[string]any
				if json.Unmarshal(b, &body) != nil {
					t.Error("body")
				}
				var got []LLMMessage
				for _, x := range body["messages"].([]any) {
					m := x.(map[string]any)
					got = append(got, LLMMessage{Role: m["role"].(string), Content: m["content"]})
				}
				want := []LLMMessage{}
				for _, x := range q.Messages {
					want = append(want, LLMMessage{Role: x["role"], Content: x["content"]})
				}
				schema := body["response_format"].(map[string]any)["json_schema"].(map[string]any)
				var expected any
				_ = json.Unmarshal(q.Schema, &expected)
				p := body["provider"].(map[string]any)
				if !reflect.DeepEqual(got, want) || !reflect.DeepEqual(schema["schema"], expected) || schema["strict"] != true || body["max_tokens"] != float64(16000) || body["tools"] != nil || body["reasoning"].(map[string]any)["enabled"] != false || p["allow_fallbacks"] != false || p["zdr"] != true || p["data_collection"] != "deny" || p["require_parameters"] != true || !reflect.DeepEqual(p["only"], []any{c.Policy.Provider}) {
					t.Error("actual HTTP differs from approved request")
				}
				if mode == "unknown" {
					w.WriteHeader(502)
					return
				}
				response := comparisonWire(c)
				response["id"] = fmt.Sprintf("PRIVATE-clock-%d", i)
				row := `{"operationDate":[],"bookingDate":[],"accountAmount":[],"accountCurrency":[],"transactionAmount":[],"transactionCurrency":[],"description":[],"direction":"","directionEvidence":[]}`
				content := `{"rows":[` + row + `,` + row + `],"header":{"openingBookDate":[],"closingBookDate":[],"openingBookBalance":[],"closingBookBalance":[],"availableBalanceObserved":[],"debitTurnover":[],"creditTurnover":[],"currency":[],"labels":{"openingBookBalance":[],"closingBookBalance":[],"availableBalanceObserved":[],"debitTurnover":[],"creditTurnover":[]}},"service":[]}`
				if mode == "protocol" {
					content = "PRIVATE not JSON"
				}
				response["choices"].([]any)[0].(map[string]any)["message"].(map[string]any)["content"] = content
				_ = json.NewEncoder(w).Encode(response)
			}))
			defer server.Close()
			r, e := NewClockPilotReasoner("synthetic")
			if e != nil {
				t.Fatal(e)
			}
			r.comparison.base = server.URL + "/"
			r.comparison.client = server.Client()
			r.comparison.provider.client = server.Client()
			r.comparison.provider.endpoint = server.URL + "/chat/completions"
			m := &s117WireMeter{}
			// Same fake meter protocol, updated total cap is checked by the new engine.
			report, e := statementqualification.RunClockPilot(context.Background(), statementqualification.Sample{Name: "cards", Document: d, SHA256: statementqualification.FixtureHash("cards")}, r, m, strings.Repeat("a", 40))
			if mode == "valid" {
				if e != nil || posts.Load() != 6 || report.Decision != statementqualification.ClockInconclusive || m.reserved != 323400 || statementqualification.ValidateClockReport(report, d) != nil {
					t.Fatal(report.Decision, posts.Load(), e)
				}
			} else if e == nil {
				t.Fatal("failure accepted")
			}
			if mode == "lower-price" && (posts.Load() != 0 || m.reserves != 0) {
				t.Fatal("changed price reached inference")
			}
			if mode == "protocol" && posts.Load() != 1 {
				t.Fatal("protocol retry")
			}
			if mode == "unknown" && (posts.Load() != 1 || m.actual != nil) {
				t.Fatal("unknown billing")
			}
		})
	}
}
