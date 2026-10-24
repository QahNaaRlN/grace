package documentqualification

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/QahNaaRlN/grace/internal/assistant"
	"github.com/QahNaaRlN/grace/internal/documentdata"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Result struct {
	InputTokens  *int         `json:"inputTokens"`
	OutputTokens *int         `json:"outputTokens"`
	Ordinal      int          `json:"ordinal"`
	Case         string       `json:"case"`
	State        string       `json:"state"`
	Reason       string       `json:"reason"`
	Measurement  *Measurement `json:"measurement,omitempty"`
	ActualMicros *int64       `json:"actualMicros,omitempty"`
	Settlement   string       `json:"settlement"`
}
type Core struct {
	Proofs   map[string]string `json:"proofs"`
	Decision string            `json:"decision"`
	Results  []Result          `json:"results"`
}
type Report struct {
	Version    string                    `json:"version"`
	Manifest   Manifest                  `json:"manifest"`
	Grant      assistant.ExperimentGrant `json:"grant"`
	StartedAt  time.Time                 `json:"startedAt"`
	FinishedAt time.Time                 `json:"finishedAt"`
	Core       Core                      `json:"core"`
	Wire       []WireEvent               `json:"wire"`
	Budget     []BudgetEvent             `json:"budget"`
}

func (r Report) Hash() string { return documentdata.Digest(r) }

type observedProvider struct {
	inner  assistant.ReasoningProvider
	out    assistant.LLMResponse
	called bool
}

func (p *observedProvider) Complete(ctx context.Context, in assistant.LLMRequest) (assistant.LLMResponse, error) {
	p.called = true
	v, e := p.inner.Complete(ctx, in)
	p.out = v
	return v, e
}

type meteredProvider struct {
	provider assistant.ReasoningProvider
	meter    assistant.UsageMeter
}

func (p meteredProvider) Complete(ctx context.Context, in assistant.LLMRequest) (assistant.LLMResponse, error) {
	return assistant.MeteredCompletion(ctx, p.provider, p.meter, in, "statement")
}
func Run(ctx context.Context, c Corpus, m Manifest, g assistant.ExperimentGrant, key string, client assistant.HTTPDoer, budget Budget) (Report, error) {
	r := Report{Version: Version, Manifest: m, Grant: g, StartedAt: time.Now().UTC()}
	if m.Validate(c, m.Source, true) != nil || g.Source != m.Source || g.ManifestSHA256 != m.Hash() || !g.Valid(time.Now()) || budget == nil {
		return r, errors.New("execution_not_authorized")
	}
	if client == nil {
		client = &http.Client{Timeout: 180 * time.Second, Transport: &http.Transport{Proxy: nil}, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	}
	parts := strings.Split(m.Candidate.Canonical, "/")
	if len(parts) != 2 {
		return r, errors.New("candidate_invalid")
	}
	wire := &tracedHTTP{inner: client, endpoint: "/api/v1/models/" + url.PathEscape(parts[0]) + "/" + url.PathEscape(parts[1]) + "/endpoints"}
	meter := &tracedBudget{inner: budget}
	r.Core = runCore(ctx, c, m, g, key, wire, meter)
	r.Wire = wire.events
	r.Budget = meter.events
	r.FinishedAt = time.Now().UTC()
	return r, nil
}
func runCore(ctx context.Context, c Corpus, m Manifest, g assistant.ExperimentGrant, key string, client assistant.HTTPDoer, budget Budget) Core {
	out := Core{Proofs: map[string]string{}, Decision: "INCONCLUSIVE", Results: []Result{}}
	for i, id := range IDs() {
		out.Results = append(out.Results, Result{Ordinal: i + 1, Case: id, State: "not_sent", Reason: "preflight", Settlement: "not_reserved"})
	}
	groups, e := assistant.NewExperimentModelGroups(*m.Candidate, g, key, client)
	if e != nil {
		return out
	}
	provider, e := assistant.NewGroupedOpenRouterProvider(key, groups, client)
	if e != nil {
		return out
	}
	observed := &observedProvider{inner: provider}
	meter := &guardedMeter{budget: budget, last: "not_reserved"}
	reasoner := &assistant.DocumentReasoner{Groups: groups, Provider: meteredProvider{observed, meter}}
	receipt := &assistant.ReceiptExtractor{Verifier: &assistant.PolicyVerifier{Groups: groups}, Provider: observed, Meter: meter}
	// Resolve both purposes and validate every concrete request before any reserve/POST.
	var total int64
	for _, v := range c.Cases {
		var n int64
		if v.Golden != nil {
			p, err := reasoner.CurrentDocumentPolicy(ctx, len(v.Document.Pages))
			if err != nil || reasoner.PrepareDocument(ctx, p, v.Document) != nil {
				return out
			}
			out.Proofs["statement"] = p.ProofSHA256
			n = p.MaximumMicros
		} else {
			req, err := receipt.PrepareExperimentDocument(ctx, v.Document)
			if err != nil {
				return out
			}
			out.Proofs["receipt"] = req.Policy.Selection.ProofSHA256
			n, err = req.Policy.ReservationMicros(req.InputTokenUpperBound, req.MaxOutputTokens)
			if err != nil {
				return out
			}
		}
		if n <= 0 || n > MaximumCallMicros {
			return out
		}
		total += n
	}
	if total > MaximumMicros {
		return out
	}
	remaining, e := budget.Remaining(ctx)
	if e != nil || remaining < total {
		return out
	}
	stop := ""
	for i, v := range c.Cases {
		if stop != "" {
			out.Results[i].Reason = stop
			continue
		}
		observed.out = assistant.LLMResponse{}
		observed.called = false
		meter.last = "not_reserved"
		r := out.Results[i]
		if v.Golden != nil {
			policy, err := reasoner.CurrentDocumentPolicy(ctx, len(v.Document.Pages))
			if err != nil {
				e = err
			} else {
				_, _, e = reasoner.CompleteDocument(ctx, policy, v.Document)
			}
		} else {
			_, e = receipt.ReadExperimentDocument(ctx, v.Document)
		}
		r.Settlement = meter.last
		if observed.out.BillingKnown {
			if observed.called && !observed.out.NotSent {
				input, output := observed.out.PromptTokens, observed.out.CompletionTokens
				r.InputTokens = &input
				r.OutputTokens = &output
			}
			n := observed.out.CostMicros
			r.ActualMicros = &n
		}
		if !observed.called || observed.out.NotSent {
			r.Reason = "pre_post_refusal"
			stop = "pre_post_refusal"
		} else {
			r.State = "observed"
			if !observed.out.BillingKnown || meter.last != "settled" {
				r.Reason = "billing_requires_review"
				stop = r.Reason
			} else if len(observed.out.Content) == 0 {
				r.Reason = "protocol_refusal"
				stop = r.Reason
			} else {
				measured := Measure(v, observed.out)
				r.Measurement = &measured
				if !measured.Protocol || !measured.Recognition || measured.NativeBinding == "fail" {
					r.State = "rejected"
					r.Reason = "quality_reject"
					out.Decision = "REJECT"
					stop = "decisive_quality_reject"
				} else if e != nil {
					r.Reason = "reader_refusal"
					stop = r.Reason
				} else {
					r.State = "completed"
					r.Reason = ""
				}
			}
		}
		out.Results[i] = r
	}
	if stop == "" {
		out.Decision = "PASS_SYNTHETIC"
	}
	return out
}

// Receipt/statement raw observations are recovered from recorded HTTP through the
// actual gateway during Replay. No saved score or BillingKnown flag is trusted.
func EncodeReport(r Report) ([]byte, error) {
	raw, e := json.Marshal(struct {
		SHA256 string `json:"sha256"`
		Report Report `json:"report"`
	}{r.Hash(), r})
	if e != nil || len(raw) > ReportLimit {
		return nil, errors.New("report_invalid")
	}
	return raw, nil
}
