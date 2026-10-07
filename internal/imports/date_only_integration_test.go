//go:build integration

package imports

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/QahNaaRlN/grace/internal/analytics"
	"github.com/QahNaaRlN/grace/internal/financialevents"
)

func TestDateOnlyHistoryPublicationAndCalendarAnalytics(t *testing.T) {
	ctx, p, r := setup(t)
	zone, err := time.LoadLocation("Asia/Yekaterinburg")
	if err != nil {
		t.Fatal(err)
	}
	from := time.Date(2026, 9, 2, 0, 0, 0, 0, zone)
	to := from.AddDate(0, 0, 1)
	a := analytics.NewRepository(p)
	purchases := func(s analytics.PeriodSummary) int64 {
		for _, c := range s.Currencies {
			if c.Currency == "RUB" {
				return c.PurchasesMinor
			}
		}
		return 0
	}
	before, err := a.PeriodSummary(ctx, from, to)
	if err != nil {
		t.Fatal(err)
	}
	b, rows := uploadOne(t, ctx, r, "date-only-history", "Расход", false)
	d := createDecision("purchase")
	d.Fields.OccurredAt = ""
	d.Fields.OccurredDate = "2026-09-02"
	d.Fields.TimeReason = ""
	d.Fields.AmountMinor = 118899
	b, err = r.Edit(ctx, b.ID, rows[0].ID, "date-only-edit", Edit{b.Revision, d})
	if err != nil {
		t.Fatal(err)
	}
	preview, err := r.Preview(ctx, b.ID, "date-only-preview", Selection{b.Revision, []string{rows[0].ID}})
	if err != nil || len(preview.Blockers) > 0 {
		t.Fatal(err, preview.Blockers)
	}
	receipt, err := r.Confirm(ctx, b.ID, "date-only-confirm", Confirmation{preview.ID, preview.Hash})
	if err != nil {
		t.Fatal(err)
	}
	e := receipt.Outcomes[0].Event
	if e.OccurredAt != nil || e.OccurredDate == nil || *e.OccurredDate != "2026-09-02" || e.DatePrecision != "date" {
		t.Fatal("unknown time lost", e)
	}
	var nullTime bool
	var date string
	if err = p.QueryRow(ctx, "SELECT occurred_at IS NULL,occurred_date::text FROM financial_events WHERE id=$1", e.ID).Scan(&nullTime, &date); err != nil || !nullTime || date != "2026-09-02" {
		t.Fatal("DB invented timestamp", err)
	}
	raw, _ := json.Marshal(e)
	var dto map[string]any
	json.Unmarshal(raw, &dto)
	if dto["occurredAt"] != nil || dto["occurredDate"] != "2026-09-02" {
		t.Fatal("API invented timestamp")
	}
	if _, err = r.Confirm(ctx, b.ID, "date-only-confirm", Confirmation{preview.ID, preview.Hash}); err != nil {
		t.Fatal(err)
	}
	after, err := a.PeriodSummary(ctx, from, to)
	if err != nil || purchases(after)-purchases(before) != 118899 {
		t.Fatal("calendar amount missing or doubled", err)
	}
	if _, err = a.PeriodSummary(ctx, from.Add(10*time.Hour), to); !errors.Is(err, analytics.ErrTimePrecision) {
		t.Fatal("partial day accepted", err)
	}
	month, err := r.Month(ctx, b.ID, "2026-10")
	if err != nil || len(month.Rows) != 1 || month.Rows[0].TargetDate != "2026-09-02" {
		t.Fatal("source month correction hidden", err)
	}
	_, _, err = financialevents.NewRepository(p).Create(ctx, financialevents.Command{Kind: "income_received", IdempotencyKey: "date-only-income", RequestHash: digest("date-only-income"), AccountID: accountID, AmountMinor: 400000, OccurredDate: "2026-09-05"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = a.IncomeRelativePeriod(ctx, time.Date(2026, 9, 6, 12, 0, 0, 0, zone), nil)
	if code, ok := analytics.IncomeAnchorErrorCodeOf(err); !ok || code != analytics.CodeIncomeTimeUnknown {
		t.Fatal("income time invented", err)
	}
}
