package statementqualification

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/QahNaaRlN/grace/internal/statementdata"
	"github.com/QahNaaRlN/grace/internal/statementextract"
)

func validReportFixture() Report {
	now := time.Now().UTC()
	actual := int64(100)
	policy, _ := (&fakeReasoner{}).CurrentPolicy(context.Background())
	r := Report{Version: ReportVersion, Time: now, Policy: policy, Profile: CurrentProfile(), MaximumMicros: 140000, ActualMicros: &actual, Passed: true}
	for i, name := range []string{"columns", "cards"} {
		r.Corpus = append(r.Corpus, CaseEvidence{name, FixtureHash(name), strings.Repeat("a", 64), strings.Repeat("b", 64), 1000})
		checks := 19 + i
		r.Results = append(r.Results, Result{Name: name, Rows: 2, Checks: checks, Correct: checks, Passed: true})
		r.Calls = append(r.Calls, Call{GenerationID: name + "-synthetic", InputTokens: 25000, OutputTokens: 3000, CostMicros: 50})
		r.Traces = append(r.Traces, Trace{Name: name, Parts: []statementdata.Proposal{{}}})
	}
	r.Digest = reportDigest(r)
	return r
}
func TestActivationReportRejectsLegacyIncompleteChangedEvidence(t *testing.T) {
	good := validReportFixture()
	if e := ValidateReport(good, time.Now()); e != nil {
		t.Fatal(e)
	}
	cases := map[string]func(*Report){
		"old_version":          func(r *Report) { r.Version = "statement.synthetic.v1" },
		"scorer_version":       func(r *Report) { r.Profile.Scorer = "old" },
		"profile":              func(r *Report) { r.Profile.Output = 2000 },
		"fixture":              func(r *Report) { r.Corpus[1].FixtureHash = r.Corpus[0].FixtureHash },
		"plan_hash":            func(r *Report) { r.Corpus[0].PlanHash = "" },
		"missing_case":         func(r *Report) { r.Results = r.Results[:1] },
		"partial_checks":       func(r *Report) { r.Results[0].Checks-- },
		"quality":              func(r *Report) { r.Results[1].Passed = false },
		"coverage":             func(r *Report) { r.Results[1].Coverage = []string{"unmapped_spans:1"} },
		"extra_call":           func(r *Report) { r.Calls = append(r.Calls, r.Calls[0]) },
		"duplicate_generation": func(r *Report) { r.Calls[1].GenerationID = r.Calls[0].GenerationID },
		"wrong_max":            func(r *Report) { r.MaximumMicros = 32000 },
		"token_bound":          func(r *Report) { r.Calls[0].InputTokens = 100001 },
		"cost_bound":           func(r *Report) { r.Calls[0].CostMicros = 70001 },
		"unknown_bill":         func(r *Report) { r.ActualMicros = nil },
		"old_time":             func(r *Report) { r.Time = time.Now().Add(-16 * time.Minute) },
		"future_time":          func(r *Report) { r.Time = time.Now().Add(time.Minute) },
		"wrong_actual":         func(r *Report) { *r.ActualMicros = 1 },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			r := validReportFixture()
			mutate(&r)
			r.Digest = reportDigest(r)
			if ValidateReport(r, time.Now()) == nil {
				t.Fatal("incompatible report accepted")
			}
		})
	}
	good.Traces[0].Parts[0].Service = []statementdata.ServiceMapping{{Reason: "modified report"}}
	if ValidateReport(good, time.Now()) == nil {
		t.Fatal("changed digest accepted")
	}
}
func TestDecodeReportRejectsAmbiguousInput(t *testing.T) {
	raw, _ := json.Marshal(validReportFixture())
	r, e := DecodeReport(bytes.NewReader(raw))
	if e != nil || ValidateReport(r, time.Now()) != nil {
		t.Fatal(e)
	}
	for _, raw := range []string{`{"passed":false,"passed":true}`, `{"passed":false,"Passed":true}`, `{"scorer_version":func(r *Report){r.Profile.Scorer="old"},
 "profile":{"Input":1,"Input":2}}`, `{"unknown":1}`, `{} {}`, strings.Repeat(" ", 524289)} {
		if _, e := DecodeReport(strings.NewReader(raw)); e == nil {
			t.Fatal("ambiguous report accepted")
		}
	}
}

type boundedReasoner struct {
	fakeReasoner
	delayBound time.Duration
	cancel     context.CancelFunc
	invalid    bool
}

func (f *boundedReasoner) Complete(ctx context.Context, _ statementextract.Policy, _ statementextract.Chunk) (statementdata.Proposal, *statementextract.Billing, error) {
	f.calls++
	deadline, ok := ctx.Deadline()
	if !ok {
		return statementdata.Proposal{}, nil, errors.New("missing_deadline")
	}
	f.delayBound = time.Until(deadline)
	if f.cancel != nil {
		f.cancel()
	}
	b := &statementextract.Billing{GenerationID: "synthetic-known", CostMicros: 10, InputTokens: 1000, OutputTokens: 100}
	if f.invalid {
		b.CostMicros = 70001
	}
	return statementdata.Proposal{}, b, errors.New("synthetic_content_failure")
}

type unavailableMeter struct{ fakeMeter }

func (m *unavailableMeter) Reserve(context.Context, int64, statementextract.Policy) (string, error) {
	m.reserves++
	return "", errors.New("budget_unavailable")
}
func minimalSamples() []Sample {
	d := statementdata.Document{Extractor: statementdata.ExtractorVersion, Pages: []statementdata.Page{{Number: 1, Width: 600, Height: 800}}, Spans: []statementdata.Span{{ID: "public", Page: 1, Block: 1, Line: 1, Word: 1, Text: "RUB", Box: statementdata.Box{XMin: 1, YMin: 1, XMax: 20, YMax: 10}}}}
	return []Sample{{"columns", d, FixtureHash("columns")}, {"cards", d, FixtureHash("cards")}}
}
func TestSmokePreflightAndCancellationNeverRetry(t *testing.T) {
	t.Run("budget_before_calls", func(t *testing.T) {
		r, m := &boundedReasoner{}, &unavailableMeter{}
		_, e := Run(context.Background(), minimalSamples(), r, m)
		if e == nil || r.calls != 0 || m.reserves != 1 || m.settles != 0 {
			t.Fatal(e, r, m)
		}
	})
	t.Run("fixture_before_reserve", func(t *testing.T) {
		s := minimalSamples()
		s[1].SHA256 = "bad"
		r, m := &boundedReasoner{}, &fakeMeter{}
		_, e := Run(context.Background(), s, r, m)
		if e == nil || r.calls != 0 || m.reserves != 0 {
			t.Fatal(e, r, m)
		}
	})
	t.Run("second_plan_span_overflow", func(t *testing.T) {
		s := minimalSamples()
		s[1].Document.Spans = nil
		for i := 0; i < 2001; i++ {
			span := s[0].Document.Spans[0]
			span.ID = fmt.Sprintf("large-%d", i)
			span.Word = i + 1
			span.Text = "12345678901234.00"
			s[1].Document.Spans = append(s[1].Document.Spans, span)
		}
		r, m := &boundedReasoner{}, &fakeMeter{}
		_, e := Run(context.Background(), s, r, m)
		if e == nil || r.calls != 0 || m.reserves != 0 {
			t.Fatal(e, r, m)
		}
	})
	for _, invalid := range []bool{false, true} {
		t.Run(map[bool]string{false: "cancel_known", true: "invalid_bill_uncertain"}[invalid], func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			r := &boundedReasoner{cancel: cancel, invalid: invalid}
			m := &fakeMeter{}
			report, e := Run(ctx, minimalSamples(), r, m)
			if e == nil || report.Passed || r.calls != 1 || m.settles != 1 || r.delayBound < 119*time.Second || r.delayBound > 120*time.Second {
				t.Fatal(e, r, m)
			}
			if invalid && m.actual != nil || !invalid && (m.actual == nil || *m.actual != 10) {
				t.Fatal("lost/fabricated billing")
			}
		})
	}
}
