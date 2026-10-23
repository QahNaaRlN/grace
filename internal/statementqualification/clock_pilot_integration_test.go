//go:build integration

package statementqualification

import (
	"context"
	"encoding/json"
	"github.com/QahNaaRlN/grace/internal/pdfextract"
	"github.com/QahNaaRlN/grace/internal/statementdata"
	"github.com/QahNaaRlN/grace/internal/statementextract"
	"os"
	"reflect"
	"testing"
)

func TestClockRealHelper(t *testing.T) {
	socket := os.Getenv("TEST_PDF_EXTRACTOR_SOCKET")
	if socket == "" {
		t.Fatal("pinned helper required")
	}
	helper, e := pdfextract.NewClient(socket)
	if e != nil {
		t.Fatal(e)
	}
	b, e := os.ReadFile("../pdfextract/testdata/cards.pdf")
	if e != nil {
		t.Fatal(e)
	}
	d, e := helper.Extract(context.Background(), b)
	if e != nil || d.Hash() != statementextract.ClockCardsDocumentHash {
		t.Fatal("helper document identity", d.Hash(), e)
	}
	if !reflect.DeepEqual(d, clockTestDocument(t)) {
		t.Fatal("public unit fixture diverges from real helper")
	}
	p := clockTestProposal(t, d)
	before, _ := json.Marshal(p)
	good, e := MeasureClock(d, p)
	if e != nil || *good.Count != 4 || !good.Result.Passed {
		t.Fatal("good", good, e)
	}
	bad, e := MeasureClock(d, clockDrop(p))
	if e != nil || *bad.Count != 0 || bad.Result.Correct != 14 || len(bad.Result.Coverage) == 0 {
		t.Fatal("bad", bad, e)
	}
	after, _ := json.Marshal(p)
	if string(before) != string(after) {
		t.Fatal("proposal mutated")
	}
	columns, e := os.ReadFile("../pdfextract/testdata/columns.pdf")
	if e != nil {
		t.Fatal(e)
	}
	col, e := helper.Extract(context.Background(), columns)
	if e != nil {
		t.Fatal(e)
	}
	q, e := statementextract.BuildClockRequest(col, "base")
	if e != nil {
		t.Fatal(e)
	}
	f := &activationFixtureReasoner{}
	proposal, _, e := f.Complete(context.Background(), q.Plan.Policy, q.Plan.Chunks[0])
	if e != nil {
		t.Fatal(e)
	}
	canonical, e := statementextract.CanonicalComparisonProposal(q.Plan, proposal)
	if e != nil {
		t.Fatal(e)
	}
	review, e := statementdata.ValidateProposal(col, canonical, []int{1}, true, "")
	if e != nil || review.Rows[0].OperationDate.Precision != "date" {
		t.Fatal("date-only", e)
	}
	proposal.Rows[0].OperationDate = append(proposal.Rows[0].OperationDate, "invented-clock")
	if _, e := statementextract.CanonicalComparisonProposal(q.Plan, proposal); e == nil {
		t.Fatal("invented time accepted")
	}
	t.Logf("CLOCK_CAUSAL helper=%s good=%d/%d clocks=%d bad=%d/%d clocks=%d; coverage failure preserved", d.Extractor, good.Result.Correct, good.Result.Checks, *good.Count, bad.Result.Correct, bad.Result.Checks, *bad.Count)
}
