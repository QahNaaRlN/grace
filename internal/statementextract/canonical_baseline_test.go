// Test-only oracle copied verbatim from f4fc56a aliases.go. Never used by production.
package statementextract

import (
	"errors"
	"github.com/QahNaaRlN/grace/internal/statementdata"
)

func baselineCanonicalF4(plan Plan, in statementdata.Proposal, valid bool) (statementdata.Proposal, error) {
	if !valid {
		return statementdata.Proposal{}, errors.New("provider_protocol")
	}
	monetary := map[string]bool{}
	for _, span := range plan.Chunks[0].Spans {
		monetary[span.ID] = span.LexicalClass == ClassNumericFragment
	}
	moneySelections := []statementdata.Selection{in.Header.OpeningBookBalance, in.Header.ClosingBookBalance, in.Header.AvailableBalanceObserved, in.Header.DebitTurnover, in.Header.CreditTurnover}
	for _, row := range in.Rows {
		moneySelections = append(moneySelections, row.AccountAmount, row.TransactionAmount)
	}
	for _, ids := range moneySelections {
		for _, id := range ids {
			if !monetary[id] {
				return statementdata.Proposal{}, errors.New("provider_protocol")
			}
		}
	}
	var failed bool
	selection := func(ids statementdata.Selection) statementdata.Selection {
		if ids == nil {
			return nil
		}
		out := make(statementdata.Selection, 0, len(ids))
		seen := map[string]bool{}
		for _, alias := range ids {
			id, ok := plan.Aliases[alias]
			if !ok || seen[alias] {
				failed = true
				return nil
			}
			seen[alias] = true
			out = append(out, id)
		}
		return out
	}
	out := in
	out.Rows = append([]statementdata.RowMapping{}, in.Rows...)
	for i := range out.Rows {
		r := &out.Rows[i]
		r.OperationDate = selection(r.OperationDate)
		r.BookingDate = selection(r.BookingDate)
		r.AccountAmount = selection(r.AccountAmount)
		r.AccountCurrency = selection(r.AccountCurrency)
		r.TransactionAmount = selection(r.TransactionAmount)
		r.TransactionCurrency = selection(r.TransactionCurrency)
		r.Description = selection(r.Description)
		r.DirectionEvidence = selection(r.DirectionEvidence)
	}
	h := &out.Header
	h.OpeningBookDate = selection(h.OpeningBookDate)
	h.ClosingBookDate = selection(h.ClosingBookDate)
	h.OpeningBookBalance = selection(h.OpeningBookBalance)
	h.ClosingBookBalance = selection(h.ClosingBookBalance)
	h.AvailableBalanceObserved = selection(h.AvailableBalanceObserved)
	h.DebitTurnover = selection(h.DebitTurnover)
	h.CreditTurnover = selection(h.CreditTurnover)
	h.Currency = selection(h.Currency)
	h.Labels = make(map[string]statementdata.Selection, len(in.Header.Labels))
	for role, ids := range in.Header.Labels {
		h.Labels[role] = selection(ids)
	}
	out.Service = append([]statementdata.ServiceMapping{}, in.Service...)
	for i := range out.Service {
		out.Service[i].Spans = selection(out.Service[i].Spans)
	}
	if failed {
		return statementdata.Proposal{}, errors.New("provider_protocol")
	}
	return out, nil
}
