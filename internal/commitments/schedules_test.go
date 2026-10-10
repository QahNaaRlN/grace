package commitments

import "testing"

func TestSchedulePreservesSettledRows(t *testing.T) {
	base := []DraftPayment{{"paid", "2026-01-01", 1000, 1000}, {"partial", "2026-02-01", 1000, 250}, {"unpaid", "2026-03-01", 1000, 0}}
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
