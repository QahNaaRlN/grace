// Package calendar holds the deterministic recurring-date rules shared by
// subscriptions (commitments) and obligation schedules.
package calendar

import "time"

// Occurrence returns the n-th date after anchor for a weekly, monthly or
// yearly period. Monthly and yearly dates keep the anchor day and clamp it to
// the last day of a shorter month (31 Jan → 28/29 Feb → 31 Mar).
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
