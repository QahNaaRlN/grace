package accounts

import (
	"encoding/json"
	"testing"
)

func TestCardValidation(t *testing.T) {
	cases := []struct {
		name, typ, bank, last string
		valid                 bool
	}{
		{"leading zeros", "debit", "Банк", "0012", true},
		{"credit", "credit", "Банк", "1234", true},
		{"missing bank", "debit", "", "1234", false},
		{"short", "debit", "Банк", "123", false},
		{"full number", "debit", "Банк", "1234567890123456", false},
		{"unicode digits", "debit", "Банк", "１２３４", false},
		{"cash card", "cash", "Банк", "1234", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r := CreateRequest{Name: "Счёт", Type: c.typ, Currency: "rub", OpeningBalanceMinor: moneyPointer(0), Card: &CardRequest{Name: "Карта", Bank: c.bank, Last4: c.last}}
			if validateRequest(&r) != c.valid {
				t.Fatal("validation mismatch")
			}
			if c.valid && r.Card.Last4 != c.last {
				t.Fatal("last4 changed")
			}
		})
	}
	r := CreateRequest{Name: "Наличные", Type: "cash", Currency: "RUB", OpeningBalanceMinor: moneyPointer(0)}
	if !validateRequest(&r) {
		t.Fatal("cash must not require card")
	}
}

func moneyPointer(value int64) *int64 { return &value }

func TestOpeningBalanceKnowledge(t *testing.T) {
	for _, tc := range []struct {
		body  string
		valid bool
	}{
		{`{"name":"Cash","type":"cash","currency":"RUB","openingBalanceMinor":0}`, true},
		{`{"name":"Cash","type":"cash","currency":"RUB","openingBalanceMinor":null,"balanceReconciled":false}`, true},
		{`{"name":"Cash","type":"cash","currency":"RUB","openingBalanceMinor":100,"balanceReconciled":false}`, true},
		{`{"name":"Cash","type":"cash","currency":"RUB","openingBalanceMinor":null}`, false},
		{`{"name":"Cash","type":"cash","currency":"RUB","balanceReconciled":false}`, false},
		{`{"name":"Cash","type":"cash","currency":"RUB","openingBalanceMinor":0,"extra":1}`, false},
	} {
		var request CreateRequest
		err := json.Unmarshal([]byte(tc.body), &request)
		if got := err == nil && validateRequest(&request); got != tc.valid {
			t.Fatalf("%s valid=%v err=%v", tc.body, got, err)
		}
	}
}
