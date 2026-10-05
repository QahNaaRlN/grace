package importreview

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"regexp"
	"strings"
	"time"
)

var ErrSnapshot = errors.New("invalid_snapshot")
var numberToken = regexp.MustCompile(`^-?(0|[1-9][0-9]*)(\.[0-9]+)?([eE][+-]?[0-9]+)?$`)

func Load(r io.Reader) (Snapshot, error) {
	body, err := io.ReadAll(io.LimitReader(r, MaxBytes+1))
	if err != nil || len(body) > MaxBytes {
		return Snapshot{}, ErrSnapshot
	}
	// Reject duplicate JSON members rather than accepting a last-value-wins snapshot.
	check := json.NewDecoder(bytes.NewReader(body))
	check.UseNumber()
	if checkJSON(check, 0) != nil {
		return Snapshot{}, ErrSnapshot
	}
	if _, err := check.Token(); !errors.Is(err, io.EOF) {
		return Snapshot{}, ErrSnapshot
	}
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	var s Snapshot
	if decoder.Decode(&s) != nil || validate(s) != nil {
		return Snapshot{}, ErrSnapshot
	}
	return s, nil
}

func checkJSON(d *json.Decoder, depth int) error {
	if depth > 40 {
		return ErrSnapshot
	}
	t, err := d.Token()
	if err != nil {
		return err
	}
	delim, ok := t.(json.Delim)
	if !ok {
		return nil
	}
	switch delim {
	case '{':
		seen := map[string]bool{}
		for d.More() {
			key, err := d.Token()
			if err != nil {
				return err
			}
			name, ok := key.(string)
			if !ok || seen[name] {
				return ErrSnapshot
			}
			seen[name] = true
			if err := checkJSON(d, depth+1); err != nil {
				return err
			}
		}
	case '[':
		for d.More() {
			if err := checkJSON(d, depth+1); err != nil {
				return err
			}
		}
	default:
		return ErrSnapshot
	}
	_, err = d.Token()
	return err
}

func validate(s Snapshot) error {
	if s.SchemaVersion != SnapshotVersion || strings.TrimSpace(s.SourceID) == "" || s.Locale != "ru_RU" || len(s.Sheets) == 0 || len(s.Sheets) > 20 {
		return ErrSnapshot
	}
	if _, err := time.Parse(time.RFC3339, s.CapturedAt); err != nil {
		return ErrSnapshot
	}
	for _, zone := range []string{s.DeclaredTimeZone, s.ConfirmedTimeZone} {
		if zone == "" {
			return ErrSnapshot
		}
		if _, err := time.LoadLocation(zone); err != nil {
			return ErrSnapshot
		}
	}
	names := map[string]bool{}
	ids := map[int64]bool{}
	totalRows, totalCells := 0, 0
	for _, sh := range s.Sheets {
		if names[sh.Title] || ids[sh.SheetID] || sh.Title == "" || sh.SheetID < 0 || sh.RowCount < 1 || sh.RowCount > MaxRows || sh.ColumnCount < 1 || sh.ColumnCount > 26 {
			return ErrSnapshot
		}
		names[sh.Title] = true
		ids[sh.SheetID] = true
		occupied := map[[2]int]bool{}
		for _, rg := range sh.Ranges {
			if rg.StartRow < 1 || rg.StartColumn < 1 || rg.StartRow > sh.RowCount || rg.StartColumn > sh.ColumnCount || len(rg.Rows) == 0 || len(rg.Rows) > sh.RowCount-rg.StartRow+1 {
				return ErrSnapshot
			}
			width := 0
			for _, row := range rg.Rows {
				if len(row.Values) > width {
					width = len(row.Values)
				}
			}
			if width == 0 || width > 26 || width > sh.ColumnCount-rg.StartColumn+1 {
				return ErrSnapshot
			}
			totalRows += len(rg.Rows)
			totalCells += width * len(rg.Rows)
			if totalRows > MaxRows || totalCells > MaxCells {
				return ErrSnapshot
			}
			for i, row := range rg.Rows {
				for j := 0; j < width; j++ {
					coord := [2]int{rg.StartRow + i, rg.StartColumn + j}
					if occupied[coord] {
						return ErrSnapshot
					}
					occupied[coord] = true
				}
				for _, cell := range row.Values {
					for _, value := range []*Value{cell.UserEnteredValue, cell.EffectiveValue} {
						if value == nil {
							continue
						}
						count := 0
						if len(value.NumberValue) > 0 {
							count++
							if !numberToken.Match(value.NumberValue) {
								return ErrSnapshot
							}
						}
						if value.StringValue != nil {
							count++
						}
						if value.BoolValue != nil {
							count++
						}
						if value.FormulaValue != nil {
							count++
						}
						if count != 1 {
							return ErrSnapshot
						}
					}
				}
			}
		}
	}
	for _, aliases := range []map[string]string{s.Mappings.Accounts, s.Mappings.Categories} {
		for alias, id := range aliases {
			if strings.TrimSpace(alias) == "" || strings.TrimSpace(id) == "" {
				return ErrSnapshot
			}
		}
	}
	for alias, f := range s.Mappings.Funding {
		if strings.TrimSpace(alias) == "" || (f.Kind != "own" && f.Kind != "borrowed" && f.Kind != "unknown") || (f.Kind != "borrowed" && f.FacilityID != "") {
			return ErrSnapshot
		}
	}
	return nil
}
