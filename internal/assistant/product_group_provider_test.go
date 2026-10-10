package assistant

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/QahNaaRlN/grace/internal/receiptdrafts"
)

type groupMeter struct {
	denied   bool
	reserved int
	settled  int
	unknown  bool
}

func (m *groupMeter) Reserve(_ context.Context, _ ModelPolicy, _ int, _ int, purpose string) (string, error) {
	if m.denied {
		return "", errors.New("budget_exhausted")
	}
	if purpose != "jev_category" {
		return "", errors.New("invalid_purpose")
	}
	m.reserved++
	return "synthetic-reservation", nil
}
func (m *groupMeter) Settle(_ context.Context, _ string, response *LLMResponse) error {
	m.settled++
	m.unknown = response == nil
	return nil
}
func TestProductGroupsBatchOnlyNamesAndPreserveUnknownBilling(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		var in struct {
			State     map[string]json.RawMessage `json:"state"`
			Questions map[string]RoutingQuestion `json:"questions"`
		}
		if json.NewDecoder(r.Body).Decode(&in) != nil || len(in.State) != 1 || in.State["product_names"] == nil || len(in.Questions) != 1 {
			t.Error("unexpected or excessive context")
		}
		w.Write([]byte(`{"model":"jev-1.13.0","answers":{"0":{"type":"choice","choice":"milk","confidence":0.95,"probabilities":{"milk":0.98,"unknown":0.02}}},"usage":{"input_tokens":100,"output_tokens":10}}`))
	}))
	defer server.Close()
	policy := policyFixture()
	policy.Model = "jev-1.13.0"
	policy.Provider = "typesafe"
	meter := &groupMeter{}
	provider, err := NewTypeSafeGroupSuggester("synthetic-test-key", policy, map[string]string{"milk": "Молоко"}, 0.9, meter)
	if err != nil {
		t.Fatal(err)
	}
	provider.endpoint = server.URL
	provider.client = server.Client()
	name := "Молоко"
	out, err := provider.Suggest(context.Background(), []receiptdrafts.Item{{Name: &name}, {Name: nil}})
	if err != nil || len(out) != 2 || out[0] == nil || out[0].Group != "milk" || out[1] != nil || meter.reserved != 1 || meter.settled != 1 || !meter.unknown || calls.Load() != 1 {
		t.Fatal(out, err, meter, calls.Load())
	}
	meter.denied = true
	if _, err = provider.Suggest(context.Background(), []receiptdrafts.Item{{Name: &name}}); err == nil || calls.Load() != 1 {
		t.Fatal("budget did not block transmission", err)
	}
}
