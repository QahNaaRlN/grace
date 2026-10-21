package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/QahNaaRlN/grace/internal/statementdata"
	"github.com/QahNaaRlN/grace/internal/statementextract"
	"github.com/QahNaaRlN/grace/internal/statementqualification"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type fake struct {
	c     statementextract.ComparisonCandidate
	mode  string
	calls int
}

func (f *fake) CurrentPolicy(context.Context) (statementextract.Policy, error) {
	return f.c.Policy, nil
}
func (f *fake) ComparisonProof(context.Context) (statementextract.ComparisonIdentityProof, error) {
	return proof(f.c), nil
}
func proof(c statementextract.ComparisonCandidate) statementextract.ComparisonIdentityProof {
	return statementextract.ComparisonIdentityProof{Candidate: c, CatalogSHA256: strings.Repeat("a", 64), EndpointSHA256: strings.Repeat("b", 64), ZDRSHA256: strings.Repeat("c", 64), CatalogURL: "https://openrouter.ai/api/v1/models", EndpointURL: "https://openrouter.ai/api/v1/models/" + c.Canonical + "/endpoints", ZDRURL: "https://openrouter.ai/api/v1/endpoints/zdr", VerifiedAt: time.Now().UTC().Add(-time.Second)}
}
func (f *fake) Complete(_ context.Context, _ statementextract.Policy, c statementextract.Chunk) (statementdata.Proposal, *statementextract.Billing, error) {
	f.calls++
	p := proof(f.c)
	bill := &statementextract.Billing{GenerationID: fmt.Sprintf("PRIVATE-%s-%d", f.c.ID, f.calls), InputTokens: 100, OutputTokens: 10, CostMicros: 10, ComparisonRoute: &statementextract.ComparisonRouteEvidence{Proof: &p, ObservedAt: time.Now().UTC(), SelectedModelClass: "pinned_canonical"}}
	part := statementdata.Proposal{Rows: []statementdata.RowMapping{{}}}
	if f.c.ID == "A" && f.calls == 2 {
		part.Rows[0].OperationDate = statementdata.Selection{"PRIVATE-UNKNOWN"}
	}
	return part, bill, nil
}

type meter struct{}

func (*meter) Reserve(context.Context, int64, statementextract.Policy) (string, error) {
	return "PRIVATE-RESERVATION", nil
}
func (*meter) Settle(context.Context, string, *int64) error            { return nil }
func (*meter) ComparisonAllowed(context.Context, string) (bool, error) { return true, nil }
func testContext() publicContext {
	return publicContext{"columns", statementqualification.FixtureHash("columns"), statementdata.Document{Extractor: statementdata.ExtractorVersion, Pages: []statementdata.Page{{Number: 1, Width: 600, Height: 800}}, Spans: []statementdata.Span{{ID: "p1-b1-l1-w1", Page: 1, Block: 1, Line: 1, Word: 1, Text: "RUB", Box: statementdata.Box{XMin: 1, YMin: 1, XMax: 20, YMax: 10}}}}}
}

func artifact(t *testing.T) ([]byte, []publicContext) {
	t.Helper()
	contexts := []publicContext{}
	samples := []statementqualification.Sample{}
	for _, n := range statementqualification.ScreeningNames() {
		c := testContext()
		c.Name = n
		c.FixtureHash = statementqualification.FixtureHash(n)
		contexts = append(contexts, c)
		samples = append(samples, statementqualification.Sample{Name: n, Document: c.Document, SHA256: c.FixtureHash})
	}
	cs := statementextract.ComparisonCandidates()
	a, b := &fake{c: cs[0]}, &fake{c: cs[1]}
	e, err := statementqualification.RunS114Comparison(context.Background(), samples, [2]statementextract.Reasoner{a, b}, &meter{})
	if err == nil || a.calls != 2 || b.calls != 1 {
		t.Fatal("causal fake")
	}
	var buf bytes.Buffer
	if statementqualification.EncodeS114Comparison(&buf, e) != nil {
		t.Fatal("writer")
	}
	return buf.Bytes(), contexts
}
func TestS115FixedReaderAndPrivacy(t *testing.T) {
	raw, c := artifact(t)
	dir := t.TempDir()
	marker := filepath.Join(dir, basename+".started")
	report := filepath.Join(dir, basename+".json")
	if os.WriteFile(marker, []byte(statementqualification.SavedDiagnosisSource+"\n"), 0600) != nil || os.WriteFile(report, raw, 0600) != nil {
		t.Fatal("fixture")
	}
	before, _ := readPrivate(report, limit)
	out, err := readReport(dir, c)
	if err != nil || !out.ArtifactsUnchanged || !out.ContextVerified || out.Tuples[2].Status != "VERIFIED" {
		t.Fatal("actual reader", err)
	}
	after, _ := readPrivate(report, limit)
	if !unchanged(before, after) {
		t.Fatal("artifact changed")
	}
	encoded, _ := json.Marshal(out)
	if bytes.Contains(encoded, []byte("PRIVATE")) || statementqualification.ValidateSavedDiagnosis(out) != nil {
		t.Fatal("private projection")
	}
	for _, mode := range []string{"mode", "symlink", "source", "missing", "partial", "oversize", "duplicate"} {
		t.Run(mode, func(t *testing.T) {
			os.Remove(report)
			os.WriteFile(report, raw, 0600)
			os.WriteFile(marker, []byte(statementqualification.SavedDiagnosisSource+"\n"), 0600)
			switch mode {
			case "mode":
				os.Chmod(report, 0644)
			case "symlink":
				os.Remove(report)
				os.Symlink(marker, report)
			case "source":
				os.WriteFile(marker, []byte(strings.Repeat("a", 40)+"\n"), 0600)
			case "missing":
				os.Remove(marker)
			case "partial":
				os.WriteFile(report, raw[:len(raw)/2], 0600)
			case "oversize":
				os.WriteFile(report, bytes.Repeat([]byte("x"), limit+1), 0600)
			case "duplicate":
				os.WriteFile(report, []byte(`{"version":"PRIVATE","Version":"PRIVATE"}`), 0600)
			}
			if _, err := readReport(dir, c); err == nil {
				t.Fatal("unsafe artifact")
			}
		})
	}
	os.Remove(report)
	os.WriteFile(report, raw, 0600)
	a, _ := readPrivate(report, limit)
	os.WriteFile(report, append(raw, ' '), 0600)
	b, _ := readPrivate(report, limit)
	if unchanged(a, b) {
		t.Fatal("mutation undetected")
	}
	for _, input := range []string{`{"a":1,"A":2}`, `{"private":"PRIVATE"}`, `null null`, strings.Repeat(" ", limit+1)} {
		var out statementqualification.SavedDiagnosis
		if strict([]byte(input), &out) == nil {
			t.Fatal("strict JSON")
		}
	}
}
func TestS115StrictUTF8(t *testing.T) {
	var out statementqualification.SavedDiagnosis
	if strict([]byte{'"', 0xff, '"'}, &out) == nil {
		t.Fatal("invalid UTF8")
	}
}

func TestS115ClosedSchemaRejectsPrivateFields(t *testing.T) {
	raw, c := artifact(t)
	out, err := project(raw, c)
	if err != nil {
		t.Fatal(err)
	}
	out.ArtifactsUnchanged = true
	good, _ := json.Marshal(out)
	for _, mutate := range []func(*statementqualification.SavedDiagnosis){
		func(d *statementqualification.SavedDiagnosis) { d.Source = "PRIVATE" },
		func(d *statementqualification.SavedDiagnosis) { d.Tuples[2].Relation = "PRIVATE" },
		func(d *statementqualification.SavedDiagnosis) { d.Tuples[0].Reason = "PRIVATE" },
		func(d *statementqualification.SavedDiagnosis) { d.ProviderCalls = 1 },
		func(d *statementqualification.SavedDiagnosis) { d.ArtifactsUnchanged = false },
	} {
		var d statementqualification.SavedDiagnosis
		json.Unmarshal(good, &d)
		mutate(&d)
		if statementqualification.ValidateSavedDiagnosis(d) == nil {
			t.Fatal("open schema")
		}
	}
}
