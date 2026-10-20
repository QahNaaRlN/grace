package statementqualification

import (
	"encoding/json"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/QahNaaRlN/grace/internal/statementdata"
	"github.com/QahNaaRlN/grace/internal/statementextract"
)

type fixtureSelector struct {
	Page       int
	Line, Text string
}
type turnoverGolden struct {
	Rows                 []map[string][]fixtureSelector
	Header, Labels       map[string][]fixtureSelector
	Service              []fixtureSelector
	ExpectedMinor        []int64
	ExpectedDescriptions []string
}

func fixtureSelection(t *testing.T, d statementdata.Document, selectors []fixtureSelector) statementdata.Selection {
	t.Helper()
	out := statementdata.Selection{}
	for _, selection := range selectors {
		found := 0
		for i, s := range d.Spans {
			if s.Page != selection.Page || s.Word != 1 {
				continue
			}
			line := []statementdata.Span{s}
			for _, next := range d.Spans[i+1:] {
				if next.Page != s.Page || next.Block != s.Block || next.Line != s.Line {
					break
				}
				line = append(line, next)
			}
			words := []string{}
			for _, span := range line {
				words = append(words, span.Text)
			}
			if strings.Join(words, " ") != selection.Line {
				continue
			}
			target := strings.Fields(selection.Text)
			for j := 0; j+len(target) <= len(words); j++ {
				if !reflect.DeepEqual(words[j:j+len(target)], target) {
					continue
				}
				found++
				for _, span := range line[j : j+len(target)] {
					out = append(out, span.ID)
				}
			}
		}
		if found != 1 {
			t.Fatalf("golden selector matched %d times: %+v", found, selection)
		}
	}
	return out
}
func turnoverFixture(t *testing.T, d statementdata.Document) (statementdata.Proposal, turnoverGolden) {
	t.Helper()
	raw, e := os.ReadFile("../pdfextract/testdata/turnover-roles-ru.golden.json")
	if e != nil {
		t.Fatal(e)
	}
	var g turnoverGolden
	if e = json.Unmarshal(raw, &g); e != nil {
		t.Fatal(e)
	}
	var p statementdata.Proposal
	for _, row := range g.Rows {
		p.Rows = append(p.Rows, statementdata.RowMapping{OperationDate: fixtureSelection(t, d, row["operationDate"]), AccountAmount: fixtureSelection(t, d, row["accountAmount"]), AccountCurrency: fixtureSelection(t, d, row["accountCurrency"]), Description: fixtureSelection(t, d, row["description"])})
	}
	p.Header = statementdata.HeaderMapping{Currency: fixtureSelection(t, d, g.Header["currency"]), AvailableBalanceObserved: fixtureSelection(t, d, g.Header["availableBalanceObserved"]), DebitTurnover: fixtureSelection(t, d, g.Header["debitTurnover"]), CreditTurnover: fixtureSelection(t, d, g.Header["creditTurnover"]), Labels: map[string]statementdata.Selection{}}
	for role, selectors := range g.Labels {
		p.Header.Labels[role] = fixtureSelection(t, d, selectors)
	}
	// Explicit service selectors from the independently authored fixture source,
	// not an automatic complement of model selections.
	p.Service = []statementdata.ServiceMapping{{Spans: fixtureSelection(t, d, g.Service), Reason: "Explicit public fixture titles, headings, repeated currency and bank identifiers"}}
	return p, g
}
func assertTurnoverFixture(t *testing.T, d statementdata.Document) {
	t.Helper()
	p, g := turnoverFixture(t, d)
	review, e := statementdata.ValidateProposal(d, p, []int{1, 2}, true, "")
	if e != nil || len(review.Reconciliation.Blockers) > 0 || review.Reconciliation.Turnover.Status != "matched" || review.Reconciliation.RowCoverage.Status != "matched" || len(review.Rows) != 2 {
		t.Fatal(review, e)
	}
	for i, row := range review.Rows {
		if row.AccountAmount.AmountMinor != g.ExpectedMinor[i] || row.AccountAmount.Currency != "RUB" || row.Description != g.ExpectedDescriptions[i] {
			t.Fatal("golden rows changed", review.Rows)
		}
	}
	if review.Header.DebitTurnover.AmountMinor != 248600 || review.Header.CreditTurnover.AmountMinor != 310500 || review.Header.OpeningBookBalance != nil || review.Header.ClosingBookBalance != nil {
		t.Fatal("golden header", review.Header)
	}
	for _, mode := range []string{"identifier_label", "same_number_identifier", "borrowed_label", "reverse_label", "unknown_label", "missing_label", "other_page_label"} {
		t.Run(mode, func(t *testing.T) {
			doc := d
			doc.Spans = append([]statementdata.Span{}, d.Spans...)
			bad, _ := turnoverFixture(t, doc)
			bic := fixtureSelection(t, doc, []fixtureSelector{{2, "БИК 044000999", "044000999"}})
			bicLabel := fixtureSelection(t, doc, []fixtureSelector{{2, "БИК 044000999", "БИК"}})
			switch mode {
			case "identifier_label", "same_number_identifier":
				bad.Header.DebitTurnover = bic
				bad.Header.Labels["debitTurnover"] = bicLabel
			case "borrowed_label":
				bad.Header.DebitTurnover = bic
			case "reverse_label":
				bad.Header.Labels["debitTurnover"] = bad.Header.Labels["creditTurnover"]
			case "missing_label":
				bad.Header.Labels["debitTurnover"] = nil
			case "other_page_label":
				bad.Header.Labels["debitTurnover"] = fixtureSelection(t, doc, []fixtureSelector{{1, "Страница 1", "Страница"}})
			}
			for i := range doc.Spans {
				span := &doc.Spans[i]
				if mode == "same_number_identifier" && span.ID == bic[0] {
					span.Text = "2486"
				}
				if mode == "unknown_label" && span.ID == bad.Header.Labels["debitTurnover"][0] {
					span.Text = "Неизвестное:"
				}
				if mode == "other_page_label" && span.ID == bad.Header.Labels["debitTurnover"][0] {
					span.Text = "Расходы:"
				}
			}

			// Fully cover the deliberately wrong answer: move displaced header
			// fragments to service and remove the newly selected service IDs.
			// This ensures coverage failure cannot stand in for role detection.
			selected := map[string]bool{}
			for _, ids := range []statementdata.Selection{bad.Header.DebitTurnover, bad.Header.CreditTurnover, bad.Header.Labels["debitTurnover"], bad.Header.Labels["creditTurnover"]} {
				for _, id := range ids {
					selected[id] = true
				}
			}
			service := statementdata.Selection{}
			for _, id := range bad.Service[0].Spans {
				if !selected[id] {
					service = append(service, id)
				}
			}
			for _, ids := range []statementdata.Selection{p.Header.DebitTurnover, p.Header.CreditTurnover, p.Header.Labels["debitTurnover"], p.Header.Labels["creditTurnover"]} {
				for _, id := range ids {
					if !selected[id] {
						service = append(service, id)
					}
				}
			}
			bad.Service[0].Spans = service
			r, e := statementdata.ValidateProposal(doc, bad, []int{1, 2}, true, "")
			if e != nil || r.Reconciliation.Turnover.Status != "incomplete" || !strings.Contains(strings.Join(r.Reconciliation.Blockers, "|"), "debitTurnover:turnover_role_unverified") {
				t.Fatal("wrong role accepted", r, e)
			}
			if r.Reconciliation.RowCoverage.Status != "matched" {
				t.Fatal("negative lost coverage", r.Reconciliation)
			}
			if mode == "same_number_identifier" && (r.Header.DebitTurnover == nil || r.Header.DebitTurnover.AmountMinor != 248600 || strings.Contains(strings.Join(r.Reconciliation.Blockers, "|"), "printed_turnover_mismatch")) {
				t.Fatal("same-number negative did not preserve value", r.Header)
			}
		})
	}
	// A delivered-alias fake passes through the real planner and canonicalizer.
	plan, e := statementextract.BuildPlan(d, statementextract.Policy{Model: "google/gemini-2.5-flash", Provider: "google-vertex/eu", ProviderName: "Google", InputMicrosPerMillion: 300000, OutputMicrosPerMillion: 2500000}, nil)
	if e != nil {
		t.Fatal(e)
	}
	raw, _ := json.Marshal(p)
	for alias, canonical := range plan.Aliases {
		raw = []byte(strings.ReplaceAll(string(raw), `"`+canonical+`"`, `"`+alias+`"`))
	}
	var wire statementdata.Proposal
	if json.Unmarshal(raw, &wire) != nil {
		t.Fatal("wire")
	}
	canonical, e := statementextract.CanonicalProposal(plan, wire)
	if e != nil {
		t.Fatal(e)
	}
	r, e := statementdata.ValidateProposal(d, canonical, []int{1, 2}, true, "")
	if e != nil || len(r.Reconciliation.Blockers) > 0 || r.Reconciliation.Turnover.Status != "matched" {
		t.Fatal("canonical fake failed", r, e)
	}
	t.Logf("RU fixture spans=%d bytes=%d aliases=%d providerCalls=0", len(d.Spans), statementextract.InputBytes(plan.Chunks[0]), len(plan.Aliases))
}
func TestTurnoverRolesPublicLayout(t *testing.T) {
	f, e := os.Open("../pdfextract/testdata/turnover-roles-ru.xml")
	if e != nil {
		t.Fatal(e)
	}
	defer f.Close()
	d, e := statementdata.DecodeLayout(f)
	if e != nil {
		t.Fatal(e)
	}
	assertTurnoverFixture(t, d)
}
