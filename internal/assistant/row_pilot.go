package assistant

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"reflect"
	"sync"
	"sync/atomic"
	"time"

	"github.com/QahNaaRlN/grace/internal/statementdata"
	"github.com/QahNaaRlN/grace/internal/statementextract"
)

// RowPilotReasoner is an operator-only A/cards transport, never product wiring.
// Construction performs no I/O. Only the separately authorized CLI calls it.
type RowPilotReasoner struct {
	comparison     *ComparisonReasoner
	doc            statementdata.Document
	grouping       *statementextract.RowGrouping
	nextRow, posts int
	calls          sync.Mutex
	metadataCount  atomic.Int32
}
type rowMetadataDoer struct {
	owner *RowPilotReasoner
	next  HTTPDoer
}

func (t rowMetadataDoer) Do(q *http.Request) (*http.Response, error) {
	if q.Method != http.MethodGet {
		return nil, errors.New("row_metadata_invalid")
	}
	for {
		n := t.owner.metadataCount.Load()
		if n >= 39 {
			return nil, errors.New("row_metadata_limit")
		}
		if t.owner.metadataCount.CompareAndSwap(n, n+1) {
			break
		}
	}
	return t.next.Do(q)
}
func NewRowPilotReasoner(key string, d statementdata.Document) (*RowPilotReasoner, error) {
	if d.Hash() != statementextract.ClockCardsDocumentHash {
		return nil, errors.New("row_fixture_invalid")
	}
	c, e := NewStatementComparisonReasoner(key, "A")
	if e != nil {
		return nil, e
	}
	c.clockExactPrices = true
	r := &RowPilotReasoner{comparison: c}
	raw, _ := json.Marshal(d)
	_ = json.Unmarshal(raw, &r.doc)
	c.client = rowMetadataDoer{r, c.client}
	return r, nil
}
func (r *RowPilotReasoner) CurrentPolicy(ctx context.Context) (statementextract.Policy, error) {
	return r.comparison.CurrentPolicy(ctx)
}
func (r *RowPilotReasoner) ComparisonProof(ctx context.Context) (statementextract.ComparisonIdentityProof, error) {
	return r.comparison.ComparisonProof(ctx)
}
func (r *RowPilotReasoner) MetadataRequests() int { return int(r.metadataCount.Load()) }
func (r *RowPilotReasoner) CompleteRowPilot(ctx context.Context, q statementextract.RowScopedRequest, proof statementextract.ComparisonIdentityProof) ([]byte, *statementextract.Billing, error) {
	fail := errors.New("row_request_invalid")
	if !r.calls.TryLock() {
		return nil, nil, fail
	}
	defer r.calls.Unlock()
	if r.posts >= 12 || !proof.ValidAt(time.Now().UTC()) || proof.Candidate != statementextract.ComparisonCandidates()[0] {
		return nil, nil, fail
	}
	r.comparison.mu.Lock()
	same := r.comparison.proof != nil && reflect.DeepEqual(*r.comparison.proof, proof)
	r.comparison.mu.Unlock()
	if !same {
		return nil, nil, fail
	}
	var want statementextract.RowScopedRequest
	var e error
	switch q.Stage {
	case "base":
		want, e = statementextract.BuildRowPilotBase(r.doc)
	case "grouping":
		want, e = statementextract.BuildRowGrouping(r.doc)
	case "row":
		if r.grouping == nil || q.RowOrdinal != r.nextRow {
			return nil, nil, fail
		}
		want, e = statementextract.BuildRowFields(r.doc, *r.grouping, q.RowOrdinal)
	default:
		return nil, nil, fail
	}
	if e != nil || !reflect.DeepEqual(q, want) {
		return nil, nil, fail
	}
	if q.Stage == "grouping" {
		r.grouping = nil
		r.nextRow = 1
	}
	r.posts++
	policy := ModelPolicy{Model: q.Policy.Model, Provider: q.Policy.Provider, ProviderName: q.Policy.ProviderName, InputMicrosPerMillion: q.Policy.InputMicrosPerMillion, OutputMicrosPerMillion: q.Policy.OutputMicrosPerMillion, VerifiedAt: proof.VerifiedAt}
	messages := []LLMMessage{}
	for _, m := range q.Messages {
		messages = append(messages, LLMMessage{Role: m["role"], Content: m["content"]})
	}
	out, e := r.comparison.provider.Complete(ctx, LLMRequest{comparisonProof: &proof, DisableReasoning: true, Policy: policy, Messages: messages, Schema: q.Schema, InputTokenUpperBound: statementextract.InputBound, MaxOutputTokens: statementextract.OutputBound})
	raw := append([]byte(nil), out.Content...)
	if !out.BillingKnown {
		if e == nil {
			e = errors.New("row_billing_unavailable")
		}
		return raw, nil, e
	}
	bill := &statementextract.Billing{ComparisonRoute: out.ComparisonRoute, GenerationID: out.ID, CostMicros: out.CostMicros, InputTokens: out.PromptTokens, OutputTokens: out.CompletionTokens}
	if e != nil {
		return raw, bill, e
	}
	if q.Stage == "grouping" {
		if g, err := statementextract.DecodeRowGrouping(r.doc, raw); err == nil {
			r.grouping = &g
		}
	}
	if q.Stage == "row" {
		r.nextRow++
	}
	return raw, bill, nil
}
