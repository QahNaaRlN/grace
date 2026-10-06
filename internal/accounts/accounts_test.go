package accounts

import "testing"

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
			r := CreateRequest{Name: "Счёт", Type: c.typ, Currency: "rub", Card: &CardRequest{Name: "Карта", Bank: c.bank, Last4: c.last}}
			if validateRequest(&r) != c.valid {
				t.Fatal("validation mismatch")
			}
			if c.valid && r.Card.Last4 != c.last {
				t.Fatal("last4 changed")
			}
		})
	}
	r := CreateRequest{Name: "Наличные", Type: "cash", Currency: "RUB"}
	if !validateRequest(&r) {
		t.Fatal("cash must not require card")
	}
}
