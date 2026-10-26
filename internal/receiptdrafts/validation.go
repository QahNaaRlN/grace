package receiptdrafts

import (
	"errors"
	"math"
	"math/big"
	"regexp"
	"strings"
	"time"

	"github.com/QahNaaRlN/grace/internal/financialevents"
	"github.com/QahNaaRlN/grace/internal/receipts"
)

var decimalValue = regexp.MustCompile(`^(0|[1-9][0-9]*)(?:[.,]([0-9]+))?$`)

func ParseScaled(value string, precision int) (int64, error) {
	value = strings.TrimSpace(value)
	if len(value) > 40 || precision < 0 || precision > 3 {
		return 0, errors.New("invalid_amount")
	}
	m := decimalValue.FindStringSubmatch(value)
	if m == nil || len(m[2]) > precision {
		return 0, errors.New("invalid_amount")
	}
	n, ok := new(big.Int).SetString(m[1]+m[2]+strings.Repeat("0", precision-len(m[2])), 10)
	if !ok || !n.IsInt64() {
		return 0, errors.New("invalid_amount")
	}
	return n.Int64(), nil
}
func value(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

type Validated struct {
	Event   financialevents.Command
	Receipt receipts.Command
	Fields  Fields
}

func Validate(f Fields) (Validated, error) {
	if f.Currency == nil || *f.Currency != "RUB" || f.Total == nil || len(f.Items) < 1 || len(f.Items) > 50 || f.Seller == nil || strings.TrimSpace(*f.Seller) == "" || len(*f.Seller) > 200 {
		return Validated{}, errors.New("extraction_incomplete")
	}
	total, err := ParseScaled(*f.Total, 2)
	if err != nil || total <= 0 {
		return Validated{}, errors.New("invalid_amount")
	}
	var instant time.Time
	date := value(f.IssuedDate)
	if f.IssuedAt != nil && *f.IssuedAt != "" {
		instant, err = time.Parse(time.RFC3339, *f.IssuedAt)
		if err != nil {
			return Validated{}, errors.New("invalid_date")
		}
	}
	if date != "" {
		if !instant.IsZero() || !financialevents.ValidCalendarDate(date) {
			return Validated{}, errors.New("invalid_date")
		}
	} else if instant.IsZero() {
		return Validated{}, errors.New("extraction_incomplete")
	}
	if f.Action != "create" && f.Action != "link_existing" {
		return Validated{}, errors.New("decision_required")
	}
	if f.Action == "create" && f.AccountID == "" || f.Action == "link_existing" && f.ExistingEventID == "" {
		return Validated{}, errors.New("decision_required")
	}
	items := []receipts.ItemInput{}
	var sum int64
	for _, i := range f.Items {
		if i.Name == nil || strings.TrimSpace(*i.Name) == "" || len(*i.Name) > 300 || i.Quantity == nil || i.UnitPrice == nil || i.LineTotal == nil {
			return Validated{}, errors.New("extraction_incomplete")
		}
		quantity, err := ParseScaled(*i.Quantity, 3)
		if err != nil || quantity <= 0 {
			return Validated{}, errors.New("invalid_quantity")
		}
		unit, err := ParseScaled(*i.UnitPrice, 2)
		if err != nil || unit < 0 {
			return Validated{}, errors.New("invalid_amount")
		}
		line, err := ParseScaled(*i.LineTotal, 2)
		if err != nil || line < 0 {
			return Validated{}, errors.New("invalid_amount")
		}
		discount := int64(0)
		if i.Discount != nil {
			discount, err = ParseScaled(*i.Discount, 2)
			if err != nil {
				return Validated{}, errors.New("invalid_amount")
			}
		}
		gross := new(big.Int).Mul(big.NewInt(quantity), big.NewInt(unit))
		gross.Add(gross, big.NewInt(500))
		gross.Quo(gross, big.NewInt(1000))
		gross.Sub(gross, big.NewInt(discount))
		if !gross.IsInt64() || gross.Int64() != line {
			return Validated{}, errors.New("totals_mismatch")
		}
		if sum > math.MaxInt64-line {
			return Validated{}, errors.New("invalid_amount")
		}
		sum += line
		items = append(items, receipts.ItemInput{Name: strings.TrimSpace(*i.Name), QuantityMilli: quantity, UnitPriceMinor: unit, LineTotalMinor: line, Brand: i.Brand, Package: i.Package, BroadGroup: i.BroadGroup})
	}
	if sum != total {
		return Validated{}, errors.New("totals_mismatch")
	}
	if len(value(f.FiscalDocumentNumber)) > 160 || len(f.CandidateReason) > 1000 {
		return Validated{}, errors.New("invalid_request")
	}
	event := financialevents.Command{Kind: "purchase", AmountMinor: total, AccountID: f.AccountID, CategoryID: f.CategoryID, OccurredAt: instant, OccurredDate: date, Merchant: strings.TrimSpace(*f.Seller), Description: "Покупка по подтверждённому чеку"}
	receipt := receipts.Command{PurchaseEventID: f.ExistingEventID, Seller: strings.TrimSpace(*f.Seller), IssuedAt: instant, IssuedDate: date, TotalMinor: total, FiscalDocumentNumber: value(f.FiscalDocumentNumber), Items: items}
	// Exact optional-field validation remains in the financial receipt module.
	check := receipt
	check.PurchaseEventID = "preview"
	check.IdempotencyKey = "preview"
	check.RequestHash = "preview"
	if err := receipts.ValidateCommand(check); err != nil {
		return Validated{}, errors.New("invalid_request")
	}
	return Validated{event, receipt, f}, nil
}
