package statementextract

import (
	"encoding/hex"
	"encoding/json"
	"strconv"

	"github.com/QahNaaRlN/grace/internal/statementdata"
)

// A persisted consent is usable only for this exact complete wire contract.
func validPlan(p Plan) bool { return planShapeValid(p, p.Policy.Valid()) }
func planShapeValid(p Plan, allowed bool) bool {
	if p.PlannerVersion != PlannerVersion || p.AliasVersion != AliasVersion || p.PromptVersion != PromptVersion || p.RedactionVersion != RedactionVersion || p.SchemaHash != Hash(json.RawMessage(ProposalSchema)) || p.Limits != currentLimits() || !allowed || p.MaximumMicros != p.Policy.CallMicros() {
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
		if span.LexicalClass != ClassNumericFragment && span.LexicalClass != ClassOther {
			return false
		}
		if span.LexicalClass == ClassNumericFragment && span.Text != "[number_"+span.ID+"]" && lexicalClass(span.Text) != ClassNumericFragment {
			return false
		}
		if span.LexicalClass == ClassOther && lexicalClass(span.Text) == ClassNumericFragment {
			return false
		}
		wireSeen[span.ID] = true
	}
	return true
}

// CanonicalProposal copies every typed selection; it never edits source text,
// direction, service reasons or the provider's original proposal in place.
func CanonicalProposal(plan Plan, in statementdata.Proposal) (statementdata.Proposal, error) {
	return canonicalProposal(plan, in, validPlan(plan))
}
func canonicalProposal(plan Plan, in statementdata.Proposal, valid bool) (statementdata.Proposal, error) {
	out, _, err := canonicalProposalDetailed(plan, in, valid)
	return out, err
}
