package statementdata

import "testing"

func TestMoneyExactAndBounded(t *testing.T) {
	for _, tc := range []struct {
		literal   string
		n         int64
		direction string
	}{{"−1\u202f234,56 ₽", 123456, "debit"}, {"+0.01 RUB", 1, "credit"}, {"90 071 992 547 409,91", MaxWireMinor, ""}, {"001.2", 120, ""}} {
		got, dir, e := ParseMoney(tc.literal, "RUB")
		if e != nil || got.AmountMinor != tc.n || dir != tc.direction {
			t.Errorf("%q: %+v %q %v", tc.literal, got, dir, e)
		}
	}
	for _, s := range []string{"1.001", "1,234.56", "1 23,45", "NaN", "1e9", "90071992547409.92", "9999999999999999999999999999", "1-2", "1 USD", ""} {
		if _, _, e := ParseMoney(s, "RUB"); e == nil {
			t.Errorf("accepted %q", s)
		}
	}
	if _, _, e := ParseMoney("1.00", ""); e == nil {
		t.Fatal("invented currency")
	}
}
func TestLocalClocksAndProjection(t *testing.T) {
	d, e := ParseLocalDate("19.10.2026 10:23:45", "")
	if e != nil || d.Local != "2026-10-19T10:23:45" || d.UTC != "" || d.Precision != "local_datetime" {
		t.Fatal(d, e)
	}
	d, e = ParseLocalDate("19.10.2026 10:23:45", "Asia/Yekaterinburg")
	if e != nil || d.UTC != "2026-10-19T05:23:45Z" {
		t.Fatal(d, e)
	}
	d, e = ParseLocalDate("19.10.2026", "Asia/Yekaterinburg")
	if e != nil || d.UTC != "" || d.Precision != "date" {
		t.Fatal("invented midnight", d, e)
	}
	for _, tc := range []struct{ s, z string }{{"31.02.2026", ""}, {"2026-10-19", "invented/bank"}, {"2026-03-29 02:30", "Europe/Berlin"}, {"2026-10-25 02:30", "Europe/Berlin"}} {
		if _, e := ParseLocalDate(tc.s, tc.z); e == nil {
			t.Errorf("accepted %+v", tc)
		}
	}
}
func TestReconciliationDoesNotInventBalance(t *testing.T) {
	rows := []Row{{Direction: "debit", AccountAmount: &Money{1234, "RUB"}}, {Direction: "credit", AccountAmount: &Money{2000, "RUB"}}}
	header := Header{AvailableBalanceObserved: &Money{7777, "RUB"}, DebitTurnover: &Money{1234, "RUB"}, CreditTurnover: &Money{2000, "RUB"}}
	r := Reconcile(rows, header)
	if r.Turnover.Status != "matched" || r.BookBalance.Status != "not_provided" || r.LedgerBalance.Status != "not_evaluated" || len(r.Blockers) != 0 {
		t.Fatal(r)
	}
	for i := range rows {
		rows[i].OperationDate = &LocalDate{Local: "2026-10-19", Precision: "date"}
	}
	header.OpeningBookDate = &LocalDate{Local: "2026-10-18", Precision: "date"}
	header.ClosingBookDate = &LocalDate{Local: "2026-10-23", Precision: "date"}
	header.OpeningBookBalance = &Money{10000, "RUB"}
	header.ClosingBookBalance = &Money{10766, "RUB"}
	if got := Reconcile(rows, header); got.BookBalance.Status != "matched" {
		t.Fatal(got)
	}
	header.ClosingBookBalance.AmountMinor++
	if got := Reconcile(rows, header); got.BookBalance.Status != "mismatch" || len(got.Blockers) == 0 {
		t.Fatal(got)
	}
	header.ClosingBookBalance = nil
	header.DebitTurnover.AmountMinor++
	if got := Reconcile(rows, header); got.Turnover.Status != "mismatch" {
		t.Fatal(got)
	}
	rows[0].AccountAmount.AmountMinor = MaxWireMinor
	rows = append(rows, Row{Direction: "debit", AccountAmount: &Money{1, "RUB"}})
	if got := Reconcile(rows, header); got.DebitMinor != nil || len(got.Blockers) == 0 {
		t.Fatal("overflow reported as known", got)
	}
}
