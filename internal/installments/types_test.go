package installments

import "testing"

func TestRowsAndSafeMoney(t *testing.T) {
	if !validRows([]Row{{"2026-10-01", 300}, {"2026-10-01", 300}}, 600, "2026-10-01") {
		t.Fatal("same-day rows are valid")
	}
	for _, rows := range [][]Row{nil, {{"2026-02-30", 600}}, {{"2026-09-30", 600}}, {{"2026-10-01", 601}}, {{"2026-10-01", -1}, {"2026-10-02", 601}}} {
		if validRows(rows, 600, "2026-10-01") {
			t.Fatalf("accepted invalid rows: %+v", rows)
		}
	}
	if validRows([]Row{{"2026-10-01", MaxMoney}, {"2026-10-02", 1}}, MaxMoney, "2026-10-01") {
		t.Fatal("overflow")
	}
}
func TestCanonicalBankRequirement(t *testing.T) {
	grace, minimum, due, other := int64(500000), int64(200000), "2026-10-30", "2026-10-25"
	r := &Requirement{Source: "bank_confirmed", GraceAmountMinor: &grace, GraceDueOn: &due, MinimumMinor: &minimum, MinimumDueOn: &due, MinimumInGrace: true}
	if n, _, ok := CanonicalRequirement(r); !ok || n != 500000 {
		t.Fatal("nested minimum must not be added")
	}
	r.MinimumDueOn = &other
	if _, _, ok := CanonicalRequirement(r); ok {
		t.Fatal("different dates require completeness review")
	}
	r.MinimumDueOn = &due
	r.MinimumInGrace = false
	if _, _, ok := CanonicalRequirement(r); ok {
		t.Fatal("unknown inclusion")
	}
	r.Source = "unknown"
	if _, _, ok := CanonicalRequirement(r); ok {
		t.Fatal("unknown is not zero")
	}
}
