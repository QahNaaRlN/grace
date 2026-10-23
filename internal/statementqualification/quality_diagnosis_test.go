package statementqualification

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/QahNaaRlN/grace/internal/statementdata"
	"github.com/QahNaaRlN/grace/internal/statementextract"
	"reflect"
	"strings"
	"testing"
	"time"
)

const qualityTestVerifier = "1111111111111111111111111111111111111111"

type qualityFake struct{ comparisonFake }

func (f *qualityFake) Complete(ctx context.Context, p statementextract.Policy, c statementextract.Chunk) (statementdata.Proposal, *statementextract.Billing, error) {
	part, bill, err := f.comparisonFake.Complete(ctx, p, c)
	part.Rows = []statementdata.RowMapping{{}}
	return part, bill, err
}
func qualityTestReport(t *testing.T) (ObservedComparisonReport, []Sample) {
	t.Helper()
	samples := observedSamples()
	cs := statementextract.ComparisonCandidates()
	a := &qualityFake{comparisonFake{candidate: cs[0]}}
	b := &qualityFake{comparisonFake{candidate: cs[1]}}
	e, err := RunS117Comparison(context.Background(), samples, [2]statementextract.Reasoner{a, b}, &fakeMeter{})
	if err == nil || e.Report.Error != "qualification_quality_failed" {
		t.Fatal("actual causal report", err)
	}
	return e, samples
}
func TestQualityReplayAndIndependentFailures(t *testing.T) {
	e, samples := qualityTestReport(t)
	before := statementextract.Hash(e)
	var wire bytes.Buffer
	if EncodeS117Comparison(&wire, e) != nil {
		t.Fatal("writer")
	}
	_, parsed, err := DecodeS117ComparisonArtifact(&wire)
	if err != nil {
		t.Fatal(err)
	}
	out, err := DiagnoseSavedQuality(*parsed, samples, qualityTestVerifier, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	out.ArtifactsUnchanged = true
	if ValidateQualityDiagnosis(out) != nil {
		raw, _ := json.Marshal(out)
		t.Fatal("closed schema", string(raw))
	}
	if before != statementextract.Hash(e) {
		t.Fatal("diagnosis mutated input")
	}
	for _, r := range out.Tuples {
		if r.Status != "VERIFIED" {
			t.Fatal("replay", r)
		}
	}
	for _, mode := range []string{"snapshot", "replay", "result", "context", "verifier", "legacy"} {
		t.Run(mode, func(t *testing.T) {
			var buf bytes.Buffer
			EncodeS117Comparison(&buf, e)
			_, v, _ := DecodeS117ComparisonArtifact(&buf)
			switch mode {
			case "snapshot":
				v.Observations[0].Snapshot = nil
				v.Observations[0].CaptureStatus = "unavailable_limit"
				v.Digest = observedDigest(*v)
			case "replay":
				v.Observations[0].Snapshot.Rows[0].Description = statementdata.Selection{"PRIVATE"}
				v.Digest = observedDigest(*v)
			case "result":
				v.Report.Tuples[0].Result.Correct++
				v.Report.Digest = comparisonDigest(v.Report)
				v.Digest = observedDigest(*v)
			case "context":
				v.Report.Tuples[0].RequestHash = strings.Repeat("a", 64)
				v.Report.Digest = comparisonDigest(v.Report)
				v.Digest = observedDigest(*v)
			case "legacy":
				v.Version = S114ObservedComparisonVersion
				v.Digest = observedDigest(*v)
			}
			source := qualityTestVerifier
			if mode == "verifier" {
				source = "PRIVATE"
			}
			got, err := DiagnoseSavedQuality(*v, samples, source, time.Now())
			if mode == "context" || mode == "legacy" || mode == "verifier" {
				if err == nil || len(got.Tuples) > 0 {
					t.Fatal("context opened")
				}
				return
			}
			if err != nil || got.Tuples[0].Status != "UNVERIFIED" || got.Tuples[1].Status != "VERIFIED" {
				t.Fatal("tuple isolation", err, got)
			}
			raw, _ := json.Marshal(got)
			if bytes.Contains(raw, []byte("PRIVATE")) {
				t.Fatal("private error")
			}
		})
	}
}
func TestQualityReasonVocabularyAndCounts(t *testing.T) {
	for _, code := range strings.Fields(qualityReasonVocabulary) {
		raw := code
		if code == "unmapped_spans" {
			raw += ":1"
		}
		_, _, _, _, ok := qualityCode(raw, 2)
		if !ok {
			t.Fatal(code)
		}
	}
	for _, raw := range []string{"row_0:overlapping_evidence", "row_3:overlapping_evidence", "row_01:overlapping_evidence", "row_1:PRIVATE", "service:PRIVATE", "PRIVATE:invalid_date", "unmapped_spans:01", "unmapped_spans:2001", "row_1:operationDate:invalid_date"} {
		_, _, _, _, ok := qualityCode(raw, 2)
		if ok {
			t.Fatal("open reason", raw)
		}
	}
	r := statementdata.Review{Rows: []statementdata.Row{{Issues: []string{"overlapping_evidence", "overlapping_evidence"}}}}
	r.Reconciliation.Blockers = []string{"row_1:overlapping_evidence", "row_1:overlapping_evidence"}
	r.Reconciliation.RowCoverage.Reasons = []string{"unmapped_spans:1"}
	reasons := qualityReasons(r, statementdata.Proposal{}, qualityPositions{}, 1)
	if len(reasons) != 5 || reasons[0].Relation != "not_exposed" || reasons[2].PublicCode != "unknown" || reasons[3].Ordinal != 2 || *reasons[4].Count != 1 {
		t.Fatal("ordering/mapping", reasons)
	}
	mismatch := qualityReasons(r, statementdata.Proposal{}, qualityPositions{}, 2)
	if mismatch[4].Status != "UNVERIFIED" || mismatch[4].Reason != "replay_mismatch" {
		t.Fatal("guessed count")
	}
	for _, raw := range []string{"row_2:description_evidence_gap", "openingBookBalance:book_balance_role_unverified", "invalid_date", "unmapped_spans:4"} {
		if got := qualityPublicCode(raw); got != ScreeningPublicCodes([]string{raw})[0] {
			t.Fatal("sanitizer drift")
		}
	}
}
func TestQualityClockNearMissAndTrace(t *testing.T) {
	cases := []struct {
		local, precision string
		want             QualityParts
	}{{"2026-10-19", "date", QualityParts{"matches", "differs", "differs", "matches"}}, {"2026-10-19T10:30:00", "local_datetime", QualityParts{"matches", "matches", "matches", "matches"}}, {"2026-10-20T11:00:00", "local_datetime", QualityParts{"differs", "differs", "matches", "matches"}}}
	for _, c := range cases {
		got := clockParts(&statementdata.LocalDate{Local: c.local, Precision: c.precision}, "cards", "operation_clock", 1)
		if *got != c.want {
			t.Fatal(got, c.want)
		}
	}
	if clockParts(nil, "cards", "booking_clock", 1).Time != "missing" {
		t.Fatal("missing clock fabricated")
	}
	for _, name := range ScreeningNames() {
		r := statementdata.Review{Rows: make([]statementdata.Row, 2)}
		got, checks := ScreeningChecks(name, r)
		if !reflect.DeepEqual(got, ScoreScreening(name, r)) || len(checks) != got.Checks {
			t.Fatal("trace changed scorer")
		}
		for _, c := range checks {
			if c.Code == "blockers" && c.RowOrdinal != 0 {
				t.Fatal("header attributed to row")
			}
		}
	}
}
func TestQualityWitnessLimitsAndSchemaPrivacy(t *testing.T) {
	lookup := qualityPositions{}
	ids := statementdata.Selection{}
	for n := 1; n <= 129; n++ {
		id := strings.Repeat("x", n)
		lookup[id] = QualityPosition{n, 1, 1, 1, n, "other"}
		ids = append(ids, id)
	}
	if ps, ok := lookup.selected(ids[:128]); !ok || len(ps) != 128 {
		t.Fatal("128")
	}
	if _, ok := lookup.selected(ids); ok {
		t.Fatal("129")
	}
	e, s := qualityTestReport(t)
	d, err := DiagnoseSavedQuality(e, s, qualityTestVerifier, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	d.ArtifactsUnchanged = true
	for _, mode := range []string{"private", "clock", "position", "source", "writes", "reason"} {
		t.Run(mode, func(t *testing.T) {
			raw, _ := json.Marshal(d)
			var v QualityDiagnosis
			json.Unmarshal(raw, &v)
			switch mode {
			case "private":
				v.Tuples[0].Checks[0].Code = "PRIVATE"
			case "clock":
				v.Tuples[0].Checks[0].Clock = &QualityParts{Date: "PRIVATE"}
			case "position":
				v.Tuples[0].Checks[0].Positions = []QualityPosition{{1, 1, 1, 1, 1, "PRIVATE"}}
			case "source":
				v.VerifierSource = "PRIVATE"
			case "writes":
				v.PersistentWritesRequested = 1
			case "reason":
				v.Tuples[0].Reasons[0].Code = "PRIVATE"
			}
			if ValidateQualityDiagnosis(v) == nil {
				t.Fatal("open projection")
			}
		})
	}
}

func TestQualityActualUnmappedBoundaryAndOutputCap(t *testing.T) {
	for _, n := range []int{128, 129} {
		d := observedSamples()[0].Document
		d.Spans = nil
		for i := 1; i <= n; i++ {
			d.Spans = append(d.Spans, statementdata.Span{ID: fmt.Sprintf("p1-b1-l1-w%d", i), Page: 1, Block: 1, Line: 1, Word: i, Text: "RUB", Box: statementdata.Box{XMin: float64(i), YMin: 1, XMax: float64(i + 1), YMax: 10}})
		}
		p := statementdata.Proposal{Rows: []statementdata.RowMapping{{}}}
		review, err := statementdata.ValidateProposal(d, p, []int{1}, true, "")
		if err != nil {
			t.Fatal(err)
		}
		plan, err := statementextract.BuildComparisonPlan(d, statementextract.ComparisonCandidates()[0].Policy)
		if err != nil {
			t.Fatal(err)
		}
		c := qualityCoverage(d, p, review, qualityLookup(d, plan))
		if c.TotalUnmapped != n || n == 128 && (c.Status != "VERIFIED" || len(c.Positions) != 128) || n == 129 && (c.Status != "UNVERIFIED" || len(c.Positions) != 0) {
			t.Fatal("actual boundary", c)
		}
	}
	e, s := qualityTestReport(t)
	out, err := DiagnoseSavedQuality(e, s, qualityTestVerifier, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	out.ArtifactsUnchanged = true
	ps := []QualityPosition{}
	for i := 1; i <= 128; i++ {
		ps = append(ps, QualityPosition{i, 1, 1, 1, i, "other"})
	}
	for i := range out.Tuples {
		for j := range out.Tuples[i].Checks {
			if out.Tuples[i].Checks[j].Status == "VERIFIED" {
				out.Tuples[i].Checks[j].Positions = ps
			}
		}
	}
	raw, _ := json.Marshal(out)
	if len(raw) < 65536 {
		t.Fatal("output cap fixture too small")
	}
	if ValidateQualityDiagnosis(out) == nil {
		t.Fatal("oversized projection accepted")
	}
}
