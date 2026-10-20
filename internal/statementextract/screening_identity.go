package statementextract

import (
	"encoding/hex"
	"time"
)

const ScreeningCanonicalModel = "google/gemini-3.1-flash-lite-20260507"

// ScreeningIdentityProof описывает только проверенную пару публичных имён.
// Это не финансовая политика, не подтверждение billing и не разрешение inference.
type ScreeningIdentityProof struct {
	Alias         string    `json:"alias"`
	Canonical     string    `json:"canonical"`
	Provider      string    `json:"provider"`
	CatalogSHA256 string    `json:"catalogSha256"`
	VerifiedAt    time.Time `json:"verifiedAt"`
}

func (p ScreeningIdentityProof) ValidAt(observed time.Time) bool {
	pin := ScreeningPolicy()
	digest, err := hex.DecodeString(p.CatalogSHA256)
	return p.Alias == pin.Model && p.Canonical == ScreeningCanonicalModel && p.Provider == pin.Provider && err == nil && len(digest) == 32 && hex.EncodeToString(digest) == p.CatalogSHA256 && !p.VerifiedAt.IsZero() && !observed.IsZero() && !p.VerifiedAt.After(observed) && observed.Sub(p.VerifiedAt) <= 10*time.Minute
}

type ScreeningRouteEvidence struct {
	Proof              *ScreeningIdentityProof `json:"proof,omitempty"`
	ObservedAt         time.Time               `json:"observedAt,omitempty"`
	SelectedModelClass string                  `json:"selectedModelClass"`
}

func (e ScreeningRouteEvidence) Sanitize() ScreeningRouteEvidence {
	switch e.SelectedModelClass {
	case "not_observed", "alias", "pinned_canonical", "other", "empty", "multiple":
	default:
		e.SelectedModelClass = "not_observed"
	}
	if e.Proof == nil || !e.Proof.ValidAt(e.ObservedAt) {
		e.Proof = nil
		e.ObservedAt = time.Time{}
	} else {
		proof := *e.Proof
		e.Proof = &proof
	}
	return e
}

func (e *ScreeningRouteEvidence) Valid(end time.Time) bool {
	return e != nil && e.Proof != nil && e.Proof.ValidAt(e.ObservedAt) && !e.ObservedAt.After(end) && (e.SelectedModelClass == "alias" || e.SelectedModelClass == "pinned_canonical")
}
