//go:build integration

package statementqualification

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/QahNaaRlN/grace/internal/statementdata"
	"github.com/QahNaaRlN/grace/internal/statementextract"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/QahNaaRlN/grace/internal/pdfextract"
)

func TestTurnoverRolesRealHelper(t *testing.T) {
	socket := os.Getenv("TEST_PDF_EXTRACTOR_SOCKET")
	if socket == "" {
		t.Skip("isolated pinned helper required")
	}
	helper, e := pdfextract.NewClient(socket)
	if e != nil {
		t.Fatal(e)
	}
	raw, e := os.ReadFile("../pdfextract/testdata/turnover-roles-ru.pdf")
	if e != nil {
		t.Fatal(e)
	}
	d, e := helper.Extract(context.Background(), raw)
	if e != nil {
		t.Fatal(e)
	}
	assertTurnoverFixture(t, d)
}

type screeningFixtureReasoner struct {
	activationFixtureReasoner
	ru    statementdata.Proposal
	badRU bool
}

func (*screeningFixtureReasoner) CurrentPolicy(context.Context) (statementextract.Policy, error) {
	return statementextract.ScreeningPolicy(), nil
}
func (f *screeningFixtureReasoner) Complete(ctx context.Context, p statementextract.Policy, c statementextract.Chunk) (statementdata.Proposal, *statementextract.Billing, error) {
	ru := false
	for _, s := range c.Spans {
		if s.Text == "12.10.2026" {
			ru = true
		}
	}
	if !ru {
		return f.activationFixtureReasoner.Complete(ctx, p, c)
	}
	f.calls++
	out := f.ru
	if f.badRU {
		out.Rows = append([]statementdata.RowMapping{}, out.Rows...)
		out.Rows[0].Description = nil
	}
	return out, &statementextract.Billing{GenerationID: "synthetic-screening-ru", CostMicros: 100, InputTokens: 25000, OutputTokens: 3000}, nil
}
func TestScreeningRealHelperThreeCases(t *testing.T) {
	socket := os.Getenv("TEST_PDF_EXTRACTOR_SOCKET")
	if socket == "" {
		t.Skip("isolated pinned helper required")
	}
	helper, e := pdfextract.NewClient(socket)
	if e != nil {
		t.Fatal(e)
	}
	samples := []Sample{}
	var ruWire statementdata.Proposal
	for _, name := range ScreeningNames() {
		raw, e := os.ReadFile("../pdfextract/testdata/" + name + ".pdf")
		if e != nil {
			t.Fatal(e)
		}
		d, e := helper.Extract(context.Background(), raw)
		if e != nil {
			t.Fatal(e)
		}
		samples = append(samples, Sample{Name: name, Document: d, SHA256: FixtureHash(name)})
		if name == "turnover-roles-ru" {
			canonical, _ := turnoverFixture(t, d)
			plan, e := statementextract.BuildScreeningPlan(d, statementextract.ScreeningPolicy())
			if e != nil {
				t.Fatal(e)
			}
			encoded, _ := json.Marshal(canonical)
			for alias, id := range plan.Aliases {
				encoded = []byte(strings.ReplaceAll(string(encoded), `"`+id+`"`, `"`+alias+`"`))
			}
			if json.Unmarshal(encoded, &ruWire) != nil {
				t.Fatal("wire")
			}
		}
	}
	for _, bad := range []bool{false, true} {
		t.Run(fmt.Sprint(bad), func(t *testing.T) {
			reasoner := &screeningFixtureReasoner{ru: ruWire, badRU: bad}
			m := &fakeMeter{}
			report, e := RunScreening(context.Background(), samples, reasoner, m)
			if reasoner.calls != 3 || m.reserves != 1 || m.reserved != 161700 || m.settles != 1 || m.actual == nil || *m.actual != 300 {
				t.Fatal("metering", reasoner.calls, m, e)
			}
			if !bad {
				if e != nil || !report.Passed || ValidateScreeningReport(report, time.Now()) != nil || ValidateReport(report, time.Now()) == nil {
					t.Fatal("screening/activation", report, e)
				}
				for _, result := range report.Results {
					t.Logf("%s %d/%d", result.Name, result.Correct, result.Checks)
				}
			} else if e == nil || report.Passed || len(report.Results) != 3 || !strings.Contains(strings.Join(report.Results[2].Codes, "|"), "record_description") {
				t.Fatal("RU scorer weakened", report, e)
			}
		})
	}
}
