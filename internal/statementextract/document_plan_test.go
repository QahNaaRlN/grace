package statementextract

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QahNaaRlN/grace/internal/statementdata"
	"github.com/go-chi/chi/v5"
)

func TestDocumentPlanExactByteAndSpanBoundaries(t *testing.T) {
	d := testDocument("a")
	p, e := BuildPlan(d, testPolicy(), []string{"span-0"})
	if e != nil {
		t.Fatal(e)
	}
	overhead := InputBytes(p.Chunks[0]) - 1
	for _, target := range []int{89999, 90000, 90001} {
		d.Spans[0].Text = strings.Repeat("a", target-overhead)
		p, e := BuildPlan(d, testPolicy(), []string{"span-0"})
		if target <= MaxInputBytes {
			if e != nil || len(p.Chunks) != 1 || InputBytes(p.Chunks[0]) != target || !validPlan(p) {
				t.Fatal(target, e)
			}
		} else if e == nil || e.Error() != "statement_atomic_block_limit" || len(p.Chunks) != 0 {
			t.Fatal("partial/oversize plan", target, e)
		}
	}
	values := make([]string, MaxPlanSpans)
	for i := range values {
		values[i] = "RUB"
	}
	d = testDocument(values...)
	p, e = BuildPlan(d, testPolicy(), nil)
	if e != nil || len(p.Aliases) != 2000 || len(p.Chunks) != 1 || !validPlan(p) {
		t.Fatal("2000 boundary", e)
	}
	d = testDocument(append(values, "RUB")...)
	if p, e := BuildPlan(d, testPolicy(), nil); e == nil || e.Error() != "statement_document_span_limit" || len(p.Chunks) != 0 {
		t.Fatal("span guard", e)
	}
}
func TestDocumentAliasesRoundTripAllTypedSelections(t *testing.T) {
	d := testDocument("19.10.2026", "-12.34", "RUB", "SYNTHETIC")
	before := d.Hash()
	p, e := BuildPlan(d, testPolicy(), nil)
	if e != nil {
		t.Fatal(e)
	}
	in := statementdata.Proposal{Rows: []statementdata.RowMapping{{OperationDate: statementdata.Selection{"s0"}, BookingDate: statementdata.Selection{"s0"}, AccountAmount: statementdata.Selection{"s1"}, AccountCurrency: statementdata.Selection{"s2"}, TransactionAmount: statementdata.Selection{"s1"}, TransactionCurrency: statementdata.Selection{"s2"}, Description: statementdata.Selection{"s3"}, Direction: "debit", DirectionEvidence: statementdata.Selection{"s3"}}}, Header: statementdata.HeaderMapping{OpeningBookDate: statementdata.Selection{"s0"}, ClosingBookDate: statementdata.Selection{"s0"}, OpeningBookBalance: statementdata.Selection{"s1"}, ClosingBookBalance: statementdata.Selection{"s1"}, AvailableBalanceObserved: statementdata.Selection{"s1"}, DebitTurnover: statementdata.Selection{"s1"}, CreditTurnover: statementdata.Selection{"s1"}, Currency: statementdata.Selection{"s2"}, Labels: map[string]statementdata.Selection{"openingBookBalance": {"s3"}}}, Service: []statementdata.ServiceMapping{{Spans: statementdata.Selection{"s3"}, Reason: "literal s0 must stay unchanged"}}}
	inputHash := Hash(in)
	out, e := CanonicalProposal(p, in)
	if e != nil {
		t.Fatal(e)
	}
	raw, _ := json.Marshal(out)
	for _, s := range []string{"span-0", "span-1", "span-2", "span-3", "literal s0 must stay unchanged", "debit"} {
		if !strings.Contains(string(raw), s) {
			t.Fatal("typed reference lost", s)
		}
	}
	if Hash(in) != inputHash || d.Hash() != before || out.Rows[0].AccountCurrency[0] != out.Header.Currency[0] {
		t.Fatal("source/input mutated or shared currency broken")
	}
	for _, bad := range []string{"span-0", "foreign", "szzzz"} {
		candidate := in
		candidate.Rows = append([]statementdata.RowMapping{}, in.Rows...)
		candidate.Rows[0].OperationDate = statementdata.Selection{bad}
		if _, e := CanonicalProposal(p, candidate); e == nil {
			t.Fatal("invalid wire ID accepted", bad)
		}
	}
	duplicate := in
	duplicate.Service = []statementdata.ServiceMapping{{Spans: statementdata.Selection{"s3", "s3"}, Reason: "duplicate"}}
	if _, e := CanonicalProposal(p, duplicate); e == nil {
		t.Fatal("duplicate wire selection accepted")
	}
}
func TestDocumentPlanContractRejectsMutationsAndLegacyConsent(t *testing.T) {
	p, e := BuildPlan(testDocument("RUB", "-1.00"), testPolicy(), nil)
	if e != nil {
		t.Fatal(e)
	}
	raw, _ := json.Marshal(p)
	changes := []func(*Plan){func(p *Plan) { p.PlannerVersion = "old" }, func(p *Plan) { p.AliasVersion = "old" }, func(p *Plan) { p.PromptVersion = "statement-span-mapping.v7.explicit-per-row-currency" }, func(p *Plan) { p.Limits.MaxInputBytes++ }, func(p *Plan) { p.Aliases["s0"] = p.Aliases["s1"] }, func(p *Plan) { p.Atoms[0][0] = "foreign" }, func(p *Plan) { p.Chunks[0].Spans[0].ID = "foreign" }, func(p *Plan) { p.Chunks[0].Context = []RedactedSpan{{ID: "s0"}} }, func(p *Plan) { p.MaximumMicros++ }}
	for i, change := range changes {
		var changed Plan
		if e := json.Unmarshal(raw, &changed); e != nil {
			t.Fatal(e)
		}
		change(&changed)
		if validPlan(changed) {
			t.Fatal("consent mutation accepted", i)
		}
	}
	// Simulate the older binary's typed decoder, which drops all added plan fields.
	var old struct {
		DocumentHash     string  `json:"documentHash"`
		PromptVersion    string  `json:"promptVersion"`
		SchemaHash       string  `json:"schemaHash"`
		RedactionVersion string  `json:"redactionVersion"`
		Policy           Policy  `json:"policy"`
		Chunks           []Chunk `json:"chunks"`
		MaximumMicros    int64   `json:"maximumMicros"`
	}
	if e := json.Unmarshal(raw, &old); e != nil {
		t.Fatal(e)
	}
	if Hash(old) == Hash(p) || old.PromptVersion == "statement-span-mapping.v7.explicit-per-row-currency" {
		t.Fatal("old binary could reuse new consent")
	}
	legacy := Plan{DocumentHash: p.DocumentHash, PromptVersion: "statement-span-mapping.v7.explicit-per-row-currency", SchemaHash: p.SchemaHash, Policy: p.Policy, Chunks: p.Chunks, MaximumMicros: 8000}
	if validPlan(legacy) {
		t.Fatal("new binary accepted old consent")
	}
}
func TestExtractionFailsBeforeServiceWhenWriteDeadlineUnavailable(t *testing.T) {
	router := chi.NewRouter()
	NewHandler(nil).Routes(router)
	request := httptest.NewRequest(http.MethodPost, "/statements/00000000-0000-0000-0000-000000000001/extractions", strings.NewReader(`{"expectedRevision":1,"previewId":"00000000-0000-0000-0000-000000000002","previewHash":"hash","consentId":"00000000-0000-0000-0000-000000000003"}`))
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	if recorder.Code != 503 || !strings.Contains(recorder.Body.String(), "statement_deadline_unavailable") {
		t.Fatal(recorder.Code, recorder.Body.String())
	}
}
