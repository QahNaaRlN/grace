package statementqualification

import (
	"bytes"
	"encoding/json"
	"io"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/QahNaaRlN/grace/internal/statementdata"
	"github.com/QahNaaRlN/grace/internal/statementextract"
)

// Finite code redaction retains row ordinals, never text from a model/source.
func rowPilotPublicCodes(in []string) []string {
	out := []string{}
	for _, code := range in {
		safe := ScreeningPublicCodes([]string{code})[0]
		if safe == "unknown" {
			p := strings.SplitN(code, ":", 2)
			if len(p) == 2 && strings.HasPrefix(p[0], "row_") {
				n, e := strconv.Atoi(strings.TrimPrefix(p[0], "row_"))
				if e == nil && n > 0 && n <= statementdata.MaxRows && ScreeningPublicCodes([]string{p[1]})[0] != "unknown" {
					safe = "row_" + strconv.Itoa(n) + ":" + p[1]
				}
			}
		}
		out = append(out, safe)
	}
	return out
}
func rowPilotPublicMeasurement(m *RowPilotMeasurement) bool {
	if m == nil {
		return true
	}
	if !wordIn(m.Stage, "base grouping row1 row2 composition validation") || !wordIn(m.Status, "complete unavailable rejected incomplete") {
		return false
	}
	if m.Status != "complete" {
		return m.Result == nil && len(m.Checks) == 0 && len(m.Precision) == 0 && !m.FullQualityPassed && wordIn(m.Cause, "partition_invalid partition_limit partition_unresolved row_subset_invalid row_coverage_incomplete context_unused request_limit_exceeded response_invalid call_unavailable billing_unavailable deadline_exceeded qualification_protocol_failed qualification_merge_failed qualification_grounding_failed")
	}
	if m.Cause != "" || m.Stage != "validation" || m.Result == nil {
		return false
	}
	r := m.Result
	if r.Name != "cards" || r.Rows < 0 || r.Rows > statementdata.MaxRows || len(m.Precision) != r.Rows || r.LatencyMS < 0 || r.LatencyMS > 1800000 {
		return false
	}
	expected := []int{0}
	if r.Rows > 0 {
		for i := 1; i < 8; i++ {
			expected = append(expected, i)
		}
	}
	if r.Rows > 1 {
		for i := 8; i < 14; i++ {
			expected = append(expected, i)
		}
	}
	for i := 14; i < 20; i++ {
		expected = append(expected, i)
	}
	if r.Checks != len(expected) || len(m.Checks) != len(expected) {
		return false
	}
	correct := 0
	codes := []string{}
	for i, j := range expected {
		c := m.Checks[i]
		if c.Ordinal != i+1 || c.RowOrdinal != clockCheckRows[j] || c.Code != clockCheckCodes[j] {
			return false
		}
		if c.Passed {
			correct++
		} else {
			codes = append(codes, c.Code)
		}
	}
	if r.Correct != correct || r.Passed != (correct == r.Checks) || !reflect.DeepEqual(r.Codes, codes) || !reflect.DeepEqual(rowPilotPublicCodes(r.Validation), r.Validation) || !reflect.DeepEqual(rowPilotPublicCodes(r.Coverage), r.Coverage) {
		return false
	}
	full := r.Passed && r.Rows == 2
	for i, p := range m.Precision {
		if p.RowOrdinal != i+1 {
			return false
		}
		full = full && p.OperationMatches && p.BookingMatches
	}
	return m.FullQualityPassed == full
}
func rowPilotPublicRegressions(as []RowPilotPublicAttempt) []RowPilotRegression {
	out := []RowPilotRegression{}
	for i := 0; i < 6; i += 2 {
		b, c := as[i], as[i+1]
		if b.Variant == "candidate" {
			b, c = c, b
		}
		v := RowPilotRegression{Pair: i/2 + 1, Checks: []ScoringCheck{}}
		if b.Measurement != nil && c.Measurement != nil && len(b.Measurement.Checks) == 20 && len(c.Measurement.Checks) == 20 {
			v.Comparable = true
			for j, x := range b.Measurement.Checks {
				if x.Passed && !c.Measurement.Checks[j].Passed {
					v.Checks = append(v.Checks, c.Measurement.Checks[j])
				}
			}
		}
		out = append(out, v)
	}
	return out
}

// ValidateRowPilotSummary validates only a closed DTO emitted after VPS replay.
// It does not attest remote execution or repeat private capture replay in CI.
func ValidateRowPilotSummary(s RowPilotSummary, source string) error {
	fail := errRowPilotReport
	if !regexp.MustCompile(`^[0-9a-f]{40}$`).MatchString(source) || s.Source != source || s.Version != RowPilotVersion || s.ActivationAuthorized || !s.ShapeVerified || !s.ReplayVerified || !validHash(s.Digest) || s.DocumentHash != "ff6f0157ec84d773ed788efd67f4635e111425b5146ee2624e6e45aab538a4e9" || s.PlanHash != "f0f25fe36340438e13ec088a90230a6499041cb26d82424b800c5b45bd86a4db" || s.MaximumMicros != RowPilotMaxMicros || s.DurationMS < 0 || s.DurationMS > 1835000 || s.MetadataRequests < 0 || s.MetadataRequests > 39 || s.AttemptedCalls < 0 || s.AttemptedCalls > 12 || s.KnownBills < 0 || s.KnownBills > s.AttemptedCalls || len(s.Requests) != s.AttemptedCalls || len(s.CallDurationsMS) != s.AttemptedCalls || len(s.Attempts) != 6 || !rowPilotClosed(s.Error) {
		return fail
	}
	if s.BillingKnown != (s.ActualMicros != nil) || s.ReservationSettled != (s.SettlementRecorded && s.BillingKnown) {
		return fail
	}
	if s.ActualMicros != nil && (*s.ActualMicros < 0 || *s.ActualMicros > RowPilotMaxMicros || s.KnownBills != s.AttemptedCalls) {
		return fail
	}
	if (s.DeliveredCalls != nil) != (s.KnownBills == s.AttemptedCalls) {
		return fail
	}
	if s.DeliveredCalls != nil && *s.DeliveredCalls != s.AttemptedCalls {
		return fail
	}
	for i, q := range s.Requests {
		prompt := statementextract.ClockBasePromptSHA
		if q.Stage == "grouping" {
			prompt = statementextract.RowGroupingPromptSHA
		}
		if q.Stage == "row" {
			prompt = statementextract.RowFieldPromptSHA
		}
		if q.PromptSHA != prompt || q.Stage != "row" && q.RowOrdinal != 0 || q.Stage == "row" && q.RowOrdinal == 0 {
			return fail
		}
		if !wordIn(q.Stage, "base grouping row") || q.RowOrdinal < 0 || q.RowOrdinal > 2 || q.InputBytes <= 0 || q.InputBytes > 90000 || !validHash(q.Hash) || !validHash(q.PromptSHA) || !validHash(q.SchemaSHA) || s.CallDurationsMS[i] < 0 || s.CallDurationsMS[i] > 121000 {
			return fail
		}
	}
	as := []RowPilotAttempt{}
	stopped := false
	decisive := false
	for i, a := range s.Attempts {
		if a.Ordinal != i+1 || a.Variant != RowPilotOrder()[i] || !wordIn(a.State, "not_sent attempted completed rejected unavailable") || !rowPilotClosed(a.StopCode) || !wordIn(a.ReplayScope, "measurement refusal_consistency") || !rowPilotPublicMeasurement(a.Measurement) {
			return fail
		}
		if a.State == "not_sent" {
			stopped = true
			if a.Measurement != nil {
				return fail
			}
		} else if stopped {
			return fail
		}
		if a.State == "completed" && (a.Measurement == nil || a.Measurement.Status != "complete") {
			return fail
		}
		if a.State == "rejected" && (a.Variant != "candidate" || a.Measurement == nil || !rowPilotDecisive(*a.Measurement)) {
			return fail
		}
		if a.State != "not_sent" && wordIn(a.State, "attempted rejected unavailable") {
			stopped = true
			decisive = a.State == "rejected"
		}
		if a.State == "not_sent" && a.StopCode != "" && (!decisive || a.StopCode != "row_pilot_decisive_reject") {
			return fail
		}
		if a.State == "not_sent" && a.StopCode == "" && decisive {
			return fail
		}
		if a.State == "unavailable" && s.Error == "" {
			return fail
		}
		if a.State == "completed" && a.StopCode != "" {
			return fail
		}
		if a.State == "rejected" && a.StopCode != "" {
			return fail
		}
		if a.State == "attempted" && (s.Error != "row_pilot_call_unavailable" || a.Measurement != nil) {
			return fail
		}
		if a.ReplayScope == "measurement" && (a.Measurement == nil || a.Measurement.Result == nil && !rowPilotSemantic(*a.Measurement)) {
			return fail
		}
		as = append(as, RowPilotAttempt{a.Ordinal, a.Variant, a.State, a.StopCode, a.Measurement})
	}
	if stopped && !decisive && s.Error == "" {
		return fail
	}
	if s.Decision != rowPilotJudge(as, s.Error) || !reflect.DeepEqual(s.Regressions, rowPilotPublicRegressions(s.Attempts)) {
		return fail
	}
	return nil
}
func DecodeRowPilotSummary(in io.Reader, source string) (RowPilotSummary, error) {
	raw, e := io.ReadAll(io.LimitReader(in, RowPilotReportLimit+1))
	if e != nil || len(raw) > RowPilotReportLimit || !utf8.Valid(raw) || clockUniqueJSON(raw) != nil {
		return RowPilotSummary{}, errRowPilotReport
	}
	var s RowPilotSummary
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	if d.Decode(&s) != nil || d.Decode(&struct{}{}) != io.EOF || ValidateRowPilotSummary(s, source) != nil {
		return RowPilotSummary{}, errRowPilotReport
	}
	normalized, _ := json.Marshal(s)
	var a, b any
	if json.Unmarshal(raw, &a) != nil || json.Unmarshal(normalized, &b) != nil || !reflect.DeepEqual(a, b) {
		return RowPilotSummary{}, errRowPilotReport
	}
	return s, nil
}

// Unavailable execution has unknown calls/cost/settlement, never invented zeroes.
type RowPilotRefusal struct {
	Version              string `json:"version"`
	Source               string `json:"source"`
	ActivationAuthorized bool   `json:"activationAuthorized"`
	ReplayVerified       bool   `json:"replayVerified"`
	Decision             string `json:"decision"`
	Error                string `json:"error"`
	RecordedCalls        *int   `json:"recordedCalls"`
	ActualMicros         *int64 `json:"actualMicros"`
	SettlementRecorded   *bool  `json:"settlementRecorded"`
}

func NewRowPilotRefusal(source, code string) RowPilotRefusal {
	raw, _ := json.Marshal(map[string]string{"error": code})
	if _, ok := DecodeRowPilotRefusal(raw); !ok {
		code = "row_pilot_report_unavailable"
	}
	return RowPilotRefusal{Version: "statement.row-pilot.s121.refusal.v1", Source: source, Decision: ClockInconclusive, Error: code}
}
func DecodeRowPilotPublicRefusal(raw []byte, source string) (RowPilotRefusal, error) {
	var r RowPilotRefusal
	if len(raw) > 1024 || !utf8.Valid(raw) || clockUniqueJSON(raw) != nil {
		return r, errRowPilotReport
	}
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	if d.Decode(&r) != nil || d.Decode(&struct{}{}) != io.EOF || !reflect.DeepEqual(r, NewRowPilotRefusal(source, r.Error)) {
		return RowPilotRefusal{}, errRowPilotReport
	}
	normalized, _ := json.Marshal(r)
	var a, b any
	if json.Unmarshal(raw, &a) != nil || json.Unmarshal(normalized, &b) != nil || !reflect.DeepEqual(a, b) {
		return RowPilotRefusal{}, errRowPilotReport
	}
	return r, nil
}
