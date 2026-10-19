package statementextract

import (
	"errors"
	"strings"

	"github.com/QahNaaRlN/grace/internal/statementdata"
)

func MergeProposals(plan Plan, parts []statementdata.Proposal) (statementdata.Proposal, error) {
	out := statementdata.Proposal{Rows: []statementdata.RowMapping{}, Header: statementdata.HeaderMapping{Labels: map[string]statementdata.Selection{}}, Service: []statementdata.ServiceMapping{}}
	if len(parts) != len(plan.Chunks) {
		return out, errors.New("statement_incomplete")
	}
	contextIDs := map[string]bool{}
	for _, chunk := range plan.Chunks {
		for _, span := range chunk.Context {
			contextIDs[span.ID] = true
		}
	}
	rows := map[string]bool{}
	fields := map[string]bool{}
	merge := func(target *statementdata.Selection, in statementdata.Selection) error {
		if len(in) == 0 {
			return nil
		}
		if len(*target) > 0 && strings.Join(*target, "\x00") != strings.Join(in, "\x00") {
			return errors.New("conflicting_header_proposals")
		}
		*target = in
		return nil
	}
	for index, p := range parts {
		allowed := map[string]bool{}
		for _, span := range plan.Chunks[index].Spans {
			id := span.ID
			if plan.PlannerVersion == PlannerVersion {
				id = plan.Aliases[id]
			}
			allowed[id] = true
		}
		for id := range contextIDs {
			allowed[id] = true
		}
		check := func(ids statementdata.Selection) error {
			for _, id := range ids {
				if !allowed[id] {
					return errors.New("proposal_outside_chunk")
				}
				fields[id] = true
			}
			return nil
		}
		for _, row := range p.Rows {
			for _, ids := range []statementdata.Selection{row.OperationDate, row.BookingDate, row.AccountAmount, row.AccountCurrency, row.TransactionAmount, row.TransactionCurrency, row.Description, row.DirectionEvidence} {
				if e := check(ids); e != nil {
					return out, e
				}
			}
			key := Hash(row)
			if !rows[key] {
				out.Rows = append(out.Rows, row)
				rows[key] = true
			}
		}
		for _, pair := range []struct {
			target *statementdata.Selection
			in     statementdata.Selection
		}{{&out.Header.OpeningBookBalance, p.Header.OpeningBookBalance}, {&out.Header.ClosingBookBalance, p.Header.ClosingBookBalance}, {&out.Header.AvailableBalanceObserved, p.Header.AvailableBalanceObserved}, {&out.Header.DebitTurnover, p.Header.DebitTurnover}, {&out.Header.CreditTurnover, p.Header.CreditTurnover}, {&out.Header.Currency, p.Header.Currency}, {&out.Header.OpeningBookDate, p.Header.OpeningBookDate}, {&out.Header.ClosingBookDate, p.Header.ClosingBookDate}} {
			if e := check(pair.in); e != nil {
				return out, e
			}
			if pair.target == &out.Header.Currency && len(*pair.target) > 0 {
				continue
			}
			if e := merge(pair.target, pair.in); e != nil {
				return out, e
			}
		}
		boundValues := map[string]statementdata.Selection{"openingBookBalance": p.Header.OpeningBookBalance, "closingBookBalance": p.Header.ClosingBookBalance, "availableBalanceObserved": p.Header.AvailableBalanceObserved, "debitTurnover": p.Header.DebitTurnover, "creditTurnover": p.Header.CreditTurnover}
		for role, ids := range p.Header.Labels {
			value, knownRole := boundValues[role]
			if !knownRole {
				return out, errors.New("invalid_header_role")
			}
			// An unbound context hint is not a financial field. Validate its
			// references, but never let it claim coverage or displace a label
			// attached to an actual value in another chunk.
			for _, id := range ids {
				if !allowed[id] {
					return out, errors.New("proposal_outside_chunk")
				}
			}
			if len(value) == 0 {
				continue
			}
			if e := check(ids); e != nil {
				return out, e
			}
			target := out.Header.Labels[role]
			if e := merge(&target, ids); e != nil {
				return out, e
			}
			out.Header.Labels[role] = target
		}
		for _, service := range p.Service {
			for _, id := range service.Spans {
				if !allowed[id] {
					return out, errors.New("proposal_outside_chunk")
				}
			}
			out.Service = append(out.Service, service)
		}
	}
	seen := map[string]bool{}
	service := []statementdata.ServiceMapping{}
	for _, group := range out.Service {
		ids := statementdata.Selection{}
		for _, id := range group.Spans {
			if contextIDs[id] && (fields[id] || seen[id]) {
				continue
			}
			ids = append(ids, id)
			seen[id] = true
		}
		if len(ids) > 0 {
			group.Spans = ids
			service = append(service, group)
		}
	}
	out.Service = service
	if len(out.Rows) > statementdata.MaxRows {
		return out, errors.New("statement_row_limit")
	}
	return out, nil
}
