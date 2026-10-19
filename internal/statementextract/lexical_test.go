package statementextract

import (
	"encoding/json"
	"github.com/QahNaaRlN/grace/internal/statementdata"
	"strings"
	"testing"
)

func TestLexicalShapeIsNotFinancialMeaningOrPlaceholderText(t *testing.T) {
	for _, s := range []string{"1", "12345678901234567890", "1 234", "1\u00a0234", "1\u202f234", "-1 234,56", "−12.34", "+100", "+", "-", "−", "0.1"} {
		if lexicalClass(s) != ClassNumericFragment {
			t.Errorf("numeric shape lost %q", s)
		}
	}
	for _, s := range []string{"SYNTHETIC", "RUB", "19.10.2026", "2026-10-19", "10:30", "1e3", "(7999)", "+7(999)1234567", "[number_s0]", "12 RUB", "1 23", strings.Repeat("1", 81)} {
		if lexicalClass(s) != ClassOther {
			t.Errorf("non-numeric shape became money %q", s)
		}
	}
}
func TestSourceOrderAliasesAndHiddenClassesRemainAuthoritative(t *testing.T) {
	d := testDocument("[number_s0]", "40702", "1\u00a0234", "1\u202f234", "-12.34", "RUB")
	d.Spans[0].ID = "z-source-first"
	d.Spans[1].ID = "a-source-second"
	before := d.Hash()
	p, e := BuildPlan(d, testPolicy(), []string{"z-source-first", "a-source-second", "span-2", "span-3"})
	if e != nil {
		t.Fatal(e)
	}
	if p.Aliases["s0"] != "z-source-first" || p.Aliases["s1"] != "a-source-second" || !validPlan(p) || d.Hash() != before {
		t.Fatal("source order/hash changed", p.Aliases, e)
	}
	for _, i := range []int{1, 2, 3} {
		s := p.Chunks[0].Spans[i]
		if s.LexicalClass != ClassNumericFragment || s.Text != "[number_"+s.ID+"]" {
			t.Fatal("hidden integer/group exposed", s.ID)
		}
	}
	if p.Chunks[0].Spans[0].Text != "[number_s0]" || p.Chunks[0].Spans[0].LexicalClass != ClassOther {
		t.Fatal("literal placeholder gained authority")
	}
	schema := string(RequestSchema(p.Chunks[0]))
	var parsed map[string]any
	if json.Unmarshal([]byte(schema), &parsed) != nil {
		t.Fatal("schema")
	}
	items := parsed["$defs"].(map[string]any)["moneySelection"].(map[string]any)["items"].(map[string]any)["enum"].([]any)
	allowed := map[string]bool{}
	for _, v := range items {
		allowed[v.(string)] = true
	}
	if allowed["s0"] || allowed["s5"] || !allowed["s1"] || !allowed["s4"] {
		t.Fatal("non-authoritative money candidates", allowed)
	}
	for _, field := range []string{"row", "header"} {
		in := statementdata.Proposal{}
		if field == "row" {
			in.Rows = []statementdata.RowMapping{{AccountAmount: statementdata.Selection{"s0"}}}
		} else {
			in.Header.DebitTurnover = statementdata.Selection{"s0"}
		}
		if _, e := CanonicalProposal(p, in); e == nil {
			t.Fatal("ignored schema permitted text-as-money", field)
		}
	}
	in := statementdata.Proposal{Rows: []statementdata.RowMapping{{AccountAmount: statementdata.Selection{"s1"}, Description: statementdata.Selection{"s0"}}}}
	if _, e := CanonicalProposal(p, in); e != nil {
		t.Fatal("integer monetary candidate or description blocked", e)
	}
	raw, _ := json.Marshal(Messages(p.Chunks[0]))
	if strings.Contains(string(raw), "lexicalClass") || strings.Contains(string(raw), "40702") || strings.Contains(string(raw), "a-source-second") {
		t.Fatal("wire leaked metadata/value/map")
	}
	old := p
	old.AliasVersion = "statement-alias.lex-base36.v1"
	if validPlan(old) {
		t.Fatal("S1 consent accepted")
	}
}
