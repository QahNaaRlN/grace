// Research only. Default check never reads env/keys or opens a network connection.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"github.com/QahNaaRlN/grace/internal/statementextract"
	"github.com/QahNaaRlN/grace/internal/statementqualification"
)

var sourceSHA = "UNBOUND"

const keyEnvironment = "GRACE_STATEMENT_PAGE_OPENAI_KEY"

type allowance struct {
	Source                      string `json:"source"`
	Approval                    string `json:"approval"`
	ManifestSHA256              string `json:"manifestSHA256"`
	CeilingMicros               int64  `json:"ceilingMicros"`
	State                       string `json:"state"`
	Endpoint                    string `json:"endpoint"`
	PricingConfirmed            bool   `json:"pricingConfirmed"`
	FundingConfirmed            bool   `json:"fundingConfirmed"`
	AccountModelAccessConfirmed bool   `json:"accountModelAccessConfirmed"`
	RegionPrivacyConfirmed      bool   `json:"regionPrivacyConfirmed"`
	NoAdditionalFeesConfirmed   bool   `json:"noAdditionalFeesConfirmed"`
}
type receipt struct {
	Source         string                               `json:"source"`
	Approval       string                               `json:"approval"`
	ManifestSHA256 string                               `json:"manifestSHA256"`
	ReservedMicros int64                                `json:"reservedMicros"`
	State          string                               `json:"state"`
	Calls          int                                  `json:"calls"`
	Report         *statementqualification.DirectReport `json:"report"`
}
type dependencies struct {
	key  func(string) string
	call func(context.Context, statementextract.DirectRequest, []string, string) statementextract.DirectObservation
}

func main() {
	deps := dependencies{key: os.Getenv, call: func(ctx context.Context, r statementextract.DirectRequest, models []string, key string) statementextract.DirectObservation {
		return (statementextract.DirectResponses{Key: key}).Do(ctx, r, models)
	}}
	if e := command(os.Args[1:], os.Stdout, deps); e != nil {
		fmt.Fprintln(os.Stderr, e)
		os.Exit(1)
	}
}
func command(args []string, w io.Writer, deps dependencies) error {
	flags := flag.NewFlagSet("statement-page-reading", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	mode := flags.String("mode", "check", "check, protocol, replay or run")
	root := flags.String("root", ".", "public source tree")
	execute := flags.Bool("execute-approved-reading", false, "future exact-source owner grant required")
	source := flags.String("authorized-source", "", "exact built source")
	approval := flags.String("approval-id", "", "one grant identifier")
	allowancePath := flags.String("allowance", "", "explicit local research allowance (never a production budget)")
	reportPath := flags.String("report", "", "saved public report for replay")
	if flags.Parse(args) != nil || flags.NArg() != 0 {
		return errors.New("invalid_arguments")
	}
	if *mode == "protocol" {
		return json.NewEncoder(w).Encode(map[string]string{"promptSHA256": statementextract.DirectHash([]byte(statementextract.DirectPrompt)), "schemaSHA256": statementextract.DirectHash(statementextract.DirectSchema())})
	}
	if *mode == "check" {
		c, e := statementqualification.LoadDirectCorpus(*root)
		if e != nil {
			return e
		}
		requests := []map[string]any{}
		for _, item := range c.Cases {
			requests = append(requests, map[string]any{"case": item.Name, "pages": len(item.Input.Pages), "rows": len(item.Golden.Rows), "requestSHA256": item.Request.SHA256, "inputTokenBound": item.Request.InputTokenBound, "imageSHA256": item.Request.ImageSHA256})
		}
		return json.NewEncoder(w).Encode(map[string]any{"profile": statementextract.DirectProfile, "manifestSHA256": c.ManifestSHA256, "cases": 4, "requests": requests, "calls": 0, "importReady": false, "V4": "NOT_AUTHORIZED"})
	}
	if *mode == "replay" {
		c, e := statementqualification.LoadDirectCorpus(*root)
		if e != nil {
			return e
		}
		raw, e := readPrivate(*reportPath, 16<<20, false)
		if e != nil {
			return e
		}
		r, e := statementqualification.VerifyDirectReport(c, raw, *source, *approval)
		if e != nil {
			return e
		}
		return json.NewEncoder(w).Encode(r)
	}
	if *mode != "run" {
		return errors.New("invalid_mode")
	}
	// Claim before preflight. Rejection, crash and uncertain calls all retain this
	// one-shot receipt. There is no retry or automatic release/cleanup operation.
	if *allowancePath == "" {
		return errors.New("allowance_required")
	}
	path := filepath.Clean(*allowancePath) + ".consumed"
	f, e := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if e != nil {
		return errors.New("one_shot_unavailable")
	}
	rec := receipt{Source: *source, Approval: *approval, State: "claimed"}
	save := func() error {
		raw, e := json.MarshalIndent(rec, "", "  ")
		if e != nil {
			return e
		}
		if _, e = f.Seek(0, 0); e != nil {
			return e
		}
		if e = f.Truncate(0); e != nil {
			return e
		}
		if _, e = f.Write(raw); e != nil {
			return e
		}
		if e = f.Sync(); e != nil {
			return e
		}
		dir, e := os.Open(filepath.Dir(path))
		if e != nil {
			return e
		}
		defer dir.Close()
		return dir.Sync()
	}
	defer f.Close()
	rec.State = "refused_before_calls"
	if e = save(); e != nil {
		return errors.New("receipt_write")
	}
	if !*execute || !regexp.MustCompile(`^[0-9a-f]{40}$`).MatchString(sourceSHA) || *source != sourceSHA || !regexp.MustCompile(`^[A-Za-z0-9_-]{1,80}$`).MatchString(*approval) {
		return errors.New("source_execute_approval_gate")
	}
	c, e := statementqualification.LoadDirectCorpus(*root)
	if e != nil {
		return e
	}
	rec.ManifestSHA256 = c.ManifestSHA256
	raw, e := readPrivate(*allowancePath, 16384, true)
	if e != nil {
		return e
	}
	var a allowance
	if statementextract.DirectJSON(raw, &a) != nil {
		return errors.New("allowance_invalid")
	}
	if a.Source != *source || a.Approval != *approval || a.ManifestSHA256 != c.ManifestSHA256 || a.State != "available" || a.CeilingMicros != statementextract.DirectCampaignCeilingMicros || a.Endpoint != "https://api.openai.com/v1/responses" || !a.PricingConfirmed || !a.FundingConfirmed || !a.AccountModelAccessConfirmed || !a.RegionPrivacyConfirmed || !a.NoAdditionalFeesConfirmed {
		return errors.New("allowance_not_confirmed")
	}
	rec.State = "reserved"
	rec.ReservedMicros = statementextract.DirectCampaignCeilingMicros
	if e = save(); e != nil {
		return errors.New("receipt_write")
	}
	// Key is deliberately accessed only after source, execution, fixed requests,
	// funding and durable reservation. Tests inject a spy; offline work never calls run.
	key := deps.key(keyEnvironment)
	if key == "" || len(key) > 1024 {
		rec.State = "refused_before_calls"
		save()
		return errors.New("key_unavailable")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 1500*time.Second)
	defer cancel()
	captures := []statementqualification.DirectCapture{}
	rec.State = "uncertain"
	if e = save(); e != nil {
		return errors.New("receipt_write")
	}
	var report statementqualification.DirectReport
	for i := 0; i < 8; i++ {
		if ctx.Err() != nil {
			break
		}
		o := deps.call(ctx, c.Cases[i%4].Request, c.Manifest.ReturnedModels, key)
		rec.Calls++
		capture := statementqualification.DirectCapture{Response: o.Response, DurationMS: o.DurationMS}
		if len(o.Response) == 0 {
			capture.Failure = o.Reason
		}
		captures = append(captures, capture)
		report, e = statementqualification.ReplayDirect(c, *source, *approval, captures)
		if e != nil {
			save()
			return errors.New("replay_failed_uncertain")
		}
		rec.Report = &report
		if e = save(); e != nil {
			return errors.New("receipt_write_uncertain")
		}
		last := report.Attempts[i]
		if report.Decision == "REJECT" || last.State != "completed" {
			break
		}
	}
	if rec.Report == nil {
		return errors.New("campaign_timeout_uncertain")
	}
	rec.State = report.AllowanceState
	if e = save(); e != nil {
		return errors.New("receipt_write_uncertain")
	}
	return json.NewEncoder(w).Encode(report)
}
func readPrivate(path string, limit int64, private bool) ([]byte, error) {
	s, e := os.Lstat(path)
	if e != nil || !s.Mode().IsRegular() || s.Mode()&os.ModeSymlink != 0 || s.Size() > limit || (private && s.Mode().Perm()&0077 != 0) {
		return nil, errors.New("local_file_invalid")
	}
	f, e := os.Open(path)
	if e != nil {
		return nil, errors.New("local_file_unavailable")
	}
	defer f.Close()
	return io.ReadAll(io.LimitReader(f, limit+1))
}
