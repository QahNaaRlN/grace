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

type savedFixtureFake struct {
	comparisonFixtureReasoner
	item string
}

func (f *savedFixtureFake) Complete(ctx context.Context, p statementextract.Policy, c statementextract.Chunk) (statementdata.Proposal, *statementextract.Billing, error) {
	part, bill, err := f.comparisonFixtureReasoner.Complete(ctx, p, c)
	if f.candidate.ID == "A" && f.calls == 2 {
		part.Rows[0].OperationDate = statementdata.Selection{f.item}
	}
	return part, bill, err
}
func TestS115RealHelperDiagnosis(t *testing.T) {
	socket := os.Getenv("TEST_PDF_EXTRACTOR_SOCKET")
	if socket == "" {
		t.Skip("pinned helper required")
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
	for _, item := range []string{samples[1].Document.Spans[0].ID, "szzzz", "PRIVATE-UNKNOWN"} {
		cs := statementextract.ComparisonCandidates()
		a := &savedFixtureFake{comparisonFixtureReasoner: comparisonFixtureReasoner{candidate: cs[0]}, item: item}
		a.omitSeparator = true
		b := &savedFixtureFake{comparisonFixtureReasoner: comparisonFixtureReasoner{candidate: cs[1]}}
		b.omit = true
		e, err := RunS114Comparison(context.Background(), samples, [2]statementextract.Reasoner{a, b}, &fakeMeter{})
		if err == nil || e.Report.Tuples[0].Result.Correct != 16 || e.Report.Tuples[1].Result.Correct != 17 {
			t.Fatal("causal P6-shaped fixture", err, e.Report.Tuples[0].Result, e.Report.Tuples[1].Result)
		}
		var private bytes.Buffer
		if EncodeS114Comparison(&private, e) != nil {
			t.Fatal("writer")
		}
		_, decoded, err := DecodeS114ComparisonArtifact(&private)
		if err != nil {
			t.Fatal(err)
		}
		out, err := DiagnoseSavedProposals(*decoded, samples, time.Now())
		if err != nil {
			t.Fatal(err)
		}
		for i := 0; i < 3; i++ {
			if out.Tuples[i].Status != "VERIFIED" {
				t.Fatalf("issue %d: %+v", i, out.Tuples[i])
			}
		}
		gap := out.Tuples[0].Gap
		if gap.IntermediateUnselected != 1 || gap.RowOrdinal != 2 || *out.Tuples[1].TotalUnmapped != 1 {
			t.Fatal("causal witnesses", out)
		}
		// Restore only a synthetic omitted separator and prove the production blocker disappears.
		plan, _ := statementextract.BuildComparisonPlan(samples[0].Document, cs[0].Policy)
		proposal, _ := statementextract.MergeProposals(plan, []statementdata.Proposal{*e.Report.Tuples[0].Proposal})
		left, right := gap.Left.SourceOrdinal, gap.Right.SourceOrdinal
		ids := statementdata.Selection{}
		for _, s := range samples[0].Document.Spans[left-1 : right] {
			ids = append(ids, s.ID)
		}
		proposal.Rows[1].Description = ids
		review, err := statementdata.ValidateProposal(samples[0].Document, proposal, []int{1}, true, "")
		if err != nil {
			t.Fatal(err)
		}
		if _, _, ok := statementdata.DescriptionGapPositions(samples[0].Document, review.Rows[1], 1); ok {
			t.Fatal("causal restoration did not remove guard")
		}
		out.ArtifactsUnchanged = true
		if ValidateSavedDiagnosis(out) != nil {
			t.Fatal("projection schema")
		}
		binary := os.Getenv("TEST_S115_BINARY")
		if binary == "" {
			t.Fatal("actual verifier required")
		}
		raw, _ := json.Marshal(out)
		command := exec.Command(binary, "validate")
		command.Stdin = bytes.NewReader(raw)
		projected, err := command.Output()
		if err != nil || bytes.Contains(projected, []byte("PRIVATE")) {
			t.Fatal("verifier projection/privacy", err)
		}
	}
}
