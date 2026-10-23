//go:build integration

package statementqualification

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/QahNaaRlN/grace/internal/pdfextract"
	"github.com/QahNaaRlN/grace/internal/statementdata"
	"github.com/QahNaaRlN/grace/internal/statementextract"
	"os"
	"os/exec"
	"testing"
	"time"
)

// The fake can select only aliases delivered in its Chunk; it cannot consult golden data.
type qualityFixtureFake struct {
	comparisonFixtureReasoner
	breakClocks, breakRoles bool
}

func (f *qualityFixtureFake) Complete(ctx context.Context, p statementextract.Policy, c statementextract.Chunk) (statementdata.Proposal, *statementextract.Billing, error) {
	part, bill, err := f.comparisonFixtureReasoner.Complete(ctx, p, c)
	part.Service = nil // Actual unmapped membership; not a guessed service classification.
	ru := false
	for _, s := range c.Spans {
		ru = ru || s.Text == "12.10.2026"
	}
	if f.breakClocks && !ru {
		for i := range part.Rows {
			if len(part.Rows[i].OperationDate) > 1 {
				part.Rows[i].OperationDate = part.Rows[i].OperationDate[:1]
			}
			if len(part.Rows[i].BookingDate) > 1 {
				part.Rows[i].BookingDate = part.Rows[i].BookingDate[:1]
			}
		}
	}
	if f.breakRoles && ru {
		for i := range part.Rows {
			part.Rows[i].BookingDate = part.Rows[i].OperationDate
		}
		part.Header.DebitTurnover = nil
	}
	return part, bill, err
}
func TestS118RealHelperDiagnosis(t *testing.T) {
	socket := os.Getenv("TEST_PDF_EXTRACTOR_SOCKET")
	if socket == "" {
		t.Skip("pinned isolated helper required")
	}
	helper, err := pdfextract.NewClient(socket)
	if err != nil {
		t.Fatal(err)
	}
	samples := []Sample{}
	for _, name := range ScreeningNames() {
		raw, err := os.ReadFile("../pdfextract/testdata/" + name + ".pdf")
		if err != nil {
			t.Fatal(err)
		}
		d, err := helper.Extract(context.Background(), raw)
		if err != nil {
			t.Fatal(err)
		}
		samples = append(samples, Sample{name, d, FixtureHash(name)})
	}
	cs := statementextract.ComparisonCandidates()
	a := &qualityFixtureFake{comparisonFixtureReasoner: comparisonFixtureReasoner{candidate: cs[0]}, breakClocks: true, breakRoles: true}
	b := &qualityFixtureFake{comparisonFixtureReasoner: comparisonFixtureReasoner{candidate: cs[1]}, breakRoles: true}
	e, err := RunS117Comparison(context.Background(), samples, [2]statementextract.Reasoner{a, b}, &fakeMeter{})
	if err == nil || e.Report.Error != "qualification_quality_failed" || a.calls != 3 || b.calls != 3 {
		t.Fatal("fake campaign", err)
	}
	var raw bytes.Buffer
	if EncodeS117Comparison(&raw, e) != nil {
		t.Fatal("writer")
	}
	_, decoded, err := DecodeS117ComparisonArtifact(&raw)
	if err != nil {
		t.Fatal(err)
	}
	out, err := DiagnoseSavedQuality(*decoded, samples, qualityTestVerifier, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	out.ArtifactsUnchanged = true
	if err := ValidateQualityDiagnosis(out); err != nil {
		v, _ := json.Marshal(out)
		t.Fatal(err, string(v))
	}
	unknown, clock, printed := false, false, false
	for i, row := range out.Tuples {
		if row.Status != "VERIFIED" {
			t.Fatal("replay", row)
		}
		scored, checks := ScreeningChecks(ScreeningNames()[i/2], func() statementdata.Review {
			p, _ := statementextract.BuildComparisonPlan(samples[i/2].Document, cs[i%2].Policy)
			merged, _ := statementextract.MergeProposals(p, []statementdata.Proposal{*e.Report.Tuples[i].Proposal})
			r, _ := statementdata.ValidateProposal(samples[i/2].Document, merged, []int{1}, true, "")
			return r
		}())
		if scored.Checks != []int{19, 19, 20, 20, 23, 23}[i] || len(checks) != scored.Checks {
			t.Fatal("legacy counts")
		}
		for _, r := range row.Reasons {
			unknown = unknown || r.Status == "VERIFIED" && r.PublicCode == "unknown" && r.Code == "overlapping_evidence"
		}
		for _, c := range row.Checks {
			if c.Status != "VERIFIED" {
				t.Fatal("causal witness", c)
			}
			clock = clock || c.Clock != nil && c.Clock.Time == "differs"
			printed = printed || c.Money != nil && c.Money.Amount == "missing"
		}
		if i < 4 && (row.Coverage.TotalUnmapped == 0 || len(row.Coverage.Positions) != row.Coverage.TotalUnmapped) {
			t.Fatal("unmapped membership")
		}
	}
	if !unknown || !clock || !printed {
		t.Fatal("missing causal family", unknown, clock, printed)
	}
	binary := os.Getenv("TEST_S118_BINARY")
	if binary == "" {
		t.Fatal("compiled verifier required")
	}
	body, _ := json.Marshal(out)
	command := exec.Command(binary, "validate")
	command.Stdin = bytes.NewReader(body)
	got, err := command.Output()
	if err != nil || bytes.Contains(got, []byte("PRIVATE")) {
		t.Fatal("actual CLI projection", err)
	}
	// Restoring the synthetic date selection, with no modification of a saved E6 artifact,
	// removes the clock failure and proves attribution to the actual scorer.
	r := statementdata.Review{Rows: []statementdata.Row{{OperationDate: &statementdata.LocalDate{Local: "2026-10-19T10:30:00", Precision: "local_datetime"}}}}
	_, checks := ScreeningChecks("cards", r)
	for _, c := range checks {
		if c.Code == "operation_clock" && !c.Passed {
			t.Fatal("clock near miss")
		}
	}
}
