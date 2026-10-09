package receipts

import "time"

type Item struct {
	ID             string  `json:"id"`
	Position       int     `json:"position"`
	Name           string  `json:"name"`
	QuantityMilli  int64   `json:"quantityMilli"`
	UnitPriceMinor int64   `json:"unitPriceMinor"`
	LineTotalMinor int64   `json:"lineTotalMinor"`
	Brand          *string `json:"brand"`
	Package        *string `json:"package"`
	BroadGroup     *string `json:"broadGroup"`
}

type Receipt struct {
	ID                   string     `json:"id"`
	PurchaseEventID      string     `json:"purchaseEventId"`
	Seller               string     `json:"seller"`
	IssuedAt             *time.Time `json:"issuedAt"`
	IssuedDate           *string    `json:"issuedDate"`
	DatePrecision        string     `json:"datePrecision"`
	Currency             string     `json:"currency"`
	TotalMinor           int64      `json:"totalMinor"`
	FiscalDocumentNumber string     `json:"fiscalDocumentNumber"`
	CreatedAt            time.Time  `json:"createdAt"`
	Items                []Item     `json:"items"`
}

type ItemInput struct {
	Name           string  `json:"name"`
	QuantityMilli  int64   `json:"quantityMilli"`
	UnitPriceMinor int64   `json:"unitPriceMinor"`
	LineTotalMinor int64   `json:"lineTotalMinor"`
	Brand          *string `json:"brand,omitempty"`
	Package        *string `json:"package,omitempty"`
	BroadGroup     *string `json:"broadGroup,omitempty"`
}
type createItemRequest = ItemInput

type createRequest struct {
	PurchaseEventID      string              `json:"purchaseEventId"`
	Seller               string              `json:"seller"`
	IssuedAt             string              `json:"issuedAt"`
	FiscalDocumentNumber string              `json:"fiscalDocumentNumber"`
	Items                []createItemRequest `json:"items"`
}

type Command struct {
	PurchaseEventID      string
	Seller               string
	IssuedAt             time.Time
	IssuedDate           string
	FiscalDocumentNumber string
	Items                []createItemRequest
	TotalMinor           int64
	IdempotencyKey       string
	RequestHash          string
}
