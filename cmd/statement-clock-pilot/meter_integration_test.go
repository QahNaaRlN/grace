//go:build integration

package main

import (
	"context"
	"github.com/QahNaaRlN/grace/internal/platform/access"
	"github.com/QahNaaRlN/grace/internal/platform/database"
	"github.com/QahNaaRlN/grace/internal/statementextract"
	"github.com/QahNaaRlN/grace/internal/statementqualification"
	"github.com/jackc/pgx/v5/pgxpool"
	"os"
	"testing"
)

func TestClockMeterSQL(t *testing.T) {
	if os.Getenv("TEST_BANK_STATEMENTS") != "true" || os.Getenv("TEST_DATABASE_URL") == "" {
		t.Skip("disposable statement database required")
	}
	ctx := context.Background()
	pool, e := pgxpool.New(ctx, os.Getenv("TEST_DATABASE_URL"))
	if e != nil {
		t.Fatal(e)
	}
	defer pool.Close()
	var allowance int64
	if pool.QueryRow(ctx, "SELECT allowance_micros FROM ai_experiment_budget WHERE singleton").Scan(&allowance) != nil {
		t.Fatal("budget")
	}
	var owner string
	if pool.QueryRow(ctx, "INSERT INTO users(display_name,role) VALUES('S119 disposable fixture','admin') RETURNING id::text").Scan(&owner) != nil {
		t.Fatal("owner")
	}
	defer func() {
		_, _ = pool.Exec(context.Background(), "DELETE FROM ai_usage WHERE owner_id=$1::uuid", owner)
		_, _ = pool.Exec(context.Background(), "DELETE FROM users WHERE id=$1::uuid", owner)
		_, _ = pool.Exec(context.Background(), "UPDATE ai_experiment_budget SET allowance_micros=$1", allowance)
	}()
	if _, e = pool.Exec(ctx, "UPDATE ai_experiment_budget SET allowance_micros=1000000"); e != nil {
		t.Fatal(e)
	}
	ctx = access.WithPrincipal(ctx, access.Principal{ID: owner, Role: "admin"})
	m := clockSQLMeter{database.ForUsers(pool)}
	p := statementextract.ComparisonCandidates()[0].Policy
	if _, e = m.Reserve(ctx, 1, p); e == nil {
		t.Fatal("wrong reserve accepted")
	}
	id, e := m.Reserve(ctx, statementqualification.ClockPilotMaxMicros, p)
	if e != nil {
		t.Fatal(e)
	}
	if ok, e := m.ComparisonAllowed(ctx, id); e != nil || !ok {
		t.Fatal("before call", e)
	}
	actual := int64(60)
	if m.Settle(ctx, id, &actual) != nil {
		t.Fatal("settle")
	}
	var before string
	if pool.QueryRow(ctx, "SELECT row_to_json(u)::text FROM ai_usage u WHERE id=$1::uuid", id).Scan(&before) != nil {
		t.Fatal("saved record")
	}
	unknown, e := m.Reserve(ctx, statementqualification.ClockPilotMaxMicros, p)
	if e != nil {
		t.Fatal(e)
	}
	if m.Settle(ctx, unknown, nil) != nil {
		t.Fatal("unknown settle")
	}
	if ok, e := m.ComparisonAllowed(ctx, unknown); e != nil || ok {
		t.Fatal("uncertain allowed", e)
	}
	var after, model, provider, purpose, status string
	var reserved int64
	var cost *int64
	if pool.QueryRow(ctx, "SELECT model,provider,purpose,status,reserved_micros,actual_micros FROM ai_usage WHERE id=$1::uuid", unknown).Scan(&model, &provider, &purpose, &status, &reserved, &cost) != nil {
		t.Fatal("unknown row")
	}
	if model != "statement-clock.s119" || provider != "A/cards" || purpose != "statement" || status != "uncertain" || reserved != 323400 || cost != nil {
		t.Fatal(model, provider, purpose, status, reserved, cost)
	}
	if pool.QueryRow(ctx, "SELECT row_to_json(u)::text FROM ai_usage u WHERE id=$1::uuid", id).Scan(&after) != nil || after != before {
		t.Fatal("settled row changed")
	}
}
