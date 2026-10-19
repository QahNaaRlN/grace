package statementextract

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"sort"
	"strconv"

	"github.com/QahNaaRlN/grace/internal/statementdata"
)

// A persisted consent is usable only for this exact complete wire contract.
func validPlan(p Plan) bool {
	if p.PlannerVersion != PlannerVersion || p.AliasVersion != AliasVersion || p.PromptVersion != PromptVersion || p.RedactionVersion != RedactionVersion || p.SchemaHash != Hash(json.RawMessage(ProposalSchema)) || p.Limits != currentLimits() || !p.Policy.Valid() || p.MaximumMicros != p.Policy.CallMicros() {
		return false
	}
	hash, e := hex.DecodeString(p.DocumentHash)
	if e != nil || len(hash) != 32 {
		return false
	}
	if len(p.Atoms) != 1 || len(p.Chunks) != 1 {
		return false
	}
	atom, chunk := p.Atoms[0], p.Chunks[0]
	if len(atom) == 0 || len(atom) > MaxPlanSpans || len(p.Aliases) != len(atom) || len(chunk.Spans) != len(atom) || chunk.Index != 1 || len(chunk.Context) != 0 || InputBytes(chunk) > MaxInputBytes {
		return false
	}
	ids := append([]string{}, atom...)
	sort.Strings(ids)
	seen := map[string]bool{}
	for i, id := range ids {
		if id == "" || seen[id] || p.Aliases["s"+strconv.FormatInt(int64(i), 36)] != id {
			return false
		}
		seen[id] = true
	}
	wireSeen := map[string]bool{}
	for i, span := range chunk.Spans {
		if wireSeen[span.ID] || p.Aliases[span.ID] != atom[i] || span.Page < 1 || span.Text == "" {
			return false
		}
		wireSeen[span.ID] = true
	}
	return true
}

// CanonicalProposal copies every typed selection; it never edits source text,
// direction, service reasons or the provider's original proposal in place.
func CanonicalProposal(plan Plan, in statementdata.Proposal) (statementdata.Proposal, error) {
	if !validPlan(plan) {
		return statementdata.Proposal{}, errors.New("provider_protocol")
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
