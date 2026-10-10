//go:build integration

package assistant

import (
	"context"
	"os"
	"sync"
	"testing"

	"github.com/QahNaaRlN/grace/internal/platform/access"
	"github.com/QahNaaRlN/grace/internal/platform/database"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestGlobalAIUsageReservationIsolationAndUncertainBilling(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL required")
	}
	p, err := pgxpool.New(context.Background(), url)
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()
	// This test runs only in the dedicated disposable SPEC-0026 database script.
	if _, err = p.Exec(context.Background(), "UPDATE ai_experiment_budget SET allowance_micros=3000,fees_micros=0"); err != nil {
		t.Fatal(err)
	}
	type reservation struct {
		id  string
		ctx context.Context
	}
	var reservations []reservation
	var mu sync.Mutex
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		var owner string
		if err = p.QueryRow(context.Background(), "INSERT INTO users(display_name,role) VALUES('Synthetic AI budget user','member') RETURNING id::text").Scan(&owner); err != nil {
			t.Fatal(err)
		}
		ctx := access.WithPrincipal(context.Background(), access.Principal{ID: owner, Role: "member"})
		wg.Add(1)
		go func() {
			defer wg.Done()
			id, err := NewUsageRepository(p).Reserve(ctx, policyFixture(), 1000, 100, "analysis")
			if err == nil {
				mu.Lock()
				reservations = append(reservations, reservation{id, ctx})
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	if len(reservations) != 4 {
		t.Fatalf("global allowance race: got %d reservations", len(reservations))
	}
	meter := NewUsageRepository(p)
	remaining, err := meter.Remaining(reservations[0].ctx)
	if err != nil || remaining != 400 {
		t.Fatal(remaining, err)
	}
	if err = meter.Settle(reservations[1].ctx, reservations[0].id, &LLMResponse{ID: "foreign", CostMicros: 0}); err == nil {
		t.Fatal("foreign settlement allowed")
	}
	var visible int
	tenant := database.ForUsers(p)
	if err = tenant.QueryRow(reservations[0].ctx, "SELECT count(*) FROM ai_usage").Scan(&visible); err != nil || visible != 1 {
		t.Fatal("RLS leaked usage", visible, err)
	}
	if err = tenant.QueryRow(reservations[0].ctx, "UPDATE ai_experiment_budget SET allowance_micros=10000000 RETURNING allowance_micros").Scan(&visible); err == nil {
		t.Fatal("financial role expanded global allowance")
	}
	out := &LLMResponse{ID: "synthetic-generation", CostMicros: 51}
	if err = meter.Settle(reservations[0].ctx, reservations[0].id, out); err != nil {
		t.Fatal(err)
	}
	if err = meter.Settle(reservations[0].ctx, reservations[0].id, out); err != nil {
		t.Fatal("replay", err)
	}
	if err = meter.Settle(reservations[0].ctx, reservations[0].id, &LLMResponse{ID: out.ID, CostMicros: 50}); err == nil {
		t.Fatal("changed billing accepted")
	}
	remaining, err = meter.Remaining(reservations[0].ctx)
	if err != nil || remaining != 999 {
		t.Fatal(remaining, err)
	}
	if err = meter.Settle(reservations[1].ctx, reservations[1].id, nil); err != nil {
		t.Fatal(err)
	}
	if remaining, err = meter.Remaining(reservations[0].ctx); err != nil || remaining != 0 {
		t.Fatal("unknown billing unfrozen", remaining, err)
	}
	if _, err = meter.Reserve(reservations[0].ctx, policyFixture(), 1, 1, "analysis"); err == nil {
		t.Fatal("unknown bill allowed more calls")
	}
	if _, err = p.Exec(context.Background(), "UPDATE ai_experiment_budget SET allowance_micros=10000000,fees_micros=1"); err == nil {
		t.Fatal("fee exceeded authorized total")
	}
}
