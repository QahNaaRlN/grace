package obligations

import "time"

type Settlement struct {
	ID                 string    `json:"id"`
	ScheduledPaymentID string    `json:"scheduledPaymentId"`
	FinancialEventID   string    `json:"financialEventId"`
	AmountMinor        int64     `json:"amountMinor"`
	CreatedAt          time.Time `json:"createdAt"`
}

type ScheduledPayment struct {
	ID             string       `json:"id"`
	Position       int          `json:"position"`
	DueOn          string       `json:"dueOn"`
	AmountMinor    int64        `json:"amountMinor"`
	PaidMinor      int64        `json:"paidMinor"`
	RemainingMinor int64        `json:"remainingMinor"`
	Status         string       `json:"status"`
	IsOverdue      bool         `json:"isOverdue"`
	DueSoon        bool         `json:"dueSoon"`
	DaysUntilDue   int          `json:"daysUntilDue"`
	Settlements    []Settlement `json:"settlements"`
}

type Obligation struct {
	ID             string             `json:"id"`
	Name           string             `json:"name"`
	Kind           string             `json:"kind"`
	Currency       string             `json:"currency"`
	TotalMinor     int64              `json:"totalMinor"`
	PaidMinor      int64              `json:"paidMinor"`
	RemainingMinor int64              `json:"remainingMinor"`
	Status         string             `json:"status"`
	CreatedAt      time.Time          `json:"createdAt"`
	Payments       []ScheduledPayment `json:"payments"`
}

type createPaymentRequest struct {
	DueOn       string `json:"dueOn"`
	AmountMinor int64  `json:"amountMinor"`
}

type createRequest struct {
	Name       string                 `json:"name"`
	Kind       string                 `json:"kind"`
	Currency   string                 `json:"currency"`
	TotalMinor int64                  `json:"totalMinor"`
	Payments   []createPaymentRequest `json:"payments"`
}

type settlementRequest struct {
	FinancialEventID string `json:"financialEventId"`
}

type CreateCommand struct {
	Name           string
	Kind           string
	Currency       string
	TotalMinor     int64
	Payments       []createPaymentRequest
	IdempotencyKey string
	RequestHash    string
	AsOf           string
}

type SettlementCommand struct {
	ObligationID     string
	PaymentID        string
	FinancialEventID string
	IdempotencyKey   string
	RequestHash      string
}
