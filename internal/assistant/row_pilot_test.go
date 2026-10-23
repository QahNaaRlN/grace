package assistant

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/QahNaaRlN/grace/internal/statementdata"
	"github.com/QahNaaRlN/grace/internal/statementextract"
)

func rowHTTPDocument(t *testing.T) statementdata.Document {
	t.Helper()
	raw, e := os.ReadFile("../statementextract/testdata/s119-cards-document.json")
	if e != nil {
		t.Fatal(e)
	}
	var d statementdata.Document
	if json.Unmarshal(raw, &d) != nil {
		t.Fatal("fixture")
	}
	return d
}
func rowHTTPGrouping(t *testing.T, d statementdata.Document) statementextract.RowGrouping {
	t.Helper()
	p, e := statementextract.BuildComparisonPlan(d, statementextract.ComparisonCandidates()[0].Policy)
	if e != nil {
		t.Fatal(e)
	}
	labels := map[string]statementdata.Selection{}
	for _, k := range []string{"openingBookBalance", "closingBookBalance", "availableBalanceObserved", "debitTurnover", "creditTurnover"} {
		labels[k] = []string{}
	}
	h := statementdata.HeaderMapping{OpeningBookDate: []string{}, ClosingBookDate: []string{}, OpeningBookBalance: []string{}, ClosingBookBalance: []string{}, AvailableBalanceObserved: []string{}, DebitTurnover: []string{}, CreditTurnover: []string{}, Currency: []string{}, Labels: labels}
	ids := []string{}
	for _, s := range p.Chunks[0].Spans[2:] {
		ids = append(ids, s.ID)
	}
	return statementextract.RowGrouping{Rows: []statementextract.RowEnvelope{{Spans: []string{p.Chunks[0].Spans[0].ID}}, {Spans: []string{p.Chunks[0].Spans[1].ID}}}, Context: []string{}, Header: h, Service: []statementdata.ServiceMapping{{Spans: ids, Reason: "Deliberately incorrect public wire-only control; no quality claim"}}, Unresolved: []string{}}
}
func TestRowPilotActualHTTPBindingAndCaps(t *testing.T) {
	d := rowHTTPDocument(t)
	before := d.Hash()
	old, e := statementextract.BuildClockRequest(d, "base")
	if e != nil {
		t.Fatal(e)
	}
	current, e := statementextract.BuildRowPilotBase(d)
	if e != nil || !reflect.DeepEqual(current.Messages, old.Messages) || !reflect.DeepEqual(current.Schema, old.Schema) || current.Policy != old.Plan.Policy || current.InputBytes != old.InputBytes {
		t.Fatal("v12 base wire changed", e)
	}
	g := rowHTTPGrouping(t, d)
	c := statementextract.ComparisonCandidates()[0]
	expected := make(chan statementextract.RowScopedRequest, 1)
	var posts atomic.Int32
	var gets atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.Method == "GET" {
			gets.Add(1)
			if req.Header.Get("Authorization") != "" {
				t.Error("metadata key")
			}
			_ = json.NewEncoder(w).Encode(comparisonMetadataFixture(c, strings.TrimPrefix(req.URL.Path, "/api/v1")))
			return
		}
		n := posts.Add(1)
		var q statementextract.RowScopedRequest
		select {
		case q = <-expected:
		default:
			t.Error("unapproved/repeated HTTP")
			w.WriteHeader(500)
			return
		}
		b, _ := io.ReadAll(req.Body)
		var got map[string]any
		_ = json.Unmarshal(b, &got)
		var messages, schema any
		raw, _ := json.Marshal(q.Messages)
		_ = json.Unmarshal(raw, &messages)
		_ = json.Unmarshal(q.Schema, &schema)
		rf := got["response_format"].(map[string]any)["json_schema"].(map[string]any)
		p := got["provider"].(map[string]any)
		if !reflect.DeepEqual(got["messages"], messages) || !reflect.DeepEqual(rf["schema"], schema) || rf["strict"] != true || got["max_tokens"] != float64(16000) || got["model"] != c.Policy.Model || got["tools"] != nil || got["reasoning"].(map[string]any)["enabled"] != false || p["allow_fallbacks"] != false || p["zdr"] != true || p["data_collection"] != "deny" || !reflect.DeepEqual(p["only"], []any{c.Policy.Provider}) {
			t.Error("wire differs")
		}
		if req.Header.Get("Authorization") != "Bearer synthetic" {
			t.Error("missing scoped key")
		}
		content := `{}`
		if q.Stage == "grouping" {
			v, _ := json.Marshal(g)
			content = string(v)
		}
		if q.Stage == "row" {
			row := statementdata.RowMapping{OperationDate: []string{}, BookingDate: []string{}, AccountAmount: []string{}, AccountCurrency: []string{}, TransactionAmount: []string{}, TransactionCurrency: []string{}, Description: g.Rows[q.RowOrdinal-1].Spans, DirectionEvidence: []string{}}
			v, _ := json.Marshal(row)
			content = string(v)
		}
		response := comparisonWire(c)
		response["id"] = strings.Repeat("g", int(n))
		response["choices"].([]any)[0].(map[string]any)["message"].(map[string]any)["content"] = content
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()
	r, e := NewRowPilotReasoner("synthetic", d)
	if e != nil {
		t.Fatal(e)
	}
	r.comparison.base = server.URL + "/api/v1/"
	r.comparison.provider.endpoint = server.URL + "/api/v1/chat/completions"
	if _, e = r.CurrentPolicy(context.Background()); e != nil {
		t.Fatal(e)
	}
	proof, e := r.ComparisonProof(context.Background())
	if e != nil {
		t.Fatal(e)
	}
	grouping, e := statementextract.BuildRowGrouping(d)
	if e != nil {
		t.Fatal(e)
	}
	// A changed prompt/request is refused before POST, even with a copied old hash.
	bad := grouping
	bad.Messages = []map[string]string{{"role": "system", "content": "PRIVATE override"}, {"role": "user", "content": "{}"}}
	if _, _, e = r.CompleteRowPilot(context.Background(), bad, proof); e == nil || posts.Load() != 0 {
		t.Fatal("override reached transport")
	}
	expected <- grouping
	if _, _, e = r.CompleteRowPilot(context.Background(), grouping, proof); e != nil {
		t.Fatal(e)
	}
	for i := 1; i <= 2; i++ {
		q, e := statementextract.BuildRowFields(d, g, i)
		if e != nil {
			t.Fatal(e)
		}
		expected <- q
		if _, _, e = r.CompleteRowPilot(context.Background(), q, proof); e != nil {
			t.Fatal(e)
		}
	}
	base, e := statementextract.BuildRowPilotBase(d)
	if e != nil {
		t.Fatal(e)
	}
	for posts.Load() < 12 {
		expected <- base
		if _, _, e = r.CompleteRowPilot(context.Background(), base, proof); e != nil {
			t.Fatal(e)
		}
	}
	if _, _, e = r.CompleteRowPilot(context.Background(), base, proof); e == nil || posts.Load() != 12 {
		t.Fatal("POST cap")
	}
	if gets.Load() != 3 || r.MetadataRequests() != 3 || d.Hash() != before {
		t.Fatal("metadata/source", gets.Load(), r.MetadataRequests())
	}
	// Force every permitted metadata refresh; request40 is rejected before network.
	for i := 1; i < 13; i++ {
		r.comparison.mu.Lock()
		r.comparison.proof = nil
		r.comparison.mu.Unlock()
		if _, e = r.ComparisonProof(context.Background()); e != nil {
			t.Fatal(e)
		}
	}
	r.comparison.mu.Lock()
	r.comparison.proof = nil
	r.comparison.mu.Unlock()
	if _, e = r.ComparisonProof(context.Background()); e == nil || gets.Load() != 39 || r.MetadataRequests() != 39 {
		t.Fatal("metadata cap", gets.Load(), e)
	}
}
