package statementdata

import (
	"encoding/json"
	"reflect"
	"sort"
	"strings"
)

const MappingVersion = "statement-review-mapping.v1"

type MappingSnapshot struct {
	Version         string   `json:"version"`
	LayoutHash      string   `json:"layoutHash"`
	Origin          string   `json:"origin"`
	DraftRevision   int64    `json:"draftRevision"`
	Proposal        Proposal `json:"proposal"`
	UnmappedSpanIDs []string `json:"unmappedSpanIds"`
}
type DraftMapping struct {
	Version             string    `json:"version"`
	Revision            int64     `json:"revision"`
	Status              string    `json:"status"`
	Proposal            *Proposal `json:"proposal"`
	UnmappedSpanIDs     []string  `json:"unmappedSpanIds"`
	UnclassifiedSpanIDs []string  `json:"unclassifiedSpanIds"`
	Issues              []string  `json:"issues"`
}

// CanonicalMapping copies selections, normalizes empty arrays for the client,
// and orders service sets only. It never fills or changes financial selections.
func CanonicalMapping(d Document, p Proposal) Proposal {
	clone := func(ids Selection) Selection { return append(Selection{}, ids...) }
	out := Proposal{Rows: make([]RowMapping, len(p.Rows)), Service: make([]ServiceMapping, len(p.Service)), Header: p.Header}
	for i, row := range p.Rows {
		out.Rows[i] = RowMapping{OperationDate: clone(row.OperationDate), BookingDate: clone(row.BookingDate), AccountAmount: clone(row.AccountAmount), AccountCurrency: clone(row.AccountCurrency), TransactionAmount: clone(row.TransactionAmount), TransactionCurrency: clone(row.TransactionCurrency), Description: clone(row.Description), Direction: row.Direction, DirectionEvidence: clone(row.DirectionEvidence)}
	}
	out.Header = HeaderMapping{OpeningBookDate: clone(p.Header.OpeningBookDate), ClosingBookDate: clone(p.Header.ClosingBookDate), OpeningBookBalance: clone(p.Header.OpeningBookBalance), ClosingBookBalance: clone(p.Header.ClosingBookBalance), AvailableBalanceObserved: clone(p.Header.AvailableBalanceObserved), DebitTurnover: clone(p.Header.DebitTurnover), CreditTurnover: clone(p.Header.CreditTurnover), Currency: clone(p.Header.Currency), Labels: map[string]Selection{}}
	for key, ids := range p.Header.Labels {
		out.Header.Labels[key] = clone(ids)
	}
	for i, group := range p.Service {
		out.Service[i] = ServiceMapping{Spans: clone(group.Spans), Reason: group.Reason}
	}
	positions := map[string]int{}
	for i, s := range d.Spans {
		positions[s.ID] = i
	}
	for _, s := range out.Service {
		sort.SliceStable(s.Spans, func(i, j int) bool { return positions[s.Spans[i]] < positions[s.Spans[j]] })
	}
	return out
}
func NewMappingSnapshot(d Document, p Proposal, r Review, origin string, revision int64) MappingSnapshot {
	if origin == "ai" || origin == "model" {
		origin = "model"
	} else {
		origin = "owner"
	}
	return MappingSnapshot{MappingVersion, d.Hash(), origin, revision, CanonicalMapping(d, p), append([]string{}, r.unmappedSpanIDs...)}
}

// MappingFor is a read-only projection. A complete snapshot is not a financial
// approval; legacy complements are unclassified, never guessed service claims.
func MappingFor(d *Document, r *Review, raw []byte, revision int64) *DraftMapping {
	if r == nil && len(raw) == 0 {
		return nil
	}
	out := &DraftMapping{Version: MappingVersion, Revision: revision, Status: "unavailable", UnclassifiedSpanIDs: []string{}, Issues: []string{}}
	if d == nil {
		out.Issues = append(out.Issues, "statement_original_gone")
		return out
	}
	if len(raw) > 0 {
		var snap MappingSnapshot
		if json.Unmarshal(raw, &snap) != nil || (snap.Origin != "model" && snap.Origin != "owner") || snap.Version != MappingVersion || snap.LayoutHash != d.Hash() || snap.DraftRevision < 1 || snap.DraftRevision > revision {
			out.Issues = append(out.Issues, "mapping_snapshot_invalid")
			return out
		}
		// Verify stored coverage through the actual validator, not a client claim.
		check, err := ValidateProposal(*d, snap.Proposal, nil, false, "")
		if err != nil || r == nil || !reflect.DeepEqual(check.Rows, r.Rows) || !reflect.DeepEqual(check.Header, r.Header) || !reflect.DeepEqual(check.unmappedSpanIDs, snap.UnmappedSpanIDs) {
			out.Issues = append(out.Issues, "mapping_snapshot_invalid")
			return out
		}
		p := CanonicalMapping(*d, snap.Proposal)
		out.Proposal = &p
		out.Status = "complete"
		out.UnmappedSpanIDs = append([]string{}, snap.UnmappedSpanIDs...)
		return out
	}
	if r == nil {
		out.Issues = append(out.Issues, "mapping_review_missing")
		return out
	}
	out.Status = "reconstructed_partial"
	out.Issues = append(out.Issues, "legacy_service_mapping_missing")
	p := Proposal{Rows: []RowMapping{}, Service: []ServiceMapping{}}
	used := map[string]bool{}
	byID := map[string]Span{}
	for _, s := range d.Spans {
		byID[s.ID] = s
	}
	fields := func(evs []Evidence, aliases map[string]string) map[string]Selection {
		result := map[string]Selection{}
		invalid := map[string]bool{}
		for _, ev := range evs {
			role := ev.Role
			if i := strings.LastIndex(role, ":"); i >= 0 {
				role = role[i+1:]
			}
			key, ok := aliases[role]
			if !ok {
				continue
			}
			ids := Selection{}
			valid := len(ev.Spans) > 0
			seen := map[string]bool{}
			for _, s := range ev.Spans {
				expected, ok := byID[s.ID]
				if !ok || expected != s || seen[s.ID] {
					valid = false
				}
				seen[s.ID] = true
				ids = append(ids, s.ID)
			}
			if old, ok := result[key]; ok && !reflect.DeepEqual(old, ids) {
				valid = false
			}
			if !valid {
				invalid[key] = true
				out.Issues = append(out.Issues, "legacy_field_evidence_invalid")
			} else {
				result[key] = ids
			}
		}
		for key := range invalid {
			delete(result, key)
		}
		for _, ids := range result {
			for _, id := range ids {
				used[id] = true
			}
		}
		return result
	}
	rowAliases := map[string]string{"operationDate": "operationDate", "bookingDate": "bookingDate", "accountAmount": "accountAmount", "accountAmountCurrency": "accountCurrency", "transactionAmount": "transactionAmount", "transactionAmountCurrency": "transactionCurrency", "description": "description", "direction": "directionEvidence"}
	for _, row := range r.Rows {
		b, _ := json.Marshal(fields(row.Evidence, rowAliases))
		var m RowMapping
		_ = json.Unmarshal(b, &m)
		m.Direction = row.Direction
		p.Rows = append(p.Rows, m)
	}
	headerAliases := map[string]string{"headerCurrency": "currency"}
	for _, key := range []string{"openingBookDate", "closingBookDate", "openingBookBalance", "closingBookBalance", "availableBalanceObserved", "debitTurnover", "creditTurnover"} {
		headerAliases[key] = key
		headerAliases[key+"Label"] = key + "Label"
	}
	h := fields(r.Header.Evidence, headerAliases)
	labels := map[string]Selection{}
	values := map[string]any{}
	for key, ids := range h {
		if strings.HasSuffix(key, "Label") {
			labels[strings.TrimSuffix(key, "Label")] = ids
		} else {
			values[key] = ids
		}
	}
	values["labels"] = labels
	b, _ := json.Marshal(values)
	_ = json.Unmarshal(b, &p.Header)
	p = CanonicalMapping(*d, p)
	out.Proposal = &p
	for _, s := range d.Spans {
		if !used[s.ID] {
			out.UnclassifiedSpanIDs = append(out.UnclassifiedSpanIDs, s.ID)
		}
	}
	return out
}
