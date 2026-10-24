package statementextract

import "github.com/QahNaaRlN/grace/internal/statementdata"

// ScreeningPolicy is an operator-only experiment identity, never a product
// consent policy. Policy.Valid deliberately continues to reject this model.
func ScreeningPolicy() Policy {
	return Policy{Model: "google/gemini-3.1-flash-lite", Provider: "google-vertex/eu", ProviderName: "Google", InputMicrosPerMillion: 275000, OutputMicrosPerMillion: 1650000}
}
func ScreeningPolicyValid(p Policy) bool { return p == ScreeningPolicy() }
func BuildScreeningPlan(d statementdata.Document, p Policy) (Plan, error) {
	return buildPlan(d, p, nil, ScreeningPolicyValid(p))
}
func CanonicalScreeningProposal(p Plan, in statementdata.Proposal) (statementdata.Proposal, error) {
	return canonicalProposal(p, in, planShapeValid(p, ScreeningPolicyValid(p.Policy)))
}
