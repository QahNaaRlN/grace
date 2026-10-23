package statementqualification

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"

	"github.com/QahNaaRlN/grace/internal/statementdata"
	"github.com/QahNaaRlN/grace/internal/statementextract"
)

// No generated Review, Issues, scores or padding of missing precision positions.
type RowPilotMeasurement struct {
	Status, Stage, Cause string
	Result               *Result
	Checks               []ScoringCheck
	Precision            []RowScopedPrecision
	FullQualityPassed    bool
}

func rowPilotMeasurement(o RowScopedOutcome) RowPilotMeasurement {
	return RowPilotMeasurement{o.Status, o.Stage, o.Cause, o.Result, o.Checks, o.Precision, o.FullQualityPassed}
}
func rowPilotBaseMeasurement(d statementdata.Document, raw []byte) RowPilotMeasurement {
	m := RowPilotMeasurement{Status: "unavailable", Stage: "base", Cause: "response_invalid"}
	if len(raw) > statementextract.RowScopedResponseLimit {
		return m
	}
	var p statementdata.Proposal
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if dec.Decode(&p) != nil || dec.Decode(&struct{}{}) != io.EOF {
		return m
	}
	plan, e := statementextract.BuildComparisonPlan(d, statementextract.ComparisonCandidates()[0].Policy)
	if e != nil {
		return m
	}
	p, e = statementextract.CanonicalComparisonProposal(plan, p)
	if e != nil {
		return m
	}
	p, e = statementextract.MergeProposals(plan, []statementdata.Proposal{p})
	if e != nil {
		return m
	}
	pages := []int{}
	for _, pg := range d.Pages {
		pages = append(pages, pg.Number)
	}
	review, e := statementdata.ValidateProposal(d, p, pages, true, "")
	if e != nil {
		return m
	}
	result, checks := ScreeningChecks("cards", review)
	m = RowPilotMeasurement{Status: "complete", Stage: "validation", Result: &result, Checks: checks, Precision: []RowScopedPrecision{}, FullQualityPassed: result.Passed && len(review.Rows) == 2}
	for i, row := range review.Rows {
		op := row.OperationDate != nil && row.OperationDate.Precision == "local_datetime"
		bk := row.BookingDate != nil && row.BookingDate.Precision == "local_datetime"
		m.Precision = append(m.Precision, RowScopedPrecision{i + 1, op, bk})
		m.FullQualityPassed = m.FullQualityPassed && op && bk
	}
	return m
}
func rowPilotSemantic(m RowPilotMeasurement) bool {
	switch m.Cause {
	case "partition_invalid", "partition_limit", "partition_unresolved", "row_subset_invalid", "row_coverage_incomplete", "context_unused":
		return m.Result == nil
	}
	return false
}
func rowPilotDecisive(m RowPilotMeasurement) bool {
	return rowPilotSemantic(m) || m.Status == "complete" && !m.FullQualityPassed
}
func rowPilotJudge(attempts []RowPilotAttempt, problem string) string {
	if problem != "" {
		return ClockInconclusive
	}
	for _, a := range attempts {
		if a.Variant == "candidate" && a.Measurement != nil && rowPilotDecisive(*a.Measurement) {
			return ClockReject
		}
	}
	gain := false
	if len(attempts) != 6 {
		return ClockInconclusive
	}
	for _, a := range attempts {
		m := a.Measurement
		if a.State != "completed" || m == nil || m.Status != "complete" || m.Result == nil {
			return ClockInconclusive
		}
		if a.Variant == "candidate" && !m.FullQualityPassed {
			return ClockReject
		}
		if a.Variant == "base" && !m.FullQualityPassed {
			gain = true
		}
	}
	if gain {
		return RowPilotKeep
	}
	return ClockInconclusive
}
func rowPilotFailure(stage, cause string) RowPilotMeasurement {
	return RowPilotMeasurement{Status: "unavailable", Stage: stage, Cause: cause}
}

var errRowPilotReport = errors.New("row_pilot_report_invalid")
