//go:build integration

package imports

import (
	"bytes"
	"github.com/QahNaaRlN/grace/internal/statementdata"
	"image"
	"image/png"
	"testing"
)

func TestStatementPlainReviewConfirmAndInvalidation(t *testing.T) {
	ctx, p, r := statementSetup(t)
	var raw bytes.Buffer
	_ = png.Encode(&raw, image.NewRGBA(image.Rect(0, 0, 64, 64)))
	source, _, e := r.UploadStatementImage(ctx, raw.Bytes(), accountID, "plain-fixture.png", "plain-upload", "plain_review")
	if e != nil {
		t.Fatal(e)
	}
	reading := statementdata.ParsePlain("| Дата операции | Дата списания | Сумма операции | Сумма карты | Описание | Карта |\n| --- | --- | --- | --- | --- | --- |\n| 19.10.2026 12:34 | 19.10.2026 13:00 | -765,43 RUB | -765,43 RUB | Кафе | 1234 |\nИтог списаний: 765,43 RUB\nИтог пополнений: 0,00 RUB\nДоступный остаток: 99,00 RUB")
	source, e = r.ManualStatementDraft(ctx, source.ID, "plain-draft", StatementManualDraft{ExpectedRevision: source.Revision, SourceSHA256: source.SHA256, Plain: &reading, Reason: "Fixture"})
	if e != nil {
		t.Fatal(e)
	}
	count := func() int { var n int; _ = p.QueryRow(ctx, "SELECT count(*) FROM postings").Scan(&n); return n }
	before := count()
	if source.DraftMapping != nil {
		t.Fatal("plain reading gained native locator workflow")
	}
	batch, e := r.Batch(ctx, *source.BatchID)
	if e != nil {
		t.Fatal(e)
	}
	records, e := r.Records(ctx, batch.ID, "", "", "", 100)
	if e != nil || len(records.Items) != 1 {
		t.Fatal(e)
	}
	id := records.Items[0].ID
	d := createDecision("purchase")
	d.Fields.AmountMinor = 76543
	d.Fields.OccurredAt = ""
	d.Fields.OccurredDate = "2026-10-19"
	batch, e = r.Edit(ctx, batch.ID, id, "plain-decision", Edit{batch.Revision, d})
	if e != nil {
		t.Fatal(e)
	}
	blocked, e := r.Preview(ctx, batch.ID, "plain-blocked", Selection{ExpectedRevision: batch.Revision, RecordIDs: []string{id}})
	if e != nil || len(blocked.Blockers) == 0 {
		t.Fatal("unreviewed import", e)
	}
	proof := PlainReviewInput{ExpectedRevision: source.Revision, SourceSHA256: source.SHA256, DraftDigest: source.Plain.Digest, ReviewedPages: []int{1}, RolesConfirmed: true}
	forged := proof
	forged.DraftDigest = "forged"
	if _, e = r.ReviewPlain(ctx, source.ID, "plain-forged", forged); e == nil {
		t.Fatal("forged review accepted")
	}
	incomplete := proof
	incomplete.ReviewedPages = nil
	if _, e = r.ReviewPlain(ctx, source.ID, "plain-incomplete", incomplete); e == nil {
		t.Fatal("incomplete pages accepted")
	}
	source, e = r.ReviewPlain(ctx, source.ID, "plain-reviewed", proof)
	if e != nil {
		t.Fatal(e)
	}
	batch, e = r.Batch(ctx, batch.ID)
	if e != nil {
		t.Fatal(e)
	}
	preview, e := r.Preview(ctx, batch.ID, "plain-ready", Selection{ExpectedRevision: batch.Revision, RecordIDs: []string{id}})
	if e != nil || len(preview.Blockers) > 0 {
		t.Fatal(preview.Blockers, e)
	}
	// An actual table edit invalidates the previously issued financial preview.
	candidate := source.Plain.Candidate
	candidate.Rows[0].Raw["description"] = "Кафе исправлено"
	source, e = r.ManualStatementDraft(ctx, source.ID, "plain-correct", StatementManualDraft{ExpectedRevision: source.Revision, SourceSHA256: source.SHA256, Plain: &candidate, Reason: "Fix description"})
	if e != nil {
		t.Fatal(e)
	}
	if _, e = r.Confirm(ctx, batch.ID, "plain-stale-confirm", Confirmation{preview.ID, preview.Hash}); e == nil || count() != before {
		t.Fatal("stale preview wrote", e)
	}
	if source.RolesConfirmed || source.Plain.ReviewedDigest != "" {
		t.Fatal("edit retained review")
	}
	// Corrected values persist; typed input cannot forge amounts.
	source, e = r.Statement(ctx, source.ID)
	if e != nil || *source.Plain.Candidate.Rows[0].Description != "Кафе исправлено" {
		t.Fatal("reload lost correction", e)
	}
	proof.ExpectedRevision = source.Revision
	proof.DraftDigest = source.Plain.Digest
	source, e = r.ReviewPlain(ctx, source.ID, "plain-recheck", proof)
	if e != nil {
		t.Fatal(e)
	}
	batch, e = r.Batch(ctx, *source.BatchID)
	if e != nil {
		t.Fatal(e)
	}
	records, e = r.Records(ctx, batch.ID, "", "", "", 100)
	if e != nil {
		t.Fatal(e)
	}
	id = records.Items[0].ID
	batch, e = r.Edit(ctx, batch.ID, id, "plain-redecision", Edit{batch.Revision, d})
	if e != nil {
		t.Fatal(e)
	}
	preview, e = r.Preview(ctx, batch.ID, "plain-ready-again", Selection{ExpectedRevision: batch.Revision, RecordIDs: []string{id}})
	if e != nil || len(preview.Blockers) > 0 {
		t.Fatal(preview.Blockers, e)
	}
	receipt, e := r.Confirm(ctx, batch.ID, "plain-confirm", Confirmation{preview.ID, preview.Hash})
	if e != nil || count() != before+1 {
		t.Fatal(e)
	}
	replay, e := r.Confirm(ctx, batch.ID, "plain-confirm", Confirmation{preview.ID, preview.Hash})
	if e != nil || replay.ID != receipt.ID || count() != before+1 {
		t.Fatal("duplicate confirm", e)
	}
	source, e = r.Statement(ctx, source.ID)
	if e != nil || source.Plain != nil || source.DeletedAt == nil {
		t.Fatal("retention kept raw state", e)
	}
	var stored []byte
	_ = p.QueryRow(ctx, "SELECT plain_state FROM statement_sources WHERE id=$1", source.ID).Scan(&stored)
	if len(stored) != 0 && string(stored) != "null" {
		t.Fatal("raw state survived deletion")
	}
}
