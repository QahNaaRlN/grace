//go:build integration

package imports

import (
	"context"
	"github.com/QahNaaRlN/grace/internal/platform/access"
	"github.com/QahNaaRlN/grace/internal/platform/database"
	"github.com/QahNaaRlN/grace/internal/receipts"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/QahNaaRlN/grace/internal/statementdata"
)

// Dedicated statement fixtures include append-only evidence and deliberately
// block downgrade. Keep legacy JSON/CSV rollback suites independent.
func statementSetup(t *testing.T) (context.Context, *pgxpool.Pool, *Repository) {
	t.Helper()
	if os.Getenv("TEST_BANK_STATEMENTS") != "true" && os.Getenv("TEST_STATEMENT_RESTORE") != "true" {
		t.Skip("dedicated statement fixture database required")
	}
	return setup(t)
}

type staticPDFExtractor struct{}

func (staticPDFExtractor) Extract(context.Context, []byte) (statementdata.Document, error) {
	values := []string{"19.10.2026", "-12.34", "RUB", "Synthetic merchant", "Available balance", "777.00"}
	d := statementdata.Document{Extractor: statementdata.ExtractorVersion, Pages: []statementdata.Page{{Number: 1, Width: 600, Height: 800}}}
	for i, v := range values {
		d.Spans = append(d.Spans, statementdata.Span{ID: []string{"date", "amount", "currency", "merchant", "available-label", "available"}[i], Page: 1, Block: i + 1, Line: 1, Word: 1, Box: statementdata.Box{XMin: 1, YMin: float64(i * 20), XMax: 100, YMax: float64(i*20 + 10)}, Text: v})
	}
	return d, nil
}
func manualStatementProposal() statementdata.Proposal {
	return statementdata.Proposal{Rows: []statementdata.RowMapping{{OperationDate: statementdata.Selection{"date"}, AccountAmount: statementdata.Selection{"amount"}, AccountCurrency: statementdata.Selection{"currency"}, Description: statementdata.Selection{"merchant"}}}, Header: statementdata.HeaderMapping{AvailableBalanceObserved: statementdata.Selection{"available"}, Currency: statementdata.Selection{"currency"}, Labels: map[string]statementdata.Selection{"availableBalanceObserved": {"available-label"}}}}
}
func TestStatementOfflineLifecycle(t *testing.T) {
	ctx, p, r := statementSetup(t)
	r.SetPDFExtractor(staticPDFExtractor{})
	count := func() int {
		var n int
		if e := p.QueryRow(ctx, "SELECT count(*) FROM postings").Scan(&n); e != nil {
			t.Fatal(e)
		}
		return n
	}
	before := count()
	s, created, e := r.UploadStatement(ctx, []byte("%PDF-synthetic-lifecycle"), accountID, "synthetic.pdf", "pdf-upload")
	if e != nil || !created {
		t.Fatal(s, e)
	}
	retry, created, e := r.UploadStatement(ctx, []byte("%PDF-synthetic-lifecycle"), accountID, "renamed.pdf", "pdf-upload")
	if e != nil || created || retry.ID != s.ID {
		t.Fatal("upload replay", e)
	}
	s, e = r.ManualStatementDraft(ctx, s.ID, "manual", StatementManualDraft{ExpectedRevision: s.Revision, Proposal: manualStatementProposal(), Reason: "Synthetic test"})
	if e != nil || s.BatchID == nil {
		t.Fatal(s, e)
	}
	if s.Review.Header.ClosingBookBalance != nil || s.Review.Reconciliation.BookBalance.Status != "not_provided" {
		t.Fatal("available treated as book balance")
	}
	batch, e := r.Batch(ctx, *s.BatchID)
	if e != nil {
		t.Fatal(e)
	}
	records, e := r.Records(ctx, batch.ID, "", "", "", 100)
	if e != nil || len(records.Items) != 1 {
		t.Fatal(records, e)
	}
	rec := records.Items[0]
	if rec.Source.PDF == nil || rec.Source.Sheet != "" || rec.Source.Row != 0 {
		t.Fatal("fake spreadsheet source")
	}
	d := createDecision("purchase")
	d.Fields.AmountMinor = 1234
	d.Fields.OccurredAt = ""
	d.Fields.OccurredDate = "2026-10-19"
	batch, e = r.Edit(ctx, batch.ID, rec.ID, "prepare", Edit{batch.Revision, d})
	if e != nil {
		t.Fatal(e)
	}
	blocked, e := r.Preview(ctx, batch.ID, "blocked-owner", Selection{ExpectedRevision: batch.Revision, RecordIDs: []string{rec.ID}})
	if e != nil || len(blocked.Blockers) == 0 {
		t.Fatal("owner coverage skipped", e)
	}
	s, e = r.EditStatement(ctx, s.ID, "owner-review", StatementEdit{ExpectedRevision: s.Revision, AccountID: accountID, Retention: "delete_after_confirmation", RolesConfirmed: true, ReviewedPages: []int{1}, Reason: "Checked every field/page"})
	if e != nil {
		t.Fatal(e)
	}
	batch, e = r.Batch(ctx, batch.ID)
	if e != nil {
		t.Fatal(e)
	}
	preview, e := r.Preview(ctx, batch.ID, "ready", Selection{ExpectedRevision: batch.Revision, RecordIDs: []string{rec.ID}})
	if e != nil || len(preview.Blockers) > 0 {
		t.Fatal(preview.Blockers, e)
	}
	if count() != before {
		t.Fatal("pre-confirmation financial write")
	}
	// Force a late source-cleanup SQL failure after the financial writer ran.
	// The event, confirmation, provenance and original must all roll back.
	if _, e = p.Exec(ctx, `CREATE FUNCTION statement_test_reject_cleanup() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN IF NEW.id::text=TG_ARGV[0] AND NEW.deleted_at IS NOT NULL THEN RAISE EXCEPTION 'synthetic cleanup failure'; END IF;RETURN NEW;END $$;`); e != nil {
		t.Fatal(e)
	}
	if _, e = p.Exec(ctx, "CREATE TRIGGER statement_test_reject_cleanup BEFORE UPDATE ON statement_sources FOR EACH ROW EXECUTE FUNCTION statement_test_reject_cleanup('"+s.ID+"')"); e != nil {
		t.Fatal(e)
	}
	_, e = r.Confirm(ctx, batch.ID, "confirm-pdf", Confirmation{preview.ID, preview.Hash})
	if e == nil || count() != before {
		t.Fatal("late SQL failure did not roll back ledger", e)
	}
	var retained bool
	if e = p.QueryRow(ctx, "SELECT content IS NOT NULL FROM documents WHERE id=$1", s.DocumentID).Scan(&retained); e != nil || !retained {
		t.Fatal("failed confirmation deleted original", e)
	}
	if _, e = p.Exec(ctx, "DROP TRIGGER statement_test_reject_cleanup ON statement_sources; DROP FUNCTION statement_test_reject_cleanup()"); e != nil {
		t.Fatal(e)
	}
	receipt, e := r.Confirm(ctx, batch.ID, "confirm-pdf", Confirmation{preview.ID, preview.Hash})
	if e != nil || count() != before+1 {
		t.Fatal(receipt, e)
	}
	replay, e := r.Confirm(ctx, batch.ID, "confirm-pdf", Confirmation{preview.ID, preview.Hash})
	if e != nil || replay.ID != receipt.ID || count() != before+1 {
		t.Fatal("confirm replay", e)
	}
	s, e = r.Statement(ctx, s.ID)
	if e != nil || s.Layout != nil || s.DeletedAt == nil || s.Review.Rows[0].Evidence[0].Literal == "" {
		t.Fatal("retention/provenance", s, e)
	}
	_, _, e = r.StatementOriginal(ctx, s.ID)
	requireCode(t, e, "statement_original_gone")
	receiptStore := receipts.NewRepository(p)
	purchaseReceipt, _, e := receiptStore.Create(ctx, receipts.Command{PurchaseEventID: *receipt.Outcomes[0].EventID, Seller: "Synthetic statement merchant", IssuedDate: "2026-10-19", Items: []receipts.ItemInput{{Name: "Synthetic item", QuantityMilli: 1000, UnitPriceMinor: 1234, LineTotalMinor: 1234}}, TotalMinor: 1234, IdempotencyKey: "statement-receipt", RequestHash: "synthetic-statement-receipt"})
	if e != nil {
		t.Fatal(e)
	}
	receiptDigest := digest(purchaseReceipt)
	// Another PDF representing the same Posting must offer the existing event.
	s2, _, e := r.UploadStatement(ctx, []byte("%PDF-regenerated-lifecycle"), accountID, "again.pdf", "pdf-again")
	if e != nil {
		t.Fatal(e)
	}
	s2, e = r.ManualStatementDraft(ctx, s2.ID, "manual-again", StatementManualDraft{ExpectedRevision: s2.Revision, Proposal: manualStatementProposal(), Reason: "Regenerated test"})
	if e != nil {
		t.Fatal(e)
	}
	page, e := r.Records(ctx, *s2.BatchID, "", "", "", 100)
	if e != nil {
		t.Fatal(e)
	}
	full, e := r.Record(ctx, *s2.BatchID, page.Items[0].ID)
	if e != nil || len(full.Candidates) != 1 || full.Candidates[0].EventID != *receipt.Outcomes[0].EventID {
		t.Fatal("Posting match lost", full.Candidates, e)
	}
	// A proposed new Posting with a changed amount is blocked even if the
	// owner selected a supported kind and supplied a review reason.
	batch2, e := r.Batch(ctx, *s2.BatchID)
	if e != nil {
		t.Fatal(e)
	}
	wrong := createDecision("purchase")
	wrong.Fields.AmountMinor = 1235
	batch2, e = r.Edit(ctx, batch2.ID, full.ID, "wrong-posting", Edit{batch2.Revision, wrong})
	if e != nil {
		t.Fatal(e)
	}
	bad, e := r.Preview(ctx, batch2.ID, "wrong-posting-preview", Selection{ExpectedRevision: batch2.Revision, RecordIDs: []string{full.ID}})
	if e != nil || !strings.Contains(strings.Join(bad.Blockers, "|"), "statement_posting_mismatch") {
		t.Fatal("changed Posting was eligible", bad.Blockers, e)
	}
	s2, e = r.EditStatement(ctx, s2.ID, "second-owner-review", StatementEdit{ExpectedRevision: s2.Revision, AccountID: accountID, Retention: "delete_after_confirmation", RolesConfirmed: true, ReviewedPages: []int{1}, Reason: "Same bank fact, regenerated document"})
	if e != nil {
		t.Fatal(e)
	}
	batch2, e = r.Batch(ctx, batch2.ID)
	if e != nil {
		t.Fatal(e)
	}
	batch2, e = r.Edit(ctx, batch2.ID, full.ID, "link-receipt-purchase", Edit{batch2.Revision, Decision{Action: "link_existing", EventID: *receipt.Outcomes[0].EventID, Reason: "Same Posting already has confirmed Receipt"}})
	if e != nil {
		t.Fatal(e)
	}
	linkedPreview, e := r.Preview(ctx, batch2.ID, "link-receipt-preview", Selection{ExpectedRevision: batch2.Revision, RecordIDs: []string{full.ID}})
	if e != nil || len(linkedPreview.Blockers) > 0 || len(linkedPreview.Changes) > 0 {
		t.Fatal(linkedPreview, e)
	}
	linked, e := r.Confirm(ctx, batch2.ID, "link-receipt-confirm", Confirmation{linkedPreview.ID, linkedPreview.Hash})
	if e != nil || count() != before+1 || *linked.Outcomes[0].EventID != *receipt.Outcomes[0].EventID {
		t.Fatal("linked import duplicated expense", linked, e)
	}
	preserved, e := receiptStore.Get(ctx, purchaseReceipt.ID)
	if e != nil || digest(preserved) != receiptDigest {
		t.Fatal("linked import rewrote Receipt", e)
	}
	// Expiry removes full state even when no draft was ever generated.
	s3, _, e := r.UploadStatement(ctx, []byte("%PDF-expiry-lifecycle"), accountID, "expiry.pdf", "pdf-expiry")
	if e != nil {
		t.Fatal(e)
	}
	r.now = func() time.Time { return time.Now().Add(8 * 24 * time.Hour) }
	s3, e = r.Statement(ctx, s3.ID)
	r.now = time.Now
	if e != nil || s3.Status != "expired" || s3.Layout != nil {
		t.Fatal(s3, e)
	}
}

func TestStatementIsolationRegroupAndConcurrentManual(t *testing.T) {
	ctx, p, r := statementSetup(t)
	r.SetPDFExtractor(staticPDFExtractor{})
	s, _, e := r.UploadStatement(ctx, []byte("%PDF-statement-isolation"), accountID, "isolation.pdf", "statement-isolation")
	if e != nil {
		t.Fatal(e)
	}
	var otherID string
	if e = p.QueryRow(ctx, "INSERT INTO users(display_name,role) VALUES('Statement other admin','admin') RETURNING id::text").Scan(&otherID); e != nil {
		t.Fatal(e)
	}
	other := access.WithPrincipal(context.Background(), access.Principal{ID: otherID, Role: "admin"})
	_, e = r.Statement(other, s.ID)
	requireCode(t, e, "not_found")
	_, _, e = r.StatementOriginal(other, s.ID)
	requireCode(t, e, "not_found")
	_, e = r.DeleteStatementOriginal(other, s.ID, "foreign-delete", StatementDelete{ExpectedRevision: s.Revision})
	requireCode(t, e, "not_found")
	tenant := database.ForUsers(p)
	tx, e := tenant.BeginTx(other, pgx.TxOptions{})
	if e != nil {
		t.Fatal(e)
	}
	for _, table := range []string{"statement_sources", "statement_audit", "documents"} {
		var n int
		if e = tx.QueryRow(other, "SELECT count(*) FROM "+table).Scan(&n); e != nil || n != 0 {
			t.Fatal("foreign admin read", table, n, e)
		}
	}
	_, e = tx.Exec(other, "INSERT INTO statement_sources(document_id,account_id,layout,page_count,original_size_bytes) VALUES($1,$2,'{}',1,20)", s.DocumentID, accountID)
	_ = tx.Rollback(other)
	if e == nil {
		t.Fatal("cross-owner FK")
	}
	router := chi.NewRouter()
	NewHandler(r).Routes(router)
	for _, suffix := range []string{"", "/original"} {
		request := httptest.NewRequest("GET", "/statements/"+s.ID+suffix, nil).WithContext(other)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, request)
		if w.Code != 404 {
			t.Fatal("foreign HTTP source", w.Code)
		}
	}
	// Same-key writers serialize and return the same persisted batch.
	input := StatementManualDraft{ExpectedRevision: s.Revision, Proposal: manualStatementProposal(), Reason: "Concurrent synthetic grouping"}
	type result struct {
		s Statement
		e error
	}
	ch := make(chan result, 2)
	for i := 0; i < 2; i++ {
		go func() { got, e := r.ManualStatementDraft(ctx, s.ID, "same-manual", input); ch <- result{got, e} }()
	}
	a, b := <-ch, <-ch
	if a.e != nil || b.e != nil || a.s.BatchID == nil || b.s.BatchID == nil || *a.s.BatchID != *b.s.BatchID {
		t.Fatal("parallel materialization", a.e, b.e)
	}
	s = a.s
	oldBatch := *s.BatchID
	s, e = r.ManualStatementDraft(ctx, s.ID, "regroup", StatementManualDraft{ExpectedRevision: s.Revision, Proposal: manualStatementProposal(), Reason: "Rechecked all source roles"})
	if e != nil || s.BatchID == nil || *s.BatchID != oldBatch || s.RolesConfirmed {
		t.Fatal("regroup lost batch or retained trust", s, e)
	}
	var archived int
	if e = p.QueryRow(ctx, "SELECT count(*) FROM statement_audit WHERE statement_id=$1 AND action='regroup_archive'", s.ID).Scan(&archived); e != nil || archived != 1 {
		t.Fatal("regroup history lost", archived, e)
	}
	// An original from the shared document table cannot be erased as if it were
	// a statement; exact content and kind remain protected by DB constraints.
	var terms string
	if e = p.QueryRow(ctx, "INSERT INTO documents(owner_id,kind,filename,sha256,content) VALUES($1,'credit_card_terms','terms.pdf',repeat('f',64),'%PDF-terms') RETURNING id::text", access.BootstrapOwnerID).Scan(&terms); e != nil {
		t.Fatal(e)
	}
	tx, e = tenant.BeginTx(ctx, pgx.TxOptions{})
	if e != nil {
		t.Fatal(e)
	}
	_, e = tx.Exec(ctx, "UPDATE documents SET content=NULL,original_deleted_at=now() WHERE id=$1", terms)
	_ = tx.Rollback(ctx)
	if e == nil {
		t.Fatal("terms original was deletable")
	}
	// Keep suppresses expiry; changing back to default restores the original TTL.
	s, e = r.EditStatement(ctx, s.ID, "keep", StatementEdit{ExpectedRevision: s.Revision, AccountID: accountID, Retention: "keep", RolesConfirmed: true, ReviewedPages: []int{1}, Reason: "Explicit synthetic retention"})
	if e != nil {
		t.Fatal(e)
	}
	r.now = func() time.Time { return time.Now().Add(8 * 24 * time.Hour) }
	kept, e := r.Statement(ctx, s.ID)
	r.now = time.Now
	if e != nil || kept.DeletedAt != nil {
		t.Fatal("keep ignored", e)
	}
}

func TestStatementRestoredEvidence(t *testing.T) {
	if os.Getenv("TEST_STATEMENT_RESTORE") != "true" {
		t.Skip("restored schema required")
	}
	ctx, p, r := statementSetup(t)
	var id string
	if e := p.QueryRow(ctx, "SELECT id::text FROM statement_sources WHERE deleted_at IS NOT NULL AND review IS NOT NULL LIMIT 1").Scan(&id); e != nil {
		t.Fatal(e)
	}
	s, e := r.Statement(ctx, id)
	if e != nil || s.Review == nil || s.Layout != nil || s.SizeBytes == 0 || s.PublishedRows != len(s.Review.Rows) {
		t.Fatal("restored evidence lost", s, e)
	}
	_, _, e = r.StatementOriginal(ctx, id)
	requireCode(t, e, "statement_original_gone")
	var fragments int
	if e = p.QueryRow(ctx, "SELECT count(*) FROM import_records WHERE batch_id=$1 AND source->'pdf'->'row'->'evidence' IS NOT NULL", *s.BatchID).Scan(&fragments); e != nil || fragments == 0 {
		t.Fatal("restored provenance missing", e)
	}
}

type documentExtractor struct{ doc statementdata.Document }

func (e documentExtractor) Extract(context.Context, []byte) (statementdata.Document, error) {
	return e.doc, nil
}
func TestStatementPartialConfirmationAndDirectSQLCleanup(t *testing.T) {
	ctx, p, r := statementSetup(t)
	doc, e := (staticPDFExtractor{}).Extract(ctx, nil)
	if e != nil {
		t.Fatal(e)
	}
	doc.Spans[1].Text = "-67.89"
	for i, value := range []string{"20.10.2026", "+98.76", "Synthetic own funds"} {
		doc.Spans = append(doc.Spans, statementdata.Span{ID: []string{"date-two", "amount-two", "merchant-two"}[i], Page: 1, Block: i + 10, Line: 1, Word: 1, Box: statementdata.Box{XMin: 1, YMin: float64(200 + i*20), XMax: 100, YMax: float64(210 + i*20)}, Text: value})
	}
	r.SetPDFExtractor(documentExtractor{doc})
	s, _, e := r.UploadStatement(ctx, []byte("%PDF-partial-statement"), accountID, "partial.pdf", "partial-upload")
	if e != nil {
		t.Fatal(e)
	}
	proposal := manualStatementProposal()
	proposal.Rows = append(proposal.Rows, statementdata.RowMapping{OperationDate: statementdata.Selection{"date-two"}, AccountAmount: statementdata.Selection{"amount-two"}, AccountCurrency: statementdata.Selection{"currency"}, Description: statementdata.Selection{"merchant-two"}})
	s, e = r.ManualStatementDraft(ctx, s.ID, "partial-draft", StatementManualDraft{ExpectedRevision: s.Revision, Proposal: proposal, Reason: "Two synthetic rows"})
	if e != nil {
		t.Fatal(e)
	}
	s, e = r.EditStatement(ctx, s.ID, "partial-review", StatementEdit{ExpectedRevision: s.Revision, AccountID: accountID, Retention: "delete_after_confirmation", RolesConfirmed: true, ReviewedPages: []int{1}, Reason: "Checked both rows and whole page"})
	if e != nil {
		t.Fatal(e)
	}
	batch, e := r.Batch(ctx, *s.BatchID)
	if e != nil {
		t.Fatal(e)
	}
	page, e := r.Records(ctx, batch.ID, "", "", "", 100)
	if e != nil {
		t.Fatal(e)
	}
	first, second := page.Items[0], page.Items[1]
	if first.Source.PDF.Ordinal != 1 {
		first, second = second, first
	}
	next, e := r.NextUnresolved(ctx, batch.ID, first.ID)
	if e != nil || next.Record == nil || next.Record.ID != second.ID {
		t.Fatal("PDF ordinal next navigation", next, e)
	}
	decision := createDecision("purchase")
	decision.Fields.AmountMinor = 6789
	decision.Fields.OccurredAt = ""
	decision.Fields.OccurredDate = "2026-10-19"
	batch, e = r.Edit(ctx, batch.ID, first.ID, "partial-first", Edit{batch.Revision, decision})
	if e != nil {
		t.Fatal(e)
	}
	preview, e := r.Preview(ctx, batch.ID, "partial-preview-one", Selection{ExpectedRevision: batch.Revision, RecordIDs: []string{first.ID}})
	if e != nil || len(preview.Blockers) > 0 {
		t.Fatal(preview, e)
	}
	if _, e = r.Confirm(ctx, batch.ID, "partial-confirm-one", Confirmation{preview.ID, preview.Hash}); e != nil {
		t.Fatal(e)
	}
	s, e = r.Statement(ctx, s.ID)
	if e != nil || s.DeletedAt != nil || s.Layout == nil || s.PublishedRows != 1 || s.Review.Reconciliation.FinancialCoverage.Status != "incomplete" {
		t.Fatal("partial confirmation discarded original", s, e)
	}
	batch, e = r.Batch(ctx, batch.ID)
	if e != nil {
		t.Fatal(e)
	}
	decision = createDecision("income_received")
	decision.Fields.AmountMinor = 9876
	incomeCategory := "20000000-0000-0000-0000-000000000004"
	decision.Fields.CategoryID = &incomeCategory
	decision.Fields.OccurredAt = ""
	decision.Fields.OccurredDate = "2026-10-20"
	batch, e = r.Edit(ctx, batch.ID, second.ID, "partial-second", Edit{batch.Revision, decision})
	if e != nil {
		t.Fatal(e)
	}
	preview, e = r.Preview(ctx, batch.ID, "partial-preview-two", Selection{ExpectedRevision: batch.Revision, RecordIDs: []string{second.ID}})
	if e != nil || len(preview.Blockers) > 0 {
		t.Fatal(preview, e)
	}
	type result struct {
		receipt Receipt
		err     error
	}
	out := make(chan result, 2)
	for i := 0; i < 2; i++ {
		go func() {
			receipt, e := r.Confirm(ctx, batch.ID, "partial-confirm-two", Confirmation{preview.ID, preview.Hash})
			out <- result{receipt, e}
		}()
	}
	a, b := <-out, <-out
	if a.err != nil || b.err != nil || a.receipt.ID != b.receipt.ID {
		t.Fatal("concurrent confirmation", a.err, b.err)
	}
	s, e = r.Statement(ctx, s.ID)
	if e != nil || s.DeletedAt == nil || s.Layout != nil || s.Review.Reconciliation.FinancialCoverage.Status != "represented" {
		t.Fatal("last confirmation did not clean original", s, e)
	}
	// Direct SQL cannot delete only the metadata or retain full extracted text.
	r.SetPDFExtractor(staticPDFExtractor{})
	direct, _, e := r.UploadStatement(ctx, []byte("%PDF-direct-sql-cleanup"), accountID, "direct.pdf", "direct-upload")
	if e != nil {
		t.Fatal(e)
	}
	tx, e := r.begin(ctx, true)
	if e != nil {
		t.Fatal(e)
	}
	_, e = tx.Exec(ctx, "UPDATE statement_sources SET layout=NULL,deleted_at=now(),status='deleted',revision=revision+1 WHERE id=$1", direct.ID)
	_ = tx.Rollback(ctx)
	if e == nil {
		t.Fatal("metadata-only deletion hid original bytes")
	}
	tx, e = r.begin(ctx, true)
	if e != nil {
		t.Fatal(e)
	}
	_, e = tx.Exec(ctx, "UPDATE documents SET content=NULL,original_deleted_at=now() WHERE id=$1", direct.DocumentID)
	if e != nil {
		_ = tx.Rollback(ctx)
		t.Fatal(e)
	}
	if e = tx.Commit(ctx); e != nil {
		t.Fatal(e)
	}
	direct, e = r.Statement(ctx, direct.ID)
	if e != nil || direct.Layout != nil || direct.DeletedAt == nil {
		t.Fatal("direct deletion retained text", direct, e)
	}
	var original []byte
	if e = p.QueryRow(ctx, "SELECT content FROM documents WHERE id=$1", direct.DocumentID).Scan(&original); e != nil || original != nil {
		t.Fatal("original bytes retained", e)
	}
}
