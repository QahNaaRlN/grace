package statementqualification

import (
	"errors"
	"reflect"
	"regexp"
	"time"

	"github.com/QahNaaRlN/grace/internal/statementdata"
	"github.com/QahNaaRlN/grace/internal/statementextract"
)

const SavedDiagnosisVersion = "statement.saved-proposal-diagnosis.v1"
const SavedDiagnosisSource = "301afd6bedc4a19559e731471336bbc4e2f74075"

type DiagnosisPosition struct {
	SourceOrdinal int `json:"sourceOrdinal"`
	Page          int `json:"page"`
	Block         int `json:"block"`
	Line          int `json:"line"`
	Word          int `json:"word"`
}
type GapDiagnosis struct {
	RowOrdinal             int               `json:"rowOrdinal"`
	Left                   DiagnosisPosition `json:"left"`
	Right                  DiagnosisPosition `json:"right"`
	IntermediateUnselected int               `json:"intermediateUnselected"`
}
type UnmappedWitness struct {
	DiagnosisPosition
	LexicalClass string `json:"lexicalClass"`
}
type SavedIssue struct {
	Ordinal       int               `json:"ordinal"`
	Kind          string            `json:"kind"`
	Status        string            `json:"status"`
	Reason        string            `json:"reason,omitempty"`
	Relation      string            `json:"relation,omitempty"`
	Gap           *GapDiagnosis     `json:"gap,omitempty"`
	TotalUnmapped *int              `json:"totalUnmapped,omitempty"`
	Witnesses     []UnmappedWitness `json:"witnesses,omitempty"`
}
type SavedDiagnosis struct {
	Version                   string       `json:"version"`
	Authorization             string       `json:"authorization"`
	Source                    string       `json:"source"`
	ContextVerified           bool         `json:"contextVerified"`
	ArtifactsUnchanged        bool         `json:"artifactsUnchanged"`
	ProviderCalls             int          `json:"providerCalls"`
	PersistentWritesRequested int          `json:"persistentWritesRequested"`
	Tuples                    []SavedIssue `json:"tuples"`
}

func position(d statementdata.Document, n int) DiagnosisPosition {
	s := d.Spans[n-1]
	return DiagnosisPosition{n, s.Page, s.Block, s.Line, s.Word}
}

var absentAliasSyntax = regexp.MustCompile(`^s(?:0|[1-9a-z][0-9a-z]*)$`)

func aliasRelation(d statementdata.Document, p statementextract.Plan, item string) string {
	if _, ok := p.Aliases[item]; ok {
		return ""
	}
	for _, s := range d.Spans {
		if s.ID == item {
			return "original_span_id"
		}
	}
	if absentAliasSyntax.MatchString(item) {
		return "canonical_alias_syntax_absent"
	}
	return "outside_alias_syntax"
}

// DiagnoseSavedProposals is pure: it has no provider, meter, filesystem or SQL dependency.
// Integrity failure produces no context or witnesses. Issue failures remain independent.
func DiagnoseSavedProposals(e ObservedComparisonReport, samples []Sample, now time.Time) (SavedDiagnosis, error) {
	fail := errors.New("diagnostic_unavailable")
	if ValidateS114Comparison(e, now) != nil || len(samples) != 3 || e.Report.Error != "qualification_protocol_failed" || e.Report.Passed {
		return SavedDiagnosis{}, fail
	}
	plans := make([]statementextract.Plan, 6)
	for i, t := range e.Report.Tuples {
		sample := samples[i/2]
		if sample.Name != ScreeningNames()[i/2] || sample.SHA256 != FixtureHash(sample.Name) || sample.Document.Extractor != statementdata.ExtractorVersion || sample.Document.Validate() != nil {
			return SavedDiagnosis{}, fail
		}
		p, err := statementextract.BuildComparisonPlan(sample.Document, e.Report.Candidates[i%2].Policy)
		if err != nil || len(p.Chunks) != 1 {
			return SavedDiagnosis{}, fail
		}
		request := struct {
			Policy   statementextract.Policy
			Messages []map[string]string
			Schema   any
		}{p.Policy, statementextract.Messages(p.Chunks[0]), statementextract.RequestSchema(p.Chunks[0])}
		if t.Evidence.DocumentHash != sample.Document.Hash() || t.Evidence.PlanHash != statementextract.Hash(p) || t.Evidence.InputBytes != statementextract.InputBytes(p.Chunks[0]) || t.RequestHash != statementextract.Hash(request) {
			return SavedDiagnosis{}, fail
		}
		plans[i] = p
		if i < 2 && (t.State != "completed" || t.Reason != "quality_failed" || t.Result == nil) || i == 2 && (t.State != "attempted" || t.Reason != "protocol_failed") || i > 2 && t.State != "not_sent" {
			return SavedDiagnosis{}, fail
		}
	}
	out := SavedDiagnosis{Version: SavedDiagnosisVersion, Authorization: S114ComparisonAuthorization, Source: SavedDiagnosisSource, ContextVerified: true, Tuples: make([]SavedIssue, 6)}
	for i := range out.Tuples {
		row := SavedIssue{Ordinal: i + 1, Kind: []string{"description_gap", "unmapped", "alias_relation", "not_applicable", "not_applicable", "not_applicable"}[i], Status: "UNVERIFIED", Reason: "snapshot_unavailable"}
		if i > 2 {
			row.Status = "not_applicable"
			row.Reason = "not_sent"
			out.Tuples[i] = row
			continue
		}
		o, t, d, p := e.Observations[i], e.Report.Tuples[i], samples[i/2].Document, plans[i]
		if o.Snapshot == nil {
			out.Tuples[i] = row
			continue
		}
		row.Reason = "replay_mismatch"
		if !ReplayComparisonObservation(&e, i, d, now).Verified {
			out.Tuples[i] = row
			continue
		}
		row.Reason = "guard_mismatch"
		if i == 2 {
			diag := o.Diagnostic
			if diag != nil && diag.Code == "unknown_alias" && diag.Scope == "row" && diag.Field == "operationDate" && diag.Ordinal == 1 && diag.SelectionIndex == 1 && diag.Role == "" && len(o.Snapshot.Rows) > 0 && len(o.Snapshot.Rows[0].OperationDate) > 0 {
				row.Relation = aliasRelation(d, p, o.Snapshot.Rows[0].OperationDate[0])
				if row.Relation != "" {
					row.Status = "VERIFIED"
					row.Reason = ""
				}
			}
			out.Tuples[i] = row
			continue
		}
		proposal, err := statementextract.MergeProposals(p, []statementdata.Proposal{*t.Proposal})
		pages := []int{}
		for _, pg := range d.Pages {
			pages = append(pages, pg.Number)
		}
		if err != nil {
			row.Reason = "replay_mismatch"
			out.Tuples[i] = row
			continue
		}
		review, err := statementdata.ValidateProposal(d, proposal, pages, true, "")
		if err != nil {
			row.Reason = "replay_mismatch"
			out.Tuples[i] = row
			continue
		}
		scored := ScoreScreening(t.Evidence.Name, review)
		scored.LatencyMS = t.Call.LatencyMS
		if !reflect.DeepEqual(scored, *t.Result) || t.FalseAcceptance != (!scored.Passed && len(scored.Validation) == 0) || scored.Checks != 19 || scored.Correct != []int{16, 17}[i] {
			row.Reason = "replay_mismatch"
			out.Tuples[i] = row
			continue
		}
		if i == 0 {
			if len(review.Rows) > 1 {
				left, right, ok := statementdata.DescriptionGapPositions(d, review.Rows[1], 1)
				if ok && right > left {
					selected := map[string]bool{}
					for _, id := range proposal.Rows[1].Description {
						selected[id] = true
					}
					count := 0
					for n := left + 1; n < right; n++ {
						if !selected[d.Spans[n-1].ID] {
							count++
						}
					}
					row.Gap = &GapDiagnosis{2, position(d, left), position(d, right), count}
					row.Status = "VERIFIED"
					row.Reason = ""
				}
			}
		} else {
			row = unmappedDiagnosis(row, d, p, proposal, review)
		}
		out.Tuples[i] = row
	}
	return out, nil
}
func unmappedDiagnosis(row SavedIssue, d statementdata.Document, p statementextract.Plan, proposal statementdata.Proposal, review statementdata.Review) SavedIssue {
	coverageBlocked := false
	for _, code := range review.Reconciliation.Blockers {
		if code == "source_coverage_incomplete" {
			coverageBlocked = true
		}
	}
	if !coverageBlocked {
		row.Reason = "guard_mismatch"
		return row
	}
	ids := statementdata.NewMappingSnapshot(d, proposal, review, "model", 1).UnmappedSpanIDs
	total := len(ids)
	row.TotalUnmapped = &total
	if total > 128 {
		row.Reason = "diagnosis_limit_exceeded"
		return row
	}
	members := map[string]bool{}
	for _, id := range ids {
		if members[id] {
			row.Reason = "replay_mismatch"
			return row
		}
		members[id] = true
	}
	classes := map[string]string{}
	for _, s := range p.Chunks[0].Spans {
		id, ok := p.Aliases[s.ID]
		if !ok {
			row.Reason = "replay_mismatch"
			return row
		}
		classes[id] = s.LexicalClass
	}
	witnesses := []UnmappedWitness{}
	for n, s := range d.Spans {
		if members[s.ID] {
			class := "other"
			if classes[s.ID] == statementextract.ClassNumericFragment {
				class = "numeric"
			} else if classes[s.ID] != statementextract.ClassOther {
				row.Reason = "replay_mismatch"
				return row
			}
			witnesses = append(witnesses, UnmappedWitness{position(d, n+1), class})
		}
	}
	if len(witnesses) != total {
		row.Reason = "replay_mismatch"
		return row
	}
	row.Witnesses = witnesses
	row.Status = "VERIFIED"
	row.Reason = ""
	return row
}
