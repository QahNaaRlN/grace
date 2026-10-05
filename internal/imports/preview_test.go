package imports

import (
	"math"
	"testing"

	"github.com/QahNaaRlN/grace/internal/importreview"
)

func TestMoneyOverflow(t *testing.T) {
	for _, pair := range [][2]int64{{math.MaxInt64, 1}, {math.MinInt64, -1}} {
		if _, ok := add(pair[0], pair[1]); ok {
			t.Fatal("overflow accepted")
		}
	}
	if sum, ok := add(math.MaxInt64, -1); !ok || sum != math.MaxInt64-1 {
		t.Fatal(sum, ok)
	}
}
func TestSelectionBounds(t *testing.T) {
	id := "10000000-0000-0000-0000-000000000001"
	for _, ids := range [][]string{nil, {id, id}, {"invalid"}, make([]string, 101)} {
		if selectionValid(ids) {
			t.Fatal("invalid selection accepted")
		}
	}
	if !selectionValid([]string{id}) {
		t.Fatal("valid selection rejected")
	}
}
func TestClosedDecisions(t *testing.T) {
	for _, d := range []Decision{{Action: "publish"}, {Action: "exclude"}, {Action: "create"}, {Action: "duplicate_of", CanonicalRecordID: "bad", Reason: "review"}, {Action: "unresolved", EventID: "bad"}, {Action: "exclude", Reason: "reason", Resolutions: []Resolution{{"x", ""}}}} {
		if validateDecision(d) == nil {
			t.Fatalf("accepted %#v", d)
		}
	}
}
func TestFieldsRejectInvalidAmountDateReference(t *testing.T) {
	for _, f := range []*Fields{nil, {Kind: "purchase", AmountMinor: 0, OccurredAt: "bad"}, {Kind: "purchase", Currency: "RUB", AmountMinor: 1, OccurredAt: "2026-10-06T12:00:00+05:00", AccountID: "foreign"}} {
		if _, err := command(f, "key"); err == nil {
			t.Fatal("invalid fields accepted")
		}
	}
}
func TestCandidatesRequireExplicitDecision(t *testing.T) {
	rec := Record{ID: "base", Decision: Decision{Action: "create"}, Source: importreview.Record{}}
	c := Candidate{ID: "record:extra", RecordID: "extra"}
	if resolved(rec, c, nil) {
		t.Fatal("candidate resolved automatically")
	}
	rec.Decision.Resolutions = []Resolution{{c.ID, "different recipients"}}
	if !resolved(rec, c, nil) {
		t.Fatal("explicit difference not accepted")
	}
	rec.Decision.Resolutions = nil
	if !resolved(rec, c, map[string]Record{"extra": {Decision: Decision{Action: "duplicate_of", CanonicalRecordID: "base"}}}) {
		t.Fatal("explicit duplicate not accepted")
	}
}
