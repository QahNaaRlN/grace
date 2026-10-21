//go:build integration

package statementqualification

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/QahNaaRlN/grace/internal/pdfextract"
	"github.com/QahNaaRlN/grace/internal/statementdata"
	"github.com/QahNaaRlN/grace/internal/statementextract"
)

func TestObservedRealHelperReplay(t *testing.T) {
	socket := os.Getenv("TEST_PDF_EXTRACTOR_SOCKET")
	if socket == "" {
		t.Skip("isolated pinned helper required")
	}
	helper, e := pdfextract.NewClient(socket)
	if e != nil {
		t.Fatal(e)
	}
	samples := []Sample{}
	for _, name := range ScreeningNames() {
		pdf, e := os.ReadFile("../pdfextract/testdata/" + name + ".pdf")
		if e != nil {
			t.Fatal(e)
		}
		d, e := helper.Extract(context.Background(), pdf)
		if e != nil {
			t.Fatal(e)
		}
		samples = append(samples, Sample{name, d, FixtureHash(name)})
	}
	for _, code := range []string{"unknown_alias", "duplicate_alias", "non_numeric_money_selection", "invalid_plan"} {
		t.Run(code, func(t *testing.T) {
			rs, a, b := observedReasoners(code)
			m := &fakeMeter{}
			e, err := RunObservedComparison(context.Background(), samples, rs, m)
			if err == nil || a.calls != 1 || b.calls != 0 || ValidateObservedComparison(e, time.Now()) != nil {
				t.Fatal("causal provider/observer failure", err)
			}
			replay := ReplayComparisonObservation(&e, 0, samples[0].Document, time.Now())
			if code == "invalid_plan" {
				if e.Observations[0].Diagnostic.Code != "invalid_plan" || replay.Verified || replay.Status != "UNVERIFIED" {
					t.Fatal("mutated transient plan must not be fabricated during replay")
				}
				return
			}
			if !replay.Verified || replay.Diagnostic.Code != code {
				t.Fatal("pinned helper replay mismatch", replay)
			}
		})
	}
	// Invalid plan is reproducible as a guard, but a report hash mismatch is never
	// repaired into a live diagnosis. The four guards use the same real document.
	plan, e := statementextract.BuildComparisonPlan(samples[0].Document, statementextract.ComparisonCandidates()[0].Policy)
	if e != nil {
		t.Fatal(e)
	}
	plan.PlannerVersion = "old"
	_, diag, err := statementextract.CanonicalComparisonProposalDiagnostic(plan, statementdata.Proposal{})
	if err == nil || diag.Code != "invalid_plan" || !diag.Valid() {
		t.Fatal("invalid plan guard")
	}
	cs := statementextract.ComparisonCandidates()
	a, b := &comparisonFixtureReasoner{candidate: cs[0]}, &comparisonFixtureReasoner{candidate: cs[1]}
	full, err := RunObservedComparison(context.Background(), samples, [2]statementextract.Reasoner{a, b}, &fakeMeter{})
	if err != nil || !full.Report.Passed || ValidateObservedComparison(full, time.Now()) != nil {
		t.Fatal("real-helper positive changed", err)
	}
	for i, tuple := range full.Report.Tuples {
		replay := ReplayComparisonObservation(&full, i, samples[i/2].Document, time.Now())
		if !replay.Verified || replay.Diagnostic != nil || tuple.Result.Correct != []int{19, 20, 23}[i/2] {
			t.Fatal("positive canonical replay drift", i, replay)
		}
	}
	mismatched := samples[0].Document
	mismatched.Extractor = "old"
	if ReplayComparisonObservation(&full, 0, mismatched, time.Now()).Verified {
		t.Fatal("hash mismatch repaired")
	}
}

func TestS114RealHelperReplay(t *testing.T) {
	socket := os.Getenv("TEST_PDF_EXTRACTOR_SOCKET")
	if socket == "" {
		t.Skip("isolated helper required")
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
		doc, err := helper.Extract(context.Background(), raw)
		if err != nil {
			t.Fatal(err)
		}
		samples = append(samples, Sample{name, doc, FixtureHash(name)})
	}
	cs := statementextract.ComparisonCandidates()
	a, b := &comparisonFixtureReasoner{candidate: cs[0]}, &comparisonFixtureReasoner{candidate: cs[1]}
	full, err := RunS114Comparison(context.Background(), samples, [2]statementextract.Reasoner{a, b}, &fakeMeter{})
	if err != nil || !full.Report.Passed || ValidateS114Comparison(full, time.Now()) != nil {
		t.Fatal("S114 positive", err)
	}
	for i, tuple := range full.Report.Tuples {
		if tuple.Result.Correct != []int{19, 20, 23}[i/2] || !ReplayComparisonObservation(&full, i, samples[i/2].Document, time.Now()).Verified {
			t.Fatal("S114 real helper replay", i)
		}
	}
	verifyS114ReplayCLI(t, full, socket)
	for _, code := range []string{"unknown_alias", "duplicate_alias", "non_numeric_money_selection"} {
		rs, a, b := observedReasoners(code)
		e, err := RunS114Comparison(context.Background(), samples, rs, &fakeMeter{})
		replay := ReplayComparisonObservation(&e, 0, samples[0].Document, time.Now())
		if err == nil || a.calls != 1 || b.calls != 0 || !replay.Verified || replay.Diagnostic.Code != code {
			t.Fatal("S114 guard replay", code)
		}
		verifyS114ReplayCLI(t, e, socket)
	}
}

func verifyS114ReplayCLI(t *testing.T, e ObservedComparisonReport, socket string) {
	t.Helper()
	binary := os.Getenv("TEST_S114_REPLAY_BINARY")
	if binary == "" {
		t.Fatal("actual replay CLI binary required")
	}
	var private bytes.Buffer
	if err := EncodeS114Comparison(&private, e); err != nil {
		t.Fatal(err)
	}
	command := exec.Command(binary, "-", "/src/internal/pdfextract/testdata", socket)
	command.Stdin = &private
	raw, err := command.Output()
	if err != nil {
		t.Fatalf("actual S114 replay CLI failed: %T %v", err, err)
	}
	var output struct {
		ProviderCalls int                `json:"providerCalls"`
		Results       []ComparisonReplay `json:"results"`
	}
	if json.Unmarshal(raw, &output) != nil || output.ProviderCalls != 0 || len(output.Results) != 6 {
		t.Fatal("CLI shape")
	}
	for i, result := range output.Results {
		captured := e.Observations[i].Snapshot != nil
		if result.Verified != captured || (captured && result.Status != "matched") || (!captured && result.Status != "unavailable_capture") {
			t.Fatal("CLI did not replay captured tuples", i)
		}
	}
	if bytes.Contains(raw, []byte("PRIVATE")) || bytes.Contains(raw, []byte("snapshot")) {
		t.Fatal("CLI privacy")
	}
}
