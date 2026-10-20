package statementqualification

import (
	"encoding/hex"
	"errors"
	"time"

	"github.com/QahNaaRlN/grace/internal/statementdata"
	"github.com/QahNaaRlN/grace/internal/statementextract"
)

const ReportVersion = "statement.synthetic.v3.lexical-records"
const ScorerVersion = "statement-fixed-corpus.v2.record-descriptions"
const MaxSmokeMicros int64 = 140000
const MaxSmokeCalls = 2
const RunTimeout = 300 * time.Second

type Profile struct {
	Planner, Prompt, Schema, Redaction, Alias, Extractor, Scorer string
	Input, Output, Bytes, Spans, CallSeconds, RunSeconds, Calls  int
	CeilingMicros                                                int64
}

func CurrentProfile() Profile {
	return Profile{statementextract.PlannerVersion, statementextract.PromptVersion, statementextract.Hash(statementextract.ProposalSchema), statementextract.RedactionVersion, statementextract.AliasVersion, statementdata.ExtractorVersion, ScorerVersion, statementextract.InputBound, statementextract.OutputBound, statementextract.MaxInputBytes, statementextract.MaxPlanSpans, statementextract.CallTimeoutSeconds, int(RunTimeout / time.Second), MaxSmokeCalls, MaxSmokeMicros}
}
func FixtureHash(name string) string {
	switch name {
	case "turnover-roles-ru":
		return "648ac4477b9a78a7a1bb43eb9ae918fd5b6a2eb338e2ee07038146141781686f"
	case "columns":
		return "097c3d35d4dd9787457b13f61bd524ff83c18d969ce50b93f7c8cecfeb569c8c"
	case "cards":
		return "d4dd02153724c1599d4d89eb073e8cfaaf3fbce1696a4d691b69d150d0ccdce6"
	}
	return ""
}

type CaseEvidence struct {
	Name         string `json:"name"`
	FixtureHash  string `json:"fixtureHash"`
	DocumentHash string `json:"documentHash"`
	PlanHash     string `json:"planHash"`
	InputBytes   int    `json:"inputBytes"`
}

func validHash(s string) bool      { b, e := hex.DecodeString(s); return e == nil && len(b) == 32 }
func reportDigest(r Report) string { r.Digest = ""; return statementextract.Hash(r) }
func validBill(b *statementextract.Billing, policy statementextract.Policy) bool {
	return b != nil && b.GenerationID != "" && len(b.GenerationID) <= 200 && b.CostMicros >= 0 && b.CostMicros <= policy.CallMicros() && b.InputTokens >= 1 && b.InputTokens <= statementextract.InputBound && b.OutputTokens >= 0 && b.OutputTokens <= statementextract.OutputBound
}

// Reports are protected operator files, not an authorization token. The digest
// detects accidental edits; it is not a signature against a malicious operator.
func ValidateReport(r Report, now time.Time) error {
	fail := errors.New("invalid_qualification_report")
	if r.Version != ReportVersion || r.Profile != CurrentProfile() || !r.Passed || r.Error != "" || r.Digest != reportDigest(r) || !r.Policy.Valid() || r.Time.After(now) || now.Sub(r.Time) > 15*time.Minute || len(r.Corpus) != 2 || len(r.Results) != 2 || len(r.Calls) != 2 || len(r.Traces) != 2 || r.MaximumMicros != 2*r.Policy.CallMicros() || r.MaximumMicros > MaxSmokeMicros || r.ActualMicros == nil {
		return fail
	}
	var total int64
	ids := map[string]bool{}
	for _, call := range r.Calls {
		bill := statementextract.Billing{GenerationID: call.GenerationID, CostMicros: call.CostMicros, InputTokens: call.InputTokens, OutputTokens: call.OutputTokens}
		if !validBill(&bill, r.Policy) || call.OutputTokens == 0 || ids[call.GenerationID] {
			return fail
		}
		ids[call.GenerationID] = true
		total += call.CostMicros
	}
	if total != *r.ActualMicros || total > r.MaximumMicros {
		return fail
	}
	for i, name := range []string{"columns", "cards"} {
		c, result, trace := r.Corpus[i], r.Results[i], r.Traces[i]
		checks := 19
		if name == "cards" {
			checks = 20
		}
		if c.Name != name || c.FixtureHash != FixtureHash(name) || !validHash(c.DocumentHash) || !validHash(c.PlanHash) || c.InputBytes <= 0 || c.InputBytes > statementextract.MaxInputBytes || result.Name != name || !result.Passed || result.Rows != 2 || result.Checks != checks || result.Correct != checks || len(result.Codes) != 0 || len(result.Validation) != 0 || len(result.Coverage) != 0 || trace.Name != name || len(trace.Parts) != 1 {
			return fail
		}
	}
	return nil
}
