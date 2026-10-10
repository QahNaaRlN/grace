package commitments

import (
	"testing"
	"time"
)

func TestAnchorDoesNotDrift(t *testing.T) {
	for _, test := range []struct{ start, period, asOf, want string }{
		{"2026-01-31", "monthly", "2026-02-01", "2026-02-28"},
		{"2026-01-31", "monthly", "2026-03-01", "2026-03-31"},
		{"2024-02-29", "yearly", "2028-01-01", "2028-02-29"},
		{"2026-01-01", "weekly", "2026-01-09", "2026-01-15"},
		{"2026-01-31", "monthly", "2026-01-31", "2026-01-31"},
	} {
		got := NextDate(test.start, test.period, "", test.asOf, false)
		if got == nil || *got != test.want {
			t.Errorf("%+v: %v", test, got)
		}
	}
	if NextDate("2026-01-31", "monthly", "2026-02-27", "2026-02-01", false) != nil {
		t.Fatal("past end")
	}
	if NextDate("2026-01-31", "monthly", "", "2026-02-01", true) != nil {
		t.Fatal("paused")
	}
	anchor, _ := time.Parse("2006-01-02", "2026-01-31")
	if Occurrence(anchor, "monthly", 2).Day() != 31 {
		t.Fatal("drift")
	}
}
