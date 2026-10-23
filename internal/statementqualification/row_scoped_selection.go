package statementqualification

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/QahNaaRlN/grace/internal/statementdata"
	"github.com/QahNaaRlN/grace/internal/statementextract"
)

// No provider implementation or runtime entry point exists for these two seams.
// The accepted D1-D3 package injects offline fakes only.
type RowScopedStages struct {
	Grouping func(context.Context, statementextract.RowScopedRequest) ([]byte, *statementextract.Billing, error)
	Row      func(context.Context, statementextract.RowScopedRequest) ([]byte, *statementextract.Billing, error)
}
type RowScopedCall struct {
	Stage        string
	RowOrdinal   int
	RequestHash  string
	ResponseHash string
	CostMicros   *int64
}
type RowScopedPrecision struct {
	RowOrdinal                       int
	OperationMatches, BookingMatches bool
}
type RowScopedOutcome struct {
	Version, Status, Stage, Cause string
	DocumentHash, PlanHash        string
	MaximumMicros                 int64
	ActualMicros                  *int64
	SettlementRecorded            bool
	Calls                         []RowScopedCall
	Result                        *Result
	Checks                        []ScoringCheck
	Precision                     []RowScopedPrecision
	FullQualityPassed             bool
	Review                        *statementdata.Review   `json:"-"`
	Proposal                      *statementdata.Proposal `json:"-"`
}

// RunRowScopedOffline is a bounded feasibility runner, not qualification or an
// authorization token. It has no client/config/SQL/SSH/default transport wiring.
func RunRowScopedOffline(ctx context.Context, s Sample, stages RowScopedStages, meter Meter) (RowScopedOutcome, error) {
	return runRowScoped(ctx, s, stages, meter)
}

// Shared bounded core. The offline entry retains fake-only wiring; the fixed
// separately authorized campaign supplies its own guarded stage transport.
func runRowScoped(ctx context.Context, s Sample, stages RowScopedStages, meter Meter) (out RowScopedOutcome, err error) {
	out = RowScopedOutcome{Version: statementextract.RowScopedVersion, Status: "unavailable", Calls: []RowScopedCall{}}
	fail := func(stage, cause, status string) error {
		out.Status = status
		out.Stage = stage
		out.Cause = cause
		out.FullQualityPassed = false
		return errors.New(cause)
	}
	refused := func(stage string, e error) error {
		cause := statementextract.RowScopedCause(e)
		status := "rejected"
		if cause == "response_invalid" {
			status = "unavailable"
		}
		return fail(stage, cause, status)
	}
	validName := false
	for _, n := range ScreeningNames() {
		if n == s.Name {
			validName = true
		}
	}
	if !validName || s.SHA256 != FixtureHash(s.Name) || stages.Grouping == nil || stages.Row == nil || meter == nil {
		err = fail("grouping", "response_invalid", "unavailable")
		return
	}
	ctx, cancel := context.WithTimeout(ctx, 390*time.Second)
	defer cancel()
	request, e := statementextract.BuildRowGrouping(s.Document)
	if e != nil {
		err = refused("grouping", e)
		return
	}
	root, e := statementextract.BuildComparisonPlan(s.Document, request.Policy)
	if e != nil {
		err = fail("grouping", "request_limit_exceeded", "rejected")
		return
	}
	out.DocumentHash = s.Document.Hash()
	out.PlanHash = statementextract.Hash(root)
	out.MaximumMicros = statementextract.RowScopedMaxCalls * request.Policy.CallMicros()
	if ctx.Err() != nil {
		err = fail("grouping", "deadline_exceeded", "unavailable")
		return
	}
	reservation, e := meter.Reserve(ctx, out.MaximumMicros, request.Policy)
	if e != nil || reservation == "" || len(reservation) > 128 {
		err = fail("grouping", "billing_unavailable", "unavailable")
		return
	}
	total := int64(0)
	known := true
	ids := map[string]bool{}
	defer func() {
		settleCtx, stop := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer stop()
		var actual *int64
		if known {
			n := total
			actual = &n
			out.ActualMicros = &n
		}
		if meter.Settle(settleCtx, reservation, actual) != nil {
			err = fail(out.Stage, "settlement_failed", "unavailable")
			return
		}
		out.SettlementRecorded = true
	}()
	call := func(r statementextract.RowScopedRequest) ([]byte, error) {
		stage := r.Stage
		if r.Stage == "row" {
			stage = fmt.Sprintf("row%d", r.RowOrdinal)
		}
		if ctx.Err() != nil {
			return nil, fail(stage, "deadline_exceeded", "unavailable")
		}
		if len(out.Calls) >= statementextract.RowScopedMaxCalls {
			return nil, fail(stage, "partition_limit", "rejected")
		}
		cctx, stop := context.WithTimeout(ctx, 120*time.Second)
		defer stop()
		complete := stages.Grouping
		if r.Stage == "row" {
			complete = stages.Row
		}
		prefixKnown := known
		// Until a valid bill returns, delivery/panic is uncertain rather than free.
		known = false
		raw, bill, callErr := complete(cctx, r)
		trace := RowScopedCall{Stage: stage, RowOrdinal: r.RowOrdinal, RequestHash: r.Hash}
		if !validBill(bill, r.Policy) || ids[bill.GenerationID] {
			known = false
			out.Calls = append(out.Calls, trace)
			return nil, fail(stage, "billing_unavailable", "unavailable")
		}
		known = prefixKnown
		ids[bill.GenerationID] = true
		n := bill.CostMicros
		trace.CostMicros = &n
		total += n
		if len(raw) <= statementextract.RowScopedResponseLimit {
			trace.ResponseHash = statementextract.PromptSHA(string(raw))
		}
		out.Calls = append(out.Calls, trace)
		if cctx.Err() != nil {
			return nil, fail(stage, "deadline_exceeded", "unavailable")
		}
		if callErr != nil {
			return nil, fail(stage, "call_unavailable", "unavailable")
		}
		if len(raw) > statementextract.RowScopedResponseLimit {
			return nil, fail(stage, "response_invalid", "unavailable")
		}
		return raw, nil
	}
	raw, e := call(request)
	if e != nil {
		return out, e
	}
	g, e := statementextract.DecodeRowGrouping(s.Document, raw)
	if e != nil {
		err = refused("grouping", e)
		return
	}
	if len(g.Unresolved) > 0 {
		err = fail("grouping", "partition_unresolved", "incomplete")
		return
	}
	rows := []statementdata.RowMapping{}
	for i := range g.Rows {
		stage := fmt.Sprintf("row%d", i+1)
		r, e := statementextract.BuildRowFields(s.Document, g, i+1)
		if e != nil {
			err = refused(stage, e)
			return
		}
		raw, e := call(r)
		if e != nil {
			return out, e
		}
		row, e := statementextract.DecodeRowFields(s.Document, g, i+1, raw)
		if e != nil {
			err = refused(stage, e)
			return
		}
		rows = append(rows, row)
	}
	p, e := statementextract.ComposeRowScoped(s.Document, g, rows)
	if e != nil {
		err = refused("composition", e)
		return
	}
	p, e = statementextract.CanonicalComparisonProposal(root, p)
	if e != nil {
		err = fail("composition", "qualification_protocol_failed", "unavailable")
		return
	}
	p, e = statementextract.MergeProposals(root, []statementdata.Proposal{p})
	if e != nil {
		err = fail("composition", "qualification_merge_failed", "unavailable")
		return
	}
	pages := []int{}
	for _, pg := range s.Document.Pages {
		pages = append(pages, pg.Number)
	}
	review, e := statementdata.ValidateProposal(s.Document, p, pages, true, "")
	if e != nil {
		err = fail("validation", "qualification_grounding_failed", "rejected")
		return
	}
	if ctx.Err() != nil {
		err = fail("validation", "deadline_exceeded", "unavailable")
		return
	}
	result, checks := ScreeningChecks(s.Name, review)
	out.Status = "complete"
	out.Stage = "validation"
	out.Result = &result
	out.Checks = checks
	out.Precision = []RowScopedPrecision{}
	out.Review = &review
	out.Proposal = &p
	out.FullQualityPassed = result.Passed && len(review.Rows) == 2
	for i, row := range review.Rows {
		expected := "date"
		if s.Name == "cards" {
			expected = "local_datetime"
		}
		op := row.OperationDate != nil && row.OperationDate.Precision == expected
		book := row.BookingDate != nil && row.BookingDate.Precision == expected
		if s.Name == "turnover-roles-ru" {
			book = row.BookingDate == nil
		}
		out.Precision = append(out.Precision, RowScopedPrecision{i + 1, op, book})
		out.FullQualityPassed = out.FullQualityPassed && op && book
	}
	return out, nil
}
