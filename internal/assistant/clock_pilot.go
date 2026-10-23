package assistant

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"math/big"

	"github.com/QahNaaRlN/grace/internal/statementdata"
	"github.com/QahNaaRlN/grace/internal/statementextract"
)

// ClockPilotReasoner is a fixed A/cards operator adapter; product constructors
// and the old comparison reasoner never use its candidate instructions.
type ClockPilotReasoner struct{ comparison *ComparisonReasoner }

func NewClockPilotReasoner(key string) (*ClockPilotReasoner, error) {
	r, e := NewStatementComparisonReasoner(key, "A")
	if e != nil {
		return nil, e
	}
	r.clockExactPrices = true
	return &ClockPilotReasoner{r}, nil
}
func (r *ClockPilotReasoner) CurrentPolicy(ctx context.Context) (statementextract.Policy, error) {
	return r.comparison.CurrentPolicy(ctx)
}
func (r *ClockPilotReasoner) ComparisonProof(ctx context.Context) (statementextract.ComparisonIdentityProof, error) {
	return r.comparison.ComparisonProof(ctx)
}
func exactClockPrice(s string, n int64) bool {
	if !decimalCost.MatchString(s) {
		return false
	}
	v, ok := new(big.Rat).SetString(s)
	return ok && v.Cmp(big.NewRat(n, 1000000000000)) == 0
}
func (r *ClockPilotReasoner) CompleteClock(ctx context.Context, d statementdata.Document, variant string) (statementdata.Proposal, *statementextract.Billing, error) {
	var proposal statementdata.Proposal
	if d.Hash() != statementextract.ClockCardsDocumentHash {
		return proposal, nil, errors.New("clock_fixture_invalid")
	}
	request, e := statementextract.BuildClockRequest(d, variant)
	if e != nil {
		return proposal, nil, e
	}
	proof, e := r.comparison.currentProof(ctx)
	if e != nil {
		return proposal, nil, e
	}
	p := request.Plan.Policy
	policy := ModelPolicy{Model: p.Model, Provider: p.Provider, ProviderName: p.ProviderName, InputMicrosPerMillion: p.InputMicrosPerMillion, OutputMicrosPerMillion: p.OutputMicrosPerMillion, VerifiedAt: proof.VerifiedAt}
	messages := []LLMMessage{}
	for _, m := range request.Messages {
		messages = append(messages, LLMMessage{Role: m["role"], Content: m["content"]})
	}
	out, e := r.comparison.provider.Complete(ctx, LLMRequest{comparisonProof: proof, DisableReasoning: true, Policy: policy, Messages: messages, Schema: request.Schema, InputTokenUpperBound: statementextract.InputBound, MaxOutputTokens: statementextract.OutputBound})
	if !out.BillingKnown {
		if e == nil {
			e = errors.New("provider_protocol")
		}
		return proposal, nil, e
	}
	bill := &statementextract.Billing{ComparisonRoute: out.ComparisonRoute, GenerationID: out.ID, CostMicros: out.CostMicros, InputTokens: out.PromptTokens, OutputTokens: out.CompletionTokens}
	if e != nil {
		return proposal, bill, e
	}
	decoder := json.NewDecoder(bytes.NewReader(out.Content))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&proposal) != nil || decoder.Decode(&struct{}{}) != io.EOF {
		return proposal, bill, errors.New("provider_protocol")
	}
	return proposal, bill, nil
}
