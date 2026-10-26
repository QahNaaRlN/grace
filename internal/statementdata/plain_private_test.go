package statementdata

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

// Explicit local-only replay. Private files never become fixtures, logs or provider input.
func TestPlainPrivateSavedAnswer(t *testing.T) {
	answer, reference := os.Getenv("GRACE_PLAIN_SAVED_ANSWER"), os.Getenv("GRACE_PLAIN_REFERENCE")
	if answer == "" || reference == "" {
		t.Skip("private local replay paths not supplied")
	}
	raw, e := os.ReadFile(answer)
	if e != nil {
		t.Fatal("saved answer unavailable")
	}
	expected, e := os.ReadFile(reference)
	if e != nil {
		t.Fatal("reference unavailable")
	}
	var rows []map[string]string
	if json.Unmarshal(expected, &rows) != nil {
		t.Fatal("reference invalid")
	}
	p := ParsePlain(string(raw))
	if p.ParseStatus != "parsed" || len(p.Rows) != len(rows) || len(rows) != 96 {
		t.Fatal("saved reply parse/count mismatch", p.ParseStatus, len(p.Rows))
	}
	norm := func(s string) string { return strings.Join(strings.Fields(s), " ") }
	checks := 0
	for i, row := range rows {
		a := p.Rows[i]
		dates := []*LocalDate{a.OperationDate, a.BookingDate}
		for j, k := range []string{"operationDate", "bookingDate"} {
			d, e := ParseLocalDate(row[k], "")
			if e != nil || dates[j] == nil || *dates[j] != *d {
				t.Errorf("date mismatch row %d field %s", i+1, k)
			}
			checks++
		}
		for _, v := range []struct {
			k, c string
			got  *Money
		}{{"transactionAmount", "transactionCurrency", a.TransactionAmount}, {"accountAmount", "accountCurrency", a.AccountAmount}} {
			m, e := signedMoney(row[v.k], row[v.c])
			if e != nil || v.got == nil || *v.got != *m {
				t.Errorf("amount/currency mismatch row %d", i+1)
			}
			checks += 2
		}
		if a.Description == nil || norm(*a.Description) != norm(row["description"]) {
			t.Errorf("description mismatch row %d", i+1)
		}
		checks++
		if (a.CardLast4 == nil && !absent(row["cardLast4"])) || (a.CardLast4 != nil && norm(*a.CardLast4) != norm(row["cardLast4"])) {
			t.Errorf("card mismatch row %d", i+1)
		}
		checks++
	}
	review, e := p.Review()
	if e != nil || len(review.Reconciliation.Blockers) != 0 || review.Reconciliation.Turnover.Status != "matched" {
		t.Error("saved summary does not reconcile")
	}
	if path := os.Getenv("GRACE_PLAIN_PARSED_OUTPUT"); path != "" {
		encoded, e := json.Marshal(p)
		if e != nil {
			t.Fatal(e)
		}
		if e = os.WriteFile(path, encoded, 0600); e != nil {
			t.Fatal(e)
		}
	}
	t.Logf("private replay rows=%d field checks=%d; no provider calls", len(rows), checks)
}
