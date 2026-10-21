//go:build integration

package statementqualification

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/QahNaaRlN/grace/internal/pdfextract"
	"github.com/QahNaaRlN/grace/internal/statementdata"
	"github.com/QahNaaRlN/grace/internal/statementextract"
	"math"
	"os"
	"testing"
	"time"
)

type comparisonFixtureReasoner struct {
	activationFixtureReasoner
	candidate statementextract.ComparisonCandidate
	badRole   bool
}

func (f *comparisonFixtureReasoner) CurrentPolicy(context.Context) (statementextract.Policy, error) {
	return f.candidate.Policy, nil
}
func (f *comparisonFixtureReasoner) Complete(ctx context.Context, p statementextract.Policy, c statementextract.Chunk) (statementdata.Proposal, *statementextract.Billing, error) {
	ru := false
	for _, s := range c.Spans {
		ru = ru || s.Text == "12.10.2026"
	}
	var proposal statementdata.Proposal
	var bill *statementextract.Billing
	var e error
	if !ru {
		proposal, bill, e = f.activationFixtureReasoner.Complete(ctx, p, c)
	} else {
		f.calls++
		// The fake sees only supplied aliases/geometry. It cannot read the layout,
		// golden selectors, original descriptions, hidden digits or canonical map.
		find := func(text string) statementextract.RedactedSpan {
			for _, s := range c.Spans {
				if s.Text == text {
					return s
				}
			}
			return statementextract.RedactedSpan{}
		}
		ids := func(texts ...string) statementdata.Selection {
			out := statementdata.Selection{}
			for _, s := range texts {
				out = append(out, find(s).ID)
			}
			return out
		}
		line := func(anchor string, delta int, numeric bool) statementdata.Selection {
			a := find(anchor)
			out := statementdata.Selection{}
			for _, s := range c.Spans {
				if s.Page == a.Page && math.Abs(float64(s.Y-a.Y-delta)) <= 1 && (!numeric || s.LexicalClass == statementextract.ClassNumericFragment) {
					out = append(out, s.ID)
				}
			}
			return out
		}
		proposal = statementdata.Proposal{Header: statementdata.HeaderMapping{Currency: ids("RUB"), AvailableBalanceObserved: ids("0,00"), DebitTurnover: line("Расходы:", 0, true), CreditTurnover: line("Пополнения:", 0, true), Labels: map[string]statementdata.Selection{"availableBalanceObserved": ids("Доступный", "остаток"), "debitTurnover": ids("Расходы:"), "creditTurnover": ids("Пополнения:")}}, Rows: []statementdata.RowMapping{{OperationDate: ids("12.10.2026"), AccountAmount: ids("-2486,00"), AccountCurrency: ids("RUB"), Description: line("12.10.2026", 25, false)}, {OperationDate: ids("13.10.2026"), AccountAmount: ids("+3105,00"), AccountCurrency: ids("RUB"), Description: line("13.10.2026", 25, false)}}}
		if f.badRole {
			proposal.Header.DebitTurnover = line("БИК", 0, true)
			proposal.Header.Labels["debitTurnover"] = ids("БИК")
		}
		used := map[string]bool{}
		raw, _ := json.Marshal(proposal)
		var visit func(any)
		visit = func(v any) {
			switch x := v.(type) {
			case string:
				used[x] = true
			case []any:
				for _, a := range x {
					visit(a)
				}
			case map[string]any:
				for _, a := range x {
					visit(a)
				}
			}
		}
		var v any
		_ = json.Unmarshal(raw, &v)
		visit(v)
		service := statementdata.Selection{}
		for _, s := range c.Spans {
			if !used[s.ID] {
				service = append(service, s.ID)
			}
		}
		proposal.Service = []statementdata.ServiceMapping{{Spans: service, Reason: "Public synthetic headings and bank identifiers"}}
		bill = &statementextract.Billing{CostMicros: 100, InputTokens: 25000, OutputTokens: 3000}
	}
	if bill != nil {
		bill.GenerationID = fmt.Sprintf("synthetic-%s-%d", f.candidate.ID, f.calls)
		bill.ComparisonRoute = comparisonTestRoute(f.candidate)
	}
	return proposal, bill, e
}
func TestComparisonRealHelperSixTuples(t *testing.T) {
	socket := os.Getenv("TEST_PDF_EXTRACTOR_SOCKET")
	if socket == "" {
		t.Skip("isolated pinned helper required")
	}
	helper, e := pdfextract.NewClient(socket)
	if e != nil {
		t.Fatal(e)
	}
	samples := []Sample{}
	for _, n := range ScreeningNames() {
		raw, e := os.ReadFile("../pdfextract/testdata/" + n + ".pdf")
		if e != nil {
			t.Fatal(e)
		}
		doc, e := helper.Extract(context.Background(), raw)
		if e != nil {
			t.Fatal(e)
		}
		samples = append(samples, Sample{n, doc, FixtureHash(n)})
	}
	for _, mode := range []string{"complete", "description_gap", "identifier_role"} {
		t.Run(mode, func(t *testing.T) {
			cs := statementextract.ComparisonCandidates()
			a, b := &comparisonFixtureReasoner{candidate: cs[0]}, &comparisonFixtureReasoner{candidate: cs[1]}
			a.omitSeparator = mode == "description_gap"
			b.badRole = mode == "identifier_role"
			m := &fakeMeter{}
			r, e := RunComparison(context.Background(), samples, [2]statementextract.Reasoner{a, b}, m)
			if a.calls != 3 || b.calls != 3 || m.reserves != 1 || m.reserved != 208500 || m.settles != 1 || m.actual == nil || *m.actual != 600 {
				t.Fatal("six-tuple metering", e, m)
			}
			if ValidateComparisonReport(r, time.Now()) != nil {
				t.Fatal("invalid report", r, e)
			}
			if mode == "complete" {
				if e != nil || !r.Passed {
					t.Fatal("complete", r, e)
				}
				for _, tuple := range r.Tuples {
					if !tuple.Result.Passed {
						t.Fatal(tuple)
					}
					t.Logf("%s/%s %d/%d", tuple.Candidate, tuple.Evidence.Name, tuple.Result.Correct, tuple.Result.Checks)
				}
				metrics, e := ComparisonMetricsFor(r, time.Now())
				if e != nil || metrics[0].Successful != 3 || metrics[1].Successful != 3 || metrics[0].KnownCostMicros != 300 || metrics[0].CostPerSuccessDenominator != 3 {
					t.Fatal("metrics", metrics, e)
				}
				raw, _ := json.Marshal(r)
				legacy, e := DecodeReport(bytes.NewReader(raw))
				if e == nil && (ValidateReport(legacy, time.Now()) == nil || ValidateScreeningReport(legacy, time.Now()) == nil) {
					t.Fatal("comparison admitted as production")
				}
				for _, mutate := range []func(*ComparisonReport){func(r *ComparisonReport) { r.Profile.Scorer = "old" }, func(r *ComparisonReport) { r.Tuples[0].Result.Correct-- }, func(r *ComparisonReport) { r.Tuples[1].Call.Route.Proof.Candidate.Canonical = "other" }, func(r *ComparisonReport) { r.Tuples[1].RequestHash = "" }, func(r *ComparisonReport) { r.ActualMicros = nil }} {
					copyRaw, _ := json.Marshal(r)
					copyReport, e := DecodeComparisonReport(bytes.NewReader(copyRaw))
					if e != nil {
						t.Fatal(e)
					}
					mutate(&copyReport)
					copyReport.Digest = comparisonDigest(copyReport)
					if ValidateComparisonReport(copyReport, time.Now()) == nil {
						t.Fatal("tampered report accepted")
					}
				}
			} else if e == nil || r.Passed {
				t.Fatal("negative passed", r, e)
			}
		})
	}
}

func (f *comparisonFixtureReasoner) ComparisonProof(context.Context) (statementextract.ComparisonIdentityProof, error) {
	return *comparisonTestRoute(f.candidate).Proof, nil
}
