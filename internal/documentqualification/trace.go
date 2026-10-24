package documentqualification

import (
	"bytes"
	"context"
	"errors"
	"github.com/QahNaaRlN/grace/internal/assistant"
	"github.com/QahNaaRlN/grace/internal/documentdata"
	"io"
	"net/http"
	"reflect"
	"strings"
	"time"
)

type Budget interface {
	assistant.UsageMeter
	Remaining(context.Context) (int64, error)
	Allowed(context.Context, string) (bool, error)
}
type BudgetEvent struct {
	Kind       string `json:"kind"`
	ID         string `json:"id,omitempty"`
	Purpose    string `json:"purpose,omitempty"`
	Amount     int64  `json:"amount"`
	Actual     *int64 `json:"actual,omitempty"`
	Generation string `json:"generation,omitempty"`
	Allowed    bool   `json:"allowed"`
	Failure    bool   `json:"failure"`
}
type tracedBudget struct {
	inner  Budget
	events []BudgetEvent
}

func (b *tracedBudget) Remaining(ctx context.Context) (int64, error) {
	n, e := b.inner.Remaining(ctx)
	b.events = append(b.events, BudgetEvent{Kind: "remaining", Amount: n, Failure: e != nil})
	return n, e
}
func (b *tracedBudget) Reserve(ctx context.Context, p assistant.ModelPolicy, input, output int, purpose string) (string, error) {
	n, e := p.ReservationMicros(input, output)
	if e != nil {
		return "", e
	}
	id, e := b.inner.Reserve(ctx, p, input, output, purpose)
	b.events = append(b.events, BudgetEvent{Kind: "reserve", ID: id, Purpose: purpose, Amount: n, Failure: e != nil})
	return id, e
}
func (b *tracedBudget) Allowed(ctx context.Context, id string) (bool, error) {
	yes, e := b.inner.Allowed(ctx, id)
	b.events = append(b.events, BudgetEvent{Kind: "guard", ID: id, Allowed: yes, Failure: e != nil})
	return yes, e
}
func (b *tracedBudget) Settle(ctx context.Context, id string, out *assistant.LLMResponse) error {
	v := BudgetEvent{Kind: "settle", ID: id}
	if out != nil {
		n := out.CostMicros
		v.Actual = &n
		v.Generation = out.ID
	}
	e := b.inner.Settle(ctx, id, out)
	v.Failure = e != nil
	b.events = append(b.events, v)
	return e
}

type guardedMeter struct {
	budget Budget
	last   string
}

func (m *guardedMeter) Reserve(ctx context.Context, p assistant.ModelPolicy, input, output int, purpose string) (string, error) {
	n, e := p.ReservationMicros(input, output)
	if e != nil || n > MaximumCallMicros {
		return "", errors.New("campaign_cap")
	}
	id, e := m.budget.Reserve(ctx, p, input, output, purpose)
	if e != nil {
		return "", errors.New("budget_refused")
	}
	m.last = "reserved"
	allowed, e := m.budget.Allowed(ctx, id)
	if e != nil || !allowed {
		zero := assistant.LLMResponse{BillingKnown: true, NotSent: true, ID: "not_sent"}
		if m.Settle(ctx, id, &zero) != nil {
			return "", errors.New("settlement_failed")
		}
		return "", errors.New("budget_refused")
	}
	return id, nil
}
func (m *guardedMeter) Settle(ctx context.Context, id string, out *assistant.LLMResponse) error {
	e := m.budget.Settle(ctx, id, out)
	m.last = "unknown"
	if e != nil {
		m.last = "failed"
	} else if out != nil {
		m.last = "settled"
	}
	return e
}

type WireEvent struct {
	Method        string `json:"method"`
	Path          string `json:"path"`
	RequestSHA256 string `json:"requestSHA256"`
	Status        int    `json:"status"`
	ContentType   string `json:"contentType"`
	Body          []byte `json:"body"`
	Failure       bool   `json:"failure"`
	DurationMS    int64  `json:"durationMs"`
}
type tracedHTTP struct {
	inner                      assistant.HTTPDoer
	events                     []WireEvent
	gets, posts, metadataBytes int
	endpoint                   string
}

func (h *tracedHTTP) Do(r *http.Request) (*http.Response, error) {
	if r.URL.User != nil || r.URL.Scheme != "https" || r.URL.Host != "openrouter.ai" || r.URL.RawQuery != "" || r.URL.Fragment != "" {
		return nil, errors.New("wire_refused")
	}
	path := r.URL.EscapedPath()
	limit := int64(1 << 20)
	if r.Method == http.MethodGet {
		if h.gets >= 6 || (path != "/api/v1/models" && path != "/api/v1/endpoints/zdr" && path != h.endpoint) {
			return nil, errors.New("wire_refused")
		}
		h.gets++
		limit = 8 << 20
	} else if r.Method == http.MethodPost && path == "/api/v1/chat/completions" {
		if h.posts >= 8 {
			return nil, errors.New("wire_refused")
		}
		h.posts++
	} else {
		return nil, errors.New("wire_refused")
	}
	var request []byte
	if r.Body != nil {
		var e error
		request, e = io.ReadAll(io.LimitReader(r.Body, (12<<20)+1))
		if e != nil || len(request) > 12<<20 {
			return nil, errors.New("wire_refused")
		}
		r.Body = io.NopCloser(bytes.NewReader(request))
	}
	if r.Method == http.MethodGet && len(request) > 0 {
		return nil, errors.New("wire_refused")
	}
	event := WireEvent{Method: r.Method, Path: path, RequestSHA256: documentdata.Hash(request)}
	started := time.Now()
	res, e := h.inner.Do(r)
	if e != nil {
		event.Failure = true
		event.DurationMS = time.Since(started).Milliseconds()
		h.events = append(h.events, event)
		return nil, errors.New("transport_unavailable")
	}
	event.Status = res.StatusCode
	event.ContentType = "other"
	if strings.EqualFold(strings.TrimSpace(strings.Split(res.Header.Get("Content-Type"), ";")[0]), "application/json") {
		event.ContentType = "application/json"
	}
	raw, e := io.ReadAll(io.LimitReader(res.Body, limit+1))
	key := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	if len(key) >= 16 && bytes.Contains(raw, []byte(key)) {
		e = errors.New("credential_echo")
		raw = nil
	}
	_ = res.Body.Close()
	if r.Method == http.MethodGet {
		h.metadataBytes += len(raw)
		if h.metadataBytes > 8<<20 {
			e = errors.New("metadata_limit")
		}
	}
	if e != nil {
		event.Failure = true
	} else {
		event.Body = raw
	}
	event.DurationMS = time.Since(started).Milliseconds()
	h.events = append(h.events, event)
	if e != nil {
		return nil, errors.New("transport_unavailable")
	}
	res.Body = io.NopCloser(bytes.NewReader(raw))
	return res, nil
}

type replayHTTP struct {
	events  []WireEvent
	index   int
	invalid bool
}

func (h *replayHTTP) Do(r *http.Request) (*http.Response, error) {
	if h.index >= len(h.events) {
		h.invalid = true
		return nil, errors.New("replay_wire_mismatch")
	}
	v := h.events[h.index]
	h.index++
	var raw []byte
	if r.Body != nil {
		raw, _ = io.ReadAll(r.Body)
	}
	if r.Method != v.Method || r.URL.Scheme != "https" || r.URL.Host != "openrouter.ai" || r.URL.EscapedPath() != v.Path || documentdata.Hash(raw) != v.RequestSHA256 {
		h.invalid = true
		return nil, errors.New("replay_wire_mismatch")
	}
	if v.Failure {
		return nil, errors.New("recorded_transport_failure")
	}
	return &http.Response{StatusCode: v.Status, Header: http.Header{"Content-Type": []string{v.ContentType}}, Body: io.NopCloser(bytes.NewReader(v.Body))}, nil
}

type replayBudget struct {
	events  []BudgetEvent
	index   int
	invalid bool
}

func (b *replayBudget) next(want BudgetEvent) (BudgetEvent, error) {
	if b.index >= len(b.events) {
		b.invalid = true
		return BudgetEvent{}, errors.New("replay_budget_mismatch")
	}
	got := b.events[b.index]
	b.index++
	switch want.Kind {
	case "remaining":
		want.Amount = got.Amount
	case "reserve":
		want.ID = got.ID
	}
	want.Failure = got.Failure
	if !reflect.DeepEqual(want, got) {
		b.invalid = true
		return got, errors.New("replay_budget_mismatch")
	}
	if got.Failure {
		return got, errors.New("recorded_budget_failure")
	}
	return got, nil
}
func (b *replayBudget) Remaining(context.Context) (int64, error) {
	v, e := b.next(BudgetEvent{Kind: "remaining"})
	return v.Amount, e
}
func (b *replayBudget) Reserve(_ context.Context, p assistant.ModelPolicy, input, output int, purpose string) (string, error) {
	n, e := p.ReservationMicros(input, output)
	if e != nil {
		return "", e
	}
	v, e := b.next(BudgetEvent{Kind: "reserve", Amount: n, Purpose: purpose})
	return v.ID, e
}
func (b *replayBudget) Allowed(_ context.Context, id string) (bool, error) {
	if b.index >= len(b.events) {
		b.invalid = true
		return false, errors.New("replay_budget_mismatch")
	}
	yes := b.events[b.index].Allowed
	v, e := b.next(BudgetEvent{Kind: "guard", ID: id, Allowed: yes})
	return v.Allowed, e
}
func (b *replayBudget) Settle(_ context.Context, id string, out *assistant.LLMResponse) error {
	v := BudgetEvent{Kind: "settle", ID: id}
	if out != nil {
		n := out.CostMicros
		v.Actual = &n
		v.Generation = out.ID
	}
	_, e := b.next(v)
	return e
}
