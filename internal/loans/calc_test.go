package loans

import (
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

func TestInterestByActualDays(t *testing.T) {
	cases := []struct {
		name     string
		balance  int64
		rate     int
		from, to string
		want     int64
	}{
		// 12 000 000 × 12% × 31 / 365 = 122 301.37
		{"month of 31 days", 12_000_000, 1200, "2026-01-15", "2026-02-15", 122_301},
		// 11 056 116 × 12% × 28 / 365 = 101 776.85
		{"february", 11_056_116, 1200, "2026-02-15", "2026-03-15", 101_777},
		// 1 000 000 × 10% × (12 / 365 + 19 / 366) = 3 287.67 + 5 191.26
		{"year end into a leap year", 1_000_000, 1000, "2027-12-20", "2028-01-20", 8_479},
		{"zero rate", 1_000_000, 0, "2026-01-01", "2026-02-01", 0},
		{"no days", 1_000_000, 1200, "2026-01-01", "2026-01-01", 0},
	}
	for _, c := range cases {
		if got := Interest(c.balance, c.rate, d(c.from), d(c.to)); got != c.want {
			t.Errorf("%s: interest %d, want %d", c.name, got, c.want)
		}
	}
}

func annuityRows() []CalcRow {
	rows := make([]CalcRow, 12)
	for i := range rows {
		rows[i] = CalcRow{DueOn: d("2026-01-15").AddDate(0, i+1, 0), AmountMinor: 1_066_185}
	}
	return rows
}

func TestCalculateAnnuity(t *testing.T) {
	in := CalcInput{Type: Annuity, PrincipalMinor: 12_000_000, RateBP: 1200, AgreementOn: d("2026-01-15"), Rows: annuityRows()}
	got := CalculateParts(in)
	first, second := got.Rows[0].Parts, got.Rows[1].Parts
	if first == nil || *first != (Parts{PrincipalMinor: 943_884, InterestMinor: 122_301}) {
		t.Fatalf("first row %+v", got.Rows[0])
	}
	if second == nil || *second != (Parts{PrincipalMinor: 1_066_185 - 101_777, InterestMinor: 101_777}) {
		t.Fatalf("second row %+v", got.Rows[1])
	}
	var principal int64
	for i, row := range got.Rows {
		if row.Parts == nil || row.Warning != "" || row.Parts.Total() != in.Rows[i].AmountMinor {
			t.Fatalf("row %d: %+v", i, row)
		}
		principal += row.Parts.PrincipalMinor
	}
	// The last row takes the rest of the principal, so the schedule repays it exactly.
	if principal != 12_000_000 || got.PrincipalTotalMinor != 12_000_000 || got.PrincipalDifferenceMinor != 0 {
		t.Fatalf("principal %d, result %+v", principal, got)
	}
}

func TestCalculateAnnuityRowBelowInterest(t *testing.T) {
	rows := annuityRows()
	rows[0].AmountMinor = 100_000 // smaller than the month's interest
	got := CalculateParts(CalcInput{Type: Annuity, PrincipalMinor: 12_000_000, RateBP: 1200, AgreementOn: d("2026-01-15"), Rows: rows})
	if got.Rows[0].Parts != nil || got.Rows[0].Warning != WarningBelowInterest {
		t.Fatalf("first row %+v", got.Rows[0])
	}
	if got.Rows[1].Parts == nil {
		t.Fatal("later rows are still calculated")
	}
}

func TestCalculateDifferentiated(t *testing.T) {
	rows := []CalcRow{
		{d("2026-04-01"), 412_230}, // 1 200 000 × 12% × 31 / 365 = 12 230.14
		{d("2026-05-01"), 410_000}, // formula: 800 000 × 12% × 30 / 365 = 7 890.41
		{d("2026-06-01"), 404_077}, // 400 000 × 12% × 31 / 365 = 4 076.71
	}
	got := CalculateParts(CalcInput{Type: Differentiated, PrincipalMinor: 1_200_000, RateBP: 1200, AgreementOn: d("2026-03-01"), Rows: rows})
	want := []Parts{{400_000, 12_230, 0}, {400_000, 10_000, 0}, {400_000, 4_077, 0}}
	for i := range rows {
		if got.Rows[i].Parts == nil || *got.Rows[i].Parts != want[i] {
			t.Fatalf("row %d: %+v", i, got.Rows[i])
		}
	}
	if got.Rows[0].Warning != "" || got.Rows[1].Warning != WarningNotDifferentiated || got.Rows[2].Warning != "" {
		t.Fatalf("warnings %+v", got.Rows)
	}
}

func TestCalculateDifferentiatedRemainderInLastRow(t *testing.T) {
	rows := []CalcRow{{d("2026-02-01"), 400_000}, {d("2026-03-01"), 400_000}, {d("2026-04-01"), 400_001}}
	got := CalculateParts(CalcInput{Type: Differentiated, PrincipalMinor: 1_000_001, RateBP: 0, AgreementOn: d("2026-01-01"), Rows: rows})
	if got.Rows[2].Parts == nil || got.Rows[2].Parts.PrincipalMinor != 333_335 || got.PrincipalDifferenceMinor != 0 {
		t.Fatalf("result %+v", got)
	}
}

func TestAllocateRow(t *testing.T) {
	row := &Parts{PrincipalMinor: 900, InterestMinor: 100, OtherMinor: 50}
	cases := []struct {
		name     string
		row      *Parts
		payments []RowPayment
		want     []*Parts
	}{
		{"full payment gives the row parts", row, []RowPayment{{AmountMinor: 1050}}, []*Parts{{900, 100, 50}}},
		{"partial payments: interest, then principal, then other", row,
			[]RowPayment{{AmountMinor: 60}, {AmountMinor: 600}, {AmountMinor: 390}},
			[]*Parts{{0, 60, 0}, {560, 40, 0}, {340, 0, 50}}},
		{"owner split reduces what is left", row,
			[]RowPayment{{AmountMinor: 500}, {AmountMinor: 550, Override: &Parts{450, 50, 50}}},
			[]*Parts{{450, 50, 0}, {450, 50, 50}}},
		{"row without parts", nil, []RowPayment{{AmountMinor: 500}, {AmountMinor: 100, Override: &Parts{100, 0, 0}}},
			[]*Parts{nil, {100, 0, 0}}},
		{"left part does not cover the payment", row,
			[]RowPayment{{AmountMinor: 1000, Override: &Parts{1000, 0, 0}}, {AmountMinor: 50}},
			[]*Parts{{1000, 0, 0}, {0, 50, 0}}},
	}
	for _, c := range cases {
		got := AllocateRow(c.row, c.payments)
		for i := range c.want {
			if (got[i] == nil) != (c.want[i] == nil) || (got[i] != nil && *got[i] != *c.want[i]) {
				t.Errorf("%s: payment %d = %+v, want %+v", c.name, i, got[i], c.want[i])
			}
		}
	}
}

func TestOutstandingAndDiscrepancy(t *testing.T) {
	issued := ptr(int64(1_000_000))
	movements := []Movement{
		{d("2026-02-15"), ptr(int64(100_000))},
		{d("2026-03-15"), ptr(int64(100_000))},
		{d("2026-04-10"), ptr(int64(300_000))}, // prepayment
		{d("2026-04-15"), ptr(int64(70_000))},
	}
	got := OutstandingOn(issued, nil, movements, d("2026-04-12"))
	if got.Basis != BasisIssued || got.PrincipalMinor == nil || *got.PrincipalMinor != 500_000 {
		t.Fatalf("from issued: %+v", got)
	}
	// The bank snapshot of 15.03 already reflects the payment of that day.
	snapshots := []Snapshot{{d("2026-03-15"), 805_000}}
	got = OutstandingOn(issued, snapshots, movements, d("2026-04-30"))
	if got.Basis != BasisSnapshot || *got.PrincipalMinor != 805_000-300_000-70_000 {
		t.Fatalf("from snapshot: %+v", got)
	}
	latest, grace, diff := Discrepancy(issued, snapshots, movements, d("2026-04-30"))
	if latest == nil || *grace != 800_000 || *diff != -5_000 {
		t.Fatalf("discrepancy %v %v %v", latest, grace, diff)
	}
	// A later snapshot is compared from the previous one.
	snapshots = append(snapshots, Snapshot{d("2026-04-15"), 435_000})
	if _, grace, diff = Discrepancy(issued, snapshots, movements, d("2026-04-30")); *grace != 435_000 || *diff != 0 {
		t.Fatalf("second discrepancy %v %v", *grace, *diff)
	}
	// A payment with an unknown split makes Grace's value unknown.
	movements = append(movements, Movement{d("2026-05-15"), nil})
	got = OutstandingOn(issued, snapshots, movements, d("2026-05-31"))
	if got.PrincipalMinor != nil || got.Unallocated != 1 {
		t.Fatalf("unknown split: %+v", got)
	}
	if got = OutstandingOn(nil, nil, nil, d("2026-05-31")); got.Basis != BasisUnknown || got.PrincipalMinor != nil {
		t.Fatalf("no basis: %+v", got)
	}
}

func TestValidParts(t *testing.T) {
	if !ValidParts(nil, nil, nil, 100) || !ValidParts(ptr(int64(0)), ptr(int64(100)), ptr(int64(0)), 100) {
		t.Fatal("valid parts rejected")
	}
	if ValidParts(ptr(int64(50)), nil, nil, 100) || ValidParts(ptr(int64(50)), ptr(int64(40)), ptr(int64(0)), 100) ||
		ValidParts(ptr(int64(-1)), ptr(int64(101)), ptr(int64(0)), 100) {
		t.Fatal("invalid parts accepted")
	}
}
