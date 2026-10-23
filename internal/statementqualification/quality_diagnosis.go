package statementqualification

import (
	"errors"
	"github.com/QahNaaRlN/grace/internal/statementdata"
	"github.com/QahNaaRlN/grace/internal/statementextract"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const QualityDiagnosisVersion = "statement.saved-quality-diagnosis.s118.v1"
const QualityArtifactSource = "24f73223b9b3370ca78c41d34e89edca9a276488"

// ScoringCheck is emitted by the actual scorer closure, without changing decisions.
type ScoringCheck struct {
	Ordinal, RowOrdinal int
	Code                string
	Passed              bool
}

func ScreeningChecks(name string, r statementdata.Review) (Result, []ScoringCheck) {
	checks := []ScoringCheck{}
	result := screeningWithTrace(name, r, func(c ScoringCheck) { checks = append(checks, c) })
	return result, checks
}

type QualityPosition struct {
	SourceOrdinal int    `json:"sourceOrdinal"`
	Page          int    `json:"page"`
	Block         int    `json:"block"`
	Line          int    `json:"line"`
	Word          int    `json:"word"`
	LexicalClass  string `json:"lexicalClass"`
}
type QualityReason struct {
	Ordinal    int               `json:"ordinal"`
	Scope      string            `json:"scope"`
	RowOrdinal int               `json:"rowOrdinal,omitempty"`
	Status     string            `json:"status"`
	Reason     string            `json:"reason,omitempty"`
	Code       string            `json:"code,omitempty"`
	Field      string            `json:"field,omitempty"`
	PublicCode string            `json:"publicCode,omitempty"`
	Relation   string            `json:"relation,omitempty"`
	Count      *int              `json:"count,omitempty"`
	Positions  []QualityPosition `json:"positions,omitempty"`
}
type QualityParts struct {
	Date      string `json:"date"`
	Time      string `json:"time"`
	Precision string `json:"precision"`
	UTC       string `json:"utc"`
}
type QualityMoneyParts struct {
	Amount   string `json:"amount"`
	Currency string `json:"currency"`
}
type QualityCheck struct {
	Ordinal    int                `json:"ordinal"`
	RowOrdinal int                `json:"rowOrdinal,omitempty"`
	Code       string             `json:"code"`
	Status     string             `json:"status"`
	Reason     string             `json:"reason,omitempty"`
	Positions  []QualityPosition  `json:"positions,omitempty"`
	Clock      *QualityParts      `json:"clock,omitempty"`
	Money      *QualityMoneyParts `json:"money,omitempty"`
	State      string             `json:"state,omitempty"`
	Links      []int              `json:"links,omitempty"`
}
type QualityCoverage struct {
	Status        string            `json:"status"`
	Reason        string            `json:"reason,omitempty"`
	TotalUnmapped int               `json:"totalUnmapped"`
	Positions     []QualityPosition `json:"positions,omitempty"`
}
type QualityTuple struct {
	Ordinal  int              `json:"ordinal"`
	Status   string           `json:"status"`
	Reason   string           `json:"reason,omitempty"`
	Checks   []QualityCheck   `json:"checks,omitempty"`
	Reasons  []QualityReason  `json:"reasons,omitempty"`
	Coverage *QualityCoverage `json:"coverage,omitempty"`
}
type QualityDiagnosis struct {
	Version                   string         `json:"version"`
	Authorization             string         `json:"authorization"`
	ArtifactSource            string         `json:"artifactSource"`
	VerifierSource            string         `json:"verifierSource"`
	ContextVerified           bool           `json:"contextVerified"`
	ArtifactsUnchanged        bool           `json:"artifactsUnchanged"`
	ProviderCalls             int            `json:"providerCalls"`
	PersistentWritesRequested int            `json:"persistentWritesRequested"`
	Tuples                    []QualityTuple `json:"tuples"`
}

// DiagnoseSavedQuality is pure. Its inputs are the immutable E6 envelope and pinned public layouts.
func DiagnoseSavedQuality(e ObservedComparisonReport, samples []Sample, verifier string, now time.Time) (QualityDiagnosis, error) {
	fail := errors.New("diagnostic_unavailable")
	if !qualitySourceSHA.MatchString(verifier) || ValidateS117Comparison(e, now) != nil || len(samples) != 3 || e.Report.Passed || e.Report.Error != "qualification_quality_failed" {
		return QualityDiagnosis{}, fail
	}
	plans := make([]statementextract.Plan, 6)
	for i, t := range e.Report.Tuples {
		s := samples[i/2]
		if s.Name != ScreeningNames()[i/2] || s.SHA256 != FixtureHash(s.Name) || s.Document.Extractor != statementdata.ExtractorVersion || s.Document.Validate() != nil || t.State != "completed" || t.Reason != "quality_failed" || t.Result == nil {
			return QualityDiagnosis{}, fail
		}
		p, err := statementextract.BuildComparisonPlan(s.Document, e.Report.Candidates[i%2].Policy)
		if err != nil || len(p.Chunks) != 1 {
			return QualityDiagnosis{}, fail
		}
		request := struct {
			Policy   statementextract.Policy
			Messages []map[string]string
			Schema   any
		}{p.Policy, statementextract.Messages(p.Chunks[0]), statementextract.RequestSchema(p.Chunks[0])}
		if t.Evidence.DocumentHash != s.Document.Hash() || t.Evidence.PlanHash != statementextract.Hash(p) || t.RequestHash != statementextract.Hash(request) || t.Evidence.InputBytes != statementextract.InputBytes(p.Chunks[0]) {
			return QualityDiagnosis{}, fail
		}
		plans[i] = p
	}
	out := QualityDiagnosis{Version: QualityDiagnosisVersion, Authorization: S117ComparisonAuthorization, ArtifactSource: QualityArtifactSource, VerifierSource: verifier, ContextVerified: true, Tuples: make([]QualityTuple, 6)}
	for i, t := range e.Report.Tuples {
		row := QualityTuple{Ordinal: i + 1, Status: "UNVERIFIED", Reason: "snapshot_unavailable"}
		d, p := samples[i/2].Document, plans[i]
		if e.Observations[i].Snapshot == nil {
			out.Tuples[i] = row
			continue
		}
		if !ReplayComparisonObservation(&e, i, d, now).Verified {
			row.Reason = "replay_mismatch"
			out.Tuples[i] = row
			continue
		}
		canonical, _, err := statementextract.CanonicalComparisonProposalDiagnostic(p, *e.Observations[i].Snapshot)
		if err != nil || !reflect.DeepEqual(canonical, *t.Proposal) {
			row.Reason = "replay_mismatch"
			out.Tuples[i] = row
			continue
		}
		proposal, err := statementextract.MergeProposals(p, []statementdata.Proposal{canonical})
		if err != nil {
			row.Reason = "replay_mismatch"
			out.Tuples[i] = row
			continue
		}
		pages := []int{}
		for _, pg := range d.Pages {
			pages = append(pages, pg.Number)
		}
		review, err := statementdata.ValidateProposal(d, proposal, pages, true, "")
		if err != nil {
			row.Reason = "replay_mismatch"
			out.Tuples[i] = row
			continue
		}
		scored, checks := ScreeningChecks(t.Evidence.Name, review)
		scored.LatencyMS = t.Call.LatencyMS
		if !reflect.DeepEqual(scored, *t.Result) || t.FalseAcceptance != (!scored.Passed && len(scored.Validation) == 0) {
			row.Reason = "replay_mismatch"
			out.Tuples[i] = row
			continue
		}
		row.Status = "VERIFIED"
		row.Reason = ""
		lookup := qualityLookup(d, p)
		if i < 4 {
			row.Coverage = qualityCoverage(d, proposal, review, lookup)
		}
		row.Reasons = qualityReasons(review, proposal, lookup, len(statementdata.NewMappingSnapshot(d, proposal, review, "model", 1).UnmappedSpanIDs))
		for _, c := range checks {
			if !c.Passed {
				row.Checks = append(row.Checks, qualityCheck(c, t.Evidence.Name, proposal, review, row.Reasons, lookup))
			}
		}
		if len(row.Reasons) > 128 {
			row = QualityTuple{Ordinal: i + 1, Status: "UNVERIFIED", Reason: "diagnosis_limit_exceeded"}
		}
		out.Tuples[i] = row
	}
	return out, nil
}

type qualityPositions map[string]QualityPosition

func qualityLookup(d statementdata.Document, p statementextract.Plan) qualityPositions {
	classes := map[string]string{}
	for _, s := range p.Chunks[0].Spans {
		cl := "other"
		if s.LexicalClass == statementextract.ClassNumericFragment {
			cl = "numeric"
		}
		classes[p.Aliases[s.ID]] = cl
	}
	out := qualityPositions{}
	for n, s := range d.Spans {
		out[s.ID] = QualityPosition{n + 1, s.Page, s.Block, s.Line, s.Word, classes[s.ID]}
	}
	return out
}
func (lookup qualityPositions) selected(ids statementdata.Selection) ([]QualityPosition, bool) {
	selected := map[string]bool{}
	for _, id := range ids {
		if _, ok := lookup[id]; !ok {
			return nil, false
		}
		selected[id] = true
	}
	if len(selected) > 128 {
		return nil, false
	}
	out := []QualityPosition{}
	// Source order is deterministic even when the selection crosses fields.
	for id := range selected {
		out = append(out, lookup[id])
	}
	// Never expose IDs or selection order as raw alias payload.
	sortQualityPositions(out)
	return out, true
}
func qualityCoverage(d statementdata.Document, p statementdata.Proposal, r statementdata.Review, l qualityPositions) *QualityCoverage {
	ids := statementdata.NewMappingSnapshot(d, p, r, "model", 1).UnmappedSpanIDs
	out := &QualityCoverage{Status: "VERIFIED", TotalUnmapped: len(ids)}
	if len(ids) > 128 {
		out.Status = "UNVERIFIED"
		out.Reason = "diagnosis_limit_exceeded"
		return out
	}
	var ok bool
	out.Positions, ok = l.selected(ids)
	if !ok || len(out.Positions) != len(ids) {
		out.Status = "UNVERIFIED"
		out.Reason = "replay_mismatch"
		out.Positions = nil
	}
	return out
}

var qualityRowCode = regexp.MustCompile(`^row_([1-9][0-9]*):([a-z_]+)$`)
var qualityCountCode = regexp.MustCompile(`^unmapped_spans:(0|[1-9][0-9]*)$`)

const qualityReasonVocabulary = "missing_evidence invalid_span_reference noncontiguous_evidence noncontiguous_date_evidence field_evidence_limit overlapping_evidence operation_date_required invalid_date invalid_timezone ambiguous_local_time account_amount_required currency_required currency_conflict invalid_currency ambiguous_money amount_overflow unsupported_account_currency zero_posting invalid_direction direction_conflict direction_required transaction_direction_conflict description_evidence_limit financial_candidate_in_description description_evidence_gap description_evidence_unverified field_label_required available_is_not_book_balance negative_turnover turnover_role_unverified book_balance_role_unverified aggregate_overflow invalid_source_rows printed_turnover_mismatch book_balance_mismatch book_balance_period_not_proven source_coverage_incomplete unmapped_spans invalid_page_review owner_page_review_required owner_field_role_review_required service_reason_required"

func wordIn(s, list string) bool {
	for _, v := range strings.Fields(list) {
		if s == v {
			return true
		}
	}
	return false
}
func qualityCode(raw string, rows int) (code, field string, row int, count *int, ok bool) {
	code = raw
	if m := qualityRowCode.FindStringSubmatch(raw); m != nil {
		n, err := strconv.Atoi(m[1])
		if err != nil || n < 1 || n > rows {
			return "", "", 0, nil, false
		}
		row = n
		code = m[2]
	} else if m := qualityCountCode.FindStringSubmatch(raw); m != nil {
		n, err := strconv.Atoi(m[1])
		if err != nil || n > 2000 {
			return "", "", 0, nil, false
		}
		code = "unmapped_spans"
		count = &n
	} else if strings.Contains(raw, ":") {
		parts := strings.Split(raw, ":")
		if len(parts) != 2 || !wordIn(parts[0], "openingBookDate closingBookDate openingBookBalance closingBookBalance availableBalanceObserved debitTurnover creditTurnover service") {
			return "", "", 0, nil, false
		}
		field, code = parts[0], parts[1]
	}
	ok = wordIn(code, qualityReasonVocabulary)
	return
}

// This is the historical public projection vocabulary, not a new sanitizer for E6.
func qualityPublicCode(raw string) string { return ScreeningPublicCodes([]string{raw})[0] }
func qualityReasons(r statementdata.Review, p statementdata.Proposal, l qualityPositions, unmappedTotal int) []QualityReason {
	out := []QualityReason{}
	add := func(scope string, row int, raw string, index int, exposed bool) {
		c, f, n, count, ok := qualityCode(raw, len(r.Rows))
		if row != 0 {
			n = row
		}
		reason := QualityReason{Ordinal: index, Scope: scope, RowOrdinal: n, Status: "UNVERIFIED", Reason: "unsupported_reason_code"}
		if !ok {
			reason.RowOrdinal = row
			out = append(out, reason)
			return
		}
		if count != nil && *count != unmappedTotal {
			reason.Reason = "replay_mismatch"
			out = append(out, reason)
			return
		}
		reason.Status = "VERIFIED"
		reason.Reason = ""
		reason.Code = c
		reason.Field = f
		reason.Count = count
		if exposed {
			reason.PublicCode = qualityPublicCode(raw)
			reason.Relation = "mapped"
		} else {
			reason.Relation = "not_exposed"
		}
		var ids statementdata.Selection
		if n > 0 && n <= len(p.Rows) {
			ids = rowSelections(p.Rows[n-1])
		} else if f != "" {
			ids = headerSelections(p.Header, f)
		}
		var good bool
		reason.Positions, good = l.selected(ids)
		if !good {
			reason = QualityReason{Ordinal: index, Scope: scope, RowOrdinal: row, Status: "UNVERIFIED", Reason: "diagnosis_limit_exceeded"}
		}
		out = append(out, reason)
	}
	for i, row := range r.Rows {
		for j, c := range row.Issues {
			add("row", i+1, c, j+1, false)
		}
	}
	for _, s := range []struct {
		name   string
		codes  []string
		public bool
	}{{"blocker", r.Reconciliation.Blockers, true}, {"coverage", r.Reconciliation.RowCoverage.Reasons, true}, {"turnover", r.Reconciliation.Turnover.Reasons, false}, {"book_balance", r.Reconciliation.BookBalance.Reasons, false}} {
		for j, c := range s.codes {
			add(s.name, 0, c, j+1, s.public)
		}
	}
	return out
}
func rowSelections(r statementdata.RowMapping) statementdata.Selection {
	out := statementdata.Selection{}
	for _, ids := range []statementdata.Selection{r.OperationDate, r.BookingDate, r.AccountAmount, r.AccountCurrency, r.TransactionAmount, r.TransactionCurrency, r.Description, r.DirectionEvidence} {
		out = append(out, ids...)
	}
	return out
}
func headerSelections(h statementdata.HeaderMapping, field string) statementdata.Selection {
	var ids statementdata.Selection
	switch field {
	case "openingBookDate":
		ids = h.OpeningBookDate
	case "closingBookDate":
		ids = h.ClosingBookDate
	case "openingBookBalance":
		ids = h.OpeningBookBalance
	case "closingBookBalance":
		ids = h.ClosingBookBalance
	case "availableBalanceObserved":
		ids = h.AvailableBalanceObserved
	case "debitTurnover":
		ids = h.DebitTurnover
	case "creditTurnover":
		ids = h.CreditTurnover
	}
	out := append(statementdata.Selection{}, ids...)
	out = append(out, h.Labels[field]...)
	return out
}
