package statementdata

import (
	"fmt"
	"math"
	"strings"
	"unicode/utf8"
)

// groundingEvidence validates retained source metadata without inventing layout.
// With a full layout, every selected span must still match its original value.
type groundingEvidence struct {
	document  *Document
	positions map[string]int
	valid     bool
}

func newGroundingEvidence(d *Document) groundingEvidence {
	g := groundingEvidence{valid: true}
	if d == nil {
		return g
	}
	if d.Validate() != nil {
		g.valid = false
		return g
	}
	g.document = d
	g.positions = make(map[string]int, len(d.Spans))
	for i, s := range d.Spans {
		g.positions[s.ID] = i
	}
	return g
}
func samePrintedLine(a, b Span) bool {
	return a.Page == b.Page && a.Block == b.Block && a.Line == b.Line
}
func orderedSpan(a, b Span) bool {
	if a.Page != b.Page {
		return a.Page < b.Page
	}
	if a.Block != b.Block {
		return a.Block < b.Block
	}
	if a.Line != b.Line {
		return a.Line < b.Line
	}
	return a.Word < b.Word
}
func (g groundingEvidence) check(ev Evidence) bool {
	if !g.valid || len(ev.Spans) == 0 || len(ev.Spans) > 2048 {
		return false
	}
	seen := map[string]bool{}
	parts := make([]string, 0, len(ev.Spans))
	previous := -1
	for i, s := range ev.Spans {
		b := s.Box
		if s.ID == "" || seen[s.ID] || s.Page < 1 || s.Page > MaxPages || s.Block < 1 || s.Line < 1 || s.Word < 1 || strings.TrimSpace(s.Text) == "" || !utf8.ValidString(s.Text) || strings.ContainsRune(s.Text, 0) || !finite(b.XMin) || !finite(b.XMax) || !finite(b.YMin) || !finite(b.YMax) || b.XMin < 0 || b.YMin < 0 || b.XMax <= b.XMin || b.YMax <= b.YMin {
			return false
		}
		seen[s.ID] = true
		if g.document != nil {
			position, ok := g.positions[s.ID]
			if !ok || g.document.Spans[position] != s || position <= previous {
				return false
			}
			previous = position
		} else if i > 0 && !orderedSpan(ev.Spans[i-1], s) {
			return false
		}
		if i > 0 && samePrintedLine(ev.Spans[i-1], s) && s.Box.XMin < ev.Spans[i-1].Box.XMin {
			return false
		}
		parts = append(parts, s.Text)
	}
	return ev.Literal == strings.Join(parts, " ")
}
func uniqueEvidence(evs []Evidence, role string) (Evidence, bool) {
	var out Evidence
	count := 0
	for _, ev := range evs {
		if ev.Role == role {
			out = ev
			count++
		}
	}
	return out, count == 1
}
func descriptionIssue(row Row, index int, g groundingEvidence) string {
	role := fmt.Sprintf("row_%d:description", index+1)
	ev, ok := uniqueEvidence(row.Evidence, role)
	if !ok {
		if row.Description == "" { // Empty unknown is allowed only when no description selection exists.
			for _, candidate := range row.Evidence {
				if candidate.Role == role {
					return "description_evidence_unverified"
				}
			}
			return ""
		}
		return "description_evidence_unverified"
	}
	if !g.check(ev) {
		return "description_evidence_unverified"
	}
	for i := 1; i < len(ev.Spans); i++ {
		a, b := ev.Spans[i-1], ev.Spans[i]
		if samePrintedLine(a, b) && b.Word != a.Word+1 {
			return "description_evidence_gap"
		}
	}
	return ""
}
func bookBalanceLabel(role, text string) bool {
	text = labelWords(text)
	switch role {
	case "openingBookBalance":
		switch text {
		case "opening book balance", "opening balance", "начальный остаток", "входящий остаток", "остаток на начало периода":
			return true
		}
	case "closingBookBalance":
		switch text {
		case "closing book balance", "closing balance", "конечный остаток", "исходящий остаток", "остаток на конец периода":
			return true
		}
	}
	return false
}
func bookValueMatches(ev Evidence, m Money) bool {
	for i := 1; i < len(ev.Spans); i++ {
		a, b := ev.Spans[i-1], ev.Spans[i]
		if !samePrintedLine(a, b) || b.Word != a.Word+1 {
			return false
		}
	}
	parsed, direction, e := ParseMoney(ev.Literal, m.Currency)
	if e != nil {
		return false
	}
	if direction == "debit" {
		parsed.AmountMinor = -parsed.AmountMinor
	}
	return parsed == m
}
func bookBalanceRoleBlockers(d *Document, h Header, g groundingEvidence) []string {
	out := []string{}
	for _, field := range []struct {
		role  string
		money *Money
	}{{"openingBookBalance", h.OpeningBookBalance}, {"closingBookBalance", h.ClosingBookBalance}} {
		if field.money == nil {
			continue
		}
		value, vok := uniqueEvidence(h.Evidence, field.role)
		label, lok := uniqueEvidence(h.Evidence, field.role+"Label")
		valid := vok && lok && g.check(value) && g.check(label) && bookValueMatches(value, *field.money) && bookBalanceLabel(field.role, label.Literal) && turnoverLabelNearValue(label, value)
		if valid && d != nil {
			for _, amount := range value.Spans {
				for _, s := range d.Spans {
					if s.Page == amount.Page && bankIdentifierLabel(s.Text) && math.Abs(s.Box.YMin-amount.Box.YMin) <= 2 && amount.Box.XMin >= s.Box.XMax-2 && amount.Box.XMin-s.Box.XMax <= 100 {
						valid = false
					}
				}
			}
		}
		if !valid {
			out = append(out, field.role+":book_balance_role_unverified")
		}
	}
	return out
}

// ApplyGroundingChecks adds source diagnostics only. Monetary values, descriptions,
// selections and persisted reviews are never repaired by this function.
func ApplyGroundingChecks(d *Document, r *Review) {
	g := newGroundingEvidence(d)
	for i := range r.Rows {
		code := descriptionIssue(r.Rows[i], i, g)
		if code == "" {
			continue
		}
		if !hasRoleIssue(r.Rows[i].Issues, code) {
			r.Rows[i].Issues = append(r.Rows[i].Issues, code)
		}
		blocker := fmt.Sprintf("row_%d:%s", i+1, code)
		if !hasRoleIssue(r.Reconciliation.Blockers, blocker) {
			r.Reconciliation.Blockers = append(r.Reconciliation.Blockers, blocker)
		}
	}
	issues := bookBalanceRoleBlockers(d, r.Header, g)
	if len(issues) > 0 {
		r.Reconciliation.BookBalance.Status = "incomplete"
		if !hasRoleIssue(r.Reconciliation.BookBalance.Reasons, "book_balance_role_unverified") {
			r.Reconciliation.BookBalance.Reasons = append(r.Reconciliation.BookBalance.Reasons, "book_balance_role_unverified")
		}
		for _, issue := range issues {
			if !hasRoleIssue(r.Reconciliation.Blockers, issue) {
				r.Reconciliation.Blockers = append(r.Reconciliation.Blockers, issue)
			}
		}
	}
}
