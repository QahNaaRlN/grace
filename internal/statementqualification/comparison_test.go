package statementqualification

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/QahNaaRlN/grace/internal/statementdata"
	"github.com/QahNaaRlN/grace/internal/statementextract"
	"strings"
	"testing"
	"time"
)

func comparisonTestRoute(c statementextract.ComparisonCandidate) *statementextract.ComparisonRouteEvidence {
	return &statementextract.ComparisonRouteEvidence{Proof: &statementextract.ComparisonIdentityProof{Candidate: c, CatalogSHA256: strings.Repeat("a", 64), EndpointSHA256: strings.Repeat("b", 64), ZDRSHA256: strings.Repeat("c", 64), CatalogURL: "https://openrouter.ai/api/v1/models", EndpointURL: "https://openrouter.ai/api/v1/models/" + c.Canonical + "/endpoints", ZDRURL: "https://openrouter.ai/api/v1/endpoints/zdr", VerifiedAt: time.Now().UTC().Add(-time.Second)}, ObservedAt: time.Now().UTC(), SelectedModelClass: "pinned_canonical"}
}

type comparisonFake struct {
	candidate statementextract.ComparisonCandidate
	calls     int
	order     *[]string
	mode      string
	cancel    context.CancelFunc
}

func (f *comparisonFake) CurrentPolicy(context.Context) (statementextract.Policy, error) {
	return f.candidate.Policy, nil
}
func (f *comparisonFake) Complete(ctx context.Context, p statementextract.Policy, c statementextract.Chunk) (statementdata.Proposal, *statementextract.Billing, error) {
	f.calls++
	if f.order != nil {
		*f.order = append(*f.order, f.candidate.ID)
	}
	if d, ok := ctx.Deadline(); !ok || time.Until(d) > 120*time.Second {
		panic("missing bound")
	}
	b := &statementextract.Billing{GenerationID: fmt.Sprintf("%s-%d", f.candidate.ID, f.calls), CostMicros: 10, InputTokens: 100, OutputTokens: 10, ComparisonRoute: comparisonTestRoute(f.candidate)}
	if f.cancel != nil {
		f.cancel()
	}
	switch f.mode {
	case "unknown":
		return statementdata.Proposal{}, nil, errors.New("PRIVATE")
	case "protocol":
		return statementdata.Proposal{}, b, errors.New("PRIVATE")
	case "cost":
		b.CostMicros = p.CallMicros() + 1
	case "duplicate":
		b.GenerationID = "duplicate"
	}
	return statementdata.Proposal{Header: statementdata.HeaderMapping{Labels: map[string]statementdata.Selection{}}}, b, nil
}
func comparisonMinimal() []Sample {
	s := minimalSamples()
	return append(s, Sample{"turnover-roles-ru", s[0].Document, FixtureHash("turnover-roles-ru")})
}
func comparisonFakes() ([2]statementextract.Reasoner, *comparisonFake, *comparisonFake) {
	cs := statementextract.ComparisonCandidates()
	a, b := &comparisonFake{candidate: cs[0]}, &comparisonFake{candidate: cs[1]}
	return [2]statementextract.Reasoner{a, b}, a, b
}

type comparisonSettleFail struct{ fakeMeter }

func (m *comparisonSettleFail) Settle(ctx context.Context, id string, n *int64) error {
	m.fakeMeter.Settle(ctx, id, n)
	return errors.New("private")
}
func TestComparisonCampaignMeteringAndStops(t *testing.T) {
	for _, mode := range []string{"quality", "unknown", "protocol", "cost", "duplicate", "budget", "settlement", "cancel", "shared_block"} {
		t.Run(mode, func(t *testing.T) {
			rs, a, b := comparisonFakes()
			order := []string{}
			a.order = &order
			b.order = &order
			m := &fakeMeter{}
			var meter Meter = m
			ctx := context.Background()
			switch mode {
			case "unknown", "protocol", "cost":
				a.mode = mode
			case "duplicate":
				a.mode = mode
				b.mode = mode
			case "budget":
				meter = &unavailableMeter{}
			case "settlement":
				meter = &comparisonSettleFail{}
			case "shared_block":
				m.blockAfter = 1
			case "cancel":
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				defer cancel()
				a.cancel = cancel
			}
			report, e := RunComparison(ctx, comparisonMinimal(), rs, meter)
			if e == nil || report.Passed {
				t.Fatal("fake cannot pass quality", e)
			}
			if mode == "quality" {
				if strings.Join(order, "") != "ABABAB" || m.reserves != 1 || m.reserved != 208500 || m.settles != 1 || m.actual == nil || *m.actual != 60 || ValidateComparisonReport(report, time.Now()) != nil {
					t.Fatal("campaign", e, report, m)
				}
				metrics, e := ComparisonMetricsFor(report, time.Now())
				if e != nil || metrics[0].Completed != 3 || metrics[0].Successful != 0 || metrics[0].Efficiency != "undefined" || metrics[0].CostPerSuccessNumerator != nil {
					t.Fatal("metrics", metrics, e)
				}
				raw, _ := json.Marshal(report)
				if _, e := DecodeComparisonReport(bytes.NewReader(raw)); e != nil {
					t.Fatal(e)
				}
				if _, e := DecodeReport(bytes.NewReader(raw)); e == nil {
					t.Fatal("comparison accepted as activation report")
				}
				report.Tuples[0].State = "not_sent"
				report.Digest = comparisonDigest(report)
				if ValidateComparisonReport(report, time.Now()) == nil {
					t.Fatal("tampered campaign accepted")
				}
			} else if mode == "budget" {
				if len(order) != 0 {
					t.Fatal("budget reached provider")
				}
			} else if mode == "settlement" {
				if report.Error != "qualification_settlement_failed" || report.Passed {
					t.Fatal("settlement reported success")
				}
			} else if mode == "duplicate" {
				if len(order) != 2 || m.actual != nil {
					t.Fatal("duplicate billing guessed", order, m)
				}
			} else {
				if len(order) != 1 || m.settles != 1 {
					t.Fatal("retry or settlement lost", order, m)
				}
				if (mode == "unknown" || mode == "cost") && m.actual != nil {
					t.Fatal("unknown guessed as zero")
				}
				if mode == "protocol" && (m.actual == nil || *m.actual != 10) {
					t.Fatal("known content billing lost")
				}
			}
		})
	}
}
func TestComparisonPreflightRejectsBeforeReserve(t *testing.T) {
	for _, mode := range []string{"hash", "order", "count", "policy"} {
		t.Run(mode, func(t *testing.T) {
			rs, a, b := comparisonFakes()
			s := comparisonMinimal()
			switch mode {
			case "hash":
				s[2].SHA256 = "bad"
			case "order":
				s[0], s[1] = s[1], s[0]
			case "count":
				s = s[:2]
			case "policy":
				b.candidate.Policy.Model = "other"
			}
			m := &fakeMeter{}
			if _, e := RunComparison(context.Background(), s, rs, m); e == nil || a.calls+b.calls != 0 || m.reserves != 0 {
				t.Fatal("unsafe preflight", e, m)
			}
		})
	}
	for _, c := range statementextract.ComparisonCandidates() {
		if _, e := statementextract.BuildPlan(comparisonMinimal()[0].Document, c.Policy, nil); e == nil {
			t.Fatal("comparison plan leaked into product")
		}
	}
	for _, raw := range []string{`{"passed":true,"Passed":false}`, `{} {}`, `{"unknown":true}`, strings.Repeat(" ", 524289)} {
		if _, e := DecodeComparisonReport(strings.NewReader(raw)); e == nil {
			t.Fatal("ambiguous report accepted")
		}
	}
}

func (f *comparisonFake) ComparisonProof(context.Context) (statementextract.ComparisonIdentityProof, error) {
	return *comparisonTestRoute(f.candidate).Proof, nil
}
