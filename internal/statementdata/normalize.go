package statementdata

import (
	"errors"
	"regexp"
	"strconv"
	"strings"
	"time"
	_ "time/tzdata"
)

var decimal = regexp.MustCompile(`^[+-]?(?:[0-9]+|[0-9]{1,3}(?: [0-9]{3})+)(?:[.,][0-9]{1,2})?$`)

// ParseMoney never uses a float, rounds, infers a currency, or accepts exponent
// notation. Signs are retained separately from the positive Posting amount.
func ParseMoney(literal, currency string) (Money, string, error) {
	if !regexp.MustCompile(`^[A-Z]{3}$`).MatchString(currency) {
		return Money{}, "", errors.New("currency_required")
	}
	s := strings.TrimSpace(literal)
	s = strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(s, "\u00a0", " "), "\u202f", " "), "−", "-")
	for _, suffix := range []string{"₽", "RUB", "руб.", "руб", currency} {
		if strings.HasSuffix(s, suffix) {
			if suffix != currency && currency != "RUB" {
				return Money{}, "", errors.New("currency_conflict")
			}
			s = strings.TrimSpace(strings.TrimSuffix(s, suffix))
			break
		}
	}
	if len(s) > 80 || !decimal.MatchString(s) {
		return Money{}, "", errors.New("ambiguous_money")
	}
	direction := ""
	if strings.HasPrefix(s, "-") {
		direction = "debit"
	} else if strings.HasPrefix(s, "+") {
		direction = "credit"
	}
	s = strings.TrimPrefix(strings.TrimPrefix(s, "-"), "+")
	s = strings.ReplaceAll(s, " ", "")
	s = strings.ReplaceAll(s, ",", ".")
	parts := strings.Split(s, ".")
	whole, e := strconv.ParseInt(parts[0], 10, 64)
	if e != nil || whole > MaxWireMinor/100 {
		return Money{}, "", errors.New("amount_overflow")
	}
	fraction := int64(0)
	if len(parts) == 2 {
		f := parts[1]
		if len(f) == 1 {
			f += "0"
		}
		fraction, _ = strconv.ParseInt(f, 10, 64)
	}
	n := whole*100 + fraction
	if n > MaxWireMinor {
		return Money{}, "", errors.New("amount_overflow")
	}
	return Money{n, currency}, direction, nil
}
func Currency(literal string) (string, error) {
	switch strings.TrimSpace(literal) {
	case "₽", "RUB", "руб.", "руб":
		return "RUB", nil
	}
	s := strings.TrimSpace(literal)
	if regexp.MustCompile(`^[A-Z]{3}$`).MatchString(s) {
		return s, nil
	}
	return "", errors.New("invalid_currency")
}

// ParseLocalDate preserves a clock without inventing its zone. Financial
// projection is date-only when UTC is absent; booking remains distinct.
func ParseLocalDate(literal, zone string) (*LocalDate, error) {
	s := strings.Join(strings.Fields(literal), " ")
	for _, layout := range []string{time.RFC3339, "02.01.2006 15:04:05 Z07:00", "02.01.2006 15:04 Z07:00", "2006-01-02 15:04:05 Z07:00", "2006-01-02 15:04 Z07:00"} {
		if t, e := time.Parse(layout, s); e == nil && t.Year() >= 1 {
			return &LocalDate{Local: t.Format("2006-01-02T15:04:05"), Precision: "datetime", TimeZone: t.Format("Z07:00"), UTC: t.UTC().Format(time.RFC3339)}, nil
		}
	}

	for _, layout := range []string{"02.01.2006 15:04:05", "02.01.2006 15:04", "02.01.2006", "2006-01-02 15:04:05", "2006-01-02 15:04", "2006-01-02"} {
		t, e := time.Parse(layout, s)
		if e != nil || t.Year() < 1 {
			continue
		}
		out := &LocalDate{Local: t.Format("2006-01-02"), Precision: "date"}
		if strings.Contains(layout, "15:") {
			out.Local = t.Format("2006-01-02T15:04:05")
			out.Precision = "local_datetime"
		}
		if zone != "" {
			loc, e := time.LoadLocation(zone)
			if e != nil {
				return nil, errors.New("invalid_timezone")
			}
			out.TimeZone = zone
			if out.Precision != "date" {
				local := time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), 0, loc)
				// Reject nonexistent clocks and DST ambiguity; an explicit offset is
				// required rather than picking one of two possible instants.
				if local.Format("2006-01-02T15:04:05") != out.Local {
					return nil, errors.New("ambiguous_local_time")
				}
				for _, delta := range []time.Duration{-2 * time.Hour, -time.Hour, time.Hour, 2 * time.Hour} {
					if local.Add(delta).Format("2006-01-02T15:04:05") == out.Local {
						return nil, errors.New("ambiguous_local_time")
					}
				}
				out.UTC = local.UTC().Format(time.RFC3339)
			}
		}
		return out, nil
	}
	return nil, errors.New("invalid_date")
}
func checkedAdd(a, b int64) (int64, bool) {
	if b > 0 && a > MaxWireMinor-b || b < 0 && a < -MaxWireMinor-b {
		return 0, false
	}
	return a + b, true
}
