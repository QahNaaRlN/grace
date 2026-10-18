// statement-admin uses only the two shipped synthetic PDFs. It cannot import
// owner documents or write financial records; only shared usage is metered.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/QahNaaRlN/grace/internal/assistant"
	"github.com/QahNaaRlN/grace/internal/pdfextract"
	"github.com/QahNaaRlN/grace/internal/platform/access"
	"github.com/QahNaaRlN/grace/internal/platform/config"
	"github.com/QahNaaRlN/grace/internal/platform/database"
	"github.com/QahNaaRlN/grace/internal/statementextract"
	"github.com/QahNaaRlN/grace/internal/statementqualification"
	"github.com/jackc/pgx/v5/pgxpool"
)

type meter struct{ pool *database.TenantPool }

func (m meter) Reserve(ctx context.Context, n int64, p statementextract.Policy) (string, error) {
	var id string
	e := m.pool.QueryRow(ctx, "SELECT grace_ai_reserve($1,$2,$3,'statement')::text", n, p.Model, p.Provider).Scan(&id)
	return id, e
}
func (m meter) Settle(ctx context.Context, id string, n *int64) error {
	var actual any
	if n != nil {
		actual = *n
	}
	var ignored string
	return m.pool.QueryRow(ctx, "SELECT grace_ai_settle($1,$2::bigint,$3)::text", id, actual, "statement-qualification:"+id).Scan(&ignored)
}
func fail(code string) {
	_ = json.NewEncoder(os.Stdout).Encode(map[string]string{"error": code})
	os.Exit(1)
}
func main() {
	mode := flag.String("mode", "check", "check, policy, qualify (synthetic only), or summary")
	flag.Parse()
	if *mode == "summary" {
		var report statementqualification.Report
		decoder := json.NewDecoder(io.LimitReader(os.Stdin, 65537))
		decoder.DisallowUnknownFields()
		if decoder.Decode(&report) != nil || report.Version != "statement.synthetic.v1" || !report.Passed || len(report.Results) != 2 {
			fail("invalid_qualification_report")
		}
		for _, result := range report.Results {
			if !result.Passed {
				fail("invalid_qualification_report")
			}
		}
		_ = json.NewEncoder(os.Stdout).Encode(map[string]any{"qualificationPassed": true, "syntheticCases": len(report.Results), "qualifiedAt": report.Time})
		return
	}
	if *mode != "check" && *mode != "policy" && *mode != "qualify" {
		fail("invalid_mode")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	cfg, e := config.Load()
	if e != nil || !cfg.AIEnabled || cfg.OpenRouterAPIKey == "" {
		fail("global_ai_or_key_unavailable")
	}
	seed := assistant.ModelPolicy{Model: "google/gemini-2.5-flash", Provider: "google-vertex/eu", ProviderName: "Google", InputMicrosPerMillion: 300000, OutputMicrosPerMillion: 2500000, VerifiedAt: time.Unix(1, 0).UTC()}
	if *mode == "check" {
		if !cfg.StatementAIEnabled || json.Unmarshal([]byte(cfg.StatementAIPolicy), &seed) != nil {
			fail("statement_ai_disabled")
		}
	}
	verifier := assistant.NewPolicyVerifier()
	policy, e := verifier.Current(ctx, seed)
	if e != nil {
		fail("statement_policy_unavailable")
	}
	if *mode == "policy" {
		_ = json.NewEncoder(os.Stdout).Encode(policy)
		return
	}
	provider, e := assistant.NewOpenRouterProvider(cfg.OpenRouterAPIKey, nil)
	if e != nil {
		fail("provider_unavailable")
	}
	reasoner := &assistant.StatementReasoner{Verifier: verifier, Policy: policy, Provider: provider}
	if _, e = reasoner.CurrentPolicy(ctx); e != nil {
		fail("statement_policy_unavailable")
	}
	if *mode == "check" {
		fmt.Println(`{"statementAI":true,"policyVerified":true}`)
		return
	}
	pool, e := pgxpool.New(ctx, cfg.DatabaseURL)
	if e != nil {
		fail("database_unavailable")
	}
	defer pool.Close()
	var ids []string
	rows, e := pool.Query(ctx, "SELECT id::text FROM users WHERE role='admin' AND NOT disabled ORDER BY id")
	if e != nil {
		fail("qualification_owner_unavailable")
	}
	for rows.Next() {
		var id string
		if rows.Scan(&id) != nil {
			fail("qualification_owner_unavailable")
		}
		ids = append(ids, id)
	}
	e = rows.Err()
	rows.Close()
	if e != nil || len(ids) != 1 {
		fail("qualification_requires_single_active_admin")
	}
	ctx = access.WithPrincipal(ctx, access.Principal{ID: ids[0], Role: "admin"})
	client, e := pdfextract.NewClient(cfg.PDFExtractorSocket)
	if e != nil {
		fail("extractor_unavailable")
	}
	samples := []statementqualification.Sample{}
	for _, name := range []string{"columns", "cards"} {
		raw, e := os.ReadFile("/usr/share/grace/statement-qualification/" + name + ".pdf")
		if e != nil {
			fail("synthetic_fixture_unavailable")
		}
		d, e := client.Extract(ctx, raw)
		if e != nil {
			fail("synthetic_extraction_failed")
		}
		samples = append(samples, statementqualification.Sample{Name: name, Document: d})
	}
	report, e := statementqualification.Run(ctx, samples, reasoner, meter{database.ForUsers(pool)})
	_ = json.NewEncoder(os.Stdout).Encode(report)
	if e != nil {
		os.Exit(1)
	}
}
