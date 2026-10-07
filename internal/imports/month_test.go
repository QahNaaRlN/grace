package imports

import (
	"fmt"
	"math"
	"slices"
	"testing"
	"time"

	"github.com/QahNaaRlN/grace/internal/financialevents"
	"github.com/QahNaaRlN/grace/internal/importreview"
)

func minor(v int64) *int64 { return &v }
func monthlyRecord(id string, amount *int64) Record {
	return Record{ID: id, Source: importreview.Record{Sheet: "Операции", Row: 5, Domain: "transaction", Normalized: importreview.Fields{Kind: "purchase", AmountMinor: amount, Currency: "RUB", Date: &importreview.Date{Local: "2026-09-01", Precision: "date"}}}, Decision: Decision{Action: "unresolved"}}
}
func TestMonthCoversEntireBatchAndUnknowns(t *testing.T) {
	rows := []Record{}
	for i := 0; i < 191; i++ {
		row := monthlyRecord(fmt.Sprint(i), minor(76275))
		row.Source.Row = 5 + i
		rows = append(rows, row)
	}
	unknown := monthlyRecord("unknown", minor(100))
	unknown.Source.Normalized.Date = nil
	rows = append(rows, unknown)
	other := monthlyRecord("other", minor(100))
	other.Source.Normalized.Date.Local = "2026-08-31"
	rows = append(rows, other)
	r := buildMonth(Batch{Revision: 7}, "2026-09", rows, nil)
	if len(r.Rows) != 191 || r.Pending != 191 || len(r.UnknownDateRows) != 1 || r.OutsidePeriod != 1 || *r.Totals[0].Source != 191*76275 || r.Complete {
		t.Fatal(r)
	}
	rows[0].Source.Normalized.AmountMinor = nil
	if buildMonth(Batch{}, "2026-09", rows, nil).Totals[0].Source != nil {
		t.Fatal("unknown sum invented")
	}
}

func TestMonthDecisionsUniqueEventsAndCorrections(t *testing.T) {
	create := monthlyRecord("create", minor(1000))
	create.Decision = Decision{Action: "create", Fields: &Fields{Kind: "purchase", AmountMinor: 900, Currency: "RUB", OccurredAt: "2026-09-01T12:00:00+05:00"}, Reason: "Исправление суммы"}
	link := monthlyRecord("link", minor(1000))
	link.Decision = Decision{Action: "link_existing", EventID: "event", Reason: "Та же покупка"}
	link.Published = true
	link2 := monthlyRecord("link2", minor(1000))
	link2.Decision = link.Decision
	duplicate := monthlyRecord("duplicate", minor(1000))
	duplicate.Decision = Decision{Action: "duplicate_of", CanonicalRecordID: "create", Reason: "Дубль"}
	duplicate.Published = true
	exclude := monthlyRecord("exclude", minor(1000))
	exclude.Decision = Decision{Action: "exclude", Reason: "Это план"}
	exclude.Published = true
	e := financialevents.Event{Kind: "purchase", OccurredAt: func() *time.Time { t := time.Date(2026, 9, 1, 7, 0, 0, 0, time.UTC); return &t }(), Postings: []financialevents.Posting{{Direction: "debit", Amount: 1000, Currency: "RUB"}}}
	rows := []Record{create, link, link2, duplicate, exclude}
	r := buildMonth(Batch{}, "2026-09", rows, map[string]financialevents.Event{"event": e})
	if *r.Totals[0].Source != 5000 || *r.Totals[0].Prepared != 900 || *r.Totals[0].Confirmed != 1000 || r.Complete {
		t.Fatal(r)
	}
	changed := false
	for _, row := range r.Rows {
		if row.RecordID == "create" {
			changed = slices.Contains(row.Differences, "amount_changed") && row.Reason != ""
		}
	}
	if !changed {
		t.Fatal("correction unexplained")
	}
	for i := range rows {
		rows[i].Published = true
	}
	if !buildMonth(Batch{}, "2026-09", rows, map[string]financialevents.Event{"event": e}).Complete {
		t.Fatal("complete month not recognized")
	}
	rows[0].Decision.Fields.OccurredAt = "2026-10-01T00:00:00+05:00"
	r = buildMonth(Batch{}, "2026-09", rows, map[string]financialevents.Event{"event": e})
	if *r.Totals[0].Confirmed != 1000 {
		t.Fatal("out of period purchase counted", r)
	}
}

func TestMonthCurrenciesOverflowAndCalendar(t *testing.T) {
	a := monthlyRecord("a", minor(math.MaxInt64))
	b := monthlyRecord("b", minor(1))
	usd := monthlyRecord("usd", minor(200))
	usd.Source.Normalized.Currency = "USD"
	r := buildMonth(Batch{}, "2026-09", []Record{a, b, usd}, nil)
	if r.Totals[0].Source != nil || *r.Totals[1].Source != 200 {
		t.Fatal("overflow or currency mixing", r.Totals)
	}
	for _, month := range []string{"2026-00", "2026-13", "2026-9", "0000-01", "2026-09-01", ""} {
		if validMonth(month) {
			t.Fatal(month)
		}
	}
	if !validMonth("2026-09") || targetLocalDate("2026-08-31T20:00:00Z") != "2026-09-01" {
		t.Fatal("local calendar broken")
	}
}

func TestMonthMissingDateNeedsExplicitReviewedDate(t *testing.T) {
	row := monthlyRecord("missing-date", minor(100))
	row.Source.Normalized.Date = nil
	row.Decision = Decision{Action: "create", Reason: "Восстановлена дата по выписке", Fields: &Fields{Kind: "purchase", AmountMinor: 100, Currency: "RUB", OccurredAt: "2026-09-01T12:00:00+05:00"}}
	r := buildMonth(Batch{}, "2026-09", []Record{row}, nil)
	if len(r.UnknownDateRows) != 1 || len(r.Rows) != 0 {
		t.Fatal("unconfirmed date invented")
	}
	row.Decision.Fields.TimeReason = "Дата и время из выписки"
	r = buildMonth(Batch{}, "2026-09", []Record{row}, nil)
	if len(r.UnknownDateRows) != 0 || len(r.Rows) != 1 || r.Rows[0].SourceDate != "" || !slices.Contains(r.Rows[0].Differences, "date_changed") || r.Complete {
		t.Fatal("reviewed date lost source precision", r)
	}
	row.Published = true
	if !buildMonth(Batch{}, "2026-09", []Record{row}, nil).Complete {
		t.Fatal("reviewed date cannot finish month")
	}
	ignored := monthlyRecord("excluded-no-date", minor(100))
	ignored.Source.Normalized.Date = nil
	ignored.Decision = Decision{Action: "exclude", Reason: "Это заметка, не расход"}
	if buildMonth(Batch{}, "2026-09", []Record{row, ignored}, nil).Complete {
		t.Fatal("unconfirmed exclusion ignored")
	}
	ignored.Published = true
	r = buildMonth(Batch{}, "2026-09", []Record{row, ignored}, nil)
	if !r.Complete || len(r.UnknownDateRows) != 1 || r.UnknownDateRows[0].Reason == "" {
		t.Fatal("confirmed exclusion blocks completion or loses evidence", r)
	}
}
