package assistant

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"github.com/QahNaaRlN/grace/internal/statementdata"
	"github.com/QahNaaRlN/grace/internal/statementextract"
	"io"
	"net/http"
	"sync"
	"time"
)

// ComparisonReasoner cannot be configured by an HTTP request or owner document.
// Only the fixed synthetic operator CLI uses it; product constructors reject it.
type ComparisonReasoner struct {
	mu        sync.Mutex
	candidate statementextract.ComparisonCandidate
	provider  *OpenRouterProvider
	client    HTTPDoer
	base      string
	proof     *statementextract.ComparisonIdentityProof
}

func NewStatementComparisonReasoner(key, id string) (*ComparisonReasoner, error) {
	c, ok := statementextract.ComparisonCandidateByID(id)
	if !ok {
		return nil, errors.New("comparison_policy_invalid")
	}
	p, e := NewStatementOpenRouterProvider(key, nil)
	if e != nil {
		return nil, e
	}
	p.comparisonID = id
	return &ComparisonReasoner{candidate: c, provider: p, client: &http.Client{Timeout: 5 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}, base: "https://openrouter.ai/api/v1/"}, nil
}
func (r *ComparisonReasoner) CurrentPolicy(ctx context.Context) (statementextract.Policy, error) {
	if _, e := r.currentProof(ctx); e != nil {
		return statementextract.Policy{}, e
	}
	return r.candidate.Policy, nil
}
func (r *ComparisonReasoner) Complete(ctx context.Context, approved statementextract.Policy, chunk statementextract.Chunk) (statementdata.Proposal, *statementextract.Billing, error) {
	var proposal statementdata.Proposal
	proof, e := r.currentProof(ctx)
	if e != nil {
		return proposal, nil, e
	}
	if approved != r.candidate.Policy || statementextract.InputBytes(chunk) > statementextract.MaxInputBytes || chunk.Index != 1 || len(chunk.Context) != 0 || len(chunk.Spans) == 0 || len(chunk.Spans) > statementextract.MaxPlanSpans {
		return proposal, nil, errors.New("provider_request")
	}
	p := approved
	policy := ModelPolicy{Model: p.Model, Provider: p.Provider, ProviderName: p.ProviderName, InputMicrosPerMillion: p.InputMicrosPerMillion, OutputMicrosPerMillion: p.OutputMicrosPerMillion, VerifiedAt: proof.VerifiedAt}
	messages := []LLMMessage{}
	for _, m := range statementextract.Messages(chunk) {
		messages = append(messages, LLMMessage{Role: m["role"], Content: m["content"]})
	}
	out, e := r.provider.Complete(ctx, LLMRequest{comparisonProof: proof, DisableReasoning: true, Policy: policy, Messages: messages, Schema: statementextract.RequestSchema(chunk), InputTokenUpperBound: statementextract.InputBound, MaxOutputTokens: statementextract.OutputBound})
	if !out.BillingKnown {
		return proposal, nil, e
	}
	bill := &statementextract.Billing{ComparisonRoute: out.ComparisonRoute, GenerationID: out.ID, CostMicros: out.CostMicros, InputTokens: out.PromptTokens, OutputTokens: out.CompletionTokens}
	if e != nil {
		return proposal, bill, e
	}
	dec := json.NewDecoder(bytes.NewReader(out.Content))
	dec.DisallowUnknownFields()
	if dec.Decode(&proposal) != nil || dec.Decode(&struct{}{}) != io.EOF {
		return proposal, bill, errors.New("provider_protocol")
	}
	return proposal, bill, nil
}
func comparisonSelectedClass(m *routerMetadata, p *statementextract.ComparisonIdentityProof) string {
	if m == nil || p == nil {
		return "not_observed"
	}
	count, model := 0, ""
	for _, e := range m.Endpoints.Available {
		if e.Selected {
			count++
			model = e.Model
		}
	}
	if count > 1 {
		return "multiple"
	}
	if count == 0 {
		return "not_observed"
	}
	if model == p.Candidate.Policy.Model {
		return "alias"
	}
	if model == p.Candidate.Canonical {
		return "pinned_canonical"
	}
	if model == "" {
		return "empty"
	}
	return "other"
}

// ComparisonProof returns an independent preflight snapshot for the private report.
func (r *ComparisonReasoner) ComparisonProof(ctx context.Context) (statementextract.ComparisonIdentityProof, error) {
	p, e := r.currentProof(ctx)
	if e != nil {
		return statementextract.ComparisonIdentityProof{}, e
	}
	return *p, nil
}
