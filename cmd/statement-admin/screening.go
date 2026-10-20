package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"github.com/QahNaaRlN/grace/internal/assistant"
	"github.com/QahNaaRlN/grace/internal/pdfextract"
	"github.com/QahNaaRlN/grace/internal/platform/access"
	"github.com/QahNaaRlN/grace/internal/platform/config"
	"github.com/QahNaaRlN/grace/internal/platform/database"
	"github.com/QahNaaRlN/grace/internal/statementdata"
	"github.com/QahNaaRlN/grace/internal/statementextract"
	"github.com/QahNaaRlN/grace/internal/statementqualification"
	"github.com/jackc/pgx/v5/pgxpool"
)

func screeningMain(mode, dir string) {
	if mode == "screening-summary" {
		r, e := statementqualification.DecodeReport(os.Stdin)
		if e != nil || r.Version != statementqualification.ScreeningReportVersion {
			fail("invalid_screening_report")
		}
		valid := statementqualification.ValidateScreeningReport(r, time.Now()) == nil
		code := regexp.MustCompile(`^[a-zA-Z0-9_:.-]{1,120}$`)
		safe := func(in []string) []string {
			out := []string{}
			for _, s := range in {
				if code.MatchString(s) {
					out = append(out, s)
				}
			}
			return out
		}
		results := []any{}
		for _, s := range r.Results {
			if s.Name != "columns" && s.Name != "cards" && s.Name != "turnover-roles-ru" {
				continue
			}
			results = append(results, map[string]any{"name": s.Name, "passed": s.Passed, "checks": s.Checks, "correct": s.Correct, "codes": safe(s.Codes), "validation": safe(s.Validation), "coverage": safe(s.Coverage)})
		}
		errorCode := ""
		if code.MatchString(r.Error) {
			errorCode = r.Error
		}
		_ = json.NewEncoder(os.Stdout).Encode(map[string]any{"screeningPassed": valid, "error": errorCode, "results": results, "activationAuthorized": false})
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 420*time.Second)
	defer cancel()
	cfg, e := config.Load()
	if e != nil || !cfg.AIEnabled || cfg.OpenRouterAPIKey == "" {
		fail("global_ai_or_key_unavailable")
	}
	if !filepath.IsAbs(dir) {
		fail("absolute_screening_directory_required")
	}
	extractor, e := pdfextract.NewClient(cfg.PDFExtractorSocket)
	if e != nil {
		fail("extractor_unavailable")
	}
	samples := []statementqualification.Sample{}
	for _, name := range statementqualification.ScreeningNames() {
		f, e := os.Open(filepath.Join(dir, name+".pdf"))
		if e != nil {
			fail("screening_fixture_unavailable")
		}
		raw, e := io.ReadAll(io.LimitReader(f, statementdata.MaxPDFBytes+1))
		_ = f.Close()
		digest := sha256.Sum256(raw)
		hash := hex.EncodeToString(digest[:])
		if e != nil || len(raw) > statementdata.MaxPDFBytes || hash != statementqualification.FixtureHash(name) {
			fail("invalid_screening_fixture")
		}
		d, e := extractor.Extract(ctx, raw)
		if e != nil {
			fail("screening_extraction_failed")
		}
		if _, e = statementextract.BuildScreeningPlan(d, statementextract.ScreeningPolicy()); e != nil {
			fail("screening_plan_invalid")
		}
		samples = append(samples, statementqualification.Sample{Name: name, Document: d, SHA256: hash})
	}
	reasoner, e := assistant.NewStatementScreeningReasoner(cfg.OpenRouterAPIKey)
	if e != nil {
		fail("screening_provider_unavailable")
	}
	if _, e = reasoner.CurrentPolicy(ctx); e != nil {
		fail("screening_policy_or_privacy_unavailable")
	}
	pool, e := pgxpool.New(ctx, cfg.DatabaseURL)
	if e != nil {
		fail("database_unavailable")
	}
	defer pool.Close()
	ids := []string{}
	rows, e := pool.Query(ctx, "SELECT id::text FROM users WHERE role='admin' AND NOT disabled ORDER BY id")
	if e != nil {
		fail("screening_owner_unavailable")
	}
	for rows.Next() {
		var id string
		if rows.Scan(&id) != nil {
			fail("screening_owner_unavailable")
		}
		ids = append(ids, id)
	}
	e = rows.Err()
	rows.Close()
	if e != nil || len(ids) != 1 {
		fail("screening_requires_single_active_admin")
	}
	ctx = access.WithPrincipal(ctx, access.Principal{ID: ids[0], Role: "admin"})
	if mode == "screening-preflight" {
		var ready bool
		e = pool.QueryRow(ctx, "SELECT grace_ai_remaining() >= $1 AND NOT EXISTS(SELECT 1 FROM ai_usage WHERE status='uncertain')", statementqualification.ScreeningMaxMicros).Scan(&ready)
		if e != nil || !ready {
			fail("screening_budget_unavailable")
		}
		_ = json.NewEncoder(os.Stdout).Encode(map[string]any{"screeningPreflight": true, "cases": 3, "providerCalls": 0, "activationAuthorized": false})
		return
	}
	report, e := statementqualification.RunScreening(ctx, samples, reasoner, meter{database.ForUsers(pool)})
	_ = json.NewEncoder(os.Stdout).Encode(report)
	if e != nil {
		os.Exit(1)
	}
}
