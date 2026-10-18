package statementqualification

import (
	"context"
	"errors"
	"testing"

	"github.com/QahNaaRlN/grace/internal/statementdata"
	"github.com/QahNaaRlN/grace/internal/statementextract"
)

type fakeReasoner struct {
	calls   int
	billing *statementextract.Billing
}

func (f *fakeReasoner) CurrentPolicy(context.Context) (statementextract.Policy, error) {
	return statementextract.Policy{Model: "google/gemini-2.5-flash", Provider: "google-vertex/eu", ProviderName: "Google", InputMicrosPerMillion: 300000, OutputMicrosPerMillion: 2500000}, nil
}
func (f *fakeReasoner) Complete(context.Context, statementextract.Policy, statementextract.Chunk) (statementdata.Proposal, *statementextract.Billing, error) {
	f.calls++
	return statementdata.Proposal{}, f.billing, errors.New("malformed")
}

type fakeMeter struct {
	reserves, settles int
	actual            *int64
}

func (f *fakeMeter) Reserve(context.Context, int64, statementextract.Policy) (string, error) {
	f.reserves++
	return "reservation", nil
}
func (f *fakeMeter) Settle(_ context.Context, _ string, n *int64) error {
	f.settles++
	f.actual = n
	return nil
}
func TestQualificationBillsMalformedAndNeverRetries(t *testing.T) {
	d := statementdata.Document{Extractor: statementdata.ExtractorVersion, Pages: []statementdata.Page{{Number: 1, Width: 600, Height: 800}}, Spans: []statementdata.Span{{ID: "s1", Page: 1, Block: 1, Line: 1, Word: 1, Text: "19.10.2026", Box: statementdata.Box{XMin: 1, YMin: 1, XMax: 100, YMax: 10}}}}
	for _, known := range []bool{false, true} {
		t.Run(map[bool]string{false: "unknown_freezes", true: "known_billed"}[known], func(t *testing.T) {
			reasoner := &fakeReasoner{}
			if known {
				reasoner.billing = &statementextract.Billing{CostMicros: 42, GenerationID: "synthetic-generation", InputTokens: 20, OutputTokens: 10}
			}
			meter := &fakeMeter{}
			report, e := Run(context.Background(), []Sample{{"columns", d}, {"cards", d}}, reasoner, meter)
			if e == nil || report.Passed || reasoner.calls != 1 || meter.reserves != 1 || meter.settles != 1 {
				t.Fatal("retry or lost settlement", e, report, meter)
			}
			if known && (meter.actual == nil || *meter.actual != 42) || !known && meter.actual != nil {
				t.Fatal("billing state wrong", meter)
			}
		})
	}
}
func TestScoreCatchesLostClockAndAvailableAsClosing(t *testing.T) {
	makeReview := func(name string) statementdata.Review {
		op1, op2 := "2026-10-19", "2026-10-21"
		book1, book2 := "2026-10-20", "2026-10-22"
		if name == "cards" {
			op1 += "T10:30:00"
			op2 += "T12:15:00"
			book1 += "T11:00:00"
			book2 += "T13:20:00"
		}
		r := statementdata.Review{Rows: []statementdata.Row{{AccountAmount: &statementdata.Money{AmountMinor: 123456, Currency: "RUB"}, Direction: "debit", OperationDate: &statementdata.LocalDate{Local: op1}, BookingDate: &statementdata.LocalDate{Local: book1}, TransactionAmount: &statementdata.Money{AmountMinor: 1500, Currency: "USD"}}, {AccountAmount: &statementdata.Money{AmountMinor: 200000, Currency: "RUB"}, Direction: "credit", OperationDate: &statementdata.LocalDate{Local: op2}, BookingDate: &statementdata.LocalDate{Local: book2}}}, Reconciliation: statementdata.Reconciliation{Turnover: statementdata.Check{Status: "matched"}, RowCoverage: statementdata.Check{Status: "matched"}, BookBalance: statementdata.Check{Status: "not_provided"}}}
		if name == "columns" {
			r.Header.AvailableBalanceObserved = &statementdata.Money{AmountMinor: 777700, Currency: "RUB"}
		} else {
			r.Header.OpeningBookBalance = &statementdata.Money{AmountMinor: 1000000, Currency: "RUB"}
			r.Header.ClosingBookBalance = &statementdata.Money{AmountMinor: 1076544, Currency: "RUB"}
			r.Reconciliation.BookBalance.Status = "matched"
		}
		return r
	}
	for _, name := range []string{"columns", "cards"} {
		r := makeReview(name)
		if !Score(name, r).Passed {
			t.Fatal("valid synthetic rejected", name, Score(name, r))
		}
		r.Rows[0].OperationDate.Local = "2026-10-19"
		if name == "cards" && Score(name, r).Passed {
			t.Fatal("lost clock passed")
		}
	}
	r := makeReview("columns")
	r.Header.ClosingBookBalance = r.Header.AvailableBalanceObserved
	if Score("columns", r).Passed {
		t.Fatal("available became closing")
	}
}
