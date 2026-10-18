//go:build integration

package statementextract

import (
	"context"
	"os"
	"sync/atomic"
	"testing"

	"github.com/QahNaaRlN/grace/internal/imports"
	"github.com/QahNaaRlN/grace/internal/pdfextract"
	"github.com/QahNaaRlN/grace/internal/platform/access"
	"github.com/QahNaaRlN/grace/internal/statementdata"
	"github.com/jackc/pgx/v5/pgxpool"
)

type groundedFixtureReasoner struct {
	proposal statementdata.Proposal
	calls    atomic.Int32
}

func (f *groundedFixtureReasoner) CurrentPolicy(context.Context) (Policy, error) {
	return testPolicy(), nil
}
func (f *groundedFixtureReasoner) Complete(context.Context, Policy, Chunk) (statementdata.Proposal, *Billing, error) {
	f.calls.Add(1)
	return f.proposal, &Billing{GenerationID: "synthetic-real-pdf-generation", CostMicros: 100, InputTokens: 2000, OutputTokens: 500}, nil
}
func TestStatementRealPDFFakeProviderLifecycle(t *testing.T) {
	socket := os.Getenv("TEST_PDF_EXTRACTOR_SOCKET")
	url := os.Getenv("TEST_DATABASE_URL")
	if socket == "" || url == "" || os.Getenv("TEST_BANK_STATEMENTS") != "true" {
		t.Skip("real isolated PDF helper and disposable database required")
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
	extractor, e := pdfextract.NewClient(socket)
	if e != nil {
		t.Fatal(e)
	}
	raw, e := os.ReadFile("../pdfextract/testdata/columns.pdf")
	if e != nil {
		t.Fatal(e)
	}
	repo := imports.NewRepository(pool)
	repo.SetPDFExtractor(extractor)
	const account = "10000000-0000-0000-0000-000000000001"
	source, _, e := repo.UploadStatement(ctx, raw, account, "synthetic-columns.pdf", "real-pdf-upload")
	if e != nil {
		t.Fatal(e)
	}
	if source.Layout == nil || source.PageCount != 2 {
		t.Fatal("lost actual PDF pages")
	}
	doc := *source.Layout
	find := func(text string) statementdata.Selection {
		for _, s := range doc.Spans {
			if s.Text == text {
				return statementdata.Selection{s.ID}
			}
		}
		t.Fatalf("missing %q", text)
		return nil
	}
	phrase := func(first string, length int) statementdata.Selection {
		for i, s := range doc.Spans {
			if s.Text == first {
				ids := statementdata.Selection{}
				for j := 0; j < length; j++ {
					ids = append(ids, doc.Spans[i+j].ID)
				}
				return ids
			}
		}
		t.Fatalf("missing phrase %q", first)
		return nil
	}
	shop := find("SHOP")
	shopIndex := 0
	for i, s := range doc.Spans {
		if s.ID == shop[0] {
			shopIndex = i
		}
	}
	proposal := statementdata.Proposal{Rows: []statementdata.RowMapping{
		{OperationDate: find("19.10.2026"), BookingDate: find("20.10.2026"), AccountAmount: find("-1234.56"), AccountCurrency: find("RUB"), TransactionAmount: find("-15.00"), TransactionCurrency: find("USD"), Description: statementdata.Selection{doc.Spans[shopIndex-1].ID, shop[0]}},
		{OperationDate: find("21.10.2026"), BookingDate: find("22.10.2026"), AccountAmount: find("+2000.00"), AccountCurrency: find("RUB"), Description: phrase("OWN", 5)},
	}, Header: statementdata.HeaderMapping{AvailableBalanceObserved: find("7777.00"), DebitTurnover: find("1234.56"), CreditTurnover: find("2000.00"), Currency: find("RUB"), Labels: map[string]statementdata.Selection{"availableBalanceObserved": phrase("Available", 2), "debitTurnover": phrase("Debit", 2), "creditTurnover": phrase("Credit", 2)}}}
	used := map[string]bool{}
	for _, r := range proposal.Rows {
		for _, ids := range []statementdata.Selection{r.OperationDate, r.BookingDate, r.AccountAmount, r.AccountCurrency, r.TransactionAmount, r.TransactionCurrency, r.Description} {
			for _, id := range ids {
				used[id] = true
			}
		}
	}
	for _, ids := range []statementdata.Selection{proposal.Header.AvailableBalanceObserved, proposal.Header.DebitTurnover, proposal.Header.CreditTurnover, proposal.Header.Currency} {
		for _, id := range ids {
			used[id] = true
		}
	}
	for _, ids := range proposal.Header.Labels {
		for _, id := range ids {
			used[id] = true
		}
	}
	service := statementdata.Selection{}
	for _, span := range doc.Spans {
		if !used[span.ID] {
			service = append(service, span.ID)
		}
	}
	proposal.Service = []statementdata.ServiceMapping{{Spans: service, Reason: "Synthetic repeated table headings and bank title reviewed"}}
	fake := &groundedFixtureReasoner{proposal: proposal}
	orchestrator := NewService(pool, repo, fake)
	count := func(table string) int {
		var n int
		if e = pool.QueryRow(ctx, "SELECT count(*) FROM "+table).Scan(&n); e != nil {
			t.Fatal(e)
		}
		return n
	}
	beforeEvents, beforePostings, beforeReceipts, beforeSettlements := count("financial_events"), count("postings"), count("receipts"), count("scheduled_payment_settlements")
	var opening int64
	if e = pool.QueryRow(ctx, "SELECT opening_balance_minor FROM accounts WHERE id=$1", account).Scan(&opening); e != nil {
		t.Fatal(e)
	}
	p, e := orchestrator.Preview(ctx, source.ID, PreviewInput{ExpectedRevision: source.Revision})
	if e != nil || len(p.Plan.Chunks) != 1 {
		t.Fatal("fixture plan split unexpectedly", len(p.Plan.Chunks), e)
	}
	if fake.calls.Load() != 0 {
		t.Fatal("provider called by upload/preview")
	}
	consent, e := orchestrator.Consent(ctx, source.ID, "real-pdf-consent", ConsentInput{source.Revision, p.ID, p.Hash})
	if e != nil {
		t.Fatal(e)
	}
	attempt, e := orchestrator.Extract(ctx, source.ID, "real-pdf-extract", ExtractionInput{source.Revision, p.ID, p.Hash, consent.ID})
	if e != nil || attempt.Status != "completed" || attempt.BatchID == nil || fake.calls.Load() != 1 {
		t.Fatal(attempt, e)
	}
	if count("financial_events") != beforeEvents || count("postings") != beforePostings || count("receipts") != beforeReceipts || count("scheduled_payment_settlements") != beforeSettlements {
		t.Fatal("pre-confirmation financial write")
	}
	source, e = repo.Statement(ctx, source.ID)
	if e != nil || source.Review.Reconciliation.Turnover.Status != "matched" || source.Review.Reconciliation.BookBalance.Status != "not_provided" || source.Review.Header.AvailableBalanceObserved == nil {
		t.Fatal("typed source reconciliation", source, e)
	}
	source, e = repo.EditStatement(ctx, source.ID, "real-pdf-role-review", imports.StatementEdit{ExpectedRevision: source.Revision, AccountID: account, Retention: "delete_after_confirmation", RolesConfirmed: true, ReviewedPages: []int{1, 2}, Reason: "Owner reviewed two synthetic pages and separate monetary columns"})
	if e != nil {
		t.Fatal(e)
	}
	batch, e := repo.Batch(ctx, *source.BatchID)
	if e != nil {
		t.Fatal(e)
	}
	page, e := repo.Records(ctx, batch.ID, "", "", "", 100)
	if e != nil || len(page.Items) != 2 {
		t.Fatal(e)
	}
	ids := []string{}
	for _, row := range page.Items {
		ids = append(ids, row.ID)
		decision := imports.Decision{Action: "exclude", Reason: "Owner's synthetic own-transfer row remains explicitly unrepresented; never invent income"}
		if row.Source.PDF.Ordinal == 1 {
			category := "20000000-0000-0000-0000-000000000001"
			decision = imports.Decision{Action: "create", Reason: "Owner confirmed synthetic purchase", Fields: &imports.Fields{Kind: "purchase", Currency: "RUB", AmountMinor: 123456, AccountID: account, CategoryID: &category, OccurredDate: "2026-10-19", Merchant: "Synthetic shop", Description: "Confirmed source row"}}
		}
		batch, e = repo.Edit(ctx, batch.ID, row.ID, "real-pdf-decision:"+row.ID, imports.Edit{ExpectedRevision: batch.Revision, Decision: decision})
		if e != nil {
			t.Fatal(e)
		}
	}
	preview, e := repo.Preview(ctx, batch.ID, "real-pdf-financial-preview", imports.Selection{ExpectedRevision: batch.Revision, RecordIDs: ids})
	if e != nil || len(preview.Blockers) > 0 || preview.NewEvents != 1 || preview.Exclusions != 1 {
		t.Fatal(preview, e)
	}
	receipt, e := repo.Confirm(ctx, batch.ID, "real-pdf-confirm", imports.Confirmation{PreviewID: preview.ID, PreviewHash: preview.Hash})
	if e != nil {
		t.Fatal(e)
	}
	replay, e := repo.Confirm(ctx, batch.ID, "real-pdf-confirm", imports.Confirmation{PreviewID: preview.ID, PreviewHash: preview.Hash})
	if e != nil || receipt.ID != replay.ID {
		t.Fatal("financial replay", e)
	}
	if count("financial_events") != beforeEvents+1 || count("postings") != beforePostings+1 || count("receipts") != beforeReceipts || count("scheduled_payment_settlements") != beforeSettlements {
		t.Fatal("source invented extra financial facts")
	}
	var afterOpening int64
	if e = pool.QueryRow(ctx, "SELECT opening_balance_minor FROM accounts WHERE id=$1", account).Scan(&afterOpening); e != nil || afterOpening != opening {
		t.Fatal("PDF changed opening balance", e)
	}
	source, e = repo.Statement(ctx, source.ID)
	if e != nil || source.Layout != nil || source.DeletedAt == nil || source.Review.Reconciliation.FinancialCoverage.Status != "incomplete" || source.ExcludedRows != 1 {
		t.Fatal("retention or excluded financial coverage", source, e)
	}
	preserved, e := orchestrator.Attempt(ctx, source.ID, attempt.ID)
	if e != nil || preserved.Policy.Provider != "google-vertex/eu" || preserved.SchemaHash == "" {
		t.Fatal("lost provenance", e)
	}
}
