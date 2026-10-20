package statementdata

import (
	"math"
	"strings"
	"unicode"
)

func labelWords(text string) string {
	return strings.Join(strings.FieldsFunc(strings.ToLower(text), func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsDigit(r) }), " ")
}

func bankIdentifierLabel(text string) bool {
	switch labelWords(text) {
	case "бик", "кпп", "инн", "bic", "kpp", "inn", "swift":
		return true
	}
	return false
}

func turnoverLabel(role, text string) bool {
	text = labelWords(text)
	if role == "debitTurnover" {
		switch text {
		case "debit turnover", "расходы", "списания", "оборот по дебету", "обороты по дебету":
			return true
		}
	} else {
		switch text {
		case "credit turnover", "пополнения", "зачисления", "оборот по кредиту", "обороты по кредиту":
			return true
		}
	}
	return false
}

func turnoverLabelNearValue(label, value Evidence) bool {
	if len(label.Spans) == 0 || len(value.Spans) == 0 {
		return false
	}
	first, amount := label.Spans[0], value.Spans[0]
	left, right := first.Box.XMin, first.Box.XMax
	for _, s := range label.Spans {
		if s.Page != first.Page || math.Abs(s.Box.YMin-first.Box.YMin) > 2 {
			return false
		}
		left = math.Min(left, s.Box.XMin)
		right = math.Max(right, s.Box.XMax)
	}
	if first.Page != amount.Page {
		return false
	}
	dy := amount.Box.YMin - first.Box.YMin
	if math.Abs(dy) <= 2 {
		return amount.Box.XMin >= right-2
	}
	return dy > 2 && dy <= 24 && amount.Box.XMin >= left-2 && amount.Box.XMin <= right+2
}

// TurnoverRoleBlockers checks explicit source meaning, never numerical equality.
// Retained evidence supports old reviews even after the full layout expires.
// Missing unselected layout context cannot be reconstructed or guessed.
func TurnoverRoleBlockers(d *Document, h Header) []string {
	out := []string{}
	for _, field := range []struct {
		role  string
		money *Money
	}{{"debitTurnover", h.DebitTurnover}, {"creditTurnover", h.CreditTurnover}} {
		if field.money == nil {
			continue
		}
		var value, label Evidence
		for _, ev := range h.Evidence {
			if ev.Role == field.role {
				value = ev
			}
			if ev.Role == field.role+"Label" {
				label = ev
			}
		}
		valid := turnoverLabel(field.role, label.Literal) && turnoverLabelNearValue(label, value)
		if d != nil {
			for _, amount := range value.Spans {
				for _, s := range d.Spans {
					if s.Page == amount.Page && bankIdentifierLabel(s.Text) && math.Abs(s.Box.YMin-amount.Box.YMin) <= 2 && amount.Box.XMin >= s.Box.XMax-2 && amount.Box.XMin-s.Box.XMax <= 100 {
						valid = false
					}
				}
			}
		}
		if !valid {
			out = append(out, field.role+":turnover_role_unverified")
		}
	}
	return out
}

// ApplyTurnoverRoleChecks only adds diagnostics. It preserves the proposal and
// parsed values so a reviewer can inspect and manually correct the exact source.
func ApplyTurnoverRoleChecks(d *Document, r *Review) {
	issues := TurnoverRoleBlockers(d, r.Header)
	if len(issues) == 0 {
		return
	}
	r.Reconciliation.Turnover.Status = "incomplete"
	for _, issue := range issues {
		if !hasRoleIssue(r.Reconciliation.Blockers, issue) {
			r.Reconciliation.Blockers = append(r.Reconciliation.Blockers, issue)
		}
	}
	if !hasRoleIssue(r.Reconciliation.Turnover.Reasons, "turnover_role_unverified") {
		r.Reconciliation.Turnover.Reasons = append(r.Reconciliation.Turnover.Reasons, "turnover_role_unverified")
	}
}
func hasRoleIssue(issues []string, issue string) bool {
	for _, s := range issues {
		if s == issue {
			return true
		}
	}
	return false
}
