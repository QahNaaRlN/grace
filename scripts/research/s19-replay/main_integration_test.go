//go:build integration

package main

import (
	"context"
	"encoding/json"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/QahNaaRlN/grace/internal/pdfextract"
	"github.com/QahNaaRlN/grace/internal/statementdata"
	"github.com/QahNaaRlN/grace/internal/statementextract"
	"github.com/QahNaaRlN/grace/internal/statementqualification"
)

func TestS19RealHelperReplay(t *testing.T) {
	socket := os.Getenv("TEST_PDF_EXTRACTOR_SOCKET")
	if socket == "" {
		t.Skip("isolated helper required")
	}
	helper, e := pdfextract.NewClient(socket)
	if e != nil {
		t.Fatal(e)
	}
	raw, e := os.ReadFile("../../../internal/pdfextract/testdata/columns.pdf")
	if e != nil {
		t.Fatal(e)
	}
	d, e := helper.Extract(context.Background(), raw)
	if e != nil {
		t.Fatal(e)
	}
	plan, e := statementextract.BuildScreeningPlan(d, statementextract.ScreeningPolicy())
	if e != nil {
		t.Fatal(e)
	}
	ids := func(text string) statementdata.Selection {
		for _, s := range d.Spans {
			if s.Text == text {
				return statementdata.Selection{s.ID}
			}
		}
		t.Fatalf("missing public fixture token %q", text)
		return nil
	}
	line := func(anchor string, allowed string) statementdata.Selection {
		anchorID := ids(anchor)[0]
		var a statementdata.Span
		for _, s := range d.Spans {
			if s.ID == anchorID {
				a = s
			}
		}
		out := statementdata.Selection{}
		for _, s := range d.Spans {
			if s.Page == a.Page && s.Block == a.Block && s.Line == a.Line && (allowed == "" || strings.Contains(" "+allowed+" ", " "+s.Text+" ")) {
				out = append(out, s.ID)
			}
		}
		return out
	}
	for _, mode := range []string{"complete", "separator", "labels", "combined"} {
		t.Run(mode, func(t *testing.T) {
			prop := statementdata.Proposal{Rows: []statementdata.RowMapping{
				{OperationDate: ids("19.10.2026"), BookingDate: ids("20.10.2026"), AccountAmount: ids("-1234.56"), AccountCurrency: ids("RUB"), TransactionAmount: ids("-15.00"), TransactionCurrency: ids("USD"), Description: line("SHOP", "")},
				{OperationDate: ids("21.10.2026"), BookingDate: ids("22.10.2026"), AccountAmount: ids("+2000.00"), AccountCurrency: ids("RUB"), Description: line("OWN", "")},
			}, Header: statementdata.HeaderMapping{Currency: ids("RUB"), DebitTurnover: ids("1234.56"), CreditTurnover: ids("2000.00"), AvailableBalanceObserved: ids("7777.00"), Labels: map[string]statementdata.Selection{"debitTurnover": line("1234.56", "Debit turnover"), "creditTurnover": line("2000.00", "Credit turnover"), "availableBalanceObserved": line("7777.00", "Available balance")}}}
			if mode == "separator" || mode == "combined" {
				kept := statementdata.Selection{}
				for _, id := range prop.Rows[1].Description {
					separator := false
					for _, s := range d.Spans {
						if s.ID == id && s.Text == "-" {
							separator = true
						}
					}
					if !separator {
						kept = append(kept, id)
					}
				}
				prop.Rows[1].Description = kept
			}
			if mode == "labels" || mode == "combined" {
				prop.Header.Labels["availableBalanceObserved"] = nil
			}
			used := map[string]bool{}
			for _, row := range prop.Rows {
				for _, s := range []statementdata.Selection{row.OperationDate, row.BookingDate, row.AccountAmount, row.AccountCurrency, row.TransactionAmount, row.TransactionCurrency, row.Description} {
					for _, id := range s {
						used[id] = true
					}
				}
			}
			for _, s := range []statementdata.Selection{prop.Header.Currency, prop.Header.DebitTurnover, prop.Header.CreditTurnover, prop.Header.AvailableBalanceObserved} {
				for _, id := range s {
					used[id] = true
				}
			}
			for _, s := range prop.Header.Labels {
				for _, id := range s {
					used[id] = true
				}
			}
			service := statementdata.Selection{}
			for _, s := range d.Spans {
				if !used[s.ID] {
					service = append(service, s.ID)
				}
			}
			prop.Service = []statementdata.ServiceMapping{{Spans: service, Reason: "Reviewed fixed synthetic service grouping"}}
			merged, e := statementextract.MergeProposals(plan, []statementdata.Proposal{prop})
			if e != nil {
				t.Fatal(e)
			}
			pages := []int{}
			for _, page := range d.Pages {
				pages = append(pages, page.Number)
			}
			r, e := statementdata.ValidateProposal(d, merged, pages, true, "")
			if e != nil {
				t.Fatal(e)
			}
			result := statementqualification.ScoreScreening("columns", r)
			expected := 19
			if mode == "labels" {
				expected = 18
			}
			if mode == "separator" || mode == "combined" {
				expected = 16
			}
			if result.Checks != 19 || result.Correct != expected {
				t.Fatalf("unexpected independent score: %+v", result)
			}
			p := projection{Version: "statement.s19.projection.v1", ExpectedSource: true, ReportIdentityVerified: true, ServiceReasonsValidated: true, ArtifactsUnchanged: true, Result: result, Parts: []statementdata.Proposal{prop}}
			for _, n := range statementqualification.ScreeningNames() {
				p.Corpus = append(p.Corpus, statementqualification.CaseEvidence{Name: n, FixtureHash: statementqualification.FixtureHash(n)})
			}
			p.Corpus[0].DocumentHash = d.Hash()
			p.Corpus[0].PlanHash = statementextract.Hash(plan)
			p.Corpus[0].InputBytes = statementextract.InputBytes(plan.Chunks[0])
			out, e := replay(p, d, statementqualification.FixtureHash("columns"))
			if e == nil || out.Verified {
				t.Fatal("historical v1 projection accepted under the new scorer")
			}
			p.Result.Correct++
			if _, e := replay(p, d, statementqualification.FixtureHash("columns")); e == nil {
				t.Fatal("different saved score accepted")
			}
			p.Result.Correct--
			p.Corpus[0].DocumentHash = strings.Repeat("0", 64)
			if _, e := replay(p, d, statementqualification.FixtureHash("columns")); e == nil {
				t.Fatal("different source accepted")
			}
		})
	}
}

// The actual S1.8 response survives as locator-only synthetic evidence.
func TestS19RetainedReplay(t *testing.T) {
	socket := os.Getenv("TEST_PDF_EXTRACTOR_SOCKET")
	if socket == "" {
		t.Skip("isolated helper required")
	}
	raw, e := os.ReadFile("../../../docs/research/statement-quality/s18-saved-trace-projection.json")
	if e != nil {
		t.Fatal(e)
	}
	var p projection
	if json.Unmarshal(raw, &p) != nil {
		t.Fatal("invalid retained projection")
	}
	helper, e := pdfextract.NewClient(socket)
	if e != nil {
		t.Fatal(e)
	}
	pdf, e := os.ReadFile("../../../internal/pdfextract/testdata/columns.pdf")
	if e != nil {
		t.Fatal(e)
	}
	d, e := helper.Extract(context.Background(), pdf)
	if e != nil {
		t.Fatal(e)
	}
	out, e := replay(p, d, statementqualification.FixtureHash("columns"))
	if e == nil || out.Verified {
		t.Fatal("historical v1 got current PASS")
	}
	historical := []string{"openingBookBalance:ambiguous_money", "closingBookBalance:ambiguous_money"}
	if p.Result.Correct != 17 || !reflect.DeepEqual(p.Result.Validation, historical) {
		t.Fatal("historical result changed")
	}
	merged, e := statementextract.MergeProposals(mustPlan(t, d), p.Parts)
	if e != nil {
		t.Fatal(e)
	}
	pages := []int{}
	for _, page := range d.Pages {
		pages = append(pages, page.Number)
	}
	review, e := statementdata.ValidateProposal(d, merged, pages, true, "")
	if e != nil {
		t.Fatal(e)
	}
	current := statementqualification.ScoreScreening("columns", review)
	expected := append(append([]string{}, historical...), "row_2:description_evidence_gap")
	if current.Correct != 16 || !reflect.DeepEqual(current.Validation, expected) || review.Rows[1].Description != "OWN TRANSFER REVIEW SEMANTICS" {
		t.Fatal("current source guard missed retained failure", current)
	}
	plan, e := statementextract.BuildScreeningPlan(d, statementextract.ScreeningPolicy())
	if e != nil {
		t.Fatal(e)
	}
	for _, id := range []string{"p1-b1-l1-w3", "p2-b1-l1-w3", "p2-b8-l1-w3"} {
		found := false
		for _, span := range d.Spans {
			if span.ID == id {
				found = true
				if span.Text != "-" {
					t.Fatal("different source separator")
				}
			}
		}
		if !found {
			t.Fatal("missing source separator")
		}
		delivered := false
		for _, span := range plan.Chunks[0].Spans {
			if plan.Aliases[span.ID] == id {
				delivered = true
				if span.Text != "-" || span.LexicalClass != statementextract.ClassNumericFragment {
					t.Fatal("different delivered sign class")
				}
			}
		}
		if !delivered {
			t.Fatal("separator not delivered")
		}
	}
	if !reflect.DeepEqual(p.Parts[0].Header.OpeningBookBalance, statementdata.Selection{"p1-b1-l1-w3"}) || !reflect.DeepEqual(p.Parts[0].Header.ClosingBookBalance, statementdata.Selection{"p2-b1-l1-w3"}) {
		t.Fatal("changed book-balance selection")
	}
	inService := false
	for _, g := range p.Parts[0].Service {
		for _, id := range g.Spans {
			if id == "p2-b8-l1-w3" {
				inService = true
			}
		}
	}
	if !inService {
		t.Fatal("missing separator not assigned to service")
	}
}

func mustPlan(t *testing.T, d statementdata.Document) statementextract.Plan {
	t.Helper()
	p, e := statementextract.BuildScreeningPlan(d, statementextract.ScreeningPolicy())
	if e != nil {
		t.Fatal(e)
	}
	return p
}
