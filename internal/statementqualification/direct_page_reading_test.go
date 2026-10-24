package statementqualification

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/QahNaaRlN/grace/internal/statementextract"
)

func directTestCorpus(t *testing.T) DirectCorpus {
	t.Helper()
	root := filepath.Join("..", "..")
	c, e := LoadDirectCorpus(root)
	if e != nil {
		t.Fatal(e)
	}
	return c
}
func cloneReading(t *testing.T, r statementextract.DirectReading) statementextract.DirectReading {
	t.Helper()
	b, _ := json.Marshal(r)
	v, e := statementextract.DecodeDirectReading(b)
	if e != nil {
		t.Fatal(e)
	}
	return v
}
func directFakeResponse(t *testing.T, r statementextract.DirectReading) json.RawMessage {
	t.Helper()
	b, _ := json.Marshal(r)
	raw, _ := json.Marshal(map[string]any{"model": statementextract.DirectModel, "status": "completed", "service_tier": "default", "output": []any{map[string]any{"type": "message", "role": "assistant", "status": "completed", "content": []any{map[string]any{"type": "output_text", "text": string(b)}}}}, "usage": map[string]int{"input_tokens": 1000, "output_tokens": 2000, "total_tokens": 3000}})
	return raw
}
func TestDirectFourPositiveControls(t *testing.T) {
	corpus := directTestCorpus(t)
	for _, c := range corpus.Cases {
		t.Run(c.Name, func(t *testing.T) {
			m, e := MeasureDirect(c, c.Golden)
			if e != nil {
				t.Fatal(e)
			}
			if !m.RecognitionPassed || !m.SourceBindingPassed {
				for _, b := range m.Binding.Checks {
					if !b.Passed {
						t.Logf("binding %+v", b)
					}
				}
				t.Fatalf("recognition=%v binding=%v blockers=%+v rowissues=%+v", m.RecognitionPassed, m.SourceBindingPassed, m.Binding.Review.Reconciliation.Blockers, m.Binding.Review.Rows)
			}
			if m.ImportReady || m.Binding.Review.Reconciliation.RowCoverage.Status == "matched" {
				t.Fatal("owner/coverage guard bypassed")
			}
			blockers := strings.Join(m.Binding.Review.Reconciliation.Blockers, ",")
			for _, s := range []string{"source_coverage_incomplete", "owner_page_review_required", "owner_field_role_review_required"} {
				if !strings.Contains(blockers, s) {
					t.Fatalf("missing %s: %s", s, blockers)
				}
			}
			if c.Name != "full32" && (m.LegacyResult == nil || len(m.LegacyChecks) == 0 || m.LegacyResult.Passed) {
				t.Fatal("unchanged old full scorer missing or deceptively passed")
			}
			t.Logf("rows=%d recognitionChecks=%d bindingChecks=%d blockers=%v", len(m.Binding.Review.Rows), len(m.Checks), len(m.Binding.Checks), m.Binding.Review.Reconciliation.Blockers)
			if c.Name == "full32" && (len(m.Binding.Review.Rows) != 32 || len(c.Document.Pages) != 3 || len(m.Checks) != 297) {
				t.Fatalf("full32 coverage %d", len(m.Checks))
			}
		})
	}
}
func TestDirectMeaningfulSemanticAndBindingNegatives(t *testing.T) {
	corpus := directTestCorpus(t)
	tests := []struct {
		name                         string
		caseIndex                    int
		mutate                       func(*statementextract.DirectReading)
		recognitionFail, bindingFail bool
	}{
		{"missing_row", 3, func(r *statementextract.DirectReading) { r.Rows = append(r.Rows[:14], r.Rows[15:]...) }, true, false},
		{"duplicate_row", 3, func(r *statementextract.DirectReading) { r.Rows = append(r.Rows, r.Rows[0]) }, true, true},
		{"one_minor", 0, func(r *statementextract.DirectReading) { r.Rows[0].Fields["accountAmount"].Value = "-1234.57" }, true, true},
		{"wrong_currency", 0, func(r *statementextract.DirectReading) { r.Rows[0].Fields["accountCurrency"].Value = "USD" }, true, true},
		{"lost_time_precision", 0, func(r *statementextract.DirectReading) { r.Rows[0].Fields["operationDate"].Value = "19.10.2026" }, true, false},
		{"invent_seconds", 0, func(r *statementextract.DirectReading) { r.Rows[0].Fields["operationDate"].Value += "" }, true, false},
		{"lost_description_word", 0, func(r *statementextract.DirectReading) { r.Rows[0].Fields["description"].Value = "SYNTHETIC" }, true, true},
		{"available_as_closing", 1, func(r *statementextract.DirectReading) {
			r.Header["closingBookBalance"] = r.Header["availableBalanceObserved"]
		}, true, true},
		{"false_quote", 0, func(r *statementextract.DirectReading) {
			r.Rows[0].Fields["description"].Evidence[0].Quote = "INVENTED SHOP"
		}, false, true},
		{"wrong_page", 1, func(r *statementextract.DirectReading) { r.Rows[1].Fields["description"].Evidence[0].Page = 1 }, false, true},
		{"cross_page_omission", 3, func(r *statementextract.DirectReading) {
			r.Rows[10].Fields["description"].Evidence = r.Rows[10].Fields["description"].Evidence[:1]
		}, false, true},
		{"injection_obeyed", 0, func(r *statementextract.DirectReading) { r.Rows[1].Fields["description"].Value = "INCOME" }, true, true},
		{"signed_direction", 0, func(r *statementextract.DirectReading) { r.Rows[0].Fields["direction"].Value = "credit" }, true, true},
	}
	tests = append(tests, struct {
		name                         string
		caseIndex                    int
		mutate                       func(*statementextract.DirectReading)
		recognitionFail, bindingFail bool
	}{"omitted_page", 3, func(r *statementextract.DirectReading) {
		rows := []statementextract.DirectRow{}
		for _, row := range r.Rows {
			if row.Anchor[0].Page != 2 {
				rows = append(rows, row)
			}
		}
		r.Rows = rows
	}, true, false})
	tests = append(tests, struct {
		name                         string
		caseIndex                    int
		mutate                       func(*statementextract.DirectReading)
		recognitionFail, bindingFail bool
	}{"header_currency_conflict", 1, func(r *statementextract.DirectReading) {
		r.Header["currency"] = &statementextract.DirectField{Value: "USD", Evidence: []statementextract.PageQuote{{Page: 1, Quote: "Original USD"}}}
	}, true, true})
	tests = append(tests, struct {
		name                         string
		caseIndex                    int
		mutate                       func(*statementextract.DirectReading)
		recognitionFail, bindingFail bool
	}{"direction_wrong_source", 0, func(r *statementextract.DirectReading) {
		r.Rows[0].Fields["direction"].Evidence = []statementextract.PageQuote{{Page: 1, Quote: "Original USD -15.00"}}
	}, false, true})
	tests[5].mutate = func(r *statementextract.DirectReading) {
		r.Rows[0].Fields["operationDate"].Value = "19.10.2026 10:30:00"
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := corpus.Cases[tt.caseIndex]
			r := cloneReading(t, c.Golden)
			tt.mutate(&r)
			m, e := MeasureDirect(c, r)
			if e != nil {
				t.Fatal(e)
			}
			if tt.recognitionFail && m.RecognitionPassed {
				t.Fatal("recognition accepted semantic defect")
			}
			if tt.bindingFail && m.SourceBindingPassed {
				t.Fatalf("binding accepted %+v", m.Binding.Checks)
			}
			if m.ImportReady {
				t.Fatal("import ready")
			}
		})
	}
	// Same signed amount occurs in distinct full32 rows. A global first match is
	// forbidden, but proper row anchors permit the two genuine repeated amounts.
	c := corpus.Cases[3]
	r := cloneReading(t, c.Golden)
	r.Rows[1].Anchor = []statementextract.PageQuote{{Page: 1, Quote: "-100.00"}}
	m, e := MeasureDirect(c, r)
	if e != nil {
		t.Fatal(e)
	}
	if m.SourceBindingPassed {
		t.Fatal("ambiguous anchor accepted")
	}
}
func TestDirectCampaignReplayAndWitness(t *testing.T) {
	c := directTestCorpus(t)
	source := strings.Repeat("a", 40)
	responses := []DirectCapture{}
	for i := 0; i < 8; i++ {
		responses = append(responses, DirectCapture{Response: directFakeResponse(t, c.Cases[i%4].Golden), DurationMS: 123})
	}
	report, e := ReplayDirect(c, source, "fake-only", responses)
	if e != nil {
		t.Fatal(e)
	}
	if report.Decision != "KEEP_FOR_FURTHER_REVIEW" || report.ImportReady || report.CalculatedProviderCostMicros == nil {
		t.Fatalf("%+v", report)
	}
	raw, _ := json.Marshal(report)
	verified, e := VerifyDirectReport(c, raw, source, "fake-only")
	if e != nil || !reflect.DeepEqual(report, verified) {
		t.Fatalf("replay %v", e)
	}
	report.ImportReady = true
	raw, _ = json.Marshal(report)
	if _, e = VerifyDirectReport(c, raw, source, "fake-only"); e == nil {
		t.Fatal("tampered importReady")
	}
	r := cloneReading(t, c.Cases[0].Golden)
	r.Rows[0].Fields["accountAmount"].Value = "-1234.57"
	bad := directFakeResponse(t, r)
	report, e = ReplayDirect(c, source, "fake-only", []DirectCapture{{Response: bad}})
	if e != nil || report.Decision != "REJECT" {
		t.Fatal(e, report.Decision)
	}
	if report.Attempts[1].State != "not_sent" {
		t.Fatal("suffix sent")
	}
	witness := false
	for _, check := range report.Attempts[0].Measurement.Checks {
		if !check.Passed && check.Field == "accountAmount" && check.Expected != nil && check.Observed != nil && len(check.Evidence) > 0 {
			witness = true
		}
	}
	if !witness {
		t.Fatal("concrete witness missing")
	}
	if _, e = ReplayDirect(c, source, "fake-only", []DirectCapture{{Response: bad}, responses[1]}); e == nil {
		t.Fatal("response after reject")
	}
	var wire map[string]any
	json.Unmarshal(bad, &wire)
	delete(wire, "usage")
	bad, _ = json.Marshal(wire)
	report, e = ReplayDirect(c, source, "fake-only", []DirectCapture{{Response: bad}})
	if e != nil || report.Decision != "REJECT" || report.AllowanceState != "uncertain" {
		t.Fatal("unknown usage hid semantic witness", e, report.Decision)
	}
	good := directFakeResponse(t, c.Cases[0].Golden)
	json.Unmarshal(good, &wire)
	delete(wire, "usage")
	unknown, _ := json.Marshal(wire)
	report, e = ReplayDirect(c, source, "fake-only", []DirectCapture{{Response: unknown}})
	if e != nil || report.Decision != "INCONCLUSIVE" || report.CalculatedProviderCostMicros != nil {
		t.Fatal("unknown usage not frozen", e)
	}
	report, e = ReplayDirect(c, source, "fake-only", []DirectCapture{{Failure: "transport_unknown_usage", DurationMS: 180000}})
	if e != nil || report.Decision != "INCONCLUSIVE" || report.Attempts[1].State != "not_sent" {
		t.Fatal("timeout retry", e)
	}
}
func TestDirectHistoricalFixturesUnchanged(t *testing.T) {
	// This test pins the bytes of the historical source used by earlier pilots.
	b, e := os.ReadFile("../statementextract/testdata/s119-cards-document.json")
	if e != nil {
		t.Fatal(e)
	}
	if statementextract.DirectHash(b) != "eefde6959b1bbfe8c94e4559fee1103dfeac98d10c0f415f8673d97b661cdbdf" {
		t.Fatal("historical source bytes changed")
	}
}

func TestDirectReplayRejectsTampering(t *testing.T) {
	c := directTestCorpus(t)
	source := strings.Repeat("a", 40)
	responses := []DirectCapture{}
	for i := 0; i < 8; i++ {
		responses = append(responses, DirectCapture{Response: directFakeResponse(t, c.Cases[i%4].Golden), DurationMS: 123})
	}
	original, e := ReplayDirect(c, source, "fake-only", responses)
	if e != nil {
		t.Fatal(e)
	}
	for _, kind := range []string{"source", "approval", "manifest", "decision", "cost", "request", "recognition", "binding", "old_checks", "blockers", "case", "not_sent"} {
		t.Run(kind, func(t *testing.T) {
			raw, _ := json.Marshal(original)
			var r DirectReport
			if json.Unmarshal(raw, &r) != nil {
				t.Fatal("test decode")
			}
			switch kind {
			case "source":
				r.Source = strings.Repeat("b", 40)
			case "approval":
				r.Approval = "different"
			case "manifest":
				r.ManifestSHA256 = "different"
			case "decision":
				r.Decision = "REJECT"
			case "cost":
				cost := int64(0)
				r.CalculatedProviderCostMicros = &cost
			case "request":
				r.Attempts[0].Observation.RequestSHA256 = "different"
			case "recognition":
				r.Attempts[0].Measurement.Checks[0].Passed = !r.Attempts[0].Measurement.Checks[0].Passed
			case "binding":
				r.Attempts[0].Measurement.SourceBindingPassed = false
			case "old_checks":
				r.Attempts[0].Measurement.LegacyChecks = nil
			case "blockers":
				r.Attempts[0].Measurement.Binding.Review.Reconciliation.Blockers = nil
			case "case":
				r.Attempts[0].Case = "full32"
			case "not_sent":
				r.Attempts[0].State = "not_sent"
			}
			raw, _ = json.Marshal(r)
			if _, e = VerifyDirectReport(c, raw, source, "fake-only"); e == nil {
				t.Fatalf("tampering accepted %s", kind)
			}
		})
	}
}
