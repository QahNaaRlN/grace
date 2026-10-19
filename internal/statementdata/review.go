package statementdata

import (
	"errors"
	"fmt"
	"math"
	"regexp"
	"sort"
	"strings"
)

var financialDescriptionToken = regexp.MustCompile(`^[+−-]?[0-9]+[.,][0-9]{2}$|^[0-9]{2}\.[0-9]{2}\.[0-9]{4}$|^[0-9]{4}-[0-9]{2}-[0-9]{2}$`)

// ValidateProposal returns reviewable issues for bad proposals; malformed
// document state/oversized proposals are rejected outright. Owner role and page
// reviews are separate inputs and cannot be supplied by model confidence.
func ValidateProposal(d Document, p Proposal, reviewedPages []int, rolesConfirmed bool, zone string) (Review, error) {
	out := Review{Rows: []Row{}, Header: Header{Evidence: []Evidence{}}}
	if err := d.Validate(); err != nil {
		return out, err
	}
	if len(p.Rows) == 0 || len(p.Rows) > MaxRows || len(p.Service) > MaxSpans {
		return out, errors.New("statement_row_limit")
	}
	positions := make(map[string]int, len(d.Spans))
	for i, s := range d.Spans {
		positions[s.ID] = i
	}
	used := map[string]string{}
	blockers := []string{}
	addIssue := func(code string) { blockers = append(blockers, code) }
	get := func(role string, ids Selection, oneLine bool, claim bool) (Evidence, error) {
		ev, err := ground(d.Spans, positions, role, ids, oneLine)
		limit := 500
		switch {
		case strings.Contains(role, "Amount") || strings.Contains(role, "Balance") || strings.Contains(role, "Turnover"):
			limit = 100
		case strings.Contains(role, "Date"):
			limit = 128
		case strings.Contains(role, "Currency"):
			limit = 16
		case strings.Contains(role, "description"):
			limit = 2000
		case role == "service":
			limit = MaxTextBytes
		}
		if err == nil && strings.Contains(role, "Date") {
			err = validateDateGeometry(ev.Spans)
		}
		if err == nil && len(ev.Literal) > limit {
			return ev, errors.New("field_evidence_limit")
		}
		if err == nil && claim {
			for _, s := range ev.Spans {
				if prev := used[s.ID]; prev != "" {
					return ev, errors.New("overlapping_evidence")
				}
				used[s.ID] = role
			}
		}
		// Headings/currency may legitimately be shared by all rows. Other selected
		// evidence is claimed once, so chunk overlap cannot create another row.
		if err == nil && !claim {
			for _, s := range ev.Spans {
				if used[s.ID] == "" {
					used[s.ID] = "shared"
				}
			}
		}
		return ev, err
	}
	currencyFor := func(role string, ids Selection) (string, Evidence, error) {
		ev, e := get(role, ids, true, false)
		if e != nil {
			return "", ev, e
		}
		c, e := Currency(ev.Literal)
		return c, ev, e
	}
	for i, m := range p.Rows {
		row := Row{Ordinal: i + 1, Evidence: []Evidence{}, Issues: []string{}}
		issue := func(e error) {
			if e != nil {
				row.Issues = append(row.Issues, e.Error())
				addIssue(fmt.Sprintf("row_%d:%s", i+1, e.Error()))
			}
		}
		for _, field := range []struct {
			role   string
			ids    Selection
			target **LocalDate
		}{{"operationDate", m.OperationDate, &row.OperationDate}, {"bookingDate", m.BookingDate, &row.BookingDate}} {
			if len(field.ids) == 0 {
				if field.role == "operationDate" {
					issue(errors.New("operation_date_required"))
				}
				continue
			}
			ev, e := get(fmt.Sprintf("row_%d:%s", i+1, field.role), field.ids, false, true)
			issue(e)
			if e != nil {
				continue
			}
			row.Evidence = append(row.Evidence, ev)
			date, e := ParseLocalDate(ev.Literal, zone)
			issue(e)
			*field.target = date
		}
		for _, field := range []struct {
			role     string
			ids, ccy Selection
			target   **Money
		}{{"accountAmount", m.AccountAmount, m.AccountCurrency, &row.AccountAmount}, {"transactionAmount", m.TransactionAmount, m.TransactionCurrency, &row.TransactionAmount}} {
			if len(field.ids) == 0 {
				if field.role == "accountAmount" {
					issue(errors.New("account_amount_required"))
				}
				continue
			}
			ev, e := get(fmt.Sprintf("row_%d:%s", i+1, field.role), field.ids, true, true)
			issue(e)
			if e != nil {
				continue
			}
			row.Evidence = append(row.Evidence, ev)
			c, cev, e := currencyFor(field.role+"Currency", field.ccy)
			issue(e)
			if e != nil {
				continue
			}
			row.Evidence = append(row.Evidence, cev)
			amount, dir, e := ParseMoney(ev.Literal, c)
			issue(e)
			if e != nil {
				continue
			}
			*field.target = &amount
			if field.role == "accountAmount" {
				if c != "RUB" {
					issue(errors.New("unsupported_account_currency"))
				}
				if amount.AmountMinor == 0 {
					issue(errors.New("zero_posting"))
				}
				row.Direction = dir
				if m.Direction != "" && m.Direction != "debit" && m.Direction != "credit" {
					issue(errors.New("invalid_direction"))
				}
				if dir != "" && m.Direction != "" && dir != m.Direction {
					issue(errors.New("direction_conflict"))
				}
				if dir == "" {
					row.Direction = m.Direction
					if m.Direction != "debit" && m.Direction != "credit" {
						issue(errors.New("direction_required"))
					}
					ev, e := get("direction", m.DirectionEvidence, true, false)
					issue(e)
					if e == nil {
						row.Evidence = append(row.Evidence, ev)
					}
				}
			} else if dir != "" && row.Direction != "" && dir != row.Direction {
				issue(errors.New("transaction_direction_conflict"))
			}
		}
		if len(m.Description) > 0 {
			ev, e := get(fmt.Sprintf("row_%d:description", i+1), m.Description, false, true)
			issue(e)
			if e == nil {
				if len(ev.Literal) > 2000 {
					issue(errors.New("description_evidence_limit"))
				}
				for _, span := range ev.Spans {
					if financialDescriptionToken.MatchString(span.Text) {
						issue(errors.New("financial_candidate_in_description"))
						break
					}
				}
				row.Description = ev.Literal
				row.Evidence = append(row.Evidence, ev)
			}
		}
		out.Rows = append(out.Rows, row)
	}
	headerFields := []struct {
		role   string
		ids    Selection
		target **Money
	}{{"openingBookBalance", p.Header.OpeningBookBalance, &out.Header.OpeningBookBalance}, {"closingBookBalance", p.Header.ClosingBookBalance, &out.Header.ClosingBookBalance}, {"availableBalanceObserved", p.Header.AvailableBalanceObserved, &out.Header.AvailableBalanceObserved}, {"debitTurnover", p.Header.DebitTurnover, &out.Header.DebitTurnover}, {"creditTurnover", p.Header.CreditTurnover, &out.Header.CreditTurnover}}
	for _, f := range headerFields {
		if len(f.ids) == 0 {
			continue
		}
		ev, e := get(f.role, f.ids, true, true)
		if e != nil {
			addIssue(f.role + ":" + e.Error())
			continue
		}
		out.Header.Evidence = append(out.Header.Evidence, ev)
		label, e := get(f.role+"Label", p.Header.Labels[f.role], false, false)
		if e != nil {
			addIssue(f.role + ":field_label_required")
		} else {
			out.Header.Evidence = append(out.Header.Evidence, label)
			lower := strings.ToLower(label.Literal)
			if (f.role == "openingBookBalance" || f.role == "closingBookBalance") && (strings.Contains(lower, "доступ") || strings.Contains(lower, "available")) {
				addIssue(f.role + ":available_is_not_book_balance")
			}
		}
		c, cev, e := currencyFor("headerCurrency", p.Header.Currency)
		if e != nil {
			addIssue(f.role + ":" + e.Error())
			continue
		}
		out.Header.Evidence = append(out.Header.Evidence, cev)
		if c != "RUB" {
			addIssue(f.role + ":unsupported_account_currency")
		}
		v, dir, e := ParseMoney(ev.Literal, c)
		if e != nil {
			addIssue(f.role + ":" + e.Error())
			continue
		}
		if (f.role == "debitTurnover" || f.role == "creditTurnover") && dir == "debit" {
			addIssue(f.role + ":negative_turnover")
		}
		if dir == "debit" {
			v.AmountMinor = -v.AmountMinor
		}
		*f.target = &v
	}
	for _, field := range []struct {
		role   string
		ids    Selection
		target **LocalDate
	}{{"openingBookDate", p.Header.OpeningBookDate, &out.Header.OpeningBookDate}, {"closingBookDate", p.Header.ClosingBookDate, &out.Header.ClosingBookDate}} {
		if len(field.ids) == 0 {
			continue
		}
		ev, e := get(field.role, field.ids, false, true)
		if e != nil {
			addIssue(field.role + ":" + e.Error())
			continue
		}
		out.Header.Evidence = append(out.Header.Evidence, ev)
		date, e := ParseLocalDate(ev.Literal, zone)
		if e != nil {
			addIssue(field.role + ":" + e.Error())
			continue
		}
		*field.target = date
	}
	for _, service := range p.Service {
		if strings.TrimSpace(service.Reason) == "" || len(service.Reason) > 2000 {
			addIssue("service_reason_required")
			continue
		}
		// Service is an explicit classification set, not a financial value.
		// Canonicalize its ordering only. Unknown/duplicate/overlapping IDs
		// still fail, and money/date/currency selections retain strict ordering.
		ids := append(Selection{}, service.Spans...)
		sort.SliceStable(ids, func(i, j int) bool {
			a, okA := positions[ids[i]]
			b, okB := positions[ids[j]]
			if !okA {
				return false
			}
			if !okB {
				return true
			}
			return a < b
		})
		if _, e := get("service", ids, false, true); e != nil {
			addIssue("service:" + e.Error())
		}
	}
	uncovered := 0
	for _, s := range d.Spans {
		if used[s.ID] == "" {
			uncovered++
		}
	}
	pageSet := map[int]bool{}
	for _, page := range reviewedPages {
		if page < 1 || page > len(d.Pages) || pageSet[page] {
			addIssue("invalid_page_review")
		}
		pageSet[page] = true
	}
	coverage := Check{"matched", []string{}}
	if uncovered > 0 {
		coverage.Status = "incomplete"
		coverage.Reasons = append(coverage.Reasons, fmt.Sprintf("unmapped_spans:%d", uncovered))
		addIssue("source_coverage_incomplete")
	}
	if len(pageSet) != len(d.Pages) {
		coverage.Status = "not_reviewed"
		coverage.Reasons = append(coverage.Reasons, "owner_page_review_required")
		addIssue("owner_page_review_required")
	}
	if !rolesConfirmed {
		addIssue("owner_field_role_review_required")
	}
	out.Reconciliation = Reconcile(out.Rows, out.Header)
	out.Reconciliation.RowCoverage = coverage
	out.Reconciliation.Blockers = append(out.Reconciliation.Blockers, blockers...)
	return out, nil
}

// Dates and clocks are often printed on adjacent lines in the same column.
// Money still requires one contiguous line; temporal evidence can span a short
// vertical block but cannot combine a clock from a different page/column.
func validateDateGeometry(spans []Span) error {
	if len(spans) < 2 {
		return nil
	}
	first := spans[0]
	for index, span := range spans[1:] {
		previous := spans[index]
		if span.Page != first.Page {
			return errors.New("noncontiguous_date_evidence")
		}
		if span.Block == previous.Block && span.Line == previous.Line {
			if span.Word != previous.Word+1 {
				return errors.New("noncontiguous_date_evidence")
			}
			continue
		}
		if math.Abs(span.Box.XMin-first.Box.XMin) > math.Max(20, first.Box.XMax-first.Box.XMin) || span.Box.YMin < first.Box.YMin-2 || span.Box.YMin-first.Box.YMin > math.Max(32, (first.Box.YMax-first.Box.YMin)*3) {
			return errors.New("noncontiguous_date_evidence")
		}
	}
	return nil
}
