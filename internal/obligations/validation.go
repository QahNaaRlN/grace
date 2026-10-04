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
