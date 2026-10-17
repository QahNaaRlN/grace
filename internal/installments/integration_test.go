//go:build integration

package installments_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/QahNaaRlN/grace/internal/accounts"
	"github.com/QahNaaRlN/grace/internal/analytics"
	"github.com/QahNaaRlN/grace/internal/creditcards"
	"github.com/QahNaaRlN/grace/internal/financialevents"
	"github.com/QahNaaRlN/grace/internal/installments"
	"github.com/QahNaaRlN/grace/internal/obligations"
	"github.com/QahNaaRlN/grace/internal/platform/access"
	"github.com/QahNaaRlN/grace/internal/platform/database"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type fixture struct {
	t               *testing.T
	pool            *pgxpool.Pool
	ctx, other      context.Context
	router          *chi.Mux
	debit, category string
	seq             int
}

func TestExistingPurchaseLinkDoesNotCreatePostings(t *testing.T) {
	f := setup(t)
	zero := int64(0)
	account, e := accounts.NewRepository(f.pool).Create(f.ctx, accounts.CreateRequest{Name: "Synthetic existing purchase card", Type: "credit", Currency: "RUB", OpeningBalanceMinor: &zero})
	if e != nil {
		t.Fatal(e)
	}
	purchase := f.event(financialevents.Command{Kind: "purchase", AccountID: account.ID, CategoryID: &f.category, AmountMinor: 1200000, OccurredDate: "2026-09-01"})
	c := f.request()
	c.Mode = "credit_card"
	c.AccountID = account.ID
	c.Purchase = nil
	c.PurchaseEventID = purchase.ID
	var plan installments.Plan
	if f.call(f.ctx, "POST", "/installment-plans", "existing-purchase", c, &plan) != 200 || *plan.AccountDebtMinor != 1200000 {
		t.Fatalf("existing link %+v", plan)
	}
	tenants := database.ForUsers(f.pool)
	var events, postings int
	if e = tenants.QueryRow(f.ctx, `SELECT (SELECT count(*) FROM financial_events),(SELECT count(*) FROM postings)`).Scan(&events, &postings); e != nil || events != 1 || postings != 1 {
		t.Fatalf("duplicated purchase %d %d %v", events, postings, e)
	}
	if f.call(f.ctx, "POST", "/installment-plans", "duplicate-purchase", c, nil) != 409 {
		t.Fatal("purchase used twice")
	}
	if e = tenants.QueryRow(f.other, `SELECT count(*) FROM installment_plans`).Scan(&events); e != nil || events != 0 {
		t.Fatal("RLS exposed another owner", e)
	}
	transfer := f.event(financialevents.Command{Kind: "internal_transfer", AccountID: f.debit, DestinationAccountID: account.ID, AmountMinor: 1200000, OccurredDate: "2026-10-16"})
	items := []installments.Allocation{}
	for _, row := range plan.Payments {
		items = append(items, installments.Allocation{PaymentID: row.ID, AmountMinor: row.AmountMinor})
	}
	if f.call(f.ctx, "PUT", "/financial-events/"+transfer.ID+"/installment-allocation", "complete-card-plan", installments.AllocationRequest{Reason: "Bank full payoff", Items: items}, nil) != 200 {
		t.Fatal("multi-row allocation")
	}
	var card creditcards.Card
	f.call(f.ctx, "GET", "/accounts/"+account.ID+"/credit-card?on=2026-10-17", "", nil, &card)
	if card.PaymentRequirement != nil || card.Status.DebtMinor == nil || *card.Status.DebtMinor != 0 {
		t.Fatalf("completed plan still requires bank override %+v", card)
	}
}

func setup(t *testing.T) *fixture {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL required")
	}
	pool, e := pgxpool.New(context.Background(), url)
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(pool.Close)
	f := &fixture{t: t, pool: pool}
	var owner, other string
	for _, v := range []*string{&owner, &other} {
		if e = pool.QueryRow(context.Background(), `INSERT INTO users(display_name,role) VALUES('Synthetic installment owner','member') RETURNING id::text`).Scan(v); e != nil {
			t.Fatal(e)
		}
	}
	f.ctx = access.WithPrincipal(context.Background(), access.Principal{ID: owner, Role: "member"})
	f.other = access.WithPrincipal(context.Background(), access.Principal{ID: other, Role: "admin"})
	if e = pool.QueryRow(context.Background(), `SELECT id::text FROM categories WHERE owner_id IS NULL AND kind='expense' LIMIT 1`).Scan(&f.category); e != nil {
		t.Fatal(e)
	}
	opening := int64(2000000)
	a, e := accounts.NewRepository(pool).Create(f.ctx, accounts.CreateRequest{Name: "Synthetic debit", Type: "debit", Currency: "RUB", OpeningBalanceMinor: &opening})
	if e != nil {
		t.Fatal(e)
	}
	f.debit = a.ID
	f.router = chi.NewRouter()
	installments.NewHandler(pool).Routes(f.router)
	creditcards.NewHandler(pool).Routes(f.router)
	return f
}
func (f *fixture) call(ctx context.Context, method, path, key string, body, out any) int {
	f.t.Helper()
	b, _ := json.Marshal(body)
	r := httptest.NewRequest(method, path, bytes.NewReader(b)).WithContext(ctx)
	if key != "" {
		r.Header.Set("Idempotency-Key", key)
	}
	w := httptest.NewRecorder()
	f.router.ServeHTTP(w, r)
	if out != nil && w.Code < 300 {
		if e := json.Unmarshal(w.Body.Bytes(), out); e != nil {
			f.t.Fatal(e)
		}
	}
	if w.Code >= 500 {
		f.t.Fatalf("%s %s: %d %s", method, path, w.Code, w.Body)
	}
	return w.Code
}
func (f *fixture) request() installments.CreateRequest {
	return installments.CreateRequest{Name: "Synthetic BNPL", Mode: "standalone", Provider: "Synthetic provider", Currency: "RUB", OpenedOn: "2026-09-01", PrincipalMinor: 1200000, Purchase: &installments.Purchase{CategoryID: f.category}, Payments: []installments.Row{{DueOn: "2026-09-01", AmountMinor: 300000}, {DueOn: "2026-09-15", AmountMinor: 300000}, {DueOn: "2026-10-01", AmountMinor: 300000}, {DueOn: "2026-10-15", AmountMinor: 300000}}}
}
func (f *fixture) event(c financialevents.Command) financialevents.Event {
	f.t.Helper()
	f.seq++
	c.IdempotencyKey = fmt.Sprintf("%s-event-%d", f.debit, f.seq)
	c.RequestHash = fmt.Sprintf("%064d", f.seq)
	v, _, e := financialevents.NewRepository(f.pool).Create(f.ctx, c)
	if e != nil {
		f.t.Fatal(e)
	}
	return v
}
func TestPurchaseDebtAndAllocations(t *testing.T) {
	f := setup(t)
	c := f.request()
	c.FirstPayment = &installments.FirstPayment{SourceAccountID: f.debit, AmountMinor: 300000, On: "2026-09-01"}
	var p installments.Plan
	if code := f.call(f.ctx, "POST", "/installment-plans", "new", c, &p); code != 200 {
		t.Fatalf("create %d", code)
	}
	if p.RemainingMinor != 900000 || p.AccountDebtMinor == nil || *p.AccountDebtMinor != 900000 || p.PaidMinor != 300000 {
		t.Fatalf("first payment %+v", p)
	}
	var replay installments.Plan
	if code := f.call(f.ctx, "POST", "/installment-plans", "new", c, &replay); code != 200 || replay.ID != p.ID {
		t.Fatal("replay")
	}
	c.Name = "Different"
	if f.call(f.ctx, "POST", "/installment-plans", "new", c, nil) != 409 {
		t.Fatal("key payload conflict")
	}
	if f.call(f.other, "GET", "/installment-plans/"+p.ID, "", nil, nil) != 404 {
		t.Fatal("owner isolation")
	}
	for n := 1; n < 4; n++ {
		event := f.event(financialevents.Command{Kind: "internal_transfer", AccountID: f.debit, DestinationAccountID: p.AccountID, AmountMinor: 300000, OccurredDate: p.Payments[n].DueOn})
		var a installments.AllocationView
		body := installments.AllocationRequest{Reason: "Bank confirmed", Items: []installments.Allocation{{PaymentID: p.Payments[n].ID, AmountMinor: 300000}}}
		path := "/financial-events/" + event.ID + "/installment-allocation"
		if code := f.call(f.ctx, "PUT", path, fmt.Sprintf("allocate-%d", n), body, &a); code != 200 || a.UnallocatedMinor != 0 {
			t.Fatalf("allocate %d %+v", code, a)
		}
		if f.call(f.ctx, "PUT", path, fmt.Sprintf("allocate-%d", n), body, nil) != 200 {
			t.Fatal("idempotency before revision")
		}
		if f.call(f.ctx, "PUT", path, fmt.Sprintf("stale-%d", n), body, nil) != 409 {
			t.Fatal("revision")
		}
	}
	var done installments.Plan
	f.call(f.ctx, "GET", "/installment-plans/"+p.ID+"?on=2026-10-16", "", nil, &done)
	if done.RemainingMinor != 0 || *done.AccountDebtMinor != 0 {
		t.Fatalf("paid %+v", done)
	}
	var before installments.Plan
	f.call(f.ctx, "GET", "/installment-plans/"+p.ID+"?on=2026-09-02", "", nil, &before)
	if before.PaidMinor != 300000 {
		t.Fatal("future payments counted in past")
	}
	from := time.Date(2026, 9, 1, 0, 0, 0, 0, time.FixedZone("app", 5*3600))
	to := from.AddDate(0, 2, 0)
	summary, e := analytics.NewRepository(f.pool).PeriodSummary(f.ctx, from, to)
	if e != nil {
		t.Fatal(e)
	}
	s := summary.Currencies[0]
	if s.PurchasesMinor != 1200000 || s.IncomeMinor != 0 || s.DebtPaymentsMinor != 0 || s.InstallmentPrincipalRepaidMinor != 1200000 {
		t.Fatalf("double expense %+v", s)
	}
	pos, e := analytics.NewRepository(f.pool).CashPosition(f.ctx, "2026-10-16", "2026-10-30")
	if e != nil {
		t.Fatal(e)
	}
	if *pos.Currencies[0].LiquidBalanceMinor != 800000 || pos.Currencies[0].ObligationsThroughMinor != 0 {
		t.Fatalf("position %+v", pos)
	}
	legacy, e := obligations.NewRepository(f.pool).Get(f.ctx, p.ObligationID, "2026-09-02")
	if e != nil || legacy.PaidMinor != 300000 || !legacy.ManagedInstallment {
		t.Fatalf("obligation %+v %v", legacy, e)
	}
	var draft installments.Draft
	draftPath := "/installment-plans/" + p.ID + "/schedule-drafts"
	f.call(f.ctx, "POST", draftPath, "paid-copy", installments.DraftRequest{Revision: 1}, &draft)
	draft.Payments[0].DueOn = "2026-09-02"
	f.call(f.ctx, "PUT", draftPath+"/"+draft.ID, "paid-edit", installments.DraftRequest{Revision: draft.Revision, Payments: draft.Payments}, &draft)
	if f.call(f.ctx, "POST", draftPath+"/"+draft.ID+"/apply", "paid-apply", map[string]int{"revision": draft.Revision, "planRevision": 1}, nil) != 409 {
		t.Fatal("paid row changed through draft")
	}
	// Full rollback if the first transfer references an inaccessible account.
	c = f.request()
	c.FirstPayment = &installments.FirstPayment{SourceAccountID: "00000000-0000-0000-0000-000000000000", AmountMinor: 300000, On: c.OpenedOn}
	var count int
	f.call(f.ctx, "POST", "/installment-plans", "invalid-first", c, nil)
	if e = database.ForUsers(f.pool).QueryRow(f.ctx, `SELECT count(*) FROM installment_plans`).Scan(&count); e != nil || count != 1 {
		t.Fatalf("non-atomic %d %v", count, e)
	}
}
func TestHistoricalAndScheduleRevision(t *testing.T) {
	f := setup(t)
	c := f.request()
	c.Historical = true
	c.Purchase = nil
	c.ObservedOn = "2026-09-15"
	c.InitialRemainingMinor = 900000
	c.Payments = c.Payments[1:]
	var p installments.Plan
	if f.call(f.ctx, "POST", "/installment-plans", "history", c, &p) != 200 || p.PurchaseEventID != nil || *p.AccountDebtMinor != 900000 {
		t.Fatal("historical")
	}
	var count int
	tenants := database.ForUsers(f.pool)
	if e := tenants.QueryRow(f.ctx, `SELECT count(*) FROM financial_events`).Scan(&count); e != nil || count != 0 {
		t.Fatal("fabricated history", e)
	}
	var d installments.Draft
	base := "/installment-plans/" + p.ID + "/schedule-drafts"
	if f.call(f.ctx, "POST", base, "draft", installments.DraftRequest{Revision: 1}, &d) != 200 {
		t.Fatal("draft")
	}
	d.Payments[0].DueOn = "2026-09-16"
	if f.call(f.ctx, "PUT", base+"/"+d.ID, "edit-draft", installments.DraftRequest{Revision: d.Revision, Payments: d.Payments}, &d) != 200 {
		t.Fatal("edit")
	}
	if f.call(f.ctx, "POST", base+"/"+d.ID+"/apply", "apply", map[string]int{"revision": d.Revision, "planRevision": 1}, &p) != 200 || p.Revision != 2 || p.Payments[0].DueOn != "2026-09-16" {
		t.Fatal("apply")
	}
	if f.call(f.ctx, "POST", base+"/"+d.ID+"/apply", "apply-stale", map[string]int{"revision": d.Revision, "planRevision": 1}, nil) != 409 {
		t.Fatal("apply stale")
	}
	from := time.Date(2026, 9, 1, 0, 0, 0, 0, time.FixedZone("app", 5*3600))
	summary, err := analytics.NewRepository(f.pool).PeriodSummary(f.ctx, from, from.AddDate(0, 1, 0))
	if err != nil || summary.Currencies[0].OpeningBalanceMinor != nil {
		t.Fatalf("invented pre-observation opening balance %+v %v", summary, err)
	}
	cash, err := analytics.NewRepository(f.pool).CashPosition(f.ctx, "2026-09-14", "2026-09-30")
	if err != nil || !cash.Currencies[0].CoverageIncomplete || cash.Currencies[0].AvailableAfterObligationsMinor != nil {
		t.Fatalf("invented pre-observation liquidity %+v %v", cash, err)
	}
	// Even the general financial API cannot write before the observation date.
	_, _, e := financialevents.NewRepository(f.pool).Create(f.ctx, financialevents.Command{Kind: "internal_transfer", AccountID: f.debit, DestinationAccountID: p.AccountID, AmountMinor: 1, OccurredDate: "2026-09-14", IdempotencyKey: "before-history", RequestHash: "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"})
	if e == nil {
		t.Fatal("history guard bypass")
	}
}
func TestConcurrentTransferAndDirectSQLGuards(t *testing.T) {
	f := setup(t)
	var p installments.Plan
	f.call(f.ctx, "POST", "/installment-plans", "new", f.request(), &p)
	event := f.event(financialevents.Command{Kind: "internal_transfer", AccountID: f.debit, DestinationAccountID: p.AccountID, AmountMinor: 300000, OccurredDate: "2026-09-01"})
	var wg sync.WaitGroup
	codes := make(chan int, 2)
	for n := 0; n < 2; n++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			codes <- f.call(f.ctx, "PUT", "/financial-events/"+event.ID+"/installment-allocation", fmt.Sprintf("race-%d", n), installments.AllocationRequest{Reason: "bank", Items: []installments.Allocation{{PaymentID: p.Payments[n].ID, AmountMinor: 300000}}}, nil)
		}(n)
	}
	wg.Wait()
	close(codes)
	success := 0
	for code := range codes {
		if code == 200 {
			success++
		} else if code != 409 {
			t.Fatalf("race %d", code)
		}
	}
	if success != 1 {
		t.Fatal("credit spent twice")
	}
	tx, e := database.ForUsers(f.pool).BeginTx(f.ctx, pgx.TxOptions{})
	if e != nil {
		t.Fatal(e)
	}
	_, e = tx.Exec(f.ctx, `UPDATE installment_payment_allocations SET amount_minor=400000 WHERE event_id::text=$1`, event.ID)
	tx.Rollback(f.ctx)
	if e == nil {
		t.Fatal("SQL credit overallocated")
	}
	tx, _ = database.ForUsers(f.pool).BeginTx(f.ctx, pgx.TxOptions{})
	_, e = tx.Exec(f.ctx, `UPDATE accounts SET opening_balance_minor=100 WHERE id::text=$1`, p.AccountID)
	tx.Rollback(f.ctx)
	if e == nil {
		t.Fatal("SQL changed debt basis")
	}
	tx, _ = database.ForUsers(f.pool).BeginTx(f.ctx, pgx.TxOptions{})
	_, e = tx.Exec(f.ctx, `UPDATE scheduled_payments SET amount_minor=400000 WHERE obligation_id::text=$1`, p.ObligationID)
	if e == nil {
		e = tx.Commit(f.ctx)
	} else {
		tx.Rollback(f.ctx)
	}
	if e == nil {
		t.Fatal("SQL schedule sum bypass")
	}
	var allocation installments.AllocationView
	allocationPath := "/financial-events/" + event.ID + "/installment-allocation"
	f.call(f.ctx, "GET", allocationPath, "", nil, &allocation)
	if f.call(f.ctx, "DELETE", allocationPath, "detach", installments.AllocationRequest{Revision: allocation.Revision, Reason: "Correct bank allocation"}, &allocation) != 200 || allocation.UnallocatedMinor != 300000 || len(allocation.History) < 2 {
		t.Fatalf("detach/audit %+v", allocation)
	}
	// Void/refund dependencies are enforced through ordinary core methods.
	if _, _, e = financialevents.NewRepository(f.pool).Create(f.ctx, financialevents.Command{Kind: "refund", OriginalEventID: *p.PurchaseEventID, AmountMinor: 1, OccurredDate: "2026-10-01", IdempotencyKey: "refund", RequestHash: "0000000000000000000000000000000000000000000000000000000000000000"}); e == nil {
		t.Fatal("refund bypass")
	}
}

func TestCardRequirementsAndNoDuplicateDebt(t *testing.T) {
	f := setup(t)
	zero := int64(0)
	account, e := accounts.NewRepository(f.pool).Create(f.ctx, accounts.CreateRequest{Name: "Synthetic credit", Type: "credit", Currency: "RUB", OpeningBalanceMinor: &zero})
	if e != nil {
		t.Fatal(e)
	}
	c := f.request()
	c.Mode = "credit_card"
	c.AccountID = account.ID
	var plan installments.Plan
	if f.call(f.ctx, "POST", "/installment-plans", "card-plan", c, &plan) != 200 {
		t.Fatal("card plan")
	}
	var card creditcards.Card
	path := "/accounts/" + account.ID + "/credit-card"
	f.call(f.ctx, "GET", path+"?on=2026-10-01", "", nil, &card)
	if card.Status.DebtMinor == nil || *card.Status.DebtMinor != 1200000 || card.PaymentRequirement == nil || card.PaymentRequirement.Source != "unknown" || card.Status.GraceAmountMinor != nil {
		t.Fatalf("double debt or invented grace %+v", card)
	}
	snapshot := map[string]any{"takenOn": "2026-10-01", "debtMinor": 1200000, "graceAmountMinor": 500000, "graceDueOn": "2026-10-30", "minimumMinor": 200000, "minimumDueOn": "2026-10-30", "note": "synthetic", "installmentsIncluded": true, "minimumInGrace": true, "confirmAllOperations": true, "periodFrom": "2026-09-01"}
	if f.call(f.ctx, "POST", path+"/snapshots", "card-bank", snapshot, &card) != 201 {
		t.Fatal("snapshot")
	}
	if f.call(f.ctx, "POST", path+"/snapshots", "card-bank", snapshot, nil) != 200 {
		t.Fatal("bank replay")
	}
	snapshot["graceAmountMinor"] = 600000
	if f.call(f.ctx, "POST", path+"/snapshots", "card-bank", snapshot, nil) != 409 {
		t.Fatal("bank key conflict")
	}
	snapshot["graceAmountMinor"] = 500000
	f.call(f.ctx, "GET", path+"?on=2026-10-01", "", nil, &card)
	if card.PaymentRequirement.Source != "bank_confirmed" || card.Status.GraceAmountMinor == nil || *card.Status.GraceAmountMinor != 500000 {
		t.Fatalf("bank %+v", card)
	}
	position, e := analytics.NewRepository(f.pool).CashPosition(f.ctx, "2026-10-01", "2026-10-30")
	if e != nil {
		t.Fatal(e)
	}
	p := position.Currencies[0]
	if p.ObligationsThroughMinor != 500000 || len(p.CardRequirements) != 1 || p.CardRequirements[0].AccountID != account.ID || len(p.ScheduledPayments) != 0 || p.CoverageIncomplete || p.AvailableAfterObligationsMinor == nil || *p.AvailableAfterObligationsMinor != 1500000 {
		t.Fatalf("double card requirement %+v", p)
	}
	position, e = analytics.NewRepository(f.pool).CashPosition(f.ctx, "2026-10-01", "2026-11-30")
	if e != nil || !position.Currencies[0].CoverageIncomplete || position.Currencies[0].AvailableAfterObligationsMinor != nil {
		t.Fatalf("future completeness %+v %v", position, e)
	}
	// Backdated/same-day payment recorded after confirmation invalidates it too.
	transfer := f.event(financialevents.Command{Kind: "internal_transfer", AccountID: f.debit, DestinationAccountID: account.ID, AmountMinor: 100000, OccurredDate: "2026-10-01"})
	f.call(f.ctx, "GET", path+"?on=2026-10-01", "", nil, &card)
	if card.PaymentRequirement.Status != "needs_confirmation" || card.Status.GraceAmountMinor != nil {
		t.Fatalf("stale requirement %+v", card.PaymentRequirement)
	}
	// Repayment principal can be allocated without changing the card ledger again.
	var a installments.AllocationView
	if f.call(f.ctx, "PUT", "/financial-events/"+transfer.ID+"/installment-allocation", "card-allocate", installments.AllocationRequest{Reason: "Bank confirmed", Items: []installments.Allocation{{PaymentID: plan.Payments[0].ID, AmountMinor: 100000}}}, &a) != 200 {
		t.Fatal("card allocation")
	}
	f.call(f.ctx, "GET", path+"?on=2026-10-01", "", nil, &card)
	if *card.Status.DebtMinor != 1100000 {
		t.Fatal("allocation changed debt")
	}
	snapshot["takenOn"] = "2099-01-01"
	if f.call(f.ctx, "POST", path+"/snapshots", "future", snapshot, nil) != 422 {
		t.Fatal("future checkpoint")
	}
}
