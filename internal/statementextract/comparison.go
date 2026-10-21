package statementextract

import (
	"encoding/hex"
	"github.com/QahNaaRlN/grace/internal/statementdata"
	"time"
)

// Comparison is an operator-only synthetic experiment, never product consent.
type ComparisonCandidate struct {
	ID        string `json:"id"`
	Policy    Policy `json:"policy"`
	Canonical string `json:"canonical"`
}

func ComparisonCandidates() []ComparisonCandidate {
	return []ComparisonCandidate{
		{"A", ScreeningPolicy(), ScreeningCanonicalModel},
		{"B", Policy{"inception/mercury-3", "inception", "Inception", 100000, 350000}, "inception/mercury-3-20261019"},
	}
}
func ComparisonCandidateByID(id string) (ComparisonCandidate, bool) {
	for _, c := range ComparisonCandidates() {
		if c.ID == id {
			return c, true
		}
	}
	return ComparisonCandidate{}, false
}
func ComparisonPolicyValid(p Policy) bool {
	for _, c := range ComparisonCandidates() {
		if c.Policy == p {
			return true
		}
	}
	return false
}
func BuildComparisonPlan(d statementdata.Document, p Policy) (Plan, error) {
	return buildPlan(d, p, nil, ComparisonPolicyValid(p))
}
func CanonicalComparisonProposal(p Plan, in statementdata.Proposal) (statementdata.Proposal, error) {
	return canonicalProposal(p, in, planShapeValid(p, ComparisonPolicyValid(p.Policy)))
}

type ComparisonIdentityProof struct {
	Candidate      ComparisonCandidate `json:"candidate"`
	CatalogSHA256  string              `json:"catalogSha256"`
	EndpointSHA256 string              `json:"endpointSha256"`
	ZDRSHA256      string              `json:"zdrSha256"`
	CatalogURL     string              `json:"catalogUrl"`
	EndpointURL    string              `json:"endpointUrl"`
	ZDRURL         string              `json:"zdrUrl"`
	VerifiedAt     time.Time           `json:"verifiedAt"`
}

func (p ComparisonIdentityProof) ValidAt(now time.Time) bool {
	c, ok := ComparisonCandidateByID(p.Candidate.ID)
	if !ok || c != p.Candidate || p.VerifiedAt.IsZero() || p.VerifiedAt.After(now) || now.Sub(p.VerifiedAt) > 10*time.Minute {
		return false
	}
	for _, s := range []string{p.CatalogSHA256, p.EndpointSHA256, p.ZDRSHA256} {
		b, e := hex.DecodeString(s)
		if e != nil || len(b) != 32 || hex.EncodeToString(b) != s {
			return false
		}
	}
	return p.CatalogURL == "https://openrouter.ai/api/v1/models" && p.EndpointURL == "https://openrouter.ai/api/v1/models/"+c.Canonical+"/endpoints" && p.ZDRURL == "https://openrouter.ai/api/v1/endpoints/zdr"
}

type ComparisonRouteEvidence struct {
	Proof              *ComparisonIdentityProof `json:"proof,omitempty"`
	ObservedAt         time.Time                `json:"observedAt"`
	SelectedModelClass string                   `json:"selectedModelClass"`
}

func (e *ComparisonRouteEvidence) Valid(id string, start, end time.Time) bool {
	return e != nil && e.Proof != nil && e.Proof.Candidate.ID == id && e.Proof.ValidAt(e.ObservedAt) && !e.ObservedAt.Before(start) && !e.ObservedAt.After(end) && (e.SelectedModelClass == "alias" || e.SelectedModelClass == "pinned_canonical")
}
