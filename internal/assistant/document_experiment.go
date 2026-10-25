package assistant

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"github.com/QahNaaRlN/grace/internal/documentdata"
	"github.com/QahNaaRlN/grace/internal/receiptdrafts"
	"github.com/QahNaaRlN/grace/internal/statementextract"
	"regexp"
	"time"
)

// ExperimentGrant is an exact-source research permission, never a qualification.
// Only the separately gated research CLI constructs this isolated registry.
type ExperimentGrant struct {
	Source         string    `json:"source"`
	Approval       string    `json:"approval"`
	ManifestSHA256 string    `json:"manifestSHA256"`
	ExpiresAt      time.Time `json:"expiresAt"`
}

func (g ExperimentGrant) Valid(now time.Time) bool {
	sha := regexp.MustCompile(`^[0-9a-f]{40}$`)
	return sha.MatchString(g.Source) && sha.MatchString(g.Approval) && regexp.MustCompile(`^[0-9a-f]{64}$`).MatchString(g.ManifestSHA256) && now.Before(g.ExpiresAt) && g.ExpiresAt.Sub(now) <= 40*time.Minute
}
func NewExperimentModelGroups(member GroupMembership, grant ExperimentGrant, key string, client HTTPDoer) (*ModelGroups, error) {
	return newExperimentGroups(member, grant, key, client, 4)
}
func newExperimentGroups(member GroupMembership, grant ExperimentGrant, key string, client HTTPDoer, maxImages int) (*ModelGroups, error) {
	if !grant.Valid(time.Now()) || member.Policy.Selection != nil {
		return nil, errors.New("experiment_not_authorized")
	}
	member.Qualification = "experiment_only:" + grant.Source + ":" + grant.Approval + ":" + grant.ManifestSHA256
	member.QualifiedUntil = grant.ExpiresAt
	member.Policy.VerifiedAt = time.Now()
	groups := []ModelGroup{}
	for _, purpose := range []string{"statement", "receipt"} {
		groups = append(groups, ModelGroup{Revision: "spec0056-q:" + grant.ManifestSHA256, Purpose: purpose, Profile: DocumentReading, Members: []GroupMembership{member}})
	}
	raw, _ := json.Marshal(groups)
	return newModelGroupsWithLimit(string(raw), key, client, true, maxImages)
}

// ReceiptDocumentProtocol shares the exact product prompt/schema with offline
// manifests. It grants no execution permission and does not access credentials.
func ReceiptDocumentProtocol() (string, json.RawMessage) {
	return receiptDocumentPrompt, receiptDocumentSchema()
}

type ReceiptDocumentResult struct {
	Complete     bool                    `json:"complete"`
	ReceiptCount int                     `json:"receiptCount"`
	Issues       []string                `json:"issues"`
	Extracted    receiptdrafts.Extracted `json:"extracted"`
}

// DecodeDocumentReceipt preserves a model's explicit unreadability for scoring;
// product ExtractDocument still rejects incomplete envelopes before draft edits.
func DecodeDocumentReceipt(out LLMResponse) (ReceiptDocumentResult, error) {
	var shape struct {
		Complete *bool           `json:"complete"`
		Count    *int            `json:"receiptCount"`
		Issues   []string        `json:"issues"`
		Receipt  json.RawMessage `json:"receipt"`
	}
	var result ReceiptDocumentResult
	if statementextract.DirectJSON(out.Content, &shape) != nil || shape.Complete == nil || shape.Count == nil || *shape.Count < 0 || *shape.Count > 50 || shape.Issues == nil || len(shape.Issues) > 32 {
		return result, errors.New("provider_protocol")
	}
	for _, s := range shape.Issues {
		if len(s) == 0 || len(s) > 200 {
			return result, errors.New("provider_protocol")
		}
	}
	var fields map[string]json.RawMessage
	if json.Unmarshal(shape.Receipt, &fields) != nil || len(fields) != 8 {
		return result, errors.New("provider_protocol")
	}
	for _, key := range []string{"seller", "issuedAt", "issuedDate", "currency", "total", "fiscalDocumentNumber", "items", "sourceText"} {
		if _, ok := fields[key]; !ok {
			return result, errors.New("provider_protocol")
		}
	}
	if bytes.Equal(bytes.TrimSpace(fields["sourceText"]), []byte("null")) {
		return result, errors.New("provider_protocol")
	}
	var items []map[string]json.RawMessage
	if json.Unmarshal(fields["items"], &items) != nil || items == nil {
		return result, errors.New("provider_protocol")
	}
	for _, item := range items {
		if len(item) != 8 {
			return result, errors.New("provider_protocol")
		}
		for _, k := range []string{"name", "quantity", "unitPrice", "lineTotal", "discount", "brand", "package", "broadGroup"} {
			if _, ok := item[k]; !ok {
				return result, errors.New("provider_protocol")
			}
		}
	}
	out.Content = shape.Receipt
	extracted, err := decodeReceipt(out)
	return ReceiptDocumentResult{Complete: *shape.Complete, ReceiptCount: *shape.Count, Issues: shape.Issues, Extracted: extracted}, err
}
func (e *ReceiptExtractor) PrepareExperimentDocument(ctx context.Context, doc documentdata.Document) (LLMRequest, error) {
	if e == nil || e.Verifier == nil || e.Verifier.Groups == nil || !e.Verifier.Groups.experiment {
		return LLMRequest{}, errors.New("experiment_not_authorized")
	}
	return e.documentRequest(ctx, doc)
}
func (e *ReceiptExtractor) ReadExperimentDocument(ctx context.Context, doc documentdata.Document) (ReceiptDocumentResult, error) {
	request, err := e.PrepareExperimentDocument(ctx, doc)
	if err != nil {
		return ReceiptDocumentResult{}, err
	}
	if e.Meter == nil {
		return ReceiptDocumentResult{}, errors.New("experiment_not_authorized")
	}
	out, err := MeteredCompletion(ctx, e.Provider, e.Meter, request, "receipt")
	if err != nil {
		return ReceiptDocumentResult{}, err
	}
	return DecodeDocumentReceipt(out)
}

// NewPlainDocumentExperiment is SPEC-0057 research only. Production constructors
// and the older fixed-corpus experiment retain strict JSON and four images.
func NewPlainDocumentExperiment(member GroupMembership, grant ExperimentGrant, key string, client HTTPDoer) (*OpenRouterProvider, *ModelGroups, error) {
	if member.MaxImages != 5 {
		return nil, nil, errors.New("experiment_not_authorized")
	}
	groups, e := newExperimentGroups(member, grant, key, client, 5)
	if e != nil {
		return nil, nil, e
	}
	provider, e := NewGroupedOpenRouterProvider(key, groups, client)
	if e != nil {
		return nil, nil, e
	}
	provider.plainDocumentExperiment = true
	return provider, groups, nil
}
