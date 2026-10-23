package statementqualification

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"reflect"
	"strings"
	"unicode/utf8"

	"github.com/QahNaaRlN/grace/internal/statementdata"
)

func EncodeClockReport(w io.Writer, r ClockReport, d statementdata.Document) error {
	if ValidateClockReport(r, d) != nil {
		return errors.New("clock_report_invalid")
	}
	raw, e := json.Marshal(r)
	if e != nil || len(raw)+1 > ClockPilotReportLimit {
		return errors.New("clock_report_invalid")
	}
	_, e = w.Write(append(raw, '\n'))
	return e
}
func DecodeClockReport(reader io.Reader, d statementdata.Document) (ClockReport, error) {
	fail := errors.New("clock_report_invalid")
	raw, e := io.ReadAll(io.LimitReader(reader, ClockPilotReportLimit+1))
	if e != nil || len(raw) > ClockPilotReportLimit || !utf8.Valid(raw) || clockUniqueJSON(raw) != nil {
		return ClockReport{}, fail
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	var r ClockReport
	if dec.Decode(&r) != nil || dec.Decode(&struct{}{}) != io.EOF || ValidateClockReport(r, d) != nil {
		return ClockReport{}, fail
	}
	// Reject case-insensitive aliases, explicit extra nulls, or other silent typed normalization.
	canonical, e := json.Marshal(r)
	if e != nil {
		return ClockReport{}, fail
	}
	var got, want any
	if json.Unmarshal(raw, &got) != nil || json.Unmarshal(canonical, &want) != nil || !reflect.DeepEqual(got, want) {
		return ClockReport{}, fail
	}
	return r, nil
}
func clockUniqueJSON(raw []byte) error {
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	nodes := 0
	var walk func(int) error
	walk = func(depth int) error {
		nodes++
		if depth > 32 || nodes > 200000 {
			return errors.New("limit")
		}
		token, e := dec.Token()
		if e != nil {
			return e
		}
		if delim, ok := token.(json.Delim); ok {
			switch delim {
			case '{':
				seen := map[string]bool{}
				for dec.More() {
					key, e := dec.Token()
					if e != nil {
						return e
					}
					s, ok := key.(string)
					if !ok {
						return errors.New("key")
					}
					s = strings.ToLower(s)
					if seen[s] {
						return errors.New("duplicate")
					}
					seen[s] = true
					if e = walk(depth + 1); e != nil {
						return e
					}
				}
			case '[':
				for dec.More() {
					if e := walk(depth + 1); e != nil {
						return e
					}
				}
			default:
				return errors.New("delimiter")
			}
			_, e = dec.Token()
			return e
		}
		return nil
	}
	if e := walk(0); e != nil {
		return e
	}
	if _, e := dec.Token(); e != io.EOF {
		return errors.New("trailing")
	}
	return nil
}

type ClockPublicSummary struct {
	Version              string `json:"version"`
	ShapeVerified        bool   `json:"shapeVerified"`
	ActivationAuthorized bool   `json:"activationAuthorized"`
	Decision             string `json:"decision"`
	Error                string `json:"error"`
	RecordedCalls        int    `json:"recordedCalls"`
	Completed            int    `json:"completed"`
	SettlementRecorded   bool   `json:"settlementRecorded"`
	ReservationSettled   bool   `json:"reservationSettled"`
	BillingKnown         bool   `json:"billingKnown"`
	ClockCounts          []*int `json:"clockCounts"`
}

func SummarizeClock(r ClockReport, d statementdata.Document) (ClockPublicSummary, error) {
	out := ClockPublicSummary{Version: ClockPilotVersion, ClockCounts: []*int{}}
	if ValidateClockReport(r, d) != nil {
		return out, errors.New("clock_report_invalid")
	}
	out.ShapeVerified = true
	out.Decision = r.Decision
	out.Error = r.Error
	out.SettlementRecorded = r.SettlementRecorded
	out.ReservationSettled = r.SettlementRecorded && r.ActualMicros != nil
	out.BillingKnown = r.ActualMicros != nil
	for _, a := range r.Attempts {
		if a.State != "not_sent" {
			out.RecordedCalls++
		}
		if a.Measurement != nil {
			out.Completed++
			if a.Measurement.Count != nil {
				n := *a.Measurement.Count
				out.ClockCounts = append(out.ClockCounts, &n)
			} else {
				out.ClockCounts = append(out.ClockCounts, nil)
			}
		} else {
			out.ClockCounts = append(out.ClockCounts, nil)
		}
	}
	return out, nil
}
