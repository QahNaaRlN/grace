package main

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/QahNaaRlN/grace/internal/statementdata"
	"github.com/QahNaaRlN/grace/internal/statementqualification"
)

func TestSyntheticDiagnosticDoesNotExportValuesReasonsOrBilling(t *testing.T) {
	amount := int64(123456)
	report := statementqualification.Report{ActualMicros: &amount, Calls: []statementqualification.Call{{GenerationID: "private-generation", InputTokens: 9999, CostMicros: 555}}, Traces: []statementqualification.Trace{{Name: "columns", Parts: []statementdata.Proposal{{Header: statementdata.HeaderMapping{Currency: statementdata.Selection{"p1-b2-l3-w4", "synthetic-private-token"}}, Service: []statementdata.ServiceMapping{{Reason: "private-service-reason"}}}}}}}
	raw, e := json.Marshal(safeDiagnostic(report))
	if e != nil {
		t.Fatal(e)
	}
	s := string(raw)
	if !strings.Contains(s, "p1-b2-l3-w4") {
		t.Fatal("safe locator missing")
	}
	for _, secret := range []string{"123456", "9999", "555", "private-generation", "synthetic-private-token", "private-service-reason"} {
		if strings.Contains(s, secret) {
			t.Fatal("diagnostic leaked restricted metadata", secret)
		}
	}
}
