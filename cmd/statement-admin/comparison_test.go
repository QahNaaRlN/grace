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

type summaryComparisonFake struct {
	candidate statementextract.ComparisonCandidate
	calls     int
}

func (f *summaryComparisonFake) CurrentPolicy(context.Context) (statementextract.Policy, error) {
	return f.candidate.Policy, nil
}
func (f *summaryComparisonFake) ComparisonProof(context.Context) (statementextract.ComparisonIdentityProof, error) {
	return statementextract.ComparisonIdentityProof{Candidate: f.candidate, CatalogSHA256: strings.Repeat("a", 64), EndpointSHA256: strings.Repeat("b", 64), ZDRSHA256: strings.Repeat("c", 64), CatalogURL: "https://openrouter.ai/api/v1/models", EndpointURL: "https://openrouter.ai/api/v1/models/" + f.candidate.Canonical + "/endpoints", ZDRURL: "https://openrouter.ai/api/v1/endpoints/zdr", VerifiedAt: time.Now().Add(-time.Second)}, nil
}
func (f *summaryComparisonFake) Complete(ctx context.Context, p statementextract.Policy, c statementextract.Chunk) (statementdata.Proposal, *statementextract.Billing, error) {
	f.calls++
	proof, _ := f.ComparisonProof(ctx)
	return statementdata.Proposal{}, &statementextract.Billing{ComparisonRoute: &statementextract.ComparisonRouteEvidence{Proof: &proof, ObservedAt: time.Now().UTC(), SelectedModelClass: "alias"}, GenerationID: fmt.Sprintf("PRIVATE-generation-%s-%d", f.candidate.ID, f.calls), CostMicros: 42, InputTokens: 100, OutputTokens: 10}, nil
}

type summaryComparisonMeter struct{}

func (summaryComparisonMeter) Reserve(context.Context, int64, statementextract.Policy) (string, error) {
	return "PRIVATE-reservation", nil
}
func (summaryComparisonMeter) Settle(context.Context, string, *int64) error { return nil }
func (summaryComparisonMeter) ComparisonAllowed(context.Context, string) (bool, error) {
	return true, nil
}
func TestComparisonSummaryUsesOnlyClosedFields(t *testing.T) {
	d := statementdata.Document{Extractor: statementdata.ExtractorVersion, Pages: []statementdata.Page{{Number: 1, Width: 600, Height: 800}}, Spans: []statementdata.Span{{ID: "source", Page: 1, Block: 1, Line: 1, Word: 1, Text: "RUB", Box: statementdata.Box{XMin: 1, YMin: 1, XMax: 20, YMax: 10}}}}
	samples := []statementqualification.Sample{}
	for _, n := range statementqualification.ScreeningNames() {
		samples = append(samples, statementqualification.Sample{Name: n, Document: d, SHA256: statementqualification.FixtureHash(n)})
	}
	cs := statementextract.ComparisonCandidates()
	rs := [2]statementextract.Reasoner{&summaryComparisonFake{candidate: cs[0]}, &summaryComparisonFake{candidate: cs[1]}}
	report, _ := statementqualification.RunComparison(context.Background(), samples, rs, summaryComparisonMeter{})
	if statementqualification.ValidateComparisonReport(report, time.Now()) != nil {
		t.Fatal("need genuine engine failure report")
	}
	for i := range report.Tuples {
		if report.Tuples[i].Result != nil {
			report.Tuples[i].Result.Codes = append(report.Tuples[i].Result.Codes, "PRIVATE-body")
			report.Tuples[i].Result.Validation = append(report.Tuples[i].Result.Validation, "PRIVATE-error")
			report.Tuples[i].Result.Coverage = append(report.Tuples[i].Result.Coverage, "PRIVATE-source")
		}
	}
	report.Digest = ""
	report.Digest = statementextract.Hash(report)
	out := comparisonSummary(report)
	if out["reportShapeVerified"] != true {
		t.Fatal("unexpected invalid report", out)
	}
	raw, _ := json.Marshal(out)
	text := string(raw)
	for _, secret := range []string{"PRIVATE", "generationId", "costMicros", "actualMicros", "reservationId", "catalogSha256"} {
		if strings.Contains(text, secret) {
			t.Fatal("summary leak", text)
		}
	}
	if out["activationAuthorized"] != false || out["comparisonPassed"] != false {
		t.Fatal("false admission")
	}
	report.Error = "PRIVATE-error"
	raw, _ = json.Marshal(comparisonSummary(report))
	if strings.Contains(string(raw), "PRIVATE") {
		t.Fatal("invalid report leak")
	}
}

type observedSummaryFake struct{ summaryComparisonFake }

func (f *observedSummaryFake) Complete(ctx context.Context, p statementextract.Policy, c statementextract.Chunk) (statementdata.Proposal, *statementextract.Billing, error) {
	_, bill, err := f.summaryComparisonFake.Complete(ctx, p, c)
	return statementdata.Proposal{Rows: []statementdata.RowMapping{{Description: statementdata.Selection{"PRIVATE-ALIAS"}}}, Service: []statementdata.ServiceMapping{{Reason: "PRIVATE-SERVICE"}}}, bill, err
}

type observedSummaryMeter struct{ summaryComparisonMeter }

func (observedSummaryMeter) Reserve(context.Context, int64, statementextract.Policy) (string, error) {
	return "11111111-2222-4333-8444-555555555555", nil
}
func TestObservedComparisonSummaryPrivatePipeline(t *testing.T) {
	d := statementdata.Document{Extractor: statementdata.ExtractorVersion, Pages: []statementdata.Page{{Number: 1, Width: 600, Height: 800}}, Spans: []statementdata.Span{{ID: "public", Page: 1, Block: 1, Line: 1, Word: 1, Text: "RUB", Box: statementdata.Box{XMin: 1, YMin: 1, XMax: 20, YMax: 10}}}}
	samples := []statementqualification.Sample{}
	for _, n := range statementqualification.ScreeningNames() {
		samples = append(samples, statementqualification.Sample{Name: n, Document: d, SHA256: statementqualification.FixtureHash(n)})
	}
	cs := statementextract.ComparisonCandidates()
	rs := [2]statementextract.Reasoner{&observedSummaryFake{summaryComparisonFake{candidate: cs[0]}}, &observedSummaryFake{summaryComparisonFake{candidate: cs[1]}}}
	report, err := statementqualification.RunObservedComparison(context.Background(), samples, rs, observedSummaryMeter{})
	if err == nil {
		t.Fatal("causal protocol failure required")
	}
	var artifact bytes.Buffer
	if statementqualification.EncodeObservedComparison(&artifact, report) != nil {
		t.Fatal("private artifact writer failed")
	}
	r, e, err := statementqualification.DecodeComparisonArtifact(bytes.NewReader(artifact.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	out := comparisonArtifactSummary(r, e)
	if out["observationsVerified"] != true || out["reportShapeVerified"] != true || out["observationUnavailableLegacy"] != false {
		t.Fatal("summary didn't validate envelope")
	}
	public, _ := json.Marshal(out)
	for _, s := range []string{"PRIVATE", "snapshot", "costMicros", "actualMicros", "generationId", "reservationId"} {
		if bytes.Contains(public, []byte(s)) {
			t.Fatal("summary privacy failure")
		}
	}
	if dir := os.Getenv("S113_TEST_EXPORT_DIR"); dir != "" {
		// Offline test fixture only, used to exercise the actual Go CLI from Python.
		if os.WriteFile(filepath.Join(dir, "observed.json"), artifact.Bytes(), 0600) != nil || os.WriteFile(filepath.Join(dir, "summary.json"), public, 0600) != nil {
			t.Fatal("offline fixture export")
		}
	}
}

func TestS114SummaryPrivatePipeline(t *testing.T) {
	d := statementdata.Document{Extractor: statementdata.ExtractorVersion, Pages: []statementdata.Page{{Number: 1, Width: 600, Height: 800}}, Spans: []statementdata.Span{{ID: "public", Page: 1, Block: 1, Line: 1, Word: 1, Text: "RUB", Box: statementdata.Box{XMin: 1, YMin: 1, XMax: 20, YMax: 10}}}}
	samples := []statementqualification.Sample{}
	for _, n := range statementqualification.ScreeningNames() {
		samples = append(samples, statementqualification.Sample{Name: n, Document: d, SHA256: statementqualification.FixtureHash(n)})
	}
	cs := statementextract.ComparisonCandidates()
	rs := [2]statementextract.Reasoner{&observedSummaryFake{summaryComparisonFake{candidate: cs[0]}}, &observedSummaryFake{summaryComparisonFake{candidate: cs[1]}}}
	e, err := statementqualification.RunS114Comparison(context.Background(), samples, rs, observedSummaryMeter{})
	if err == nil {
		t.Fatal("causal failure expected")
	}
	var raw bytes.Buffer
	if statementqualification.EncodeS114Comparison(&raw, e) != nil {
		t.Fatal("writer")
	}
	out := s114ComparisonSummary(e)
	if out["reportShapeVerified"] != true || out["observationsVerified"] != true {
		t.Fatal("summary")
	}
	public, _ := json.Marshal(out)
	for _, secret := range []string{"PRIVATE", "snapshot", "generationId", "reservationId", "actualMicros"} {
		if bytes.Contains(public, []byte(secret)) {
			t.Fatal("private leak")
		}
	}
	if dir := os.Getenv("S114_TEST_EXPORT_DIR"); dir != "" {
		if os.WriteFile(filepath.Join(dir, "observed.json"), raw.Bytes(), 0600) != nil || os.WriteFile(filepath.Join(dir, "summary.json"), public, 0600) != nil {
			t.Fatal("export")
		}
	}
}

func TestS117SummaryPrivatePipeline(t *testing.T) {
	d := statementdata.Document{Extractor: statementdata.ExtractorVersion, Pages: []statementdata.Page{{Number: 1, Width: 600, Height: 800}}, Spans: []statementdata.Span{{ID: "public", Page: 1, Block: 1, Line: 1, Word: 1, Text: "RUB", Box: statementdata.Box{XMin: 1, YMin: 1, XMax: 20, YMax: 10}}}}
	samples := []statementqualification.Sample{}
	for _, n := range statementqualification.ScreeningNames() {
		samples = append(samples, statementqualification.Sample{Name: n, Document: d, SHA256: statementqualification.FixtureHash(n)})
	}
	cs := statementextract.ComparisonCandidates()
	rs := [2]statementextract.Reasoner{&observedSummaryFake{summaryComparisonFake{candidate: cs[0]}}, &observedSummaryFake{summaryComparisonFake{candidate: cs[1]}}}
	e, err := statementqualification.RunS117Comparison(context.Background(), samples, rs, observedSummaryMeter{})
	if err == nil {
		t.Fatal("causal failure expected")
	}
	var raw bytes.Buffer
	if statementqualification.EncodeS117Comparison(&raw, e) != nil {
		t.Fatal("writer")
	}
	out := s117ComparisonSummary(e)
	if out["reportShapeVerified"] != true || out["observationsVerified"] != true {
		t.Fatal("summary")
	}
	public, _ := json.Marshal(out)
	for _, secret := range []string{"PRIVATE", "snapshot", "generationId", "reservationId", "actualMicros"} {
		if bytes.Contains(public, []byte(secret)) {
			t.Fatal("private leak")
		}
	}
	if dir := os.Getenv("S117_TEST_EXPORT_DIR"); dir != "" {
		if os.WriteFile(filepath.Join(dir, "observed.json"), raw.Bytes(), 0600) != nil || os.WriteFile(filepath.Join(dir, "summary.json"), public, 0600) != nil {
			t.Fatal("export")
		}
	}
}
