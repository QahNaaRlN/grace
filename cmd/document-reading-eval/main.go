// Research-only Q1–Q3 preparation. Live execution requires separate exact-source Q4.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"github.com/QahNaaRlN/grace/internal/assistant"
	q "github.com/QahNaaRlN/grace/internal/documentqualification"
	"github.com/QahNaaRlN/grace/internal/platform/access"
	"github.com/jackc/pgx/v5/pgxpool"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

var sourceSHA = "UNBOUND"

type credentials struct{ key, dsn, owner string }
type dependencies struct {
	executable  func() (string, error)
	credentials func() (credentials, error)
	run         func(context.Context, q.Corpus, q.Manifest, assistant.ExperimentGrant, credentials) (q.Report, error)
}

func main() {
	if e := command(os.Args[1:], os.Stdin, os.Stdout, dependencies{os.Executable, readCredentials, execute}); e != nil {
		fmt.Fprintln(os.Stderr, e.Error())
		os.Exit(1)
	}
}
func command(args []string, input io.Reader, out io.Writer, deps dependencies) error {
	flags := flag.NewFlagSet("document-reading-eval", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	mode := flags.String("mode", "check", "check, replay, summary, validate-public, run")
	root := flags.String("root", ".", "public source tree for check only")
	export := flags.String("export", "", "explicit new offline bundle directory")
	bundle := flags.String("bundle", "", "saved offline bundle for replay")
	authorized := flags.String("authorized-source", "", "separately approved built SHA")
	approvedManifest := flags.String("approved-manifest", "", "separately approved manifest SHA256")
	approval := flags.String("approval-id", "", "owner receipt commit")
	executeApproved := flags.Bool("execute-approved-reading", false, "separate Q4 grant required")
	if flags.Parse(args) != nil || flags.NArg() != 0 {
		return errors.New("arguments_invalid")
	}
	if *mode == "check" {
		if *executeApproved || *bundle != "" {
			return errors.New("arguments_invalid")
		}
		c, e := q.Load(*root)
		if e != nil {
			return errors.New("corpus_invalid")
		}
		raw, e := q.ReadPublic(*root, "internal/documentqualification/testdata/candidate.json", 16384)
		if e != nil {
			return errors.New("candidate_invalid")
		}
		candidate, e := q.DecodeCandidate(raw)
		if e != nil {
			return e
		}
		m := q.NewManifest(c, sourceSHA, candidate)
		if m.Validate(c, sourceSHA, false) != nil {
			return errors.New("manifest_invalid")
		}
		if *export != "" {
			if e = q.WriteBundle(*export, c, m); e != nil {
				return e
			}
		}
		return json.NewEncoder(out).Encode(m)
	}
	if *mode != "run" && *mode != "replay" && *mode != "summary" && *mode != "validate-public" {
		return errors.New("mode_invalid")
	}
	validSHA := regexp.MustCompile(`^[0-9a-f]{40}$`)
	if !validSHA.MatchString(sourceSHA) || *authorized != sourceSHA || *root != "." || *export != "" {
		return errors.New("source_not_authorized")
	}
	path, e := deps.executable()
	if e != nil {
		return errors.New("bundle_invalid")
	}
	dir := filepath.Dir(path)
	if *mode == "run" {
		if !*executeApproved || *bundle != "" || !validSHA.MatchString(*approval) || dir != "/tmp/grace-document-reading-q-"+sourceSHA {
			return errors.New("execution_not_authorized")
		}
	} else if *bundle != "" {
		dir = *bundle
	}
	c, m, e := q.LoadBundle(dir, sourceSHA)
	if e != nil {
		return e
	}
	if *mode == "validate-public" {
		raw, e := io.ReadAll(io.LimitReader(input, 65537))
		if e == nil {
			if s, e := q.ValidateSummary(raw, m); e == nil {
				return json.NewEncoder(out).Encode(s)
			}
		}
		_ = json.NewEncoder(out).Encode(map[string]string{"version": q.Version, "source": sourceSHA, "manifestSHA256": m.Hash(), "status": "report_unavailable"})
		return errors.New("summary_invalid")
	}
	if *mode == "replay" || *mode == "summary" {
		raw, e := io.ReadAll(io.LimitReader(input, q.ReportLimit+1))
		if e != nil {
			return errors.New("report_invalid")
		}
		r, e := q.Replay(context.Background(), c, m, raw)
		if e != nil {
			return e
		}
		return json.NewEncoder(out).Encode(q.Summarize(r))
	}
	if m.Validate(c, sourceSHA, true) != nil || *approvedManifest != m.Hash() {
		return errors.New("manifest_not_authorized")
	}
	grant := assistant.ExperimentGrant{Source: sourceSHA, Approval: *approval, ManifestSHA256: m.Hash(), ExpiresAt: time.Now().Add(35 * time.Minute)}
	// Structural policy validation precedes any environment or credential access.
	if _, e = assistant.NewExperimentModelGroups(*m.Candidate, grant, "", nil); e != nil {
		return errors.New("candidate_invalid")
	}
	secrets, e := deps.credentials()
	if e != nil {
		return errors.New("credentials_unavailable")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()
	r, e := deps.run(ctx, c, m, grant, secrets)
	if e != nil {
		return errors.New("campaign_unavailable")
	}
	raw, e := q.EncodeReport(r)
	if e != nil {
		return e
	}
	_, e = out.Write(append(raw, '\n'))
	return e
}
func readCredentials() (credentials, error) {
	var c credentials
	if os.Getenv("GRACE_AI_ENABLED") != "true" {
		return c, errors.New("ai_disabled")
	}
	c.key = strings.TrimSpace(os.Getenv("OPENROUTER_API_KEY"))
	file := strings.TrimSpace(os.Getenv("OPENROUTER_API_KEY_FILE"))
	if file != "" {
		if c.key != "" || !filepath.IsAbs(file) {
			return c, errors.New("key_invalid")
		}
		f, e := os.Open(file)
		if e != nil {
			return c, errors.New("key_invalid")
		}
		raw, e := io.ReadAll(io.LimitReader(f, 8193))
		_ = f.Close()
		if e != nil || len(raw) > 8192 {
			return c, errors.New("key_invalid")
		}
		c.key = strings.TrimSpace(string(raw))
	}
	c.dsn = os.Getenv("DATABASE_URL")
	c.owner = os.Getenv("GRACE_STATEMENT_OPERATOR_OWNER_ID")
	if len(c.key) < 16 || len(c.key) > 8192 || strings.ContainsAny(c.key, "\r\n") || c.dsn == "" || !regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`).MatchString(c.owner) {
		return credentials{}, errors.New("credentials_unavailable")
	}
	return c, nil
}
func execute(ctx context.Context, c q.Corpus, m q.Manifest, g assistant.ExperimentGrant, secret credentials) (q.Report, error) {
	pool, e := pgxpool.New(ctx, secret.dsn)
	if e != nil {
		return q.Report{}, errors.New("database_unavailable")
	}
	defer pool.Close()
	ctx = access.WithPrincipal(ctx, access.Principal{ID: secret.owner, Role: "admin"})
	return q.Run(ctx, c, m, g, secret.key, nil, q.NewSQLBudget(pool))
}
