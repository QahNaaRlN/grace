package analytics

import (
	"errors"
	"strings"
	"time"
)

const dateLayout = "2006-01-02"

var (
	ErrInvalidPeriod      = errors.New("invalid analytics period")
	ErrInvalidCashHorizon = errors.New("invalid cash position horizon")
)

func ParsePeriod(fromRaw, toRaw string) (time.Time, time.Time, error) {
	fromRaw = strings.TrimSpace(fromRaw)
	toRaw = strings.TrimSpace(toRaw)
	from, fromErr := time.Parse(time.RFC3339, fromRaw)
	to, toErr := time.Parse(time.RFC3339, toRaw)
	if fromRaw == "" || toRaw == "" || fromErr != nil || toErr != nil || !from.Before(to) || to.After(from.AddDate(5, 0, 0)) {
		return time.Time{}, time.Time{}, ErrInvalidPeriod
	}
	return from, to, nil
}

func ParseCashHorizon(asOfRaw, throughRaw string, now time.Time) (string, string, error) {
	asOfRaw = strings.TrimSpace(asOfRaw)
	if asOfRaw == "" {
		asOfRaw = now.UTC().Format(dateLayout)
	}
	throughRaw = strings.TrimSpace(throughRaw)
	asOf, asOfErr := time.Parse(dateLayout, asOfRaw)
	through, throughErr := time.Parse(dateLayout, throughRaw)
	if throughRaw == "" || asOfErr != nil || throughErr != nil || asOf.Format(dateLayout) != asOfRaw ||
		through.Format(dateLayout) != throughRaw || through.Before(asOf) || through.After(asOf.AddDate(5, 0, 0)) {
		return "", "", ErrInvalidCashHorizon
	}
	return asOfRaw, throughRaw, nil
}
