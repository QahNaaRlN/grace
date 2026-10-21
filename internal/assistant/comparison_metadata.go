package assistant

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"github.com/QahNaaRlN/grace/internal/statementextract"
	"io"
	"net/http"
	"time"
)

func (r *ComparisonReasoner) metadata(ctx context.Context, path string, out any) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	req, e := http.NewRequestWithContext(ctx, http.MethodGet, r.base+path, nil)
	if e != nil {
		return "", e
	}
	resp, e := r.client.Do(req)
	if e != nil {
		return "", e
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return "", errors.New("comparison_metadata_unverified")
	}
	raw, e := io.ReadAll(io.LimitReader(resp.Body, (8<<20)+1))
	if e != nil || len(raw) > 8<<20 || json.Unmarshal(raw, out) != nil {
		return "", errors.New("comparison_metadata_unverified")
	}
	h := sha256.Sum256(raw)
	return hex.EncodeToString(h[:]), nil
}
func (r *ComparisonReasoner) currentProof(ctx context.Context) (*statementextract.ComparisonIdentityProof, error) {
	if r == nil {
		return nil, errors.New("comparison_metadata_unverified")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.proof != nil && r.proof.ValidAt(time.Now().UTC()) {
		out := *r.proof
		return &out, nil
	}
	r.proof = nil
	fail := errors.New("comparison_metadata_unverified")
	c, ok := statementextract.ComparisonCandidateByID(r.candidate.ID)
	if !ok || c != r.candidate || r.client == nil || r.provider == nil || r.provider.comparisonID != c.ID {
		return nil, fail
	}
	var catalog struct {
		Data []struct {
			ID        string `json:"id"`
			Canonical string `json:"canonical_slug"`
			Reasoning struct {
				Mandatory bool     `json:"mandatory"`
				Supported []string `json:"supported_efforts"`
			} `json:"reasoning"`
		} `json:"data"`
	}
	ch, e := r.metadata(ctx, "models", &catalog)
	if e != nil {
		return nil, fail
	}
	count := 0
	for _, m := range catalog.Data {
		if m.ID == c.Policy.Model {
			count++
			none := false
			for _, effort := range m.Reasoning.Supported {
				none = none || effort == "none"
			}
			if m.Canonical != c.Canonical || m.Reasoning.Mandatory || !none {
				return nil, fail
			}
		}
	}
	if count != 1 {
		return nil, fail
	}
	var endpoints struct {
		Data struct {
			Endpoints []struct {
				Model      string   `json:"model_id"`
				Tag        string   `json:"tag"`
				Name       string   `json:"provider_name"`
				Status     *int     `json:"status"`
				Context    int      `json:"context_length"`
				Output     int      `json:"max_completion_tokens"`
				Input      *int     `json:"max_prompt_tokens"`
				Parameters []string `json:"supported_parameters"`
				Pricing    struct {
					Input  string `json:"prompt"`
					Output string `json:"completion"`
				} `json:"pricing"`
			} `json:"endpoints"`
		} `json:"data"`
	}
	path := "models/" + c.Canonical + "/endpoints"
	eh, e := r.metadata(ctx, path, &endpoints)
	if e != nil {
		return nil, fail
	}
	count = 0
	for _, ep := range endpoints.Data.Endpoints {
		if ep.Model == c.Policy.Model && ep.Tag == c.Policy.Provider {
			count++
			params := map[string]bool{}
			for _, s := range ep.Parameters {
				params[s] = true
			}
			if ep.Name != c.Policy.ProviderName || ep.Status == nil || *ep.Status != 0 || !statementCapabilityFits(ep.Context, ep.Output, ep.Input, params) || !params["structured_outputs"] || !params["response_format"] || !params["max_tokens"] || !priceWithin(ep.Pricing.Input, c.Policy.InputMicrosPerMillion) || !priceWithin(ep.Pricing.Output, c.Policy.OutputMicrosPerMillion) {
				return nil, fail
			}
		}
	}
	if count != 1 {
		return nil, fail
	}
	var zdr struct {
		Data []struct {
			Model  string `json:"model_id"`
			Tag    string `json:"tag"`
			Status *int   `json:"status"`
		} `json:"data"`
	}
	zh, e := r.metadata(ctx, "endpoints/zdr", &zdr)
	if e != nil {
		return nil, fail
	}
	count = 0
	for _, ep := range zdr.Data {
		if ep.Model == c.Policy.Model && ep.Tag == c.Policy.Provider {
			count++
			if ep.Status == nil || *ep.Status != 0 {
				return nil, fail
			}
		}
	}
	if count != 1 {
		return nil, fail
	}
	proof := statementextract.ComparisonIdentityProof{Candidate: c, CatalogSHA256: ch, EndpointSHA256: eh, ZDRSHA256: zh, CatalogURL: "https://openrouter.ai/api/v1/models", EndpointURL: "https://openrouter.ai/api/v1/" + path, ZDRURL: "https://openrouter.ai/api/v1/endpoints/zdr", VerifiedAt: time.Now().UTC()}
	if !proof.ValidAt(time.Now().UTC()) {
		return nil, fail
	}
	r.proof = &proof
	out := proof
	return &out, nil
}
