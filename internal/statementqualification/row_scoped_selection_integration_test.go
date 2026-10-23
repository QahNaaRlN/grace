//go:build integration

package statementqualification

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/QahNaaRlN/grace/internal/pdfextract"
	"github.com/QahNaaRlN/grace/internal/statementextract"
)

func TestRowScopedRealHelper(t *testing.T) {
	socket := os.Getenv("TEST_PDF_EXTRACTOR_SOCKET")
	if socket == "" {
		t.Fatal("pinned helper required")
	}
	helper, e := pdfextract.NewClient(socket)
	if e != nil {
		t.Fatal(e)
	}
	for _, name := range ScreeningNames() {
		t.Run(name, func(t *testing.T) {
			raw, e := os.ReadFile("../pdfextract/testdata/" + name + ".pdf")
			if e != nil {
				t.Fatal(e)
			}
			digest := sha256.Sum256(raw)
			if hex.EncodeToString(digest[:]) != FixtureHash(name) {
				t.Fatal("public PDF hash changed")
			}
			d, e := helper.Extract(context.Background(), raw)
			if e != nil {
				t.Fatal(e)
			}
			before := d.Hash()
			base, baseChecks, baseBytes := rowScopedBaseline(t, d, name)
			if !base.Passed {
				t.Fatal("one-call base full golden", base)
			}
			m := &rowScopedMeter{}
			f := &rowScopedFake{t: t, meter: m}
			good, e := rowScopedRun(t, d, name, f)
			if good.Result != nil && good.Result.Checks != map[string]int{"columns": 19, "cards": 20, "turnover-roles-ru": 23}[name] {
				t.Fatal("full trace count changed", good.Result)
			}
			if e != nil || !good.FullQualityPassed || good.Result == nil || !good.Result.Passed || good.Result.Checks != len(good.Checks) || f.calls != 3 || m.actual == nil || *m.actual != 30 {
				t.Fatal("full actual pipeline", good, e, m)
			}
			if before != d.Hash() {
				t.Fatal("source changed")
			}
			t.Logf("S121_PUBLIC name=%s documentHash=%s planHash=%s checks=%d/%d calls=%d precision=%v", name, before, good.PlanHash, good.Result.Correct, good.Result.Checks, f.calls, good.Precision)
			trace, e := json.Marshal(struct {
				BaselineChecks, CandidateChecks []ScoringCheck
				Precision                       []RowScopedPrecision
			}{baseChecks, good.Checks, good.Precision})
			if e != nil {
				t.Fatal(e)
			}
			t.Logf("S121_TRACE name=%s json=%s", name, trace)
			totalBytes := 0
			for _, request := range f.requests {
				totalBytes += request.InputBytes
			}
			t.Logf("S121_FAKE_BASELINE name=%s baseChecks=%d/%d baseCalls=1 baseBytes=%d candidateCalls=%d candidateBytes=%d", name, base.Correct, base.Checks, baseBytes, len(f.requests), totalBytes)
			for _, r := range f.requests {
				t.Logf("S121_REQUEST name=%s stage=%s row=%d bytes=%d prompt=%s schema=%s request=%s", name, r.Stage, r.RowOrdinal, r.InputBytes, r.PromptSHA, statementextract.PromptSHA(string(r.Schema)), r.Hash)
			}
			modes := []string{"prefix", "suffix", "separator", "missing_description", "missing_row", "wrong_fx", "missing_currency"}
			if name == "cards" {
				modes = append(modes, "missing_clock")
			}
			if name == "turnover-roles-ru" {
				modes = []string{"prefix", "suffix", "separator", "missing_description", "missing_row", "identifier", "missing_available_label", "missing_currency"}
			}
			for _, mode := range modes {
				t.Run(mode, func(t *testing.T) {
					meter := &rowScopedMeter{}
					fake := &rowScopedFake{t: t, meter: meter, groupFault: mode}
					if mode == "missing_description" || mode == "missing_clock" || mode == "missing_currency" {
						fake.groupFault = ""
						fake.fieldFault = mode
					}
					if mode == "wrong_fx" {
						fake.fieldFault = mode
					}
					out, err := rowScopedRun(t, d, name, fake)
					if out.FullQualityPassed || meter.settles != 1 || meter.actual == nil || *meter.actual != int64(fake.calls*10) {
						t.Fatal("counterexample/accounting", out, err, meter)
					}
					if mode == "missing_description" || mode == "missing_clock" && name == "cards" {
						if err == nil || out.Result != nil || out.Cause != "row_coverage_incomplete" {
							t.Fatal("omitted owned fragment accepted", out, err)
						}
						return
					}
					// columns contains no printed clock: a request to omit a clock is a no-op,
					// so that case is excluded below rather than manufactured into a failure.
					if err != nil || out.Result == nil || out.Result.Passed {
						t.Fatal("actual scorer failure required", mode, out, err)
					}
					if mode == "missing_currency" {
						found := false
						for _, check := range out.Checks {
							if check.RowOrdinal == 1 && check.Code == "account_amount_currency" && !check.Passed {
								found = true
							}
						}
						if !found {
							t.Fatal("header currency silently filled row", out)
						}
					}
					if mode == "prefix" || mode == "suffix" || mode == "separator" {
						if !strings.Contains(strings.Join(out.Result.Codes, "|"), "record_description") {
							t.Fatal("description golden failure lost", out)
						}
					}
				})
			}
		})
	}
}
