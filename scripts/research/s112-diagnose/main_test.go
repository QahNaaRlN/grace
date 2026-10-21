package main

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/QahNaaRlN/grace/internal/statementdata"
	"github.com/QahNaaRlN/grace/internal/statementextract"
	"github.com/QahNaaRlN/grace/internal/statementqualification"
)

type fake struct {
	c     statementextract.ComparisonCandidate
	mode  string
	calls int
}

func (f *fake) CurrentPolicy(context.Context) (statementextract.Policy, error) {
	return f.c.Policy, nil
}
func (f *fake) ComparisonProof(context.Context) (statementextract.ComparisonIdentityProof, error) {
	return proof(f.c), nil
}
func proof(c statementextract.ComparisonCandidate) statementextract.ComparisonIdentityProof {
	return statementextract.ComparisonIdentityProof{Candidate: c, CatalogSHA256: strings.Repeat("a", 64), EndpointSHA256: strings.Repeat("b", 64), ZDRSHA256: strings.Repeat("c", 64), CatalogURL: "https://openrouter.ai/api/v1/models", EndpointURL: "https://openrouter.ai/api/v1/models/" + c.Canonical + "/endpoints", ZDRURL: "https://openrouter.ai/api/v1/endpoints/zdr", VerifiedAt: time.Now().UTC().Add(-time.Second)}
}
func (f *fake) Complete(_ context.Context, _ statementextract.Policy, _ statementextract.Chunk) (statementdata.Proposal, *statementextract.Billing, error) {
	f.calls++
	p := proof(f.c)
	b := &statementextract.Billing{GenerationID: "PRIVATE-GENERATION", InputTokens: 100, OutputTokens: 10, CostMicros: 10, ComparisonRoute: &statementextract.ComparisonRouteEvidence{Proof: &p, ObservedAt: time.Now().UTC(), SelectedModelClass: "alias"}}
	if f.mode == "merge" {
		return statementdata.Proposal{Header: statementdata.HeaderMapping{Labels: map[string]statementdata.Selection{"PRIVATE-ROLE": nil}}}, b, nil
	}
	return statementdata.Proposal{Rows: []statementdata.RowMapping{{Description: statementdata.Selection{"PRIVATE-UNKNOWN-ALIAS"}}}}, b, nil
}

type meter struct{}

func (*meter) Reserve(context.Context, int64, statementextract.Policy) (string, error) {
	return "PRIVATE-RESERVATION", nil
}
func (*meter) Settle(context.Context, string, *int64) error            { return nil }
func (*meter) ComparisonAllowed(context.Context, string) (bool, error) { return true, nil }
func testContext() publicContext {
	return publicContext{statementqualification.FixtureHash("columns"), statementdata.Document{Extractor: statementdata.ExtractorVersion, Pages: []statementdata.Page{{Number: 1, Width: 600, Height: 800}}, Spans: []statementdata.Span{{ID: "p1-b1-l1-w1", Page: 1, Block: 1, Line: 1, Word: 1, Text: "RUB", Box: statementdata.Box{XMin: 1, YMin: 1, XMax: 20, YMax: 10}}}}}
}
func campaign(t *testing.T, c publicContext, mode string) statementqualification.ComparisonReport {
	t.Helper()
	cs := statementextract.ComparisonCandidates()
	a, b := &fake{c: cs[0], mode: mode}, &fake{c: cs[1]}
	samples := []statementqualification.Sample{}
	for _, n := range statementqualification.ScreeningNames() {
		samples = append(samples, statementqualification.Sample{Name: n, Document: c.Document, SHA256: statementqualification.FixtureHash(n)})
	}
	r, e := statementqualification.RunComparison(context.Background(), samples, [2]statementextract.Reasoner{a, b}, &meter{})
	if e == nil || r.Error != "qualification_protocol_failed" || a.calls != 1 || b.calls != 0 || statementqualification.ValidateComparisonReport(r, time.Now()) != nil {
		t.Fatal("causal fake did not reach protocol stage", e)
	}
	if (r.Tuples[0].Proposal != nil) != (mode == "merge") {
		t.Fatal("proposal retention boundary")
	}
	return r
}
func encoded(r statementqualification.ComparisonReport) []byte {
	r.Digest = ""
	r.Digest = statementextract.Hash(r)
	raw, _ := json.Marshal(r)
	return raw
}
func TestCausalStagesAndPrivacy(t *testing.T) { checkStages(t, testContext()) }
func checkStages(t *testing.T, c publicContext) {
	t.Helper()
	r := campaign(t, c, "canonical")
	raw := encoded(r)
	out, e := project(raw, c, time.Now())
	if e != nil || out.Stage != "canonicalization" || out.Proposal != nil || out.ExactGuard != "UNVERIFIED" || out.ActualMicros != 10 {
		t.Fatal("canonical stage", e)
	}
	out.ArtifactsUnchanged = true
	re, e := replay(out, c)
	if e != nil || re.Replay != "unavailable_missing_raw_proposal" || re.ProviderCalls != 0 {
		t.Fatal("missing raw guessed", e)
	}
	safe, _ := json.Marshal(re)
	if bytes.Contains(safe, []byte("PRIVATE")) {
		t.Fatal("private data leaked")
	}
	// Real RunComparison merge refusal retains a proposal, but an unknown role is
	// deliberately not exported or guessed by this reader's accepted privacy contract.
	merge := campaign(t, c, "merge")
	if _, e := project(encoded(merge), c, time.Now()); e == nil {
		t.Fatal("unknown role must remain UNVERIFIED")
	}
	// A saved canonical proposal with a free-text service reason is safely copied;
	// a mismatch between claimed merge failure and current merge stops replay.
	r.Tuples[0].Proposal = &statementdata.Proposal{Service: []statementdata.ServiceMapping{{Spans: statementdata.Selection{c.Document.Spans[0].ID, c.Document.Spans[0].ID}, Reason: "PRIVATE-SERVICE"}}}
	out, e = project(encoded(r), c, time.Now())
	if e != nil || out.Stage != "merge" {
		t.Fatal("retained proposal", e)
	}
	safe, _ = json.Marshal(out)
	if bytes.Contains(safe, []byte("PRIVATE")) || len(out.Proposal.Service[0].Spans) != 2 {
		t.Fatal("privacy or duplicate loss")
	}
	out.ArtifactsUnchanged = true
	if _, e := replay(out, c); e == nil {
		t.Fatal("mismatched merge claimed as replay")
	}
}
func TestReportTamperingAndBounds(t *testing.T) {
	c := testContext()
	base := campaign(t, c, "canonical")
	for _, mode := range []string{"digest", "profile", "source-model", "time", "fixture", "document", "plan", "request", "route", "order", "state", "cost", "unknown", "proposal-role", "proposal-direction", "proposal-locator", "service-empty"} {
		t.Run(mode, func(t *testing.T) {
			var r statementqualification.ComparisonReport
			json.Unmarshal(encoded(base), &r)
			switch mode {
			case "profile":
				r.Profile.Scorer = "old"
			case "source-model":
				r.Candidates[0].Canonical = "other"
			case "time":
				r.FinishedAt = time.Now().Add(time.Hour)
			case "fixture":
				r.Tuples[0].Evidence.FixtureHash = strings.Repeat("a", 64)
			case "document":
				r.Tuples[0].Evidence.DocumentHash = strings.Repeat("a", 64)
				r.Tuples[1].Evidence.DocumentHash = r.Tuples[0].Evidence.DocumentHash
			case "plan":
				r.Tuples[0].Evidence.PlanHash = strings.Repeat("a", 64)
			case "request":
				r.Tuples[0].RequestHash = strings.Repeat("a", 64)
			case "route":
				r.Tuples[0].Call.Route.SelectedModelClass = "PRIVATE"
			case "order":
				r.Tuples[1].Candidate = "A"
			case "state":
				r.Tuples[1].State = "attempted"
			case "cost":
				*r.ActualMicros = 0
			case "unknown":
				r.ActualMicros = nil
			case "proposal-role":
				r.Tuples[0].Proposal = &statementdata.Proposal{Header: statementdata.HeaderMapping{Labels: map[string]statementdata.Selection{"PRIVATE": nil}}}
			case "proposal-direction":
				r.Tuples[0].Proposal = &statementdata.Proposal{Rows: []statementdata.RowMapping{{Direction: "PRIVATE"}}}
			case "proposal-locator":
				r.Tuples[0].Proposal = &statementdata.Proposal{Rows: []statementdata.RowMapping{{Description: statementdata.Selection{"PRIVATE"}}}}
			case "service-empty":
				r.Tuples[0].Proposal = &statementdata.Proposal{Service: []statementdata.ServiceMapping{{Reason: ""}}}
			}
			raw := encoded(r)
			if mode == "digest" {
				raw = bytes.Replace(raw, []byte(`"digest":"`), []byte(`"digest":"f`), 1)
			}
			if _, e := project(raw, c, time.Now()); e == nil {
				t.Fatal("tampering accepted")
			}
		})
	}
	for _, raw := range []string{`{"passed":true,"Passed":false}`, `{} {}`, strings.Repeat("[", 22) + "0" + strings.Repeat("]", 22), `[` + strings.Repeat("0,", 5000) + `0]`, strings.Repeat(" ", limit+1), `{"extra":"PRIVATE"}`} {
		var r statementqualification.ComparisonReport
		if strict([]byte(raw), &r) == nil {
			t.Fatal("unbounded/ambiguous JSON accepted")
		}
	}
}
func TestPrivateReaderAndLock(t *testing.T) {
	c := testContext()
	raw := encoded(campaign(t, c, "canonical"))
	dir := t.TempDir()
	m := filepath.Join(dir, basename+".started")
	r := filepath.Join(dir, basename+".json")
	l := filepath.Join(dir, "statement-screening.lock")
	write := func() {
		os.WriteFile(m, []byte(source+"\n"), 0600)
		os.Chmod(m, 0600)
		os.WriteFile(r, raw, 0600)
		os.Chmod(r, 0600)
		os.WriteFile(l, nil, 0600)
	}
	write()
	beforeM, _ := readPrivate(m, 128)
	beforeR, _ := readPrivate(r, limit)
	out, e := readReport(dir, c)
	if e != nil || !out.ArtifactsUnchanged {
		t.Fatal("read-only reader", e)
	}
	afterM, _ := readPrivate(m, 128)
	afterR, _ := readPrivate(r, limit)
	if !unchanged(beforeM, afterM) || !unchanged(beforeR, afterR) {
		t.Fatal("reader wrote artifacts")
	}
	for _, mode := range []string{"source", "permissions", "symlink", "oversize", "missing", "lock", "fifo"} {
		t.Run(mode, func(t *testing.T) {
			defer func() { os.Remove(m); os.Remove(r); os.Remove(l); write() }()
			var lock *os.File
			switch mode {
			case "source":
				os.WriteFile(m, []byte("wrong\n"), 0600)
			case "permissions":
				os.Chmod(r, 0644)
			case "symlink":
				os.Remove(r)
				os.Symlink(m, r)
			case "oversize":
				os.WriteFile(r, bytes.Repeat([]byte(" "), limit+1), 0600)
			case "missing":
				os.Remove(r)
			case "lock":
				lock, _ = os.Open(l)
				defer lock.Close()
				syscall.Flock(int(lock.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
			case "fifo":
				os.Remove(r)
				syscall.Mkfifo(r, 0600)
			}
			if _, e := readReport(dir, c); e == nil {
				t.Fatal("unsafe artifact accepted")
			}
		})
	}
}
