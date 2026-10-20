package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
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
		_ = json.NewEncoder(os.Stdout).Encode(screeningSummary(r))
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

func screeningSummary(r statementqualification.Report) map[string]any {
	valid := statementqualification.ValidateScreeningReport(r, time.Now()) == nil
	results := []any{}
	for _, s := range r.Results {
		if s.Name != "columns" && s.Name != "cards" && s.Name != "turnover-roles-ru" {
			continue
		}
		results = append(results, map[string]any{"name": s.Name, "passed": s.Passed, "checks": s.Checks, "correct": s.Correct, "codes": screeningPublicCodes(s.Codes), "validation": screeningPublicCodes(s.Validation), "coverage": screeningPublicCodes(s.Coverage)})
	}
	classes := []string{}
	for i, call := range r.Calls {
		if i >= 3 {
			break
		}
		if call.ScreeningRoute != nil {
			classes = append(classes, call.ScreeningRoute.Sanitize().SelectedModelClass)
		}
	}
	diagnosticCode := ""
	selectedClass := "not_observed"
	if r.Failure != nil {
		diagnosticCode = r.Failure.Sanitize().Code
		if route := r.Failure.Sanitize().Route; route != nil {
			selectedClass = route.SelectedModelClass
		}
	}
	errorCode := statementqualification.ScreeningErrorCode(r.Error)
	return map[string]any{"screeningPassed": valid, "error": errorCode, "diagnosticCode": diagnosticCode, "selectedModelClass": selectedClass, "selectedModelClasses": classes, "results": results, "activationAuthorized": false}
}

func screeningPublicCodes(in []string) []string {
	out := []string{}
	for _, code := range in {
		switch code {
		case "row_count", "account_amount_currency", "direction", "operation_clock", "booking_clock", "record_description", "row_issues", "fx_column", "turnover", "coverage", "blockers", "absent_book_balances", "available_balance", "book_balances", "opening_book_value", "closing_book_value", "absent_booking", "absent_fx", "printed_debit_turnover", "printed_credit_turnover", "absent_book_dates",
			"account_amount_required", "aggregate_overflow", "ambiguous_local_time", "ambiguous_money", "amount_overflow", "available_is_not_book_balance", "book_balance_mismatch", "book_balance_not_checked", "book_balance_period_not_proven", "currency_conflict", "currency_required", "description_evidence_limit", "direction_conflict", "direction_required", "field_evidence_limit", "financial_candidate_in_description", "historical_anchor_not_proven", "invalid_currency", "invalid_date", "invalid_direction", "invalid_span_reference", "invalid_timezone", "legacy_field_evidence_invalid", "legacy_service_mapping_missing", "mapping_review_missing", "mapping_snapshot_invalid", "missing_evidence", "no_financial_confirmation", "noncontiguous_date_evidence", "noncontiguous_evidence", "operation_date_required", "overlapping_evidence", "owner_field_role_review_required", "owner_page_review_required", "printed_turnover_mismatch", "service_reason_required", "source_coverage_incomplete", "statement_row_limit", "transaction_direction_conflict", "turnover_not_checked", "turnover_role_unverified", "unsupported_account_currency", "zero_posting":
			out = append(out, code)
		default:
			out = append(out, "unknown")
		}
	}
	return out
}
