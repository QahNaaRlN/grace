//go:build integration

package statementqualification

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/QahNaaRlN/grace/internal/pdfextract"
	"github.com/QahNaaRlN/grace/internal/statementextract"
)

type s117ReplayMeter struct{ fakeMeter }

func (m *s117ReplayMeter) Reserve(ctx context.Context, n int64, p statementextract.Policy) (string, error) {
	_, err := m.fakeMeter.Reserve(ctx, n, p)
	return "66666666-7777-4888-8999-000000000000", err
}

func TestS117RealHelperReplay(t *testing.T) {
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
	full, err := RunS117Comparison(context.Background(), samples, [2]statementextract.Reasoner{a, b}, &s117ReplayMeter{})
	if err != nil || !full.Report.Passed || ValidateS117Comparison(full, time.Now()) != nil {
		t.Fatal("S117 positive", err)
	}
	for i, tuple := range full.Report.Tuples {
		if tuple.Result.Correct != []int{19, 20, 23}[i/2] || !ReplayComparisonObservation(&full, i, samples[i/2].Document, time.Now()).Verified {
			t.Fatal("S117 real helper replay", i)
		}
	}
	verifyS117ReplayCLI(t, full, socket)
	// Only an explicit disposable export directory gets private synthetic bytes.
	// The inspector test reads this original writer output without relabelling.
	if dir := os.Getenv("S117_REAL_EXPORT_DIR"); dir != "" {
		var raw bytes.Buffer
		if EncodeS117Comparison(&raw, full) != nil || os.WriteFile(filepath.Join(dir, "observed.json"), raw.Bytes(), 0600) != nil {
			t.Fatal("private fixture export")
		}
	}

	for _, code := range []string{"unknown_alias", "duplicate_alias", "non_numeric_money_selection"} {
		rs, a, b := observedReasoners(code)
		e, err := RunS117Comparison(context.Background(), samples, rs, &fakeMeter{})
		replay := ReplayComparisonObservation(&e, 0, samples[0].Document, time.Now())
		if err == nil || a.calls != 1 || b.calls != 0 || !replay.Verified || replay.Diagnostic.Code != code {
			t.Fatal("S117 guard replay", code)
		}
		verifyS117ReplayCLI(t, e, socket)
	}
}

func verifyS117ReplayCLI(t *testing.T, e ObservedComparisonReport, socket string) {
	t.Helper()
	binary := os.Getenv("TEST_S117_REPLAY_BINARY")
	if binary == "" {
		t.Fatal("actual replay CLI binary required")
	}
	var private bytes.Buffer
	if err := EncodeS117Comparison(&private, e); err != nil {
		t.Fatal(err)
	}
	command := exec.Command(binary, "-", "/src/internal/pdfextract/testdata", socket)
	command.Stdin = &private
	raw, err := command.Output()
	if err != nil {
		t.Fatalf("actual S117 replay CLI failed: %T %v", err, err)
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
