package statementqualification

import (
	"encoding/json"
	"errors"
	"regexp"
	"strconv"
)

var qualitySourceSHA = regexp.MustCompile(`^[0-9a-f]{40}$`)

const qualityFailures = "snapshot_unavailable context_mismatch replay_mismatch guard_mismatch unsupported_reason_code diagnosis_limit_exceeded unavailable_runtime"
const qualityScorerCodes = "row_count account_amount_currency direction operation_clock booking_clock record_description row_issues fx_column turnover coverage blockers absent_book_balances available_balance book_balances opening_book_value closing_book_value absent_booking absent_fx printed_debit_turnover printed_credit_turnover absent_book_dates"

func qualityPositionsValid(ps []QualityPosition) bool {
	if len(ps) > 128 {
		return false
	}
	last := 0
	for _, p := range ps {
		if p.SourceOrdinal <= last || p.SourceOrdinal > 2000 || p.Page < 1 || p.Page > 50 || p.Block < 1 || p.Block > 1000000 || p.Line < 1 || p.Line > 1000000 || p.Word < 1 || p.Word > 1000000 || !wordIn(p.LexicalClass, "numeric other") {
			return false
		}
		last = p.SourceOrdinal
	}
	return true
}
func emptyQualityCheck(c QualityCheck) bool {
	return len(c.Positions) == 0 && c.Clock == nil && c.Money == nil && c.State == "" && len(c.Links) == 0
}
func ValidateQualityDiagnosis(d QualityDiagnosis) error {
	fail := errors.New("diagnostic_unavailable")
	if d.Version != QualityDiagnosisVersion || d.Authorization != S117ComparisonAuthorization || d.ArtifactSource != QualityArtifactSource || !qualitySourceSHA.MatchString(d.VerifierSource) || !d.ContextVerified || !d.ArtifactsUnchanged || d.ProviderCalls != 0 || d.PersistentWritesRequested != 0 || len(d.Tuples) != 6 {
		return fail
	}
	for i, t := range d.Tuples {
		if t.Ordinal != i+1 {
			return fail
		}
		if t.Status == "UNVERIFIED" {
			if !wordIn(t.Reason, qualityFailures) || len(t.Checks) != 0 || len(t.Reasons) != 0 || t.Coverage != nil {
				return fail
			}
			continue
		}
		if t.Status != "VERIFIED" || t.Reason != "" || len(t.Reasons) > 128 || len(t.Checks) < 1 || len(t.Checks) > 23 {
			return fail
		}
		if (i < 4) != (t.Coverage != nil) {
			return fail
		}
		if v := t.Coverage; v != nil {
			if v.TotalUnmapped < 0 || v.TotalUnmapped > 2000 {
				return fail
			}
			if v.Status == "UNVERIFIED" {
				if !wordIn(v.Reason, qualityFailures) || len(v.Positions) != 0 {
					return fail
				}
			} else if v.Status != "VERIFIED" || v.Reason != "" || len(v.Positions) != v.TotalUnmapped || !qualityPositionsValid(v.Positions) {
				return fail
			}
		}
		counters := map[string]int{}
		for _, r := range t.Reasons {
			key := r.Scope + ":" + strconv.Itoa(r.RowOrdinal)
			// Blocker indexes span all rows, while row issue indexes restart per row.
			if r.Scope != "row" {
				key = r.Scope
			}
			counters[key]++
			if r.Ordinal != counters[key] || !wordIn(r.Scope, "row blocker coverage turnover book_balance") || r.RowOrdinal < 0 || r.RowOrdinal > 2000 {
				return fail
			}
			if r.Status == "UNVERIFIED" {
				if !wordIn(r.Reason, qualityFailures) || r.Code != "" || r.Field != "" || r.PublicCode != "" || r.Relation != "" || r.Count != nil || len(r.Positions) != 0 {
					return fail
				}
				continue
			}
			if r.Status != "VERIFIED" || r.Reason != "" || !wordIn(r.Code, qualityReasonVocabulary) || !qualityPositionsValid(r.Positions) {
				return fail
			}
			if r.Field != "" && !wordIn(r.Field, "openingBookDate closingBookDate openingBookBalance closingBookBalance availableBalanceObserved debitTurnover creditTurnover service") {
				return fail
			}
			if r.RowOrdinal > 0 && r.Field != "" || r.Scope == "row" && r.RowOrdinal == 0 {
				return fail
			}
			if (r.Code == "unmapped_spans") != (r.Count != nil) {
				return fail
			}
			raw := r.Code
			if r.Count != nil {
				if *r.Count < 0 || *r.Count > 2000 {
					return fail
				}
				raw += ":" + strconv.Itoa(*r.Count)
				if t.Coverage != nil && *r.Count != t.Coverage.TotalUnmapped {
					return fail
				}
			}
			if r.Scope == "blocker" && r.RowOrdinal > 0 {
				raw = "row_" + strconv.Itoa(r.RowOrdinal) + ":" + raw
			}
			if r.Field != "" {
				raw = r.Field + ":" + raw
			}
			if wordIn(r.Scope, "blocker coverage") {
				if r.Relation != "mapped" || r.PublicCode != qualityPublicCode(raw) {
					return fail
				}
			} else if r.Relation != "not_exposed" || r.PublicCode != "" {
				return fail
			}
		}
		previous := 0
		for _, c := range t.Checks {
			if c.Ordinal <= previous || c.Ordinal > []int{19, 19, 20, 20, 23, 23}[i] || c.RowOrdinal < 0 || c.RowOrdinal > 2 || !wordIn(c.Code, qualityScorerCodes) {
				return fail
			}
			previous = c.Ordinal
			if c.Status == "UNVERIFIED" {
				if !wordIn(c.Reason, qualityFailures) || !emptyQualityCheck(c) {
					return fail
				}
				continue
			}
			if c.Status != "VERIFIED" || c.Reason != "" || !qualityPositionsValid(c.Positions) {
				return fail
			}
			last := 0
			for _, n := range c.Links {
				if n <= last || n > len(t.Reasons) {
					return fail
				}
				last = n
			}
			if c.Clock != nil {
				for _, v := range []string{c.Clock.Date, c.Clock.Time, c.Clock.Precision, c.Clock.UTC} {
					if !wordIn(v, "missing matches differs unavailable") {
						return fail
					}
				}
			}
			if c.Money != nil {
				if !wordIn(c.Money.Amount, "missing matches differs unavailable") || !wordIn(c.Money.Currency, "missing matches differs unavailable") {
					return fail
				}
			}
			switch c.Code {
			case "operation_clock", "booking_clock":
				if c.RowOrdinal == 0 || c.Clock == nil || c.Money != nil || c.State != "" || len(c.Links) != 0 {
					return fail
				}
			case "printed_debit_turnover", "printed_credit_turnover":
				if i < 4 || c.RowOrdinal != 0 || c.Money == nil || c.Clock != nil || c.State != "" {
					return fail
				}
			case "row_issues":
				if c.RowOrdinal == 0 || c.Clock != nil || c.Money != nil || c.State != "" || len(c.Links) == 0 {
					return fail
				}
			case "coverage":
				if c.RowOrdinal != 0 || c.Clock != nil || c.Money != nil || !wordIn(c.State, "matched incomplete not_reviewed") {
					return fail
				}
			case "turnover":
				if c.RowOrdinal != 0 || c.Clock != nil || c.Money != nil || !wordIn(c.State, "matched mismatch incomplete not_provided") {
					return fail
				}
			case "blockers":
				if c.RowOrdinal != 0 || c.Clock != nil || c.Money != nil || c.State != "" || len(c.Links) == 0 {
					return fail
				}
			default:
				return fail
			}
		}
	}
	raw, err := json.Marshal(d)
	if err != nil || len(raw)+1 > 65536 {
		return fail
	}
	return nil
}
