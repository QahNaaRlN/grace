package assistant

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/QahNaaRlN/grace/internal/statementextract"
)

type statementProviderStub struct {
	request LLMRequest
	content json.RawMessage
}

func (p *statementProviderStub) Complete(_ context.Context, in LLMRequest) (LLMResponse, error) {
	p.request = in
	return LLMResponse{ID: "synthetic-generation", Content: p.content, CostMicros: 10, PromptTokens: 100, CompletionTokens: 100}, nil
}
func TestStatementReasonerUsesTextOnlyBoundedTransportAndRetainsBillingOnBadJSON(t *testing.T) {
	p := &statementProviderStub{content: json.RawMessage(`{"rows":[],"header":{},"service":[],"amountMinor":1}`)}
	r := &StatementReasoner{Policy: ModelPolicy{Model: "google/gemini-2.5-flash", Provider: "google-vertex/eu", ProviderName: "Google", InputMicrosPerMillion: 300000, OutputMicrosPerMillion: 2500000, VerifiedAt: time.Now().Add(-time.Minute)}, Provider: p}
	policy, e := r.CurrentPolicy(context.Background())
	if e != nil {
		t.Fatal(e)
	}
	_, billing, e := r.Complete(context.Background(), policy, statementextract.Chunk{Index: 1, Spans: []statementextract.RedactedSpan{{ID: "span", Page: 1, Text: "[text_span]"}}})
	if e == nil || billing == nil || billing.CostMicros != 10 {
		t.Fatal("malformed output became free", billing, e)
	}
	if p.request.Policy.Vision || p.request.InputTokenUpperBound != 10000 || p.request.MaxOutputTokens != 2000 || len(p.request.Messages) != 2 || !json.Valid(p.request.Schema) {
		t.Fatal(p.request)
	}
	r.Policy.Provider = "google-vertex"
	if _, e = r.CurrentPolicy(context.Background()); e == nil {
		t.Fatal("silently accepted generic provider alias")
	}
}
