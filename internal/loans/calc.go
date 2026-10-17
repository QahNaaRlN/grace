// Package loans splits loan payments into principal, interest and other parts,
// computes the outstanding principal for a date and compares it with bank
// snapshots (SPEC-0053). The calculation is deterministic and works on whole days.
package loans

import (
	"math/big"
	"time"
)

const (
	Annuity        = "annuity"
	Differentiated = "differentiated"
	TypeUnknown    = "unknown"

	SourceBank       = "bank"
	SourceCalculated = "calculated"

	dayLayout = "2006-01-02"
	// A differentiated row whose interest differs from the formula by more than
	// this does not look like a differentiated payment.
	toleranceMinor = 100
)

// Parts is one split of money between principal, interest and other charges
// (fees, insurance, penalties).
type Parts struct {
	PrincipalMinor int64 `json:"principalMinor"`
	InterestMinor  int64 `json:"interestMinor"`
	OtherMinor     int64 `json:"otherMinor"`
}

func (p Parts) Total() int64 { return p.PrincipalMinor + p.InterestMinor + p.OtherMinor }

func (p Parts) add(o Parts) Parts {
	return Parts{p.PrincipalMinor + o.PrincipalMinor, p.InterestMinor + o.InterestMinor, p.OtherMinor + o.OtherMinor}
}

func (p Parts) valid() bool {
	return p.PrincipalMinor >= 0 && p.InterestMinor >= 0 && p.OtherMinor >= 0
}

// ValidParts reports whether optional parts are either absent or non-negative and
// add up to the amount.
func ValidParts(principal, interest, other *int64, amount int64) bool {
	if principal == nil && interest == nil && other == nil {
		return true
	}
	if principal == nil || interest == nil || other == nil {
		return false
	}
	p := Parts{*principal, *interest, *other}
	return p.valid() && p.Total() == amount
}

// RowPayment is money that paid one schedule row, in the order it was paid.
// Override is the split the owner entered; nil means "derive it from the row".
type RowPayment struct {
	AmountMinor int64
	Override    *Parts
}

// AllocateRow splits the payments of one schedule row. Owner splits are kept and
// reduce what is left of the row; every other payment takes interest first, then
// principal, then other charges. A nil result means the split is unknown: the row
// has no parts, or what is left of it no longer covers the payment.
func AllocateRow(row *Parts, payments []RowPayment) []*Parts {
	out := make([]*Parts, len(payments))
	var left Parts
	if row != nil {
		left = *row
	}
	for i, p := range payments {
		if p.Override != nil {
			o := *p.Override
			out[i] = &o
			left = Parts{max(left.PrincipalMinor-o.PrincipalMinor, 0), max(left.InterestMinor-o.InterestMinor, 0), max(left.OtherMinor-o.OtherMinor, 0)}
		}
	}
	if row == nil {
		return out
	}
	for i, p := range payments {
		if p.Override != nil {
			continue
		}
		if left.Total() < p.AmountMinor {
			continue
		}
		rest := p.AmountMinor
		take := func(available *int64) int64 {
			v := min(*available, rest)
			*available -= v
			rest -= v
			return v
		}
		var split Parts
		split.InterestMinor = take(&left.InterestMinor)
		split.PrincipalMinor = take(&left.PrincipalMinor)
		split.OtherMinor = take(&left.OtherMinor)
		out[i] = &split
	}
	return out
}

// CalcRow is one schedule row given to the part calculation.
type CalcRow struct {
	DueOn       time.Time
	AmountMinor int64
}

// CalcInput is what the calculation of parts needs from the loan agreement.
type CalcInput struct {
	Type           string
	PrincipalMinor int64
	RateBP         int
	AgreementOn    time.Time
	Rows           []CalcRow
}

const (
	WarningBelowInterest      = "payment_below_interest"
	WarningPrincipalExhausted = "principal_exhausted"
	WarningNotDifferentiated  = "not_differentiated"
)

// CalcRowResult is the calculated split of a row; nil Parts means the row could
// not be split and keeps its previous parts.
type CalcRowResult struct {
	Parts   *Parts
	Warning string
}

type CalcResult struct {
	Rows []CalcRowResult
	// PrincipalTotalMinor is the principal of the calculated rows; the difference
	// to the issued principal is shown to the owner.
	PrincipalTotalMinor      int64
	PrincipalDifferenceMinor int64
}

// CalculateParts fills the principal and interest of existing schedule rows by the
// repayment type. Interest of a period is the principal left × the annual rate ×
// the days of the period / the days of its year (365 or 366); a period crossing a
// year end is split by years, and the result is rounded half up to a minor unit.
func CalculateParts(in CalcInput) CalcResult {
	result := CalcResult{Rows: make([]CalcRowResult, len(in.Rows))}
	balance := in.PrincipalMinor
	from := in.AgreementOn
	n := int64(len(in.Rows))
	for i, row := range in.Rows {
		interest := Interest(balance, in.RateBP, from, row.DueOn)
		last := i == len(in.Rows)-1
		var principal int64
		var warning string
		switch in.Type {
		case Annuity:
			principal = row.AmountMinor - interest
			if last {
				principal = balance
				interest = row.AmountMinor - principal
			}
		case Differentiated:
			principal = in.PrincipalMinor / n
			if last {
				principal = balance
			}
			formula := interest
			interest = row.AmountMinor - principal
			if diff := interest - formula; diff > toleranceMinor || diff < -toleranceMinor {
				warning = WarningNotDifferentiated
			}
		}
		switch {
		case interest < 0 || principal < 0:
			result.Rows[i] = CalcRowResult{Warning: WarningBelowInterest}
		case principal > balance:
			result.Rows[i] = CalcRowResult{Warning: WarningPrincipalExhausted}
		default:
			result.Rows[i] = CalcRowResult{Parts: &Parts{PrincipalMinor: principal, InterestMinor: interest}, Warning: warning}
			balance -= principal
			result.PrincipalTotalMinor += principal
		}
		from = row.DueOn
	}
	result.PrincipalDifferenceMinor = result.PrincipalTotalMinor - in.PrincipalMinor
	return result
}

// Interest accrued on balance from `from` (exclusive) to `to` (inclusive).
func Interest(balance int64, rateBP int, from, to time.Time) int64 {
	if balance <= 0 || rateBP <= 0 || !to.After(from) {
		return 0
	}
	total := new(big.Rat)
	for start := from; start.Before(to); {
		yearEnd := time.Date(start.Year()+1, 1, 1, 0, 0, 0, 0, time.UTC)
		end := to
		if yearEnd.Before(to) {
			end = yearEnd
		}
		days := int64(end.Sub(start).Hours() / 24)
		yearDays := int64(365)
		if isLeap(start.Year()) {
			yearDays = 366
		}
		part := new(big.Rat).SetFrac(
			new(big.Int).Mul(new(big.Int).Mul(big.NewInt(balance), big.NewInt(int64(rateBP))), big.NewInt(days)),
			big.NewInt(10000*yearDays))
		total.Add(total, part)
		start = end
	}
	return roundHalfUp(total)
}

func roundHalfUp(r *big.Rat) int64 {
	num := new(big.Int).Mul(r.Num(), big.NewInt(2))
	num.Add(num, r.Denom())
	den := new(big.Int).Mul(r.Denom(), big.NewInt(2))
	return new(big.Int).Div(num, den).Int64()
}

func isLeap(year int) bool { return year%4 == 0 && (year%100 != 0 || year%400 == 0) }

// Snapshot is the outstanding principal the bank showed at the end of a day.
type Snapshot struct {
	On             time.Time
	PrincipalMinor int64
}

// Movement is principal repaid on a day; nil means the split of that payment is
// not known.
type Movement struct {
	On             time.Time
	PrincipalMinor *int64
}

const (
	BasisSnapshot = "snapshot"
	BasisIssued   = "issued"
	BasisUnknown  = "unknown"
)

// Outstanding is the principal left at the end of a day by Grace's data.
type Outstanding struct {
	PrincipalMinor *int64
	Basis          string
	BasisOn        *time.Time
	Unallocated    int
}

// OutstandingOn starts from the latest snapshot on or before `on` (else from the
// issued principal) and subtracts the principal of later payments up to `on`.
// A snapshot already reflects the payments of its own day.
func OutstandingOn(issued *int64, snapshots []Snapshot, movements []Movement, on time.Time) Outstanding {
	var base *Snapshot
	for i := range snapshots {
		if !snapshots[i].On.After(on) && (base == nil || snapshots[i].On.After(base.On)) {
			base = &snapshots[i]
		}
	}
	return outstandingFrom(issued, base, movements, on)
}

func outstandingFrom(issued *int64, base *Snapshot, movements []Movement, on time.Time) Outstanding {
	var result Outstanding
	var value int64
	switch {
	case base != nil:
		value, result.Basis = base.PrincipalMinor, BasisSnapshot
		basisOn := base.On
		result.BasisOn = &basisOn
	case issued != nil:
		value, result.Basis = *issued, BasisIssued
	default:
		return Outstanding{Basis: BasisUnknown}
	}
	for _, m := range movements {
		if m.On.After(on) || (base != nil && !m.On.After(base.On)) {
			continue
		}
		if m.PrincipalMinor == nil {
			result.Unallocated++
			continue
		}
		value -= *m.PrincipalMinor
	}
	if result.Unallocated == 0 {
		result.PrincipalMinor = &value
	}
	return result
}

// Discrepancy compares the latest snapshot on or before `on` with Grace's value for
// its day, computed from the previous snapshot or the issued principal. Nil when
// there is no snapshot or Grace's value is unknown.
func Discrepancy(issued *int64, snapshots []Snapshot, movements []Movement, on time.Time) (*Snapshot, *int64, *int64) {
	var latest, previous *Snapshot
	for i := range snapshots {
		s := &snapshots[i]
		if s.On.After(on) {
			continue
		}
		if latest == nil || s.On.After(latest.On) {
			previous, latest = latest, s
		} else if previous == nil || s.On.After(previous.On) {
			previous = s
		}
	}
	if latest == nil {
		return nil, nil, nil
	}
	grace := outstandingFrom(issued, previous, movements, latest.On)
	if grace.PrincipalMinor == nil {
		return latest, nil, nil
	}
	diff := *grace.PrincipalMinor - latest.PrincipalMinor
	return latest, grace.PrincipalMinor, &diff
}

// ParseDay reads a calendar day.
func ParseDay(value string) (time.Time, error) { return time.Parse(dayLayout, value) }

// FormatDay writes a calendar day.
func FormatDay(t time.Time) string { return t.Format(dayLayout) }

// Today is the current calendar day in the application time zone (UTC+5).
func Today() time.Time {
	now := time.Now().UTC().Add(5 * time.Hour)
	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
}
