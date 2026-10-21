package statementqualification

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"reflect"
	"time"
	"unicode/utf8"

	"github.com/QahNaaRlN/grace/internal/statementdata"
	"github.com/QahNaaRlN/grace/internal/statementextract"
)

const ObservedComparisonVersion = "statement.comparison.observed.v1"
const ComparisonArtifactLimit = 524288
const ComparisonSnapshotLimit = 32768

type ComparisonObservation struct {
	Ordinal       int                                   `json:"ordinal"`
	Stage         string                                `json:"stage"`
	CaptureStatus string                                `json:"captureStatus"`
	Snapshot      *statementdata.Proposal               `json:"snapshot,omitempty"`
	Diagnostic    *statementextract.CanonicalDiagnostic `json:"diagnostic,omitempty"`
}
type ObservedComparisonReport struct {
	Version      string                  `json:"version"`
	Report       ComparisonReport        `json:"report"`
	Observations []ComparisonObservation `json:"observations"`
	Digest       string                  `json:"digest"`
}

func observedDigest(e ObservedComparisonReport) string {
	e.Digest = ""
	return statementextract.Hash(e)
}

// Called only by the synthetic operator path. Product/legacy runners do not capture.
func RunObservedComparison(ctx context.Context, samples []Sample, reasoners [2]statementextract.Reasoner, meter Meter) (ObservedComparisonReport, error) {
	return runObservedComparisonFor(ctx, samples, reasoners, meter, legacyComparison)
}
func runObservedComparisonFor(ctx context.Context, samples []Sample, reasoners [2]statementextract.Reasoner, meter Meter, campaign comparisonCampaign) (ObservedComparisonReport, error) {
	observations := make([]ComparisonObservation, 6)
	for i := range observations {
		observations[i] = ComparisonObservation{Ordinal: i + 1, Stage: "not_reached", CaptureStatus: "not_reached"}
	}
	r, err := runComparisonFor(ctx, samples, reasoners, meter, observations, campaign)
	e := ObservedComparisonReport{Version: campaign.version(), Report: r, Observations: observations}
	trimObservedSnapshots(&e)
	return e, err
}
func snapshotWithinBounds(p statementdata.Proposal) bool {
	if len(p.Rows) > 5000 || len(p.Service) > 2000 {
		return false
	}
	text := func(s string, max int) bool { return utf8.ValidString(s) && len(s) <= max }
	selection := func(ids statementdata.Selection) bool {
		if len(ids) > 2000 {
			return false
		}
		for _, s := range ids {
			if !text(s, 64) {
				return false
			}
		}
		return true
	}
	for _, r := range p.Rows {
		if !text(r.Direction, 64) {
			return false
		}
		for _, ids := range []statementdata.Selection{r.OperationDate, r.BookingDate, r.AccountAmount, r.AccountCurrency, r.TransactionAmount, r.TransactionCurrency, r.Description, r.DirectionEvidence} {
			if !selection(ids) {
				return false
			}
		}
	}
	h := p.Header
	for _, ids := range []statementdata.Selection{h.OpeningBookDate, h.ClosingBookDate, h.OpeningBookBalance, h.ClosingBookBalance, h.AvailableBalanceObserved, h.DebitTurnover, h.CreditTurnover, h.Currency} {
		if !selection(ids) {
			return false
		}
	}
	for role, ids := range h.Labels {
		if !text(role, 128) || !selection(ids) {
			return false
		}
	}
	for _, s := range p.Service {
		if !text(s.Reason, 2000) || !selection(s.Spans) {
			return false
		}
	}
	raw, err := json.Marshal(p)
	return err == nil && len(raw) <= ComparisonSnapshotLimit
}
func captureComparisonSnapshot(p statementdata.Proposal) *statementdata.Proposal {
	if !snapshotWithinBounds(p) {
		return nil
	}
	raw, _ := json.Marshal(p)
	var out statementdata.Proposal
	if json.Unmarshal(raw, &out) != nil {
		return nil
	}
	return &out
}

// Removing whole snapshots never changes the underlying v1 report or metering.
func trimObservedSnapshots(e *ObservedComparisonReport) {
	for i := len(e.Observations) - 1; i >= -1; i-- {
		e.Digest = observedDigest(*e)
		raw, err := json.Marshal(e)
		if err == nil && len(raw) < ComparisonArtifactLimit {
			return
		}
		if i >= 0 && e.Observations[i].Snapshot != nil {
			e.Observations[i].Snapshot = nil
			e.Observations[i].CaptureStatus = "unavailable_report_limit"
		}
	}
	e.Digest = observedDigest(*e)
}
func ValidateObservedComparison(e ObservedComparisonReport, now time.Time) error {
	return validateObservedComparisonFor(e, now, legacyComparison)
}
func validateObservedComparisonFor(e ObservedComparisonReport, now time.Time, campaign comparisonCampaign) error {
	fail := errors.New("invalid_observed_comparison")
	raw, err := json.Marshal(e)
	if err != nil || len(raw) >= ComparisonArtifactLimit || !observedJSONBounds(raw) || e.Version != campaign.version() || e.Digest != observedDigest(e) || len(e.Observations) != 6 || validateComparisonReportFor(e.Report, now, campaign) != nil {
		return fail
	}
	for i, o := range e.Observations {
		t := e.Report.Tuples[i]
		if o.Ordinal != i+1 {
			return fail
		}
		if o.Stage == "not_reached" {
			if o.CaptureStatus != "not_reached" || o.Snapshot != nil || o.Diagnostic != nil || t.Proposal != nil || t.State == "completed" || t.Reason == "protocol_failed" {
				return fail
			}
			continue
		}
		if t.Diagnostic != nil || t.Call == nil || t.Call.OutputTokens == 0 || !t.Call.Route.Valid(t.Candidate, e.Report.Time, e.Report.FinishedAt) || t.State == "not_sent" {
			return fail
		}
		switch o.CaptureStatus {
		case "captured":
			if o.Snapshot == nil || !snapshotWithinBounds(*o.Snapshot) {
				return fail
			}
		case "unavailable_limit", "unavailable_report_limit":
			if o.Snapshot != nil {
				return fail
			}
		default:
			return fail
		}
		switch o.Stage {
		case "canonicalization_failed":
			if t.State != "attempted" || t.Reason != "protocol_failed" || t.Proposal != nil || o.Diagnostic == nil || !o.Diagnostic.Valid() {
				return fail
			}
		case "canonicalization_passed":
			if t.Proposal == nil || o.Diagnostic != nil {
				return fail
			}
		default:
			return fail
		}
	}
	return nil
}

// Both formats remain explicit. A bare v1 has no inferred observation data.
func DecodeComparisonArtifact(reader io.Reader) (ComparisonReport, *ObservedComparisonReport, error) {
	return decodeComparisonArtifactFor(reader, legacyComparison, true)
}
func decodeComparisonArtifactFor(reader io.Reader, campaign comparisonCampaign, allowBare bool) (ComparisonReport, *ObservedComparisonReport, error) {
	raw, err := io.ReadAll(io.LimitReader(reader, ComparisonArtifactLimit+1))
	if err != nil || len(raw) > ComparisonArtifactLimit {
		return ComparisonReport{}, nil, errors.New("invalid_comparison_artifact")
	}
	var header struct {
		Version string `json:"version"`
	}
	if json.Unmarshal(raw, &header) != nil {
		return ComparisonReport{}, nil, errors.New("invalid_comparison_artifact")
	}
	if allowBare && header.Version == ComparisonReportVersion {
		r, err := DecodeComparisonReport(bytes.NewReader(raw))
		return r, nil, err
	}
	var e ObservedComparisonReport
	if header.Version != campaign.version() || !observedJSONBounds(raw) || decodeOperatorReport(bytes.NewReader(raw), &e) != nil || validateObservedComparisonFor(e, time.Now().UTC(), campaign) != nil {
		return ComparisonReport{}, nil, errors.New("invalid_comparison_artifact")
	}
	return e.Report, &e, nil
}
func EncodeObservedComparison(w io.Writer, e ObservedComparisonReport) error {
	return encodeObservedComparisonFor(w, e, legacyComparison)
}
func encodeObservedComparisonFor(w io.Writer, e ObservedComparisonReport, campaign comparisonCampaign) error {
	if validateObservedComparisonFor(e, time.Now().UTC(), campaign) != nil {
		return errors.New("comparison_artifact_unverified")
	}
	raw, err := json.Marshal(e)
	if err != nil {
		return errors.New("comparison_artifact_unverified")
	}
	raw = append(raw, '\n')
	if len(raw) > ComparisonArtifactLimit {
		return errors.New("comparison_artifact_unverified")
	}
	_, err = w.Write(raw)
	return err
}

// Closed observations only: neither private snapshot nor arbitrary error strings.
func PublicComparisonObservations(e *ObservedComparisonReport, now time.Time) ([]ComparisonObservation, bool) {
	if e == nil {
		return nil, true
	}
	if validateKnownObservedComparison(*e, now) != nil {
		return nil, false
	}
	out := make([]ComparisonObservation, len(e.Observations))
	for i, o := range e.Observations {
		out[i] = o
		out[i].Snapshot = nil
	}
	return out, true
}

type ComparisonReplay struct {
	Stage      string                                `json:"stage,omitempty"`
	Verified   bool                                  `json:"verified"`
	Status     string                                `json:"status"`
	Diagnostic *statementextract.CanonicalDiagnostic `json:"diagnostic,omitempty"`
}

func ReplayComparisonObservation(e *ObservedComparisonReport, index int, document statementdata.Document, now time.Time) ComparisonReplay {
	unavailable := ComparisonReplay{Status: "UNVERIFIED"}
	if e == nil {
		return ComparisonReplay{Status: "unavailable_legacy"}
	}
	if validateKnownObservedComparison(*e, now) != nil || index < 0 || index >= 6 {
		return unavailable
	}
	o := e.Observations[index]
	t := e.Report.Tuples[index]
	if o.Snapshot == nil {
		return ComparisonReplay{Status: "unavailable_capture", Stage: o.Stage}
	}
	p, err := statementextract.BuildComparisonPlan(document, e.Report.Candidates[index%2].Policy)
	request := struct {
		Policy   statementextract.Policy
		Messages []map[string]string
		Schema   any
	}{}
	if err != nil || len(p.Chunks) != 1 {
		return unavailable
	}
	request.Policy = p.Policy
	request.Messages = statementextract.Messages(p.Chunks[0])
	request.Schema = statementextract.RequestSchema(p.Chunks[0])
	if document.Hash() != t.Evidence.DocumentHash || statementextract.Hash(p) != t.Evidence.PlanHash || statementextract.InputBytes(p.Chunks[0]) != t.Evidence.InputBytes || statementextract.Hash(request) != t.RequestHash {
		return unavailable
	}
	part, diag, err := statementextract.CanonicalComparisonProposalDiagnostic(p, *o.Snapshot)
	if o.Stage == "canonicalization_failed" {
		if err == nil || !reflect.DeepEqual(diag, o.Diagnostic) {
			return unavailable
		}
	} else {
		if err != nil || t.Proposal == nil || !reflect.DeepEqual(part, *t.Proposal) {
			return unavailable
		}
	}
	return ComparisonReplay{Verified: true, Status: "matched", Stage: o.Stage, Diagnostic: diag}
}
