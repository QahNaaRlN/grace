package creditcards

import (
	"slices"
	"testing"
	"time"
)

func d(value string) time.Time {
	t, err := ParseDay(value)
	if err != nil {
		panic(err)
	}
	return t
}
func ptr[T any](v T) *T { return &v }

// A synthetic card in the shape of the common "up to 55 days" tariff: statement on
// the 5th, 25 days to pay; minimum is at most 14% of the debt, at least 600 ₽,
// rounded up to 100 ₽ and never above the debt.
func statementCard() *Terms {
	return &Terms{
		EffectiveFrom:    d("2026-01-01"),
		LimitMinor:       ptr(int64(10_000_000)),
		GraceModel:       GraceStatement,
		StatementDay:     ptr(5),
		PaymentDays:      ptr(25),
		MinPercentBP:     ptr(1400),
		MinAmountMinor:   ptr(int64(60_000)),
		MinRoundingMinor: ptr(int64(10_000)),
		MinPercentIsCap:  true,
	}
}

func ledger(movements ...Movement) Ledger {
	return Ledger{Start: d("2026-09-01"), Movements: movements}
}
func m(on string, amount int64) Movement { return Movement{d(on), amount} }

func TestStatementGraceAmountAndDate(t *testing.T) {
	l := ledger(
		m("2026-09-03", 1_000_000), // 10 000 ₽ before the statement
		m("2026-09-05", 500_000),   // on the statement day: still in it
		m("2026-09-06", 200_000),   // next period
	)
	s := Calculate(statementCard(), l, d("2026-09-10"))
	if s.DebtMinor != 1_700_000 || *s.AvailableMinor != 8_300_000 {
		t.Fatalf("debt %d available %d", s.DebtMinor, *s.AvailableMinor)
	}
	if *s.GraceAmountMinor != 1_500_000 || FormatDay(*s.GraceDueOn) != "2026-09-30" || s.GraceEstimate {
		t.Fatalf("grace %d due %s estimate %v", *s.GraceAmountMinor, FormatDay(*s.GraceDueOn), s.GraceEstimate)
	}
	if FormatDay(*s.StatementClosedOn) != "2026-09-05" || FormatDay(*s.NextStatementOn) != "2026-10-05" {
		t.Fatalf("statements %s %s", *s.StatementClosedOn, *s.NextStatementOn)
	}
	if s.GraceAtRisk || s.GraceLost {
		t.Fatalf("not at risk yet: %+v", s)
	}
}

func TestPartialRepaymentAndRisk(t *testing.T) {
	l := ledger(m("2026-09-03", 1_500_000), m("2026-09-20", -600_000))
	s := Calculate(statementCard(), l, d("2026-09-25"))
	if *s.GraceAmountMinor != 900_000 || s.GraceAtRisk {
		t.Fatalf("five days before the date: %d risk %v", *s.GraceAmountMinor, s.GraceAtRisk)
	}
	s = Calculate(statementCard(), l, d("2026-09-28"))
	if !s.GraceAtRisk {
		t.Fatal("two days before the date with money still owed is at risk")
	}
	paid := ledger(m("2026-09-03", 1_500_000), m("2026-09-20", -1_500_000))
	if s = Calculate(statementCard(), paid, d("2026-09-29")); *s.GraceAmountMinor != 0 || s.GraceAtRisk {
		t.Fatalf("fully paid: %d risk %v", *s.GraceAmountMinor, s.GraceAtRisk)
	}
}

func TestMissedGraceWindowMovesToNextStatement(t *testing.T) {
	l := ledger(m("2026-09-03", 1_500_000), m("2026-09-20", -600_000), m("2026-10-02", 100_000))
	s := Calculate(statementCard(), l, d("2026-10-03"))
	if !s.GraceLost || !s.GraceEstimate {
		t.Fatalf("window of the 5 Sep statement is over: %+v", s)
	}
	if *s.GraceAmountMinor != 1_000_000 || FormatDay(*s.GraceDueOn) != "2026-10-30" {
		t.Fatalf("estimate for the next statement: %d %s", *s.GraceAmountMinor, FormatDay(*s.GraceDueOn))
	}
}

func TestMinimumPaymentRule(t *testing.T) {
	cases := []struct {
		debt, want int64
	}{
		{10_000_000, 1_400_000}, // 14% of 100 000 ₽
		{12_345_600, 1_730_000}, // 14% = 17 283,84 ₽ → up to 17 300 ₽
		{300_000, 60_000},       // 14% < 600 ₽ → 600 ₽
		{50_000, 50_000},        // never above the debt
		{0, 0},
		{-10_000, 0},
	}
	for _, c := range cases {
		if got := minimumFor(statementCard(), c.debt); got != c.want {
			t.Errorf("debt %d: minimum %d, want %d", c.debt, got, c.want)
		}
	}
	s := Calculate(statementCard(), ledger(m("2026-09-03", 10_000_000), m("2026-09-15", -400_000)), d("2026-09-20"))
	if *s.MinimumMinor != 1_000_000 || FormatDay(*s.MinimumDueOn) != "2026-09-30" || !s.MinimumIsUpperBound {
		t.Fatalf("remaining minimum %d due %s cap %v", *s.MinimumMinor, FormatDay(*s.MinimumDueOn), s.MinimumIsUpperBound)
	}
	unknown := statementCard()
	unknown.MinPercentBP, unknown.MinAmountMinor = nil, nil
	s = Calculate(unknown, ledger(m("2026-09-03", 100_000)), d("2026-09-10"))
	if s.MinimumMinor != nil || !slices.Contains(s.Unknown, UnknownMinimumRule) {
		t.Fatalf("unknown rule must stay unknown: %+v", s)
	}
}

func TestUnpaidPreviousMinimumLosesGrace(t *testing.T) {
	l := Ledger{Start: d("2026-08-01"), Movements: []Movement{m("2026-08-03", 1_000_000), m("2026-09-10", 50_000)}}
	s := Calculate(statementCard(), l, d("2026-09-12"))
	// The 5 Aug statement owed 1 400 ₽ by 30 Aug and nothing was paid.
	if s.OverdueMinimumMinor != 140_000 || !s.GraceLost || !s.GraceAtRisk {
		t.Fatalf("overdue %d lost %v risk %v", s.OverdueMinimumMinor, s.GraceLost, s.GraceAtRisk)
	}
	if *s.MinimumMinor != 280_000 {
		t.Fatalf("minimum includes the overdue one: %d", *s.MinimumMinor)
	}
}

func TestShortMonthsAndLeapYear(t *testing.T) {
	cases := []struct{ on, closed, next string }{
		{"2026-02-15", "2026-01-31", "2026-02-28"},
		{"2026-03-01", "2026-02-28", "2026-03-31"},
		{"2028-03-10", "2028-02-29", "2028-03-31"},
		{"2026-12-31", "2026-12-31", "2027-01-31"},
	}
	for _, c := range cases {
		closed, next := statementAround(d(c.on), 31)
		if FormatDay(closed) != c.closed || FormatDay(next) != c.next {
			t.Errorf("%s: %s %s, want %s %s", c.on, FormatDay(closed), FormatDay(next), c.closed, c.next)
		}
	}
}

func TestStatementBeforeTrackingIsUnknown(t *testing.T) {
	l := Ledger{Start: d("2026-09-20"), OpeningDebtMinor: 300_000}
	s := Calculate(statementCard(), l, d("2026-09-25"))
	if s.GraceAmountMinor != nil || s.MinimumMinor != nil || !slices.Contains(s.Unknown, UnknownHistory) {
		t.Fatalf("statement of 5 Sep predates tracking: %+v", s)
	}
	if s.DebtMinor != 300_000 {
		t.Fatalf("debt still known: %d", s.DebtMinor)
	}
}

func TestFromFirstPurchase(t *testing.T) {
	terms := &Terms{GraceModel: GraceFromFirstPurchase, GraceDays: ptr(120)}
	l := ledger(m("2026-09-10", 100_000), m("2026-10-01", -100_000), m("2026-10-05", 50_000), m("2026-10-07", 20_000))
	s := Calculate(terms, l, d("2026-10-10"))
	if *s.GraceAmountMinor != 70_000 || FormatDay(*s.GraceDueOn) != "2027-02-02" || s.GraceLost {
		t.Fatalf("grace %d due %s", *s.GraceAmountMinor, FormatDay(*s.GraceDueOn))
	}
	if !slices.Contains(s.Unknown, UnknownStatementDay) || !slices.Contains(s.Unknown, UnknownLimit) {
		t.Fatalf("minimum and limit unknown without terms: %v", s.Unknown)
	}
	if s = Calculate(terms, l, d("2027-02-03")); !s.GraceLost {
		t.Fatal("past the 120th day the grace period is lost")
	}
	opened := Ledger{Start: d("2026-09-01"), OpeningDebtMinor: 10_000}
	if s = Calculate(terms, opened, d("2026-09-05")); !slices.Contains(s.Unknown, UnknownHistory) {
		t.Fatalf("period opened before tracking: %v", s.Unknown)
	}
}

func TestNoTermsAndUnknownModel(t *testing.T) {
	l := ledger(m("2026-09-03", 100_000))
	s := Calculate(nil, l, d("2026-09-10"))
	if s.DebtMinor != 100_000 || s.AvailableMinor != nil || !slices.Equal(s.Unknown, []string{UnknownTerms}) {
		t.Fatalf("without terms only the debt is known: %+v", s)
	}
	s = Calculate(&Terms{GraceModel: GraceUnknown, LimitMinor: ptr(int64(500_000))}, l, d("2026-09-10"))
	if *s.AvailableMinor != 400_000 || s.GraceAmountMinor != nil || !slices.Contains(s.Unknown, UnknownGraceModel) {
		t.Fatalf("unknown model: %+v", s)
	}
}

func TestOverpaidCard(t *testing.T) {
	s := Calculate(statementCard(), ledger(m("2026-09-03", 100_000), m("2026-09-04", -150_000)), d("2026-09-10"))
	if s.DebtMinor != -50_000 || *s.AvailableMinor != 10_050_000 || *s.GraceAmountMinor != 0 || *s.MinimumMinor != 0 {
		t.Fatalf("overpaid card: %+v", s)
	}
}
