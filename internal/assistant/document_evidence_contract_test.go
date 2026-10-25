package assistant

import (
	"context"
	"encoding/json"
	"github.com/QahNaaRlN/grace/internal/statementdata"
	"github.com/QahNaaRlN/grace/internal/statementextract"
	"reflect"
	"strings"
	"testing"
)

func TestSActualGroupedWireCarriesEvidenceContract(t *testing.T) {
	f := &requestPricingFixture{omitRequest: true, image: "0.000000275"}
	groups, err := NewExperimentModelGroups(pricingMember(), pricingGrant(), "offline", f)
	if err != nil {
		t.Fatal(err)
	}
	p, err := groups.Resolve(context.Background(), "statement", 1)
	if err != nil {
		t.Fatal(err)
	}
	schema := statementdata.VisualSchema()
	request, err := DocumentRequest(p, groupSource(t), statementextract.DocumentPrompt, schema, 16000)
	if err != nil {
		t.Fatal(err)
	}
	provider, err := NewGroupedOpenRouterProvider("offline", groups, f)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = provider.Complete(context.Background(), request); err != nil {
		t.Fatal(err)
	}
	response := f.inner.wire["response_format"].(map[string]any)
	actual := response["json_schema"].(map[string]any)
	var expected any
	_ = json.Unmarshal(schema, &expected)
	if f.inner.posts != 1 || actual["strict"] != true || !reflect.DeepEqual(actual["schema"], expected) {
		t.Fatal("actual wire lost schema constraints")
	}
	raw, _ := json.Marshal(f.inner.wire["messages"])
	if !strings.Contains(string(raw), "fractional normalized") || !strings.Contains(string(raw), "empty evidence") || statementextract.DocumentPromptVersion != "statement.document-reading.0056.v3" {
		t.Fatal("actual wire lost evidence instructions")
	}
	route := f.inner.wire["provider"].(map[string]any)
	if route["allow_fallbacks"] != false {
		t.Fatal("fallback enabled")
	}
}
