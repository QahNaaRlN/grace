package statementqualification

import (
	"errors"
	"github.com/QahNaaRlN/grace/internal/statementextract"
	"reflect"
	"time"
)

// ValidateComparisonReport validates both complete failures and interrupted
// campaigns. Successful admission is a separate predicate, never activation.
func ValidateComparisonReport(r ComparisonReport, now time.Time) error {
	fail := errors.New("invalid_comparison_report")
	if r.Version != ComparisonReportVersion || r.Authorization != ComparisonAuthorization || r.Profile != ComparisonProfile() || !reflect.DeepEqual(r.Candidates, statementextract.ComparisonCandidates()) || r.Digest != comparisonDigest(r) || r.Time.IsZero() || r.Time.After(r.FinishedAt) || r.FinishedAt.After(now) || r.FinishedAt.Sub(r.Time) > 906*time.Second || len(r.Tuples) != 6 || r.MaximumMicros != ComparisonMaxMicros || r.ReservationID == "" || len(r.ReservationID) > 128 || ComparisonErrorCode(r.Error) != r.Error {
		return fail
	}
	if len(r.Proofs) != 2 {
		return fail
	}
	for i, p := range r.Proofs {
		if p.Candidate != r.Candidates[i] || !p.ValidAt(p.VerifiedAt) || p.VerifiedAt.After(r.FinishedAt) {
			return fail
		}
	}
	total := int64(0)
	ids := map[string]bool{}
	stopped := false
	allPassed := true
	allCompleted := true
	for i, t := range r.Tuples {
		c := r.Candidates[i%2]
		n := ScreeningNames()[i/2]
		checks := []int{19, 20, 23}[i/2]
		if t.Candidate != c.ID || t.Evidence.Name != n || t.Evidence.FixtureHash != FixtureHash(n) || !validHash(t.Evidence.DocumentHash) || !validHash(t.Evidence.PlanHash) || !validHash(t.RequestHash) || t.Evidence.InputBytes < 1 || t.Evidence.InputBytes > statementextract.MaxInputBytes {
			return fail
		}
		if i%2 == 1 && (t.Evidence.DocumentHash != r.Tuples[i-1].Evidence.DocumentHash || t.Evidence.InputBytes != r.Tuples[i-1].Evidence.InputBytes) {
			return fail
		}
		if t.State == "not_sent" {
			stopped = true
			allPassed = false
			allCompleted = false
			if t.Call != nil || t.Result != nil || t.Proposal != nil || t.Diagnostic != nil || t.FalseAcceptance || t.Reason != "campaign_not_started" {
				return fail
			}
			continue
		}
		if stopped {
			return fail
		}
		switch t.State {
		case "attempted":
			stopped = true
			allPassed = false
			allCompleted = false
		case "completed":
		default:
			return fail
		}
		if t.Call != nil {
			bill := &statementextract.Billing{GenerationID: t.Call.GenerationID, CostMicros: t.Call.CostMicros, InputTokens: t.Call.InputTokens, OutputTokens: t.Call.OutputTokens}
			if !validBill(bill, c.Policy) || ids[bill.GenerationID] || t.Call.LatencyMS < 0 {
				return fail
			}
			ids[bill.GenerationID] = true
			total += bill.CostMicros
		}
		if t.State == "completed" {
			if t.Diagnostic != nil || t.Call == nil || !t.Call.Route.Valid(c.ID, r.Time, r.FinishedAt) || t.Call.OutputTokens == 0 || t.Proposal == nil {
				return fail
			}
			switch t.Reason {
			case "grounding_failed":
				if t.Result != nil {
					return fail
				}
				allPassed = false
			case "", "quality_failed":
				s := t.Result
				if s == nil || s.Name != n || s.Checks < 1 || s.Checks > checks || s.Correct < 0 || s.Correct > s.Checks || s.LatencyMS != t.Call.LatencyMS || t.FalseAcceptance != (!s.Passed && len(s.Validation) == 0) {
					return fail
				}
				good := s.Passed && s.Rows == 2 && s.Checks == checks && s.Correct == checks && len(s.Codes) == 0 && len(s.Validation) == 0 && len(s.Coverage) == 0
				if (t.Reason == "") != good || (t.Reason == "quality_failed" && s.Passed) {
					return fail
				}
				allPassed = allPassed && good
			default:
				return fail
			}
		} else {
			if t.FalseAcceptance || t.Result != nil || (t.Reason != "provider_failed" && t.Reason != "protocol_failed") {
				return fail
			}
		}
	}
	if r.ActualMicros != nil && (*r.ActualMicros != total || total > ComparisonMaxMicros) {
		return fail
	}
	if r.Passed != (allPassed && allCompleted && r.ActualMicros != nil && r.Error == "") {
		return fail
	}
	if r.Passed && r.Error != "" || !r.Passed && r.Error == "" || allCompleted && r.Error != "" && r.Error != "qualification_quality_failed" && r.Error != "qualification_settlement_failed" {
		return fail
	}
	return nil
}
func ComparisonErrorCode(code string) string {
	switch code {
	case "", "invalid_qualification_corpus", "invalid_qualification_fixture", "comparison_policy_invalid", "qualification_plan_invalid", "qualification_cap_exceeded", "qualification_budget_unavailable", "qualification_timeout", "comparison_provider_failed", "qualification_cost_overrun", "qualification_protocol_failed", "qualification_quality_failed", "qualification_settlement_failed":
		return code
	}
	return "unknown"
}

type ComparisonMetrics struct {
	FalseAcceptances int    `json:"falseAcceptances"`
	Candidate        string `json:"candidate"`
	Attempted        int    `json:"attempted"`
	Completed        int    `json:"completed"`
	Successful       int    `json:"successful"`
	Quality          string `json:"quality"`
	Efficiency       string `json:"efficiency"`
	KnownCostMicros  int64  `json:"knownCostMicros"`
	// Numerator/denominator preserve exact cost without fractional-money floats.
	CostPerSuccessNumerator   *int64  `json:"costPerSuccessNumerator"`
	CostPerSuccessDenominator int     `json:"costPerSuccessDenominator"`
	LatenciesMS               []int64 `json:"latenciesMs"`
}

func ComparisonMetricsFor(r ComparisonReport, now time.Time) ([]ComparisonMetrics, error) {
	if e := ValidateComparisonReport(r, now); e != nil {
		return nil, e
	}
	out := []ComparisonMetrics{}
	for j, c := range r.Candidates {
		m := ComparisonMetrics{Candidate: c.ID, Quality: "FAIL", Efficiency: "UNVERIFIED", LatenciesMS: []int64{}}
		for i := j; i < 6; i += 2 {
			t := r.Tuples[i]
			if t.State != "not_sent" {
				m.Attempted++
			}
			if t.State == "completed" {
				m.Completed++
			}
			if t.FalseAcceptance {
				m.FalseAcceptances++
			}
			if t.Result != nil && t.Result.Passed {
				m.Successful++
			}
			if t.Call != nil {
				m.KnownCostMicros += t.Call.CostMicros
				m.LatenciesMS = append(m.LatenciesMS, t.Call.LatencyMS)
			}
		}
		if m.Completed < 3 {
			m.Quality = "UNVERIFIED"
		} else if m.Successful == 3 {
			m.Quality = "PASS"
		}
		campaignComplete := true
		for _, t := range r.Tuples {
			campaignComplete = campaignComplete && t.State == "completed"
		}
		if campaignComplete && r.ActualMicros != nil && r.Error != "qualification_settlement_failed" {
			m.Efficiency = "undefined"
			if m.Successful > 0 {
				m.Efficiency = "observed"
				n := m.KnownCostMicros
				m.CostPerSuccessNumerator = &n
				m.CostPerSuccessDenominator = m.Successful
			}
		}
		out = append(out, m)
	}
	return out, nil
}
