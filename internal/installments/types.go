// Package installments connects purchases, debt accounts and bank schedules
// without creating a second expense when principal is repaid (SPEC-0054).
package installments

import (
	"errors"
	"strings"
	"time"
)

const MaxMoney int64 = 9007199254740991

var ErrInvalid = errors.New("Проверьте данные рассрочки, счёт, покупку и график.")
var ErrConflict = errors.New("Данные изменились или операция уже связана. Обновите данные и проверьте распределение.")

type Payment struct {
	ID             string `json:"id"`
	Position       int    `json:"position"`
	DueOn          string `json:"dueOn"`
	AmountMinor    int64  `json:"amountMinor"`
	PaidMinor      int64  `json:"paidMinor"`
	RemainingMinor int64  `json:"remainingMinor"`
}
type Plan struct {
	Drafts                []Draft   `json:"drafts"`
	ID                    string    `json:"id"`
	ObligationID          string    `json:"obligationId"`
	Name                  string    `json:"name"`
	Mode                  string    `json:"mode"`
	Provider              string    `json:"provider"`
	ContractNumber        string    `json:"contractNumber"`
	Currency              string    `json:"currency"`
	AccountID             string    `json:"accountId"`
	PurchaseEventID       *string   `json:"purchaseEventId"`
	OpenedOn              string    `json:"openedOn"`
	ObservedOn            string    `json:"observedOn"`
	PrincipalMinor        int64     `json:"principalMinor"`
	InitialRemainingMinor int64     `json:"initialRemainingMinor"`
	PaidMinor             int64     `json:"paidMinor"`
	RemainingMinor        int64     `json:"remainingMinor"`
	AccountDebtMinor      *int64    `json:"accountDebtMinor"`
	DifferenceMinor       *int64    `json:"differenceMinor"`
	Historical            bool      `json:"historical"`
	Reminders             bool      `json:"reminders"`
	Revision              int       `json:"revision"`
	Payments              []Payment `json:"payments"`
}
type Row struct {
	DueOn       string `json:"dueOn"`
	AmountMinor int64  `json:"amountMinor"`
}
type Purchase struct {
	CategoryID  string `json:"categoryId"`
	Merchant    string `json:"merchant"`
	Description string `json:"description"`
}
type FirstPayment struct {
	SourceAccountID string `json:"sourceAccountId"`
	AmountMinor     int64  `json:"amountMinor"`
	On              string `json:"on"`
}
type CreateRequest struct {
	Name                  string        `json:"name"`
	Mode                  string        `json:"mode"`
	Provider              string        `json:"provider"`
	ContractNumber        string        `json:"contractNumber"`
	Currency              string        `json:"currency"`
	AccountID             string        `json:"accountId"`
	PurchaseEventID       string        `json:"purchaseEventId"`
	Purchase              *Purchase     `json:"purchase"`
	OpenedOn              string        `json:"openedOn"`
	ObservedOn            string        `json:"observedOn"`
	PrincipalMinor        int64         `json:"principalMinor"`
	InitialRemainingMinor int64         `json:"initialRemainingMinor"`
	Historical            bool          `json:"historical"`
	Reminders             bool          `json:"reminders"`
	Payments              []Row         `json:"payments"`
	FirstPayment          *FirstPayment `json:"firstPayment"`
}
type Allocation struct {
	PaymentID   string `json:"paymentId"`
	AmountMinor int64  `json:"amountMinor"`
}
type AllocationRequest struct {
	Revision int          `json:"revision"`
	Reason   string       `json:"reason"`
	Items    []Allocation `json:"items"`
}
type AllocationView struct {
	AccountID        string              `json:"accountId"`
	Currency         string              `json:"currency"`
	EventID          string              `json:"eventId"`
	Revision         int                 `json:"revision"`
	CreditMinor      int64               `json:"creditMinor"`
	UnallocatedMinor int64               `json:"unallocatedMinor"`
	Items            []Allocation        `json:"items"`
	History          []AllocationHistory `json:"history"`
}
type AllocationHistory struct {
	Previous  any       `json:"previous"`
	Current   any       `json:"current"`
	Reason    string    `json:"reason"`
	ChangedAt time.Time `json:"changedAt"`
}
type Draft struct {
	ID           string `json:"id"`
	Revision     int    `json:"revision"`
	PlanRevision int    `json:"planRevision"`
	Payments     []Row  `json:"payments"`
	Applied      bool   `json:"applied"`
}
type DraftRequest struct {
	Revision int   `json:"revision"`
	Payments []Row `json:"payments"`
}

func validDay(s string) bool {
	t, e := time.Parse("2006-01-02", s)
	return e == nil && t.Year() > 0 && t.Format("2006-01-02") == s
}
func validRows(rows []Row, total int64, minDay string) bool {
	if len(rows) == 0 || len(rows) > 600 || total <= 0 || total > MaxMoney {
		return false
	}
	var sum int64
	prev := minDay
	for _, r := range rows {
		if !validDay(r.DueOn) || r.DueOn < prev || r.AmountMinor <= 0 || r.AmountMinor > total-sum {
			return false
		}
		sum += r.AmountMinor
		prev = r.DueOn
	}
	return sum == total
}
func (c *CreateRequest) normalize() error {
	c.Name = strings.TrimSpace(c.Name)
	c.Provider = strings.TrimSpace(c.Provider)
	c.Currency = strings.ToUpper(strings.TrimSpace(c.Currency))
	if c.ObservedOn == "" {
		c.ObservedOn = c.OpenedOn
	}
	if c.InitialRemainingMinor == 0 && !c.Historical {
		c.InitialRemainingMinor = c.PrincipalMinor
	}
	if len([]rune(c.Name)) < 1 || len([]rune(c.Name)) > 200 || len([]rune(c.Provider)) < 1 || len([]rune(c.Provider)) > 200 || len([]rune(c.ContractNumber)) > 200 || len(c.Currency) != 3 {
		return ErrInvalid
	}
	for _, v := range c.Currency {
		if v < 'A' || v > 'Z' {
			return ErrInvalid
		}
	}
	if c.Mode != "standalone" && c.Mode != "credit_card" {
		return ErrInvalid
	}
	if !validDay(c.OpenedOn) || !validDay(c.ObservedOn) || c.ObservedOn < c.OpenedOn || c.ObservedOn > Today() || c.PrincipalMinor <= 0 || c.PrincipalMinor > MaxMoney || c.InitialRemainingMinor > c.PrincipalMinor || !validRows(c.Payments, c.InitialRemainingMinor, c.ObservedOn) {
		return ErrInvalid
	}
	if c.Mode == "credit_card" && c.AccountID == "" {
		return ErrInvalid
	}
	if c.Purchase != nil && (c.PurchaseEventID != "" || c.Historical || c.Purchase.CategoryID == "") {
		return ErrInvalid
	}
	if !c.Historical && (c.ObservedOn != c.OpenedOn || c.InitialRemainingMinor != c.PrincipalMinor || (c.Purchase == nil && c.PurchaseEventID == "")) {
		return ErrInvalid
	}
	if p := c.FirstPayment; p != nil {
		if p.SourceAccountID == "" || !validDay(p.On) || p.On < c.ObservedOn || p.On > Today() || p.AmountMinor <= 0 || p.AmountMinor > c.InitialRemainingMinor {
			return ErrInvalid
		}
	}
	return nil
}
func Today() string { return time.Now().UTC().Add(5 * time.Hour).Format("2006-01-02") }
