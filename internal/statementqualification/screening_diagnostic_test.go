package statementqualification

import (
	"context"
	"encoding/json"
	"github.com/QahNaaRlN/grace/internal/statementdata"
	"github.com/QahNaaRlN/grace/internal/statementextract"
	"strings"
	"testing"
	"time"
)

type diagnosticReasoner struct {
	screeningFailureReasoner
	diagnostic statementextract.ScreeningDiagnostic
}

func (d *diagnosticReasoner) Error() string { return "PRIVATE-error-text" }
func (d *diagnosticReasoner) ScreeningDiagnostic() statementextract.ScreeningDiagnostic {
	return d.diagnostic
}
func (d *diagnosticReasoner) Complete(context.Context, statementextract.Policy, statementextract.Chunk) (statementdata.Proposal, *statementextract.Billing, error) {
	d.calls++
	return statementdata.Proposal{}, d.billing, d
}

func TestScreeningDiagnosticNeverAuthorizesOrSettlesCorrelation(t *testing.T) {
	doc := statementdata.Document{Extractor: statementdata.ExtractorVersion, Pages: []statementdata.Page{{Number: 1, Width: 600, Height: 800}}, Spans: []statementdata.Span{{ID: "s", Page: 1, Block: 1, Line: 1, Word: 1, Text: "-1.00", Box: statementdata.Box{XMin: 1, YMin: 1, XMax: 10, YMax: 10}}}}
	samples := []Sample{}
	for _, name := range ScreeningNames() {
		samples = append(samples, Sample{name, doc, FixtureHash(name)})
	}
	for _, known := range []bool{false, true} {
		for _, code := range []string{"metadata_missing", "PRIVATE-unrecognized-code"} {
			f := &diagnosticReasoner{diagnostic: statementextract.ScreeningDiagnostic{Code: code, HTTPStatus: 200, ContentType: "json", CorrelationID: "gen-SYNTHETIC-private-correlation"}}
			if known {
				f.billing = &statementextract.Billing{GenerationID: "verified-synthetic", CostMicros: 100, InputTokens: 10, OutputTokens: 10}
			}
			meter := &fakeMeter{}
			report, err := RunScreening(context.Background(), samples, f, meter)
			want := f.diagnostic.Sanitize().Code
			if err == nil || err.Error() != "qualification_provider_failed_"+want || strings.Contains(err.Error(), "PRIVATE") || report.Failure == nil || report.Failure.Code != want || report.Failure.HTTPStatus != 200 || report.Failure.CorrelationID != f.diagnostic.CorrelationID || len(report.Results) != 0 || report.Passed || f.calls != 1 || meter.reserves != 1 || meter.settles != 1 {
				t.Fatal("diagnostic failure boundary", report, err, meter, f.calls)
			}
			if known {
				if meter.actual == nil || *meter.actual != 100 || len(report.Calls) != 1 {
					t.Fatal("lost known billing")
				}
			} else if meter.actual != nil || report.ActualMicros != nil || len(report.Calls) != 0 {
				t.Fatal("unverified correlation became known billing")
			}
			if ValidateReport(report, time.Now()) == nil || ValidateScreeningReport(report, time.Now()) == nil {
				t.Fatal("failed report authorized")
			}
			raw, _ := json.Marshal(report)
			if !strings.Contains(string(raw), f.diagnostic.CorrelationID) {
				t.Fatal("private correlation lost")
			}
		}
	}
	// A diagnostic cannot be attached to a successful activation or screening report.
	for _, screen := range []bool{false, true} {
		r := validReportFixture()
		if screen {
			r = screeningReportFixture()
		}
		r.Failure = &statementextract.ScreeningDiagnostic{Code: "unknown"}
		r.Digest = reportDigest(r)
		if screen && ValidateScreeningReport(r, time.Now()) == nil || !screen && ValidateReport(r, time.Now()) == nil {
			t.Fatal("diagnostic report passed")
		}
	}
	legacy := screeningReportFixture()
	legacy.Version = "statement.screening.v1.gemini31-ru-roles"
	legacy.Digest = reportDigest(legacy)
	if ValidateScreeningReport(legacy, time.Now()) == nil {
		t.Fatal("legacy profile passed new validator")
	}
}
