package financialevents

import "time"

type Posting struct {
	ID        string `json:"id"`
	AccountID string `json:"accountId"`
	Account   string `json:"accountName"`
	Direction string `json:"direction"`
	Amount    int64  `json:"amountMinor"`
	Currency  string `json:"currency"`
}

type Event struct {
	ID              string     `json:"id"`
	Kind            string     `json:"kind"`
	Source          string     `json:"source"`
	CategoryID      *string    `json:"categoryId"`
	CategoryName    *string    `json:"categoryName"`
	CategoryIcon    *string    `json:"categoryIcon"`
	OriginalEventID *string    `json:"originalEventId"`
	Merchant        string     `json:"merchant"`
	Description     string     `json:"description"`
	OccurredAt      *time.Time `json:"occurredAt"`
	OccurredDate    *string    `json:"occurredDate"`
	DatePrecision   string     `json:"datePrecision"`
	CreatedAt       time.Time  `json:"createdAt"`
	Revision        int        `json:"revision"`
	EditedByUser    bool       `json:"editedByUser"`
	VoidedAt        *time.Time `json:"voidedAt"`
	VoidReason      *string    `json:"voidReason"`
	Postings        []Posting  `json:"postings"`
}

type Command struct {
	// AllowUncategorized is internal to reviewed imports, never accepted by manual HTTP requests.
	AllowUncategorized   bool
	Kind                 string
	IdempotencyKey       string
	RequestHash          string
	AccountID            string
	DestinationAccountID string
	CategoryID           *string
	OriginalEventID      string
	AmountMinor          int64
	Merchant             string
	Description          string
	OccurredAt           time.Time
	OccurredDate         string
}

type singleAccountRequest struct {
	AccountID   string  `json:"accountId"`
	CategoryID  *string `json:"categoryId,omitempty"`
	AmountMinor int64   `json:"amountMinor"`
	Merchant    string  `json:"merchant"`
	Description string  `json:"description"`
	OccurredAt  string  `json:"occurredAt"`
}

type transferRequest struct {
	SourceAccountID      string `json:"sourceAccountId"`
	DestinationAccountID string `json:"destinationAccountId"`
	AmountMinor          int64  `json:"amountMinor"`
	Description          string `json:"description"`
	OccurredAt           string `json:"occurredAt"`
}

type refundRequest struct {
	OriginalEventID string `json:"originalEventId"`
	AmountMinor     int64  `json:"amountMinor"`
	Merchant        string `json:"merchant"`
	Description     string `json:"description"`
	OccurredAt      string `json:"occurredAt"`
}
