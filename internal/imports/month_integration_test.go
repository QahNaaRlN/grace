//go:build integration

package imports

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/QahNaaRlN/grace/internal/analytics"
	"github.com/QahNaaRlN/grace/internal/platform/access"
	"github.com/go-chi/chi/v5"
)

func TestCSVMainBlockDuplicateNeedsReview(t *testing.T) {
	ctx, _, r := setup(t)
	b, err := r.UploadCSV(ctx, csvFixture(',', 2), "main-duplicates", "main-duplicates-upload")
	if err != nil {
		t.Fatal(err)
	}
	rows, err := r.Records(ctx, b.ID, "", "transaction", "", 100)
	if err != nil {
		t.Fatal(err)
	}
	d := createDecision("purchase")
	d.Fields.AmountMinor = 76275
	d.Fields.OccurredAt = "2026-09-01T12:00:00+05:00"
	b, err = r.Edit(ctx, b.ID, rows.Items[0].ID, "main-duplicate-edit", Edit{b.Revision, d})
	if err != nil {
		t.Fatal(err)
	}
	detail, err := r.Record(ctx, b.ID, rows.Items[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, c := range detail.Candidates {
		if c.RecordID == rows.Items[1].ID {
			found = true
		}
	}
	if !found {
		t.Fatal("duplicate inside main block hidden")
	}
	preview, err := r.Preview(ctx, b.ID, "main-duplicate-preview", Selection{b.Revision, []string{rows.Items[0].ID}})
	if err != nil || len(preview.Blockers) == 0 {
		t.Fatal("unreviewed main duplicate accepted", err)
	}
	// Compare reviewed fields too, rather than only immutable source amounts.
	d.Fields.AmountMinor = 78001
	b, err = r.Edit(ctx, b.ID, rows.Items[0].ID, "main-correct-first", Edit{b.Revision, d})
	if err != nil {
		t.Fatal(err)
	}
	b, err = r.Edit(ctx, b.ID, rows.Items[1].ID, "main-correct-second", Edit{b.Revision, d})
	if err != nil {
		t.Fatal(err)
	}
	detail, err = r.Record(ctx, b.ID, rows.Items[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	found = false
	for _, c := range detail.Candidates {
		if c.RecordID == rows.Items[1].ID {
			found = true
		}
	}
	if !found {
		t.Fatal("corrected duplicate hidden")
	}
}

func TestCSVMonthLifecycleAndHistoricalAnalytics(t *testing.T) {
	ctx, p, r := setup(t)
	raw := csvFixture(',', 191)
	b, err := r.UploadCSV(ctx, raw, "monthly", "monthly-upload")
	if err != nil {
		t.Fatal(err)
	}
	page, err := r.Records(ctx, b.ID, "", "transaction", "", 100)
	if err != nil || len(page.Items) != 100 || page.NextCursor == "" {
		t.Fatal("pagination", err)
	}
	report, err := r.Month(ctx, b.ID, "2026-09")
	if err != nil || len(report.Rows) != 191 || report.Pending != 191 || *report.Totals[0].Source != 191*76275 {
		t.Fatal("month lost rows", err, report)
	}
	var retained []byte
	if err = p.QueryRow(ctx, "SELECT snapshot FROM import_batches WHERE id=$1", b.ID).Scan(&retained); err != nil || !bytes.Equal(retained, raw) {
		t.Fatal("CSV bytes lost", err)
	}
	retry, err := r.UploadCSV(ctx, raw, "monthly", "monthly-retry")
	if err != nil || retry.ID != b.ID {
		t.Fatal("duplicate CSV", err)
	}
	_, err = r.UploadCSV(ctx, raw, "other-source", "monthly-conflict")
	requireCode(t, err, "source_conflict")
	_, err = r.Month(ctx, b.ID, "2026-13")
	requireCode(t, err, "invalid_month")
	from := time.Date(2026, 9, 1, 0, 0, 0, 0, time.FixedZone("Yekaterinburg", 5*3600))
	to := from.AddDate(0, 1, 0)
	analyticsRepo := analytics.NewRepository(p)
	before, err := analyticsRepo.PeriodSummary(ctx, from, to)
	if err != nil {
		t.Fatal(err)
	}
	purchases := func(s analytics.PeriodSummary) int64 {
		for _, c := range s.Currencies {
			if c.Currency == "RUB" {
				return c.PurchasesMinor
			}
		}
		return 0
	}
	// Confirm a separate synthetic one-row history, preserving September's date.
	b, err = r.UploadCSV(ctx, csvFixture(';', 1), "single-month", "single-upload")
	if err != nil {
		t.Fatal(err)
	}
	page, err = r.Records(ctx, b.ID, "", "transaction", "", 100)
	if err != nil {
		t.Fatal(err)
	}
	d := createDecision("purchase")
	d.Fields.AmountMinor = 76275
	d.Fields.OccurredAt = "2026-09-01T12:00:00+05:00"
	b, err = r.Edit(ctx, b.ID, page.Items[0].ID, "single-edit", Edit{b.Revision, d})
	if err != nil {
		t.Fatal(err)
	}
	report, err = r.Month(ctx, b.ID, "2026-09")
	if err != nil || report.Complete || *report.Totals[0].Prepared != 76275 || *report.Totals[0].Confirmed != 0 {
		t.Fatal("prepared report", err, report)
	}
	unchanged, err := analyticsRepo.PeriodSummary(ctx, from, to)
	if err != nil || purchases(unchanged) != purchases(before) {
		t.Fatal("draft changed ledger", err)
	}
	preview, err := r.Preview(ctx, b.ID, "single-preview", Selection{b.Revision, []string{page.Items[0].ID}})
	if err != nil || len(preview.Blockers) > 0 {
		t.Fatal(err, preview.Blockers)
	}
	receipt, err := r.Confirm(ctx, b.ID, "single-confirm", Confirmation{preview.ID, preview.Hash})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = r.Confirm(ctx, b.ID, "single-confirm", Confirmation{preview.ID, preview.Hash}); err != nil {
		t.Fatal(err)
	}
	report, err = r.Month(ctx, b.ID, "2026-09")
	if err != nil || !report.Complete || *report.Totals[0].Confirmed != 76275 || *report.Totals[0].Prepared != 0 {
		t.Fatal("confirmed report", err, report)
	}
	after, err := analyticsRepo.PeriodSummary(ctx, from, to)
	if err != nil || purchases(after)-purchases(before) != 76275 {
		t.Fatal("history missing or doubled", err)
	}
	if receipt.Outcomes[0].Event.OccurredAt.In(from.Location()).Format("2006-01-02") != "2026-09-01" {
		t.Fatal("historical date changed")
	}
	var otherID string
	if err = p.QueryRow(ctx, "INSERT INTO users(display_name,role) VALUES('Month report other owner','admin') RETURNING id::text").Scan(&otherID); err != nil {
		t.Fatal(err)
	}
	other := access.WithPrincipal(context.Background(), access.Principal{ID: otherID, Role: "admin"})
	_, err = r.Month(other, b.ID, "2026-09")
	requireCode(t, err, "not_found")
	router := chi.NewRouter()
	NewHandler(r).Routes(router)
	for _, tc := range []struct {
		owner  context.Context
		status int
	}{{ctx, 200}, {other, 404}} {
		req := httptest.NewRequest(http.MethodGet, "/batches/"+b.ID+"/month?month=2026-09", nil).WithContext(tc.owner)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != tc.status {
			t.Fatal("report HTTP isolation", w.Code, w.Body.String())
		}
		if w.Code == 200 {
			var v MonthReport
			if err = json.Unmarshal(w.Body.Bytes(), &v); err != nil || !v.Complete {
				t.Fatal(err)
			}
		}
	}
}
