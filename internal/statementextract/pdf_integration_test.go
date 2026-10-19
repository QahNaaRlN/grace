//go:build integration

package statementextract

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
	"io"
	"math"
	"net/http"
	"net/http/httptest"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/QahNaaRlN/grace/internal/imports"
	"github.com/QahNaaRlN/grace/internal/pdfextract"
	"github.com/QahNaaRlN/grace/internal/platform/access"
	"github.com/QahNaaRlN/grace/internal/statementdata"
	"github.com/jackc/pgx/v5/pgxpool"
)

type groundedFixtureReasoner struct {
	calls         atomic.Int32
	delay         time.Duration
	started       chan struct{}
	knownOnCancel bool
}

func (f *groundedFixtureReasoner) CurrentPolicy(context.Context) (Policy, error) {
	return testPolicy(), nil
}
func (f *groundedFixtureReasoner) Complete(ctx context.Context, _ Policy, chunk Chunk) (statementdata.Proposal, *Billing, error) {
	f.calls.Add(1)
	if f.started != nil {
		close(f.started)
	}
	select {
	case <-ctx.Done():
		if f.knownOnCancel {
			return statementdata.Proposal{}, &Billing{GenerationID: "synthetic-cancelled-known", CostMicros: 100, InputTokens: 2000, OutputTokens: 500}, ctx.Err()
		}
		return statementdata.Proposal{}, nil, ctx.Err()
	case <-time.After(f.delay):
	}
	proposal, e := columnsWireProposal(chunk)
	return proposal, &Billing{GenerationID: "synthetic-real-pdf-generation", CostMicros: 100, InputTokens: 2000, OutputTokens: 500}, e
}

// All selection decisions use only the actual delivered tuples. The fake has no
// canonical map, original PDF/layout, hidden financial fields or global rows.
func columnsWireProposal(chunk Chunk) (statementdata.Proposal, error) {
	if chunk.Index != 1 || len(chunk.Context) != 0 {
		return statementdata.Proposal{}, fmt.Errorf("not a whole-document request")
	}
	find := func(text string) RedactedSpan {
		for _, s := range chunk.Spans {
			if s.Text == text {
				return s
			}
		}
		return RedactedSpan{}
	}
	id := func(text string) statementdata.Selection { return statementdata.Selection{find(text).ID} }
	line := func(anchor RedactedSpan, delta int, words map[string]bool) statementdata.Selection {
		ids := statementdata.Selection{}
		for _, s := range chunk.Spans {
			if s.Page == anchor.Page && math.Abs(float64(s.Y-anchor.Y-delta)) <= 1 && (words == nil || words[s.Text]) {
				ids = append(ids, s.ID)
			}
		}
		return ids
	}
	p := statementdata.Proposal{Rows: []statementdata.RowMapping{
		{OperationDate: id("19.10.2026"), BookingDate: id("20.10.2026"), AccountAmount: id("-1234.56"), AccountCurrency: id("RUB"), TransactionAmount: id("-15.00"), TransactionCurrency: id("USD"), Description: line(find("19.10.2026"), 20, nil)},
		{OperationDate: id("21.10.2026"), BookingDate: id("22.10.2026"), AccountAmount: id("+2000.00"), AccountCurrency: id("RUB"), Description: line(find("21.10.2026"), 20, nil)},
	}, Header: statementdata.HeaderMapping{AvailableBalanceObserved: id("7777.00"), DebitTurnover: id("1234.56"), CreditTurnover: id("2000.00"), Currency: id("RUB"), Labels: map[string]statementdata.Selection{
		"availableBalanceObserved": line(find("7777.00"), 0, map[string]bool{"Available": true, "balance": true}), "debitTurnover": line(find("1234.56"), 0, map[string]bool{"Debit": true, "turnover": true}), "creditTurnover": line(find("2000.00"), 0, map[string]bool{"Credit": true, "turnover": true}),
	}}}
	used := map[string]bool{}
	for _, r := range p.Rows {
		for _, ids := range []statementdata.Selection{r.OperationDate, r.BookingDate, r.AccountAmount, r.AccountCurrency, r.TransactionAmount, r.TransactionCurrency, r.Description} {
			for _, id := range ids {
				if id == "" {
					return p, fmt.Errorf("missing wire field")
				}
				used[id] = true
			}
		}
	}
	for _, ids := range []statementdata.Selection{p.Header.AvailableBalanceObserved, p.Header.DebitTurnover, p.Header.CreditTurnover, p.Header.Currency} {
		for _, id := range ids {
			if id == "" {
				return p, fmt.Errorf("missing wire header")
			}
			used[id] = true
		}
	}
	for _, ids := range p.Header.Labels {
		for _, id := range ids {
			used[id] = true
		}
	}
	service := statementdata.Selection{}
	for _, span := range chunk.Spans {
		if !used[span.ID] {
			service = append(service, span.ID)
		}
	}
	p.Service = []statementdata.ServiceMapping{{Spans: service, Reason: "Reviewed synthetic page titles and unused headings"}}
	return p, nil
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
	fake := &groundedFixtureReasoner{delay: 80 * time.Millisecond}
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
	// Real handler + middleware + tenant SQL path, with a slow fake provider
	// exceeding the short global write timeout. No external LLM is involved.
	router := chi.NewRouter()
	router.Use(cors.Handler(cors.Options{AllowedOrigins: []string{"https://grace.test"}}))
	router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r.WithContext(access.WithPrincipal(r.Context(), access.Principal{ID: access.BootstrapOwnerID, Role: "admin"})))
		})
	})
	router.Route("/api/imports", NewHandler(orchestrator).Routes)
	router.Get("/ordinary", func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(80 * time.Millisecond)
		_, _ = w.Write([]byte("ordinary"))
	})
	server := httptest.NewUnstartedServer(router)
	server.Config.WriteTimeout = 20 * time.Millisecond
	server.Start()
	defer server.Close()
	client := &http.Client{Transport: &http.Transport{DisableKeepAlives: true}, Timeout: 5 * time.Second}
	body, _ := json.Marshal(ExtractionInput{source.Revision, p.ID, p.Hash, consent.ID})
	request, _ := http.NewRequest(http.MethodPost, server.URL+"/api/imports/statements/"+source.ID+"/extractions", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Idempotency-Key", "real-pdf-extract")
	response, e := client.Do(request)
	if e != nil {
		t.Fatal("extraction HTTP deadline", e)
	}
	responseBody, _ := io.ReadAll(response.Body)
	response.Body.Close()
	if response.StatusCode != 200 {
		t.Fatalf("extraction HTTP status %d %s", response.StatusCode, responseBody)
	}
	var attempt Attempt
	e = json.Unmarshal(responseBody, &attempt)
	if response, e := client.Get(server.URL + "/ordinary"); e == nil {
		response.Body.Close()
		t.Fatal("global write timeout was widened")
	}

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
	if len(source.Review.Rows) != 2 || source.Review.Rows[0].AccountAmount.AmountMinor != 123456 || source.Review.Rows[0].AccountAmount.Currency != "RUB" || source.Review.Rows[0].OperationDate.Local != "2026-10-19" || source.Review.Rows[0].BookingDate.Local != "2026-10-20" || source.Review.Rows[0].TransactionAmount.AmountMinor != 1500 || source.Review.Rows[0].TransactionAmount.Currency != "USD" || source.Review.Rows[1].AccountAmount.AmountMinor != 200000 || source.Review.Rows[1].OperationDate.Local != "2026-10-21" || source.Review.Rows[1].BookingDate.Local != "2026-10-22" || source.Review.Reconciliation.RowCoverage.Status != "not_reviewed" {
		t.Fatal("wire to canonical financial mapping wrong", source.Review)
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
	// Browser cancellation reaches the fake provider and still persists verified
	// billing, without materializing another draft or changing the ledger.
	cards, e := os.ReadFile("../pdfextract/testdata/cards.pdf")
	if e != nil {
		t.Fatal(e)
	}
	cancelled, _, e := repo.UploadStatement(ctx, cards, account, "cancelled-cards.pdf", "real-http-cancel-upload")
	if e != nil {
		t.Fatal(e)
	}
	cp, e := orchestrator.Preview(ctx, cancelled.ID, PreviewInput{ExpectedRevision: cancelled.Revision})
	if e != nil {
		t.Fatal(e)
	}
	cc, e := orchestrator.Consent(ctx, cancelled.ID, "real-http-cancel-consent", ConsentInput{cancelled.Revision, cp.ID, cp.Hash})
	if e != nil {
		t.Fatal(e)
	}
	fake.delay = 5 * time.Second
	fake.started = make(chan struct{})
	fake.knownOnCancel = true
	cancelCtx, cancelRequest := context.WithCancel(context.Background())
	defer cancelRequest()
	cancelBody, _ := json.Marshal(ExtractionInput{cancelled.Revision, cp.ID, cp.Hash, cc.ID})
	cancelReq, _ := http.NewRequestWithContext(cancelCtx, http.MethodPost, server.URL+"/api/imports/statements/"+cancelled.ID+"/extractions", bytes.NewReader(cancelBody))
	cancelReq.Header.Set("Content-Type", "application/json")
	cancelReq.Header.Set("Idempotency-Key", "real-http-cancel")
	result := make(chan error, 1)
	go func() {
		resp, err := client.Do(cancelReq)
		if resp != nil {
			resp.Body.Close()
		}
		result <- err
	}()
	select {
	case <-fake.started:
	case <-time.After(2 * time.Second):
		t.Fatal("cancel provider never started")
	}
	cancelRequest()
	select {
	case err := <-result:
		if err == nil {
			t.Fatal("browser cancellation ignored")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("cancel request hung")
	}
	var status string
	var actual int64
	deadline := time.Now().Add(2 * time.Second)
	for {
		e = pool.QueryRow(ctx, "SELECT a.status,u.actual_micros FROM statement_extraction_attempts a JOIN ai_usage u ON u.id=a.reservation_id WHERE a.statement_id=$1 AND a.status<>'running'", cancelled.ID).Scan(&status, &actual)
		if e == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("cancel billing not persisted", e)
		}
		time.Sleep(10 * time.Millisecond)
	}
	if status == "completed" || actual != 100 || fake.calls.Load() != 2 || count("financial_events") != beforeEvents+1 || count("postings") != beforePostings+1 {
		t.Fatal("cancel caused draft/write or lost known bill", status, actual)
	}

}
