package obligations

import (
	"errors"
	"strings"
	"time"
)

const dateLayout = "2006-01-02"

var ErrInvalidAsOf = errors.New("invalid asOf date")

func ParseAsOf(raw string, now time.Time) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return now.UTC().Format(dateLayout), nil
	}
	parsed, err := time.Parse(dateLayout, raw)
	if err != nil || parsed.Format(dateLayout) != raw {
		return "", ErrInvalidAsOf
	}
	return raw, nil
}

// validParts accepts a complete non-negative split that adds up to the amount.
func validParts(principal, interest, other *int64, amount int64) bool {
	if principal == nil || interest == nil || other == nil {
		return false
	}
	return *principal >= 0 && *interest >= 0 && *other >= 0 &&
		*principal <= amount && *interest <= amount && *other <= amount &&
		*principal+*interest+*other == amount
}
