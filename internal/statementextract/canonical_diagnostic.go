package statementextract

import (
	"errors"
	"sort"

	"github.com/QahNaaRlN/grace/internal/statementdata"
)

const CanonicalDiagnosticVersion = "statement.canonicalization-diagnostic.v1"

// CanonicalDiagnostic contains no model-provided strings or aliases.
type CanonicalDiagnostic struct {
	Version        string `json:"version"`
	Stage          string `json:"stage"`
	Code           string `json:"code"`
	Scope          string `json:"scope"`
	Field          string `json:"field"`
	Ordinal        int    `json:"ordinal"`
	SelectionIndex int    `json:"selectionIndex"`
	Role           string `json:"role"`
}

func (d CanonicalDiagnostic) Valid() bool {
	if d.Version != CanonicalDiagnosticVersion || d.Stage != "canonicalization" {
		return false
	}
	if d.Code == "invalid_plan" {
		return d.Scope == "plan" && d.Field == "plan" && d.Ordinal == 0 && d.SelectionIndex == 0 && d.Role == ""
	}
	if d.Code != "unknown_alias" && d.Code != "duplicate_alias" && d.Code != "non_numeric_money_selection" {
		return false
	}
	if d.SelectionIndex < 1 || d.Ordinal < 0 {
		return false
	}
	money := false
	switch d.Scope {
	case "header":
		if d.Ordinal != 0 || d.Role != "" {
			return false
		}
		switch d.Field {
		case "openingBookBalance", "closingBookBalance", "availableBalanceObserved", "debitTurnover", "creditTurnover":
			money = true
		case "openingBookDate", "closingBookDate", "currency":
		default:
			return false
		}
	case "row":
		if d.Ordinal < 1 || d.Role != "" {
			return false
		}
		switch d.Field {
		case "accountAmount", "transactionAmount":
			money = true
		case "operationDate", "bookingDate", "accountCurrency", "transactionCurrency", "description", "directionEvidence":
		default:
			return false
		}
	case "header_label":
		if d.Ordinal != 0 || d.Field != "labels" || !closedCanonicalRole(d.Role) {
			return false
		}
	case "service":
		if d.Ordinal < 1 || d.Field != "spans" || d.Role != "" {
			return false
		}
	default:
		return false
	}
	return d.Code != "non_numeric_money_selection" || money
}
func closedCanonicalRole(role string) bool {
	switch role {
	case "openingBookBalance", "closingBookBalance", "availableBalanceObserved", "debitTurnover", "creditTurnover", "unknown":
		return true
	}
	return false
}

// This operator-only entry point shares every predicate with legacy callers.
func CanonicalComparisonProposalDiagnostic(p Plan, in statementdata.Proposal) (statementdata.Proposal, *CanonicalDiagnostic, error) {
	return canonicalProposalDetailed(p, in, planShapeValid(p, ComparisonPolicyValid(p.Policy)))
}

type canonicalField struct {
	target       *statementdata.Selection
	scope, field string
	ordinal      int
	role         string
}

func canonicalFields(p *statementdata.Proposal) []canonicalField {
	fields := []canonicalField{}
	for i := range p.Rows {
		r := &p.Rows[i]
		for _, f := range []canonicalField{{&r.OperationDate, "row", "operationDate", i + 1, ""}, {&r.BookingDate, "row", "bookingDate", i + 1, ""}, {&r.AccountAmount, "row", "accountAmount", i + 1, ""}, {&r.AccountCurrency, "row", "accountCurrency", i + 1, ""}, {&r.TransactionAmount, "row", "transactionAmount", i + 1, ""}, {&r.TransactionCurrency, "row", "transactionCurrency", i + 1, ""}, {&r.Description, "row", "description", i + 1, ""}, {&r.DirectionEvidence, "row", "directionEvidence", i + 1, ""}} {
			fields = append(fields, f)
		}
	}
	h := &p.Header
	fields = append(fields, canonicalField{&h.OpeningBookDate, "header", "openingBookDate", 0, ""}, canonicalField{&h.ClosingBookDate, "header", "closingBookDate", 0, ""}, canonicalField{&h.OpeningBookBalance, "header", "openingBookBalance", 0, ""}, canonicalField{&h.ClosingBookBalance, "header", "closingBookBalance", 0, ""}, canonicalField{&h.AvailableBalanceObserved, "header", "availableBalanceObserved", 0, ""}, canonicalField{&h.DebitTurnover, "header", "debitTurnover", 0, ""}, canonicalField{&h.CreditTurnover, "header", "creditTurnover", 0, ""}, canonicalField{&h.Currency, "header", "currency", 0, ""})
	return fields
}
func canonicalProposalDetailed(plan Plan, in statementdata.Proposal, valid bool) (statementdata.Proposal, *CanonicalDiagnostic, error) {
	fail := func(code, scope, field string, ordinal, index int, role string) (statementdata.Proposal, *CanonicalDiagnostic, error) {
		return statementdata.Proposal{}, &CanonicalDiagnostic{CanonicalDiagnosticVersion, "canonicalization", code, scope, field, ordinal, index, role}, errors.New("provider_protocol")
	}
	if !valid {
		return fail("invalid_plan", "plan", "plan", 0, 0, "")
	}
	monetary := map[string]bool{}
	for _, s := range plan.Chunks[0].Spans {
		monetary[s.ID] = s.LexicalClass == ClassNumericFragment
	}
	h := in.Header
	money := []canonicalField{{&h.OpeningBookBalance, "header", "openingBookBalance", 0, ""}, {&h.ClosingBookBalance, "header", "closingBookBalance", 0, ""}, {&h.AvailableBalanceObserved, "header", "availableBalanceObserved", 0, ""}, {&h.DebitTurnover, "header", "debitTurnover", 0, ""}, {&h.CreditTurnover, "header", "creditTurnover", 0, ""}}
	for i := range in.Rows {
		r := &in.Rows[i]
		money = append(money, canonicalField{&r.AccountAmount, "row", "accountAmount", i + 1, ""}, canonicalField{&r.TransactionAmount, "row", "transactionAmount", i + 1, ""})
	}
	for _, f := range money {
		for j, id := range *f.target {
			if !monetary[id] {
				code := "non_numeric_money_selection"
				if _, ok := plan.Aliases[id]; !ok {
					code = "unknown_alias"
				}
				return fail(code, f.scope, f.field, f.ordinal, j+1, "")
			}
		}
	}
	out := in
	out.Rows = append([]statementdata.RowMapping{}, in.Rows...)
	out.Service = append([]statementdata.ServiceMapping{}, in.Service...)
	out.Header.Labels = make(map[string]statementdata.Selection, len(in.Header.Labels))
	mapSelection := func(ids statementdata.Selection) (statementdata.Selection, string, int) {
		if ids == nil {
			return nil, "", 0
		}
		mapped := make(statementdata.Selection, 0, len(ids))
		seen := map[string]bool{}
		for j, alias := range ids {
			id, ok := plan.Aliases[alias]
			if !ok {
				return nil, "unknown_alias", j + 1
			}
			if seen[alias] {
				return nil, "duplicate_alias", j + 1
			}
			seen[alias] = true
			mapped = append(mapped, id)
		}
		return mapped, "", 0
	}
	for _, f := range canonicalFields(&out) {
		ids, code, j := mapSelection(*f.target)
		if code != "" {
			return fail(code, f.scope, f.field, f.ordinal, j, "")
		}
		*f.target = ids
	}
	roles := make([]string, 0, len(in.Header.Labels))
	for role := range in.Header.Labels {
		roles = append(roles, role)
	}
	sort.Strings(roles)
	for _, role := range roles {
		ids, code, j := mapSelection(in.Header.Labels[role])
		if code != "" {
			closed := role
			if !closedCanonicalRole(closed) {
				closed = "unknown"
			}
			return fail(code, "header_label", "labels", 0, j, closed)
		}
		out.Header.Labels[role] = ids
	}
	for i := range out.Service {
		ids, code, j := mapSelection(out.Service[i].Spans)
		if code != "" {
			return fail(code, "service", "spans", i+1, j, "")
		}
		out.Service[i].Spans = ids
	}
	return out, nil, nil
}
