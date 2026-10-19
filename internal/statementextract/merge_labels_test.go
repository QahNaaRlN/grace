package statementextract

import (
	"github.com/QahNaaRlN/grace/internal/statementdata"
	"testing"
)

func TestMergeUnboundContextLabelDoesNotConflictWithBoundHeader(t *testing.T) {
	context := []RedactedSpan{{ID: "label-a"}, {ID: "label-b"}, {ID: "amount"}, {ID: "different-amount"}}
	plan := Plan{Chunks: []Chunk{{Context: context}, {Context: context}}}
	first := statementdata.Proposal{Header: statementdata.HeaderMapping{OpeningBookBalance: statementdata.Selection{"amount"}, Labels: map[string]statementdata.Selection{"openingBookBalance": {"label-a"}}}}
	second := statementdata.Proposal{Header: statementdata.HeaderMapping{Labels: map[string]statementdata.Selection{"openingBookBalance": {"label-b"}}}}
	out, err := MergeProposals(plan, []statementdata.Proposal{first, second})
	if err != nil || len(out.Header.Labels["openingBookBalance"]) != 1 || out.Header.Labels["openingBookBalance"][0] != "label-a" {
		t.Fatal("unbound hint conflicted with actual header", err, out.Header)
	}
	second.Header.OpeningBookBalance = statementdata.Selection{"amount"}
	if _, err = MergeProposals(plan, []statementdata.Proposal{first, second}); err == nil {
		t.Fatal("conflicting bound labels accepted")
	}
	second.Header.Labels["openingBookBalance"] = statementdata.Selection{"label-a"}
	second.Header.OpeningBookBalance = statementdata.Selection{"different-amount"}
	if _, err = MergeProposals(plan, []statementdata.Proposal{first, second}); err == nil {
		t.Fatal("conflicting actual monetary values accepted")
	}
	second.Header.OpeningBookBalance = nil
	second.Header.Labels["openingBookBalance"] = statementdata.Selection{"fabricated"}
	if _, err = MergeProposals(plan, []statementdata.Proposal{first, second}); err == nil {
		t.Fatal("unknown unbound reference accepted")
	}
}
func TestUnboundLabelsNeverClaimCoverageOrRemoveExplicitService(t *testing.T) {
	d := testDocument("19.10.2026", "-12.34", "RUB", "Opening")
	plan, err := BuildPlan(d, testPolicy(), nil)
	if err != nil {
		t.Fatal(err)
	}
	p := statementdata.Proposal{Rows: []statementdata.RowMapping{{OperationDate: statementdata.Selection{"span-0"}, AccountAmount: statementdata.Selection{"span-1"}, AccountCurrency: statementdata.Selection{"span-2"}}}, Header: statementdata.HeaderMapping{Labels: map[string]statementdata.Selection{"openingBookBalance": {"span-3"}}}}
	out, err := MergeProposals(plan, []statementdata.Proposal{p})
	if err != nil {
		t.Fatal(err)
	}
	r, err := statementdata.ValidateProposal(d, out, []int{1}, true, "")
	if err != nil || r.Reconciliation.RowCoverage.Status != "incomplete" {
		t.Fatal("orphan label falsely covered source", r, err)
	}
	p.Service = []statementdata.ServiceMapping{{Spans: statementdata.Selection{"span-3"}, Reason: "Explicit unused heading"}}
	out, err = MergeProposals(plan, []statementdata.Proposal{p})
	if err != nil {
		t.Fatal(err)
	}
	r, err = statementdata.ValidateProposal(d, out, []int{1}, true, "")
	if err != nil || r.Reconciliation.RowCoverage.Status != "matched" || len(r.Reconciliation.Blockers) != 0 {
		t.Fatal("orphan label removed explicit classification", r, err)
	}
}
