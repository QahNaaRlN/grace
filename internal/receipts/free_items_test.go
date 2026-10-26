package receipts

import (
	"math"
	"testing"
	"time"
)

func TestFreeReceiptItemsKeepPositivePurchaseAndOverflowGuards(t *testing.T) {
	fixture := func() Command {
		return Command{PurchaseEventID: "purchase", Seller: "Synthetic shop", IssuedDate: "2026-08-14", TotalMinor: 100, IdempotencyKey: "receipt", RequestHash: "hash", Items: []ItemInput{{Name: "Paid", QuantityMilli: 1000, UnitPriceMinor: 100, LineTotalMinor: 100}, {Name: "Free", QuantityMilli: 2000, UnitPriceMinor: 0, LineTotalMinor: 0}}}
	}
	if err := ValidateCommand(fixture()); err != nil {
		t.Fatal(err)
	}
	for _, change := range []func(*Command){
		func(c *Command) { c.Items = c.Items[1:]; c.TotalMinor = 0 },
		func(c *Command) { c.TotalMinor = -1 },
		func(c *Command) { c.Items[1].QuantityMilli = 0 },
		func(c *Command) { c.Items[1].UnitPriceMinor = -1 },
		func(c *Command) { c.Items[1].LineTotalMinor = -1 },
		func(c *Command) { c.Items[0].LineTotalMinor = math.MaxInt64; c.Items[1].LineTotalMinor = 1 },
	} {
		c := fixture()
		change(&c)
		if ValidateCommand(c) == nil {
			t.Fatal("unsafe receipt accepted")
		}
	}
	c := fixture()
	request := createRequest{PurchaseEventID: c.PurchaseEventID, Seller: c.Seller, IssuedAt: time.Date(2026, 8, 14, 7, 8, 9, 0, time.UTC).Format(time.RFC3339), Items: c.Items}
	if total, _, ok := normalizeAndValidate(&request); !ok || total != 100 {
		t.Fatal("HTTP rejected free item")
	}
	request.Items = request.Items[1:]
	if _, _, ok := normalizeAndValidate(&request); ok {
		t.Fatal("HTTP accepted all-free purchase")
	}
}
