package statementextract

import (
	"encoding/json"
	"math/rand"
	"reflect"
	"strings"
	"testing"

	"github.com/QahNaaRlN/grace/internal/statementdata"
)

func diagnosticPlan(t *testing.T) Plan {
	t.Helper()
	d := statementdata.Document{Extractor: statementdata.ExtractorVersion, Pages: []statementdata.Page{{Number: 1, Width: 600, Height: 800}}, Spans: []statementdata.Span{}}
	for i, text := range []string{"123.45", "RUB", "SYNTHETIC"} {
		d.Spans = append(d.Spans, statementdata.Span{ID: []string{"p1-b1-l1-w1", "p1-b1-l1-w2", "p1-b1-l1-w3"}[i], Page: 1, Block: 1, Line: 1, Word: i + 1, Text: text, Box: statementdata.Box{XMin: float64(i*30 + 1), YMin: 1, XMax: float64(i*30 + 20), YMax: 10}})
	}
	p, e := BuildComparisonPlan(d, ComparisonCandidates()[0].Policy)
	if e != nil {
		t.Fatal(e)
	}
	return p
}
func TestCanonicalDiagnosticBaselineAndPriority(t *testing.T) {
	cases := []struct {
		name, code, field, scope string
		mutate                   func(*Plan, *statementdata.Proposal)
	}{
		{"invalid-plan", "invalid_plan", "plan", "plan", func(p *Plan, in *statementdata.Proposal) {
			p.PlannerVersion = "old"
			in.Header.DebitTurnover = statementdata.Selection{"PRIVATE"}
		}},
		{"unknown-money", "unknown_alias", "debitTurnover", "header", func(_ *Plan, in *statementdata.Proposal) {
			in.Header.DebitTurnover = statementdata.Selection{"PRIVATE"}
		}},
		{"text-money", "non_numeric_money_selection", "accountAmount", "row", func(_ *Plan, in *statementdata.Proposal) {
			in.Rows = []statementdata.RowMapping{{AccountAmount: statementdata.Selection{"s1"}}}
		}},
		{"numeric-duplicate", "duplicate_alias", "accountAmount", "row", func(_ *Plan, in *statementdata.Proposal) {
			in.Rows = []statementdata.RowMapping{{AccountAmount: statementdata.Selection{"s0", "s0"}}}
		}},
		{"unknown-description", "unknown_alias", "description", "row", func(_ *Plan, in *statementdata.Proposal) {
			in.Rows = []statementdata.RowMapping{{Description: statementdata.Selection{"PRIVATE"}}}
		}},
		{"service", "unknown_alias", "spans", "service", func(_ *Plan, in *statementdata.Proposal) {
			in.Service = []statementdata.ServiceMapping{{Spans: statementdata.Selection{"PRIVATE"}}}
		}},
		{"priority-money-first", "non_numeric_money_selection", "closingBookBalance", "header", func(_ *Plan, in *statementdata.Proposal) {
			in.Header.ClosingBookBalance = statementdata.Selection{"s1"}
			in.Rows = []statementdata.RowMapping{{Description: statementdata.Selection{"PRIVATE"}}}
		}},
		{"header-label-stable", "unknown_alias", "labels", "header_label", func(_ *Plan, in *statementdata.Proposal) {
			in.Header.Labels = map[string]statementdata.Selection{"PRIVATE-Z": {"s0", "s0"}, "PRIVATE-A": {"missing"}}
		}},
		{"valid-numeric", "", "", "", func(_ *Plan, in *statementdata.Proposal) {
			in.Rows = []statementdata.RowMapping{{AccountAmount: statementdata.Selection{"s0"}}}
		}},
		{"cross-field-duplicate", "", "", "", func(_ *Plan, in *statementdata.Proposal) {
			in.Header.DebitTurnover = statementdata.Selection{"s0"}
			in.Header.CreditTurnover = statementdata.Selection{"s0"}
		}},
		{"unknown-role-direction", "", "", "", func(_ *Plan, in *statementdata.Proposal) {
			in.Header.Labels = map[string]statementdata.Selection{"PRIVATE": {"s0"}}
			in.Rows = []statementdata.RowMapping{{Direction: "PRIVATE", Description: statementdata.Selection{}}}
		}},
		{"nil-empty", "", "", "", func(_ *Plan, in *statementdata.Proposal) {
			in.Rows = []statementdata.RowMapping{{AccountAmount: nil, Description: statementdata.Selection{}}}
			in.Service = []statementdata.ServiceMapping{{Spans: statementdata.Selection{}, Reason: "PRIVATE"}}
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := diagnosticPlan(t)
			in := statementdata.Proposal{}
			tc.mutate(&p, &in)
			before, _ := json.Marshal(in)
			want, werr := baselineCanonicalF4(p, in, planShapeValid(p, ComparisonPolicyValid(p.Policy)))
			out, diag, err := CanonicalComparisonProposalDiagnostic(p, in)
			if (err != nil) != (werr != nil) || !reflect.DeepEqual(out, want) {
				t.Fatal("f4fc56a behavior changed")
			}
			if err != nil && (err.Error() != werr.Error() || err.Error() != "provider_protocol") {
				t.Fatal("legacy error changed")
			}
			legacy, lerr := CanonicalComparisonProposal(p, in)
			if !reflect.DeepEqual(legacy, out) || (lerr != nil) != (err != nil) {
				t.Fatal("legacy wrapper drift")
			}
			if tc.code == "" {
				if diag != nil {
					t.Fatal("PASS diagnostic")
				}
			} else {
				if diag == nil || !diag.Valid() || diag.Code != tc.code || diag.Field != tc.field || diag.Scope != tc.scope {
					t.Fatal("wrong closed diagnostic", diag)
				}
				for i := 0; i < 30; i++ {
					_, again, _ := CanonicalComparisonProposalDiagnostic(p, in)
					if !reflect.DeepEqual(diag, again) {
						t.Fatal("nondeterministic label selection")
					}
				}
				raw, _ := json.Marshal(diag)
				if strings.Contains(string(raw), "PRIVATE") {
					t.Fatal("raw alias or role leaked")
				}
			}
			after, _ := json.Marshal(in)
			if string(before) != string(after) {
				t.Fatal("source proposal mutated")
			}
		})
	}
}
func TestCanonicalDiagnosticDifferentialGeneratedCases(t *testing.T) {
	rng := rand.New(rand.NewSource(113))
	pool := []string{"s0", "s1", "s2", "PRIVATE", ""}
	selection := func() statementdata.Selection {
		if rng.Intn(4) == 0 {
			return nil
		}
		s := statementdata.Selection{}
		for n := rng.Intn(5); n > 0; n-- {
			s = append(s, pool[rng.Intn(len(pool))])
		}
		return s
	}
	for i := 0; i < 500; i++ {
		p := diagnosticPlan(t)
		if i%11 == 0 {
			p.AliasVersion = "old"
		}
		in := statementdata.Proposal{Rows: []statementdata.RowMapping{{AccountAmount: selection(), Description: selection(), OperationDate: selection(), Direction: "arbitrary"}}, Header: statementdata.HeaderMapping{DebitTurnover: selection(), Labels: map[string]statementdata.Selection{"PRIVATE": selection(), "debitTurnover": selection()}}, Service: []statementdata.ServiceMapping{{Spans: selection(), Reason: "PRIVATE"}}}
		want, werr := baselineCanonicalF4(p, in, planShapeValid(p, ComparisonPolicyValid(p.Policy)))
		got, d, err := CanonicalComparisonProposalDiagnostic(p, in)
		if !reflect.DeepEqual(want, got) || (werr != nil) != (err != nil) || (err != nil && (!d.Valid() || err.Error() != werr.Error())) {
			t.Fatal("baseline differential mismatch", i)
		}
	}
}
