// Offline structural controls for the saved Q4.2 diagnosis; no HTTP/SQL/keys.
package main

import (
	"encoding/json"
	"fmt"
	"github.com/QahNaaRlN/grace/internal/statementdata"
	"github.com/QahNaaRlN/grace/internal/statementextract"
	"os"
)

func fixture(scaled, uncited bool) statementdata.VisualReading {
	box := [4]float64{100, 200, 300, 400}
	if scaled {
		for i := range box {
			box[i] /= 1000
		}
	}
	quote := func() []statementdata.ImageQuote {
		return []statementdata.ImageQuote{{Page: 1, Quote: "synthetic", Box: box}}
	}
	field := func() *statementdata.VisualField {
		return &statementdata.VisualField{Value: "synthetic", Evidence: quote()}
	}
	v := statementdata.VisualReading{Header: map[string]*statementdata.VisualField{}, Rows: []statementdata.VisualRow{}}
	for _, k := range statementdata.VisualHeaderFields {
		v.Header[k] = field()
	}
	v.Header["availableBalanceObserved"] = nil
	for i := 0; i < 2; i++ {
		row := statementdata.VisualRow{Anchor: quote(), Fields: map[string]*statementdata.VisualField{}, Issues: []string{}}
		for _, k := range statementdata.VisualRowFields {
			row.Fields[k] = field()
		}
		v.Rows = append(v.Rows, row)
	}
	if uncited {
		for _, k := range []string{"transactionAmount", "transactionCurrency"} {
			v.Rows[1].Fields[k].Evidence = []statementdata.ImageQuote{}
		}
	}
	return v
}
func result(v statementdata.VisualReading) map[string]bool {
	raw, _ := json.Marshal(v)
	var decoded statementdata.VisualReading
	decodedOK := statementextract.DirectJSON(raw, &decoded) == nil
	return map[string]bool{"directJSONPassed": decodedOK, "visualValidationPassed": decodedOK && decoded.Validate(1) == nil}
}
func main() {
	original := fixture(false, true)
	scaled := fixture(true, true)
	nulled := fixture(true, true)
	nulled.Rows[1].Fields["transactionAmount"] = nil
	nulled.Rows[1].Fields["transactionCurrency"] = nil
	r := map[string]any{
		"version":       "document-reading.q42.offline-structural-probe.v1",
		"providerCalls": 0, "sqlCalls": 0, "usesPrivatePayload": false,
		"scope":                        "synthetic structural controls, not private response replay or quality scoring",
		"integerBoxesAndUncitedFields": result(original),
		"scaledBoxesUncitedFields":     result(scaled),
		"integerBoxesAllFieldsCited":   result(fixture(false, false)),
		"scaledBoxesUncitedFieldsNull": result(nulled),
		"scaledBoxesAllFieldsCited":    result(fixture(true, false)),
	}
	var schema map[string]any
	if json.Unmarshal(statementdata.VisualSchema(), &schema) != nil {
		os.Exit(1)
	}
	props := schema["properties"].(map[string]any)
	row := props["rows"].(map[string]any)["items"].(map[string]any)["properties"].(map[string]any)
	anchors := row["anchor"].(map[string]any)
	coords := anchors["items"].(map[string]any)["properties"].(map[string]any)["box"].(map[string]any)["items"].(map[string]any)
	_, min := coords["minimum"]
	_, max := coords["maximum"]
	fields := row["fields"].(map[string]any)["properties"].(map[string]any)
	field := fields["transactionAmount"].(map[string]any)["anyOf"].([]any)[1].(map[string]any)
	evidence := field["properties"].(map[string]any)["evidence"].(map[string]any)
	_, evidenceMin := evidence["minItems"]
	_, anchorMin := anchors["minItems"]
	r["actualVisualSchema"] = map[string]bool{"boxCoordinateMinimumDeclared": min, "boxCoordinateMaximumDeclared": max, "nonNullFieldEvidenceMinItemsDeclared": evidenceMin, "anchorMinItemsDeclared": anchorMin}
	for _, name := range []string{"integerBoxesAndUncitedFields", "scaledBoxesUncitedFields", "integerBoxesAllFieldsCited"} {
		got := r[name].(map[string]bool)
		if !got["directJSONPassed"] || got["visualValidationPassed"] {
			fmt.Fprintln(os.Stderr, "negative control failed")
			os.Exit(1)
		}
	}
	for _, name := range []string{"scaledBoxesUncitedFieldsNull", "scaledBoxesAllFieldsCited"} {
		got := r[name].(map[string]bool)
		if !got["directJSONPassed"] || !got["visualValidationPassed"] {
			fmt.Fprintln(os.Stderr, "positive control failed")
			os.Exit(1)
		}
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if enc.Encode(r) != nil {
		os.Exit(1)
	}
}
