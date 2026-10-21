package statementdata

import (
	"encoding/json"
	"fmt"
	"math"
	"reflect"
	"strings"
	"testing"
)

func gapFixture(token string) (Document, Proposal) {
	d := syntheticDocument("19.10.2026", "-100.00", "RUB")
	words := []string{"OWN", "TRANSFER", token, "REVIEW", "SEMANTICS"}
	for i, text := range words {
		d.Spans = append(d.Spans, Span{ID: fmt.Sprintf("desc%d", i+1), Page: 1, Block: 4, Line: 1, Word: i + 1, Box: Box{float64(20 * i), 100, float64(20*i + 19), 110}, Text: text})
	}
	p := Proposal{Rows: []RowMapping{{OperationDate: Selection{"s0"}, AccountAmount: Selection{"s1"}, AccountCurrency: Selection{"s2"}, Description: Selection{"desc1", "desc2", "desc4", "desc5"}}}, Service: []ServiceMapping{{Spans: Selection{"desc3"}, Reason: "Synthetic selected service"}}}
	return d, p
}
func encoded(v any) string { raw, _ := json.Marshal(v); return string(raw) }
func TestDescriptionGapIsGeneralSourceGuard(t *testing.T) {
	for _, token := range []string{"-", "+", "middle", "[text_x]"} {
		for _, mapped := range []bool{true, false} {
			t.Run(token+fmt.Sprint(mapped), func(t *testing.T) {
				d, p := gapFixture(token)
				if !mapped {
					p.Service = nil
				}
				before := encoded(p)
				r, e := ValidateProposal(d, p, []int{1}, true, "")
				if e != nil {
					t.Fatal(e)
				}
				if !contains(r.Rows[0].Issues, "description_evidence_gap") || !contains(r.Reconciliation.Blockers, "row_1:description_evidence_gap") {
					t.Fatal("gap accepted", r)
				}
				if r.Rows[0].Description != "OWN TRANSFER REVIEW SEMANTICS" || r.Rows[0].AccountAmount.AmountMinor != 10000 || r.Reconciliation.DebitMinor == nil || *r.Reconciliation.DebitMinor != 10000 || before != encoded(p) {
					t.Fatal("source/value repaired", r)
				}
				if mapped && r.Reconciliation.RowCoverage.Status != "matched" {
					t.Fatal("service coverage changed")
				}
				if !mapped && !contains(r.Reconciliation.Blockers, "source_coverage_incomplete") {
					t.Fatal("existing coverage blocker lost")
				}
				once := encoded(r)
				ApplyGroundingChecks(&d, &r)
				ApplyGroundingChecks(nil, &r)
				if encoded(r) != once {
					t.Fatal("non-idempotent retained check", r)
				}
			})
		}
	}
}
func TestDescriptionSegmentsAndRetainedIntegrity(t *testing.T) {
	for _, mode := range []string{"complete", "line_wrap", "block_wrap", "page_wrap", "rename_import_text", "empty_unknown", "missing", "zero_metadata", "reversed", "duplicate", "bad_box", "nan_box", "literal_conflict", "layout_conflict"} {
		t.Run(mode, func(t *testing.T) {
			d, p := gapFixture("-")
			p.Rows[0].Description = Selection{"desc1", "desc2", "desc3", "desc4", "desc5"}
			p.Service = nil
			r, e := ValidateProposal(d, p, []int{1}, true, "")
			if e != nil || len(r.Rows[0].Issues) > 0 {
				t.Fatal("positive invalid", e, r)
			}
			ev := &r.Rows[0].Evidence[len(r.Rows[0].Evidence)-1]
			switch mode {
			case "line_wrap":
				for i := 2; i < len(ev.Spans); i++ {
					ev.Spans[i].Line = 2
					ev.Spans[i].Word = i - 1
					ev.Spans[i].Box.YMin += 20
					ev.Spans[i].Box.YMax += 20
				}
			case "block_wrap":
				for i := 2; i < len(ev.Spans); i++ {
					ev.Spans[i].Block++
					ev.Spans[i].Word = i - 1
				}
			case "page_wrap":
				for i := 2; i < len(ev.Spans); i++ {
					ev.Spans[i].Page = 2
					ev.Spans[i].Word = i - 1
				}
			case "rename_import_text":
				r.Rows[0].Description = "Owner merchant description"
			case "empty_unknown":
				r.Rows[0].Description = ""
				r.Rows[0].Evidence = nil
			case "missing":
				r.Rows[0].Evidence = nil
			case "zero_metadata":
				ev.Spans[0].Word = 0
			case "reversed":
				ev.Spans[0], ev.Spans[1] = ev.Spans[1], ev.Spans[0]
			case "duplicate":
				ev.Spans[1] = ev.Spans[0]
			case "bad_box":
				ev.Spans[0].Box.XMax = ev.Spans[0].Box.XMin
			case "nan_box":
				ev.Spans[0].Box.YMin = math.NaN()
			case "literal_conflict":
				ev.Literal = "Fabricated retained literal"
			case "layout_conflict":
				d.Spans[3].Text = "Altered layout"
			}
			layout := (*Document)(nil)
			if mode == "layout_conflict" {
				layout = &d
			}
			ApplyGroundingChecks(layout, &r)
			good := mode == "complete" || mode == "line_wrap" || mode == "block_wrap" || mode == "page_wrap" || mode == "rename_import_text" || mode == "empty_unknown"
			if (len(r.Rows[0].Issues) == 0) != good {
				t.Fatal(mode, r.Rows[0].Issues)
			}
			if !good && !contains(r.Reconciliation.Blockers, "row_1:description_evidence_unverified") {
				t.Fatal("missing closed blocker", r.Reconciliation)
			}
		})
	}
}

func bookRoleFixture(role, label string) (Document, Review) {
	l := Span{ID: "label", Page: 1, Block: 1, Line: 1, Word: 1, Box: Box{30, 100, 110, 110}, Text: label}
	v := Span{ID: "value", Page: 1, Block: 1, Line: 1, Word: 2, Box: Box{140, 100, 180, 110}, Text: "100.00"}
	h := Header{Evidence: []Evidence{{Role: role, Spans: []Span{v}, Literal: v.Text}, {Role: role + "Label", Spans: []Span{l}, Literal: l.Text}}}
	if role == "openingBookBalance" {
		h.OpeningBookBalance = &Money{10000, "RUB"}
	} else {
		h.ClosingBookBalance = &Money{10000, "RUB"}
	}
	return Document{Extractor: ExtractorVersion, Pages: []Page{{1, 600, 800}}, Spans: []Span{l, v}}, Review{Header: h, Reconciliation: Reconciliation{BookBalance: Check{Status: "matched"}}}
}
func TestBookRoleLabelsGeometryAndValues(t *testing.T) {
	for role, labels := range map[string][]string{"openingBookBalance": {"opening book balance", " OPENING BALANCE: ", "начальный остаток", "входящий остаток", "остаток на начало периода"}, "closingBookBalance": {"closing book balance", "closing balance", "конечный остаток", "исходящий остаток", "остаток на конец периода"}} {
		for _, label := range append(labels, "SYNTHETIC BANK", "Available balance", "Debit turnover", "Unknown balance", "Opening balance 100", "Closing balance 100", "БИК") {
			d, r := bookRoleFixture(role, label)
			before := encoded(r.Header)
			good := bookBalanceLabel(role, label)
			// Every explicit positive phrase is independently expected, not just the classifier.
			listed := false
			for _, v := range labels {
				if v == label {
					listed = true
				}
			}
			if listed != good {
				t.Fatal("dictionary disagreement", label)
			}
			ApplyGroundingChecks(&d, &r)
			ApplyGroundingChecks(nil, &r)
			if (len(r.Reconciliation.Blockers) == 0) != listed || before != encoded(r.Header) {
				t.Fatal(role, label, r)
			}
			if !listed && (r.Reconciliation.BookBalance.Status != "incomplete" || len(r.Reconciliation.Blockers) != 1) {
				t.Fatal("diagnostic status/dedup", r)
			}
		}
		other := "closing balance"
		if role == "closingBookBalance" {
			other = "opening balance"
		}
		for _, mode := range []string{"next_line", "other_page", "distant", "left", "wrong_column", "wrapped_label", "missing_value", "missing_label", "duplicate_value", "literal_conflict", "value_mismatch", "wrong_role", "no_metadata"} {
			t.Run(role+"/"+mode, func(t *testing.T) {
				_, r := bookRoleFixture(role, labels[0])
				switch mode {
				case "next_line":
					r.Header.Evidence[0].Spans[0].Box = Box{30, 120, 80, 130}
				case "other_page":
					r.Header.Evidence[1].Spans[0].Page = 2
				case "distant":
					r.Header.Evidence[0].Spans[0].Box = Box{30, 160, 80, 170}
				case "left":
					r.Header.Evidence[0].Spans[0].Box = Box{1, 100, 20, 110}
				case "wrong_column":
					r.Header.Evidence[0].Spans[0].Box = Box{300, 120, 350, 130}
				case "wrapped_label":
					extra := r.Header.Evidence[1].Spans[0]
					extra.ID = "label2"
					extra.Word++
					extra.Box.YMin += 15
					extra.Box.YMax += 15
					r.Header.Evidence[1].Spans = append(r.Header.Evidence[1].Spans, extra)
					r.Header.Evidence[1].Literal += " " + extra.Text
				case "missing_value":
					r.Header.Evidence = r.Header.Evidence[1:]
				case "missing_label":
					r.Header.Evidence = r.Header.Evidence[:1]
				case "duplicate_value":
					r.Header.Evidence = append(r.Header.Evidence, r.Header.Evidence[0])
				case "literal_conflict":
					r.Header.Evidence[1].Literal = other
				case "value_mismatch":
					if role == "openingBookBalance" {
						r.Header.OpeningBookBalance.AmountMinor++
					} else {
						r.Header.ClosingBookBalance.AmountMinor++
					}
				case "wrong_role":
					r.Header.Evidence[1].Literal = other
					r.Header.Evidence[1].Spans[0].Text = other
				case "no_metadata":
					r.Header.Evidence[0].Spans[0].Word = 0
				}
				ApplyGroundingChecks(nil, &r)
				if (len(r.Reconciliation.Blockers) == 0) != (mode == "next_line") {
					t.Fatal(mode, r.Reconciliation)
				}
			})
		}
	}
}
func TestBookRoleIdentifierContextAndNilParsing(t *testing.T) {
	for _, bank := range []string{"БИК", "КПП", "ИНН", "BIC", "KPP", "INN", "SWIFT"} {
		d, r := bookRoleFixture("openingBookBalance", "opening balance")
		d.Spans = append(d.Spans, Span{ID: "bank", Page: 1, Block: 2, Line: 1, Word: 1, Box: Box{115, 100, 135, 110}, Text: bank})
		ApplyGroundingChecks(&d, &r)
		if !contains(r.Reconciliation.Blockers, "openingBookBalance:book_balance_role_unverified") {
			t.Fatal("identifier accepted", bank)
		}
	}
	d, r := bookRoleFixture("openingBookBalance", "opening balance")
	r.Header.OpeningBookBalance = nil
	r.Reconciliation.Blockers = []string{"openingBookBalance:ambiguous_money"}
	before := encoded(r)
	ApplyGroundingChecks(&d, &r)
	if encoded(r) != before {
		t.Fatal("nil parsing invented a role/value", r)
	}
	// Corrupt selected metadata cannot borrow trust from an existing full layout.
	d, r = bookRoleFixture("closingBookBalance", "closing balance")
	r.Header.Evidence[0].Spans[0].Text = "200.00"
	r.Header.Evidence[0].Literal = "200.00"
	r.Header.ClosingBookBalance.AmountMinor = 20000
	ApplyGroundingChecks(&d, &r)
	if !reflect.DeepEqual(r.Reconciliation.Blockers, []string{"closingBookBalance:book_balance_role_unverified"}) {
		t.Fatal("layout disagreement accepted", r)
	}
	if bookBalanceLabel("openingBookBalance", strings.Repeat("opening balance ", 2)) {
		t.Fatal("ambiguous composite label accepted")
	}
}

func TestGroundingPreservesSplitSignedMoney(t *testing.T) {
	d := syntheticDocument("19.10.2026", "RUB", "Opening balance")
	d.Spans[2].Box = Box{30, 100, 100, 110}
	a := Span{ID: "a", Page: 1, Block: 4, Line: 1, Word: 1, Box: Box{120, 100, 140, 110}, Text: "-1"}
	b := Span{ID: "b", Page: 1, Block: 4, Line: 1, Word: 2, Box: Box{145, 100, 180, 110}, Text: "234.56"}
	d.Spans = append(d.Spans, a, b)
	p := Proposal{Rows: []RowMapping{{OperationDate: Selection{"s0"}, AccountAmount: Selection{"a", "b"}, AccountCurrency: Selection{"s1"}}}, Service: []ServiceMapping{{Spans: Selection{"s2"}, Reason: "Unused synthetic heading"}}}
	review, e := ValidateProposal(d, p, []int{1}, true, "")
	if e != nil || len(review.Reconciliation.Blockers) > 0 || review.Rows[0].AccountAmount.AmountMinor != 123456 || review.Rows[0].Direction != "debit" {
		t.Fatal("split signed row changed", e, review)
	}
	h := Header{OpeningBookBalance: &Money{-123456, "RUB"}, Evidence: []Evidence{{Role: "openingBookBalance", Literal: "-1 234.56", Spans: []Span{a, b}}, {Role: "openingBookBalanceLabel", Literal: d.Spans[2].Text, Spans: []Span{d.Spans[2]}}}}
	book := Review{Header: h, Reconciliation: Reconciliation{BookBalance: Check{Status: "incomplete"}}}
	before := encoded(book.Header)
	ApplyGroundingChecks(&d, &book)
	if len(book.Reconciliation.Blockers) > 0 || before != encoded(book.Header) {
		t.Fatal("split signed book balance changed", book)
	}
}

func TestDescriptionBoundaryOmissionsRemainOutsideGapGuarantee(t *testing.T) {
	d, p := gapFixture("-")
	p.Rows[0].Description = Selection{"desc2", "desc3", "desc4"}
	p.Service = []ServiceMapping{{Spans: Selection{"desc1", "desc5"}, Reason: "Synthetic boundary ambiguity"}}
	r, e := ValidateProposal(d, p, []int{1}, true, "")
	if e != nil || len(r.Rows[0].Issues) > 0 || r.Reconciliation.RowCoverage.Status != "matched" {
		t.Fatal("boundary policy broadened", e, r)
	}
	// Complete-description golden checks remain responsible for this case.
	if r.Rows[0].Description != "TRANSFER - REVIEW" {
		t.Fatal("boundary evidence auto-expanded")
	}
}

func TestDescriptionUsesExtractorOrderWithoutInferringWritingDirection(t *testing.T) {
	d, p := gapFixture("-")
	p.Rows[0].Description = Selection{"desc1", "desc2", "desc3", "desc4", "desc5"}
	p.Service = nil
	for i := 3; i < len(d.Spans); i++ {
		x := float64(120 - (i-3)*20)
		d.Spans[i].Box.XMin = x
		d.Spans[i].Box.XMax = x + 19
	}
	r, e := ValidateProposal(d, p, []int{1}, true, "")
	if e != nil || len(r.Rows[0].Issues) > 0 {
		t.Fatal("authentic extractor word order rejected", e, r)
	}
	ApplyGroundingChecks(nil, &r)
	if len(r.Rows[0].Issues) > 0 || r.Rows[0].Description != "OWN TRANSFER - REVIEW SEMANTICS" {
		t.Fatal("retained reading direction inferred", r)
	}
}
