package commitments

import (
	"time"

	"github.com/QahNaaRlN/grace/internal/platform/calendar"
)

// Occurrence preserves the original day even after February or a short month.
func Occurrence(anchor time.Time, period string, n int) time.Time {
	return calendar.Occurrence(anchor, period, n)
}

func NextDate(start, period, end, asOf string, paused bool) *string {
	if paused {
		return nil
	}
	anchor, err := time.Parse("2006-01-02", start)
	if err != nil {
		return nil
	}
	today, err := time.Parse("2006-01-02", asOf)
	if err != nil {
		return nil
	}
	// Jump near the target rather than iterating through years of weekly dates.
	n := 0
	if today.After(anchor) {
		switch period {
		case "weekly":
			n = int(today.Sub(anchor).Hours()/24) / 7
		case "monthly":
			n = (today.Year()-anchor.Year())*12 + int(today.Month()-anchor.Month())
		case "yearly":
			n = today.Year() - anchor.Year()
		default:
			return nil
		}
	}
	date := Occurrence(anchor, period, n)
	if date.Before(today) {
		date = Occurrence(anchor, period, n+1)
	}
	value := date.Format("2006-01-02")
	if end != "" && value > end {
		return nil
	}
	return &value
}
