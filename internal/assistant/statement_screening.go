package assistant

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/QahNaaRlN/grace/internal/statementextract"
	"io"
	"net/http"
	"strings"
	"time"
)

// NewStatementScreeningReasoner creates a separate operator-only path. The
// product constructor, product planner and activation report reject its policy.
func NewStatementScreeningReasoner(key string) (*StatementReasoner, error) {
	p, e := NewStatementOpenRouterProvider(key, nil)
	if e != nil {
		return nil, e
	}
	p.screeningDocument = true
	pin := statementextract.ScreeningPolicy()
	return &StatementReasoner{screening: true, Verifier: NewPolicyVerifier(), Provider: p, Policy: ModelPolicy{Model: pin.Model, Provider: pin.Provider, ProviderName: pin.ProviderName, InputMicrosPerMillion: pin.InputMicrosPerMillion, OutputMicrosPerMillion: pin.OutputMicrosPerMillion, VerifiedAt: time.Unix(1, 0).UTC()}}, nil
}

func (r *StatementReasoner) verifyScreeningZDR(ctx context.Context) error {
	fail := errors.New("screening_privacy_unverified")
	if r.Verifier == nil {
		return fail
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	url := strings.TrimSuffix(r.Verifier.endpoint, "models/") + "endpoints/zdr"
	req, e := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if e != nil {
		return fail
	}
	response, e := r.Verifier.client.Do(req)
	if e != nil {
		return fail
	}
	defer response.Body.Close()
	if response.StatusCode != 200 {
		return fail
	}
	raw, e := io.ReadAll(io.LimitReader(response.Body, (8<<20)+1))
	if e != nil || len(raw) > 8<<20 {
		return fail
	}
	var metadata struct {
		Data []struct {
			Model  string `json:"model_id"`
			Tag    string `json:"tag"`
			Status *int   `json:"status"`
		}
	}
	if json.Unmarshal(raw, &metadata) != nil {
		return fail
	}
	for _, p := range metadata.Data {
		if p.Model == r.Policy.Model && p.Tag == r.Policy.Provider && p.Status != nil && *p.Status == 0 {
			return nil
		}
	}
	return fail
}
