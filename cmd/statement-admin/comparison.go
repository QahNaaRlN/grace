package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"github.com/QahNaaRlN/grace/internal/assistant"
	"github.com/QahNaaRlN/grace/internal/pdfextract"
	"github.com/QahNaaRlN/grace/internal/platform/access"
	"github.com/QahNaaRlN/grace/internal/platform/config"
	"github.com/QahNaaRlN/grace/internal/platform/database"
	"github.com/QahNaaRlN/grace/internal/statementdata"
	"github.com/QahNaaRlN/grace/internal/statementextract"
	"github.com/QahNaaRlN/grace/internal/statementqualification"
	"github.com/jackc/pgx/v5/pgxpool"
	"io"
	"os"
	"path/filepath"
	"time"
)

// A single ledger row represents the fixed A+B campaign, never a per-model
// allowance. The private report carries its exact reservation UUID.
type comparisonMeter struct{ meter }

func (m comparisonMeter) Reserve(ctx context.Context, n int64, _ statementextract.Policy) (string, error) {
	var id string
	e := m.pool.QueryRow(ctx, "SELECT grace_ai_reserve($1,'statement-comparison.s111','fixed-A-B','statement')::text", n).Scan(&id)
	return id, e
}
func comparisonMain(mode, dir string) {
	if mode == "comparison-summary" {
		r, e := statementqualification.DecodeComparisonReport(os.Stdin)
		if e != nil {
			fail("invalid_comparison_report")
		}
		out := comparisonSummary(r)
		_ = json.NewEncoder(os.Stdout).Encode(out)
		if out["reportShapeVerified"] != true {
			os.Exit(1)
		}
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 900*time.Second)
	defer cancel()
	cfg, e := config.Load()
	if e != nil || !cfg.AIEnabled || cfg.OpenRouterAPIKey == "" {
		fail("global_ai_or_key_unavailable")
	}
	if !filepath.IsAbs(dir) {
		fail("absolute_comparison_directory_required")
	}
	extractor, e := pdfextract.NewClient(cfg.PDFExtractorSocket)
	if e != nil {
		fail("extractor_unavailable")
	}
	samples := []statementqualification.Sample{}
	for _, name := range statementqualification.ScreeningNames() {
		f, e := os.Open(filepath.Join(dir, name+".pdf"))
		if e != nil {
			fail("comparison_fixture_unavailable")
		}
		raw, e := io.ReadAll(io.LimitReader(f, statementdata.MaxPDFBytes+1))
		_ = f.Close()
		h := sha256.Sum256(raw)
		digest := hex.EncodeToString(h[:])
		if e != nil || len(raw) > statementdata.MaxPDFBytes || digest != statementqualification.FixtureHash(name) {
			fail("invalid_comparison_fixture")
		}
		doc, e := extractor.Extract(ctx, raw)
		if e != nil {
			fail("comparison_extraction_failed")
		}
		samples = append(samples, statementqualification.Sample{Name: name, Document: doc, SHA256: digest})
	}
	var reasoners [2]statementextract.Reasoner
	for i, c := range statementextract.ComparisonCandidates() {
		reasoners[i], e = assistant.NewStatementComparisonReasoner(cfg.OpenRouterAPIKey, c.ID)
		if e != nil {
			fail("comparison_provider_unavailable")
		}
	}
	if _, e = statementqualification.ComparisonPreflight(ctx, samples, reasoners); e != nil {
		fail("comparison_preflight_failed")
	}
	pool, e := pgxpool.New(ctx, cfg.DatabaseURL)
	if e != nil {
		fail("database_unavailable")
	}
	defer pool.Close()
	ids := []string{}
	rows, e := pool.Query(ctx, "SELECT id::text FROM users WHERE role='admin' AND NOT disabled ORDER BY id")
	if e != nil {
		fail("comparison_owner_unavailable")
	}
	for rows.Next() {
		var id string
		if rows.Scan(&id) != nil {
			fail("comparison_owner_unavailable")
		}
		ids = append(ids, id)
	}
	e = rows.Err()
	rows.Close()
	if e != nil || len(ids) != 1 {
		fail("comparison_requires_single_active_admin")
	}
	ctx = access.WithPrincipal(ctx, access.Principal{ID: ids[0], Role: "admin"})
	if mode == "comparison-preflight" {
		var ready bool
		e = pool.QueryRow(ctx, "SELECT grace_ai_remaining() >= $1 AND NOT EXISTS(SELECT 1 FROM ai_usage WHERE status='uncertain')", statementqualification.ComparisonMaxMicros).Scan(&ready)
		if e != nil || !ready {
			fail("comparison_budget_unavailable")
		}
		_ = json.NewEncoder(os.Stdout).Encode(map[string]any{"comparisonPreflight": true, "candidates": 2, "tuples": 6, "providerCalls": 0, "activationAuthorized": false})
		return
	}
	report, e := statementqualification.RunComparison(ctx, samples, reasoners, comparisonMeter{meter{database.ForUsers(pool)}})
	_ = json.NewEncoder(os.Stdout).Encode(report)
	if e != nil {
		os.Exit(1)
	}
}
func comparisonSummary(r statementqualification.ComparisonReport) map[string]any {
	valid := statementqualification.ValidateComparisonReport(r, time.Now().UTC()) == nil
	out := map[string]any{"reportShapeVerified": valid, "comparisonPassed": valid && r.Passed, "activationAuthorized": false, "error": statementqualification.ComparisonErrorCode(r.Error)}
	if !valid {
		return out
	}
	tuples := []any{}
	for _, t := range r.Tuples {
		row := map[string]any{"candidate": t.Candidate, "name": t.Evidence.Name, "state": t.State, "reason": t.Reason, "falseAcceptance": t.FalseAcceptance}
		if t.Diagnostic != nil {
			row["diagnosticCode"] = t.Diagnostic.Sanitize().Code
		}
		if t.Result != nil {
			row["passed"] = t.Result.Passed
			row["checks"] = t.Result.Checks
			row["correct"] = t.Result.Correct
			row["codes"] = screeningPublicCodes(t.Result.Codes)
			row["validation"] = screeningPublicCodes(t.Result.Validation)
			row["coverage"] = screeningPublicCodes(t.Result.Coverage)
		}
		tuples = append(tuples, row)
	}
	out["tuples"] = tuples
	out["billingKnown"] = r.ActualMicros != nil
	return out
}

func (m comparisonMeter) ComparisonAllowed(ctx context.Context, id string) (bool, error) {
	var ready bool
	e := m.pool.QueryRow(ctx, "SELECT grace_statement_call_allowed($1::uuid)", id).Scan(&ready)
	return ready, e
}
