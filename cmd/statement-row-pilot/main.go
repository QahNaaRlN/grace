// Fixed public S1.21 operator. No product request can select it.
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
type rowSQLMeter struct{ pool *database.TenantPool }

func (m rowSQLMeter) Reserve(ctx context.Context, n int64, p statementextract.Policy) (string, error) {
	if n != statementqualification.RowPilotMaxMicros || p != statementextract.ComparisonCandidates()[0].Policy {
		return "", errors.New("row_pilot_budget_invalid")
	}
	var id string
	e := m.pool.QueryRow(ctx, "SELECT grace_ai_reserve($1,'statement-row.s121','A/cards','statement')::text", n).Scan(&id)
	return id, e
}
func (m rowSQLMeter) Settle(ctx context.Context, id string, n *int64) error {
	var actual any
	if n != nil {
		actual = *n
	}
	var ignored string
	return m.pool.QueryRow(ctx, "SELECT grace_ai_settle($1,$2::bigint,$3)::text", id, actual, "statement-qualification:"+id).Scan(&ignored)
}
func (m rowSQLMeter) ComparisonAllowed(ctx context.Context, id string) (bool, error) {
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
		return d, errors.New("row_pilot_fixture_invalid")
	}
	return d, nil
}
func main() {
	flags := flag.NewFlagSet("statement-row-pilot", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	mode := flags.String("mode", "check", "check, run, summary, validate-public")
	authorized := flags.String("authorized-source", "", "separately authorized exact source")
	execute := flags.Bool("execute-approved-pilot", false, "separately approved one-shot only")
	if flags.Parse(os.Args[1:]) != nil || flags.NArg() != 0 {
		fail("row_pilot_arguments_invalid")
	}
	if *mode == "check" {
		doc, e := statementextract.RowPilotPublicDocument()
		if e != nil {
			fail("row_pilot_fixture_invalid")
		}
		base, e := statementextract.BuildRowPilotBase(doc)
		if e != nil {
			fail("row_pilot_fixture_invalid")
		}
		group, e := statementextract.BuildRowGrouping(doc)
		if e != nil {
			fail("row_pilot_fixture_invalid")
		}
		plan, e := statementextract.BuildComparisonPlan(doc, base.Policy)
		if e != nil {
			fail("row_pilot_fixture_invalid")
		}
		_ = json.NewEncoder(os.Stdout).Encode(map[string]any{
			"version": statementqualification.RowPilotVersion, "source": sourceSHA, "activationAuthorized": false, "executionAuthorized": false,
			"documentHash": doc.Hash(), "fixtureHash": statementqualification.FixtureHash("cards"), "planHash": statementextract.Hash(plan),
			"baseRequest": base.Hash, "baseSchema": statementextract.PromptSHA(string(base.Schema)), "groupingRequest": group.Hash, "groupingSchema": statementextract.PromptSHA(string(group.Schema)),
			"basePromptSha256": statementextract.ClockBasePromptSHA, "groupingPromptSha256": statementextract.RowGroupingPromptSHA, "rowPromptSha256": statementextract.RowFieldPromptSHA,
			"rowRequests": "derived_from_actual_grouping_each_attempt", "order": statementqualification.RowPilotOrder(), "maximumMicros": statementqualification.RowPilotMaxMicros,
			"maximumPOST": 12, "maximumMetadataGET": 39, "inputBound": statementextract.InputBound, "outputBound": statementextract.OutputBound, "responseLimit": statementextract.RowScopedResponseLimit,
			"operatorPrincipal": "private_GRACE_STATEMENT_OPERATOR_OWNER_ID_required", "sampling": map[string]any{"reasoningEnabled": false, "stream": false, "temperature": "omitted/provider-default", "seed": "omitted"},
		})
		return
	}
	if !regexp.MustCompile(`^[0-9a-f]{40}$`).MatchString(sourceSHA) || *authorized != sourceSHA {
		fail("row_pilot_source_not_authorized")
	}
	if *mode == "validate-public" {
		raw, e := io.ReadAll(io.LimitReader(os.Stdin, statementqualification.RowPilotReportLimit+1))
		if e == nil && len(raw) <= statementqualification.RowPilotReportLimit {
			if s, e := statementqualification.DecodeRowPilotSummary(bytes.NewReader(raw), sourceSHA); e == nil {
				_ = json.NewEncoder(os.Stdout).Encode(s)
				return
			}
			if s, e := statementqualification.DecodeRowPilotPublicRefusal(raw, sourceSHA); e == nil {
				_ = json.NewEncoder(os.Stdout).Encode(s)
				os.Exit(1)
			}
			if code, ok := statementqualification.DecodeRowPilotRefusal(raw); ok {
				_ = json.NewEncoder(os.Stdout).Encode(statementqualification.NewRowPilotRefusal(sourceSHA, code))
				os.Exit(1)
			}
		}
		_ = json.NewEncoder(os.Stdout).Encode(statementqualification.NewRowPilotRefusal(sourceSHA, "row_pilot_report_unavailable"))
		os.Exit(1)
	}
	if *mode == "run" && !*execute {
		fail("row_pilot_authorization_missing")
	}
	dir := "/tmp/grace-statement-row-s121-" + sourceSHA
	doc, e := readDocument(dir)
	if e != nil {
		fail("row_pilot_fixture_invalid")
	}
	if *mode == "summary" {
		raw, readErr := io.ReadAll(io.LimitReader(os.Stdin, statementqualification.RowPilotReportLimit+1))
		if readErr != nil || len(raw) > statementqualification.RowPilotReportLimit {
			fail("row_pilot_report_invalid")
		}
		r, e := statementqualification.DecodeRowPilotReport(bytes.NewReader(raw), doc)
		if e != nil {
			// Configuration/helper refusals happen before a typed campaign report.
			// Preserve their closed cause without inventing calls, cost or ledger state.
			if code, ok := statementqualification.DecodeRowPilotRefusal(raw); ok {
				_ = json.NewEncoder(os.Stdout).Encode(statementqualification.NewRowPilotRefusal(sourceSHA, code))
				os.Exit(1)
			}
			fail("row_pilot_report_invalid")
		}
		if r.Source != sourceSHA {
			fail("row_pilot_report_invalid")
		}
		summary, e := statementqualification.SummarizeRowPilot(r, doc)
		if e != nil {
			fail("row_pilot_report_invalid")
		}
		_ = json.NewEncoder(os.Stdout).Encode(summary)
		return
	}
	if *mode != "run" {
		fail("row_pilot_mode_invalid")
	}
	if !*execute {
		fail("row_pilot_authorization_missing")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 1800*time.Second)
	defer cancel()
	cfg, e := config.Load()
	if e != nil || !cfg.AIEnabled || cfg.OpenRouterAPIKey == "" {
		fail("row_pilot_global_ai_unavailable")
	}
	f, e := os.Open(filepath.Join(dir, "cards.pdf"))
	if e != nil {
		fail("row_pilot_fixture_invalid")
	}
	raw, e := io.ReadAll(io.LimitReader(f, statementdata.MaxPDFBytes+1))
	_ = f.Close()
	h := sha256.Sum256(raw)
	if e != nil || len(raw) > statementdata.MaxPDFBytes || hex.EncodeToString(h[:]) != statementqualification.FixtureHash("cards") {
		fail("row_pilot_fixture_invalid")
	}
	helper, e := pdfextract.NewClient(cfg.PDFExtractorSocket)
	if e != nil {
		fail("row_pilot_helper_unavailable")
	}
	actualDoc, e := helper.Extract(ctx, raw)
	if e != nil || actualDoc.Hash() != doc.Hash() {
		fail("row_pilot_fixture_invalid")
	}
	pool, e := pgxpool.New(ctx, cfg.DatabaseURL)
	if e != nil {
		fail("row_pilot_database_unavailable")
	}
	defer pool.Close()
	owner := strings.TrimSpace(os.Getenv("GRACE_STATEMENT_OPERATOR_OWNER_ID"))
	if !regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`).MatchString(owner) {
		fail("row_pilot_operator_owner_unavailable")
	}
	ctx = access.WithPrincipal(ctx, access.Principal{ID: owner, Role: "admin"})
	reasoner, e := assistant.NewRowPilotReasoner(cfg.OpenRouterAPIKey, actualDoc)
	if e != nil {
		fail("row_pilot_provider_unavailable")
	}
	r, runErr := statementqualification.RunRowPilot(ctx, statementqualification.Sample{Name: "cards", Document: actualDoc, SHA256: statementqualification.FixtureHash("cards")}, reasoner, rowSQLMeter{database.ForUsers(pool)}, sourceSHA)
	if statementqualification.EncodeRowPilotReport(os.Stdout, r, actualDoc) != nil {
		fail("row_pilot_report_invalid")
	}
	if runErr != nil {
		os.Exit(1)
	}
}
