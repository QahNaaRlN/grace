//go:build integration

package statementqualification

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"testing"
	"time"

	"github.com/QahNaaRlN/grace/internal/pdfextract"
	"github.com/QahNaaRlN/grace/internal/statementdata"
	"github.com/QahNaaRlN/grace/internal/statementextract"
)

type activationFixtureReasoner struct {
	calls   int
	omit    bool
	wrongFX bool
}

func (f *activationFixtureReasoner) CurrentPolicy(ctx context.Context) (statementextract.Policy, error) {
	return (&fakeReasoner{}).CurrentPolicy(ctx)
}
func (f *activationFixtureReasoner) Complete(ctx context.Context, _ statementextract.Policy, c statementextract.Chunk) (statementdata.Proposal, *statementextract.Billing, error) {
	f.calls++
	if c.Index != 1 || len(c.Context) != 0 {
		return statementdata.Proposal{}, nil, fmt.Errorf("split fixture")
	}
	// Decisions use only the delivered tuples and the printed public fixture
	// structure. No canonical map or original layout is available to the fake.
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
	line := func(anchor string, delta int, words map[string]bool) statementdata.Selection {
		a := find(anchor)
		out := statementdata.Selection{}
		for _, s := range c.Spans {
			if s.Page == a.Page && math.Abs(float64(s.Y-a.Y-delta)) <= 1 && (words == nil || words[s.Text]) {
				out = append(out, s.ID)
			}
		}
		return out
	}
	cards := find("10000.00").ID != ""
	p := statementdata.Proposal{Rows: []statementdata.RowMapping{
		{OperationDate: ids("19.10.2026"), BookingDate: ids("20.10.2026"), AccountAmount: ids("-1234.56"), AccountCurrency: ids("RUB"), TransactionAmount: ids("-15.00"), TransactionCurrency: ids("USD"), Description: line("19.10.2026", 20, nil)},
		{OperationDate: ids("21.10.2026"), BookingDate: ids("22.10.2026"), AccountAmount: ids("+2000.00"), AccountCurrency: ids("RUB"), Description: line("21.10.2026", 20, nil)},
	}, Header: statementdata.HeaderMapping{Currency: ids("RUB"), DebitTurnover: ids("1234.56"), CreditTurnover: ids("2000.00"), Labels: map[string]statementdata.Selection{
		"debitTurnover": line("1234.56", 0, map[string]bool{"Debit": true, "turnover": true}), "creditTurnover": line("2000.00", 0, map[string]bool{"Credit": true, "turnover": true}),
	}}}
	if cards {
		p.Rows[0].OperationDate = ids("19.10.2026", "10:30")
		p.Rows[0].BookingDate = ids("20.10.2026", "11:00")
		p.Rows[1].OperationDate = ids("21.10.2026", "12:15")
		p.Rows[1].BookingDate = ids("22.10.2026", "13:20")
		p.Rows[0].Description = line("19.10.2026", 80, nil)
		p.Rows[1].Description = line("21.10.2026", 60, nil)
		p.Header.OpeningBookBalance = ids("10000.00")
		p.Header.ClosingBookBalance = ids("10765.44")
		p.Header.OpeningBookDate = ids("18.10.2026")
		p.Header.ClosingBookDate = ids("23.10.2026")
		p.Header.Labels["openingBookBalance"] = line("10000.00", 0, map[string]bool{"Opening": true, "book": true, "balance": true})
		p.Header.Labels["closingBookBalance"] = line("10765.44", 0, map[string]bool{"Closing": true, "book": true, "balance": true})
	} else {
		p.Header.AvailableBalanceObserved = ids("7777.00")
		p.Header.Labels["availableBalanceObserved"] = line("7777.00", 0, map[string]bool{"Available": true, "balance": true})
	}
	used := map[string]bool{}
	for _, r := range p.Rows {
		for _, sel := range []statementdata.Selection{r.OperationDate, r.BookingDate, r.AccountAmount, r.AccountCurrency, r.TransactionAmount, r.TransactionCurrency, r.Description} {
			for _, id := range sel {
				if id == "" {
					return p, nil, fmt.Errorf("missing wire field")
				}
				used[id] = true
			}
		}
	}
	for _, sel := range []statementdata.Selection{p.Header.OpeningBookBalance, p.Header.ClosingBookBalance, p.Header.OpeningBookDate, p.Header.ClosingBookDate, p.Header.AvailableBalanceObserved, p.Header.DebitTurnover, p.Header.CreditTurnover, p.Header.Currency} {
		for _, id := range sel {
			used[id] = true
		}
	}
	for _, sel := range p.Header.Labels {
		for _, id := range sel {
			used[id] = true
		}
	}
	service := statementdata.Selection{}
	for _, s := range c.Spans {
		if !used[s.ID] {
			service = append(service, s.ID)
		}
	}
	if f.omit && len(service) > 0 {
		service = service[1:]
	}
	p.Service = []statementdata.ServiceMapping{{Spans: service, Reason: "Reviewed public synthetic headings and unused currency labels"}}
	if f.wrongFX {
		p.Rows[0].TransactionCurrency = ids("RUB")
	}
	return p, &statementextract.Billing{GenerationID: fmt.Sprintf("synthetic-%d", f.calls), CostMicros: 100, InputTokens: 25000, OutputTokens: 3000}, nil
}
func TestActivationRealHelperSmoke(t *testing.T) {
	socket := os.Getenv("TEST_PDF_EXTRACTOR_SOCKET")
	if socket == "" {
		t.Skip("isolated real helper required")
	}
	helper, e := pdfextract.NewClient(socket)
	if e != nil {
		t.Fatal(e)
	}
	samples := []Sample{}
	for _, name := range []string{"columns", "cards"} {
		raw, e := os.ReadFile("../pdfextract/testdata/" + name + ".pdf")
		if e != nil {
			t.Fatal(e)
		}
		hash := sha256.Sum256(raw)
		d, e := helper.Extract(context.Background(), raw)
		if e != nil {
			t.Fatal(e)
		}
		samples = append(samples, Sample{name, d, hex.EncodeToString(hash[:])})
	}
	for _, mode := range []string{"complete", "missing_coverage", "wrong_fx"} {
		t.Run(mode, func(t *testing.T) {
			fake := &activationFixtureReasoner{omit: mode == "missing_coverage", wrongFX: mode == "wrong_fx"}
			meter := &fakeMeter{}
			report, e := Run(context.Background(), samples, fake, meter)
			if mode == "complete" {
				if e != nil || !report.Passed || fake.calls != 2 || meter.reserved != 140000 || meter.actual == nil || *meter.actual != 200 {
					t.Fatal(e, report, meter)
				}
				raw, _ := json.Marshal(report)
				decoded, e := DecodeReport(bytes.NewReader(raw))
				if e != nil || ValidateReport(decoded, time.Now()) != nil {
					t.Fatal("current real report cannot activate", e, report.Results)
				}
			} else {
				if e == nil || report.Passed || fake.calls != 1 || meter.actual == nil || *meter.actual != 100 {
					t.Fatal("negative accepted or retry/lost bill", e, report, meter)
				}
			}
		})
	}
}
