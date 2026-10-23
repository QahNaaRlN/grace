package statementqualification

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"reflect"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/QahNaaRlN/grace/internal/statementdata"
	"github.com/QahNaaRlN/grace/internal/statementextract"
)

func rowPilotClosed(code string) bool {
	if code == "" {
		return true
	}
	for _, s := range strings.Fields("row_pilot_context_invalid row_pilot_metadata_unavailable row_pilot_budget_unavailable row_pilot_call_limit row_pilot_call_unavailable row_pilot_billing_unavailable row_pilot_route_unavailable row_pilot_capture_failed row_pilot_protocol_failed row_pilot_timeout row_pilot_settlement_failed row_pilot_decisive_reject") {
		if code == s {
			return true
		}
	}
	return false
}
func rowPilotReplayAttempt(a RowPilotAttempt, calls []RowPilotCall, d statementdata.Document) (RowPilotMeasurement, error) {
	if a.Measurement == nil {
		return RowPilotMeasurement{}, errRowPilotReport
	}
	if a.Variant == "base" {
		if len(calls) > 1 {
			return RowPilotMeasurement{}, errRowPilotReport
		}
		q, e := statementextract.BuildRowPilotBase(d)
		if e != nil {
			return RowPilotMeasurement{}, errRowPilotReport
		}
		if len(calls) == 0 {
			if a.StopCode == "" {
				return RowPilotMeasurement{}, errRowPilotReport
			}
			return rowPilotFailure("base", "call_unavailable"), nil
		}
		c := calls[0]
		if c.Request != rowPilotEvidence(q) {
			return RowPilotMeasurement{}, errRowPilotReport
		}
		if c.Error != "" {
			return rowPilotFailure("base", "call_unavailable"), nil
		}
		return rowPilotBaseMeasurement(d, c.Raw), nil
	}
	i := 0
	missing := false
	call := func(_ context.Context, q statementextract.RowScopedRequest) ([]byte, *statementextract.Billing, error) {
		if i >= len(calls) {
			missing = true
			return nil, nil, errors.New("replay_refusal")
		}
		c := calls[i]
		i++
		if c.Request != rowPilotEvidence(q) {
			return nil, nil, errRowPilotReport
		}
		if c.Error != "" {
			return nil, c.Bill, errors.New("replay_refusal")
		}
		return append([]byte(nil), c.Raw...), c.Bill, nil
	}
	out, _ := runRowScoped(context.Background(), Sample{"cards", d, FixtureHash("cards")}, RowScopedStages{call, call}, rowPilotLocalMeter{})
	if i != len(calls) || missing && a.StopCode == "" {
		return RowPilotMeasurement{}, errRowPilotReport
	}
	// A mismatched request must never hide behind an expected unavailable result.
	// Reconstruct each consumed request hash from the core's actual call trace.
	if len(out.Calls) < len(calls) {
		return RowPilotMeasurement{}, errRowPilotReport
	}
	for j, c := range calls {
		if out.Calls[j].RequestHash != c.Request.Hash {
			return RowPilotMeasurement{}, errRowPilotReport
		}
	}
	m := rowPilotMeasurement(out)
	if a.Measurement.Cause == "deadline_exceeded" {
		// Timing cannot be replayed as quality. Validate a closed timeout refusal,
		// leaving every score/check/precision absent, after replaying the captured prefix.
		timed := a.StopCode == "row_pilot_timeout"
		for _, c := range calls {
			timed = timed || c.Error == "row_pilot_timeout"
		}
		if !timed || a.Measurement.Result != nil || len(a.Measurement.Checks) != 0 || len(a.Measurement.Precision) != 0 || a.Measurement.FullQualityPassed || a.Measurement.Status != "unavailable" {
			return RowPilotMeasurement{}, errRowPilotReport
		}
		validStage := a.Measurement.Stage == m.Stage
		if len(calls) > 0 {
			last := calls[len(calls)-1].Request
			stage := last.Stage
			if stage == "row" {
				stage = fmt.Sprintf("row%d", last.RowOrdinal)
			}
			validStage = validStage || a.Measurement.Stage == stage
		}
		if !validStage {
			return RowPilotMeasurement{}, errRowPilotReport
		}
		return *a.Measurement, nil
	}
	return m, nil
}

// ValidateRowPilotReport replays actual selections, never trusts a producer score.
// Route/billing attestations are shape/consistency checks, not a cryptographic log.
func ValidateRowPilotReport(r RowPilotReport, d statementdata.Document) error {
	fail := errRowPilotReport
	if r.Version != RowPilotVersion || r.Authorization != RowPilotAuthorization || !regexp.MustCompile(`^[0-9a-f]{40}$`).MatchString(r.Source) || r.Digest != rowPilotDigest(r) || r.MaximumMicros != RowPilotMaxMicros || r.Time.IsZero() || r.FinishedAt.Before(r.Time) || r.FinishedAt.Sub(r.Time) > 1835e9 || r.MetadataRequests < 0 || r.MetadataRequests > 39 || len(r.Attempts) != 6 || r.Calls == nil || len(r.Calls) > 12 || !rowPilotClosed(r.Error) || d.Hash() != statementextract.ClockCardsDocumentHash || r.DocumentHash != d.Hash() {
		return fail
	}
	plan, e := statementextract.BuildComparisonPlan(d, statementextract.ComparisonCandidates()[0].Policy)
	if e != nil || r.PlanHash != statementextract.Hash(plan) {
		return fail
	}
	if r.Proof != nil && (r.Proof.Candidate != statementextract.ComparisonCandidates()[0] || !r.Proof.ValidAt(r.ProofAt) || r.ProofAt.Before(r.Time) || r.ProofAt.After(r.FinishedAt) || r.MetadataRequests < 3) {
		return fail
	}
	if r.ReservationID != "" && (len(r.ReservationID) > 128 || r.Proof == nil) {
		return fail
	}
	if r.ReservationID == "" && (len(r.Calls) > 0 || r.ActualMicros != nil || r.SettlementRecorded) {
		return fail
	}
	ids := map[string]bool{}
	known := true
	total := int64(0)
	previousAttempt := 0
	last := r.Time
	per := make([][]RowPilotCall, 6)
	for i, c := range r.Calls {
		if r.ReservationID == "" || c.Ordinal != i+1 || c.AttemptOrdinal < 1 || c.AttemptOrdinal > 6 || c.AttemptOrdinal < previousAttempt || c.StartedAt.Before(last) || c.FinishedAt.Before(c.StartedAt) || c.FinishedAt.After(r.FinishedAt) || c.FinishedAt.Sub(c.StartedAt) > 121e9 || !rowPilotClosed(c.Error) || c.Proof.Candidate != statementextract.ComparisonCandidates()[0] || !c.Proof.ValidAt(c.StartedAt) || c.RawBytes < 0 || c.RawBytes > 1<<20 {
			return fail
		}
		previousAttempt = c.AttemptOrdinal
		last = c.FinishedAt
		if c.RawBytes == 0 {
			if len(c.Raw) != 0 || c.RawHash != "" {
				return fail
			}
		} else if c.RawBytes <= statementextract.RowScopedResponseLimit {
			if len(c.Raw) != c.RawBytes || statementextract.PromptSHA(string(c.Raw)) != c.RawHash {
				return fail
			}
		} else if len(c.Raw) != 0 || !validHash(c.RawHash) || c.Error == "" {
			return fail
		}
		valid := validBill(c.Bill, plan.Policy)
		if !valid || ids[c.Bill.GenerationID] {
			known = false
			if c.Error != "row_pilot_billing_unavailable" && !(c.Error == "row_pilot_call_unavailable" && c.Bill == nil) {
				return fail
			}
		} else {
			ids[c.Bill.GenerationID] = true
			total += c.Bill.CostMicros
			route := c.Bill.ComparisonRoute
			if c.Error == "" && (c.Bill.OutputTokens < 1 || route == nil || !route.Valid("A", c.StartedAt, c.FinishedAt) || !reflect.DeepEqual(route.Proof, &c.Proof)) {
				return fail
			}
		}
		if c.Error != "" && (i != len(r.Calls)-1 || r.Error != c.Error && r.Error != "row_pilot_settlement_failed") {
			return fail
		}
		per[c.AttemptOrdinal-1] = append(per[c.AttemptOrdinal-1], c)
	}
	if total > RowPilotMaxMicros {
		return fail
	}
	if r.ReservationID != "" {
		if known {
			if r.ActualMicros == nil || *r.ActualMicros != total {
				return fail
			}
		} else if r.ActualMicros != nil {
			return fail
		}
		if !r.SettlementRecorded && r.Error == "" {
			return fail
		}
	}
	stopped := false
	decisive := false
	for i, a := range r.Attempts {
		if a.Ordinal != i+1 || a.Variant != RowPilotOrder()[i] || !rowPilotClosed(a.StopCode) {
			return fail
		}
		if a.State == "not_sent" {
			stopped = true
			if len(per[i]) != 0 || a.Measurement != nil {
				return fail
			}
			if a.StopCode != "" && (!decisive || a.StopCode != "row_pilot_decisive_reject") {
				return fail
			}
			continue
		}
		if stopped || r.ReservationID == "" || a.StopCode == "row_pilot_decisive_reject" {
			return fail
		}
		if a.Measurement == nil {
			if a.State != "attempted" || r.Error != "row_pilot_call_unavailable" || len(per[i]) == 0 || per[i][len(per[i])-1].Bill != nil {
				return fail
			}
			stopped = true
			continue
		}
		replay, e := rowPilotReplayAttempt(a, per[i], d)
		if e != nil || !reflect.DeepEqual(replay, *a.Measurement) {
			return fail
		}
		switch a.State {
		case "completed":
			if replay.Status != "complete" || a.StopCode != "" {
				return fail
			}
		case "rejected":
			if a.Variant != "candidate" || !rowPilotDecisive(replay) || a.StopCode != "" {
				return fail
			}
			stopped = true
			decisive = true
		case "unavailable":
			if r.Error == "" || replay.Result != nil {
				return fail
			}
			stopped = true
		default:
			return fail
		}
	}
	if stopped && !decisive && r.Error == "" {
		return fail
	}
	if r.Decision != rowPilotJudge(r.Attempts, r.Error) {
		return fail
	}
	if r.Decision == RowPilotKeep && (!r.SettlementRecorded || r.ActualMicros == nil) {
		return fail
	}
	return nil
}
func EncodeRowPilotReport(w io.Writer, r RowPilotReport, d statementdata.Document) error {
	if ValidateRowPilotReport(r, d) != nil {
		return errRowPilotReport
	}
	raw, e := json.Marshal(r)
	if e != nil || len(raw)+1 > RowPilotReportLimit {
		return errRowPilotReport
	}
	_, e = w.Write(append(raw, '\n'))
	return e
}
func DecodeRowPilotReport(in io.Reader, d statementdata.Document) (RowPilotReport, error) {
	raw, e := io.ReadAll(io.LimitReader(in, RowPilotReportLimit+1))
	if e != nil || len(raw) > RowPilotReportLimit || !utf8.Valid(raw) || clockUniqueJSON(raw) != nil {
		return RowPilotReport{}, errRowPilotReport
	}
	var r RowPilotReport
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if dec.Decode(&r) != nil || dec.Decode(&struct{}{}) != io.EOF || ValidateRowPilotReport(r, d) != nil {
		return RowPilotReport{}, errRowPilotReport
	}
	normalized, e := json.Marshal(r)
	var a, b any
	if e != nil || json.Unmarshal(raw, &a) != nil || json.Unmarshal(normalized, &b) != nil || !reflect.DeepEqual(a, b) {
		return RowPilotReport{}, errRowPilotReport
	}
	return r, nil
}

// The public DTO has no raw content, provider IDs, proofs or arbitrary error strings.
type RowPilotSummary struct {
	DurationMS                                                       int64
	CallDurationsMS                                                  []int64
	BillingKnown, ReservationSettled                                 bool
	Version, Source, Digest, DocumentHash, PlanHash, Decision, Error string
	ActivationAuthorized                                             bool
	ShapeVerified, ReplayVerified                                    bool
	MaximumMicros                                                    int64
	ActualMicros                                                     *int64
	AttemptedCalls, KnownBills, MetadataRequests                     int
	DeliveredCalls                                                   *int
	SettlementRecorded                                               bool
	Attempts                                                         []RowPilotPublicAttempt
	Requests                                                         []RowPilotRequestEvidence
	Regressions                                                      []RowPilotRegression
}
type RowPilotPublicAttempt struct {
	Ordinal                               int
	Variant, State, StopCode, ReplayScope string
	Measurement                           *RowPilotMeasurement
}
type RowPilotRegression struct {
	Pair       int
	Comparable bool
	Checks     []ScoringCheck
}

func SummarizeRowPilot(r RowPilotReport, d statementdata.Document) (RowPilotSummary, error) {
	if ValidateRowPilotReport(r, d) != nil {
		return RowPilotSummary{}, errRowPilotReport
	}
	out := RowPilotSummary{Version: RowPilotVersion, Source: r.Source, Digest: r.Digest, DocumentHash: r.DocumentHash, PlanHash: r.PlanHash, Decision: r.Decision, Error: r.Error, ShapeVerified: true, ReplayVerified: true, MaximumMicros: r.MaximumMicros, ActualMicros: r.ActualMicros, AttemptedCalls: len(r.Calls), MetadataRequests: r.MetadataRequests, SettlementRecorded: r.SettlementRecorded, Attempts: []RowPilotPublicAttempt{}, Requests: []RowPilotRequestEvidence{}, Regressions: []RowPilotRegression{}}
	out.DurationMS = r.FinishedAt.Sub(r.Time).Milliseconds()
	out.CallDurationsMS = []int64{}
	out.BillingKnown = r.ActualMicros != nil
	out.ReservationSettled = r.SettlementRecorded && out.BillingKnown
	seen := map[string]bool{}
	for _, c := range r.Calls {
		out.CallDurationsMS = append(out.CallDurationsMS, c.FinishedAt.Sub(c.StartedAt).Milliseconds())
		out.Requests = append(out.Requests, c.Request)
		if validBill(c.Bill, statementextract.ComparisonCandidates()[0].Policy) && !seen[c.Bill.GenerationID] {
			out.KnownBills++
			seen[c.Bill.GenerationID] = true
		}
	}
	if out.KnownBills == len(r.Calls) {
		n := len(r.Calls)
		out.DeliveredCalls = &n
	}
	for _, a := range r.Attempts {
		scope := "refusal_consistency"
		if a.Measurement != nil && (a.Measurement.Result != nil || rowPilotSemantic(*a.Measurement)) {
			scope = "measurement"
		}
		out.Attempts = append(out.Attempts, RowPilotPublicAttempt{a.Ordinal, a.Variant, a.State, a.StopCode, scope, a.Measurement})
	}
	for i := 0; i < 6; i += 2 {
		b, c := r.Attempts[i], r.Attempts[i+1]
		if b.Variant == "candidate" {
			b, c = c, b
		}
		reg := RowPilotRegression{Pair: i/2 + 1, Checks: []ScoringCheck{}}
		if b.Measurement != nil && c.Measurement != nil && len(b.Measurement.Checks) == 20 && len(c.Measurement.Checks) == 20 {
			reg.Comparable = true
			for j, x := range b.Measurement.Checks {
				if x.Passed && !c.Measurement.Checks[j].Passed {
					reg.Checks = append(reg.Checks, c.Measurement.Checks[j])
				}
			}
		}
		out.Regressions = append(out.Regressions, reg)
	}
	raw, _ := json.Marshal(out)
	var detached RowPilotSummary
	_ = json.Unmarshal(raw, &detached)
	for _, a := range detached.Attempts {
		if a.Measurement != nil && a.Measurement.Result != nil {
			v := a.Measurement.Result
			v.Codes = rowPilotPublicCodes(v.Codes)
			v.Validation = rowPilotPublicCodes(v.Validation)
			v.Coverage = rowPilotPublicCodes(v.Coverage)
		}
	}
	return detached, nil
}

// DecodeRowPilotRefusal is confined to the fixed CLI's closed pre-campaign codes.
func DecodeRowPilotRefusal(raw []byte) (string, bool) {
	if len(raw) > 256 || !utf8.Valid(raw) || clockUniqueJSON(raw) != nil {
		return "", false
	}
	var v struct {
		Error string `json:"error"`
	}
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	if d.Decode(&v) != nil || d.Decode(&struct{}{}) != io.EOF {
		return "", false
	}
	normalized, _ := json.Marshal(v)
	var a, b any
	if json.Unmarshal(raw, &a) != nil || json.Unmarshal(normalized, &b) != nil || !reflect.DeepEqual(a, b) {
		return "", false
	}
	allowed := rowPilotClosed(v.Error) && v.Error != ""
	for _, code := range strings.Fields("row_pilot_arguments_invalid row_pilot_source_not_authorized row_pilot_authorization_missing row_pilot_fixture_invalid row_pilot_global_ai_unavailable row_pilot_helper_unavailable row_pilot_database_unavailable row_pilot_operator_owner_unavailable row_pilot_provider_unavailable row_pilot_report_invalid row_pilot_report_unavailable row_pilot_mode_invalid") {
		allowed = allowed || v.Error == code
	}
	return v.Error, allowed
}
