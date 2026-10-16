package commitments

import "testing"

func TestSchedulePreservesSettledRows(t *testing.T) {
	base := []DraftPayment{
		{SourceID: "paid", DueOn: "2026-01-01", AmountMinor: 1000, PaidMinor: 1000},
		{SourceID: "partial", DueOn: "2026-02-01", AmountMinor: 1000, PaidMinor: 250},
		{SourceID: "unpaid", DueOn: "2026-03-01", AmountMinor: 1000},
	}
	if total, ok := validateSchedule(base, base); !ok || total != 3000 {
		t.Fatal("valid plan rejected")
	}
	changed := append([]DraftPayment{}, base...)
	changed[2].AmountMinor = 1500
	if _, ok := validateSchedule(changed, base); !ok {
		t.Fatal("unpaid edit rejected")
	}
	for _, change := range []func([]DraftPayment) []DraftPayment{
		func(p []DraftPayment) []DraftPayment { p[0].AmountMinor = 1200; return p },
		func(p []DraftPayment) []DraftPayment { p[1].DueOn = "2026-02-02"; return p },
		func(p []DraftPayment) []DraftPayment { return p[1:] },
		func(p []DraftPayment) []DraftPayment { p[2].PaidMinor = 100; return p },
		func(p []DraftPayment) []DraftPayment { p[2].SourceID = "foreign"; return p },
		func(p []DraftPayment) []DraftPayment { p[2].DueOn = p[1].DueOn; return p },
		func(p []DraftPayment) []DraftPayment { p[2].AmountMinor = 9007199254740991; return p },
	} {
		if _, ok := validateSchedule(change(append([]DraftPayment{}, base...)), base); ok {
			t.Fatal("unsafe edit accepted")
		}
	}
}

func ptr[T any](v T) *T { return &v }

// SPEC-0053: parts may be added to paid rows, must add up to the amount, and keep
// "calculated" only when the owner kept Grace's calculation.
func TestScheduleParts(t *testing.T) {
	base := []DraftPayment{{SourceID: "paid", DueOn: "2026-01-01", AmountMinor: 1000, PaidMinor: 1000}, {DueOn: "2026-02-01", AmountMinor: 1000}}
	withParts := append([]DraftPayment{}, base...)
	withParts[0].PrincipalMinor, withParts[0].InterestMinor, withParts[0].OtherMinor = ptr(int64(900)), ptr(int64(100)), ptr(int64(0))
	withParts[1].PrincipalMinor, withParts[1].InterestMinor, withParts[1].OtherMinor = ptr(int64(950)), ptr(int64(50)), ptr(int64(0))
	withParts[1].PartsSource = ptr("calculated")
	if _, ok := validateSchedule(withParts, base); !ok {
		t.Fatal("parts on a paid row rejected")
	}
	if *withParts[0].PartsSource != "bank" || *withParts[1].PartsSource != "calculated" {
		t.Fatalf("sources %v %v", *withParts[0].PartsSource, *withParts[1].PartsSource)
	}
	broken := append([]DraftPayment{}, withParts...)
	broken[1].InterestMinor = ptr(int64(60))
	if _, ok := validateSchedule(broken, base); ok {
		t.Fatal("parts that do not add up accepted")
	}
	partial := append([]DraftPayment{}, base...)
	partial[1].PrincipalMinor = ptr(int64(1000))
	if _, ok := validateSchedule(partial, base); ok {
		t.Fatal("incomplete parts accepted")
	}
}
