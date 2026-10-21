package statementqualification

import (
	"errors"
	"github.com/QahNaaRlN/grace/internal/statementdata"
)

// ValidateSavedDiagnosis is a closed numeric/enum boundary, including error cases.
func ValidateSavedDiagnosis(d SavedDiagnosis) error {
	fail := errors.New("diagnostic_unavailable")
	if d.Version != SavedDiagnosisVersion || d.Authorization != S114ComparisonAuthorization || d.Source != SavedDiagnosisSource || !d.ContextVerified || !d.ArtifactsUnchanged || d.ProviderCalls != 0 || d.PersistentWritesRequested != 0 || len(d.Tuples) != 6 {
		return fail
	}
	pos := func(p DiagnosisPosition) bool {
		return p.SourceOrdinal > 0 && p.SourceOrdinal <= 2000 && p.Page > 0 && p.Page <= statementdata.MaxPages && p.Block > 0 && p.Line > 0 && p.Word > 0 && p.Block <= 1000000 && p.Line <= 1000000 && p.Word <= 1000000
	}
	for i, r := range d.Tuples {
		if r.Ordinal != i+1 || r.Kind != []string{"description_gap", "unmapped", "alias_relation", "not_applicable", "not_applicable", "not_applicable"}[i] {
			return fail
		}
		if i > 2 {
			if r.Status != "not_applicable" || r.Reason != "not_sent" || r.Relation != "" || r.Gap != nil || r.TotalUnmapped != nil || len(r.Witnesses) != 0 {
				return fail
			}
			continue
		}
		if r.Status == "UNVERIFIED" {
			switch r.Reason {
			case "snapshot_unavailable", "context_mismatch", "guard_mismatch", "replay_mismatch", "diagnosis_limit_exceeded", "unavailable_runtime":
			default:
				return fail
			}
			if r.Gap != nil || r.Relation != "" || len(r.Witnesses) != 0 || r.TotalUnmapped != nil && (i != 1 || *r.TotalUnmapped < 0 || *r.TotalUnmapped > 2000) {
				return fail
			}
			continue
		}
		if r.Status != "VERIFIED" || r.Reason != "" {
			return fail
		}
		switch i {
		case 0:
			g := r.Gap
			if g == nil || r.Relation != "" || r.TotalUnmapped != nil || len(r.Witnesses) > 0 || g.RowOrdinal != 2 || !pos(g.Left) || !pos(g.Right) || g.Left.SourceOrdinal >= g.Right.SourceOrdinal || g.Left.Page != g.Right.Page || g.Left.Block != g.Right.Block || g.Left.Line != g.Right.Line || g.Right.Word <= g.Left.Word+1 || g.IntermediateUnselected < 1 || g.IntermediateUnselected > g.Right.SourceOrdinal-g.Left.SourceOrdinal-1 {
				return fail
			}
		case 1:
			if r.Gap != nil || r.Relation != "" || r.TotalUnmapped == nil || *r.TotalUnmapped < 0 || *r.TotalUnmapped > 128 || len(r.Witnesses) != *r.TotalUnmapped {
				return fail
			}
			previous := 0
			for _, w := range r.Witnesses {
				if !pos(w.DiagnosisPosition) || w.SourceOrdinal <= previous || w.LexicalClass != "numeric" && w.LexicalClass != "other" {
					return fail
				}
				previous = w.SourceOrdinal
			}
		case 2:
			if r.Gap != nil || r.TotalUnmapped != nil || len(r.Witnesses) > 0 {
				return fail
			}
			switch r.Relation {
			case "original_span_id", "canonical_alias_syntax_absent", "outside_alias_syntax":
			default:
				return fail
			}
		}
	}
	return nil
}
