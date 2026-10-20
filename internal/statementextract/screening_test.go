package statementextract

import (
	"github.com/QahNaaRlN/grace/internal/statementdata"
	"testing"
)

func TestScreeningCannotAuthorizeProductConsent(t *testing.T) {
	d := testDocument("19.10.2026", "-1.00", "RUB")
	p := ScreeningPolicy()
	if p.Valid() {
		t.Fatal("candidate became product policy")
	}
	if _, e := BuildPlan(d, p, nil); e == nil {
		t.Fatal("product planner accepted candidate")
	}
	plan, e := BuildScreeningPlan(d, p)
	if e != nil || plan.Policy != p || plan.MaximumMicros != 53900 {
		t.Fatal(plan, e)
	}
	if validPlan(plan) {
		t.Fatal("candidate persisted consent accepted")
	}
	wire := statementdata.Proposal{Rows: []statementdata.RowMapping{{AccountAmount: statementdata.Selection{"s1"}}}}
	if _, e = CanonicalProposal(plan, wire); e == nil {
		t.Fatal("product canonicalizer accepted candidate")
	}
	if _, e = CanonicalScreeningProposal(plan, wire); e != nil {
		t.Fatal(e)
	}
	plan.PromptVersion = "old"
	if _, e = CanonicalScreeningProposal(plan, wire); e == nil {
		t.Fatal("stale screening plan accepted")
	}
	for _, change := range []func(*Policy){func(p *Policy) { p.Model = "google/gemini-3.8-flash" }, func(p *Policy) { p.Provider = "google-ai-studio" }, func(p *Policy) { p.InputMicrosPerMillion++ }, func(p *Policy) { p.OutputMicrosPerMillion++ }} {
		p := ScreeningPolicy()
		change(&p)
		if _, e = BuildScreeningPlan(d, p); e == nil {
			t.Fatal("unapproved experiment", p)
		}
	}
}
