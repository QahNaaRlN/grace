package statementdata

import "testing"

func TestTurnoverRoleGeometryAndRetainedEvidence(t *testing.T) {
	for _, role := range []string{"debitTurnover", "creditTurnover"} {
		label := "Расходы:"
		if role == "creditTurnover" {
			label = "Пополнения:"
		}
		for _, mode := range []string{"same_line", "next_line", "other_page", "distant", "left_of_label", "wrong_column", "wrapped_label", "missing_evidence", "reversed"} {
			t.Run(role+"/"+mode, func(t *testing.T) {
				l := Span{ID: "label", Page: 1, Box: Box{30, 100, 100, 110}, Text: label}
				v := Span{ID: "value", Page: 1, Box: Box{130, 100, 180, 110}, Text: "100.00"}
				h := Header{Evidence: []Evidence{{Role: role, Spans: []Span{v}, Literal: v.Text}, {Role: role + "Label", Spans: []Span{l}, Literal: l.Text}}}
				if role == "debitTurnover" {
					h.DebitTurnover = &Money{10000, "RUB"}
				} else {
					h.CreditTurnover = &Money{10000, "RUB"}
				}
				switch mode {
				case "next_line":
					h.Evidence[0].Spans[0].Box = Box{30, 120, 80, 130}
				case "other_page":
					h.Evidence[1].Spans[0].Page = 2
				case "distant":
					h.Evidence[0].Spans[0].Box.YMin = 200
				case "left_of_label":
					h.Evidence[0].Spans[0].Box.XMin = 1
				case "wrong_column":
					h.Evidence[0].Spans[0].Box = Box{300, 120, 350, 130}
				case "wrapped_label":
					extra := l
					extra.Box.YMin += 10
					h.Evidence[1].Spans = append(h.Evidence[1].Spans, extra)
				case "missing_evidence":
					h.Evidence = nil
				case "reversed":
					h.Evidence[1].Literal = "Credit turnover"
					if role == "creditTurnover" {
						h.Evidence[1].Literal = "Debit turnover"
					}
				}
				good := mode == "same_line" || mode == "next_line"
				issues := TurnoverRoleBlockers(nil, h)
				if (len(issues) == 0) != good {
					t.Fatal(issues)
				}
				r := Review{Header: h, Reconciliation: Reconciliation{Turnover: Check{Status: "matched"}}}
				ApplyTurnoverRoleChecks(nil, &r)
				ApplyTurnoverRoleChecks(nil, &r)
				if !good && (r.Reconciliation.Turnover.Status != "incomplete" || len(r.Reconciliation.Blockers) != 1) {
					t.Fatal("legacy or duplicate diagnostic", r.Reconciliation)
				}
			})
		}
	}
	if len(TurnoverRoleBlockers(nil, Header{})) != 0 {
		t.Fatal("invented missing turnover")
	}
}

func TestBankIdentifierContextBlocksEvenBorrowedNearbyPositiveLabel(t *testing.T) {
	label := Span{Page: 1, Text: "Расходы:", Box: Box{1, 100, 40, 110}}
	bank := Span{Page: 1, Text: "БИК", Box: Box{45, 100, 60, 110}}
	value := Span{Page: 1, Text: "100", Box: Box{70, 100, 100, 110}}
	h := Header{DebitTurnover: &Money{10000, "RUB"}, Evidence: []Evidence{{Role: "debitTurnover", Literal: value.Text, Spans: []Span{value}}, {Role: "debitTurnoverLabel", Literal: label.Text, Spans: []Span{label}}}}
	if len(TurnoverRoleBlockers(nil, h)) != 0 {
		t.Fatal("fixture lacks valid label geometry")
	}
	if len(TurnoverRoleBlockers(&Document{Spans: []Span{label, bank, value}}, h)) != 1 {
		t.Fatal("nearby bank role ignored")
	}
	bank.Page = 2
	if len(TurnoverRoleBlockers(&Document{Spans: []Span{label, bank, value}}, h)) != 0 {
		t.Fatal("unrelated bank label blocked turnover")
	}
}

func TestTurnoverLabelsDoNotEraseDigitsOrMixDirections(t *testing.T) {
	for _, text := range []string{"Расходы 123", "Расходы123", "БИК Расходы", "Расходы Пополнения", "Debit turnover 100"} {
		if turnoverLabel("debitTurnover", text) {
			t.Fatal("ambiguous label accepted", text)
		}
	}
	for _, text := range []string{" Расходы: ", "СПИСАНИЯ", "Обороты по дебету", "Debit turnover:"} {
		if !turnoverLabel("debitTurnover", text) {
			t.Fatal("supported label lost", text)
		}
	}
}
