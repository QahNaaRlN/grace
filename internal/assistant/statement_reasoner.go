package assistant

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"time"

	"github.com/QahNaaRlN/grace/internal/statementdata"
	"github.com/QahNaaRlN/grace/internal/statementextract"
)

// StatementReasoner reuses the existing pinned/ZDR/no-fallback transport. It
// has no tools or financial writer and is separately disabled by default.
type StatementReasoner struct {
	Verifier          *PolicyVerifier
	Policy            ModelPolicy
	Provider          ReasoningProvider
	screening         bool
	screeningVerified time.Time
}

func (r *StatementReasoner) CurrentPolicy(ctx context.Context) (statementextract.Policy, error) {
	if r == nil || r.Provider == nil || r.Policy.Vision || r.Policy.ImageTokenUpperBound != 0 {
		return statementextract.Policy{}, errors.New("ai_disabled")
	}
	if profile, ok := r.Provider.(interface{ StatementDocumentProfile() bool }); !ok || !profile.StatementDocumentProfile() {
		return statementextract.Policy{}, errors.New("provider_request")
	}
	if r.screening && time.Since(r.screeningVerified) > 10*time.Minute {
		if e := r.verifyScreeningZDR(ctx); e != nil {
			return statementextract.Policy{}, e
		}
		r.screeningVerified = time.Now()
	}
	p, e := r.Verifier.CurrentStatement(ctx, r.Policy)
	if e != nil {
		return statementextract.Policy{}, e
	}
	out := statementextract.Policy{Model: p.Model, Provider: p.Provider, ProviderName: p.ProviderName, InputMicrosPerMillion: p.InputMicrosPerMillion, OutputMicrosPerMillion: p.OutputMicrosPerMillion}
	allowed := out.Valid()
	if r.screening {
		allowed = statementextract.ScreeningPolicyValid(out)
	}
	if !allowed {
		return out, errors.New("price_unknown")
	}
	return out, nil
}
func (r *StatementReasoner) Complete(ctx context.Context, approved statementextract.Policy, chunk statementextract.Chunk) (statementdata.Proposal, *statementextract.Billing, error) {
	var proposal statementdata.Proposal
	p, e := r.CurrentPolicy(ctx)
	if e != nil {
		return proposal, nil, e
	}
	if statementextract.Hash(p) != statementextract.Hash(approved) || statementextract.InputBytes(chunk) > statementextract.MaxInputBytes || chunk.Index != 1 || len(chunk.Context) != 0 || len(chunk.Spans) == 0 || len(chunk.Spans) > statementextract.MaxPlanSpans {
		return proposal, nil, errors.New("provider_request")
	}
	policy, e := r.Verifier.CurrentStatement(ctx, r.Policy)
	if e != nil {
		return proposal, nil, e
	}
	messages := []LLMMessage{}
	for _, m := range statementextract.Messages(chunk) {
		messages = append(messages, LLMMessage{Role: m["role"], Content: m["content"]})
	}
	out, e := r.Provider.Complete(ctx, LLMRequest{DisableReasoning: true, Policy: policy, Messages: messages, Schema: statementextract.RequestSchema(chunk), InputTokenUpperBound: statementextract.InputBound, MaxOutputTokens: statementextract.OutputBound})
	if e != nil && !out.BillingKnown {
		return proposal, nil, e
	}
	billing := &statementextract.Billing{GenerationID: out.ID, CostMicros: out.CostMicros, InputTokens: out.PromptTokens, OutputTokens: out.CompletionTokens}
	if e != nil {
		return proposal, billing, e
	}
	dec := json.NewDecoder(bytes.NewReader(out.Content))
	dec.DisallowUnknownFields()
	if e = dec.Decode(&proposal); e != nil {
		return proposal, billing, errors.New("provider_protocol")
	}
	if e = dec.Decode(&struct{}{}); e != io.EOF {
		return proposal, billing, errors.New("provider_protocol")
	}
	return proposal, billing, nil
}
