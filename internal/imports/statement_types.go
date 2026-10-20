package imports

import (
	"context"
	"time"

	"github.com/QahNaaRlN/grace/internal/statementdata"
)

type PDFExtractor interface {
	Extract(context.Context, []byte) (statementdata.Document, error)
}
type Statement struct {
	DraftMapping   *statementdata.DraftMapping `json:"draftMapping,omitempty"`
	SizeBytes      int                         `json:"sizeBytes"`
	PublishedRows  int                         `json:"publishedRows"`
	ExcludedRows   int                         `json:"excludedRows"`
	PageCount      int                         `json:"pageCount"`
	ID             string                      `json:"id"`
	DocumentID     string                      `json:"documentId"`
	AccountID      string                      `json:"accountId"`
	BatchID        *string                     `json:"batchId"`
	Revision       int64                       `json:"revision"`
	FileName       string                      `json:"fileName"`
	SHA256         string                      `json:"sha256"`
	Status         string                      `json:"status"`
	Retention      string                      `json:"retention"`
	ExpiresAt      time.Time                   `json:"expiresAt"`
	DeletedAt      *time.Time                  `json:"deletedAt"`
	CreatedAt      time.Time                   `json:"createdAt"`
	RolesConfirmed bool                        `json:"rolesConfirmed"`
	ReviewedPages  []int                       `json:"reviewedPages"`
	Review         *statementdata.Review       `json:"review"`
	Layout         *statementdata.Document     `json:"layout,omitempty"`
}
type StatementManualDraft struct {
	ExpectedRevision int64                  `json:"expectedRevision"`
	Proposal         statementdata.Proposal `json:"proposal"`
	Reason           string                 `json:"reason"`
}
type StatementEdit struct {
	ExpectedRevision int64  `json:"expectedRevision"`
	AccountID        string `json:"accountId"`
	Retention        string `json:"retention"`
	RolesConfirmed   bool   `json:"rolesConfirmed"`
	ReviewedPages    []int  `json:"reviewedPages"`
	Reason           string `json:"reason"`
}
type StatementDelete struct {
	ExpectedRevision int64 `json:"expectedRevision"`
}

type StatementDocument struct {
	ID          string `json:"id"`
	FileName    string `json:"fileName"`
	SHA256      string `json:"sha256"`
	SizeBytes   int    `json:"sizeBytes"`
	ContentType string `json:"contentType"`
}
type StatementUploadResult struct {
	Statement Statement         `json:"statement"`
	Document  StatementDocument `json:"document"`
	BatchID   *string           `json:"batchId"`
}
