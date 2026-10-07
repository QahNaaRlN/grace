package financialevents

import (
	"time"
	_ "time/tzdata"
)

func ValidCalendarDate(value string) bool {
	t, err := time.Parse("2006-01-02", value)
	return err == nil && len(value) == 10 && t.Year() >= 1 && t.Format("2006-01-02") == value
}
func validOccurrence(c Command) bool {
	if c.OccurredDate != "" {
		return c.OccurredAt.IsZero() && ValidCalendarDate(c.OccurredDate)
	}
	return !c.OccurredAt.IsZero()
}
func (e Event) OccurrenceValue() string {
	if e.OccurredDate != nil {
		return *e.OccurredDate
	}
	if e.OccurredAt != nil {
		return e.OccurredAt.Format(time.RFC3339)
	}
	return ""
}
