// Fixed public S1.19 operator. No product request can select it.
package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
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

var sourceSHA string // Set by the exact-source CI bundle build, never environment.
type clockSQLMeter struct{ pool *database.TenantPool }

func (m clockSQLMeter) Reserve(ctx context.Context, n int64, p statementextract.Policy) (string, error) {
	if n != statementqualification.ClockPilotMaxMicros || p != statementextract.ComparisonCandidates()[0].Policy {
		return "", errors.New("clock_budget_invalid")
	}
	var id string
	e := m.pool.QueryRow(ctx, "SELECT grace_ai_reserve($1,'statement-clock.s119','A/cards','statement')::text", n).Scan(&id)
	return id, e
}
func (m clockSQLMeter) Settle(ctx context.Context, id string, n *int64) error {
	var actual any
	if n != nil {
		actual = *n
	}
	var ignored string
	return m.pool.QueryRow(ctx, "SELECT grace_ai_settle($1,$2::bigint,$3)::text", id, actual, "statement-qualification:"+id).Scan(&ignored)
}
func (m clockSQLMeter) ComparisonAllowed(ctx context.Context, id string) (bool, error) {
	var ok bool
	e := m.pool.QueryRow(ctx, "SELECT grace_statement_call_allowed($1::uuid)", id).Scan(&ok)
	return ok, e
}
func fail(code string) {
	_ = json.NewEncoder(os.Stdout).Encode(map[string]string{"error": code})
	os.Exit(1)
}
func readDocument(dir string) (statementdata.Document, error) {
	var d statementdata.Document
	f, e := os.Open(filepath.Join(dir, "cards-document.json"))
	if e != nil {
		return d, e
	}
	defer f.Close()
	dec := json.NewDecoder(io.LimitReader(f, 32769))
	dec.DisallowUnknownFields()
	if dec.Decode(&d) != nil || dec.Decode(&struct{}{}) != io.EOF || d.Validate() != nil || d.Hash() != statementextract.ClockCardsDocumentHash {
		return d, errors.New("clock_fixture_invalid")
	}
	return d, nil
}
func main() {
	mode := flag.String("mode", "check", "check, run, summary")
	authorized := flag.String("authorized-source", "", "separately authorized exact source")
	flag.Parse()
	if *mode == "check" {
		_ = json.NewEncoder(os.Stdout).Encode(map[string]any{
			"version": statementqualification.ClockPilotVersion, "source": sourceSHA, "activationAuthorized": false,
			"basePromptSha256": statementextract.ClockBasePromptSHA, "candidatePromptSha256": statementextract.ClockCandidatePromptSHA,
			"documentHash": statementextract.ClockCardsDocumentHash, "order": statementqualification.ClockOrder(), "maximumMicros": statementqualification.ClockPilotMaxMicros,
			"inputBound": statementextract.InputBound, "outputBound": statementextract.OutputBound,
			"sampling": map[string]any{"reasoningEnabled": false, "stream": false, "temperature": "omitted/provider-default", "seed": "omitted"},
		})
		return
	}
	if !regexp.MustCompile(`^[0-9a-f]{40}$`).MatchString(sourceSHA) || *authorized != sourceSHA || flag.NArg() != 0 {
		fail("clock_source_not_authorized")
	}
	dir := "/tmp/grace-statement-clock-s119-" + sourceSHA
	doc, e := readDocument(dir)
	if e != nil {
		fail("clock_fixture_invalid")
	}
	if *mode == "summary" {
		raw, readErr := io.ReadAll(io.LimitReader(os.Stdin, statementqualification.ClockPilotReportLimit+1))
		if readErr != nil || len(raw) > statementqualification.ClockPilotReportLimit {
			fail("clock_report_invalid")
		}
		r, e := statementqualification.DecodeClockReport(bytes.NewReader(raw), doc)
		if e != nil {
			// Configuration/helper refusals happen before a typed campaign report.
			// Preserve their closed cause without inventing calls, cost or ledger state.
			var refusal map[string]string
			if len(raw) <= 256 && json.Unmarshal(raw, &refusal) == nil && len(refusal) == 1 && closedRefusal(refusal["error"]) {
				_ = json.NewEncoder(os.Stdout).Encode(map[string]any{"version": "statement.clock-pilot.s119.refusal.v1", "activationAuthorized": false, "decision": statementqualification.ClockInconclusive, "error": refusal["error"]})
				os.Exit(1)
			}
			fail("clock_report_invalid")
		}
		if r.Source != sourceSHA {
			fail("clock_report_invalid")
		}
		summary, e := statementqualification.SummarizeClock(r, doc)
		if e != nil {
			fail("clock_report_invalid")
		}
		_ = json.NewEncoder(os.Stdout).Encode(summary)
		return
	}
	if *mode != "run" {
		fail("clock_mode_invalid")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 900*time.Second)
	defer cancel()
	cfg, e := config.Load()
	if e != nil || !cfg.AIEnabled || cfg.OpenRouterAPIKey == "" {
		fail("clock_global_ai_unavailable")
	}
	f, e := os.Open(filepath.Join(dir, "cards.pdf"))
	if e != nil {
		fail("clock_fixture_invalid")
	}
	raw, e := io.ReadAll(io.LimitReader(f, statementdata.MaxPDFBytes+1))
	_ = f.Close()
	h := sha256.Sum256(raw)
	if e != nil || len(raw) > statementdata.MaxPDFBytes || hex.EncodeToString(h[:]) != statementqualification.FixtureHash("cards") {
		fail("clock_fixture_invalid")
	}
	helper, e := pdfextract.NewClient(cfg.PDFExtractorSocket)
	if e != nil {
		fail("clock_helper_unavailable")
	}
	actualDoc, e := helper.Extract(ctx, raw)
	if e != nil || actualDoc.Hash() != doc.Hash() {
		fail("clock_fixture_invalid")
	}
	pool, e := pgxpool.New(ctx, cfg.DatabaseURL)
	if e != nil {
		fail("clock_database_unavailable")
	}
	defer pool.Close()
	rows, e := pool.Query(ctx, "SELECT id::text FROM users WHERE role='admin' AND NOT disabled ORDER BY id")
	if e != nil {
		fail("clock_owner_unavailable")
	}
	ids := []string{}
	for rows.Next() {
		var id string
		if rows.Scan(&id) != nil {
			rows.Close()
			fail("clock_owner_unavailable")
		}
		ids = append(ids, id)
	}
	e = rows.Err()
	rows.Close()
	if e != nil || len(ids) != 1 {
		fail("clock_requires_single_admin")
	}
	ctx = access.WithPrincipal(ctx, access.Principal{ID: ids[0], Role: "admin"})
	reasoner, e := assistant.NewClockPilotReasoner(cfg.OpenRouterAPIKey)
	if e != nil {
		fail("clock_provider_unavailable")
	}
	r, runErr := statementqualification.RunClockPilot(ctx, statementqualification.Sample{Name: "cards", Document: actualDoc, SHA256: statementqualification.FixtureHash("cards")}, reasoner, clockSQLMeter{database.ForUsers(pool)}, sourceSHA)
	if statementqualification.EncodeClockReport(os.Stdout, r, actualDoc) != nil {
		fail("clock_report_invalid")
	}
	if runErr != nil {
		os.Exit(1)
	}
}

func closedRefusal(code string) bool {
	for _, allowed := range strings.Fields("clock_source_not_authorized clock_fixture_invalid clock_global_ai_unavailable clock_helper_unavailable clock_database_unavailable clock_owner_unavailable clock_requires_single_admin clock_provider_unavailable clock_report_invalid clock_mode_invalid") {
		if code == allowed {
			return true
		}
	}
	return false
}
