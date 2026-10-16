// Package creditcards keeps credit card terms and computes, for a date, the debt,
// the amount and date that keep the grace period, and the minimum payment
// (SPEC-0051). The calculation is deterministic and works on whole days.
package creditcards

import "time"

const (
	GraceStatement         = "statement"
	GraceFromFirstPurchase = "from_first_purchase"
	GraceNone              = "none"
	GraceUnknown           = "unknown"
	dayLayout              = "2006-01-02"
	riskDays               = 3
	UnknownTerms           = "terms"
	UnknownLimit           = "limit"
	UnknownGraceModel      = "grace_model"
	UnknownMinimumRule     = "minimum_rule"
	UnknownInterest        = "interest"
	UnknownHistory         = "history"
	UnknownStatementDay    = "statement_day"
	UnknownDebt            = "debt"
)

// Terms is one version of a card's conditions. Nil means the owner does not know it.
type Terms struct {
	EffectiveFrom       time.Time
	LimitMinor          *int64
	GraceModel          string
	StatementDay        *int
	PaymentDays         *int
	GraceDays           *int
	MinPercentBP        *int
	MinAmountMinor      *int64
	MinRoundingMinor    *int64
	MinPercentIsCap     bool
	MinIncludesInterest bool
}

// Movement changes the debt on a day: positive for spending, negative for a
// repayment or refund.
type Movement struct {
	On          time.Time
	AmountMinor int64
}

// Ledger is the card account from the first tracked day. OpeningDebtMinor is the
// debt before Start (negative when the card was overpaid).
type Ledger struct {
	Start            time.Time
	OpeningDebtMinor int64
	Movements        []Movement
}

type Status struct {
	DebtMinor           int64
	AvailableMinor      *int64
	StatementClosedOn   *time.Time
	NextStatementOn     *time.Time
	GraceAmountMinor    *int64
	GraceDueOn          *time.Time
	GraceEstimate       bool
	GraceLost           bool
	GraceAtRisk         bool
	MinimumMinor        *int64
	MinimumDueOn        *time.Time
	MinimumIsUpperBound bool
	OverdueMinimumMinor int64
	Unknown             []string
}

// Calculate returns the card status at the end of day `on`.
func Calculate(terms *Terms, ledger Ledger, on time.Time) Status {
	on = day(on)
	status := Status{DebtMinor: debtAt(ledger, on), Unknown: []string{}}
	if terms == nil {
		status.Unknown = append(status.Unknown, UnknownTerms)
		return status
	}
	if terms.LimitMinor != nil {
		available := *terms.LimitMinor - status.DebtMinor
		status.AvailableMinor = &available
	} else {
		status.Unknown = append(status.Unknown, UnknownLimit)
	}
	switch terms.GraceModel {
	case GraceStatement:
		statementGrace(terms, ledger, on, &status)
	case GraceFromFirstPurchase:
		firstPurchaseGrace(terms, ledger, on, &status)
		if terms.StatementDay != nil && terms.PaymentDays != nil {
			statementMinimum(terms, ledger, on, &status)
		} else {
			status.Unknown = append(status.Unknown, UnknownStatementDay)
		}
	case GraceNone:
		if terms.StatementDay != nil && terms.PaymentDays != nil {
			statementMinimum(terms, ledger, on, &status)
		} else {
			status.Unknown = append(status.Unknown, UnknownStatementDay)
		}
	default:
		status.Unknown = append(status.Unknown, UnknownGraceModel)
	}
	if terms.MinIncludesInterest {
		status.Unknown = append(status.Unknown, UnknownInterest)
	}
	status.Unknown = unique(status.Unknown)
	return status
}

func unique(values []string) []string {
	out := make([]string, 0, len(values))
	seen := map[string]bool{}
	for _, v := range values {
		if !seen[v] {
			seen[v] = true
			out = append(out, v)
		}
	}
	return out
}

func statementGrace(terms *Terms, ledger Ledger, on time.Time, status *Status) {
	if terms.StatementDay == nil || terms.PaymentDays == nil {
		status.Unknown = append(status.Unknown, UnknownStatementDay)
		return
	}
	closed, next := statementAround(on, *terms.StatementDay)
	status.StatementClosedOn, status.NextStatementOn = &closed, &next
	due := closed.AddDate(0, 0, *terms.PaymentDays)
	statementMinimum(terms, ledger, on, status)
	if closed.Before(ledger.Start.AddDate(0, 0, -1)) {
		status.Unknown = append(status.Unknown, UnknownHistory)
		return
	}
	remaining := max(debtAt(ledger, closed)-paidBetween(ledger, closed, on), 0)
	if on.After(due) {
		// The window for the closed statement is over; what matters now is the next one.
		if remaining > 0 {
			status.GraceLost = true
		}
		nextDue := next.AddDate(0, 0, *terms.PaymentDays)
		estimate := max(status.DebtMinor, 0)
		status.GraceAmountMinor, status.GraceDueOn, status.GraceEstimate = &estimate, &nextDue, true
		return
	}
	status.GraceAmountMinor, status.GraceDueOn = &remaining, &due
	if status.OverdueMinimumMinor > 0 {
		status.GraceLost = true
	}
	status.GraceAtRisk = status.GraceLost || (remaining > 0 && daysBetween(on, due) < riskDays)
}

func firstPurchaseGrace(terms *Terms, ledger Ledger, on time.Time, status *Status) {
	if terms.GraceDays == nil {
		status.Unknown = append(status.Unknown, UnknownGraceModel)
		return
	}
	if status.DebtMinor <= 0 {
		zero := int64(0)
		status.GraceAmountMinor = &zero
		return
	}
	// The period starts with the first purchase after the debt was fully repaid.
	debt := ledger.OpeningDebtMinor
	var started *time.Time
	for _, m := range sorted(ledger.Movements, on) {
		before := debt
		debt += m.AmountMinor
		if debt <= 0 {
			started = nil
		} else if before <= 0 && m.AmountMinor > 0 {
			start := m.On
			started = &start
		}
	}
	if started == nil {
		// The purchase that opened the period happened before tracking began.
		status.Unknown = append(status.Unknown, UnknownHistory)
		return
	}
	due := started.AddDate(0, 0, *terms.GraceDays)
	amount := status.DebtMinor
	status.GraceAmountMinor, status.GraceDueOn = &amount, &due
	status.GraceLost = on.After(due)
	status.GraceAtRisk = status.GraceLost || daysBetween(on, due) < riskDays
}

// statementMinimum estimates the minimum payment of the last closed statement
// and adds what is still unpaid from the statement before it.
func statementMinimum(terms *Terms, ledger Ledger, on time.Time, status *Status) {
	if terms.MinPercentBP == nil && terms.MinAmountMinor == nil {
		status.Unknown = append(status.Unknown, UnknownMinimumRule)
		return
	}
	closed, _ := statementAround(on, *terms.StatementDay)
	if closed.Before(ledger.Start.AddDate(0, 0, -1)) {
		status.Unknown = append(status.Unknown, UnknownHistory)
		return
	}
	due := closed.AddDate(0, 0, *terms.PaymentDays)
	previous, _ := statementAround(closed.AddDate(0, 0, -1), *terms.StatementDay)
	if !previous.Before(ledger.Start.AddDate(0, 0, -1)) {
		previousDue := previous.AddDate(0, 0, *terms.PaymentDays)
		owed := minimumFor(terms, debtAt(ledger, previous))
		status.OverdueMinimumMinor = max(owed-paidBetween(ledger, previous, minDay(previousDue, on)), 0)
	}
	minimum := minimumFor(terms, debtAt(ledger, closed)) + status.OverdueMinimumMinor
	remaining := max(minimum-paidBetween(ledger, closed, on), 0)
	status.MinimumMinor, status.MinimumDueOn = &remaining, &due
	status.MinimumIsUpperBound = terms.MinPercentIsCap
}

func minimumFor(terms *Terms, debt int64) int64 {
	if debt <= 0 {
		return 0
	}
	var value int64
	if terms.MinPercentBP != nil {
		value = ceilDiv(debt*int64(*terms.MinPercentBP), 10000)
	}
	if terms.MinAmountMinor != nil {
		value = max(value, *terms.MinAmountMinor)
	}
	if terms.MinRoundingMinor != nil && *terms.MinRoundingMinor > 1 {
		value = ceilDiv(value, *terms.MinRoundingMinor) * *terms.MinRoundingMinor
	}
	return min(value, debt)
}

// statementAround returns the last statement date on or before `on` and the next one.
func statementAround(on time.Time, statementDay int) (time.Time, time.Time) {
	this := statementIn(on.Year(), on.Month(), statementDay)
	if this.After(on) {
		previousMonth := time.Date(on.Year(), on.Month()-1, 1, 0, 0, 0, 0, time.UTC)
		return statementIn(previousMonth.Year(), previousMonth.Month(), statementDay), this
	}
	nextMonth := time.Date(on.Year(), on.Month()+1, 1, 0, 0, 0, 0, time.UTC)
	return this, statementIn(nextMonth.Year(), nextMonth.Month(), statementDay)
}

// A statement day past the end of a short month falls on its last day.
func statementIn(year int, month time.Month, statementDay int) time.Time {
	last := time.Date(year, month+1, 0, 0, 0, 0, 0, time.UTC).Day()
	return time.Date(year, month, min(statementDay, last), 0, 0, 0, 0, time.UTC)
}

func debtAt(ledger Ledger, on time.Time) int64 {
	debt := ledger.OpeningDebtMinor
	for _, m := range ledger.Movements {
		if !day(m.On).After(on) {
			debt += m.AmountMinor
		}
	}
	return debt
}

// paidBetween sums repayments after `from` up to and including `to`.
func paidBetween(ledger Ledger, from, to time.Time) int64 {
	var paid int64
	for _, m := range ledger.Movements {
		d := day(m.On)
		if m.AmountMinor < 0 && d.After(from) && !d.After(to) {
			paid -= m.AmountMinor
		}
	}
	return paid
}

func sorted(movements []Movement, until time.Time) []Movement {
	out := make([]Movement, 0, len(movements))
	for _, m := range movements {
		if !day(m.On).After(until) {
			out = append(out, Movement{day(m.On), m.AmountMinor})
		}
	}
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && out[j].On.Before(out[j-1].On); j-- {
			out[j], out[j-1] = out[j-1], out[j]
		}
	}
	return out
}

func day(t time.Time) time.Time { return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC) }
func daysBetween(from, to time.Time) int {
	return int(day(to).Sub(day(from)).Hours() / 24)
}
func minDay(a, b time.Time) time.Time {
	if a.Before(b) {
		return a
	}
	return b
}
func ceilDiv(a, b int64) int64 { return (a + b - 1) / b }

// ParseDay reads YYYY-MM-DD as a UTC calendar day.
func ParseDay(value string) (time.Time, error) { return time.Parse(dayLayout, value) }

// FormatDay writes a calendar day as YYYY-MM-DD.
func FormatDay(t time.Time) string { return t.Format(dayLayout) }
