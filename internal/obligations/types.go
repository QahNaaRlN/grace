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
	PrincipalMinor *int64       `json:"principalMinor"`
	InterestMinor  *int64       `json:"interestMinor"`
	OtherMinor     *int64       `json:"otherMinor"`
	PartsSource    *string      `json:"partsSource"`
	PaidMinor      int64        `json:"paidMinor"`
	RemainingMinor int64        `json:"remainingMinor"`
	Status         string       `json:"status"`
	IsOverdue      bool         `json:"isOverdue"`
	DueSoon        bool         `json:"dueSoon"`
	DaysUntilDue   int          `json:"daysUntilDue"`
	Settlements    []Settlement `json:"settlements"`
}

type Obligation struct {
	ManagedInstallment bool               `json:"managedInstallment,omitempty"`
	ID                 string             `json:"id"`
	Name               string             `json:"name"`
	Kind               string             `json:"kind"`
	Currency           string             `json:"currency"`
	TotalMinor         int64              `json:"totalMinor"`
	PaidMinor          int64              `json:"paidMinor"`
	RemainingMinor     int64              `json:"remainingMinor"`
	Status             string             `json:"status"`
	CreatedAt          time.Time          `json:"createdAt"`
	Payments           []ScheduledPayment `json:"payments"`
}

type createPaymentRequest struct {
	DueOn       string `json:"dueOn"`
	AmountMinor int64  `json:"amountMinor"`
	// SPEC-0053: optional split of a loan row; all three or none.
	PrincipalMinor *int64 `json:"principalMinor,omitempty"`
	InterestMinor  *int64 `json:"interestMinor,omitempty"`
	OtherMinor     *int64 `json:"otherMinor,omitempty"`
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
	// SPEC-0053: for a loan, book the amount above the row remainder as a prepayment.
	PrepayRemainder bool `json:"prepayRemainder"`
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
	PrepayRemainder  bool
	IdempotencyKey   string
	RequestHash      string
}
