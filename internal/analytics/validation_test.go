package analytics

import (
	"errors"
	"testing"
	"time"
)

func TestParseIncomeAnchorQuery(t *testing.T) {
	const categoryID = "20000000-0000-0000-0000-000000000004"
	asOf, parsedCategory, err := ParseIncomeAnchorQuery("2026-10-05T12:00:00+05:00", categoryID)
	if err != nil || parsedCategory == nil || *parsedCategory != categoryID || asOf.Location() == time.UTC {
		t.Fatalf("unexpected parsed query: %v %v %v", asOf, parsedCategory, err)
	}

	asOf, parsedCategory, err = ParseIncomeAnchorQuery("2026-10-05T07:00:00Z", "")
	if err != nil || parsedCategory != nil || !asOf.Equal(time.Date(2026, 10, 5, 7, 0, 0, 0, time.UTC)) {
		t.Fatalf("unexpected unfiltered query: %v %v %v", asOf, parsedCategory, err)
	}
}

func TestParseIncomeAnchorQueryRejectsInvalidInput(t *testing.T) {
	for _, test := range []struct {
		name, asOf, categoryID string
	}{
		{name: "missing asOf"},
		{name: "asOf without offset", asOf: "2026-10-05T12:00:00"},
		{name: "bad category", asOf: "2026-10-05T12:00:00Z", categoryID: "not-a-uuid"},
		{name: "noncanonical category", asOf: "2026-10-05T12:00:00Z", categoryID: "20000000000000000000000000000004"},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, _, err := ParseIncomeAnchorQuery(test.asOf, test.categoryID)
			if !errors.Is(err, ErrInvalidIncomeAnchorQuery) {
				t.Fatalf("expected invalid query, got %v", err)
			}
		})
	}
}
