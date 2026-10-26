package receipts

import (
	"math"
	"strings"

	"github.com/QahNaaRlN/grace/internal/financialevents"
)

func ValidateCommand(c Command) error {
	if c.TotalMinor <= 0 || c.PurchaseEventID == "" || strings.TrimSpace(c.Seller) == "" || len(c.Seller) > 200 || len(c.FiscalDocumentNumber) > 160 || c.IdempotencyKey == "" || len(c.IdempotencyKey) > 160 || c.RequestHash == "" || len(c.Items) < 1 || len(c.Items) > 200 {
		return ErrInvalid
	}
	if c.IssuedDate != "" {
		if !c.IssuedAt.IsZero() || !financialevents.ValidCalendarDate(c.IssuedDate) {
			return ErrInvalid
		}
	} else if c.IssuedAt.IsZero() {
		return ErrInvalid
	}
	var total int64
	for _, i := range c.Items {
		if strings.TrimSpace(i.Name) == "" || len(i.Name) > 300 || i.QuantityMilli <= 0 || i.UnitPriceMinor < 0 || i.LineTotalMinor < 0 || total > math.MaxInt64-i.LineTotalMinor {
			return ErrInvalid
		}
		total += i.LineTotalMinor
		for _, v := range []*string{i.Brand, i.Package, i.BroadGroup} {
			if v != nil && (strings.TrimSpace(*v) == "" || len(*v) > 300) {
				return ErrInvalid
			}
		}
	}
	if total != c.TotalMinor {
		return ErrInvalid
	}
	return nil
}
