package statementqualification

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/QahNaaRlN/grace/internal/statementdata"
	"github.com/QahNaaRlN/grace/internal/statementextract"
	"strings"
	"testing"
	"time"
)

type savedFake struct {
	comparisonFake
	item string
}

func (f *savedFake) Complete(ctx context.Context, p statementextract.Policy, c statementextract.Chunk) (statementdata.Proposal, *statementextract.Billing, error) {
	part, bill, err := f.comparisonFake.Complete(ctx, p, c)
	part.Rows = []statementdata.RowMapping{{}}
	if f.candidate.ID == "A" && f.calls == 2 {
		part.Rows = []statementdata.RowMapping{{OperationDate: statementdata.Selection{f.item}, Description: statementdata.Selection{"PRIVATE-SECOND-DEFECT"}}}
	}
	return part, bill, err
}
func TestSavedIndependentAliasAndContext(t *testing.T) {
	samples := observedSamples()
	for _, item := range []string{samples[1].Document.Spans[0].ID, "szzzz", "PRIVATE-ALIAS", "s00", "sA"} {
		t.Run(item, func(t *testing.T) {
			cs := statementextract.ComparisonCandidates()
			a := &savedFake{comparisonFake: comparisonFake{candidate: cs[0]}, item: item}
			b := &savedFake{comparisonFake: comparisonFake{candidate: cs[1]}}
			e, err := RunS114Comparison(context.Background(), samples, [2]statementextract.Reasoner{a, b}, &fakeMeter{})
			if err == nil || a.calls != 2 || b.calls != 1 {
				t.Fatal("causal stop")
			}
			var encoded bytes.Buffer
			if EncodeS114Comparison(&encoded, e) != nil {
				t.Fatal("writer")
			}
			_, decoded, err := DecodeS114ComparisonArtifact(&encoded)
			if err != nil {
				t.Fatal(err)
			}
			before := statementextract.Hash(e)
			out, err := DiagnoseSavedProposals(*decoded, samples, time.Now())
			if err != nil {
				t.Fatal(err)
			}
			if out.Tuples[2].Status != "VERIFIED" || out.Tuples[0].Status != "UNVERIFIED" || out.Tuples[1].Status != "UNVERIFIED" {
				t.Fatal("partial results lost", out)
			}
			expected := "outside_alias_syntax"
			if item == samples[1].Document.Spans[0].ID {
				expected = "original_span_id"
			}
			if item == "szzzz" {
				expected = "canonical_alias_syntax_absent"
			}
			if out.Tuples[2].Relation != expected || before != statementextract.Hash(e) {
				t.Fatal("relation/mutation")
			}
			out.ArtifactsUnchanged = true
			if ValidateSavedDiagnosis(out) != nil {
				t.Fatal("closed projection")
			}
			raw, _ := json.Marshal(out)
			if bytes.Contains(raw, []byte("PRIVATE")) || bytes.Contains(raw, []byte(item)) && strings.HasPrefix(item, "szz") {
				t.Fatal("private output")
			}
			// A mismatched earlier replay cannot hide the independently verified alias.
			decoded.Observations[0].Snapshot.Header.Currency = statementdata.Selection{"PRIVATE-REPLAY"}
			decoded.Digest = observedDigest(*decoded)
			partial, err := DiagnoseSavedProposals(*decoded, samples, time.Now())
			if err != nil || partial.Tuples[0].Reason != "replay_mismatch" || partial.Tuples[2].Status != "VERIFIED" {
				t.Fatal("independence")
			}
			decoded.Report.Tuples[0].RequestHash = strings.Repeat("f", 64)
			decoded.Report.Digest = comparisonDigest(decoded.Report)
			decoded.Digest = observedDigest(*decoded)
			if _, err := DiagnoseSavedProposals(*decoded, samples, time.Now()); err == nil {
				t.Fatal("context mismatch")
			}
		})
	}
}
func TestSavedUnmappedLimitsAndMembership(t *testing.T) {
	for _, n := range []int{128, 129} {
		d := observedSamples()[0].Document
		d.Spans = nil
		for i := 0; i < n; i++ {
			d.Spans = append(d.Spans, statementdata.Span{ID: fmt.Sprintf("PRIVATE-%d", i), Page: 1, Block: 1, Line: i + 1, Word: 1, Text: "RUB", Box: statementdata.Box{XMin: 1, YMin: float64(i + 1), XMax: 20, YMax: float64(i + 2)}})
		}
		p, err := statementextract.BuildComparisonPlan(d, statementextract.ComparisonCandidates()[0].Policy)
		if err != nil {
			t.Fatal(err)
		}
		proposal := statementdata.Proposal{Rows: []statementdata.RowMapping{{}}}
		review, err := statementdata.ValidateProposal(d, proposal, []int{1}, true, "")
		if err != nil {
			t.Fatal(err)
		}
		row := unmappedDiagnosis(SavedIssue{Ordinal: 2, Kind: "unmapped", Status: "UNVERIFIED"}, d, p, proposal, review)
		if row.TotalUnmapped == nil || *row.TotalUnmapped != n {
			t.Fatal("actual membership")
		}
		if n == 128 && (row.Status != "VERIFIED" || len(row.Witnesses) != n) || n == 129 && (row.Status != "UNVERIFIED" || row.Reason != "diagnosis_limit_exceeded" || len(row.Witnesses) != 0) {
			t.Fatal("limit", row)
		}
		// Actual service membership removes exactly one source span; no golden selectors.
		proposal.Service = []statementdata.ServiceMapping{{Spans: statementdata.Selection{d.Spans[0].ID}, Reason: "PRIVATE-service"}}
		review, err = statementdata.ValidateProposal(d, proposal, []int{1}, true, "")
		if err != nil {
			t.Fatal(err)
		}
		row = unmappedDiagnosis(SavedIssue{Status: "UNVERIFIED"}, d, p, proposal, review)
		if *row.TotalUnmapped != n-1 || row.Witnesses[0].SourceOrdinal != 2 {
			t.Fatal("service membership ignored")
		}
	}
}
