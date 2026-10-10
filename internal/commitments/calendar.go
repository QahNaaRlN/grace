package commitments

import "time"

// Occurrence preserves the original day even after February or a short month.
func Occurrence(anchor time.Time, period string, n int) time.Time {
	if period == "weekly" {
		return anchor.AddDate(0, 0, 7*n)
	}
	months := n
	if period == "yearly" {
		months *= 12
	}
	first := time.Date(anchor.Year(), anchor.Month()+time.Month(months), 1, 0, 0, 0, 0, anchor.Location())
	last := first.AddDate(0, 1, -1).Day()
	day := anchor.Day()
	if day > last {
		day = last
	}
	return first.AddDate(0, 0, day-1)
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
