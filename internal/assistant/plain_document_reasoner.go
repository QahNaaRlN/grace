package assistant

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/QahNaaRlN/grace/internal/documentdata"
	"github.com/QahNaaRlN/grace/internal/statementdata"
	"github.com/QahNaaRlN/grace/internal/statementextract"
)

func (r *DocumentReasoner) PlainPolicy(ctx context.Context, images int) (statementextract.DocumentPolicy, error) {
	p, e := r.CurrentDocumentPolicy(ctx, images)
	if e != nil {
		return p, e
	}
	if p.Profile != PlainDocumentReading {
		return p, errors.New("plain_profile_unavailable")
	}
	return p, nil
}
func plainRequest(policy statementextract.DocumentPolicy, doc documentdata.Document) (LLMRequest, error) {
	var p ModelPolicy
	if !policy.Valid() || json.Unmarshal(policy.Snapshot, &p) != nil || p.Selection == nil || p.Selection.Profile != PlainDocumentReading {
		return LLMRequest{}, errors.New("policy_changed")
	}
	if p.InputMicrosPerMillion > 275000 || p.OutputMicrosPerMillion > 1650000 || p.RequestFeeMicros > 1000 || p.ImageFeeMicros > 1 || p.ImageTokenUpperBound > 4200 {
		return LLMRequest{}, errors.New("price_unknown")
	}
	maximum, e := p.ReservationMicros(30000, 16000)
	if e != nil || maximum != policy.MaximumMicros || p.Model != policy.Model || p.Provider != policy.Provider || p.Selection.ProofSHA256 != policy.ProofSHA256 {
		return LLMRequest{}, errors.New("policy_changed")
	}
	return DocumentRequest(p, doc, fmt.Sprintf(statementextract.PlainPrompt, len(doc.Pages)), nil, 16000)
}
func (r *DocumentReasoner) PreparePlain(ctx context.Context, p statementextract.DocumentPolicy, doc documentdata.Document) error {
	request, e := plainRequest(p, doc)
	if e != nil {
		return e
	}
	if !r.Groups.check(request.Policy) {
		return errors.New("policy_changed")
	}
	return nil
}
func (r *DocumentReasoner) CompletePlain(ctx context.Context, p statementextract.DocumentPolicy, doc documentdata.Document) (statementdata.PlainReading, *statementextract.Billing, error) {
	request, e := plainRequest(p, doc)
	if e != nil {
		return statementdata.PlainReading{}, nil, e
	}
	if !r.Groups.check(request.Policy) {
		return statementdata.PlainReading{}, nil, errors.New("policy_changed")
	}
	out, e := r.Provider.Complete(ctx, request)
	var bill *statementextract.Billing
	if out.BillingKnown {
		bill = &statementextract.Billing{GenerationID: out.ID, CostMicros: out.CostMicros, InputTokens: out.PromptTokens, OutputTokens: out.CompletionTokens}
	}
	if e != nil {
		return statementdata.PlainReading{}, bill, e
	}
	return statementdata.ParsePlain(string(out.Content)), bill, nil
}
