package statementqualification

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/QahNaaRlN/grace/internal/statementextract"
)

type rowPilotFake struct {
	t                       *testing.T
	mode                    string
	calls, proofs, metadata int
	requests                []statementextract.RowScopedRequest
	cancel                  context.CancelFunc
}

func (f *rowPilotFake) CurrentPolicy(context.Context) (statementextract.Policy, error) {
	f.metadata = 3
	if f.mode == "proof" {
		return statementextract.Policy{}, errors.New("PRIVATE")
	}
	return statementextract.ComparisonCandidates()[0].Policy, nil
}
func (f *rowPilotFake) ComparisonProof(context.Context) (statementextract.ComparisonIdentityProof, error) {
	f.proofs++
	if f.mode == "refresh" && f.proofs > 1 {
		return statementextract.ComparisonIdentityProof{}, errors.New("PRIVATE")
	}
	return *comparisonTestRoute(statementextract.ComparisonCandidates()[0]).Proof, nil
}
func (f *rowPilotFake) MetadataRequests() int { return f.metadata }
func (f *rowPilotFake) CompleteRowPilot(ctx context.Context, q statementextract.RowScopedRequest, proof statementextract.ComparisonIdentityProof) ([]byte, *statementextract.Billing, error) {
	f.calls++
	f.requests = append(f.requests, q)
	if deadline, ok := ctx.Deadline(); !ok || time.Until(deadline) > 120*time.Second {
		f.t.Fatal("call deadline")
	}
	c, numeric := rowScopedDelivered(f.t, q)
	var raw []byte
	switch q.Stage {
	case "base":
		p := rowScopedDeliveredProposal(f.t, c, numeric)
		if f.mode != "tie" {
			p = clockDrop(p)
		}
		if f.mode == "base_empty" {
			p.Rows = p.Rows[:0]
		}
		if f.mode == "base_missing" {
			p.Rows = p.Rows[:1]
		}
		raw = rowScopedPayload(f.t, p)
	case "grouping":
		fault := ""
		if f.mode == "prefix" || f.mode == "suffix" {
			fault = f.mode
		}
		if f.mode == "semantic" {
			fault = "duplicate_owner"
		}
		raw = rowScopedPayload(f.t, rowScopedGroupingMapping(f.t, c, numeric, fault))
	case "row":
		r := rowScopedFieldMapping(c, numeric)
		if f.mode == "missing_clock" && len(r.OperationDate) > 1 {
			r.OperationDate = r.OperationDate[:1]
		}
		if f.mode == "quality" {
			r.AccountCurrency = nil
		}
		raw = rowScopedPayload(f.t, r)
	}
	now := time.Now().UTC()
	bill := &statementextract.Billing{GenerationID: fmt.Sprintf("PRIVATE-row-%d", f.calls), InputTokens: 100, OutputTokens: 100, CostMicros: 10, ComparisonRoute: &statementextract.ComparisonRouteEvidence{Proof: &proof, ObservedAt: now, SelectedModelClass: "alias"}}
	if f.calls == 3 {
		switch f.mode {
		case "unknown":
			return raw, nil, errors.New("PRIVATE")
		case "cost":
			bill.CostMicros = 53901
		case "duplicate":
			bill.GenerationID = "PRIVATE-row-1"
		case "route":
			bill.ComparisonRoute.SelectedModelClass = "other"
		case "capture":
			raw = []byte(strings.Repeat("X", statementextract.RowScopedResponseLimit+1))
		case "protocol":
			raw = []byte(`{"extra":[]}`)
		case "panic":
			panic("PRIVATE client")
		case "cancel":
			f.cancel()
		}
	}
	return raw, bill, nil
}

type rowPilotTestMeter struct {
	reserves, settles, allowed int
	maximum                    int64
	actual                     *int64
	mode                       string
}

func (m *rowPilotTestMeter) Reserve(ctx context.Context, n int64, _ statementextract.Policy) (string, error) {
	m.reserves++
	m.maximum = n
	if d, ok := ctx.Deadline(); !ok || time.Until(d) > 1800*time.Second {
		return "", errors.New("deadline")
	}
	if m.mode == "reserve" {
		return "", errors.New("PRIVATE")
	}
	return "fake-row-reservation", nil
}
func (m *rowPilotTestMeter) ComparisonAllowed(context.Context, string) (bool, error) {
	m.allowed++
	return m.mode != "block", nil
}
func (m *rowPilotTestMeter) Settle(ctx context.Context, _ string, n *int64) error {
	m.settles++
	if n != nil {
		v := *n
		m.actual = &v
	}
	if ctx.Err() != nil {
		return errors.New("cancelled settle")
	}
	if d, ok := ctx.Deadline(); !ok || time.Until(d) > 5*time.Second {
		return errors.New("settle deadline")
	}
	if m.mode == "settle" {
		return errors.New("PRIVATE")
	}
	return nil
}
func TestRowPilotCampaignAndReplay(t *testing.T) {
	d := clockTestDocument(t)
	for _, mode := range []string{"keep", "base_missing", "base_empty", "tie", "missing_clock", "prefix", "suffix", "quality", "semantic", "proof", "refresh", "reserve", "block", "unknown", "cost", "duplicate", "route", "capture", "protocol", "panic", "cancel", "settle", "cancel_before"} {
		t.Run(mode, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			if mode == "cancel_before" {
				cancel()
			}
			f := &rowPilotFake{t: t, mode: mode, cancel: cancel}
			m := &rowPilotTestMeter{mode: mode}
			report, e := RunRowPilot(ctx, Sample{"cards", d, FixtureHash("cards")}, f, m, strings.Repeat("a", 40))
			if report.Version != RowPilotVersion || len(report.Calls) > 12 || m.reserves > 1 || m.settles > 1 || strings.Contains(report.Error, "PRIVATE") {
				t.Fatal(report, e, m)
			}
			want := ClockInconclusive
			if mode == "keep" || mode == "base_missing" {
				want = RowPilotKeep
			}
			if mode == "quality" || mode == "semantic" || mode == "missing_clock" || mode == "prefix" || mode == "suffix" {
				want = ClockReject
			}
			if report.Decision != want {
				t.Fatalf("decision=%s want=%s err=%v attempts=%+v", report.Decision, want, e, report.Attempts)
			}
			if mode == "keep" || mode == "tie" {
				if e != nil || f.calls != 12 || m.reserves != 1 || m.settles != 1 || m.maximum != 646800 || m.actual == nil || *m.actual != 120 {
					t.Fatal("campaign/accounting", f.calls, m, e)
				}
			}
			if mode == "quality" || mode == "semantic" || mode == "missing_clock" || mode == "prefix" || mode == "suffix" {
				if e != nil || f.calls > 4 || report.Attempts[2].State != "not_sent" || report.Attempts[2].StopCode != "row_pilot_decisive_reject" {
					t.Fatal("early reject", e, f.calls, report.Attempts)
				}
			}
			if mode == "unknown" || mode == "cost" || mode == "duplicate" || mode == "panic" {
				if m.actual != nil || report.ActualMicros != nil || f.calls != 3 {
					t.Fatal("uncertain delivery", m, report.ActualMicros, f.calls)
				}
			}
			if e2 := ValidateRowPilotReport(report, d); e2 != nil {
				t.Fatalf("invalid produced report mode=%s err=%v run=%v calls=%+v attempts=%+v", mode, e2, e, report.Calls, report.Attempts)
			}
			var encoded bytes.Buffer
			if EncodeRowPilotReport(&encoded, report, d) != nil {
				t.Fatal("encode")
			}
			decoded, e2 := DecodeRowPilotReport(&encoded, d)
			if e2 != nil || decoded.Digest != report.Digest {
				t.Fatal("decode", e2)
			}
			summary, e2 := SummarizeRowPilot(report, d)
			if e2 != nil || !summary.ReplayVerified || summary.ActivationAuthorized || summary.Decision != want {
				t.Fatal("summary", e2, summary)
			}
			if e := ValidateRowPilotSummary(summary, report.Source); e != nil {
				t.Fatalf("public mode=%s: %v summary=%+v", mode, e, summary)
			}
			out, _ := json.Marshal(summary)
			if _, e := DecodeRowPilotSummary(bytes.NewReader(out), report.Source); e != nil {
				t.Fatal("public decode", e)
			}
			if strings.Contains(string(out), "PRIVATE") || strings.Contains(string(out), "GenerationID") || strings.Contains(string(out), "RawBytes") {
				t.Fatal("private output", string(out))
			}
			t.Logf("S121_PILOT_FAKE mode=%s decision=%s attempted=%d settled=%t", mode, report.Decision, f.calls, report.SettlementRecorded)
		})
	}
}
func TestRowPilotReplayRejectsMutatedMetrics(t *testing.T) {
	d := clockTestDocument(t)
	f := &rowPilotFake{t: t, mode: "keep"}
	r, e := RunRowPilot(context.Background(), Sample{"cards", d, FixtureHash("cards")}, f, &rowPilotTestMeter{}, strings.Repeat("a", 40))
	if e != nil {
		t.Fatal(e)
	}
	for _, mode := range []string{"check", "precision", "score", "request", "raw", "count", "cost", "source", "decision", "slot", "private_code", "duplicate_key", "unknown_key", "case_key", "trailing"} {
		t.Run(mode, func(t *testing.T) {
			raw, _ := json.Marshal(r)
			var x RowPilotReport
			_ = json.Unmarshal(raw, &x)
			switch mode {
			case "check":
				x.Attempts[1].Measurement.Checks[0].Passed = false
			case "precision":
				x.Attempts[1].Measurement.Precision[0].OperationMatches = false
			case "score":
				x.Attempts[1].Measurement.Result.Correct--
			case "request":
				x.Calls[1].Request.Hash = strings.Repeat("0", 64)
			case "raw":
				x.Calls[1].Raw = []byte(`{}`)
				x.Calls[1].RawBytes = 2
				x.Calls[1].RawHash = statementextract.PromptSHA("{}")
			case "count":
				x.Calls = x.Calls[:len(x.Calls)-1]
			case "cost":
				*x.ActualMicros = 0
			case "source":
				x.Source = "invalid"
			case "decision":
				x.Decision = ClockReject
			case "slot":
				x.Attempts[2].State = "not_sent"
			case "private_code":
				x.Error = "PRIVATE"
			}
			x.Digest = rowPilotDigest(x)
			raw, _ = json.Marshal(x)
			switch mode {
			case "duplicate_key":
				raw = append([]byte(`{"Version":"bad",`), raw[1:]...)
			case "unknown_key":
				raw = append([]byte(`{"extra":null,`), raw[1:]...)
			case "case_key":
				raw = bytes.Replace(raw, []byte(`"Source"`), []byte(`"source"`), 1)
			case "trailing":
				raw = append(raw, []byte(`{}`)...)
			}
			if _, e := DecodeRowPilotReport(bytes.NewReader(raw), d); e == nil {
				t.Fatal("mutation accepted", mode)
			}
		})
	}
}

func TestRowPilotPublicSummaryMutations(t *testing.T) {
	d := clockTestDocument(t)
	source := strings.Repeat("a", 40)
	r, e := RunRowPilot(context.Background(), Sample{"cards", d, FixtureHash("cards")}, &rowPilotFake{t: t, mode: "keep"}, &rowPilotTestMeter{}, source)
	if e != nil {
		t.Fatal(e)
	}
	s, e := SummarizeRowPilot(r, d)
	if e != nil {
		t.Fatal(e)
	}
	for _, mode := range []string{"plan", "prompt", "code", "source", "precision", "cost", "decision", "private", "duplicate", "case", "unknown"} {
		t.Run(mode, func(t *testing.T) {
			raw, _ := json.Marshal(s)
			var x RowPilotSummary
			_ = json.Unmarshal(raw, &x)
			switch mode {
			case "plan":
				x.PlanHash = strings.Repeat("0", 64)
			case "prompt":
				x.Requests[0].PromptSHA = strings.Repeat("0", 64)
			case "code":
				x.Attempts[1].Measurement.Checks[0].Code = "PRIVATE"
			case "source":
				x.Source = strings.Repeat("b", 40)
			case "precision":
				x.Attempts[1].Measurement.Precision[0].OperationMatches = false
			case "cost":
				*x.ActualMicros = 646801
			case "decision":
				x.Decision = ClockReject
			case "private":
				x.Attempts[0].Measurement.Result.Validation = []string{"PRIVATE"}
			}
			raw, _ = json.Marshal(x)
			if mode == "duplicate" {
				raw = append([]byte(`{"source":"bad",`), raw[1:]...)
			}
			if mode == "case" {
				raw = bytes.Replace(raw, []byte(`"Source"`), []byte(`"source"`), 1)
			}
			if mode == "unknown" {
				raw = append([]byte(`{"private":"data",`), raw[1:]...)
			}
			if _, e := DecodeRowPilotSummary(bytes.NewReader(raw), source); e == nil {
				t.Fatal("public mutation accepted", mode)
			}
		})
	}
	s.Attempts[1].Measurement.Checks[0].Passed = false
	if ValidateRowPilotReport(r, d) != nil {
		t.Fatal("summary aliases private report")
	}
	for _, code := range []string{"row_pilot_report_unavailable", "row_pilot_operator_owner_unavailable", "PRIVATE"} {
		f := NewRowPilotRefusal(source, code)
		raw, _ := json.Marshal(f)
		if _, e := DecodeRowPilotPublicRefusal(raw, source); e != nil {
			t.Fatal(e)
		}
		if bytes.Contains(raw, []byte("PRIVATE")) || f.ActualMicros != nil || f.RecordedCalls != nil || f.SettlementRecorded != nil {
			t.Fatal("unknown projected as known")
		}
		f.RecordedCalls = new(int)
		raw, _ = json.Marshal(f)
		if _, e := DecodeRowPilotPublicRefusal(raw, source); e == nil {
			t.Fatal("invented zero calls accepted")
		}
	}
}
