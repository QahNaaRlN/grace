package obligations

import "testing"

func TestDueSoonWindows(t *testing.T) {
	for _, c := range []struct {
		due       string
		remaining int64
		days      int
		soon      bool
	}{
		{"2026-10-14", 100, 0, true},
		{"2026-10-15", 100, 1, true},
		{"2026-10-17", 100, 3, true},
		{"2026-10-18", 100, 4, false},
		{"2026-10-13", 100, -1, false},
		{"2026-10-15", 0, 1, false},
		{"2026-11-01", 100, 18, false},
		{"2027-01-01", 100, 79, false},
	} {
		days := DueIn("2026-10-14", c.due)
		if days != c.days || DueSoon(days, c.remaining) != c.soon {
			t.Fatalf("%s: days %d soon %v", c.due, days, DueSoon(days, c.remaining))
		}
	}
}
