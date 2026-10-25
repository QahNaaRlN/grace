package statementdata

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestVPortableSchemaKeepsSemanticChecksInGo(t *testing.T) {
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
	if coords["type"] != "number" || coords["minimum"] != nil || coords["maximum"] != nil {
		t.Fatal("portable numeric shape lost")
	}
	box := anchor["items"].(map[string]any)["properties"].(map[string]any)["box"].(map[string]any)
	if box["minItems"] != float64(4) || box["maxItems"] != float64(4) {
		t.Fatal("four coordinate shape lost")
	}
	fields := row["fields"].(map[string]any)["properties"].(map[string]any)
	for _, k := range VisualRowFields {
		alternatives := fields[k].(map[string]any)["anyOf"].([]any)
		if alternatives[0].(map[string]any)["type"] != "null" {
			t.Fatal("uncited null forbidden", k)
		}
		evidence := alternatives[1].(map[string]any)["properties"].(map[string]any)["evidence"].(map[string]any)
		for _, quotes := range []map[string]any{anchor, evidence} {
			if quotes["type"] != "array" || quotes["minItems"] != nil || quotes["maxItems"] != nil {
				t.Fatal("portable quote shape lost", k)
			}
		}
	}
	// Semantic acceptance is independent of model schema; the same Go validator refuses both classes.
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
