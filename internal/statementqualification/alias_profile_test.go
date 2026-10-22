package statementqualification

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/QahNaaRlN/grace/internal/statementextract"
)

func TestAliasSchemaProfileRejectsPreviousQualification(t *testing.T) {
	for _, field := range []string{"planner", "schema"} {
		for _, screening := range []bool{false, true} {
			r := validReportFixture()
			validate := ValidateReport
			if screening {
				r = screeningReportFixture()
				validate = ValidateScreeningReport
			}
			if validate(r, time.Now()) != nil {
				t.Fatal("invalid current oracle")
			}
			if field == "planner" {
				r.Profile.Planner = "statement-document-atom.v2.lexical-grounding"
			} else {
				r.Profile.Schema = "52a36f2b6159b7e9a3e2fc0509ba8c688142eebd12b60be1556d7165da56f0bf"
			}
			r.Digest = reportDigest(r)
			if validate(r, time.Now()) == nil {
				t.Fatal("old qualification profile accepted", field, screening)
			}
		}
	}
}
func TestAliasSchemaComparisonRequestIdentity(t *testing.T) {
	rs, _, _ := comparisonFakes()
	report, plans, err := comparisonPrepare(context.Background(), comparisonMinimal(), rs)
	if err != nil || len(plans) != 6 {
		t.Fatal(err)
	}
	for i, p := range plans {
		request := struct {
			Policy   statementextract.Policy
			Messages []map[string]string
			Schema   any
		}{p.Policy, statementextract.Messages(p.Chunks[0]), statementextract.RequestSchema(p.Chunks[0])}
		if report.Tuples[i].RequestHash != statementextract.Hash(request) {
			t.Fatal("actual schema absent from request hash")
		}
		var schema map[string]any
		if json.Unmarshal(statementextract.RequestSchema(p.Chunks[0]), &schema) != nil {
			t.Fatal("schema")
		}
		// Remove membership only, retaining all other request identity fields.
		delete(schema["$defs"].(map[string]any)["alias"].(map[string]any), "enum")
		raw, _ := json.Marshal(schema)
		request.Schema = json.RawMessage(raw)
		if report.Tuples[i].RequestHash == statementextract.Hash(request) {
			t.Fatal("enum mutation not bound")
		}
		if report.Tuples[i].Evidence.PlanHash != statementextract.Hash(p) || report.Profile.Planner != statementextract.PlannerVersion || report.Profile.Schema != statementextract.Hash(statementextract.ProposalSchema) {
			t.Fatal("profile/plan identity missing")
		}
	}
}
