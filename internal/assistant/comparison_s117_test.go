package assistant

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/QahNaaRlN/grace/internal/statementdata"
	"github.com/QahNaaRlN/grace/internal/statementextract"
	"github.com/QahNaaRlN/grace/internal/statementqualification"
)

type s117WireMeter struct {
	reserves, settles, allowed, blockAfter int
	reserved                               int64
	actual                                 *int64
	settleFail                             bool
}

func (m *s117WireMeter) Reserve(_ context.Context, n int64, _ statementextract.Policy) (string, error) {
	m.reserves++
	m.reserved = n
	return "66666666-7777-4888-8999-000000000000", nil
}
func (m *s117WireMeter) ComparisonAllowed(context.Context, string) (bool, error) {
	m.allowed++
	return m.blockAfter == 0 || m.allowed <= m.blockAfter, nil
}
func (m *s117WireMeter) Settle(ctx context.Context, _ string, n *int64) error {
	m.settles++
	m.actual = n
	if d, ok := ctx.Deadline(); !ok || time.Until(d) > 5*time.Second {
		return errors.New("settlement bound")
	}
	if m.settleFail {
		return errors.New("PRIVATE settlement")
	}
	return nil
}

// Actual HTTP bytes from the fixed reasoners, through the S1.17 engine. The
// server returns wire proposals, never canonical source IDs or report fixtures.
func TestS117CampaignActualHTTP(t *testing.T) {
	for _, mode := range []string{"quality", "protocol", "unknown", "identity", "shared-block", "settlement", "sixth-plan"} {
		t.Run(mode, func(t *testing.T) {
			cs := statementextract.ComparisonCandidates()
			samples := []statementqualification.Sample{}
			for j, n := range statementqualification.ScreeningNames() {
				d := statementdata.Document{Extractor: statementdata.ExtractorVersion, Pages: []statementdata.Page{{Number: 1, Width: 600, Height: 800}}}
				for i, text := range []string{fmt.Sprintf("%d.10.2026", 10+j), "-12.34", "RUB"} {
					d.Spans = append(d.Spans, statementdata.Span{ID: fmt.Sprintf("PRIVATE-source-%d", i), Page: 1, Block: 1, Line: 1, Word: i + 1, Text: text, Box: statementdata.Box{XMin: float64(i * 100), YMin: 1, XMax: float64(i*100 + 50), YMax: 10}})
				}
				samples = append(samples, statementqualification.Sample{Name: n, Document: d, SHA256: statementqualification.FixtureHash(n)})
			}
			plans := []statementextract.Plan{}
			for _, s := range samples {
				for _, c := range cs {
					p, e := statementextract.BuildComparisonPlan(s.Document, c.Policy)
					if e != nil {
						t.Fatal(e)
					}
					plans = append(plans, p)
				}
			}
			posts := 0
			m := &s117WireMeter{settleFail: mode == "settlement"}
			if mode == "shared-block" {
				m.blockAfter = 1
			}
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				if req.Method != "POST" {
					t.Error("unexpected HTTP method")
					w.WriteHeader(400)
					return
				}
				i := posts
				posts++
				if i >= 6 {
					t.Error("retry or fallback")
					w.WriteHeader(400)
					return
				}
				if m.reserves != 1 || m.reserved != 208500 || m.allowed != posts {
					t.Error("call without aggregate reserve/allowed")
				}
				raw, _ := io.ReadAll(req.Body)
				if strings.Contains(string(raw), "PRIVATE") {
					t.Error("source identity leaked")
				}
				var body struct {
					Model     string       `json:"model"`
					Messages  []LLMMessage `json:"messages"`
					MaxTokens int          `json:"max_tokens"`
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
					Format struct {
						Type   string `json:"type"`
						Schema struct {
							Strict bool            `json:"strict"`
							Value  json.RawMessage `json:"schema"`
						} `json:"json_schema"`
					} `json:"response_format"`
				}
				if json.Unmarshal(raw, &body) != nil {
					t.Error("request JSON")
					return
				}
				c := cs[i%2]
				chunk := plans[i].Chunks[0]
				if body.Model != c.Policy.Model || !reflect.DeepEqual(body.Provider.Only, []string{c.Policy.Provider}) || body.Provider.Fallback || !body.Provider.ZDR || body.Provider.Collection != "deny" || !body.Provider.Require || body.MaxTokens != 16000 || body.Reasoning.Enabled || body.Format.Type != "json_schema" || !body.Format.Schema.Strict {
					t.Error("wire policy/order")
				}
				expected := []LLMMessage{}
				for _, v := range statementextract.Messages(chunk) {
					expected = append(expected, LLMMessage{Role: v["role"], Content: v["content"]})
				}
				var got, want map[string]any
				json.Unmarshal(body.Format.Schema.Value, &got)
				json.Unmarshal(statementextract.RequestSchema(chunk), &want)
				if !reflect.DeepEqual(got, want) || !reflect.DeepEqual(body.Messages, expected) {
					t.Error("request differs from approved tuple")
				}
				defs := got["$defs"].(map[string]any)
				if !reflect.DeepEqual(defs["alias"].(map[string]any)["enum"], []any{"s0", "s1", "s2"}) || !reflect.DeepEqual(defs["moneySelection"].(map[string]any)["items"].(map[string]any)["enum"], []any{"s1"}) {
					t.Error("v3 membership missing")
				}
				if mode == "unknown" {
					w.WriteHeader(502)
					return
				}
				wire := comparisonWire(c)
				wire["id"] = fmt.Sprintf("PRIVATE-generation-%d", posts)
				if mode == "identity" {
					wire["provider"] = "PRIVATE wrong provider"
				}
				if mode == "protocol" {
					wire["choices"].([]any)[0].(map[string]any)["message"].(map[string]any)["content"] = "PRIVATE malformed"
				}
				json.NewEncoder(w).Encode(wire)
			}))
			defer server.Close()
			var rs [2]statementextract.Reasoner
			for i, c := range cs {
				r := comparisonTestReasoner(t, c)
				r.client = doerFunc(func(req *http.Request) (*http.Response, error) {
					return jsonResponse(comparisonMetadataFixture(c, req.URL.Path)), nil
				})
				r.provider.client = server.Client()
				r.provider.endpoint = server.URL
				rs[i] = r
			}
			if mode == "sixth-plan" {
				samples[2].Document.Pages = nil
			}
			e, err := statementqualification.RunS117Comparison(context.Background(), samples, rs, m)
			if err == nil {
				t.Fatal("synthetic quality cannot pass")
			}
			if mode == "sixth-plan" {
				if posts != 0 || m.reserves != 0 {
					t.Fatal("unplanned tuples reserved or sent")
				}
				return
			}
			if m.reserves != 1 || m.settles != 1 || m.reserved != 208500 {
				t.Fatal("aggregate accounting")
			}
			expected := 6
			if mode == "protocol" || mode == "unknown" || mode == "identity" || mode == "shared-block" {
				expected = 1
			}
			if posts != expected {
				t.Fatal("stop/continuation", posts, expected)
			}
			if mode == "unknown" || mode == "identity" {
				if e.Report.ActualMicros != nil || m.actual != nil {
					t.Fatal("unknown cost guessed")
				}
			} else if e.Report.ActualMicros == nil || *e.Report.ActualMicros != int64(posts*1000) {
				t.Fatal("known billing lost")
			}
			if mode == "settlement" && e.Report.Error != "qualification_settlement_failed" {
				t.Fatal("settlement failure hidden")
			}
			if statementqualification.ValidateS117Comparison(e, time.Now()) != nil {
				t.Fatal("actual report invalid")
			}
			for i, tuple := range e.Report.Tuples {
				request := struct {
					Policy   statementextract.Policy
					Messages []map[string]string
					Schema   any
				}{cs[i%2].Policy, statementextract.Messages(plans[i].Chunks[0]), statementextract.RequestSchema(plans[i].Chunks[0])}
				if tuple.RequestHash != statementextract.Hash(request) || tuple.Evidence.PlanHash != statementextract.Hash(plans[i]) || e.Report.Profile.Planner != "statement-document-atom.v3.alias-enum" {
					t.Fatal("actual profile/plan/request differs")
				}
			}
		})
	}
}
