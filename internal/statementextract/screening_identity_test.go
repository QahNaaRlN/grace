package statementextract

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestScreeningRouteSanitization(t *testing.T) {
	now := time.Now().UTC()
	proof := &ScreeningIdentityProof{Alias: ScreeningPolicy().Model, Canonical: ScreeningCanonicalModel, Provider: ScreeningPolicy().Provider, CatalogSHA256: strings.Repeat("a", 64), VerifiedAt: now.Add(-time.Second)}
	for _, class := range []string{"not_observed", "alias", "pinned_canonical", "other", "empty", "multiple", "PRIVATE-unknown"} {
		route := ScreeningRouteEvidence{Proof: proof, ObservedAt: now, SelectedModelClass: class}
		safe := route.Sanitize()
		if class == "PRIVATE-unknown" && safe.SelectedModelClass != "not_observed" {
			t.Fatal("unknown class reflected")
		}
		if safe.Proof == proof || safe.Proof == nil {
			t.Fatal("proof copy")
		}
		raw, _ := json.Marshal(safe)
		if strings.Contains(string(raw), "PRIVATE") {
			t.Fatal("sentinel reflected")
		}
	}
	bad := *proof
	bad.Alias = "PRIVATE-alias"
	bad.Canonical = "PRIVATE-canonical"
	bad.Provider = "PRIVATE-provider"
	bad.CatalogSHA256 = "PRIVATE-digest"
	safe := (ScreeningRouteEvidence{Proof: &bad, ObservedAt: now, SelectedModelClass: "PRIVATE-class"}).Sanitize()
	raw, _ := json.Marshal(safe)
	if strings.Contains(string(raw), "PRIVATE") || safe.Proof != nil {
		t.Fatal("invalid proof leaked")
	}
}
