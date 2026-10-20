package assistant

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/QahNaaRlN/grace/internal/statementextract"
)

// Proof передаётся внутри screening-запроса и не меняет общую ModelPolicy.
func (v *PolicyVerifier) screeningIdentity(ctx context.Context) (*statementextract.ScreeningIdentityProof, error) {
	fail := errors.New("screening_identity_unverified")
	if v == nil || v.client == nil {
		return nil, fail
	}
	v.mu.Lock()
	defer v.mu.Unlock()
	now := time.Now().UTC()
	if v.screeningProof != nil && v.screeningProof.ValidAt(now) {
		out := *v.screeningProof
		return &out, nil
	}
	// Неудачное обновление не оставляет stale proof пригодным для запроса.
	v.screeningProof = nil
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimSuffix(v.endpoint, "/"), nil)
	if err != nil {
		return nil, fail
	}
	response, err := v.client.Do(req)
	if err != nil {
		return nil, fail
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, fail
	}
	raw, err := io.ReadAll(io.LimitReader(response.Body, (8<<20)+1))
	if err != nil || len(raw) > 8<<20 {
		return nil, fail
	}
	var metadata struct {
		Data []struct {
			ID        string `json:"id"`
			Canonical string `json:"canonical_slug"`
		} `json:"data"`
	}
	if json.Unmarshal(raw, &metadata) != nil {
		return nil, fail
	}
	pin := statementextract.ScreeningPolicy()
	matches := 0
	for _, model := range metadata.Data {
		if model.ID == pin.Model {
			matches++
			if model.Canonical != statementextract.ScreeningCanonicalModel {
				return nil, fail
			}
		}
	}
	if matches != 1 {
		return nil, fail
	}
	digest := sha256.Sum256(raw)
	proof := statementextract.ScreeningIdentityProof{Alias: pin.Model, Canonical: statementextract.ScreeningCanonicalModel, Provider: pin.Provider, CatalogSHA256: hex.EncodeToString(digest[:]), VerifiedAt: time.Now().UTC()}
	v.screeningProof = &proof
	out := proof
	return &out, nil
}

func selectedModelClass(metadata *routerMetadata) string {
	if metadata == nil {
		return "not_observed"
	}
	count, model := 0, ""
	for _, endpoint := range metadata.Endpoints.Available {
		if endpoint.Selected {
			count++
			model = endpoint.Model
		}
	}
	if count > 1 {
		return "multiple"
	}
	if count == 0 {
		return "not_observed"
	}
	switch model {
	case "":
		return "empty"
	case statementextract.ScreeningPolicy().Model:
		return "alias"
	case statementextract.ScreeningCanonicalModel:
		return "pinned_canonical"
	default:
		return "other"
	}
}
