package statementqualification

import (
	"context"
	"github.com/QahNaaRlN/grace/internal/statementdata"
	"github.com/QahNaaRlN/grace/internal/statementextract"
	"strings"
	"testing"
	"time"
)

type screeningFailureReasoner struct{ fakeReasoner }

func (*screeningFailureReasoner) CurrentPolicy(context.Context) (statementextract.Policy, error) {
	return statementextract.ScreeningPolicy(), nil
}
func TestScreeningStopSettleAndPreflight(t *testing.T) {
	d := statementdata.Document{Extractor: statementdata.ExtractorVersion, Pages: []statementdata.Page{{Number: 1, Width: 600, Height: 800}}, Spans: []statementdata.Span{{ID: "s", Page: 1, Block: 1, Line: 1, Word: 1, Text: "-1.00", Box: statementdata.Box{XMin: 1, YMin: 1, XMax: 10, YMax: 10}}}}
	samples := []Sample{}
	for _, n := range ScreeningNames() {
		samples = append(samples, Sample{n, d, FixtureHash(n)})
	}
	for _, mode := range []string{"known", "unknown", "hash", "order", "policy"} {
		t.Run(mode, func(t *testing.T) {
			f := &screeningFailureReasoner{}
			f.billing = &statementextract.Billing{GenerationID: "synthetic", CostMicros: 100, InputTokens: 10, OutputTokens: 10}
			if mode == "unknown" {
				f.billing = nil
			}
			in := append([]Sample{}, samples...)
			if mode == "hash" {
				in[2].SHA256 = "bad"
			}
			if mode == "order" {
				in[0], in[1] = in[1], in[0]
			}
			m := &fakeMeter{}
			var reasoner statementextract.Reasoner = f
			if mode == "policy" {
				reasoner = &fakeReasoner{}
			}
			report, e := RunScreening(context.Background(), in, reasoner, m)
			if e == nil || report.Passed || ValidateReport(report, time.Now()) == nil {
				t.Fatal("invalid screening accepted", report, e)
			}
			if mode == "hash" || mode == "order" || mode == "policy" {
				if m.reserves != 0 || f.calls != 0 {
					t.Fatal("invalid preflight reached inference", m, f.calls)
				}
				return
			}
			if f.calls != 1 || m.reserves != 1 || m.reserved != 161700 || m.settles != 1 {
				t.Fatal("retry/reservation", f.calls, m)
			}
			if mode == "unknown" && m.actual != nil || mode == "known" && (m.actual == nil || *m.actual != 100) {
				t.Fatal("lost billing", m)
			}
		})
	}
}
func screeningReportFixture() Report {
	r := validReportFixture()
	r.Version = ScreeningReportVersion
	r.Profile = ScreeningProfile()
	r.Policy = statementextract.ScreeningPolicy()
	r.MaximumMicros = 161700
	*r.ActualMicros = 150
	name := "turnover-roles-ru"
	r.Corpus = append(r.Corpus, CaseEvidence{name, FixtureHash(name), strings.Repeat("a", 64), strings.Repeat("b", 64), 1000})
	r.Results = append(r.Results, Result{Name: name, Rows: 2, Checks: 23, Correct: 23, Passed: true})
	r.Calls = append(r.Calls, Call{GenerationID: "ru-synthetic", InputTokens: 100, OutputTokens: 100, CostMicros: 50})
	r.Traces = append(r.Traces, Trace{Name: name, Parts: []statementdata.Proposal{{}}})
	r.Digest = reportDigest(r)
	return r
}
func TestScreeningReportNeverActivates(t *testing.T) {
	good := screeningReportFixture()
	if ValidateScreeningReport(good, time.Now()) != nil || ValidateReport(good, time.Now()) == nil {
		t.Fatal("screening/activation boundary")
	}
	for _, mutate := range []func(*Report){func(r *Report) { r.Calls = r.Calls[:2] }, func(r *Report) { r.Policy.Provider = "google-ai-studio" }, func(r *Report) { r.Profile.Scorer = "old" }, func(r *Report) { r.Corpus[2].FixtureHash = "bad" }, func(r *Report) { r.Calls[2].GenerationID = r.Calls[0].GenerationID }, func(r *Report) { r.ActualMicros = nil }, func(r *Report) { r.MaximumMicros++ }} {
		r := screeningReportFixture()
		mutate(&r)
		r.Digest = reportDigest(r)
		if ValidateScreeningReport(r, time.Now()) == nil {
			t.Fatal("tampered screening passed")
		}
	}
}
