package statementextract

import (
	"context"
	"time"

	"github.com/QahNaaRlN/grace/internal/imports"
	"github.com/QahNaaRlN/grace/internal/statementdata"
)

type Billing struct {
	ScreeningRoute            *ScreeningRouteEvidence
	GenerationID              string
	CostMicros                int64
	InputTokens, OutputTokens int
}
type Reasoner interface {
	CurrentPolicy(context.Context) (Policy, error)
	Complete(context.Context, Policy, Chunk) (statementdata.Proposal, *Billing, error)
}
type PreviewInput struct {
	ExpectedRevision int64    `json:"expectedRevision"`
	VisibleSpanIDs   []string `json:"visibleSpanIds"`
}
type Preview struct {
	ID             string    `json:"id"`
	StatementID    string    `json:"statementId"`
	SourceRevision int64     `json:"sourceRevision"`
	Hash           string    `json:"hash"`
	Plan           Plan      `json:"plan"`
	ExpiresAt      time.Time `json:"expiresAt"`
}
type ConsentInput struct {
	ExpectedRevision int64  `json:"expectedRevision"`
	PreviewID        string `json:"previewId"`
	PreviewHash      string `json:"previewHash"`
}
type Consent struct {
	ID        string `json:"id"`
	PreviewID string `json:"previewId"`
}
type ExtractionInput struct {
	ExpectedRevision int64  `json:"expectedRevision"`
	PreviewID        string `json:"previewId"`
	PreviewHash      string `json:"previewHash"`
	ConsentID        string `json:"consentId"`
}
type Attempt struct {
	Policy           Policy  `json:"policy"`
	PromptVersion    string  `json:"promptVersion"`
	SchemaHash       string  `json:"schemaHash"`
	RedactionVersion string  `json:"redactionVersion"`
	ID               string  `json:"id"`
	StatementID      string  `json:"statementId"`
	Status           string  `json:"status"`
	ErrorCode        *string `json:"errorCode"`
	BatchID          *string `json:"batchId"`
}
type Fault struct {
	Code   string
	Status int
}

func (e *Fault) Error() string            { return e.Code }
func fault(code string, status int) error { return &Fault{code, status} }

type Materializer interface {
	Statement(context.Context, string) (imports.Statement, error)
}
