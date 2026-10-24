//go:build integration

package documentqualification

import (
	"context"
	"github.com/QahNaaRlN/grace/internal/assistant"
	"github.com/QahNaaRlN/grace/internal/platform/access"
	"github.com/jackc/pgx/v5/pgxpool"
	"os"
	"sync"
	"testing"
	"time"
)

func TestQSQLBudget(t *testing.T) {
	if os.Getenv("TEST_Q_DOCUMENTS") != "true" {
		t.Skip("disposable database required")
	}
	ctx := context.Background()
	pool, e := pgxpool.New(ctx, os.Getenv("TEST_DATABASE_URL"))
	if e != nil {
		t.Fatal(e)
	}
	defer pool.Close()
	var owner, second string
	for _, dst := range []*string{&owner, &second} {
		if e = pool.QueryRow(ctx, "INSERT INTO users(display_name,role) VALUES('Q public experiment','admin') RETURNING id::text").Scan(dst); e != nil {
			t.Fatal(e)
		}
	}
	ownerCtx := access.WithPrincipal(ctx, access.Principal{ID: owner, Role: "admin"})
	otherCtx := access.WithPrincipal(ctx, access.Principal{ID: second, Role: "admin"})
	// Explicit fixture funding exists only in this disposable test database.
	if _, e = pool.Exec(ctx, "UPDATE ai_experiment_budget SET allowance_micros=10000000,fees_micros=0 WHERE singleton"); e != nil {
		t.Fatal(e)
	}
	budget := NewSQLBudget(pool)
	before, e := budget.Remaining(ownerCtx)
	if e != nil {
		t.Fatal(e)
	}
	var financeBefore, financeAfter int
	if e = pool.QueryRow(ctx, "SELECT (SELECT count(*) FROM postings)+(SELECT count(*) FROM financial_events)").Scan(&financeBefore); e != nil {
		t.Fatal(e)
	}
	c := corpus(t)
	m, r := executeSQL(t, ownerCtx, c, &fakeHTTP{c: c}, budget)
	if r.Core.Decision != "PASS_SYNTHETIC" {
		t.Fatal(r.Core.Decision, r.Core.Results)
	}
	after, e := budget.Remaining(ownerCtx)
	if e != nil || before-after != 80 {
		t.Fatal("shared spend not settled", before, after, e)
	}
	raw, _ := EncodeReport(r)
	if _, e = Replay(ctx, c, m, raw); e != nil {
		t.Fatal(e)
	}
	if e = pool.QueryRow(ctx, "SELECT (SELECT count(*) FROM postings)+(SELECT count(*) FROM financial_events)").Scan(&financeAfter); e != nil || financeAfter != financeBefore {
		t.Fatal("financial write", e, financeBefore, financeAfter)
	}
	// Real row lock and global allowance serialize different owners.
	_, e = pool.Exec(ctx, "UPDATE ai_experiment_budget SET allowance_micros=150080 WHERE singleton")
	if e != nil {
		t.Fatal(e)
	}
	p := candidate().Policy
	p.VerifiedAt = time.Now()
	p.InputMicrosPerMillion = 10000000
	p.OutputMicrosPerMillion = 10000000
	var mu sync.Mutex
	type hold struct {
		id  string
		ctx context.Context
	}
	ids := []hold{}
	var wg sync.WaitGroup
	for _, cx := range []context.Context{ownerCtx, otherCtx} {
		wg.Add(1)
		go func(cx context.Context) {
			defer wg.Done()
			id, e := budget.Reserve(cx, p, 10000, 1000, "receipt")
			if e == nil {
				mu.Lock()
				ids = append(ids, hold{id, cx})
				mu.Unlock()

			}
		}(cx)
	}
	wg.Wait()
	if len(ids) != 1 {
		t.Fatal("concurrent shared allowance exceeded", len(ids))
	}
	for _, held := range ids {
		zero := assistant.LLMResponse{ID: "not_sent", BillingKnown: true}
		if e := budget.Settle(held.ctx, held.id, &zero); e != nil {
			t.Fatal(e)
		}
	}

	if _, e = pool.Exec(ctx, "UPDATE ai_experiment_budget SET allowance_micros=10000000 WHERE singleton"); e != nil {
		t.Fatal(e)
	}
	id, e := budget.Reserve(ownerCtx, p, 100, 100, "statement")
	if e != nil {
		t.Fatal(e)
	}
	cancelled, cancel := context.WithCancel(ownerCtx)
	cancel()
	zero := assistant.LLMResponse{ID: "cancelled-zero", BillingKnown: true}
	if e = budget.Settle(cancelled, id, &zero); e != nil {
		t.Fatal("cancellation lost settlement", e)
	}
	id, e = budget.Reserve(ownerCtx, p, 100, 100, "statement")
	if e != nil {
		t.Fatal(e)
	}
	if e = budget.Settle(ownerCtx, id, nil); e != nil {
		t.Fatal(e)
	}
	if _, e = budget.Reserve(otherCtx, p, 100, 100, "receipt"); e == nil {
		t.Fatal("unknown bill did not freeze shared allowance")
	}
}
func executeSQL(t *testing.T, ctx context.Context, c Corpus, f *fakeHTTP, b Budget) (Manifest, Report) {
	t.Helper()
	m := NewManifest(c, sourceForSQL(), candidate())
	r, e := Run(ctx, c, m, grant(m), "offline-live-credential", f, b)
	if e != nil {
		t.Fatal(e)
	}
	return m, r
}
func sourceForSQL() string { return "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" }
