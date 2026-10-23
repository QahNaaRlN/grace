package statementqualification

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"regexp"
	"time"

	"github.com/QahNaaRlN/grace/internal/statementdata"
	"github.com/QahNaaRlN/grace/internal/statementextract"
)

const ClockPilotVersion = "statement.clock-pilot.s119.v1"
const ClockPilotAuthorization = "s119-clock-pilot"
const ClockPilotMaxMicros int64 = 323400
const ClockPilotReportLimit = 1048576
const ClockKeep = "KEEP_FOR_FURTHER_REVIEW"
const ClockReject = "REJECT"
const ClockInconclusive = "INCONCLUSIVE"

func ClockOrder() [6]string {
	return [6]string{"base", "candidate", "candidate", "base", "base", "candidate"}
}

type ClockMeasurement struct {
	Result     Result         `json:"result"`
	Checks     []ScoringCheck `json:"checks"`
	Comparable bool           `json:"comparable"`
	Count      *int           `json:"count"`
	Precision  [4]bool        `json:"precision"`
}

// MeasureClock always derives its review and trace from actual raw alias selections.
func MeasureClock(d statementdata.Document, raw statementdata.Proposal) (ClockMeasurement, error) {
	req, e := statementextract.BuildClockRequest(d, "base")
	if e != nil {
		return ClockMeasurement{}, e
	}
	p, e := statementextract.CanonicalComparisonProposal(req.Plan, raw)
	if e != nil {
		return ClockMeasurement{}, errors.New("clock_protocol_failed")
	}
	p, e = statementextract.MergeProposals(req.Plan, []statementdata.Proposal{p})
	if e != nil {
		return ClockMeasurement{}, errors.New("clock_protocol_failed")
	}
	pages := []int{}
	for _, pg := range d.Pages {
		pages = append(pages, pg.Number)
	}
	review, e := statementdata.ValidateProposal(d, p, pages, true, "")
	if e != nil {
		return ClockMeasurement{}, errors.New("clock_validation_unavailable")
	}
	result, checks := ScreeningChecks("cards", review)
	m := ClockMeasurement{Result: result, Checks: checks}
	if len(review.Rows) != 2 {
		return m, nil
	}
	count := 0
	clocks := 0
	for _, c := range checks {
		if c.Code == "operation_clock" || c.Code == "booking_clock" {
			clocks++
			if c.Passed {
				count++
			}
		}
	}
	if clocks != 4 || len(checks) != 20 {
		return m, errors.New("clock_trace_invalid")
	}
	for i, row := range review.Rows {
		m.Precision[2*i] = row.OperationDate != nil && row.OperationDate.Precision == "local_datetime"
		m.Precision[2*i+1] = row.BookingDate != nil && row.BookingDate.Precision == "local_datetime"
	}
	m.Comparable = true
	m.Count = &count
	return m, nil
}
func DecideClock(ms []ClockMeasurement, complete bool) string {
	if !complete || len(ms) != 6 {
		return ClockInconclusive
	}
	for _, m := range ms {
		if !m.Comparable || m.Count == nil || len(m.Checks) != 20 {
			return ClockInconclusive
		}
	}
	gain := false
	order := ClockOrder()
	for i := 0; i < 6; i += 2 {
		b, c := ms[i], ms[i+1]
		if order[i] == "candidate" {
			b, c = c, b
		}
		if *c.Count != 4 {
			return ClockReject
		}
		for _, p := range c.Precision {
			if !p {
				return ClockReject
			}
		}
		if *b.Count < 4 {
			gain = true
		}
		for k, x := range b.Checks {
			y := c.Checks[k]
			if x.Code != y.Code || x.RowOrdinal != y.RowOrdinal || x.Ordinal != y.Ordinal {
				return ClockInconclusive
			}
			if x.Code != "operation_clock" && x.Code != "booking_clock" && x.Passed && !y.Passed {
				return ClockReject
			}
		}
	}
	if gain {
		return ClockKeep
	}
	return ClockInconclusive
}

type ClockRequestEvidence struct {
	Variant, Hash, PromptSHA string
	InputBytes               int
}
type ClockAttempt struct {
	Ordinal     int                       `json:"ordinal"`
	Variant     string                    `json:"variant"`
	State       string                    `json:"state"`
	Reason      string                    `json:"reason"`
	StartedAt   time.Time                 `json:"startedAt"`
	FinishedAt  time.Time                 `json:"finishedAt"`
	Call        *statementextract.Billing `json:"call,omitempty"`
	Snapshot    *statementdata.Proposal   `json:"snapshot,omitempty"`
	Measurement *ClockMeasurement         `json:"measurement,omitempty"`
}
type ClockReport struct {
	Version       string                                    `json:"version"`
	Authorization string                                    `json:"authorization"`
	Source        string                                    `json:"source"`
	Digest        string                                    `json:"digest"`
	DocumentHash  string                                    `json:"documentHash"`
	PlanHash      string                                    `json:"planHash"`
	SchemaHash    string                                    `json:"schemaHash"`
	Requests      []ClockRequestEvidence                    `json:"requests"`
	Proof         *statementextract.ComparisonIdentityProof `json:"proof,omitempty"`
	Time          time.Time                                 `json:"time"`
	FinishedAt    time.Time                                 `json:"finishedAt"`
	ReservationID string                                    `json:"reservationId"`
	MaximumMicros int64                                     `json:"maximumMicros"`
	ActualMicros  *int64                                    `json:"actualMicros"`
	Settled       bool                                      `json:"settled"`
	Attempts      []ClockAttempt                            `json:"attempts"`
	Decision      string                                    `json:"decision"`
	Error         string                                    `json:"error"`
}
type ClockReasoner interface {
	CurrentPolicy(context.Context) (statementextract.Policy, error)
	ComparisonProof(context.Context) (statementextract.ComparisonIdentityProof, error)
	CompleteClock(context.Context, statementdata.Document, string) (statementdata.Proposal, *statementextract.Billing, error)
}
type ClockMeter interface {
	Meter
	ComparisonAllowed(context.Context, string) (bool, error)
}

func clockDigest(r ClockReport) string { r.Digest = ""; return statementextract.Hash(r) }
func clockContext(d statementdata.Document) ([]ClockRequestEvidence, string, string, error) {
	evidence := []ClockRequestEvidence{}
	plan, schema := "", ""
	for _, v := range []string{"base", "candidate"} {
		q, e := statementextract.BuildClockRequest(d, v)
		if e != nil {
			return nil, "", "", e
		}
		plan = statementextract.Hash(q.Plan)
		schema = q.Plan.SchemaHash
		evidence = append(evidence, ClockRequestEvidence{v, q.Hash, q.PromptSHA, q.InputBytes})
	}
	return evidence, plan, schema, nil
}
func RunClockPilot(ctx context.Context, s Sample, reasoner ClockReasoner, meter ClockMeter, source string) (report ClockReport, err error) {
	ctx, cancel := context.WithTimeout(ctx, 900*time.Second)
	defer cancel()
	report = ClockReport{Version: ClockPilotVersion, Authorization: ClockPilotAuthorization, Source: source, Time: time.Now().UTC(), MaximumMicros: ClockPilotMaxMicros, Decision: ClockInconclusive}
	for i, v := range ClockOrder() {
		report.Attempts = append(report.Attempts, ClockAttempt{Ordinal: i + 1, Variant: v, State: "not_sent", Reason: "not_started"})
	}
	defer func() {
		report.FinishedAt = time.Now().UTC()
		if err != nil {
			report.Error = err.Error()
			report.Decision = ClockInconclusive
		}
		report.Digest = clockDigest(report)
	}()
	fail := func(s string) error { return errors.New(s) }
	if !regexp.MustCompile(`^[0-9a-f]{40}$`).MatchString(source) || s.Name != "cards" || s.SHA256 != FixtureHash("cards") || s.Document.Hash() != statementextract.ClockCardsDocumentHash || reasoner == nil || meter == nil {
		return report, fail("clock_context_invalid")
	}
	report.DocumentHash = s.Document.Hash()
	report.Requests, report.PlanHash, report.SchemaHash, err = clockContext(s.Document)
	if err != nil {
		return report, fail("clock_context_invalid")
	}
	policy := statementextract.ComparisonCandidates()[0].Policy
	p, e := reasoner.CurrentPolicy(ctx)
	if e != nil || p != policy {
		return report, fail("clock_preflight_failed")
	}
	proof, e := reasoner.ComparisonProof(ctx)
	if e != nil || proof.Candidate != statementextract.ComparisonCandidates()[0] || !proof.ValidAt(time.Now().UTC()) {
		return report, fail("clock_preflight_failed")
	}
	report.Proof = &proof
	id, e := meter.Reserve(ctx, ClockPilotMaxMicros, policy)
	if e != nil || id == "" || len(id) > 128 {
		return report, fail("clock_budget_unavailable")
	}
	report.ReservationID = id
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
		if meter.Settle(settleCtx, id, actual) != nil {
			err = fail("clock_settlement_failed")
		} else {
			report.Settled = true
		}
	}()
	ids := map[string]bool{}
	ms := []ClockMeasurement{}
	for i := range report.Attempts {
		a := &report.Attempts[i]
		if ctx.Err() != nil {
			return report, fail("clock_timeout")
		}
		ready, e := meter.ComparisonAllowed(ctx, id)
		if e != nil || !ready {
			return report, fail("clock_budget_unavailable")
		}
		a.State = "attempted"
		a.Reason = "provider_failed"
		a.StartedAt = time.Now().UTC()
		var doc statementdata.Document
		raw, _ := json.Marshal(s.Document)
		_ = json.Unmarshal(raw, &doc)
		callCtx, c := context.WithTimeout(ctx, 120*time.Second)
		proposal, bill, e := reasoner.CompleteClock(callCtx, doc, a.Variant)
		c()
		a.FinishedAt = time.Now().UTC()
		if !validBill(bill, policy) || ids[bill.GenerationID] {
			known = false
			return report, fail("clock_provider_failed")
		}
		ids[bill.GenerationID] = true
		total += bill.CostMicros
		raw, _ = json.Marshal(bill)
		var copyBill statementextract.Billing
		_ = json.Unmarshal(raw, &copyBill)
		a.Call = &copyBill
		if e != nil && bill.OutputTokens > 0 && bill.ComparisonRoute.Valid("A", a.StartedAt, a.FinishedAt) {
			a.Reason = "protocol_failed"
			return report, fail("clock_protocol_failed")
		}
		if e != nil || bill.OutputTokens == 0 || !bill.ComparisonRoute.Valid("A", a.StartedAt, a.FinishedAt) {
			return report, fail("clock_provider_failed")
		}
		if total > ClockPilotMaxMicros {
			return report, fail("clock_cost_overrun")
		}
		a.Snapshot = captureComparisonSnapshot(proposal)
		if a.Snapshot == nil {
			a.Reason = "capture_failed"
			return report, fail("clock_capture_failed")
		}
		m, e := MeasureClock(s.Document, *a.Snapshot)
		if e != nil {
			a.Reason = "protocol_failed"
			return report, fail("clock_protocol_failed")
		}
		a.Measurement = &m
		a.State = "completed"
		a.Reason = "quality_failed"
		if m.Result.Passed {
			a.Reason = "quality_passed"
		}
		ms = append(ms, m)
	}
	report.Decision = DecideClock(ms, true)
	return report, nil
}

// ValidateClockReport replays every completed snapshot. A report is evidence,
// never an authorization token, and its private fields never enter the summary.
func ValidateClockReport(r ClockReport, d statementdata.Document) error {
	fail := errors.New("clock_report_invalid")
	if r.Version != ClockPilotVersion || r.Authorization != ClockPilotAuthorization || !regexp.MustCompile(`^[0-9a-f]{40}$`).MatchString(r.Source) || r.Digest != clockDigest(r) || r.MaximumMicros != ClockPilotMaxMicros || r.FinishedAt.Before(r.Time) || len(r.Attempts) != 6 {
		return fail
	}
	requests, plan, schema, e := clockContext(d)
	if e != nil || d.Hash() != statementextract.ClockCardsDocumentHash || r.DocumentHash != d.Hash() || r.PlanHash != plan || r.SchemaHash != schema || !reflect.DeepEqual(requests, r.Requests) {
		return fail
	}
	if r.ReservationID == "" {
		if r.ActualMicros != nil || r.Settled || r.Decision != ClockInconclusive || !wordIn(r.Error, "clock_preflight_failed clock_budget_unavailable") {
			return fail
		}
		for i, a := range r.Attempts {
			if a.Ordinal != i+1 || a.Variant != ClockOrder()[i] || a.Reason != "not_started" || !a.StartedAt.IsZero() || !a.FinishedAt.IsZero() || a.State != "not_sent" || a.Snapshot != nil || a.Call != nil || a.Measurement != nil {
				return fail
			}
		}
		if r.Proof != nil && (!r.Proof.ValidAt(r.FinishedAt) || r.Proof.Candidate != statementextract.ComparisonCandidates()[0]) {
			return fail
		}
		return nil
	}
	if r.Proof == nil || r.Proof.Candidate != statementextract.ComparisonCandidates()[0] || !r.Proof.ValidAt(r.Proof.VerifiedAt) || r.Proof.VerifiedAt.After(r.FinishedAt) {
		return fail
	}
	ms := []ClockMeasurement{}
	total := int64(0)
	seen := map[string]bool{}
	stopped := false
	attempted := 0
	for i, a := range r.Attempts {
		if a.Snapshot != nil && !snapshotWithinBounds(*a.Snapshot) {
			return fail
		}
		if a.Ordinal != i+1 || a.Variant != ClockOrder()[i] {
			return fail
		}
		if a.State == "not_sent" {
			if a.Reason != "not_started" || a.Call != nil || a.Snapshot != nil || a.Measurement != nil || !a.StartedAt.IsZero() || !a.FinishedAt.IsZero() {
				return fail
			}
			stopped = true
			continue
		}
		if stopped || a.StartedAt.Before(r.Time) || a.FinishedAt.Before(a.StartedAt) || a.FinishedAt.After(r.FinishedAt) {
			return fail
		}
		attempted++
		if a.Call != nil {
			if !validBill(a.Call, statementextract.ComparisonCandidates()[0].Policy) || seen[a.Call.GenerationID] {
				return fail
			}
			seen[a.Call.GenerationID] = true
			total += a.Call.CostMicros
		}
		if a.State == "completed" {
			if a.Call == nil || !a.Call.ComparisonRoute.Valid("A", a.StartedAt, a.FinishedAt) || a.Snapshot == nil || !snapshotWithinBounds(*a.Snapshot) || a.Measurement == nil {
				return fail
			}
			m, e := MeasureClock(d, *a.Snapshot)
			if e != nil || !reflect.DeepEqual(m, *a.Measurement) {
				return fail
			}
			ms = append(ms, m)
			reason := "quality_failed"
			if m.Result.Passed {
				reason = "quality_passed"
			}
			if a.Reason != reason {
				return fail
			}
		} else if a.State == "attempted" {
			if a.Measurement != nil || !wordIn(a.Reason, "provider_failed capture_failed protocol_failed") {
				return fail
			}
			stopped = true
		} else {
			return fail
		}
	}
	if total > ClockPilotMaxMicros || r.ActualMicros != nil && *r.ActualMicros != total {
		return fail
	}
	if r.ReservationID == "" || len(r.ReservationID) > 128 {
		return fail
	}
	if r.Error == "" {
		if attempted != 6 || len(ms) != 6 || !r.Settled || r.ActualMicros == nil || r.Decision != DecideClock(ms, true) {
			return fail
		}
	} else {
		if !wordIn(r.Error, "clock_timeout clock_budget_unavailable clock_provider_failed clock_cost_overrun clock_capture_failed clock_protocol_failed clock_settlement_failed") || r.Decision != ClockInconclusive {
			return fail
		}
	}
	raw, e := json.Marshal(r)
	if e != nil || len(raw)+1 > ClockPilotReportLimit {
		return fail
	}
	return nil
}
