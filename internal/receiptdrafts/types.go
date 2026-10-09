package receiptdrafts

import (
	"context"
	"encoding/json"
	"time"
)

type Item struct {
	Name            *string          `json:"name"`
	Quantity        *string          `json:"quantity"`
	UnitPrice       *string          `json:"unitPrice"`
	LineTotal       *string          `json:"lineTotal"`
	Discount        *string          `json:"discount"`
	Brand           *string          `json:"brand"`
	Package         *string          `json:"package"`
	BroadGroup      *string          `json:"broadGroup"`
	GroupSuggestion *GroupSuggestion `json:"groupSuggestion,omitempty"`
}
type GroupSuggestion struct {
	Group           string             `json:"group"`
	Confidence      float64            `json:"confidence"`
	Probabilities   map[string]float64 `json:"probabilities"`
	QuestionVersion string             `json:"questionVersion"`
}
type Fields struct {
	Seller               *string `json:"seller"`
	IssuedAt             *string `json:"issuedAt"`
	IssuedDate           *string `json:"issuedDate"`
	Currency             *string `json:"currency"`
	Total                *string `json:"total"`
	FiscalDocumentNumber *string `json:"fiscalDocumentNumber"`
	Items                []Item  `json:"items"`
	AccountID            string  `json:"accountId"`
	CategoryID           *string `json:"categoryId"`
	Action               string  `json:"action"`
	ExistingEventID      string  `json:"existingEventId"`
	CandidateReason      string  `json:"candidateReason"`
}
type Extracted struct {
	Fields     Fields `json:"fields"`
	SourceText string `json:"sourceText"`
	Model      string `json:"model"`
	Provider   string `json:"provider"`
	CostMicros string `json:"costMicros"`
}
type Extractor interface {
	Extract(context.Context, []byte) (Extracted, error)
	Available() bool
}
type GroupSuggester interface {
	Suggest(context.Context, []Item) ([]*GroupSuggestion, error)
}
type Draft struct {
	ID            string      `json:"id"`
	Revision      int64       `json:"revision"`
	Status        string      `json:"status"`
	Fields        Fields      `json:"fields"`
	Extractions   []Extracted `json:"extractions"`
	Warnings      []string    `json:"warnings"`
	PhotoStatus   string      `json:"photoStatus"`
	ExpiresAt     time.Time   `json:"expiresAt"`
	ReceiptID     *string     `json:"receiptId"`
	EventID       *string     `json:"eventId"`
	ImageID       string      `json:"-"`
	ImageHash     string      `json:"-"`
	ExtractionKey string      `json:"-"`
}
type Edit struct {
	Revision int64  `json:"revision"`
	Fields   Fields `json:"fields"`
}
type Candidate struct {
	EventID     string `json:"eventId"`
	Description string `json:"description"`
	AmountMinor string `json:"amountMinor"`
	Currency    string `json:"currency"`
	HasReceipt  bool   `json:"hasReceipt"`
}
type Preview struct {
	ID              string          `json:"id"`
	Hash            string          `json:"hash"`
	Revision        int64           `json:"revision"`
	ExpiresAt       time.Time       `json:"expiresAt"`
	Action          string          `json:"action"`
	AmountMinor     string          `json:"amountMinor"`
	Currency        string          `json:"currency"`
	AccountID       string          `json:"accountId"`
	ExistingEventID string          `json:"existingEventId"`
	Candidates      []Candidate     `json:"candidates"`
	Warnings        []string        `json:"warnings"`
	Payload         json.RawMessage `json:"-"`
}
