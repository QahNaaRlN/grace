package statementqualification

import (
	"github.com/QahNaaRlN/grace/internal/statementdata"
	"regexp"
	"strconv"
)

var groundingPublicRow = regexp.MustCompile(`^row_([1-9][0-9]{0,3}):(description_evidence_gap|description_evidence_unverified)$`)

func groundingPublicCode(code string) bool {
	switch code {
	case "description_evidence_gap", "description_evidence_unverified", "book_balance_role_unverified", "openingBookBalance:book_balance_role_unverified", "closingBookBalance:book_balance_role_unverified":
		return true
	}
	match := groundingPublicRow.FindStringSubmatch(code)
	if match == nil {
		return false
	}
	ordinal, e := strconv.Atoi(match[1])
	return e == nil && ordinal <= statementdata.MaxRows
}
func ScreeningPublicCodes(in []string) []string {
	out := []string{}
	for _, code := range in {
		if groundingPublicCode(code) {
			out = append(out, code)
			continue
		}
		switch code {
		case "row_count", "account_amount_currency", "direction", "operation_clock", "booking_clock", "record_description", "row_issues", "fx_column", "turnover", "coverage", "blockers", "absent_book_balances", "available_balance", "book_balances", "opening_book_value", "closing_book_value", "absent_booking", "absent_fx", "printed_debit_turnover", "printed_credit_turnover", "absent_book_dates",
			"account_amount_required", "aggregate_overflow", "ambiguous_local_time", "ambiguous_money", "amount_overflow", "available_is_not_book_balance", "book_balance_mismatch", "book_balance_not_checked", "book_balance_period_not_proven", "currency_conflict", "currency_required", "description_evidence_limit", "direction_conflict", "direction_required", "field_evidence_limit", "financial_candidate_in_description", "historical_anchor_not_proven", "invalid_currency", "invalid_date", "invalid_direction", "invalid_span_reference", "invalid_timezone", "legacy_field_evidence_invalid", "legacy_service_mapping_missing", "mapping_review_missing", "mapping_snapshot_invalid", "missing_evidence", "no_financial_confirmation", "noncontiguous_date_evidence", "noncontiguous_evidence", "operation_date_required", "overlapping_evidence", "owner_field_role_review_required", "owner_page_review_required", "printed_turnover_mismatch", "service_reason_required", "source_coverage_incomplete", "statement_row_limit", "transaction_direction_conflict", "turnover_not_checked", "turnover_role_unverified", "unsupported_account_currency", "zero_posting":
			out = append(out, code)
		default:
			out = append(out, "unknown")
		}
	}
	return out
}
