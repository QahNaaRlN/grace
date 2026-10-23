package statementqualification

import (
	"github.com/QahNaaRlN/grace/internal/statementdata"
	"sort"
	"strings"
)

func sortQualityPositions(p []QualityPosition) {
	sort.Slice(p, func(i, j int) bool { return p[i].SourceOrdinal < p[j].SourceOrdinal })
}
func compareQuality(ok bool) string {
	if ok {
		return "matches"
	}
	return "differs"
}
func clockParts(v *statementdata.LocalDate, name, field string, row int) *QualityParts {
	if v == nil {
		return &QualityParts{"missing", "missing", "missing", "missing"}
	}
	if row < 1 || row > 2 || !wordIn(name, "columns cards") {
		return &QualityParts{"unavailable", "unavailable", "unavailable", "unavailable"}
	}
	// Independent pinned public golden; no data is copied into the saved proposal.
	dates := []string{"2026-10-19", "2026-10-21"}
	clocks := []string{"10:30:00", "12:15:00"}
	if field == "booking_clock" {
		dates = []string{"2026-10-20", "2026-10-22"}
		clocks = []string{"11:00:00", "13:20:00"}
	}
	pieces := strings.SplitN(v.Local, "T", 2)
	actualClock := ""
	if len(pieces) == 2 {
		actualClock = pieces[1]
	}
	expectedClock, precision := "", "date"
	if name == "cards" {
		expectedClock = clocks[row-1]
		precision = "local_datetime"
	}
	return &QualityParts{compareQuality(pieces[0] == dates[row-1]), compareQuality(actualClock == expectedClock), compareQuality(v.Precision == precision), compareQuality(v.UTC == "")}
}
func qualityCheck(c ScoringCheck, name string, p statementdata.Proposal, r statementdata.Review, reasons []QualityReason, l qualityPositions) QualityCheck {
	out := QualityCheck{Ordinal: c.Ordinal, RowOrdinal: c.RowOrdinal, Code: c.Code, Status: "VERIFIED"}
	var ids statementdata.Selection
	link := func(keep func(QualityReason) bool) {
		for i, v := range reasons {
			if keep(v) {
				out.Links = append(out.Links, i+1)
			}
		}
	}
	switch c.Code {
	case "operation_clock", "booking_clock":
		if c.RowOrdinal < 1 || c.RowOrdinal > len(r.Rows) || c.RowOrdinal > len(p.Rows) {
			out.Status = "UNVERIFIED"
			out.Reason = "guard_mismatch"
			return out
		}
		value := r.Rows[c.RowOrdinal-1].OperationDate
		ids = p.Rows[c.RowOrdinal-1].OperationDate
		if c.Code == "booking_clock" {
			value = r.Rows[c.RowOrdinal-1].BookingDate
			ids = p.Rows[c.RowOrdinal-1].BookingDate
		}
		out.Clock = clockParts(value, name, c.Code, c.RowOrdinal)
	case "printed_debit_turnover", "printed_credit_turnover":
		value, expected, field := r.Header.DebitTurnover, int64(248600), "debitTurnover"
		if c.Code == "printed_credit_turnover" {
			value, expected, field = r.Header.CreditTurnover, 310500, "creditTurnover"
		}
		ids = headerSelections(p.Header, field)
		if name != "turnover-roles-ru" {
			out.Status = "UNVERIFIED"
			out.Reason = "guard_mismatch"
			return out
		}
		if value == nil {
			out.Money = &QualityMoneyParts{"missing", "missing"}
		} else {
			out.Money = &QualityMoneyParts{compareQuality(value.AmountMinor == expected), compareQuality(value.Currency == "RUB")}
		}
		link(func(v QualityReason) bool { return v.Field == field })
	case "row_issues":
		if c.RowOrdinal < 1 || c.RowOrdinal > len(p.Rows) {
			out.Status = "UNVERIFIED"
			out.Reason = "guard_mismatch"
			return out
		}
		ids = rowSelections(p.Rows[c.RowOrdinal-1])
		link(func(v QualityReason) bool { return v.Scope == "row" && v.RowOrdinal == c.RowOrdinal })
	case "coverage":
		out.State = r.Reconciliation.RowCoverage.Status
		link(func(v QualityReason) bool { return v.Scope == "coverage" })
	case "blockers":
		link(func(v QualityReason) bool { return v.Scope == "blocker" })
	case "turnover":
		out.State = r.Reconciliation.Turnover.Status
		ids = append(headerSelections(p.Header, "debitTurnover"), headerSelections(p.Header, "creditTurnover")...)
		link(func(v QualityReason) bool {
			return v.Scope == "turnover" || v.Scope == "row" || v.Scope == "blocker" && (v.Code == "invalid_source_rows" || v.Code == "aggregate_overflow" || v.Code == "printed_turnover_mismatch" || v.Code == "turnover_role_unverified")
		})
	default:
		out.Status = "UNVERIFIED"
		out.Reason = "guard_mismatch"
		return out
	}
	var ok bool
	out.Positions, ok = l.selected(ids)
	if !ok {
		return QualityCheck{Ordinal: c.Ordinal, RowOrdinal: c.RowOrdinal, Code: c.Code, Status: "UNVERIFIED", Reason: "diagnosis_limit_exceeded"}
	}
	return out
}
