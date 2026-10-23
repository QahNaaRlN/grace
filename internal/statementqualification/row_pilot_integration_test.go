//go:build integration

package statementqualification

import (
	"context"
	"github.com/QahNaaRlN/grace/internal/pdfextract"
	"github.com/QahNaaRlN/grace/internal/statementextract"
	"os"
	"strings"
	"testing"
)

func TestRowPilotRealHelper(t *testing.T) {
	socket := os.Getenv("TEST_PDF_EXTRACTOR_SOCKET")
	if socket == "" {
		t.Fatal("pinned helper required")
	}
	helper, e := pdfextract.NewClient(socket)
	if e != nil {
		t.Fatal(e)
	}
	raw, e := os.ReadFile("../pdfextract/testdata/cards.pdf")
	if e != nil {
		t.Fatal(e)
	}
	d, e := helper.Extract(context.Background(), raw)
	if e != nil || d.Hash() != statementextract.ClockCardsDocumentHash {
		t.Fatal("actual pinned document", e)
	}
	f := &rowPilotFake{t: t, mode: "keep"}
	m := &rowPilotTestMeter{}
	r, e := RunRowPilot(context.Background(), Sample{"cards", d, FixtureHash("cards")}, f, m, strings.Repeat("a", 40))
	if e != nil || r.Decision != RowPilotKeep || len(r.Calls) != 12 || ValidateRowPilotReport(r, d) != nil {
		t.Fatal("actual helper pipeline", e, r.Decision)
	}
	s, e := SummarizeRowPilot(r, d)
	if e != nil || ValidateRowPilotSummary(s, r.Source) != nil {
		t.Fatal("replay projection", e)
	}
	for _, a := range r.Attempts {
		if a.Measurement == nil || len(a.Measurement.Checks) != 20 {
			t.Fatal("full trace")
		}
		if a.Variant == "candidate" && (!a.Measurement.FullQualityPassed || len(a.Measurement.Precision) != 2) {
			t.Fatal("full target")
		}
	}
	t.Logf("S121_PILOT_HELPER calls=%d reserve=%d settle=%d candidate=20/20 precision=4/4 decision=%s FAKE_ONLY", len(r.Calls), m.reserves, m.settles, r.Decision)
}
