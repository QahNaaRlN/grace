package receipts

import "testing"

func TestNormalizeAndValidateUsesLineTotals(t *testing.T) {
	request := createRequest{
		PurchaseEventID: " purchase-id ",
		Seller:          " Магазин ",
		IssuedAt:        "2026-10-04T12:00:00Z",
		Items: []createItemRequest{
			{Name: " Молоко ", QuantityMilli: 1000, UnitPriceMinor: 10000, LineTotalMinor: 9990},
			{Name: " Хлеб ", QuantityMilli: 2000, UnitPriceMinor: 5000, LineTotalMinor: 10000},
		},
	}

	total, _, ok := normalizeAndValidate(&request)
	if !ok {
		t.Fatal("expected request to be valid")
	}
	if total != 19990 {
		t.Fatalf("expected line total sum 19990, got %d", total)
	}
	if request.PurchaseEventID != "purchase-id" || request.Seller != "Магазин" || request.Items[0].Name != "Молоко" {
		t.Fatalf("expected normalized request, got %#v", request)
	}
}

func TestNormalizeAndValidateRejectsInvalidItems(t *testing.T) {
	request := createRequest{
		PurchaseEventID: "purchase-id", Seller: "Магазин",
		IssuedAt: "2026-10-04T12:00:00Z",
		Items:    []createItemRequest{{Name: " ", QuantityMilli: 1000, UnitPriceMinor: 100, LineTotalMinor: 100}},
	}
	if _, _, ok := normalizeAndValidate(&request); ok {
		t.Fatal("expected invalid item to be rejected")
	}
}
