package statementdata

import (
	"encoding/json"
	"reflect"
	"testing"
)

func mappingFixture() (Document, Proposal) {
	d := syntheticDocument("19.10.2026", "-2456.78", "RUB", "Synthetic merchant", "Debit turnover", "2", "456.78", "Synthetic service", "Credit turnover", "0.00")
	d.Spans[6].Block = d.Spans[5].Block
	d.Spans[6].Word = 2
	d.Spans[6].Box = Box{101, 75, 160, 85}
	return d, Proposal{Rows: []RowMapping{{OperationDate: Selection{"s0"}, AccountAmount: Selection{"s1"}, AccountCurrency: Selection{"s2"}, Description: Selection{"s3"}}}, Header: HeaderMapping{DebitTurnover: Selection{"s5", "s6"}, CreditTurnover: Selection{"s9"}, Currency: Selection{"s2"}, Labels: map[string]Selection{"debitTurnover": {"s4"}, "creditTurnover": {"s8"}}}, Service: []ServiceMapping{{Selection{"s7"}, "Synthetic service reason"}}}
}
func TestMappingSplitTotalsAndRoundTrip(t *testing.T) {
	d, p := mappingFixture()
	full, e := ValidateProposal(d, p, []int{1}, true, "")
	if e != nil || full.Reconciliation.Turnover.Status != "matched" {
		t.Fatal(full, e)
	}
	p.Header.DebitTurnover = Selection{"s6"}
	partial, e := ValidateProposal(d, p, []int{1}, true, "")
	if e != nil || partial.Reconciliation.Turnover.Status != "mismatch" {
		t.Fatal(partial, e)
	}
	snap := NewMappingSnapshot(d, p, partial, "model", 2)
	raw, _ := json.Marshal(snap)
	mapped := MappingFor(&d, &partial, raw, 8)
	if mapped.Status != "complete" || mapped.Revision != 8 || !reflect.DeepEqual(mapped.UnmappedSpanIDs, []string{"s5"}) || mapped.Proposal.Service[0].Reason != "Synthetic service reason" || snap.Origin != "model" {
		t.Fatal(mapped, snap)
	}
	if !reflect.DeepEqual(mapped.Proposal.Header.DebitTurnover, p.Header.DebitTurnover) {
		t.Fatal("selection was silently repaired")
	}
}
func TestMappingLegacyDoesNotGuessService(t *testing.T) {
	d, p := mappingFixture()
	r, e := ValidateProposal(d, p, []int{1}, true, "")
	if e != nil {
		t.Fatal(e)
	}
	// Use serialized evidence: the private transient coverage must not be required.
	raw, _ := json.Marshal(r)
	_ = json.Unmarshal(raw, &r)
	m := MappingFor(&d, &r, nil, 3)
	if m.Status != "reconstructed_partial" || m.UnmappedSpanIDs != nil || !reflect.DeepEqual(m.UnclassifiedSpanIDs, []string{"s7"}) || len(m.Proposal.Service) != 0 {
		t.Fatal(m)
	}
	if !reflect.DeepEqual(m.Proposal.Header.DebitTurnover, p.Header.DebitTurnover) || !reflect.DeepEqual(m.Proposal.Rows[0].AccountCurrency, p.Rows[0].AccountCurrency) || m.Proposal.Rows[0].Direction != "debit" {
		t.Fatal(m.Proposal)
	}
	r.Rows = append(r.Rows, r.Rows[0])
	m = MappingFor(&d, &r, nil, 3)
	if len(m.Proposal.Rows) != 2 {
		t.Fatal("lost a legacy row")
	}
	r.Header.Evidence = append(r.Header.Evidence, Evidence{Role: "headerCurrency", Spans: []Span{d.Spans[3]}})
	m = MappingFor(&d, &r, nil, 3)
	if len(m.Proposal.Header.Currency) != 0 || !contains(m.Issues, "legacy_field_evidence_invalid") {
		t.Fatal("conflicting evidence inferred", m)
	}
}
func TestMappingUnavailableAndTampering(t *testing.T) {
	d, p := mappingFixture()
	r, _ := ValidateProposal(d, p, nil, false, "")
	snap := NewMappingSnapshot(d, p, r, "owner", 2)
	raw, _ := json.Marshal(snap)
	if MappingFor(nil, &r, raw, 2).Status != "unavailable" {
		t.Fatal("deleted original mapping exposed")
	}
	if MappingFor(&d, nil, nil, 1) != nil {
		t.Fatal("blank source reconstructed")
	}
	for _, change := range []func(*MappingSnapshot){func(s *MappingSnapshot) { s.LayoutHash = "wrong" }, func(s *MappingSnapshot) { s.Version = "old" },
		func(s *MappingSnapshot) { s.Origin = "unrecognized" }, func(s *MappingSnapshot) { s.UnmappedSpanIDs = []string{"s0"} }, func(s *MappingSnapshot) { s.DraftRevision = 9 }} {
		bad := snap
		change(&bad)
		b, _ := json.Marshal(bad)
		if MappingFor(&d, &r, b, 2).Status != "unavailable" {
			t.Fatal("corrupt snapshot used")
		}
	}
	changed := r
	changed.Rows = append([]Row{}, r.Rows...)
	changed.Rows[0].Description = "Changed after snapshot"
	if MappingFor(&d, &changed, raw, 3).Status != "unavailable" {
		t.Fatal("stale snapshot hid changed review")
	}
	r.Rows[0].Evidence[0].Spans[0].Text = "modified"
	m := MappingFor(&d, &r, nil, 3)
	if len(m.Proposal.Rows[0].OperationDate) != 0 || !contains(m.Issues, "legacy_field_evidence_invalid") {
		t.Fatal("forged legacy span accepted")
	}
}

func TestMappingRejectedServiceClaimDoesNotConsumePrefix(t *testing.T) {
	d := syntheticDocument("Synthetic free text", "-1.00", "19.10.2026", "RUB", "Synthetic merchant")
	p := Proposal{Rows: []RowMapping{{OperationDate: Selection{"s2"}, AccountAmount: Selection{"s1"}, AccountCurrency: Selection{"s3"}, Description: Selection{"s4"}}}, Service: []ServiceMapping{{Selection{"s0", "s1"}, "Invalid overlap"}}}
	r, e := ValidateProposal(d, p, []int{1}, true, "")
	if e != nil || !contains(r.Reconciliation.Blockers, "overlapping_evidence") || !reflect.DeepEqual(r.unmappedSpanIDs, []string{"s0"}) {
		t.Fatal("rejected group hid an unmapped fragment", r, e)
	}
}
