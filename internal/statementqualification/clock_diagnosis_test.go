package statementqualification

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/QahNaaRlN/grace/internal/statementdata"
	"github.com/QahNaaRlN/grace/internal/statementextract"
)

type diagnosisFake struct {
	clockFake
	kind string
}

func (f *diagnosisFake) CompleteClock(ctx context.Context, d statementdata.Document, v string) (statementdata.Proposal, *statementextract.Billing, error) {
	p, b, e := f.clockFake.CompleteClock(ctx, d, v)
	if f.kind == "shared" {
		p.Rows[1].Description = p.Rows[0].Description
	}
	if v == "candidate" {
		if f.kind == "target" || f.kind == "combined" {
			p = clockDrop(p)
		}
		if f.kind != "target" {
			p.Rows[0].TransactionCurrency = p.Rows[0].AccountCurrency
		}
	}
	return p, b, e
}
func diagnosisReport(t *testing.T, kind string) (ClockReport, []byte) {
	t.Helper()
	d := clockTestDocument(t)
	r, e := RunClockPilot(context.Background(), Sample{"cards", d, FixtureHash("cards")}, &diagnosisFake{clockFake: clockFake{t: t}, kind: kind}, &clockMeterFake{}, ClockProducerSource)
	if e != nil || r.Decision != ClockReject {
		t.Fatal("actual causal report", e, r.Decision)
	}
	var b bytes.Buffer
	if EncodeClockReport(&b, r, d) != nil {
		t.Fatal("encode actual report")
	}
	return r, b.Bytes()
}
func TestClockDiagnosisActualPipeline(t *testing.T) {
	reader := strings.Repeat("1", 40)
	for _, kind := range []string{"regression", "shared", "target", "combined"} {
		t.Run(kind, func(t *testing.T) {
			r, raw := diagnosisReport(t, kind)
			d, e := DiagnoseClock(raw, clockTestDocument(t), reader)
			if kind == "target" || kind == "combined" {
				if ClockDiagnosisReason(e) != "summary_mismatch" {
					t.Fatal("changed historical summary", e)
				}
				// These causes are reachable through the real pipeline, but are not the
				// historical counts. Exercise differences using its replayed measurements.
				attempts := []ClockDiagnosisAttempt{}
				for _, a := range r.Attempts {
					m := a.Measurement
					checks := []ClockDiagnosisCheck{}
					for _, c := range m.Checks {
						checks = append(checks, ClockDiagnosisCheck{c.Ordinal, c.RowOrdinal, c.Code, c.Passed})
					}
					attempts = append(attempts, ClockDiagnosisAttempt{Ordinal: a.Ordinal, Variant: a.Variant, ClockCount: *m.Count, Precision: m.Precision, Checks: checks})
				}
				pairs, reasons := clockPairDifferences(attempts)
				if !pairs[0].CandidateClockTargetFailed || len(pairs[0].PrecisionFailures) != 4 || len(reasons) < 2 {
					t.Fatal("lost target/precision", pairs, reasons)
				}
				if kind == "combined" && len(pairs[0].NonClockRegressions) == 0 {
					t.Fatal("lost combined regression")
				}
				return
			}
			if e != nil || ValidateClockDiagnosis(d, reader) != nil {
				t.Fatal(e)
			}
			if len(d.Attempts) != 6 || len(d.Pairs) != 3 || len(d.RejectReasons) != 1 || d.RejectReasons[0] != "non_clock_regression" || d.Accounting.ActualMicros != 60 {
				t.Fatal("incomplete projection")
			}
			total := 0
			for _, a := range d.Attempts {
				total += len(a.Checks)
				if a.Ordinal != 1 && a.Ordinal != 4 && a.Ordinal != 5 && a.ClockCount != 4 {
					t.Fatal("clock gain")
				}
			}
			if total != 120 || len(d.Pairs[0].NonClockRegressions) == 0 || d.Pairs[1].BaseOrdinal != 4 || d.Pairs[1].CandidateOrdinal != 3 {
				t.Fatal("trace or reversed pair lost")
			}
			if kind == "shared" {
				for _, a := range d.Attempts {
					if a.Checks[12].Passed {
						t.Fatal("shared failure hidden")
					}
				}
			}
			encoded, _ := json.Marshal(d)
			again, e := DiagnoseClock(raw, clockTestDocument(t), reader)
			other, _ := json.Marshal(again)
			if e != nil || !bytes.Equal(encoded, other) || bytes.Contains(encoded, []byte("PRIVATE")) || ValidateClockOutput(encoded, reader) != nil {
				t.Fatal("determinism/privacy")
			}
			if kind == "regression" && os.Getenv("TEST_S120_OUTPUT") != "" {
				// Explicit hermetic fixture export for CLI/transport tests. Never a live report.
				dir := os.Getenv("TEST_S120_OUTPUT")
				ctx, e := os.ReadFile("../statementextract/testdata/s119-cards-document.json")
				if e != nil {
					t.Fatal(e)
				}
				for n, b := range map[string][]byte{"report.json": raw, "context.json": ctx, "projection.json": encoded} {
					if os.WriteFile(filepath.Join(dir, n), b, 0600) != nil {
						t.Fatal("export fixture")
					}
				}
			}
		})
	}
}
func TestClockDiagnosisRejectsForgedEvidence(t *testing.T) {
	reader := strings.Repeat("1", 40)
	r, raw := diagnosisReport(t, "regression")
	d := clockTestDocument(t)
	for _, kind := range []string{"precision", "score", "digest", "context", "source", "summary", "unknown"} {
		t.Run(kind, func(t *testing.T) {
			var changed ClockReport
			json.Unmarshal(raw, &changed)
			switch kind {
			case "precision":
				changed.Attempts[1].Measurement.Precision[0] = false
			case "score":
				changed.Attempts[1].Measurement.Result.Correct++
			case "digest":
				changed.Digest = strings.Repeat("f", 64)
			case "context":
				changed.DocumentHash = strings.Repeat("f", 64)
			case "source":
				changed.Source = strings.Repeat("f", 40)
			case "summary":
				changed.Decision = ClockKeep
			case "unknown":
				changed.Attempts[1].Measurement.Checks[0].Code = "PRIVATE"
			}
			if kind != "digest" {
				changed.Digest = clockDigest(changed)
			}
			b, _ := json.Marshal(changed)
			if _, e := DiagnoseClock(b, d, reader); e == nil {
				t.Fatal("forged cached evidence accepted")
			}
		})
	}
	if r.Digest == "" || clockCodeKnown("PRIVATE") {
		t.Fatal("closed vocabulary")
	}
	out, e := DiagnoseClock(raw, d, reader)
	if e != nil {
		t.Fatal(e)
	}
	good, _ := json.Marshal(out)
	for _, mutate := range []func(*ClockDiagnosis){
		func(x *ClockDiagnosis) { x.Attempts[0].Checks = x.Attempts[0].Checks[:19] },
		func(x *ClockDiagnosis) { x.Attempts[0].Checks[0].Code = "PRIVATE" },
		func(x *ClockDiagnosis) { x.Attempts[0].Correct++ },
		func(x *ClockDiagnosis) { x.Pairs[0].NonClockRegressions = []ClockRegression{} },
		func(x *ClockDiagnosis) { x.RejectReasons = []string{} },
		func(x *ClockDiagnosis) { x.Accounting.ActualMicros++ },
		func(x *ClockDiagnosis) { x.Accounting.CallCostsMicros[0] = -1 },
		func(x *ClockDiagnosis) { x.Accounting.CallCostsMicros[0] = 9223372036854775807 },
		func(x *ClockDiagnosis) { x.ReaderSource = strings.Repeat("f", 40) },
		func(x *ClockDiagnosis) { x.ActivationAuthorized = true },
	} {
		var x ClockDiagnosis
		json.Unmarshal(good, &x)
		mutate(&x)
		b, _ := json.Marshal(x)
		if ValidateClockOutput(b, reader) == nil {
			t.Fatal("forged closed output")
		}
	}
	for _, bad := range [][]byte{append(append([]byte{}, good...), []byte(" null")...), bytes.Replace(good, []byte(`"providerCalls":0`), []byte(`"providerCalls":null`), 1), bytes.Replace(good, []byte(`"providerCalls":0`), []byte(`"ProviderCalls":0`), 1), bytes.Replace(good, []byte(`"providerCalls":0`), []byte(`"providerCalls":0,"providerCalls":0`), 1), bytes.Replace(good, []byte(`"precision":[`), []byte(`"precision":[false,`), 1)} {
		if ValidateClockOutput(bad, reader) == nil {
			t.Fatal("open JSON shape")
		}
	}
	if !reflect.DeepEqual(ClockOrder(), [6]string{"base", "candidate", "candidate", "base", "base", "candidate"}) {
		t.Fatal("producer order")
	}
}
