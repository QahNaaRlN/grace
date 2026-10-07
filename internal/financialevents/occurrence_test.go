package financialevents

import (
	"testing"
	"time"
)

func TestOccurrencePreservesCalendarPrecision(t *testing.T) {
	for _, date := range []string{"2026-09-02", "2024-02-29"} {
		if !validOccurrence(Command{OccurredDate: date}) {
			t.Fatal(date)
		}
	}
	for _, date := range []string{"2026-02-30", "0000-01-01", "2026-9-2", "2026-09-02T00:00:00Z"} {
		if validOccurrence(Command{OccurredDate: date}) {
			t.Fatal("invalid date accepted", date)
		}
	}
	if validOccurrence(Command{}) || validOccurrence(Command{OccurredDate: "2026-09-02", OccurredAt: time.Now()}) {
		t.Fatal("ambiguous occurrence accepted")
	}
	date := "2026-09-02"
	e := Event{OccurredDate: &date}
	if e.OccurrenceValue() != date || e.OccurredAt != nil {
		t.Fatal("time invented")
	}
}
