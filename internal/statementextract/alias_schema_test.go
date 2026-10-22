package statementextract

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/QahNaaRlN/grace/internal/statementdata"
)

// Explicit contract oracle: do not discover paths from the generated schema.
type selectionCase struct {
	scope, field    string
	money, currency bool
}

var aliasSelectionCases = []selectionCase{
	{"row", "operationDate", false, false}, {"row", "bookingDate", false, false},
	{"row", "description", false, false}, {"row", "directionEvidence", false, false},
	{"row", "accountCurrency", false, true}, {"row", "transactionCurrency", false, true},
	{"row", "accountAmount", true, false}, {"row", "transactionAmount", true, false},
	{"header", "openingBookDate", false, false}, {"header", "closingBookDate", false, false},
	{"header", "currency", false, true}, {"header", "openingBookBalance", true, false},
	{"header", "closingBookBalance", true, false}, {"header", "availableBalanceObserved", true, false},
	{"header", "debitTurnover", true, false}, {"header", "creditTurnover", true, false},
	{"label", "openingBookBalance", false, false}, {"label", "closingBookBalance", false, false},
	{"label", "availableBalanceObserved", false, false}, {"label", "debitTurnover", false, false},
	{"label", "creditTurnover", false, false}, {"service", "spans", false, false},
}

func schemaObject(t *testing.T, raw json.RawMessage) map[string]any {
	t.Helper()
	var root map[string]any
	if err := json.Unmarshal(raw, &root); err != nil {
		t.Fatal(err)
	}
	return root
}
func resolveSchema(t *testing.T, root map[string]any, node any) map[string]any {
	t.Helper()
	obj := node.(map[string]any)
	for i := 0; i < 8; i++ {
		ref, ok := obj["$ref"].(string)
		if !ok {
			return obj
		}
		if !strings.HasPrefix(ref, "#/$defs/") {
			t.Fatalf("unexpected ref %q", ref)
		}
		obj = root["$defs"].(map[string]any)[strings.TrimPrefix(ref, "#/$defs/")].(map[string]any)
	}
	t.Fatal("reference cycle")
	return nil
}
func selectionSchema(t *testing.T, root map[string]any, tc selectionCase) map[string]any {
	t.Helper()
	props := root["properties"].(map[string]any)
	var node map[string]any
	switch tc.scope {
	case "row":
		node = props["rows"].(map[string]any)["items"].(map[string]any)
	case "header":
		node = props["header"].(map[string]any)
	case "label":
		node = props["header"].(map[string]any)["properties"].(map[string]any)["labels"].(map[string]any)
	case "service":
		node = props["service"].(map[string]any)["items"].(map[string]any)
	default:
		t.Fatal(tc.scope)
	}
	return resolveSchema(t, root, node["properties"].(map[string]any)[tc.field])
}

// A test-only evaluator of the selection fragment, independent of Go's alias
// membership validator. Unexpected keywords fail rather than being ignored.
func schemaAcceptsSelection(t *testing.T, root, array map[string]any, values []any) bool {
	t.Helper()
	if array["type"] != "array" {
		t.Fatal("not an array")
	}
	for key := range array {
		if key != "type" && key != "items" && key != "minItems" && key != "maxItems" && key != "description" {
			t.Fatalf("unsupported array keyword %s", key)
		}
	}
	if n, ok := array["minItems"].(float64); ok && len(values) < int(n) {
		return false
	}
	if n, ok := array["maxItems"].(float64); ok && len(values) > int(n) {
		return false
	}
	item := resolveSchema(t, root, array["items"])
	for key := range item {
		if key != "type" && key != "enum" {
			t.Fatalf("unsupported item keyword %s", key)
		}
	}
	if item["type"] != "string" {
		t.Fatal("not a string item")
	}
	for _, value := range values {
		s, ok := value.(string)
		if !ok {
			return false
		}
		if enum, ok := item["enum"].([]any); ok {
			found := false
			for _, id := range enum {
				if id == s {
					found = true
				}
			}
			if !found {
				return false
			}
		}
	}
	return true
}
func TestAliasSchemaAllSelectionPaths(t *testing.T) {
	p, err := BuildPlan(testDocument("19.10.2026", "-12.34", "RUB", "PRIVATE DESCRIPTION"), testPolicy(), nil)
	if err != nil {
		t.Fatal(err)
	}
	rawSchema := RequestSchema(p.Chunks[0])
	root := schemaObject(t, rawSchema)
	if strings.Count(string(rawSchema), `"s0"`) != 1 || strings.Count(string(rawSchema), `"s1"`) != 2 {
		t.Fatal("alias enum duplicated instead of shared definitions")
	}
	if len(aliasSelectionCases) != 22 {
		t.Fatal("incomplete oracle")
	}
	for _, tc := range aliasSelectionCases {
		t.Run(tc.scope+"/"+tc.field, func(t *testing.T) {
			array := selectionSchema(t, root, tc)
			want := []any{"s0", "s1", "s2", "s3"}
			if tc.money {
				want = []any{"s1"}
			}
			if !reflect.DeepEqual(resolveSchema(t, root, array["items"])["enum"], want) {
				t.Fatal("wrong membership", array)
			}
			if !schemaAcceptsSelection(t, root, array, []any{}) {
				t.Fatal("empty selection rejected")
			}
			for _, id := range []any{"s0", "s1", "s2", "s3", "szz", "S0", "s0 ", "span-0", "[number_s1]", 123, nil} {
				allowed := false
				for _, v := range want {
					if v == id {
						allowed = true
					}
				}
				if schemaAcceptsSelection(t, root, array, []any{id}) != allowed {
					t.Fatalf("wrong acceptance for %v", id)
				}
			}
			if tc.currency && schemaAcceptsSelection(t, root, array, []any{"s0", "s1"}) {
				t.Fatal("currency cardinality widened")
			}
		})
	}
	props := root["properties"].(map[string]any)
	for _, obj := range []map[string]any{root, props["rows"].(map[string]any)["items"].(map[string]any), props["header"].(map[string]any), props["header"].(map[string]any)["properties"].(map[string]any)["labels"].(map[string]any), props["service"].(map[string]any)["items"].(map[string]any)} {
		if obj["additionalProperties"] != false || len(obj["required"].([]any)) != len(obj["properties"].(map[string]any)) {
			t.Fatal("required/object contract widened")
		}
	}
	row := props["rows"].(map[string]any)["items"].(map[string]any)["properties"].(map[string]any)
	if !reflect.DeepEqual(row["direction"].(map[string]any)["enum"], []any{"", "debit", "credit"}) || props["service"].(map[string]any)["items"].(map[string]any)["properties"].(map[string]any)["reason"].(map[string]any)["type"] != "string" {
		t.Fatal("non-reference strings changed")
	}
}
func TestAliasSchemaEmptySetsAndDeliveredOrder(t *testing.T) {
	for _, c := range []Chunk{{}, {Spans: []RedactedSpan{{ID: "s0", Text: "[number_s0]", LexicalClass: ClassOther}}}} {
		root := schemaObject(t, RequestSchema(c))
		for _, tc := range aliasSelectionCases {
			array := selectionSchema(t, root, tc)
			if !schemaAcceptsSelection(t, root, array, []any{}) {
				t.Fatal("empty rejected", tc)
			}
			if (len(c.Spans) == 0 || tc.money) && schemaAcceptsSelection(t, root, array, []any{"s0"}) {
				t.Fatal("empty set permits a reference", tc)
			}
		}
		if strings.Contains(string(RequestSchema(c)), `"enum":[]`) {
			t.Fatal("invalid empty enum")
		}
	}
	c := Chunk{Context: []RedactedSpan{{ID: "s2", LexicalClass: ClassOther}, {ID: "s1", LexicalClass: ClassNumericFragment}}, Spans: []RedactedSpan{{ID: "s1", LexicalClass: ClassNumericFragment}, {ID: "s0", LexicalClass: ClassOther}}}
	root := schemaObject(t, RequestSchema(c))
	defs := root["$defs"].(map[string]any)
	if !reflect.DeepEqual(defs["alias"].(map[string]any)["enum"], []any{"s2", "s1", "s0"}) || !reflect.DeepEqual(defs["moneySelection"].(map[string]any)["items"].(map[string]any)["enum"], []any{"s1"}) {
		t.Fatal("unstable or duplicate IDs")
	}
}
func TestAliasSchemaIsolationAndGoGuards(t *testing.T) {
	template := string(ProposalSchema)
	p, err := BuildPlan(testDocument("RUB", "-1.00"), testPolicy(), nil)
	if err != nil {
		t.Fatal(err)
	}
	before := Hash(p.Chunks[0])
	expected := string(RequestSchema(p.Chunks[0]))
	var wg sync.WaitGroup
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			c := Chunk{Spans: []RedactedSpan{{ID: fmt.Sprintf("only_%d", i), LexicalClass: ClassOther}}}
			for j := 0; j < 8; j++ {
				if string(RequestSchema(p.Chunks[0])) != expected || !strings.Contains(string(RequestSchema(c)), fmt.Sprintf(`"only_%d"`, i)) {
					t.Error("cross-document schema mutation")
				}
			}
		}(i)
	}
	wg.Wait()
	if string(ProposalSchema) != template || Hash(p.Chunks[0]) != before {
		t.Fatal("input/template mutated")
	}
	for _, tc := range aliasSelectionCases {
		for _, ids := range [][]string{{"szz"}, {"s1", "s1"}, {"s0"}} {
			if ids[0] == "s0" && !tc.money {
				continue
			}
			fields := map[string]any{tc.field: ids}
			tree := map[string]any{}
			switch tc.scope {
			case "row":
				tree["rows"] = []any{fields}
			case "header":
				tree["header"] = fields
			case "label":
				tree["header"] = map[string]any{"labels": fields}
			case "service":
				tree["service"] = []any{fields}
			}
			raw, _ := json.Marshal(tree)
			var proposal statementdata.Proposal
			if json.Unmarshal(raw, &proposal) != nil {
				t.Fatal("bad test input")
			}
			if _, err := CanonicalProposal(p, proposal); err == nil {
				t.Fatal("schema-ignoring provider escaped Go guard", tc, ids)
			}
		}
	}
}
func TestAliasSchemaInvalidatesV2Plans(t *testing.T) {
	p, err := BuildPlan(testDocument("RUB", "-1.00"), testPolicy(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if PlannerVersion != "statement-document-atom.v3.alias-enum" {
		t.Fatal("identity not advanced")
	}
	old := p
	old.PlannerVersion = "statement-document-atom.v2.lexical-grounding"
	old.SchemaHash = "52a36f2b6159b7e9a3e2fc0509ba8c688142eebd12b60be1556d7165da56f0bf"
	if Hash(old) == Hash(p) || currentPlanContract(old) || old.SchemaHash == p.SchemaHash {
		t.Fatal("old contract accepted")
	}
	// The frozen v2 binary checks both its literal planner version and schema hash.
	if p.PlannerVersion == old.PlannerVersion || p.SchemaHash == old.SchemaHash {
		t.Fatal("rollback binary could reuse consent")
	}
}
