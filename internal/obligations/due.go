package obligations

import "time"

// DueSoonDays is how many days before its date a payment is «скоро»: the same
// windows as the 3- and 1-day email reminders of SPEC-0034 (SPEC-0046).
const DueSoonDays = 3

// DueIn is the number of calendar days from asOf to dueOn (both YYYY-MM-DD,
// Asia/Yekaterinburg dates); negative when the date has passed.
func DueIn(asOf, dueOn string) int {
	from, errFrom := time.Parse(time.DateOnly, asOf)
	to, errTo := time.Parse(time.DateOnly, dueOn)
	if errFrom != nil || errTo != nil {
		return 0
	}
	return int(to.Sub(from).Hours() / 24)
}

// DueSoon reports an unpaid remainder due today or within DueSoonDays.
func DueSoon(days int, remainingMinor int64) bool {
	return remainingMinor > 0 && days >= 0 && days <= DueSoonDays
}
