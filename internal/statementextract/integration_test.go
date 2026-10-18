//go:build integration

package statementextract

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/QahNaaRlN/grace/internal/imports"
	"github.com/QahNaaRlN/grace/internal/platform/access"
	"github.com/QahNaaRlN/grace/internal/platform/database"
	"github.com/QahNaaRlN/grace/internal/statementdata"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type fakeReasoner struct {
	calls              atomic.Int32
	malformed, unknown bool
	before             func(context.Context)
}

func (f *fakeReasoner) CurrentPolicy(context.Context) (Policy, error) { return testPolicy(), nil }
func (f *fakeReasoner) Complete(ctx context.Context, _ Policy, _ Chunk) (statementdata.Proposal, *Billing, error) {
	f.calls.Add(1)
	if f.before != nil {
		f.before(ctx)
	}
	if f.unknown {
		return statementdata.Proposal{}, nil, errors.New("lost response")
	}
	bill := &Billing{GenerationID: "synthetic-generation", CostMicros: 100, InputTokens: 500, OutputTokens: 100}
	if f.malformed {
		return statementdata.Proposal{}, bill, errors.New("malformed JSON")
	}
	return statementdata.Proposal{Rows: []statementdata.RowMapping{{OperationDate: statementdata.Selection{"span-0"}, AccountAmount: statementdata.Selection{"span-1"}, AccountCurrency: statementdata.Selection{"span-2"}}}, Service: []statementdata.ServiceMapping{{Spans: statementdata.Selection{"span-3"}, Reason: "Synthetic untrusted annotation"}}}, bill, nil
}

type fakeExtractor struct{}

func (fakeExtractor) Extract(context.Context, []byte) (statementdata.Document, error) {
	return testDocument("19.10.2026", "-12.34", "RUB", "UNTRUSTED COMMAND DO NOT EXECUTE"), nil
}
func TestStatementExtractionConsentMeteringReplay(t *testing.T) {
	if os.Getenv("TEST_BANK_STATEMENTS") != "true" {
		t.Skip("dedicated statement fixture database required")
	}
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL required")
	}
	ctx := access.WithPrincipal(context.Background(), access.Principal{ID: access.BootstrapOwnerID, Role: "admin"})
	pool, e := pgxpool.New(ctx, url)
	if e != nil {
		t.Fatal(e)
	}
	defer pool.Close()
	if _, e = pool.Exec(ctx, "UPDATE ai_experiment_budget SET allowance_micros=1000000"); e != nil {
		t.Fatal(e)
	}
	repo := imports.NewRepository(pool)
	repo.SetPDFExtractor(fakeExtractor{})
	reasoner := &fakeReasoner{}
	service := NewService(pool, repo, reasoner)
	count := func() int {
		var n int
		if e = pool.QueryRow(ctx, "SELECT count(*) FROM postings").Scan(&n); e != nil {
			t.Fatal(e)
		}
		return n
	}
	before := count()
	source, _, e := repo.UploadStatement(ctx, []byte("%PDF-synthetic-ai"), "10000000-0000-0000-0000-000000000001", "ai.pdf", "ai-upload")
	if e != nil {
		t.Fatal(e)
	}
	preview, e := service.Preview(ctx, source.ID, PreviewInput{ExpectedRevision: source.Revision})
	if e != nil {
		t.Fatal(e)
	}
	if reasoner.calls.Load() != 0 {
		t.Fatal("inference before consent")
	}
	in := ExtractionInput{ExpectedRevision: source.Revision, PreviewID: preview.ID, PreviewHash: preview.Hash, ConsentID: "00000000-0000-0000-0000-000000000000"}
	if _, e = service.Extract(ctx, source.ID, "no-consent", in); e == nil {
		t.Fatal("missing consent accepted")
	}
	if reasoner.calls.Load() != 0 {
		t.Fatal("called before consent")
	}
	consent, e := service.Consent(ctx, source.ID, "consent", ConsentInput{source.Revision, preview.ID, preview.Hash})
	if e != nil {
		t.Fatal(e)
	}
	in.ConsentID = consent.ID
	attempt, e := service.Extract(ctx, source.ID, "extract", in)
	if e != nil || attempt.Status != "completed" || attempt.BatchID == nil {
		t.Fatal(attempt, e)
	}
	again, e := service.Extract(ctx, source.ID, "extract", in)
	if e != nil || again.ID != attempt.ID || reasoner.calls.Load() != 1 {
		t.Fatal("paid replay", again, e)
	}
	if count() != before {
		t.Fatal("AI wrote ledger")
	}
	var reserved, actual int64
	var status string
	if e = pool.QueryRow(ctx, "SELECT reserved_micros,actual_micros,status FROM ai_usage WHERE id=(SELECT reservation_id FROM statement_extraction_attempts WHERE id=$1)", attempt.ID).Scan(&reserved, &actual, &status); e != nil || reserved != 8000 || actual != 100 || status != "settled" {
		t.Fatal(reserved, actual, status, e)
	}
	source, e = repo.Statement(ctx, source.ID)
	if e != nil || source.Review == nil || len(source.Review.Reconciliation.Blockers) == 0 {
		t.Fatal("owner role/page gate skipped", e)
	}

	// Every AI resource is owner-scoped even when the other principal is admin.
	var otherID string
	if e = pool.QueryRow(ctx, "INSERT INTO users(display_name,role) VALUES('Statement AI other admin','admin') RETURNING id::text").Scan(&otherID); e != nil {
		t.Fatal(e)
	}
	other := access.WithPrincipal(context.Background(), access.Principal{ID: otherID, Role: "admin"})
	_, e = service.Attempt(other, source.ID, attempt.ID)
	if e == nil {
		t.Fatal("foreign admin read attempt")
	}
	foreignTx, e := database.ForUsers(pool).BeginTx(other, pgx.TxOptions{})
	if e != nil {
		t.Fatal(e)
	}
	for _, table := range []string{"statement_extraction_previews", "statement_extraction_consents", "statement_extraction_attempts", "statement_extraction_calls"} {
		var n int
		var forced bool
		if e = foreignTx.QueryRow(other, "SELECT count(*) FROM "+table).Scan(&n); e != nil || n != 0 {
			t.Fatal("foreign AI rows", table, n, e)
		}
		if e = foreignTx.QueryRow(other, "SELECT relrowsecurity AND relforcerowsecurity FROM pg_class WHERE oid=$1::regclass", table).Scan(&forced); e != nil || !forced {
			t.Fatal("AI RLS not forced", table, e)
		}
	}
	_, e = foreignTx.Exec(other, "INSERT INTO statement_extraction_consents(statement_id,preview_id,preview_hash,creation_key) VALUES($1,$2,$3,'foreign-ai-consent')", source.ID, preview.ID, preview.Hash)
	_ = foreignTx.Rollback(other)
	if e == nil {
		t.Fatal("cross-owner compound consent FK")
	}
	_, e = service.Consent(ctx, source.ID, "consent", ConsentInput{ExpectedRevision: 99, PreviewID: preview.ID, PreviewHash: preview.Hash})
	if e == nil {
		t.Fatal("consent key accepted changed revision")
	}
	// Concurrent source edits cancel a claimed attempt; a late paid result may
	// settle usage but cannot materialize a stale draft or resurrect an original.
	stale, _, e := repo.UploadStatement(ctx, []byte("%PDF-synthetic-ai-stale"), source.AccountID, "stale.pdf", "ai-stale")
	if e != nil {
		t.Fatal(e)
	}
	sp, e := service.Preview(ctx, stale.ID, PreviewInput{ExpectedRevision: stale.Revision})
	if e != nil {
		t.Fatal(e)
	}
	sc, e := service.Consent(ctx, stale.ID, "consent-stale", ConsentInput{stale.Revision, sp.ID, sp.Hash})
	if e != nil {
		t.Fatal(e)
	}
	started, release := make(chan struct{}), make(chan struct{})
	reasoner.before = func(context.Context) { close(started); <-release }
	type extractionResult struct {
		attempt Attempt
		err     error
	}
	done := make(chan extractionResult, 1)
	go func() {
		a, e := service.Extract(ctx, stale.ID, "extract-stale", ExtractionInput{stale.Revision, sp.ID, sp.Hash, sc.ID})
		done <- extractionResult{a, e}
	}()
	select {
	case <-started:
	case <-time.After(5 * time.Second):
		t.Fatal("provider not started")
	}
	_, e = repo.EditStatement(ctx, stale.ID, "edit-inflight", imports.StatementEdit{ExpectedRevision: stale.Revision, AccountID: stale.AccountID, Retention: "keep", Reason: "Changed source review during inference"})
	if e != nil {
		t.Fatal(e)
	}
	close(release)
	result := <-done
	reasoner.before = nil
	if result.err != nil || result.attempt.Status != "failed" || result.attempt.BatchID != nil {
		t.Fatal("stale result materialized", result)
	}
	// Known-cost malformed output is charged, and retries consume the same
	// document's 32-call lifetime rather than receiving a fresh budget on reload.
	limited, _, e := repo.UploadStatement(ctx, []byte("%PDF-synthetic-ai-limit"), source.AccountID, "limit.pdf", "ai-limit")
	if e != nil {
		t.Fatal(e)
	}
	lp, e := service.Preview(ctx, limited.ID, PreviewInput{ExpectedRevision: limited.Revision})
	if e != nil {
		t.Fatal(e)
	}
	lc, e := service.Consent(ctx, limited.ID, "consent-limit", ConsentInput{limited.Revision, lp.ID, lp.Hash})
	if e != nil {
		t.Fatal(e)
	}
	reasoner.malformed = true
	beforeLimit := reasoner.calls.Load()
	for index := 0; index < MaxCalls; index++ {
		a, e := service.Extract(ctx, limited.ID, fmt.Sprintf("limit-%d", index), ExtractionInput{limited.Revision, lp.ID, lp.Hash, lc.ID})
		if e != nil || a.Status != "failed" || a.BatchID != nil {
			t.Fatal("malformed proposal accepted", index, a, e)
		}
	}
	_, e = service.Extract(ctx, limited.ID, "over-limit", ExtractionInput{limited.Revision, lp.ID, lp.Hash, lc.ID})
	if e == nil || reasoner.calls.Load() != beforeLimit+MaxCalls {
		t.Fatal("document call limit reset", e)
	}
	reasoner.malformed = false

	// A separate unknown-cost response freezes the shared budget and never retries.
	beforeUnknown := reasoner.calls.Load()
	reasoner.unknown = true
	source2, _, e := repo.UploadStatement(ctx, []byte("%PDF-synthetic-ai-lost"), source.AccountID, "lost.pdf", "ai-lost")
	if e != nil {
		t.Fatal(e)
	}
	preview, e = service.Preview(ctx, source2.ID, PreviewInput{ExpectedRevision: source2.Revision})
	if e != nil {
		t.Fatal(e)
	}
	consent, e = service.Consent(ctx, source2.ID, "consent-lost", ConsentInput{source2.Revision, preview.ID, preview.Hash})
	if e != nil {
		t.Fatal(e)
	}
	in = ExtractionInput{source2.Revision, preview.ID, preview.Hash, consent.ID}
	attempt, e = service.Extract(ctx, source2.ID, "lost", in)
	if e != nil || attempt.Status != "uncertain" {
		t.Fatal(attempt, e)
	}
	_, e = service.Extract(ctx, source2.ID, "lost", in)
	if e != nil || reasoner.calls.Load() != beforeUnknown+1 {
		t.Fatal("unknown call retried", e)
	}
	tx, e := service.begin(ctx)
	if e != nil {
		t.Fatal(e)
	}
	var remaining int64
	e = tx.QueryRow(ctx, "SELECT grace_ai_remaining()").Scan(&remaining)
	_ = tx.Rollback(ctx)
	if e != nil || remaining != 0 {
		t.Fatal("unknown cost released budget", remaining, e)
	}
	// Deletion clears every full preview payload while preserving consent/cost IDs.
	_, e = repo.DeleteStatementOriginal(ctx, source2.ID, "delete-lost", imports.StatementDelete{ExpectedRevision: source2.Revision})
	if e != nil {
		t.Fatal(e)
	}
	preservedAttempt, e := service.Attempt(ctx, source2.ID, attempt.ID)
	if e != nil || preservedAttempt.Policy.Model != testPolicy().Model || preservedAttempt.PromptVersion != PromptVersion || preservedAttempt.SchemaHash == "" {
		t.Fatal("deleted plan discarded model provenance", preservedAttempt, e)
	}
	var plans int
	if e = pool.QueryRow(ctx, "SELECT count(*) FROM statement_extraction_previews WHERE statement_id=$1 AND plan IS NOT NULL", source2.ID).Scan(&plans); e != nil || plans != 0 {
		t.Fatal("preview is a hidden full-text archive", plans, e)
	}
}
