package assistant

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/QahNaaRlN/grace/internal/documentdata"
	"github.com/QahNaaRlN/grace/internal/statementdata"
	"github.com/QahNaaRlN/grace/internal/statementextract"
)

type DocumentReasoner struct {
	Groups   *ModelGroups
	Provider ReasoningProvider
}

func (r *DocumentReasoner) CurrentDocumentPolicy(ctx context.Context, images int) (statementextract.DocumentPolicy, error) {
	if r == nil || r.Groups == nil || r.Provider == nil {
		return statementextract.DocumentPolicy{}, errors.New("ai_disabled")
	}
	p, e := r.Groups.Resolve(ctx, "statement", images)
	if e != nil {
		return statementextract.DocumentPolicy{}, e
	}
	maximum, e := p.ReservationMicros(30000, 16000)
	if e != nil {
		return statementextract.DocumentPolicy{}, e
	}
	raw, _ := json.Marshal(p)
	return statementextract.DocumentPolicy{InputMicrosPerMillion: p.InputMicrosPerMillion, OutputMicrosPerMillion: p.OutputMicrosPerMillion, Model: p.Model, Provider: p.Provider, ProviderName: p.DisplayProvider(), Profile: p.Selection.Profile, GroupRevision: p.Selection.GroupRevision, ProofSHA256: p.Selection.ProofSHA256, MaximumMicros: maximum, Snapshot: raw}, nil
}
func (r *DocumentReasoner) CompleteDocument(ctx context.Context, policy statementextract.DocumentPolicy, doc documentdata.Document) (statementdata.VisualReading, *statementextract.Billing, error) {
	var p ModelPolicy
	if !policy.Valid() || json.Unmarshal(policy.Snapshot, &p) != nil || !r.Groups.check(p) || p.Selection == nil || p.Selection.Purpose != "statement" || p.Model != policy.Model || p.Provider != policy.Provider || p.Selection.ProofSHA256 != policy.ProofSHA256 {
		return statementdata.VisualReading{}, &statementextract.Billing{GenerationID: "not_sent"}, errors.New("policy_changed")
	}
	maximum, e := p.ReservationMicros(30000, 16000)
	if e != nil || maximum != policy.MaximumMicros {
		return statementdata.VisualReading{}, &statementextract.Billing{GenerationID: "not_sent"}, errors.New("price_unknown")
	}
	if p.Selection.Profile == PlainDocumentReading {
		return statementdata.VisualReading{}, nil, errors.New("plain_reader_required")
	}
	request, e := DocumentRequest(p, doc, statementextract.DocumentPrompt, statementdata.VisualSchema(), 16000)
	if e != nil {
		return statementdata.VisualReading{}, &statementextract.Billing{GenerationID: "not_sent"}, e
	}
	out, e := r.Provider.Complete(ctx, request)
	var bill *statementextract.Billing
	if out.BillingKnown {
		bill = &statementextract.Billing{GenerationID: out.ID, CostMicros: out.CostMicros, InputTokens: out.PromptTokens, OutputTokens: out.CompletionTokens}
	}
	if e != nil {
		return statementdata.VisualReading{}, bill, e
	}
	var reading statementdata.VisualReading
	if statementextract.DirectJSON(out.Content, &reading) != nil || reading.Validate(len(doc.Pages)) != nil {
		return reading, bill, errors.New("incomplete_reading")
	}
	return reading, bill, nil
}
func (h *AIHandler) DocumentReasoner() *DocumentReasoner {
	if h.provider == nil || h.verifier == nil || h.verifier.Groups == nil {
		return nil
	}
	return &DocumentReasoner{Groups: h.verifier.Groups, Provider: h.provider}
}

func (r *DocumentReasoner) PrepareDocument(ctx context.Context, policy statementextract.DocumentPolicy, doc documentdata.Document) error {
	var p ModelPolicy
	if r == nil || r.Groups == nil || !policy.Valid() || json.Unmarshal(policy.Snapshot, &p) != nil || !r.Groups.check(p) {
		return errors.New("policy_changed")
	}
	if p.Selection == nil || p.Selection.Profile != DocumentReading {
		return errors.New("plain_reader_required")
	}
	_, e := DocumentRequest(p, doc, statementextract.DocumentPrompt, statementdata.VisualSchema(), 16000)
	return e
}
