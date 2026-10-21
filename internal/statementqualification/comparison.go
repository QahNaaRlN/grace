package statementqualification

import (
	"context"
	"errors"
	"github.com/QahNaaRlN/grace/internal/statementdata"
	"github.com/QahNaaRlN/grace/internal/statementextract"
	"time"
)

const ComparisonReportVersion = "statement.comparison.v1.grounding"
const ComparisonMaxMicros int64 = 208500
const ComparisonAuthorization = "s111-comparison-20261021"

func ComparisonProfile() Profile {
	p := ScreeningProfile()
	p.Calls = 6
	p.RunSeconds = 900
	p.CeilingMicros = ComparisonMaxMicros
	return p
}

type ComparisonCall struct {
	Route        *statementextract.ComparisonRouteEvidence `json:"route,omitempty"`
	GenerationID string                                    `json:"generationId"`
	InputTokens  int                                       `json:"inputTokens"`
	OutputTokens int                                       `json:"outputTokens"`
	CostMicros   int64                                     `json:"costMicros"`
	LatencyMS    int64                                     `json:"latencyMs"`
}
type ComparisonTuple struct {
	Diagnostic      *statementextract.ScreeningDiagnostic `json:"diagnostic,omitempty"`
	FalseAcceptance bool                                  `json:"falseAcceptance"`
	Candidate       string                                `json:"candidate"`
	Evidence        CaseEvidence                          `json:"evidence"`
	RequestHash     string                                `json:"requestHash"`
	State           string                                `json:"state"`
	Reason          string                                `json:"reason"`
	Call            *ComparisonCall                       `json:"call,omitempty"`
	Result          *Result                               `json:"result,omitempty"`
	Proposal        *statementdata.Proposal               `json:"proposal,omitempty"`
}
type ComparisonReport struct {
	Proofs        []statementextract.ComparisonIdentityProof `json:"proofs"`
	Version       string                                     `json:"version"`
	Authorization string                                     `json:"authorization"`
	Digest        string                                     `json:"digest"`
	Profile       Profile                                    `json:"profile"`
	Candidates    []statementextract.ComparisonCandidate     `json:"candidates"`
	Time          time.Time                                  `json:"time"`
	FinishedAt    time.Time                                  `json:"finishedAt"`
	ReservationID string                                     `json:"reservationId"`
	MaximumMicros int64                                      `json:"maximumMicros"`
	ActualMicros  *int64                                     `json:"actualMicros"`
	Tuples        []ComparisonTuple                          `json:"tuples"`
	Error         string                                     `json:"error"`
	Passed        bool                                       `json:"passed"`
}

func comparisonDigest(r ComparisonReport) string { r.Digest = ""; return statementextract.Hash(r) }
func comparisonPrepare(ctx context.Context, samples []Sample, reasoners [2]statementextract.Reasoner) (ComparisonReport, []statementextract.Plan, error) {
	return comparisonPrepareFor(ctx, samples, reasoners, legacyComparison)
}
func comparisonPrepareFor(ctx context.Context, samples []Sample, reasoners [2]statementextract.Reasoner, campaign comparisonCampaign) (ComparisonReport, []statementextract.Plan, error) {
	r := ComparisonReport{Version: ComparisonReportVersion, Authorization: campaign.authorization(), Profile: ComparisonProfile(), Candidates: statementextract.ComparisonCandidates(), Time: time.Now().UTC(), Tuples: []ComparisonTuple{}}
	fail := func(code string) (ComparisonReport, []statementextract.Plan, error) {
		r.Error = code
		return r, nil, errors.New(code)
	}
	if len(samples) != 3 {
		return fail("invalid_qualification_corpus")
	}
	for i, n := range ScreeningNames() {
		if samples[i].Name != n || samples[i].SHA256 != FixtureHash(n) {
			return fail("invalid_qualification_fixture")
		}
	}
	for i, c := range r.Candidates {
		if reasoners[i] == nil {
			return fail("comparison_policy_invalid")
		}
		p, e := reasoners[i].CurrentPolicy(ctx)
		if e != nil || p != c.Policy {
			return fail("comparison_policy_invalid")
		}
		provider, ok := reasoners[i].(interface {
			ComparisonProof(context.Context) (statementextract.ComparisonIdentityProof, error)
		})
		if !ok {
			return fail("comparison_policy_invalid")
		}
		proof, e := provider.ComparisonProof(ctx)
		if e != nil || proof.Candidate != c || !proof.ValidAt(time.Now().UTC()) {
			return fail("comparison_policy_invalid")
		}
		r.Proofs = append(r.Proofs, proof)
	}
	plans := []statementextract.Plan{}
	for _, s := range samples {
		for _, c := range r.Candidates {
			p, e := statementextract.BuildComparisonPlan(s.Document, c.Policy)
			if e != nil || len(p.Chunks) != 1 || p.MaximumMicros != c.Policy.CallMicros() {
				return fail("qualification_plan_invalid")
			}
			plans = append(plans, p)
			r.MaximumMicros += p.MaximumMicros
			// Canonical request-input hash: policy + messages + schema. Fixed wire
			// flags/limits are pinned by Profile and exercised by HTTP wire tests.
			request := struct {
				Policy   statementextract.Policy
				Messages []map[string]string
				Schema   any
			}{c.Policy, statementextract.Messages(p.Chunks[0]), statementextract.RequestSchema(p.Chunks[0])}
			r.Tuples = append(r.Tuples, ComparisonTuple{Candidate: c.ID, Evidence: CaseEvidence{s.Name, s.SHA256, s.Document.Hash(), statementextract.Hash(p), statementextract.InputBytes(p.Chunks[0])}, RequestHash: statementextract.Hash(request), State: "not_sent", Reason: "campaign_not_started"})
		}
	}
	if len(plans) != 6 || r.MaximumMicros != ComparisonMaxMicros {
		return fail("qualification_cap_exceeded")
	}
	return r, plans, nil
}
func ComparisonPreflight(ctx context.Context, samples []Sample, reasoners [2]statementextract.Reasoner) (ComparisonReport, error) {
	r, _, e := comparisonPrepare(ctx, samples, reasoners)
	return r, e
}

// RunComparison reserves all six tuples once. Only known, protocol-valid
// proposals may continue after a deterministic quality/grounding failure.
func RunComparison(ctx context.Context, samples []Sample, reasoners [2]statementextract.Reasoner, meter Meter) (ComparisonReport, error) {
	return runComparison(ctx, samples, reasoners, meter, nil)
}
func runComparison(ctx context.Context, samples []Sample, reasoners [2]statementextract.Reasoner, meter Meter, observations []ComparisonObservation) (report ComparisonReport, err error) {
	return runComparisonFor(ctx, samples, reasoners, meter, observations, legacyComparison)
}
func runComparisonFor(ctx context.Context, samples []Sample, reasoners [2]statementextract.Reasoner, meter Meter, observations []ComparisonObservation, campaign comparisonCampaign) (report ComparisonReport, err error) {
	ctx, cancel := context.WithTimeout(ctx, 900*time.Second)
	defer cancel()
	report, plans, err := comparisonPrepareFor(ctx, samples, reasoners, campaign)
	defer func() {
		report.FinishedAt = time.Now().UTC()
		if err != nil {
			report.Error = err.Error()
			report.Passed = false
		}
		report.Digest = comparisonDigest(report)
	}()
	if err != nil {
		return report, err
	}
	allowed, ok := meter.(interface {
		ComparisonAllowed(context.Context, string) (bool, error)
	})
	if meter == nil || !ok {
		return report, errors.New("qualification_budget_unavailable")
	}
	reservation, e := meter.Reserve(ctx, ComparisonMaxMicros, report.Candidates[0].Policy)
	if e != nil || reservation == "" || len(reservation) > 128 {
		return report, errors.New("qualification_budget_unavailable")
	}
	report.ReservationID = reservation
	total := int64(0)
	known := true
	defer func() {
		var actual *int64
		if known {
			n := total
			actual = &n
			report.ActualMicros = &n
		}
		settleCtx, c := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer c()
		if meter.Settle(settleCtx, reservation, actual) != nil {
			err = errors.New("qualification_settlement_failed")
		}
	}()
	ids := map[string]bool{}
	qualityFailed := false
	for i := range report.Tuples {
		tuple := &report.Tuples[i]
		// A context cancellation before Complete is a known zero additional call.
		if ctx.Err() != nil {
			return report, errors.New("qualification_timeout")
		}
		ready, e := allowed.ComparisonAllowed(ctx, reservation)
		if e != nil || !ready {
			return report, errors.New("qualification_budget_unavailable")
		}
		tuple.State = "attempted"
		tuple.Reason = "provider_failed"
		callCtx, c := context.WithTimeout(ctx, 120*time.Second)
		start := time.Now().UTC()
		p := report.Candidates[i%2].Policy
		part, bill, e := reasoners[i%2].Complete(callCtx, p, plans[i].Chunks[0])
		c()
		if e != nil || !validBill(bill, p) {
			diagnostic := statementextract.ScreeningDiagnostic{Code: "unknown"}
			var detail interface {
				ScreeningDiagnostic() statementextract.ScreeningDiagnostic
			}
			if errors.As(e, &detail) {
				diagnostic = detail.ScreeningDiagnostic()
			}
			diagnostic = diagnostic.Sanitize()
			tuple.Diagnostic = &diagnostic
		}
		if !validBill(bill, p) {
			known = false
			return report, errors.New("comparison_provider_failed")
		}
		if ids[bill.GenerationID] {
			known = false
			return report, errors.New("comparison_provider_failed")
		}
		total += bill.CostMicros
		tuple.Call = &ComparisonCall{Route: bill.ComparisonRoute, GenerationID: bill.GenerationID, InputTokens: bill.InputTokens, OutputTokens: bill.OutputTokens, CostMicros: bill.CostMicros, LatencyMS: time.Since(start).Milliseconds()}
		if e != nil || bill.OutputTokens == 0 || !bill.ComparisonRoute.Valid(tuple.Candidate, start, time.Now().UTC()) || ids[bill.GenerationID] {
			return report, errors.New("comparison_provider_failed")
		}
		ids[bill.GenerationID] = true
		// Per-call bounds imply both candidate subcaps and the whole-campaign cap.
		if total > ComparisonMaxMicros {
			return report, errors.New("qualification_cost_overrun")
		}
		if observations != nil {
			observations[i].Snapshot = captureComparisonSnapshot(part)
			observations[i].CaptureStatus = "captured"
			if observations[i].Snapshot == nil {
				observations[i].CaptureStatus = "unavailable_limit"
			}
			observations[i].Stage = "canonicalization_passed"
			var detail *statementextract.CanonicalDiagnostic
			part, detail, e = statementextract.CanonicalComparisonProposalDiagnostic(plans[i], part)
			if e != nil {
				observations[i].Stage = "canonicalization_failed"
				observations[i].Diagnostic = detail
			}
		} else {
			part, e = statementextract.CanonicalComparisonProposal(plans[i], part)
		}
		if e != nil {
			tuple.Reason = "protocol_failed"
			return report, errors.New("qualification_protocol_failed")
		}
		tuple.Proposal = &part
		proposal, e := statementextract.MergeProposals(plans[i], []statementdata.Proposal{part})
		if e != nil {
			tuple.Reason = "protocol_failed"
			return report, errors.New("qualification_protocol_failed")
		}
		pages := []int{}
		for _, pg := range samples[i/2].Document.Pages {
			pages = append(pages, pg.Number)
		}
		review, e := statementdata.ValidateProposal(samples[i/2].Document, proposal, pages, true, "")
		tuple.State = "completed"
		if e != nil {
			tuple.Reason = "grounding_failed"
			qualityFailed = true
			continue
		}
		result := ScoreScreening(tuple.Evidence.Name, review)
		result.LatencyMS = tuple.Call.LatencyMS
		tuple.Result = &result
		tuple.FalseAcceptance = !result.Passed && len(result.Validation) == 0
		tuple.Reason = ""
		if !result.Passed {
			tuple.Reason = "quality_failed"
			qualityFailed = true
		}
	}
	if qualityFailed {
		return report, errors.New("qualification_quality_failed")
	}
	report.Passed = true
	return report, nil
}
