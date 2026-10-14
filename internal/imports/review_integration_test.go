//go:build integration

package imports

import (
	"encoding/csv"
	"fmt"
	"strings"
	"testing"
	"time"
)

// scaleFixture writes n distinct CSV rows (no duplicate candidates); every
// tenth one mentions coffee.
func scaleFixture(n int) []byte {
	var body strings.Builder
	w := csv.NewWriter(&body)
	_ = w.Write([]string{"Финансовое здоровье"})
	_ = w.Write([]string{""})
	_ = w.Write([]string{"Описание листа"})
	_ = w.Write([]string{"Дата", "Тип операции", "Описание", "Категория", "Подкатегория", "Сумма", "Источник денег", "Счёт", "Приоритет", "Регулярный", "Получатель", "Комментарий", "Документ", "Статус", "ID"})
	for i := 0; i < n; i++ {
		description := fmt.Sprintf("Покупка %d", i)
		if i%10 == 0 {
			description = fmt.Sprintf("Кофе_%d%%", i)
		}
		_ = w.Write([]string{fmt.Sprintf("%02d.09.2026", i%28+1), "Расход", description, "Еда", "", fmt.Sprintf("%d,%02d", 100+i, i%100), "Собственные средства", "Карта", "", "Нет", "Магазин", "", "", "", fmt.Sprintf("scale-%d", i)})
	}
	w.Flush()
	return []byte(body.String())
}

// SPEC-0044: history facts, server filters/sort/search, the unresolved walk
// and preview by filter on 5 000 rows, with a page under 300 ms.
func TestReviewAtScale(t *testing.T) {
	ctx, _, r := setup(t)
	b, err := r.UploadCSVNamed(ctx, scaleFixture(5000), "scale", "scale-upload", `C:\выписки\сентябрь.csv`)
	if err != nil {
		t.Fatal(err)
	}
	if b.FileName == nil || *b.FileName != "сентябрь.csv" || b.Format == nil || *b.Format != "csv" {
		t.Fatalf("file facts: %v %v", b.FileName, b.Format)
	}
	if b.Total != 5000 || b.Reviewed != 0 || *b.PeriodFrom != "2026-09-01" || *b.PeriodTo != "2026-09-28" || b.LastConfirmation != nil {
		t.Fatalf("summary: total %d reviewed %d period %v–%v", b.Total, b.Reviewed, *b.PeriodFrom, *b.PeriodTo)
	}
	// The summary count and the «С вопросами» filter agree.
	questions, err := r.RecordsQuery(ctx, b.ID, RecordQuery{Limit: 1, RecordFilter: RecordFilter{Status: "needs_review"}})
	if err != nil || b.NeedsReview == 0 || *questions.Total != b.NeedsReview {
		t.Fatalf("needs review: summary %d, filter %v, %v", b.NeedsReview, questions.Total, err)
	}

	// AC5: a sorted, searched page of 5 000 rows stays under 300 ms.
	start := time.Now()
	page, err := r.RecordsQuery(ctx, b.ID, RecordQuery{Limit: 100, Sort: "-amount", RecordFilter: RecordFilter{Domain: "transaction"}})
	elapsed := time.Since(start)
	if err != nil || len(page.Items) != 100 || *page.Total != 5000 || page.NextCursor != "100" {
		t.Fatalf("page: %d items, total %v, cursor %q, %v", len(page.Items), page.Total, page.NextCursor, err)
	}
	if elapsed > 300*time.Millisecond {
		t.Fatalf("page took %v", elapsed)
	}
	if *page.Items[0].Source.Normalized.AmountMinor != 509999 {
		t.Fatalf("largest amount first, got %d", *page.Items[0].Source.Normalized.AmountMinor)
	}
	start = time.Now()
	if history, err := r.Batches(ctx, "", 100); err != nil || len(history.Items) == 0 || time.Since(start) > 300*time.Millisecond {
		t.Fatalf("history: %v in %v", err, time.Since(start))
	}
	last, err := r.RecordsQuery(ctx, b.ID, RecordQuery{Limit: 100, Cursor: "4950", Sort: "row"})
	if err != nil || len(last.Items) != 50 || last.NextCursor != "" || last.Items[49].Source.Row != 5004 {
		t.Fatalf("last page: %d %q %v", len(last.Items), last.NextCursor, err)
	}

	// Search is literal: % and _ do not act as wildcards.
	coffee, err := r.RecordsQuery(ctx, b.ID, RecordQuery{Limit: 10, RecordFilter: RecordFilter{Q: "кофе"}})
	if err != nil || *coffee.Total != 500 {
		t.Fatalf("search: %v %v", coffee.Total, err)
	}
	literal, _ := r.RecordsQuery(ctx, b.ID, RecordQuery{Limit: 10, RecordFilter: RecordFilter{Q: "е_10%"}})
	if *literal.Total != 1 {
		t.Fatalf("literal search matched %d", *literal.Total)
	}
	if _, err = r.RecordsQuery(ctx, b.ID, RecordQuery{Limit: 10, Sort: "id; DROP"}); err == nil {
		t.Fatal("unknown sort accepted")
	}

	// AC3: next unresolved walks forward and wraps around.
	first, err := r.RecordsQuery(ctx, b.ID, RecordQuery{Limit: 3, Sort: "row"})
	if err != nil {
		t.Fatal(err)
	}
	next, err := r.NextUnresolved(ctx, b.ID, first.Items[0].ID)
	if err != nil || next.Record == nil || next.Record.ID != first.Items[1].ID || next.Remaining != 5000 {
		t.Fatalf("next: %+v %v", next, err)
	}
	b, err = r.Edit(ctx, b.ID, first.Items[1].ID, "scale-edit", Edit{ExpectedRevision: b.Revision, Decision: Decision{Action: "exclude", Reason: "Проверено"}})
	if err != nil {
		t.Fatal(err)
	}
	next, _ = r.NextUnresolved(ctx, b.ID, first.Items[0].ID)
	if next.Record.ID != first.Items[2].ID || next.Remaining != 4999 {
		t.Fatalf("decided row not skipped: %+v", next)
	}
	tail, _ := r.RecordsQuery(ctx, b.ID, RecordQuery{Limit: 1, Cursor: "4999", Sort: "row"})
	wrapped, _ := r.NextUnresolved(ctx, b.ID, tail.Items[0].ID)
	if wrapped.Record.ID != first.Items[0].ID {
		t.Fatal("walk does not wrap around")
	}
	if b, _ = r.Batch(ctx, b.ID); b.Reviewed != 1 || b.Counts["exclude"] != 1 {
		t.Fatalf("progress: %d", b.Reviewed)
	}

	// AC4: preview by filter for 500 rows; more than 1 000 is refused.
	preview, err := r.Preview(ctx, b.ID, "scale-preview", Selection{ExpectedRevision: b.Revision, Filter: &RecordFilter{Q: "кофе", Status: "unresolved"}})
	if err != nil || len(preview.Lines) != 500 {
		t.Fatalf("filter preview: %d lines %v", len(preview.Lines), err)
	}
	_, err = r.Preview(ctx, b.ID, "scale-too-many", Selection{ExpectedRevision: b.Revision, Filter: &RecordFilter{Domain: "transaction"}})
	requireCode(t, err, "selection_too_large")
	_, err = r.Preview(ctx, b.ID, "scale-both", Selection{ExpectedRevision: b.Revision, RecordIDs: []string{first.Items[0].ID}, Filter: &RecordFilter{}})
	requireCode(t, err, "invalid_selection")
	_, err = r.Preview(ctx, b.ID, "scale-stale", Selection{ExpectedRevision: b.Revision - 1, Filter: &RecordFilter{Q: "кофе"}})
	requireCode(t, err, "revision_conflict")
}
