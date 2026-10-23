package statementqualification

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"regexp"
	"time"

	"github.com/QahNaaRlN/grace/internal/statementextract"
)

const RowPilotVersion = "statement.row-pilot.s121.v1"
const RowPilotAuthorization = "s121-row-pilot"
const RowPilotKeep = "KEEP_FOR_SCALING_REVIEW"
const RowPilotMaxMicros int64 = 646800
const RowPilotReportLimit = 1048576

func RowPilotOrder() [6]string {
	return [6]string{"base", "candidate", "candidate", "base", "base", "candidate"}
}

type RowPilotReasoner interface {
	CurrentPolicy(context.Context) (statementextract.Policy, error)
	ComparisonProof(context.Context) (statementextract.ComparisonIdentityProof, error)
	MetadataRequests() int
	CompleteRowPilot(context.Context, statementextract.RowScopedRequest, statementextract.ComparisonIdentityProof) ([]byte, *statementextract.Billing, error)
}
type RowPilotRequestEvidence struct {
	Stage                      string
	RowOrdinal                 int
	Hash, PromptSHA, SchemaSHA string
	InputBytes                 int
}

func rowPilotEvidence(q statementextract.RowScopedRequest) RowPilotRequestEvidence {
	return RowPilotRequestEvidence{q.Stage, q.RowOrdinal, q.Hash, q.PromptSHA, statementextract.PromptSHA(string(q.Schema)), q.InputBytes}
}

type RowPilotCall struct {
	Ordinal, AttemptOrdinal int
	Request                 RowPilotRequestEvidence
	StartedAt, FinishedAt   time.Time
	Proof                   statementextract.ComparisonIdentityProof
	Raw                     []byte `json:",omitempty"`
	RawHash                 string
	RawBytes                int
	Bill                    *statementextract.Billing `json:",omitempty"`
	Error                   string
}
type RowPilotAttempt struct {
	Ordinal                  int
	Variant, State, StopCode string
	Measurement              *RowPilotMeasurement `json:",omitempty"`
}
type RowPilotReport struct {
	Version, Authorization, Source, Digest, DocumentHash, PlanHash string
	Time, FinishedAt, ProofAt                                      time.Time
	Proof                                                          *statementextract.ComparisonIdentityProof `json:",omitempty"`
	ReservationID                                                  string
	MaximumMicros                                                  int64
	ActualMicros                                                   *int64
	SettlementRecorded                                             bool
	MetadataRequests                                               int
	Attempts                                                       []RowPilotAttempt
	Calls                                                          []RowPilotCall
	Decision, Error                                                string
}

func rowPilotDigest(r RowPilotReport) string { r.Digest = ""; return statementextract.Hash(r) }

// Local measurement accounting never touches the shared ledger. Campaign owns
// exactly one external reserve/settle; per-document meters only exercise bounds.
type rowPilotLocalMeter struct{}

func (rowPilotLocalMeter) Reserve(context.Context, int64, statementextract.Policy) (string, error) {
	return "row-local-measurement", nil
}
func (rowPilotLocalMeter) Settle(context.Context, string, *int64) error { return nil }
func RunRowPilot(ctx context.Context, s Sample, reasoner RowPilotReasoner, meter ClockMeter, source string) (report RowPilotReport, err error) {
	ctx, cancel := context.WithTimeout(ctx, 1800*time.Second)
	defer cancel()
	report = RowPilotReport{Version: RowPilotVersion, Authorization: RowPilotAuthorization, Source: source, Time: time.Now().UTC(), MaximumMicros: RowPilotMaxMicros, Calls: []RowPilotCall{}, Attempts: []RowPilotAttempt{}, Decision: ClockInconclusive}
	for i, v := range RowPilotOrder() {
		report.Attempts = append(report.Attempts, RowPilotAttempt{Ordinal: i + 1, Variant: v, State: "not_sent"})
	}
	fail := func(code string) error { report.Error = code; return errors.New(code) }
	defer func() {
		if recover() != nil {
			report.Error = "row_pilot_call_unavailable"
			err = errors.New(report.Error)
		}
		if reasoner != nil {
			report.MetadataRequests = reasoner.MetadataRequests()
		}
		report.FinishedAt = time.Now().UTC()
		report.Decision = rowPilotJudge(report.Attempts, report.Error)
		report.Digest = rowPilotDigest(report)
	}()
	if !regexp.MustCompile(`^[0-9a-f]{40}$`).MatchString(source) || s.Name != "cards" || s.SHA256 != FixtureHash("cards") || s.Document.Hash() != statementextract.ClockCardsDocumentHash || reasoner == nil || meter == nil {
		err = fail("row_pilot_context_invalid")
		return
	}
	report.DocumentHash = s.Document.Hash()
	base, e := statementextract.BuildRowPilotBase(s.Document)
	if e != nil {
		err = fail("row_pilot_context_invalid")
		return
	}
	plan, e := statementextract.BuildComparisonPlan(s.Document, base.Policy)
	if e != nil {
		err = fail("row_pilot_context_invalid")
		return
	}
	report.PlanHash = statementextract.Hash(plan)
	if ctx.Err() != nil {
		err = fail("row_pilot_timeout")
		return
	}
	p, e := reasoner.CurrentPolicy(ctx)
	if e != nil || p != base.Policy {
		err = fail("row_pilot_metadata_unavailable")
		return
	}
	proof, e := reasoner.ComparisonProof(ctx)
	report.ProofAt = time.Now().UTC()
	if e != nil || proof.Candidate != statementextract.ComparisonCandidates()[0] || !proof.ValidAt(report.ProofAt) {
		err = fail("row_pilot_metadata_unavailable")
		return
	}
	report.Proof = &proof
	if ctx.Err() != nil {
		err = fail("row_pilot_timeout")
		return
	}
	id, e := meter.Reserve(ctx, RowPilotMaxMicros, p)
	if e != nil || id == "" || len(id) > 128 {
		err = fail("row_pilot_budget_unavailable")
		return
	}
	report.ReservationID = id
	total := int64(0)
	known := true
	ids := map[string]bool{}
	defer func() {
		var actual *int64
		if known {
			n := total
			actual = &n
			report.ActualMicros = &n
		}
		settleCtx, stop := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer stop()
		if meter.Settle(settleCtx, id, actual) != nil {
			err = fail("row_pilot_settlement_failed")
		} else {
			report.SettlementRecorded = true
		}
	}()
	for i := range report.Attempts {
		a := &report.Attempts[i]
		a.State = "attempted"
		call := func(callCtx context.Context, q statementextract.RowScopedRequest) (raw []byte, bill *statementextract.Billing, callErr error) {
			stop := func(code string) ([]byte, *statementextract.Billing, error) {
				a.StopCode = code
				return nil, nil, fail(code)
			}
			if callCtx.Err() != nil {
				return stop("row_pilot_timeout")
			}
			allowed, e := meter.ComparisonAllowed(callCtx, id)
			if e != nil || !allowed {
				return stop("row_pilot_budget_unavailable")
			}
			proof, e := reasoner.ComparisonProof(callCtx)
			now := time.Now().UTC()
			if e != nil || proof.Candidate != statementextract.ComparisonCandidates()[0] || !proof.ValidAt(now) {
				return stop("row_pilot_metadata_unavailable")
			}
			if len(report.Calls) >= 12 {
				return stop("row_pilot_call_limit")
			}
			report.Calls = append(report.Calls, RowPilotCall{Ordinal: len(report.Calls) + 1, AttemptOrdinal: i + 1, Request: rowPilotEvidence(q), StartedAt: now, Proof: proof, Error: "row_pilot_call_unavailable"})
			rec := &report.Calls[len(report.Calls)-1]
			prefixKnown := known
			known = false
			defer func() { rec.FinishedAt = time.Now().UTC() }()
			raw, bill, e = reasoner.CompleteRowPilot(callCtx, q, proof)
			rec.RawBytes = len(raw)
			if len(raw) > 0 {
				rec.RawHash = statementextract.PromptSHA(string(raw))
			}
			if len(raw) <= statementextract.RowScopedResponseLimit {
				rec.Raw = append([]byte(nil), raw...)
			}
			if bill != nil {
				b, _ := json.Marshal(bill)
				_ = json.Unmarshal(b, &rec.Bill)
			}
			if !validBill(bill, p) || ids[bill.GenerationID] {
				rec.Error = "row_pilot_billing_unavailable"
				return nil, bill, fail(rec.Error)
			}
			known = prefixKnown
			ids[bill.GenerationID] = true
			total += bill.CostMicros
			if callCtx.Err() != nil {
				rec.Error = "row_pilot_timeout"
				return nil, bill, fail(rec.Error)
			}
			if bill.ComparisonRoute == nil || !bill.ComparisonRoute.Valid("A", rec.StartedAt, time.Now().UTC()) || !reflect.DeepEqual(bill.ComparisonRoute.Proof, &proof) {
				rec.Error = "row_pilot_route_unavailable"
				return nil, bill, fail(rec.Error)
			}
			if e != nil || bill.OutputTokens == 0 {
				rec.Error = "row_pilot_call_unavailable"
				return nil, bill, fail(rec.Error)
			}
			if len(raw) > statementextract.RowScopedResponseLimit {
				rec.Error = "row_pilot_capture_failed"
				return nil, bill, fail(rec.Error)
			}
			rec.Error = ""
			return raw, bill, nil
		}
		var m RowPilotMeasurement
		if a.Variant == "base" {
			callCtx, stop := context.WithTimeout(ctx, 120*time.Second)
			raw, _, e := call(callCtx, base)
			stop()
			if e != nil {
				m = rowPilotFailure("base", "call_unavailable")
			} else {
				m = rowPilotBaseMeasurement(s.Document, raw)
			}
		} else {
			out, e := runRowScoped(ctx, s, RowScopedStages{Grouping: call, Row: call}, rowPilotLocalMeter{})
			m = rowPilotMeasurement(out)
			if e != nil && report.Error == "" && !rowPilotSemantic(m) {
				if ctx.Err() != nil || out.Cause == "deadline_exceeded" {
					a.StopCode = "row_pilot_timeout"
					_ = fail(a.StopCode)
				} else {
					_ = fail("row_pilot_protocol_failed")
				}
			}
		}
		a.Measurement = &m
		if report.Error != "" {
			a.State = "unavailable"
			return report, errors.New(report.Error)
		}
		if m.Status != "complete" && !rowPilotSemantic(m) {
			a.State = "unavailable"
			err = fail("row_pilot_protocol_failed")
			return
		}
		if a.Variant == "candidate" && rowPilotDecisive(m) {
			a.State = "rejected"
			for j := i + 1; j < len(report.Attempts); j++ {
				report.Attempts[j].StopCode = "row_pilot_decisive_reject"
			}
			return report, nil
		}
		a.State = "completed"
	}
	return report, nil
}
