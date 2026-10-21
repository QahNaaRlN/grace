// S1.12 reads one historical synthetic report. No provider, meter or SQL calls.
package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"syscall"
	"time"

	"github.com/QahNaaRlN/grace/internal/pdfextract"
	"github.com/QahNaaRlN/grace/internal/statementdata"
	"github.com/QahNaaRlN/grace/internal/statementextract"
	"github.com/QahNaaRlN/grace/internal/statementqualification"
)

const source = "012f604033de9c6a75bf531b8f715323d0fda57d"
const root = "/opt/grace/shared"
const basename = "statement-comparison-s111-comparison-20261021"
const limit = 524288

var unavailable = errors.New("s112_unverified")

type publicContext struct {
	FixtureHash string                 `json:"fixtureHash"`
	Document    statementdata.Document `json:"document"`
}
type tuple struct {
	Candidate       string `json:"candidate"`
	Name            string `json:"name"`
	State           string `json:"state"`
	Reason          string `json:"reason"`
	ProposalPresent bool   `json:"proposalPresent"`
	CostMicros      *int64 `json:"costMicros,omitempty"`
	LatencyMS       *int64 `json:"latencyMs,omitempty"`
}
type projection struct {
	Version            string                              `json:"version"`
	Verified           bool                                `json:"verified"`
	DigestVerified     bool                                `json:"digestVerified"`
	ArtifactsUnchanged bool                                `json:"artifactsUnchanged"`
	Stage              string                              `json:"stage"`
	ExactGuard         string                              `json:"exactGuard"`
	RecordedCalls      int                                 `json:"recordedCalls"`
	CompletedTuples    int                                 `json:"completedTuples"`
	BillingKnown       bool                                `json:"billingKnown"`
	ActualMicros       int64                               `json:"actualMicros"`
	Time               time.Time                           `json:"time"`
	FinishedAt         time.Time                           `json:"finishedAt"`
	Evidence           statementqualification.CaseEvidence `json:"evidence"`
	RequestHash        string                              `json:"requestHash"`
	Tuples             []tuple                             `json:"tuples"`
	Proposal           *statementdata.Proposal             `json:"proposal,omitempty"`
}
type replayResult struct {
	Verified      bool       `json:"verified"`
	ProviderCalls int        `json:"providerCalls"`
	Stage         string     `json:"stage"`
	ExactGuard    string     `json:"exactGuard"`
	Replay        string     `json:"replay"`
	Projection    projection `json:"projection"`
}

// Validate bounds and duplicates before typed decoding, including Go's case-insensitive keys.
func strict(raw []byte, out any) error {
	if len(raw) > limit {
		return unavailable
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	var walk func(int) error
	walk = func(depth int) error {
		if depth > 20 {
			return unavailable
		}
		t, e := dec.Token()
		if e != nil {
			return unavailable
		}
		if s, ok := t.(string); ok && len(s) > 10000 {
			return unavailable
		}
		if d, ok := t.(json.Delim); ok {
			n := 0
			seen := map[string]bool{}
			for dec.More() {
				n++
				if n > 5000 {
					return unavailable
				}
				if d == '{' {
					k, e := dec.Token()
					s, ok := k.(string)
					if e != nil || !ok || len(s) > 128 || seen[strings.ToLower(s)] {
						return unavailable
					}
					seen[strings.ToLower(s)] = true
				}
				if e := walk(depth + 1); e != nil {
					return unavailable
				}
			}
			end, e := dec.Token()
			if e != nil || !(d == '{' && end == json.Delim('}') || d == '[' && end == json.Delim(']')) {
				return unavailable
			}
		}
		return nil
	}
	if walk(0) != nil {
		return unavailable
	}
	if _, e := dec.Token(); e != io.EOF {
		return unavailable
	}
	dec = json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if dec.Decode(out) != nil {
		return unavailable
	}
	return nil
}
func input(r io.Reader, out any) error {
	raw, e := io.ReadAll(io.LimitReader(r, limit+1))
	if e != nil {
		return unavailable
	}
	return strict(raw, out)
}
func planFor(c publicContext) (statementextract.Plan, error) {
	if c.FixtureHash != statementqualification.FixtureHash("columns") || c.Document.Extractor != statementdata.ExtractorVersion || c.Document.Validate() != nil {
		return statementextract.Plan{}, unavailable
	}
	return statementextract.BuildComparisonPlan(c.Document, statementextract.ComparisonCandidates()[0].Policy)
}
func requestHash(p statementextract.Plan) string {
	return statementextract.Hash(struct {
		Policy   statementextract.Policy
		Messages []map[string]string
		Schema   any
	}{p.Policy, statementextract.Messages(p.Chunks[0]), statementextract.RequestSchema(p.Chunks[0])})
}
func evidenceMatches(c publicContext, p statementextract.Plan, e statementqualification.CaseEvidence, h string) bool {
	return e.Name == "columns" && e.FixtureHash == c.FixtureHash && e.DocumentHash == c.Document.Hash() && e.PlanHash == statementextract.Hash(p) && e.InputBytes == statementextract.InputBytes(p.Chunks[0]) && h == requestHash(p)
}

// Only public source locators, closed directions/roles and a fixed replacement reason survive.
func sanitize(in statementdata.Proposal, d statementdata.Document) (statementdata.Proposal, error) {
	raw, _ := json.Marshal(in)
	var out statementdata.Proposal
	if strict(raw, &out) != nil {
		return out, unavailable
	}
	valid := map[string]bool{}
	for _, s := range d.Spans {
		valid[s.ID] = true
	}
	check := func(ids statementdata.Selection) bool {
		if len(ids) > 2000 {
			return false
		}
		for _, id := range ids {
			if !valid[id] {
				return false
			}
		}
		return true
	}
	if len(out.Rows) > 5000 || len(out.Service) > 2000 {
		return out, unavailable
	}
	for _, r := range out.Rows {
		if r.Direction != "" && r.Direction != "debit" && r.Direction != "credit" {
			return out, unavailable
		}
		for _, ids := range []statementdata.Selection{r.OperationDate, r.BookingDate, r.AccountAmount, r.AccountCurrency, r.TransactionAmount, r.TransactionCurrency, r.Description, r.DirectionEvidence} {
			if !check(ids) {
				return out, unavailable
			}
		}
	}
	h := out.Header
	for _, ids := range []statementdata.Selection{h.OpeningBookDate, h.ClosingBookDate, h.OpeningBookBalance, h.ClosingBookBalance, h.AvailableBalanceObserved, h.DebitTurnover, h.CreditTurnover, h.Currency} {
		if !check(ids) {
			return out, unavailable
		}
	}
	for role, ids := range h.Labels {
		switch role {
		case "openingBookBalance", "closingBookBalance", "availableBalanceObserved", "debitTurnover", "creditTurnover":
		default:
			return out, unavailable
		}
		if !check(ids) {
			return out, unavailable
		}
	}
	for i, g := range out.Service {
		if !check(g.Spans) || len(g.Reason) > 2000 || strings.TrimSpace(g.Reason) == "" {
			return out, unavailable
		}
		out.Service[i].Reason = "Reviewed fixed synthetic service grouping"
	}
	return out, nil
}
func project(raw []byte, c publicContext, now time.Time) (projection, error) {
	var r statementqualification.ComparisonReport
	if strict(raw, &r) != nil || statementqualification.ValidateComparisonReport(r, now) != nil {
		return projection{}, unavailable
	}
	// This diagnostic is intentionally fixed to the observed interrupted campaign.
	if r.Error != "qualification_protocol_failed" || r.Passed || r.ActualMicros == nil || r.Tuples[0].State != "attempted" || r.Tuples[0].Reason != "protocol_failed" || r.Tuples[0].Diagnostic != nil {
		return projection{}, unavailable
	}
	p, e := planFor(c)
	if e != nil || !evidenceMatches(c, p, r.Tuples[0].Evidence, r.Tuples[0].RequestHash) {
		return projection{}, unavailable
	}
	// Candidate B has the same public columns, but its own policy/plan/request hashes.
	b, e := statementextract.BuildComparisonPlan(c.Document, statementextract.ComparisonCandidates()[1].Policy)
	if e != nil || !evidenceMatches(c, b, r.Tuples[1].Evidence, r.Tuples[1].RequestHash) {
		return projection{}, unavailable
	}
	out := projection{Version: "statement.s112.projection.v1", Verified: true, DigestVerified: true, Stage: "canonicalization", ExactGuard: "UNVERIFIED", RecordedCalls: 1, BillingKnown: true, ActualMicros: *r.ActualMicros, Time: r.Time, FinishedAt: r.FinishedAt, Evidence: r.Tuples[0].Evidence, RequestHash: r.Tuples[0].RequestHash, Tuples: []tuple{}}
	for i, t := range r.Tuples {
		row := tuple{Candidate: t.Candidate, Name: t.Evidence.Name, State: t.State, Reason: t.Reason, ProposalPresent: t.Proposal != nil}
		if i == 0 {
			if t.Call == nil || t.Call.OutputTokens == 0 || !t.Call.Route.Valid("A", r.Time, r.FinishedAt) {
				return projection{}, unavailable
			}
			row.CostMicros = &t.Call.CostMicros
			row.LatencyMS = &t.Call.LatencyMS
		} else if t.State != "not_sent" {
			return projection{}, unavailable
		}
		out.Tuples = append(out.Tuples, row)
	}
	if r.Tuples[0].Proposal != nil {
		prop, e := sanitize(*r.Tuples[0].Proposal, c.Document)
		if e != nil {
			return projection{}, unavailable
		}
		out.Stage = "merge"
		out.Proposal = &prop
	}
	return out, nil
}

type snapshot struct {
	Data []byte
	Info os.FileInfo
	Hash [32]byte
}

func readPrivate(path string, max int) (snapshot, error) {
	fd, e := syscall.Open(path, syscall.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0)
	if e != nil {
		return snapshot{}, unavailable
	}
	f := os.NewFile(uintptr(fd), path)
	defer f.Close()
	st, e := f.Stat()
	if e != nil || !st.Mode().IsRegular() || st.Mode().Perm() != 0600 || st.Size() > int64(max) {
		return snapshot{}, unavailable
	}
	raw, e := io.ReadAll(io.LimitReader(f, int64(max+1)))
	if e != nil || len(raw) > max {
		return snapshot{}, unavailable
	}
	return snapshot{raw, st, sha256.Sum256(raw)}, nil
}
func unchanged(a, b snapshot) bool {
	return os.SameFile(a.Info, b.Info) && a.Info.Size() == b.Info.Size() && a.Info.Mode() == b.Info.Mode() && a.Info.ModTime() == b.Info.ModTime() && a.Hash == b.Hash
}
func readReport(dir string, c publicContext) (projection, error) {
	fd, e := syscall.Open(filepath.Join(dir, "statement-screening.lock"), syscall.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0)
	if e != nil {
		return projection{}, unavailable
	}
	lock := os.NewFile(uintptr(fd), "lock")
	defer lock.Close()
	st, e := lock.Stat()
	if e != nil || !st.Mode().IsRegular() {
		return projection{}, unavailable
	}
	if syscall.Flock(fd, syscall.LOCK_EX|syscall.LOCK_NB) != nil {
		return projection{}, unavailable
	}
	defer syscall.Flock(fd, syscall.LOCK_UN)
	markerPath := filepath.Join(dir, basename+".started")
	reportPath := filepath.Join(dir, basename+".json")
	marker, e := readPrivate(markerPath, 128)
	if e != nil || string(marker.Data) != source+"\n" {
		return projection{}, unavailable
	}
	report, e := readPrivate(reportPath, limit)
	if e != nil {
		return projection{}, unavailable
	}
	out, e := project(report.Data, c, time.Now().UTC())
	if e != nil {
		return projection{}, unavailable
	}
	ma, e := readPrivate(markerPath, 128)
	if e != nil || !unchanged(marker, ma) {
		return projection{}, unavailable
	}
	ra, e := readPrivate(reportPath, limit)
	if e != nil || !unchanged(report, ra) {
		return projection{}, unavailable
	}
	out.ArtifactsUnchanged = true
	return out, nil
}
func replay(out projection, c publicContext) (replayResult, error) {
	p, e := planFor(c)
	if e != nil || !out.Verified || !out.DigestVerified || !out.ArtifactsUnchanged || out.Version != "statement.s112.projection.v1" || out.RecordedCalls != 1 || out.CompletedTuples != 0 || !out.BillingKnown || out.ActualMicros < 0 || out.ActualMicros > 53900 || !evidenceMatches(c, p, out.Evidence, out.RequestHash) || len(out.Tuples) != 6 || out.ExactGuard != "UNVERIFIED" {
		return replayResult{}, unavailable
	}
	for i, t := range out.Tuples {
		cs := statementextract.ComparisonCandidates()
		if t.Candidate != cs[i%2].ID || t.Name != statementqualification.ScreeningNames()[i/2] {
			return replayResult{}, unavailable
		}
		if i == 0 {
			if t.State != "attempted" || t.Reason != "protocol_failed" || t.CostMicros == nil || *t.CostMicros != out.ActualMicros || t.LatencyMS == nil || *t.LatencyMS < 0 || t.ProposalPresent != (out.Proposal != nil) {
				return replayResult{}, unavailable
			}
		} else if t.State != "not_sent" || t.Reason != "campaign_not_started" || t.CostMicros != nil || t.LatencyMS != nil || t.ProposalPresent {
			return replayResult{}, unavailable
		}
	}
	r := replayResult{Verified: true, Stage: out.Stage, ExactGuard: "UNVERIFIED", Replay: "unavailable_missing_raw_proposal", Projection: out}
	if out.Proposal == nil {
		if out.Stage != "canonicalization" {
			return replayResult{}, unavailable
		}
		return r, nil
	}
	if out.Stage != "merge" {
		return replayResult{}, unavailable
	}
	prop, e := sanitize(*out.Proposal, c.Document)
	if e != nil || !reflect.DeepEqual(prop, *out.Proposal) {
		return replayResult{}, unavailable
	}
	_, e = statementextract.MergeProposals(p, []statementdata.Proposal{prop})
	if e == nil {
		return replayResult{}, unavailable
	}
	switch e.Error() {
	case "statement_incomplete", "conflicting_header_proposals", "proposal_outside_chunk", "invalid_header_role":
		r.ExactGuard = e.Error()
	default:
		return replayResult{}, unavailable
	}
	r.Replay = "matched_saved_merge_failure"
	return r, nil
}
func run() error {
	if len(os.Args) == 4 && os.Args[1] == "prepare" {
		pdf, e := os.ReadFile(os.Args[2])
		if e != nil {
			return unavailable
		}
		h := sha256.Sum256(pdf)
		hash := hex.EncodeToString(h[:])
		if hash != statementqualification.FixtureHash("columns") {
			return unavailable
		}
		helper, e := pdfextract.NewClient(os.Args[3])
		if e != nil {
			return unavailable
		}
		ctx, cancel := context.WithTimeout(context.Background(), 40*time.Second)
		defer cancel()
		d, e := helper.Extract(ctx, pdf)
		if e != nil {
			return unavailable
		}
		c := publicContext{hash, d}
		if _, e = planFor(c); e != nil {
			return unavailable
		}
		return json.NewEncoder(os.Stdout).Encode(c)
	}
	if len(os.Args) == 3 && os.Args[1] == "replay" {
		raw, e := os.ReadFile(os.Args[2])
		if e != nil {
			return unavailable
		}
		var c publicContext
		if strict(raw, &c) != nil {
			return unavailable
		}
		var p projection
		if input(os.Stdin, &p) != nil {
			return unavailable
		}
		out, e := replay(p, c)
		if e != nil {
			return unavailable
		}
		return json.NewEncoder(os.Stdout).Encode(out)
	}
	if len(os.Args) != 1 {
		return unavailable
	}
	var c publicContext
	if input(os.Stdin, &c) != nil {
		return unavailable
	}
	out, e := readReport(root, c)
	if e != nil {
		return unavailable
	}
	return json.NewEncoder(os.Stdout).Encode(out)
}
func main() {
	if run() != nil {
		fmt.Println(`{"error":"s112_diagnostic_unavailable","verified":false}`)
		os.Exit(1)
	}
}
