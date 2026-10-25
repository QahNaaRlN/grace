package statementdata

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestSEvidenceSchemaClosesObservedProtocolGaps(t *testing.T) {
	if bytes.Contains(VisualSchema(), []byte(`"$ref"`)) || bytes.Contains(VisualSchema(), []byte(`"$defs"`)) {
		t.Fatal("inline compatibility contract lost")
	}
	var root map[string]any
	if err := json.Unmarshal(VisualSchema(), &root); err != nil {
		t.Fatal(err)
	}
	row := root["properties"].(map[string]any)["rows"].(map[string]any)["items"].(map[string]any)["properties"].(map[string]any)
	anchor := row["anchor"].(map[string]any)
	coords := anchor["items"].(map[string]any)["properties"].(map[string]any)["box"].(map[string]any)["items"].(map[string]any)
	minimum, minOK := coords["minimum"].(float64)
	maximum, maxOK := coords["maximum"].(float64)
	if !minOK || !maxOK || minimum != 0 || maximum != 1 {
		t.Fatal("schema permits observed integer coordinate scale")
	}
	fields := row["fields"].(map[string]any)["properties"].(map[string]any)
	for _, k := range VisualRowFields {
		alternatives := fields[k].(map[string]any)["anyOf"].([]any)
		if alternatives[0].(map[string]any)["type"] != "null" {
			t.Fatal("uncited null forbidden", k)
		}
		evidence := alternatives[1].(map[string]any)["properties"].(map[string]any)["evidence"].(map[string]any)
		for _, quotes := range []map[string]any{anchor, evidence} {
			low, ok := quotes["minItems"].(float64)
			high, ok2 := quotes["maxItems"].(float64)
			if !ok || !ok2 || low != 1 || high != 8 {
				t.Fatal("schema permits empty/unbounded evidence", k)
			}
		}
	}
	// Both independent classes from Q4.2 remain refused by the unchanged validator.
	for _, which := range []string{"integer_box", "empty_evidence", "both"} {
		v := visualFixture()
		if which != "empty_evidence" {
			v.Rows[0].Anchor[0].Box = [4]float64{100, 200, 300, 400}
		}
		if which != "integer_box" {
			v.Rows[0].Fields["description"].Evidence = []ImageQuote{}
		}
		if v.Validate(1) == nil {
			t.Fatal("observed invalid class accepted", which)
		}
	}
	v := visualFixture()
	v.Rows[0].Fields["transactionAmount"] = nil
	v.Rows[0].Fields["transactionCurrency"] = nil
	if err := v.Validate(1); err != nil {
		t.Fatal("fractional/null control", err)
	}
}
