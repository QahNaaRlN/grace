package obligations

import "testing"

func TestNormalizeAndValidateCreate(t *testing.T) {
	request := createRequest{
		Name: " Рассрочка ", Kind: "installment", Currency: " rub ", TotalMinor: 300,
		Payments: []createPaymentRequest{
			{DueOn: "2026-11-01", AmountMinor: 100},
			{DueOn: "2026-12-01", AmountMinor: 200},
		},
	}
	if !normalizeAndValidateCreate(&request) {
		t.Fatal("expected valid fixed obligation")
	}
	if request.Name != "Рассрочка" || request.Currency != "RUB" {
		t.Fatalf("expected normalized request, got %#v", request)
	}
}

func TestNormalizeAndValidateCreateRejectsInvalidSchedule(t *testing.T) {
	tests := []struct {
		name     string
		payments []createPaymentRequest
		total    int64
	}{
		{name: "sum mismatch", payments: []createPaymentRequest{{DueOn: "2026-11-01", AmountMinor: 100}}, total: 101},
		{name: "duplicate date", payments: []createPaymentRequest{{DueOn: "2026-11-01", AmountMinor: 100}, {DueOn: "2026-11-01", AmountMinor: 100}}, total: 200},
		{name: "decreasing dates", payments: []createPaymentRequest{{DueOn: "2026-12-01", AmountMinor: 100}, {DueOn: "2026-11-01", AmountMinor: 100}}, total: 200},
		{name: "invalid date", payments: []createPaymentRequest{{DueOn: "2026-02-30", AmountMinor: 100}}, total: 100},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := createRequest{Name: "Долг", Kind: "loan", Currency: "RUB", TotalMinor: test.total, Payments: test.payments}
			if normalizeAndValidateCreate(&request) {
				t.Fatal("expected schedule to be rejected")
			}
		})
	}
}

func TestValidCurrency(t *testing.T) {
	for _, value := range []string{"RU", "RUBB", "R1B", "руб"} {
		if validCurrency(value) {
			t.Fatalf("expected %q to be invalid", value)
		}
	}
	if !validCurrency("RUB") {
		t.Fatal("expected RUB to be valid")
	}
}
