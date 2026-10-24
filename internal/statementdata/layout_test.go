package statementdata

import (
	"fmt"
	"strings"
	"testing"
)

func xmlLayout(words string) string {
	return `<?xml version="1.0"?><html><body><doc><page width="600" height="800"><flow><block><line>` + words + `</line></block></flow></page></doc></body></html>`
}
func TestLayoutGrounding(t *testing.T) {
	raw := xmlLayout(`<word xMin="1" yMin="2" xMax="10" yMax="12">-1</word><word xMin="11" yMin="2" xMax="25" yMax="12">234,56</word>`)
	d, e := DecodeLayout(strings.NewReader(raw))
	if e != nil {
		t.Fatal(e)
	}
	d2, e := DecodeLayout(strings.NewReader(raw))
	if e != nil || d.Hash() != d2.Hash() {
		t.Fatal("unstable locators")
	}
	ev, e := d.Ground("amount", Selection{d.Spans[0].ID, d.Spans[1].ID}, true)
	if e != nil || ev.Literal != "-1 234,56" {
		t.Fatal(ev, e)
	}
	for _, ids := range []Selection{{"fabricated"}, {d.Spans[1].ID, d.Spans[0].ID}, {d.Spans[0].ID, d.Spans[0].ID}} {
		if _, e := d.Ground("amount", ids, true); e == nil {
			t.Fatal("ungrounded evidence", ids)
		}
	}
	d.Spans[1].Line++
	if _, e := d.Ground("amount", Selection{d.Spans[0].ID, d.Spans[1].ID}, true); e == nil {
		t.Fatal("cross-line amount")
	}
}
func TestLayoutLimitsAndUntrustedXML(t *testing.T) {
	for _, raw := range []string{xmlLayout(`<word xMin="NaN" yMin="2" xMax="10" yMax="12">1</word>`), xmlLayout(`<word xMin="1" yMin="2" xMax="601" yMax="12">1</word>`), xmlLayout(``), `<page width="600" height="800">`, `<!DOCTYPE html [<!ENTITY x SYSTEM "file:///etc/passwd">]>` + xmlLayout(`<word xMin="1" yMin="2" xMax="10" yMax="12">&x;</word>`)} {
		if _, e := DecodeLayout(strings.NewReader(raw)); e == nil {
			t.Fatal("accepted invalid XML")
		}
	}
	pages := "<doc>"
	for i := 0; i < MaxPages+1; i++ {
		pages += `<page width="600" height="800"><block><line><word xMin="1" yMin="2" xMax="10" yMax="12">1</word></line></block></page>`
	}
	pages += "</doc>"
	if _, e := DecodeLayout(strings.NewReader(pages)); e == nil {
		t.Fatal("accepted too many pages")
	}
}
func syntheticDocument(values ...string) Document {
	d := Document{Extractor: ExtractorVersion, Pages: []Page{{1, 600, 800}}, Spans: []Span{}}
	for i, v := range values {
		d.Spans = append(d.Spans, Span{ID: fmt.Sprintf("s%d", i), Page: 1, Block: i + 1, Line: 1, Word: 1, Box: Box{1, float64(i * 15), 100, float64(i*15 + 10)}, Text: v})
	}
	return d
}
func contains(ss []string, part string) bool {
	for _, s := range ss {
		if strings.Contains(s, part) {
			return true
		}
	}
	return false
}
func TestProposalCoverageRolesAndAmounts(t *testing.T) {
	d := syntheticDocument("19.10.2026 10:30", "20.10.2026 11:00", "-1 234,56", "RUB", "USD", "-15.00", "Synthetic merchant", "Available balance", "7 777.00", "Debit turnover", "1234.56", "Credit turnover", "0.00")
	p := Proposal{Rows: []RowMapping{{OperationDate: Selection{"s0"}, BookingDate: Selection{"s1"}, AccountAmount: Selection{"s2"}, AccountCurrency: Selection{"s3"}, TransactionCurrency: Selection{"s4"}, TransactionAmount: Selection{"s5"}, Description: Selection{"s6"}}}, Header: HeaderMapping{AvailableBalanceObserved: Selection{"s8"}, DebitTurnover: Selection{"s10"}, CreditTurnover: Selection{"s12"}, Currency: Selection{"s3"}, Labels: map[string]Selection{"availableBalanceObserved": {"s7"}, "debitTurnover": {"s9"}, "creditTurnover": {"s11"}}}}
	r, e := ValidateProposal(d, p, []int{1}, true, "")
	if e != nil || len(r.Reconciliation.Blockers) > 0 {
		t.Fatal(r, e)
	}
	if r.Rows[0].AccountAmount.AmountMinor != 123456 || r.Rows[0].TransactionAmount.AmountMinor != 1500 || r.Rows[0].BookingDate.Local != "2026-10-20T11:00:00" || r.Rows[0].OperationDate.UTC != "" || r.Header.ClosingBookBalance != nil {
		t.Fatal(r)
	}
	r, e = ValidateProposal(d, p, nil, false, "")
	if e != nil || !contains(r.Reconciliation.Blockers, "owner_page_review_required") || !contains(r.Reconciliation.Blockers, "owner_field_role_review_required") {
		t.Fatal(r, e)
	}
	p.Rows = append(p.Rows, p.Rows[0])
	r, e = ValidateProposal(d, p, []int{1}, true, "")
	if e != nil || !contains(r.Reconciliation.Blockers, "overlapping_evidence") {
		t.Fatal(r, e)
	}
	p.Rows = p.Rows[:1]
	p.Header.ClosingBookBalance = p.Header.AvailableBalanceObserved
	p.Header.AvailableBalanceObserved = nil
	p.Header.Labels["closingBookBalance"] = Selection{"s7"}
	r, e = ValidateProposal(d, p, []int{1}, true, "")
	if e != nil || !contains(r.Reconciliation.Blockers, "available_is_not_book_balance") {
		t.Fatal(r, e)
	}
	p.Header.ClosingBookBalance = nil
	p.Rows[0].AccountAmount = Selection{"invented"}
	r, e = ValidateProposal(d, p, []int{1}, true, "")
	if e != nil || !contains(r.Reconciliation.Blockers, "invalid_span_reference") || !contains(r.Reconciliation.Blockers, "source_coverage_incomplete") {
		t.Fatal(r, e)
	}
}

func TestDateAndClockAcrossAdjacentLinesRetainBothClocks(t *testing.T) {
	d := syntheticDocument("19.10.2026", "10:30:45", "20.10.2026", "11:45:00", "-12.34", "RUB", "Synthetic merchant")
	proposal := Proposal{Rows: []RowMapping{{OperationDate: Selection{"s0", "s1"}, BookingDate: Selection{"s2", "s3"}, AccountAmount: Selection{"s4"}, AccountCurrency: Selection{"s5"}, Description: Selection{"s6"}}}}
	review, e := ValidateProposal(d, proposal, []int{1}, true, "")
	if e != nil || len(review.Reconciliation.Blockers) > 0 {
		t.Fatal(review.Reconciliation, e)
	}
	row := review.Rows[0]
	if row.OperationDate.Local != "2026-10-19T10:30:45" || row.BookingDate.Local != "2026-10-20T11:45:00" || row.OperationDate.UTC != "" {
		t.Fatal(row)
	}
	proposal.Rows[0].OperationDate = Selection{"s0", "s3"}
	review, e = ValidateProposal(d, proposal, []int{1}, true, "")
	if e != nil || !contains(review.Reconciliation.Blockers, "noncontiguous_date_evidence") {
		t.Fatal("combined unrelated clocks", review, e)
	}
}

func TestMoneyCannotCombineDifferentLines(t *testing.T) {
	d := syntheticDocument("19.10.2026", "-1", "234,56", "RUB", "Synthetic merchant")
	p := Proposal{Rows: []RowMapping{{OperationDate: Selection{"s0"}, AccountAmount: Selection{"s1", "s2"}, AccountCurrency: Selection{"s3"}, Description: Selection{"s4"}}}}
	review, e := ValidateProposal(d, p, []int{1}, true, "")
	if e != nil || !contains(review.Reconciliation.Blockers, "noncontiguous_evidence") {
		t.Fatal("money combined across separate lines", review, e)
	}
}

func TestVisualLayoutRetainsNativeSpansAlongsideScannedPage(t *testing.T) {
	raw := strings.Replace(xmlLayout(`<word xMin="1" yMin="2" xMax="10" yMax="12">source</word>`), "</doc>", `<page width="600" height="800"></page></doc>`, 1)
	if _, e := DecodeLayout(strings.NewReader(raw)); e == nil {
		t.Fatal("legacy text extractor accepted empty-text page")
	}
	d, e := DecodeVisualLayout(strings.NewReader(raw))
	if e != nil || len(d.Pages) != 2 || len(d.Spans) != 1 || d.Spans[0].Text != "source" {
		t.Fatal(d, e)
	}
	if d.Validate() == nil || d.ValidateVisualLayout() != nil {
		t.Fatal("visual relaxation leaked into native validation")
	}
}
