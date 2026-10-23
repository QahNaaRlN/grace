package statementqualification

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"reflect"
	"regexp"
	"unicode/utf8"

	"github.com/QahNaaRlN/grace/internal/statementdata"
	"github.com/QahNaaRlN/grace/internal/statementextract"
)

const ClockDiagnosisVersion = "statement.clock-diagnosis.s120.v1"
const ClockProducerSource = "8da7f45b105105d6596032850893bb690febebd8"
const ClockProducerRun int64 = 39983522098
const ClockContextSHA256 = "eefde6959b1bbfe8c94e4559fee1103dfeac98d10c0f415f8673d97b661cdbdf"
const ClockDiagnosisLimit = 65536

var clockSHA40 = regexp.MustCompile(`^[0-9a-f]{40}$`)
var clockSHA64 = regexp.MustCompile(`^[0-9a-f]{64}$`)

// Only closed codes cross the private-report boundary.
func ClockDiagnosisReason(err error) string {
	if err != nil && wordIn(err.Error(), "snapshot_unavailable source_mismatch context_mismatch report_invalid snapshot_changed summary_mismatch unsupported_check_code diagnosis_limit_exceeded output_mismatch unavailable_runtime") {
		return err.Error()
	}
	return "unavailable_runtime"
}

type ClockDiagnosisCheck struct {
	Ordinal    int    `json:"ordinal"`
	RowOrdinal int    `json:"rowOrdinal"`
	Code       string `json:"code"`
	Passed     bool   `json:"passed"`
}
type ClockDiagnosisAttempt struct {
	Ordinal       int                   `json:"ordinal"`
	Variant       string                `json:"variant"`
	ClockCount    int                   `json:"clockCount"`
	Precision     [4]bool               `json:"precision"`
	QualityPassed bool                  `json:"qualityPassed"`
	Correct       int                   `json:"correct"`
	ChecksCount   int                   `json:"checksCount"`
	Checks        []ClockDiagnosisCheck `json:"checks"`
}
type ClockRegression struct {
	Ordinal         int    `json:"ordinal"`
	RowOrdinal      int    `json:"rowOrdinal"`
	Code            string `json:"code"`
	BasePassed      bool   `json:"basePassed"`
	CandidatePassed bool   `json:"candidatePassed"`
}
type ClockPrecisionFailure struct {
	RowOrdinal int    `json:"rowOrdinal"`
	Field      string `json:"field"`
}
type ClockDiagnosisPair struct {
	PairOrdinal                int                     `json:"pairOrdinal"`
	BaseOrdinal                int                     `json:"baseOrdinal"`
	CandidateOrdinal           int                     `json:"candidateOrdinal"`
	NonClockRegressions        []ClockRegression       `json:"nonClockRegressions"`
	PrecisionFailures          []ClockPrecisionFailure `json:"precisionFailures"`
	CandidateClockTargetFailed bool                    `json:"candidateClockTargetFailed"`
}
type ClockDiagnosisAccounting struct {
	CallCostsMicros    [6]int64 `json:"callCostsMicros"`
	ActualMicros       int64    `json:"actualMicros"`
	MaximumMicros      int64    `json:"maximumMicros"`
	BillingKnown       bool     `json:"billingKnown"`
	SettlementRecorded bool     `json:"settlementRecorded"`
}
type ClockDiagnosis struct {
	Version                   string                   `json:"version"`
	ReaderSource              string                   `json:"readerSource"`
	ProducerSource            string                   `json:"producerSource"`
	ReportSHA256              string                   `json:"reportSha256"`
	ReportDigest              string                   `json:"reportDigest"`
	ProducerRun               int64                    `json:"producerRun"`
	Status                    string                   `json:"status"`
	ReplayVerified            int                      `json:"replayVerified"`
	Decision                  string                   `json:"decision"`
	ActivationAuthorized      bool                     `json:"activationAuthorized"`
	Attempts                  []ClockDiagnosisAttempt  `json:"attempts"`
	Pairs                     []ClockDiagnosisPair     `json:"pairs"`
	RejectReasons             []string                 `json:"rejectReasons"`
	Accounting                ClockDiagnosisAccounting `json:"accounting"`
	ProviderCalls             int                      `json:"providerCalls"`
	SQLCalls                  int                      `json:"sqlCalls"`
	PersistentWritesRequested int                      `json:"persistentWritesRequested"`
}
type ClockDiagnosisFailure struct {
	Version              string `json:"version"`
	ReaderSource         string `json:"readerSource"`
	Status               string `json:"status"`
	Reason               string `json:"reason"`
	ActivationAuthorized bool   `json:"activationAuthorized"`
}

// This is a wire-shape allowlist for the existing cards trace, not a scorer.
var clockCheckCodes = []string{"row_count", "account_amount_currency", "direction", "operation_clock", "booking_clock", "record_description", "row_issues", "fx_column", "account_amount_currency", "direction", "operation_clock", "booking_clock", "record_description", "row_issues", "turnover", "coverage", "blockers", "book_balances", "opening_book_value", "closing_book_value"}
var clockCheckRows = []int{0, 1, 1, 1, 1, 1, 1, 1, 2, 2, 2, 2, 2, 2, 0, 0, 0, 0, 0, 0}

func clockCodeKnown(code string) bool {
	for _, c := range clockCheckCodes {
		if code == c {
			return true
		}
	}
	return false
}
func clockCheckIsTime(code string) bool { return code == "operation_clock" || code == "booking_clock" }

// Pair differences depend only on public booleans; both the reader and the
// independent wire validator must retain the complete trace, including shared failures.
func clockPairDifferences(attempts []ClockDiagnosisAttempt) ([]ClockDiagnosisPair, []string) {
	pairs := []ClockDiagnosisPair{}
	target, precision, regression := false, false, false
	for i := 0; i < 6; i += 2 {
		b, c := attempts[i], attempts[i+1]
		if b.Variant == "candidate" {
			b, c = c, b
		}
		p := ClockDiagnosisPair{PairOrdinal: i/2 + 1, BaseOrdinal: b.Ordinal, CandidateOrdinal: c.Ordinal, NonClockRegressions: []ClockRegression{}, PrecisionFailures: []ClockPrecisionFailure{}, CandidateClockTargetFailed: c.ClockCount != 4}
		target = target || p.CandidateClockTargetFailed
		for k, ok := range c.Precision {
			if !ok {
				p.PrecisionFailures = append(p.PrecisionFailures, ClockPrecisionFailure{k/2 + 1, []string{"operation", "booking"}[k%2]})
				precision = true
			}
		}
		for k, x := range b.Checks {
			y := c.Checks[k]
			if !clockCheckIsTime(x.Code) && x.Passed && !y.Passed {
				p.NonClockRegressions = append(p.NonClockRegressions, ClockRegression{x.Ordinal, x.RowOrdinal, x.Code, true, false})
				regression = true
			}
		}
		pairs = append(pairs, p)
	}
	reasons := []string{}
	if target {
		reasons = append(reasons, "candidate_clock_target_failed")
	}
	if precision {
		reasons = append(reasons, "candidate_precision_failed")
	}
	if regression {
		reasons = append(reasons, "non_clock_regression")
	}
	return pairs, reasons
}

// DiagnoseClock accepts only the original strict decoder/replay. The cached
// measurements are copied only after every one has matched an actual replay.
func DiagnoseClock(raw []byte, d statementdata.Document, reader string) (ClockDiagnosis, error) {
	fail := func(code string) (ClockDiagnosis, error) { return ClockDiagnosis{}, errors.New(code) }
	if !clockSHA40.MatchString(reader) {
		return fail("source_mismatch")
	}
	if d.Validate() != nil || d.Hash() != statementextract.ClockCardsDocumentHash {
		return fail("context_mismatch")
	}
	if len(raw) > ClockPilotReportLimit {
		return fail("diagnosis_limit_exceeded")
	}
	r, e := DecodeClockReport(bytes.NewReader(raw), d)
	if e != nil {
		return fail("report_invalid")
	}
	if r.Source != ClockProducerSource {
		return fail("source_mismatch")
	}
	if r.Error != "" || r.Decision != ClockReject || !r.SettlementRecorded || r.ActualMicros == nil {
		return fail("summary_mismatch")
	}
	out := ClockDiagnosis{Version: ClockDiagnosisVersion, ReaderSource: reader, ProducerSource: r.Source, ReportDigest: r.Digest, ProducerRun: ClockProducerRun, Status: "VERIFIED", ReplayVerified: 6, Decision: ClockReject, Attempts: []ClockDiagnosisAttempt{}, Accounting: ClockDiagnosisAccounting{ActualMicros: *r.ActualMicros, MaximumMicros: ClockPilotMaxMicros, BillingKnown: true, SettlementRecorded: true}}
	hash := sha256.Sum256(raw)
	out.ReportSHA256 = hex.EncodeToString(hash[:])
	for i, a := range r.Attempts {
		if a.State != "completed" || a.Call == nil || a.Measurement == nil || !a.Measurement.Comparable || a.Measurement.Count == nil || *a.Measurement.Count != [6]int{0, 4, 4, 0, 0, 4}[i] {
			return fail("summary_mismatch")
		}
		m := a.Measurement
		item := ClockDiagnosisAttempt{Ordinal: a.Ordinal, Variant: a.Variant, ClockCount: *m.Count, Precision: m.Precision, QualityPassed: m.Result.Passed, Correct: m.Result.Correct, ChecksCount: m.Result.Checks, Checks: []ClockDiagnosisCheck{}}
		for _, c := range m.Checks {
			if !clockCodeKnown(c.Code) {
				return fail("unsupported_check_code")
			}
			item.Checks = append(item.Checks, ClockDiagnosisCheck{c.Ordinal, c.RowOrdinal, c.Code, c.Passed})
		}
		out.Attempts = append(out.Attempts, item)
		out.Accounting.CallCostsMicros[i] = a.Call.CostMicros
	}
	out.Pairs, out.RejectReasons = clockPairDifferences(out.Attempts)
	if ValidateClockDiagnosis(out, reader) != nil {
		return fail("output_mismatch")
	}
	return out, nil
}

// ValidateClockDiagnosis recomputes counts, pair differences, reject causes and
// accounting from the closed output. No report, identities or raw values are needed.
func ValidateClockDiagnosis(d ClockDiagnosis, reader string) error {
	fail := errors.New("output_mismatch")
	if !clockSHA40.MatchString(reader) || d.ReaderSource != reader || d.Version != ClockDiagnosisVersion || d.ProducerSource != ClockProducerSource || !clockSHA64.MatchString(d.ReportSHA256) || !clockSHA64.MatchString(d.ReportDigest) || d.ProducerRun != ClockProducerRun || d.Status != "VERIFIED" || d.ReplayVerified != 6 || d.Decision != ClockReject || d.ActivationAuthorized || d.ProviderCalls != 0 || d.SQLCalls != 0 || d.PersistentWritesRequested != 0 || len(d.Attempts) != 6 {
		return fail
	}
	ms := []ClockMeasurement{}
	for i, a := range d.Attempts {
		if a.Ordinal != i+1 || a.Variant != ClockOrder()[i] || len(a.Checks) != 20 || a.ChecksCount != 20 {
			return fail
		}
		count, correct := 0, 0
		checks := []ScoringCheck{}
		for j, c := range a.Checks {
			if c.Ordinal != j+1 || c.RowOrdinal != clockCheckRows[j] || c.Code != clockCheckCodes[j] {
				return fail
			}
			if c.Passed {
				correct++
				if clockCheckIsTime(c.Code) {
					count++
				}
			}
			checks = append(checks, ScoringCheck{c.Ordinal, c.RowOrdinal, c.Code, c.Passed})
		}
		if a.ClockCount != count || count != [6]int{0, 4, 4, 0, 0, 4}[i] || a.Correct != correct || a.QualityPassed != (correct == 20) {
			return fail
		}
		ms = append(ms, ClockMeasurement{Comparable: true, Count: &count, Precision: a.Precision, Checks: checks})
	}
	pairs, reasons := clockPairDifferences(d.Attempts)
	if len(reasons) == 0 || !reflect.DeepEqual(d.Pairs, pairs) || !reflect.DeepEqual(d.RejectReasons, reasons) || DecideClock(ms, true) != ClockReject {
		return fail
	}
	a := d.Accounting
	sum := int64(0)
	if a.MaximumMicros != ClockPilotMaxMicros || !a.BillingKnown || !a.SettlementRecorded {
		return fail
	}
	for _, n := range a.CallCostsMicros {
		if n < 0 || n > ClockPilotMaxMicros-sum {
			return fail
		}
		sum += n
	}
	if a.ActualMicros != sum {
		return fail
	}
	return nil
}

// StrictClockJSON rejects missing/null fields, aliases, duplicate keys, excess
// array elements and silent numeric normalization by exact canonical shape.
func StrictClockJSON(raw []byte, out any, limit int) error {
	fail := errors.New("output_mismatch")
	if len(raw) > limit || !utf8.Valid(raw) || clockUniqueJSON(raw) != nil {
		return fail
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if dec.Decode(out) != nil || dec.Decode(&struct{}{}) != io.EOF {
		return fail
	}
	canonical, e := json.Marshal(out)
	if e != nil {
		return fail
	}
	decode := func(b []byte) any {
		var v any
		d := json.NewDecoder(bytes.NewReader(b))
		d.UseNumber()
		if d.Decode(&v) != nil {
			return nil
		}
		return v
	}
	if !reflect.DeepEqual(decode(raw), decode(canonical)) {
		return fail
	}
	return nil
}

func ValidateClockOutput(raw []byte, reader string) error {
	if len(raw) > ClockDiagnosisLimit {
		return errors.New("diagnosis_limit_exceeded")
	}
	var tag struct {
		Status string `json:"status"`
	}
	if json.Unmarshal(raw, &tag) != nil {
		return errors.New("output_mismatch")
	}
	if tag.Status == "UNVERIFIED" {
		var f ClockDiagnosisFailure
		if StrictClockJSON(raw, &f, ClockDiagnosisLimit) != nil || !clockSHA40.MatchString(reader) || f.Version != ClockDiagnosisVersion || f.ReaderSource != reader || f.Status != "UNVERIFIED" || f.ActivationAuthorized || ClockDiagnosisReason(errors.New(f.Reason)) != f.Reason {
			return errors.New("output_mismatch")
		}
		return nil
	}
	var d ClockDiagnosis
	if StrictClockJSON(raw, &d, ClockDiagnosisLimit) != nil {
		return errors.New("output_mismatch")
	}
	return ValidateClockDiagnosis(d, reader)
}
