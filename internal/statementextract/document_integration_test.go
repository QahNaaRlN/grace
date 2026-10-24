//go:build integration

package statementextract_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/QahNaaRlN/grace/internal/documentdata"
	"github.com/QahNaaRlN/grace/internal/imports"
	"github.com/QahNaaRlN/grace/internal/pdfextract"
	"github.com/QahNaaRlN/grace/internal/platform/access"
	"github.com/QahNaaRlN/grace/internal/platform/database"
	"github.com/QahNaaRlN/grace/internal/statementdata"
	"github.com/QahNaaRlN/grace/internal/statementextract"
	"github.com/jackc/pgx/v5/pgxpool"
	"os"
	"strings"
	"testing"
)

type s56Reader struct {
	reading   statementdata.VisualReading
	calls     int
	unknown   bool
	preflight bool
}

func (r *s56Reader) CurrentDocumentPolicy(context.Context, int) (statementextract.DocumentPolicy, error) {
	return statementextract.DocumentPolicy{Model: "fixture/document", Provider: "fixture/eu", ProviderName: "Fixture", Profile: "document_reading", GroupRevision: "test-v1", ProofSHA256: strings.Repeat("a", 64), MaximumMicros: 10000, Snapshot: json.RawMessage(`{"fixture":true}`)}, nil
}
func (r *s56Reader) PrepareDocument(_ context.Context, _ statementextract.DocumentPolicy, doc documentdata.Document) error {
	if r.preflight {
		return errors.New("document_limit")
	}
	return doc.Validate()
}
func (r *s56Reader) CompleteDocument(_ context.Context, _ statementextract.DocumentPolicy, doc documentdata.Document) (statementdata.VisualReading, *statementextract.Billing, error) {
	r.calls++
	if r.unknown {
		return statementdata.VisualReading{}, nil, errors.New("provider_unavailable")
	}
	if doc.Validate() != nil {
		return statementdata.VisualReading{}, nil, errors.New("document_limit")
	}
	return r.reading, &statementextract.Billing{GenerationID: fmt.Sprintf("fixture-%d", r.calls), CostMicros: 100, InputTokens: 500, OutputTokens: 500}, nil
}
func s56Reading(t *testing.T, name string) statementdata.VisualReading {
	t.Helper()
	raw, e := os.ReadFile("testdata/direct-page-reading/" + name + ".golden.json")
	if e != nil {
		t.Fatal(e)
	}
	var v statementdata.VisualReading
	if json.Unmarshal(raw, &v) != nil {
		t.Fatal("golden invalid")
	}
	fix := func(qs []statementdata.ImageQuote) {
		for i := range qs {
			qs[i].Box = [4]float64{0, 0, 1, 1}
		}
	}
	for _, f := range v.Header {
		if f != nil {
			fix(f.Evidence)
		}
	}
	for _, r := range v.Rows {
		fix(r.Anchor)
		for _, f := range r.Fields {
			if f != nil {
				fix(f.Evidence)
			}
		}
	}
	return v
}
func s56Pool(t *testing.T) (context.Context, *pgxpool.Pool) {
	t.Helper()
	if os.Getenv("TEST_S56_DOCUMENTS") != "true" {
		t.Skip("dedicated disposable database required")
	}
	p, e := pgxpool.New(context.Background(), os.Getenv("TEST_DATABASE_URL"))
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(p.Close)
	return context.Background(), p
}
func s56Owner(t *testing.T, p *pgxpool.Pool) (context.Context, string, string) {
	t.Helper()
	var owner, account, counter string
	if e := p.QueryRow(context.Background(), "INSERT INTO users(display_name,role) VALUES('S56 synthetic owner','member') RETURNING id::text").Scan(&owner); e != nil {
		t.Fatal(e)
	}
	ctx := access.WithPrincipal(context.Background(), access.Principal{ID: owner, Role: "member"})
	tenant := database.ForUsers(p)
	for _, target := range []*string{&account, &counter} {
		if e := tenant.QueryRow(ctx, "INSERT INTO accounts(name,type,currency,opening_balance_minor) VALUES('S56 synthetic account','debit','RUB',1000000) RETURNING id::text").Scan(target); e != nil {
			t.Fatal(e)
		}
	}
	return ctx, account, counter
}
func TestS56StatementDocumentLifecycle(t *testing.T) {
	_, pool := s56Pool(t)
	if _, e := pool.Exec(context.Background(), "UPDATE ai_experiment_budget SET allowance_micros=1000000"); e != nil {
		t.Fatal(e)
	}
	helper, e := pdfextract.NewClient(os.Getenv("TEST_PDF_EXTRACTOR_SOCKET"))
	if e != nil {
		t.Fatal(e)
	}
	for _, tc := range []struct {
		name, path, golden string
		image              bool
	}{{"photo", "testdata/direct-page-reading/cards/page-1.png", "cards", true}, {"scan", "../pdfextract/testdata/full32-scan.pdf", "full32", false}, {"native", "../pdfextract/testdata/cards.pdf", "cards", false}} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, account, counter := s56Owner(t, pool)
			tenant := database.ForUsers(pool)
			repo := imports.NewRepository(pool)
			repo.SetPDFExtractor(helper)
			count := func() int {
				var n int
				if e := tenant.QueryRow(ctx, "SELECT count(*) FROM postings").Scan(&n); e != nil {
					t.Fatal(e)
				}
				return n
			}
			if count() != 0 {
				t.Fatal("fixture not isolated")
			}
			raw, e := os.ReadFile(tc.path)
			if e != nil {
				t.Fatal(e)
			}
			var source imports.Statement
			if tc.image {
				source, _, e = repo.UploadStatementImage(ctx, raw, account, "synthetic.png", "upload")
			} else {
				source, _, e = repo.UploadStatement(ctx, raw, account, "synthetic.pdf", "upload")
			}
			if e != nil {
				t.Fatal(e)
			}
			if len(source.Pages) == 0 {
				t.Fatal("real helper render missing")
			}
			reading := s56Reading(t, tc.golden)
			reader := &s56Reader{reading: reading}
			service := statementextract.NewService(pool, repo, nil)
			service.SetDocumentReasoner(reader)
			if _, e = service.Preview(ctx, source.ID, statementextract.PreviewInput{ExpectedRevision: source.Revision - 1}); e == nil {
				t.Fatal("stale source accepted")
			}
			reader.preflight = true
			var before int
			_ = tenant.QueryRow(ctx, "SELECT count(*) FROM ai_usage").Scan(&before)
			if _, e = service.Preview(ctx, source.ID, statementextract.PreviewInput{ExpectedRevision: source.Revision}); e == nil {
				t.Fatal("failed preflight accepted")
			}
			var after int
			_ = tenant.QueryRow(ctx, "SELECT count(*) FROM ai_usage").Scan(&after)
			if before != after || reader.calls != 0 {
				t.Fatal("preflight reserved or called")
			}
			reader.preflight = false
			preview, e := service.Preview(ctx, source.ID, statementextract.PreviewInput{ExpectedRevision: source.Revision})
			if e != nil {
				t.Fatal(e)
			}
			if _, e = service.Extract(ctx, source.ID, "no-consent", statementextract.ExtractionInput{ExpectedRevision: source.Revision, PreviewID: preview.ID, PreviewHash: preview.Hash, ConsentID: "00000000-0000-0000-0000-000000000000"}); e == nil || reader.calls != 0 {
				t.Fatal("missing consent called")
			}
			consent, e := service.Consent(ctx, source.ID, "consent", statementextract.ConsentInput{ExpectedRevision: source.Revision, PreviewID: preview.ID, PreviewHash: preview.Hash})
			if e != nil {
				t.Fatal(e)
			}
			in := statementextract.ExtractionInput{ExpectedRevision: source.Revision, PreviewID: preview.ID, PreviewHash: preview.Hash, ConsentID: consent.ID}
			attempt, e := service.Extract(ctx, source.ID, "extract", in)
			if e != nil || attempt.Status != "completed" {
				t.Fatal(attempt, e)
			}
			replay, e := service.Extract(ctx, source.ID, "extract", in)
			if e != nil || replay.ID != attempt.ID || reader.calls != 1 {
				t.Fatal("extraction replay called", e)
			}
			source, e = repo.Statement(ctx, source.ID)
			if e != nil || source.Review == nil || len(source.Review.Rows) != len(reading.Rows) {
				t.Fatal(source, e)
			}
			for i, r := range source.Review.Rows {
				want, _ := statementdata.ParseLocalDate(reading.Rows[i].Fields["operationDate"].Value, "")
				if r.OperationDate == nil || r.OperationDate.Local != want.Local || r.Description != reading.Rows[i].Fields["description"].Value {
					t.Fatal("recognition lost row/precision/description", i)
				}
			}
			var foreign string
			_ = pool.QueryRow(ctx, "INSERT INTO users(display_name,role) VALUES('S56 foreign owner','member') RETURNING id::text").Scan(&foreign)
			foreignCtx := access.WithPrincipal(context.Background(), access.Principal{ID: foreign, Role: "member"})
			if _, e = repo.Statement(foreignCtx, source.ID); e == nil {
				t.Fatal("foreign source visible")
			}
			if _, e = repo.StatementPage(foreignCtx, source.ID, 1); e == nil {
				t.Fatal("foreign image visible")
			}
			batch, e := repo.Batch(ctx, *source.BatchID)
			if e != nil {
				t.Fatal(e)
			}
			page, e := repo.Records(ctx, batch.ID, "", "", "", 100)
			if e != nil {
				t.Fatal(e)
			}
			ids := []string{}
			for _, rec := range page.Items {
				ids = append(ids, rec.ID)
			}
			blocked, e := repo.Preview(ctx, batch.ID, "unreviewed", imports.Selection{ExpectedRevision: batch.Revision, RecordIDs: ids})
			if e != nil || len(blocked.Blockers) == 0 || count() != 0 {
				t.Fatal("unreviewed source importable", blocked, e)
			}
			if source.ReadingMode == "image_review" {
				if source.Visual == nil || source.Layout == nil || len(source.Layout.Spans) != 0 {
					t.Fatal("scan gained fake native spans")
				}
				proof := imports.VisualReviewInput{ExpectedRevision: source.Revision, SourceSHA256: source.SHA256, Rows: source.Visual.RowDigests, ReviewedPages: []int{}, RolesConfirmed: true}
				for i := 1; i <= source.PageCount; i++ {
					proof.ReviewedPages = append(proof.ReviewedPages, i)
				}
				bad := proof
				bad.Rows = map[string]string{"row-1": "forged"}
				if _, e = repo.ReviewVisual(ctx, source.ID, "forged", bad); e == nil {
					t.Fatal("forged visual proof accepted")
				}
				source, e = repo.ReviewVisual(ctx, source.ID, "owner-visual", proof)
				if e != nil {
					t.Fatal(e)
				}
				// An actual edit invalidates its proof, including changing back to the original.
				corrected := reading
				rawCopy, _ := json.Marshal(reading)
				_ = json.Unmarshal(rawCopy, &corrected)
				corrected.Rows[0].Fields["description"].Value += " synthetic correction"
				source, e = repo.ManualStatementDraft(ctx, source.ID, "edit-visual", imports.StatementManualDraft{ExpectedRevision: source.Revision, SourceSHA256: source.SHA256, Reading: &corrected, Reason: "Owner corrected a synthetic field"})
				if e != nil {
					t.Fatal(e)
				}
				if !source.RolesConfirmed || len(source.ReviewedPages) != source.PageCount || len(source.Visual.ReviewedRows) != len(reading.Rows)-1 {
					t.Fatal("edit revoked unchanged review evidence")
				}
				if _, ok := source.Visual.ReviewedRows["row-1"]; ok {
					t.Fatal("edit kept old proof")
				}
				source, e = repo.ManualStatementDraft(ctx, source.ID, "restore-visual", imports.StatementManualDraft{ExpectedRevision: source.Revision, SourceSHA256: source.SHA256, Reading: &reading, Reason: "Owner restored the exact printed description"})
				if e != nil {
					t.Fatal(e)
				}
				proof.ExpectedRevision = source.Revision
				proof.Rows = source.Visual.RowDigests
				source, e = repo.ReviewVisual(ctx, source.ID, "recheck-visual", proof)
				if e != nil {
					t.Fatal(e)
				}
			} else {
				if source.DraftMapping == nil || source.DraftMapping.Proposal == nil {
					t.Fatal("native proposal missing")
				}
				proposal := *source.DraftMapping.Proposal
				// Explicit owner disposition of visible non-operation headings in this fixed
				// synthetic fixture. The production reader never supplies this service repair.
				if len(source.DraftMapping.UnmappedSpanIDs) > 0 {
					proposal.Service = append(proposal.Service, statementdata.ServiceMapping{Spans: source.DraftMapping.UnmappedSpanIDs, Reason: "Owner reviewed synthetic non-operation headings against every printed row"})
				}
				source, e = repo.ManualStatementDraft(ctx, source.ID, "native-owner-mapping", imports.StatementManualDraft{ExpectedRevision: source.Revision, Proposal: proposal, Reason: "Owner reviewed every native source fragment"})
				if e != nil {
					t.Fatal(e)
				}
				pages := []int{}
				for i := 1; i <= source.PageCount; i++ {
					pages = append(pages, i)
				}
				source, e = repo.EditStatement(ctx, source.ID, "native-owner-pages", imports.StatementEdit{ExpectedRevision: source.Revision, AccountID: account, Retention: "delete_after_confirmation", RolesConfirmed: true, ReviewedPages: pages, Reason: "Owner reviewed source pages and roles"})
				if e != nil {
					t.Fatal(e)
				}
			}
			batch, e = repo.Batch(ctx, *source.BatchID)
			if e != nil {
				t.Fatal(e)
			}
			page, e = repo.Records(ctx, batch.ID, "", "", "", 100)
			if e != nil {
				t.Fatal(e)
			}
			ids = []string{}
			expectedDelta := int64(0)
			expectedPosts := len(page.Items)
			expense, income := "20000000-0000-0000-0000-000000000001", "20000000-0000-0000-0000-000000000004"
			for _, rec := range page.Items {
				row := rec.Source.PDF.Row
				f := &imports.Fields{Kind: "purchase", AccountID: account, CategoryID: &expense, Currency: "RUB", AmountMinor: row.AccountAmount.AmountMinor, OccurredDate: row.OperationDate.Local[:10], Description: row.Description, TimeReason: "Only the printed local date is projected; timezone unknown"}
				expectedDelta -= f.AmountMinor
				if row.Direction == "credit" {
					f.Kind = "income_received"
					f.CategoryID = &income
					expectedDelta += 2 * f.AmountMinor
					if strings.Contains(strings.ToLower(row.Description), "перевод") || strings.Contains(strings.ToLower(row.Description), "transfer") {
						f.Kind = "internal_transfer"
						f.AccountID = counter
						f.DestinationAccountID = account
						f.CategoryID = nil
						expectedPosts++
					}
				}
				decision := imports.Decision{Action: "create", Reason: "Owner selected semantics from the synthetic source", Fields: f}
				detail, detailErr := repo.Record(ctx, batch.ID, rec.ID)
				if detailErr != nil {
					t.Fatal(detailErr)
				}
				for _, c := range detail.Candidates {
					decision.Resolutions = append(decision.Resolutions, imports.Resolution{CandidateID: c.ID, Reason: "Owner checked distinct synthetic operation"})
				}
				batch, e = repo.Edit(ctx, batch.ID, rec.ID, "decision-"+rec.ID, imports.Edit{ExpectedRevision: batch.Revision, Decision: decision})
				if e != nil {
					t.Fatal(e)
				}
				ids = append(ids, rec.ID)
			}
			ready, e := repo.Preview(ctx, batch.ID, "ready", imports.Selection{ExpectedRevision: batch.Revision, RecordIDs: ids})
			if e != nil || len(ready.Blockers) > 0 {
				t.Fatal("reviewed result blocked", ready.Blockers, e)
			}
			if count() != 0 {
				t.Fatal("pre-confirmation write")
			}
			receipt, e := repo.Confirm(ctx, batch.ID, "confirm", imports.Confirmation{PreviewID: ready.ID, PreviewHash: ready.Hash})
			if e != nil || count() != expectedPosts {
				t.Fatal(receipt, e, count(), expectedPosts)
			}
			again, e := repo.Confirm(ctx, batch.ID, "confirm", imports.Confirmation{PreviewID: ready.ID, PreviewHash: ready.Hash})
			if e != nil || again.ID != receipt.ID || count() != expectedPosts {
				t.Fatal("confirmation duplicated", e)
			}
			var actualDelta int64
			if e = tenant.QueryRow(ctx, "SELECT COALESCE(sum(CASE WHEN direction='debit' THEN -amount_minor ELSE amount_minor END),0) FROM postings WHERE account_id=$1", account).Scan(&actualDelta); e != nil || actualDelta != expectedDelta {
				t.Fatal("wrong source account delta", actualDelta, expectedDelta, e)
			}
			retained, e := repo.Statement(ctx, source.ID)
			if e != nil || retained.DeletedAt == nil || len(retained.Pages) != 0 || retained.Review == nil {
				t.Fatal("retention lost evidence or kept render", e)
			}
			if _, e = repo.StatementPage(ctx, source.ID, 1); e == nil {
				t.Fatal("deleted source page visible")
			}
			var duplicate imports.Statement
			var created bool
			if tc.image {
				duplicate, created, e = repo.UploadStatementImage(ctx, raw, account, "renamed.png", "repeat-upload")
			} else {
				duplicate, created, e = repo.UploadStatement(ctx, raw, account, "renamed.pdf", "repeat-upload")
			}
			if e != nil || created || duplicate.ID != source.ID || count() != expectedPosts || reader.calls != 1 {
				t.Fatal("repeat file created source/provider/financial duplicates", e)
			}
		})
	}
}

func TestS56UnknownBillingFreezesAndDeletedSourceCannotRun(t *testing.T) {
	_, pool := s56Pool(t)
	ctx, account, _ := s56Owner(t, pool)
	repo := imports.NewRepository(pool)
	raw, e := os.ReadFile("testdata/direct-page-reading/cards/page-1.png")
	if e != nil {
		t.Fatal(e)
	}
	source, _, e := repo.UploadStatementImage(ctx, raw, account, "synthetic.png", "upload-unknown")
	if e != nil {
		t.Fatal(e)
	}
	reader := &s56Reader{unknown: true}
	service := statementextract.NewService(pool, repo, nil)
	service.SetDocumentReasoner(reader)
	preview, e := service.Preview(ctx, source.ID, statementextract.PreviewInput{ExpectedRevision: source.Revision})
	if e != nil {
		t.Fatal(e)
	}
	consent, e := service.Consent(ctx, source.ID, "unknown-consent", statementextract.ConsentInput{ExpectedRevision: source.Revision, PreviewID: preview.ID, PreviewHash: preview.Hash})
	if e != nil {
		t.Fatal(e)
	}
	in := statementextract.ExtractionInput{ExpectedRevision: source.Revision, PreviewID: preview.ID, PreviewHash: preview.Hash, ConsentID: consent.ID}
	attempt, e := service.Extract(ctx, source.ID, "unknown-extract", in)
	if e != nil || attempt.Status != "uncertain" || reader.calls != 1 {
		t.Fatal(attempt, e, reader.calls)
	}
	if _, e = service.Extract(ctx, source.ID, "unknown-extract", in); e != nil || reader.calls != 1 {
		t.Fatal("uncertain replay called", e)
	}
	var frozen bool
	if e = pool.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM ai_usage WHERE status='uncertain')").Scan(&frozen); e != nil || !frozen {
		t.Fatal("unknown bill lost", e)
	}
	if _, e = service.Extract(ctx, source.ID, "new-key-after-unknown", in); e == nil || reader.calls != 1 {
		t.Fatal("unknown bill allowed retry", e)
	}
	var posts int
	if e = database.ForUsers(pool).QueryRow(ctx, "SELECT count(*) FROM postings").Scan(&posts); e != nil || posts != 0 {
		t.Fatal(posts, e)
	}
	if _, e = repo.DeleteStatementOriginal(ctx, source.ID, "delete-uncertain-source", imports.StatementDelete{ExpectedRevision: source.Revision}); e != nil {
		t.Fatal(e)
	}
	if _, e = repo.StatementPage(ctx, source.ID, 1); e == nil {
		t.Fatal("deleted page visible")
	}
	if _, e = service.Preview(ctx, source.ID, statementextract.PreviewInput{ExpectedRevision: source.Revision}); e == nil || reader.calls != 1 {
		t.Fatal("deleted source usable", e)
	}
}
