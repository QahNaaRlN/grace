package transactions

import "testing"

func TestValidateCreateRequest(t *testing.T) {
	categoryID := "20000000-0000-0000-0000-000000000001"
	valid := CreateRequest{
		AccountID:   "10000000-0000-0000-0000-000000000001",
		CategoryID:  &categoryID,
		AmountMinor: 15990,
		Direction:   "expense",
		Kind:        "purchase",
		Source:      "manual",
		Merchant:    "Магазин",
		OccurredAt:  "2026-10-04T12:00:00Z",
	}

	tests := []struct {
		name    string
		mutate  func(*CreateRequest)
		wantErr bool
	}{
		{name: "valid purchase"},
		{name: "zero amount", mutate: func(request *CreateRequest) { request.AmountMinor = 0 }, wantErr: true},
		{name: "direction mismatch", mutate: func(request *CreateRequest) { request.Direction = "income" }, wantErr: true},
		{name: "purchase without category", mutate: func(request *CreateRequest) { request.CategoryID = nil }, wantErr: true},
		{name: "unsupported internal transfer", mutate: func(request *CreateRequest) { request.Kind = "internal_transfer" }, wantErr: true},
		{name: "unknown source", mutate: func(request *CreateRequest) { request.Source = "llm" }, wantErr: true},
		{name: "invalid date", mutate: func(request *CreateRequest) { request.OccurredAt = "tomorrow" }, wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := valid
			if test.mutate != nil {
				test.mutate(&request)
			}
			_, err := validateCreateRequest(request)
			if (err != nil) != test.wantErr {
				t.Fatalf("validateCreateRequest() error = %v, wantErr %v", err, test.wantErr)
			}
		})
	}
}
