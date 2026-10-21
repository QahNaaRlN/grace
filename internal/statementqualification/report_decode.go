package statementqualification

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"strings"
)

// The protected report still uses a single unambiguous JSON document.
func DecodeReport(reader io.Reader) (Report, error) {
	var out Report
	e := decodeOperatorReport(reader, &out)
	return out, e
}
func DecodeComparisonReport(reader io.Reader) (ComparisonReport, error) {
	var out ComparisonReport
	e := decodeOperatorReport(reader, &out)
	return out, e
}
func decodeOperatorReport(reader io.Reader, out any) error {
	raw, e := io.ReadAll(io.LimitReader(reader, 524289))
	if e != nil || len(raw) > 524288 {
		return errors.New("invalid_qualification_report")
	}
	tokens := json.NewDecoder(bytes.NewReader(raw))
	var value func() error
	value = func() error {
		t, e := tokens.Token()
		if e != nil {
			return e
		}
		if d, ok := t.(json.Delim); ok {
			switch d {
			case '{':
				keys := map[string]bool{}
				for tokens.More() {
					k, e := tokens.Token()
					if e != nil {
						return e
					}
					key, ok := k.(string)
					if !ok || keys[strings.ToLower(key)] {
						return errors.New("duplicate_report_key")
					}
					keys[strings.ToLower(key)] = true
					if e = value(); e != nil {
						return e
					}
				}
				t, e = tokens.Token()
				if e != nil || t != json.Delim('}') {
					return errors.New("invalid_report_object")
				}
			case '[':
				for tokens.More() {
					if e = value(); e != nil {
						return e
					}
				}
				t, e = tokens.Token()
				if e != nil || t != json.Delim(']') {
					return errors.New("invalid_report_array")
				}
			default:
				return errors.New("invalid_report_delimiter")
			}
		}
		return nil
	}
	if e = value(); e != nil {
		return e
	}
	if _, e = tokens.Token(); e != io.EOF {
		return errors.New("trailing_report_data")
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	e = dec.Decode(out)
	return e
}
