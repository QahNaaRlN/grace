package assistant

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// Refresh only the approval timestamp of an already approved model, provider and
// price ceiling. Routing still enforces ZDR/deny/no-fallback on every inference.
// Metadata requests contain no user questions, images or financial data.
type PolicyVerifier struct {
	mu        sync.Mutex
	client    HTTPDoer
	endpoint  string
	checked   map[string]ModelPolicy
	attempted map[string]time.Time
}

func NewPolicyVerifier() *PolicyVerifier {
	return &PolicyVerifier{client: &http.Client{Timeout: 5 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}, endpoint: "https://openrouter.ai/api/v1/models/", checked: map[string]ModelPolicy{}, attempted: map[string]time.Time{}}
}
func (v *PolicyVerifier) Current(ctx context.Context, approved ModelPolicy) (ModelPolicy, error) {
	if v == nil {
		if approved.Valid(time.Now()) {
			return approved, nil
		}
		return ModelPolicy{}, errors.New("price_unknown")
	}
	identity := approved
	identity.VerifiedAt = time.Time{}
	key := aiHash(identity)
	v.mu.Lock()
	defer v.mu.Unlock()
	policy := approved
	if cached, ok := v.checked[key]; ok {
		policy = cached
	}
	now := time.Now()
	if policy.Valid(now) && now.Sub(policy.VerifiedAt) < 23*time.Hour {
		return policy, nil
	}
	if last := v.attempted[key]; !last.IsZero() && now.Sub(last) < time.Minute {
		return ModelPolicy{}, errors.New("price_unknown")
	}
	v.attempted[key] = now
	shape := approved
	shape.VerifiedAt = now
	if !shape.Valid(now) || approved.VerifiedAt.IsZero() || approved.VerifiedAt.After(now) {
		return ModelPolicy{}, errors.New("price_unknown")
	}
	parts := strings.Split(approved.Model, "/")
	for i := range parts {
		parts[i] = url.PathEscape(parts[i])
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, v.endpoint+strings.Join(parts, "/")+"/endpoints", nil)
	if err != nil {
		return ModelPolicy{}, errors.New("price_unknown")
	}
	response, err := v.client.Do(req)
	if err != nil {
		return ModelPolicy{}, errors.New("price_unknown")
	}
	defer response.Body.Close()
	if response.StatusCode != 200 {
		return ModelPolicy{}, errors.New("price_unknown")
	}
	raw, err := io.ReadAll(io.LimitReader(response.Body, (1<<20)+1))
	if err != nil || len(raw) > 1<<20 {
		return ModelPolicy{}, errors.New("price_unknown")
	}
	var metadata struct {
		Data struct {
			Endpoints []struct {
				ModelID    string   `json:"model_id"`
				Tag        string   `json:"tag"`
				Name       string   `json:"provider_name"`
				Status     int      `json:"status"`
				Parameters []string `json:"supported_parameters"`
				Pricing    struct {
					Prompt     string `json:"prompt"`
					Completion string `json:"completion"`
					Image      string `json:"image"`
				} `json:"pricing"`
			} `json:"endpoints"`
		} `json:"data"`
	}
	if json.Unmarshal(raw, &metadata) != nil {
		return ModelPolicy{}, errors.New("price_unknown")
	}
	for _, endpoint := range metadata.Data.Endpoints {
		if endpoint.Tag != approved.Provider || endpoint.ModelID != approved.Model || endpoint.Name != approved.DisplayProvider() || endpoint.Status != 0 {
			continue
		}
		parameters := map[string]bool{}
		for _, p := range endpoint.Parameters {
			parameters[p] = true
		}
		if !parameters["structured_outputs"] || !parameters["response_format"] || !parameters["max_tokens"] {
			continue
		}
		if !priceWithin(endpoint.Pricing.Prompt, approved.InputMicrosPerMillion) || !priceWithin(endpoint.Pricing.Completion, approved.OutputMicrosPerMillion) {
			continue
		}
		if approved.Vision && !priceWithin(endpoint.Pricing.Image, approved.InputMicrosPerMillion) {
			continue
		}
		approved.VerifiedAt = now
		v.checked[key] = approved
		return approved, nil
	}
	return ModelPolicy{}, errors.New("price_unknown")
}

func approvedPolicyShape(policy ModelPolicy) bool {
	now := time.Now()
	if policy.VerifiedAt.IsZero() || policy.VerifiedAt.After(now) {
		return false
	}
	policy.VerifiedAt = now
	return policy.Valid(now)
}
func priceWithin(perToken string, ceilingMicrosPerMillion int64) bool {
	if !decimalCost.MatchString(perToken) {
		return false
	}
	price, ok := new(big.Rat).SetString(perToken)
	if !ok || price.Sign() <= 0 {
		return false
	}
	ceiling := big.NewRat(ceilingMicrosPerMillion, 1000000000000)
	return price.Cmp(ceiling) <= 0
}
