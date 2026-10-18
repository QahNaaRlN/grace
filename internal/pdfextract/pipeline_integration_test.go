//go:build integration

package pdfextract

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/QahNaaRlN/grace/internal/statementdata"
)

func TestRealPDFLayoutPipeline(t *testing.T) {
	socket := os.Getenv("TEST_PDF_EXTRACTOR_SOCKET")
	if socket == "" {
		t.Skip("isolated helper socket required")
	}
	client, e := NewClient(socket)
	if e != nil {
		t.Fatal(e)
	}
	for _, name := range []string{"columns", "cards"} {
		t.Run(name, func(t *testing.T) {
			raw, e := os.ReadFile("testdata/" + name + ".pdf")
			if e != nil {
				t.Fatal(e)
			}
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			d, e := client.Extract(ctx, raw)
			if e != nil {
				t.Fatal(e)
			}
			again, e := client.Extract(ctx, raw)
			if e != nil || d.Hash() != again.Hash() {
				t.Fatal("unstable extraction", e)
			}
			if name == "columns" && len(d.Pages) != 2 || name == "cards" && len(d.Pages) != 1 {
				t.Fatal("lost page")
			}
			find := func(literal string) statementdata.Selection {
				for _, s := range d.Spans {
					if s.Text == literal {
						return statementdata.Selection{s.ID}
					}
				}
				t.Fatalf("missing %q", literal)
				return nil
			}
			selectText := func(literal string) statementdata.Selection {
				parts := strings.Fields(literal)
				for i := range d.Spans {
					if d.Spans[i].Text != parts[0] {
						continue
					}
					ids := statementdata.Selection{}
					for j, part := range parts {
						if i+j >= len(d.Spans) || d.Spans[i+j].Text != part {
							break
						}
						ids = append(ids, d.Spans[i+j].ID)
					}
					if len(ids) == len(parts) {
						return ids
					}
				}
				t.Fatalf("missing phrase %q", literal)
				return nil
			}
			currency := find("RUB")
			p := statementdata.Proposal{Rows: []statementdata.RowMapping{
				{OperationDate: find("19.10.2026"), BookingDate: find("20.10.2026"), AccountAmount: find("-1234.56"), AccountCurrency: currency, TransactionAmount: find("-15.00"), TransactionCurrency: find("USD"), Description: selectText("SYNTHETIC SHOP")},
				{OperationDate: find("21.10.2026"), BookingDate: find("22.10.2026"), AccountAmount: find("+2000.00"), AccountCurrency: currency},
			}, Header: statementdata.HeaderMapping{DebitTurnover: find("1234.56"), CreditTurnover: find("2000.00"), Currency: currency, Labels: map[string]statementdata.Selection{"debitTurnover": selectText("Debit turnover"), "creditTurnover": selectText("Credit turnover")}}}
			if name == "columns" {
				p.Header.AvailableBalanceObserved = find("7777.00")
				p.Header.Labels["availableBalanceObserved"] = selectText("Available balance")
			} else {
				p.Header.OpeningBookDate = find("18.10.2026")
				p.Header.ClosingBookDate = find("23.10.2026")
				p.Header.OpeningBookBalance = find("10000.00")
				p.Header.ClosingBookBalance = find("10765.44")
				p.Header.Labels["openingBookBalance"] = selectText("Opening book balance")
				p.Header.Labels["closingBookBalance"] = selectText("Closing book balance")
			}
			claimed := map[string]bool{}
			for _, r := range p.Rows {
				for _, ids := range []statementdata.Selection{r.OperationDate, r.BookingDate, r.AccountAmount, r.AccountCurrency, r.TransactionAmount, r.TransactionCurrency, r.Description} {
					for _, id := range ids {
						claimed[id] = true
					}
				}
			}
			for _, ids := range []statementdata.Selection{p.Header.DebitTurnover, p.Header.CreditTurnover, p.Header.Currency, p.Header.AvailableBalanceObserved, p.Header.OpeningBookBalance, p.Header.ClosingBookBalance, p.Header.OpeningBookDate, p.Header.ClosingBookDate} {
				for _, id := range ids {
					claimed[id] = true
				}
			}
			for _, ids := range p.Header.Labels {
				for _, id := range ids {
					claimed[id] = true
				}
			}
			service := statementdata.Selection{}
			for _, s := range d.Spans {
				if !claimed[s.ID] {
					service = append(service, s.ID)
				}
			}
			p.Service = []statementdata.ServiceMapping{{Spans: service, Reason: "Synthetic ground truth: repeated headings and intentionally untrusted source text"}}
			pages := []int{1}
			if name == "columns" {
				pages = append(pages, 2)
			}
			review, e := statementdata.ValidateProposal(d, p, pages, true, "")
			if e != nil || len(review.Reconciliation.Blockers) > 0 {
				t.Fatal(review.Reconciliation, e)
			}
			if review.Reconciliation.Turnover.Status != "matched" || review.Rows[0].AccountAmount.AmountMinor != 123456 || review.Rows[0].TransactionAmount.AmountMinor != 1500 {
				t.Fatal(review)
			}
			if name == "columns" && review.Reconciliation.BookBalance.Status != "not_provided" {
				t.Fatal("available used as closing")
			}
			if name == "cards" && review.Reconciliation.BookBalance.Status != "matched" {
				t.Fatal(review.Reconciliation)
			}
			// A schema-valid omission remains a blocker despite preserved printed totals.
			p.Rows = p.Rows[:1]
			review, e = statementdata.ValidateProposal(d, p, pages, true, "")
			if e != nil || len(review.Reconciliation.Blockers) == 0 {
				t.Fatal("omitted row accepted", e)
			}
		})
	}
	if _, e = client.Extract(context.Background(), []byte("%PDF-corrupt")); e == nil {
		t.Fatal("malformed PDF accepted")
	}
}
