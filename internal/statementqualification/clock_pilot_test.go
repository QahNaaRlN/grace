package statementqualification

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/QahNaaRlN/grace/internal/statementdata"
	"github.com/QahNaaRlN/grace/internal/statementextract"
	"os"
	"strings"
	"testing"
	"time"
)

func clockTestDocument(t *testing.T) statementdata.Document {
	t.Helper()
	raw, e := os.ReadFile("../statementextract/testdata/s119-cards-document.json")
	if e != nil {
		t.Fatal(e)
	}
	var d statementdata.Document
	if json.Unmarshal(raw, &d) != nil {
		t.Fatal("fixture")
	}
	return d
}
func clockTestProposal(t *testing.T, d statementdata.Document) statementdata.Proposal {
	t.Helper()
	q, e := statementextract.BuildClockRequest(d, "base")
	if e != nil {
		t.Fatal(e)
	}
	f := &activationFixtureReasoner{}
	out, _, e := f.Complete(context.Background(), q.Plan.Policy, q.Plan.Chunks[0])
	if e != nil {
		t.Fatal(e)
	}
	return out
}
func clockDrop(p statementdata.Proposal) statementdata.Proposal {
	raw, _ := json.Marshal(p)
	var out statementdata.Proposal
	_ = json.Unmarshal(raw, &out)
	for i := range out.Rows {
		out.Rows[i].OperationDate = out.Rows[i].OperationDate[:1]
		out.Rows[i].BookingDate = out.Rows[i].BookingDate[:1]
	}
	return out
}
func TestClockCausalAndJudge(t *testing.T) {
	d := clockTestDocument(t)
	p := clockTestProposal(t, d)
	good, e := MeasureClock(d, p)
	if e != nil || good.Count == nil || *good.Count != 4 || !good.Result.Passed {
		t.Fatal("good", good, e)
	}
	bad, e := MeasureClock(d, clockDrop(p))
	if e != nil || bad.Count == nil || *bad.Count != 0 || bad.Result.Passed || len(bad.Result.Coverage) == 0 {
		t.Fatal("bad", bad, e)
	}
	seq := []ClockMeasurement{bad, good, good, bad, bad, good}
	if DecideClock(seq, true) != ClockKeep || DecideClock(seq, false) != ClockInconclusive {
		t.Fatal("keep/incomplete")
	}
	tie := []ClockMeasurement{good, good, good, good, good, good}
	if DecideClock(tie, true) != ClockInconclusive {
		t.Fatal("tie")
	}
	seq[1] = bad
	if DecideClock(seq, true) != ClockReject {
		t.Fatal("reject")
	}
	wrong := clockTestProposal(t, d)
	wrong.Rows[0].TransactionCurrency = wrong.Rows[0].AccountCurrency
	regression, e := MeasureClock(d, wrong)
	if e != nil || *regression.Count != 4 || regression.Result.Passed {
		t.Fatal("non-clock regression", e)
	}
	seq = []ClockMeasurement{bad, regression, good, bad, bad, good}
	if DecideClock(seq, true) != ClockReject {
		t.Fatal("regression hidden")
	}
	one := clockTestProposal(t, d)
	one.Rows = one.Rows[:1]
	na, e := MeasureClock(d, one)
	if e != nil || na.Comparable || na.Count != nil {
		t.Fatal("N/A", e)
	}
	seq[0] = na
	if DecideClock(seq, true) != ClockInconclusive {
		t.Fatal("N/A guessed")
	}
	foreign := clockTestProposal(t, d)
	foreign.Rows[0].OperationDate[1] = foreign.Rows[1].BookingDate[1]
	m, e := MeasureClock(d, foreign)
	if e == nil && m.Count != nil && *m.Count == 4 {
		t.Fatal("neighbor clock")
	}
	invented := clockTestProposal(t, d)
	invented.Rows[0].OperationDate = append(invented.Rows[0].OperationDate, "invented-10:30")
	if _, e := MeasureClock(d, invented); e == nil {
		t.Fatal("invented time")
	}
}

type clockFake struct {
	t          *testing.T
	calls      int
	fail, mode string
	orders     []string
}

func (f *clockFake) CurrentPolicy(context.Context) (statementextract.Policy, error) {
	if f.fail == "proof" {
		return statementextract.Policy{}, errors.New("PRIVATE")
	}
	return statementextract.ComparisonCandidates()[0].Policy, nil
}
func (f *clockFake) ComparisonProof(context.Context) (statementextract.ComparisonIdentityProof, error) {
	return *comparisonTestRoute(statementextract.ComparisonCandidates()[0]).Proof, nil
}
func (f *clockFake) CompleteClock(ctx context.Context, d statementdata.Document, v string) (statementdata.Proposal, *statementextract.Billing, error) {
	f.calls++
	f.orders = append(f.orders, v)
	if _, ok := ctx.Deadline(); !ok {
		f.t.Fatal("call without deadline")
	}
	if f.fail == "unknown" {
		return statementdata.Proposal{}, nil, errors.New("PRIVATE")
	}
	p := clockTestProposal(f.t, d)
	if v == "base" || f.mode == "reject" {
		p = clockDrop(p)
	}
	proof := *comparisonTestRoute(statementextract.ComparisonCandidates()[0]).Proof
	now := time.Now().UTC()
	b := &statementextract.Billing{GenerationID: fmt.Sprintf("PRIVATE-%d", f.calls), InputTokens: 100, OutputTokens: 10, CostMicros: 10, ComparisonRoute: &statementextract.ComparisonRouteEvidence{Proof: &proof, ObservedAt: now, SelectedModelClass: "alias"}}
	if f.fail == "protocol" {
		p.Rows[0].OperationDate = []string{"foreign"}
	}
	if f.fail == "route" {
		b.ComparisonRoute.SelectedModelClass = "other"
	}
	if f.fail == "capture" {
		p.Service = append(p.Service, statementdata.ServiceMapping{Spans: []string{}, Reason: strings.Repeat("PRIVATE", 6000)})
	}
	return p, b, nil
}

type clockMeterFake struct {
	n                          int64
	reserves, settles, allowed int
	actual                     *int64
	fail                       string
}

func (m *clockMeterFake) Reserve(context.Context, int64, statementextract.Policy) (string, error) {
	m.reserves++
	m.n = ClockPilotMaxMicros
	if m.fail == "reserve" {
		return "", errors.New("PRIVATE")
	}
	return "11111111-1111-4111-8111-111111111111", nil
}
func (m *clockMeterFake) ComparisonAllowed(context.Context, string) (bool, error) {
	m.allowed++
	return m.fail != "block", nil
}
func (m *clockMeterFake) Settle(ctx context.Context, _ string, n *int64) error {
	m.settles++
	m.actual = n
	d, ok := ctx.Deadline()
	if !ok || time.Until(d) > 5*time.Second {
		return errors.New("deadline")
	}
	if m.fail == "settle" {
		return errors.New("PRIVATE")
	}
	return nil
}
func TestClockPilotOrderAccountingAndStop(t *testing.T) {
	for _, mode := range []string{"keep", "reject", "proof", "reserve", "block", "unknown", "route", "protocol", "capture", "settle"} {
		t.Run(mode, func(t *testing.T) {
			f := &clockFake{t: t, fail: mode, mode: mode}
			m := &clockMeterFake{fail: mode}
			r, e := RunClockPilot(context.Background(), Sample{"cards", clockTestDocument(t), FixtureHash("cards")}, f, m, strings.Repeat("a", 40))
			if mode == "keep" || mode == "reject" {
				if e != nil || f.calls != 6 || m.reserves != 1 || m.settles != 1 || m.actual == nil || *m.actual != 60 || m.n != 323400 {
					t.Fatal(r, e)
				}
				want := ClockKeep
				if mode == "reject" {
					want = ClockReject
				}
				if r.Decision != want {
					t.Fatal(r.Decision)
				}
				if ValidateClockReport(r, clockTestDocument(t)) != nil {
					t.Fatal("valid report rejected")
				}
				r.Attempts[0].Measurement.Result.Correct++
				r.Digest = clockDigest(r)
				if ValidateClockReport(r, clockTestDocument(t)) == nil {
					t.Fatal("forged score")
				}
			} else {
				if e == nil || r.Decision != ClockInconclusive || strings.Contains(r.Error, "PRIVATE") {
					t.Fatal(r, e)
				}
				if mode == "proof" || mode == "reserve" || mode == "block" {
					if f.calls != 0 {
						t.Fatal("proof before POST")
					}
				} else if mode == "settle" {
					if f.calls != 6 {
						t.Fatal("settlement")
					}
				} else if f.calls != 1 {
					t.Fatal("retry", f.calls)
				}
				if ValidateClockReport(r, clockTestDocument(t)) != nil {
					t.Fatal("failure report cannot be inspected", mode, r.Error)
				}
				if mode == "unknown" && m.actual != nil {
					t.Fatal("unknown fabricated cost")
				}
			}
		})
	}
}
func TestClockReportRoundTripAndPrivacy(t *testing.T) {
	d := clockTestDocument(t)
	f := &clockFake{t: t}
	m := &clockMeterFake{}
	r, e := RunClockPilot(context.Background(), Sample{"cards", d, FixtureHash("cards")}, f, m, strings.Repeat("a", 40))
	if e != nil {
		t.Fatal(e)
	}
	var out bytes.Buffer
	if EncodeClockReport(&out, r, d) != nil {
		t.Fatal("encode")
	}
	decoded, e := DecodeClockReport(bytes.NewReader(out.Bytes()), d)
	if e != nil {
		t.Fatal(e)
	}
	summary, e := SummarizeClock(decoded, d)
	if e != nil {
		t.Fatal(e)
	}
	raw, _ := json.Marshal(summary)
	if bytes.Contains(raw, []byte("PRIVATE")) || bytes.Contains(raw, []byte("snapshot")) || summary.ActivationAuthorized || summary.RecordedCalls != 6 || !summary.BillingKnown {
		t.Fatal("unsafe summary")
	}
	for _, bad := range []string{strings.Replace(out.String(), `"source":`, `"Source":`, 1), strings.Replace(out.String(), `"source":`, `"source":"x","source":`, 1), out.String() + "{}", strings.Replace(out.String(), `"version":`, `"PRIVATE":"x","version":`, 1), strings.Repeat("x", ClockPilotReportLimit+1)} {
		if _, e := DecodeClockReport(strings.NewReader(bad), d); e == nil {
			t.Fatal("malformed report accepted")
		}
	}
}
