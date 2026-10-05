package importreview

import (
	"encoding/json"
	"errors"
	"math/big"
	"regexp"
	"strings"
	"time"
	_ "time/tzdata"
)

func cellValue(c Cell) *Value {
	if c.EffectiveValue != nil {
		return c.EffectiveValue
	}
	if c.UserEnteredValue != nil && c.UserEnteredValue.FormulaValue == nil {
		return c.UserEnteredValue
	}
	return nil
}
func text(c Cell) string {
	v := cellValue(c)
	if v == nil {
		return ""
	}
	if v.StringValue != nil {
		return *v.StringValue
	}
	if len(v.NumberValue) > 0 {
		return string(v.NumberValue)
	}
	if v.BoolValue != nil {
		if *v.BoolValue {
			return "true"
		}
		return "false"
	}
	return ""
}
func formula(c Cell) string {
	if c.UserEnteredValue != nil && c.UserEnteredValue.FormulaValue != nil {
		return *c.UserEnteredValue.FormulaValue
	}
	return ""
}
func meaningful(c Cell) bool {
	v := cellValue(c)
	return v != nil && (v.StringValue == nil || strings.TrimSpace(*v.StringValue) != "")
}

var plainDecimal = regexp.MustCompile(`^[+-]?\d+(?:[.,]\d+)?$`)
var groupedDecimal = regexp.MustCompile(`^[+-]?\d{1,3}(?: \d{3})+(?:[.,]\d+)?$`)

// Money accepts an exact decimal, rejects ambiguous grouping/precision and never rounds.
func Money(c Cell) (*int64, error) {
	v := cellValue(c)
	if v == nil || v.StringValue != nil && strings.TrimSpace(*v.StringValue) == "" {
		return nil, nil
	}
	s := text(c)
	if v.StringValue != nil {
		s = strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(s), "₽"))
		s = strings.ReplaceAll(strings.ReplaceAll(s, "\u00a0", " "), "\u202f", " ")
		if !plainDecimal.MatchString(s) && !groupedDecimal.MatchString(s) {
			return nil, errors.New("ambiguous_money")
		}
		s = strings.ReplaceAll(strings.ReplaceAll(s, " ", ""), ",", ".")
	} else if len(v.NumberValue) == 0 {
		return nil, errors.New("ambiguous_money")
	}
	// JSON exponent notation is bounded by the input limit; bound exponent/length again
	// before big.Rat to prevent a tiny input asking for an enormous integer allocation.
	if len(s) > 80 {
		return nil, errors.New("ambiguous_money")
	}
	if i := strings.IndexAny(s, "eE"); i >= 0 {
		e, ok := new(big.Int).SetString(strings.TrimPrefix(s[i+1:], "+"), 10)
		if !ok || !e.IsInt64() || e.Int64() < -30 || e.Int64() > 30 {
			return nil, errors.New("ambiguous_money")
		}
	}
	r, ok := new(big.Rat).SetString(s)
	if !ok {
		return nil, errors.New("ambiguous_money")
	}
	r.Mul(r, big.NewRat(100, 1))
	if !r.IsInt() || !r.Num().IsInt64() {
		return nil, errors.New("ambiguous_money")
	}
	n := r.Num().Int64()
	return &n, nil
}

func NormalizeDate(c Cell, zone string) (*Date, error) {
	v := cellValue(c)
	if v == nil || v.StringValue != nil && strings.TrimSpace(*v.StringValue) == "" {
		return nil, nil
	}
	loc, err := time.LoadLocation(zone)
	if err != nil {
		return nil, errors.New("invalid_date")
	}
	if len(v.NumberValue) > 0 {
		s := string(v.NumberValue)
		if len(s) > 24 || strings.ContainsAny(s, "eE") {
			return nil, errors.New("invalid_date")
		}
		r, ok := new(big.Rat).SetString(s)
		if !ok || r.Sign() < 0 || r.Cmp(big.NewRat(110000, 1)) > 0 {
			return nil, errors.New("invalid_date")
		}
		days := new(big.Int).Quo(r.Num(), r.Denom())
		day := time.Date(1899, 12, 30, 0, 0, 0, 0, loc).AddDate(0, 0, int(days.Int64()))
		if r.IsInt() {
			return &Date{Local: day.Format("2006-01-02"), Precision: "date", TimeZone: zone}, nil
		}
		fraction := new(big.Rat).Sub(r, new(big.Rat).SetInt(days))
		fraction.Mul(fraction, big.NewRat(86400, 1))
		// Decimal serial fractions often approximate a clock time. Preserve a known
		// calendar date and raw fraction rather than rounding it to an invented instant.
		if !fraction.IsInt() {
			return &Date{Local: day.Format("2006-01-02"), Precision: "serial_fraction", TimeZone: zone}, nil
		}
		local := day.Add(time.Duration(fraction.Num().Int64()) * time.Second)
		return &Date{Local: local.Format("2006-01-02T15:04:05"), Precision: "serial_second", TimeZone: zone, UTC: local.UTC().Format(time.RFC3339)}, nil
	}
	s := strings.TrimSpace(text(c))
	for _, layout := range []string{"2006-01-02", "02.01.2006"} {
		if t, e := time.ParseInLocation(layout, s, loc); e == nil && t.Format(layout) == s {
			return &Date{Local: t.Format("2006-01-02"), Precision: "date", TimeZone: zone}, nil
		}
	}
	for _, layout := range []string{"2006-01-02 15:04", "2006-01-02 15:04:05", "2006-01-02T15:04:05", "02.01.2006 15:04"} {
		if t, e := time.ParseInLocation(layout, s, loc); e == nil && t.Format(layout) == s {
			precision := "second"
			if len(s) == 16 {
				precision = "minute"
			}
			return &Date{Local: t.Format("2006-01-02T15:04:05"), Precision: precision, TimeZone: zone, UTC: t.UTC().Format(time.RFC3339)}, nil
		}
	}
	if t, e := time.Parse(time.RFC3339, s); e == nil {
		return &Date{Local: t.In(loc).Format("2006-01-02T15:04:05"), Precision: "second", TimeZone: zone, UTC: t.UTC().Format(time.RFC3339)}, nil
	}
	return nil, errors.New("invalid_date")
}

func boolCell(c Cell) (*bool, bool) {
	v := cellValue(c)
	if v == nil || text(c) == "" {
		return nil, true
	}
	if v.BoolValue != nil {
		return v.BoolValue, true
	}
	switch strings.ToLower(strings.TrimSpace(text(c))) {
	case "да", "true":
		b := true
		return &b, true
	case "нет", "false":
		b := false
		return &b, true
	}
	return nil, false
}
func canonical(s string) string { return strings.ToLower(strings.Join(strings.Fields(s), " ")) }
func jsonEqual(a, b Cell) bool {
	x, _ := json.Marshal(a)
	y, _ := json.Marshal(b)
	return string(x) == string(y)
}
