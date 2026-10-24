package statementdata

import (
	"strings"
	"testing"
)

func visualFixture() VisualReading {
	fields := map[string]*VisualField{}
	for _, k := range VisualRowFields {
		fields[k] = nil
	}
	header := map[string]*VisualField{}
	for _, k := range VisualHeaderFields {
		header[k] = nil
	}
	for k, v := range map[string]string{"operationDate": "25.10.2026 12:34", "accountAmount": "-12.34", "accountCurrency": "RUB", "direction": "debit", "description": "Synthetic purchase"} {
		fields[k] = &VisualField{Value: v, Evidence: []ImageQuote{{Page: 1, Quote: v, Box: [4]float64{0, 0, 1, 1}}}}
	}
	return VisualReading{Header: header, Rows: []VisualRow{{Anchor: []ImageQuote{{Page: 1, Quote: "25.10.2026", Box: [4]float64{0, 0, 1, 1}}}, Fields: fields, Issues: []string{}}}}
}
func TestVisualAmountsPrecisionAndOwnerDigests(t *testing.T) {
	v := visualFixture()
	review, e := v.Review(1)
	if e != nil {
		t.Fatal(e)
	}
	r := review.Rows[0]
	if r.AccountAmount.AmountMinor != 1234 || r.OperationDate.Precision != "local_datetime" || r.OperationDate.UTC != "" || len(r.Evidence[0].Spans) != 0 || len(review.Reconciliation.Blockers) != 0 {
		t.Fatal(review)
	}
	if review.Reconciliation.BookBalance.Status != "not_provided" || review.Reconciliation.Turnover.Status != "not_provided" {
		t.Fatal("unknown totals invented")
	}
	state := VisualState{SourceSHA256: strings.Repeat("a", 64), Candidate: v}
	before := state.RowDigest(0)
	state.Candidate.Rows[0].Fields["description"].Value += " corrected"
	if before == state.RowDigest(0) {
		t.Fatal("edit retained owner digest")
	}
	state.SourceSHA256 = strings.Repeat("b", 64)
	if before == state.RowDigest(0) {
		t.Fatal("source replacement retained owner digest")
	}
}
func TestVisualRefusesTruncationMissingCurrencyAndInvalidRegions(t *testing.T) {
	v := visualFixture()
	v.Rows[0].Fields["accountCurrency"] = nil
	r, e := v.Review(1)
	if e != nil || len(r.Reconciliation.Blockers) == 0 {
		t.Fatal("missing currency ready", r, e)
	}
	v = visualFixture()
	v.Rows[0].Fields["direction"].Value = "credit"
	r, e = v.Review(1)
	if e != nil || len(r.Reconciliation.Blockers) == 0 {
		t.Fatal("sign conflict ready")
	}
	v = visualFixture()
	v.Rows[0].Anchor[0].Box = [4]float64{0, 0, 1, 2}
	if v.Validate(1) == nil {
		t.Fatal("invalid region")
	}
	v = visualFixture()
	v.Rows[0].Anchor[0].Page = 2
	if v.Validate(1) == nil {
		t.Fatal("absent source page")
	}
	v = visualFixture()
	for len(v.Rows) <= 200 {
		v.Rows = append(v.Rows, v.Rows[0])
	}
	if v.Validate(1) == nil {
		t.Fatal("truncated oversized reading")
	}
}
