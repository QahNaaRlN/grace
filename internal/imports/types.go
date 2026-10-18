// Package imports persists reviewed source records and confirms ledger writes.
// Financial state remains in financialevents; review and preview never write it.
package imports

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"regexp"
	"time"

	"github.com/QahNaaRlN/grace/internal/financialevents"
	"github.com/QahNaaRlN/grace/internal/importreview"
)

type Fault struct {
	Code   string
	Status int
}

func (e *Fault) Error() string            { return e.Code }
func fault(code string, status int) error { return &Fault{code, status} }

var ErrNotFound = fault("not_found", 404)
var uuid = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

func digest(v any) string       { b, _ := json.Marshal(v); return hashBytes(b) }
func hashBytes(b []byte) string { h := sha256.Sum256(b); return hex.EncodeToString(h[:]) }

type Fields struct {
	Kind                 string  `json:"kind"`
	Currency             string  `json:"currency"`
	AmountMinor          int64   `json:"amountMinor"`
	AccountID            string  `json:"accountId,omitempty"`
	DestinationAccountID string  `json:"destinationAccountId,omitempty"`
	CategoryID           *string `json:"categoryId,omitempty"`
	OriginalEventID      string  `json:"originalEventId,omitempty"`
	OccurredAt           string  `json:"occurredAt,omitempty"`
	OccurredDate         string  `json:"occurredDate,omitempty"`
	Merchant             string  `json:"merchant"`
	Description          string  `json:"description"`
	TimeReason           string  `json:"timeReason,omitempty"`
}
type Resolution struct {
	CandidateID string `json:"candidateId"`
	Reason      string `json:"reason"`
}
type Decision struct {
	Action            string       `json:"action"`
	Reason            string       `json:"reason,omitempty"`
	Fields            *Fields      `json:"fields,omitempty"`
	EventID           string       `json:"eventId,omitempty"`
	CanonicalRecordID string       `json:"canonicalRecordId,omitempty"`
	Resolutions       []Resolution `json:"resolutions,omitempty"`
}
type Batch struct {
	StatementID  *string `json:"statementId,omitempty"`
	created      bool
	ID           string                `json:"id"`
	Source       importreview.Metadata `json:"source"`
	PayloadHash  string                `json:"payloadHash"`
	Revision     int64                 `json:"revision"`
	Status       string                `json:"status"`
	CreatedAt    time.Time             `json:"createdAt"`
	Counts       map[string]int        `json:"counts"`
	DomainCounts map[string]int        `json:"domainCounts"`
	// SPEC-0044: what the history and the review summary show.
	FileName         *string              `json:"fileName"`
	Format           *string              `json:"format"`
	PeriodFrom       *string              `json:"periodFrom"`
	PeriodTo         *string              `json:"periodTo"`
	Total            int                  `json:"total"`
	Reviewed         int                  `json:"reviewed"`
	NeedsReview      int                  `json:"needsReview"`
	LastConfirmation *ConfirmationSummary `json:"lastConfirmation"`
}
type ConfirmationSummary struct {
	ID            string    `json:"id"`
	At            time.Time `json:"at"`
	CreatedEvents int       `json:"createdEvents"`
	Outcomes      int       `json:"outcomes"`
}
type Candidate struct {
	ID          string                 `json:"id"`
	RecordID    string                 `json:"recordId,omitempty"`
	EventID     string                 `json:"eventId,omitempty"`
	Rule        string                 `json:"rule"`
	Differences []string               `json:"differences"`
	Source      *importreview.Record   `json:"source,omitempty"`
	Event       *financialevents.Event `json:"event,omitempty"`
}
type Audit struct {
	Revision  int64     `json:"revision"`
	ActorID   string    `json:"actorId"`
	CreatedAt time.Time `json:"createdAt"`
	Decision  Decision  `json:"decision"`
}
type Record struct {
	ID         string              `json:"id"`
	Source     importreview.Record `json:"source"`
	Decision   Decision            `json:"decision"`
	Published  bool                `json:"published"`
	Candidates []Candidate         `json:"candidates"`
	History    []Audit             `json:"history"`
	EventID    *string             `json:"eventId"`
	// Questions = source issues + match candidates; Go counts them at upload.
	Questions int `json:"questions"`
}
type Change struct {
	AccountID   string `json:"accountId"`
	Currency    string `json:"currency"`
	DeltaMinor  int64  `json:"deltaMinor"`
	AccountName string `json:"accountName"`
}
type Line struct {
	PDFOrdinal  *int               `json:"pdfOrdinal,omitempty"`
	SourceKind  string             `json:"sourceKind,omitempty"`
	RecordID    string             `json:"recordId"`
	SourceSheet string             `json:"sourceSheet,omitempty"`
	SourceRow   int                `json:"sourceRow,omitempty"`
	Decision    Decision           `json:"decision"`
	SourceDate  *importreview.Date `json:"sourceDate,omitempty"`
	Candidates  []Candidate        `json:"candidates"`
	Blockers    []string           `json:"blockers"`
	Warnings    []string           `json:"warnings"`
	Postings    []Change           `json:"postings"`
}
type Preview struct {
	StatementRevision int64     `json:"statementRevision,omitempty"`
	ID                string    `json:"id"`
	BatchID           string    `json:"batchId"`
	Revision          int64     `json:"revision"`
	Hash              string    `json:"hash"`
	ExpiresAt         time.Time `json:"expiresAt"`
	Lines             []Line    `json:"lines"`
	Changes           []Change  `json:"changes"`
	Blockers          []string  `json:"blockers"`
	NewEvents         int       `json:"newEvents"`
	Links             int       `json:"links"`
	Exclusions        int       `json:"exclusions"`
}
type Outcome struct {
	RecordID string                 `json:"recordId"`
	Action   string                 `json:"action"`
	EventID  *string                `json:"eventId"`
	Event    *financialevents.Event `json:"event,omitempty"`
}
type Receipt struct {
	ID        string    `json:"id"`
	BatchID   string    `json:"batchId"`
	PreviewID string    `json:"previewId"`
	CreatedAt time.Time `json:"createdAt"`
	Outcomes  []Outcome `json:"outcomes"`
}
type Edit struct {
	ExpectedRevision int64    `json:"expectedRevision"`
	Decision         Decision `json:"decision"`
}
type Selection struct {
	ExpectedRevision int64         `json:"expectedRevision"`
	RecordIDs        []string      `json:"recordIds,omitempty"`
	Filter           *RecordFilter `json:"filter,omitempty"`
}

// RecordFilter selects rows by what the review list shows, not by ids.
type RecordFilter struct {
	Domain string `json:"domain,omitempty"`
	Status string `json:"status,omitempty"`
	Q      string `json:"q,omitempty"`
}
type Confirmation struct {
	PreviewID   string `json:"previewId"`
	PreviewHash string `json:"previewHash"`
}
type Page[T any] struct {
	Items      []T    `json:"items"`
	NextCursor string `json:"nextCursor,omitempty"`
	// Total is the number of rows matching the filter (records only).
	Total *int `json:"total,omitempty"`
}

func validateDecision(d Decision) error {
	switch d.Action {
	case "unresolved":
	case "exclude":
		if d.Reason == "" {
			return fault("reason_required", 422)
		}
	case "create":
		if d.Fields == nil {
			return fault("fields_required", 422)
		}
	case "link_existing":
		if !uuid.MatchString(d.EventID) || d.Reason == "" {
			return fault("invalid_link", 422)
		}
	case "duplicate_of":
		if !uuid.MatchString(d.CanonicalRecordID) || d.Reason == "" {
			return fault("invalid_duplicate", 422)
		}
	default:
		return fault("invalid_action", 400)
	}
	if len(d.Reason) > 2000 || len(d.Resolutions) > 1000 {
		return fault("review_limit", 422)
	}
	if d.Action != "create" && d.Fields != nil || d.Action != "link_existing" && d.EventID != "" || d.Action != "duplicate_of" && d.CanonicalRecordID != "" {
		return fault("invalid_action_fields", 400)
	}
	seen := map[string]bool{}
	for _, r := range d.Resolutions {
		if r.CandidateID == "" || r.Reason == "" || len(r.Reason) > 2000 || seen[r.CandidateID] {
			return fault("invalid_resolution", 422)
		}
		seen[r.CandidateID] = true
	}
	return nil
}
func asFault(err error) *Fault {
	var f *Fault
	if errors.As(err, &f) {
		return f
	}
	return &Fault{"internal_error", 500}
}
