//go:build integration

package main

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/QahNaaRlN/grace/internal/platform/access"
	"github.com/QahNaaRlN/grace/internal/platform/database"
	"github.com/QahNaaRlN/grace/internal/statementdata"
	"github.com/QahNaaRlN/grace/internal/statementextract"
	"github.com/QahNaaRlN/grace/internal/statementqualification"
	"github.com/jackc/pgx/v5/pgxpool"
)

type s114UnknownSQLReasoner struct{ summaryComparisonFake }

func (f *s114UnknownSQLReasoner) Complete(context.Context, statementextract.Policy, statementextract.Chunk) (statementdata.Proposal, *statementextract.Billing, error) {
	f.calls++
	return statementdata.Proposal{}, nil, errors.New("PRIVATE lost response")
}

func TestS114StatementMeterSQL(t *testing.T) {
	if os.Getenv("TEST_BANK_STATEMENTS") != "true" || os.Getenv("TEST_DATABASE_URL") == "" {
		t.Skip("disposable statement database required")
	}
	pool, err := pgxpool.New(context.Background(), os.Getenv("TEST_DATABASE_URL"))
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	ctx := context.Background()
	var allowance int64
	if err = pool.QueryRow(ctx, "SELECT allowance_micros FROM ai_experiment_budget WHERE singleton").Scan(&allowance); err != nil {
		t.Fatal(err)
	}
	var owner string
	if err = pool.QueryRow(ctx, "INSERT INTO users(display_name,role) VALUES('S114 disposable SQL fixture','admin') RETURNING id::text").Scan(&owner); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if _, err := pool.Exec(context.Background(), "DELETE FROM ai_usage WHERE owner_id=$1::uuid", owner); err != nil {
			t.Error(err)
		}
		if _, err := pool.Exec(context.Background(), "DELETE FROM users WHERE id=$1::uuid", owner); err != nil {
			t.Error(err)
		}
		if _, err := pool.Exec(context.Background(), "UPDATE ai_experiment_budget SET allowance_micros=$1", allowance); err != nil {
			t.Error(err)
		}
	}()
	if _, err = pool.Exec(ctx, "UPDATE ai_experiment_budget SET allowance_micros=1000000"); err != nil {
		t.Fatal(err)
	}
	ctx = access.WithPrincipal(ctx, access.Principal{ID: owner, Role: "admin"})
	old := comparisonMeter{meter{database.ForUsers(pool)}}
	m := s114ComparisonMeter{old}
	oldID, err := old.Reserve(ctx, 208500, statementextract.Policy{})
	if err != nil {
		t.Fatal(err)
	}
	oldCost := int64(9)
	if err = old.Settle(ctx, oldID, &oldCost); err != nil {
		t.Fatal(err)
	}
	var before string
	if err = pool.QueryRow(ctx, "SELECT row_to_json(u)::text FROM ai_usage u WHERE id=$1::uuid", oldID).Scan(&before); err != nil {
		t.Fatal(err)
	}
	doc := statementdata.Document{Extractor: statementdata.ExtractorVersion, Pages: []statementdata.Page{{Number: 1, Width: 600, Height: 800}}, Spans: []statementdata.Span{{ID: "public", Page: 1, Block: 1, Line: 1, Word: 1, Text: "RUB", Box: statementdata.Box{XMin: 1, YMin: 1, XMax: 20, YMax: 10}}}}
	samples := []statementqualification.Sample{}
	for _, n := range statementqualification.ScreeningNames() {
		samples = append(samples, statementqualification.Sample{Name: n, Document: doc, SHA256: statementqualification.FixtureHash(n)})
	}
	cs := statementextract.ComparisonCandidates()
	a, b := &observedSummaryFake{summaryComparisonFake{candidate: cs[0]}}, &observedSummaryFake{summaryComparisonFake{candidate: cs[1]}}
	e, err := statementqualification.RunS114Comparison(ctx, samples, [2]statementextract.Reasoner{a, b}, m)
	if err == nil || a.calls != 1 || b.calls != 0 || statementqualification.ValidateS114Comparison(e, time.Now()) != nil {
		t.Fatal("causal canonical failure", err)
	}
	var model, provider, purpose, status, generation string
	var reserved, actual int64
	if err = pool.QueryRow(ctx, "SELECT model,provider,purpose,status,generation_id,reserved_micros,actual_micros FROM ai_usage WHERE id=$1::uuid AND owner_id=$2::uuid", e.Report.ReservationID, owner).Scan(&model, &provider, &purpose, &status, &generation, &reserved, &actual); err != nil {
		t.Fatal(err)
	}
	if model != "statement-comparison.s114" || provider != "fixed-A-B" || purpose != "statement" || status != "settled" || generation != "statement-qualification:"+e.Report.ReservationID || reserved != 208500 || actual != 42 || e.Report.ActualMicros == nil || *e.Report.ActualMicros != actual {
		t.Fatal("new exact reservation correlation")
	}
	var count int
	if err = pool.QueryRow(ctx, "SELECT count(*) FROM ai_usage WHERE owner_id=$1::uuid AND model='statement-comparison.s114'", owner).Scan(&count); err != nil || count != 1 {
		t.Fatal("not one reservation", err)
	}
	var after string
	if err = pool.QueryRow(ctx, "SELECT row_to_json(u)::text FROM ai_usage u WHERE id=$1::uuid", oldID).Scan(&after); err != nil || after != before {
		t.Fatal("historical row changed", err)
	}
	unknown := &s114UnknownSQLReasoner{summaryComparisonFake{candidate: cs[0]}}
	e, err = statementqualification.RunS114Comparison(ctx, samples, [2]statementextract.Reasoner{unknown, b}, m)
	if err == nil || unknown.calls != 1 || e.Report.ActualMicros != nil {
		t.Fatal("unknown billing became zero")
	}
	var actualUnknown *int64
	if err = pool.QueryRow(ctx, "SELECT status,actual_micros FROM ai_usage WHERE id=$1::uuid", e.Report.ReservationID).Scan(&status, &actualUnknown); err != nil || status != "uncertain" || actualUnknown != nil {
		t.Fatal("uncertain not preserved", err)
	}
	if _, err = m.Reserve(ctx, 208500, statementextract.Policy{}); err == nil {
		t.Fatal("shared uncertain block bypassed")
	}
	if _, err = m.Reserve(context.Background(), 208500, statementextract.Policy{}); err == nil {
		t.Fatal("missing owner accepted")
	}
}
