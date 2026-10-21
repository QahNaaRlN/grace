package assistant

import (
	"errors"
	"net/http"
	"time"

	"github.com/QahNaaRlN/grace/internal/statementextract"
)

const statementMaxEnvelopeBytes = 100000

// A separate server-owned constructor keeps analysis/receipt limits unchanged.
// The profile cannot be selected by a document or a public request field.
func NewStatementOpenRouterProvider(key string, client HTTPDoer) (*OpenRouterProvider, error) {
	if client == nil {
		client = &http.Client{Timeout: statementextract.CallTimeoutSeconds * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	}
	if existing, ok := client.(*http.Client); ok {
		clone := *existing
		clone.Timeout = statementextract.CallTimeoutSeconds * time.Second
		clone.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
		client = &clone
	}
	provider, e := NewOpenRouterProvider(key, client)
	if e != nil {
		return nil, e
	}
	provider.statementDocument = true
	return provider, nil
}
func (p *OpenRouterProvider) StatementDocumentProfile() bool { return p != nil && p.statementDocument }
func (p *OpenRouterProvider) reservation(policy ModelPolicy, input, output int) (int64, error) {
	if !p.statementDocument {
		return policy.ReservationMicros(input, output)
	}
	pin := statementextract.Policy{Model: policy.Model, Provider: policy.Provider, ProviderName: policy.ProviderName, InputMicrosPerMillion: policy.InputMicrosPerMillion, OutputMicrosPerMillion: policy.OutputMicrosPerMillion}
	allowed := pin.Valid()
	if p.screeningDocument {
		allowed = statementextract.ScreeningPolicyValid(pin)
	}
	if p.comparisonID != "" {
		c, ok := statementextract.ComparisonCandidateByID(p.comparisonID)
		allowed = ok && c.Policy == pin && !p.screeningDocument && time.Since(policy.VerifiedAt) <= 10*time.Minute
	}
	if !policy.Valid(time.Now()) || !allowed || policy.Vision || policy.ImageTokenUpperBound != 0 || input < 1 || input > statementextract.InputBound || output < 1 || output > statementextract.OutputBound {
		return 0, errors.New("price_unknown")
	}
	return (policy.InputMicrosPerMillion*int64(input) + policy.OutputMicrosPerMillion*int64(output) + 999999) / 1000000, nil
}

func statementRequestTimeout() time.Duration {
	return statementextract.CallTimeoutSeconds * time.Second
}

func statementCapabilityFits(contextLength, maxCompletion int, maxPrompt *int, parameters map[string]bool) bool {
	return contextLength >= statementextract.InputBound+statementextract.OutputBound && maxCompletion >= statementextract.OutputBound && (maxPrompt == nil || *maxPrompt >= statementextract.InputBound) && parameters["reasoning"]
}
