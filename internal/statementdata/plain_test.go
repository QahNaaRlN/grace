package statementdata

import (
	"strings"
	"testing"
)

const plainSample = `| Дата и время операции | Дата и время списания | Сумма и валюта операции | Сумма и валюта карты | Описание | Карта |
| --- | --- | --- | --- | --- | --- |
| 18.10.2026 12:34 | 19.10.2026 01:02 | -123,45 RUB | -123,45 ₽ | Кафе \| кофе | 1234 |
| 17.10.2026 11:25 | 17.10.2026 11:26 | +200,00 RUB | +200,00 RUB | Свой перевод | 1234 |
Период: 01.10.2026 — 19.10.2026
Доступный остаток на 19.10.2026: 10,00 ₽
Итог пополнений: 200,00 ₽
Итог списаний: 123,45 ₽`

func TestPlainReadingPreservesValuesAndKeepsFinancialRoles(t *testing.T) {
	p := ParsePlain(plainSample)
	if p.ParseStatus != "parsed" || len(p.Rows) != 2 {
		t.Fatal(p.ParseStatus, len(p.Rows))
	}
	a := p.Rows[0]
	if a.OperationDate.Local != "2026-10-18T12:34:00" || a.BookingDate.Local != "2026-10-19T01:02:00" || a.OperationDate.UTC != "" || a.AccountAmount.AmountMinor != -12345 || *a.Description != "Кафе | кофе" || *a.CardLast4 != "1234" {
		t.Fatal(a)
	}
	if p.RawAnswer != plainSample {
		t.Fatal("raw reply changed")
	}
	review, e := p.Review()
	if e != nil || review.Reconciliation.Turnover.Status != "matched" || len(review.Reconciliation.Blockers) > 0 {
		t.Fatal(review.Reconciliation, e)
	}
	if review.Header.OpeningBookBalance != nil || review.Header.ClosingBookBalance != nil || review.Rows[0].AccountAmount.AmountMinor != 12345 || len(review.Rows[0].Evidence) != 0 {
		t.Fatal("invented balance/evidence or wrong posting")
	}
}
func TestPlainReadingNeverRepairsMissingClockOrBadTable(t *testing.T) {
	p := ParsePlain(strings.Replace(plainSample, "18.10.2026 12:34", "18.10.2026", 1))
	if p.Rows[0].OperationDate.Precision != "date" || p.Rows[0].OperationDate.Local != "2026-10-18" {
		t.Fatal("invented clock")
	}
	for _, raw := range []string{strings.Replace(plainSample, `Кафе \| кофе`, `Кафе | кофе`, 1), plainSample + "\n| broken |", strings.Replace(plainSample, "| 18.10.2026", "18.10.2026", 1)} {
		p := ParsePlain(raw)
		if p.ParseStatus != "parse_error" {
			t.Fatal("ambiguous/truncated row accepted", p.ParseStatus)
		}
	}
	p = ParsePlain(strings.Replace(plainSample, "-123,45 ₽", "-123,45 XYZ", 1))
	if p.Rows[0].AccountAmount == nil || p.Rows[0].AccountAmount.Currency == "RUB" || len(p.Rows[0].Issues) == 0 {
		t.Fatal("currency inferred")
	}
}
func TestPlainDraftRecomputesTypedFieldsAndDigest(t *testing.T) {
	p := ParsePlain(plainSample)
	p.Rows[0].AccountAmount.AmountMinor = 1
	p.Rows[0].Issues = nil
	if e := p.Normalize(); e != nil || p.Rows[0].AccountAmount.AmountMinor != -12345 {
		t.Fatal("trusted forged typed value", e)
	}
	state := PlainState{SourceSHA256: strings.Repeat("a", 64), Candidate: p}
	before := state.CurrentDigest()
	state.Candidate.Rows[0].Raw["description"] = "Исправлено"
	if before == state.CurrentDigest() {
		t.Fatal("edit retained review digest")
	}
	p.Rows[0].Raw["accountAmount"] = "1e3"
	if e := p.Normalize(); e != nil || p.Rows[0].AccountAmount != nil || len(p.Rows[0].Issues) == 0 {
		t.Fatal("non-decimal accepted", e)
	}
}
func TestPlainReadingDoesNotSilentlyDropRowsAtLimit(t *testing.T) {
	lines := strings.Split(plainSample, "\n")
	raw := strings.Join(lines[:2], "\n") + "\n" + strings.Repeat(lines[2]+"\n", 201)
	if p := ParsePlain(raw); p.ParseStatus != "parse_error" {
		t.Fatal("truncated 201 rows into valid draft")
	}
}

func TestPlainHeaderErrorKeepsEditableRawAnswer(t *testing.T) {
	p := ParsePlain(plainSample)
	p.Header["creditTurnover"] = "ambiguous"
	review, e := p.Review()
	if e != nil || len(review.Reconciliation.Blockers) == 0 || p.RawAnswer != plainSample {
		t.Fatal("bad header discarded answer", e)
	}
}

func TestPlainForeignOriginalCurrencyAndHeaderOrder(t *testing.T) {
	p := ParsePlain(strings.Replace(plainSample, "-123,45 RUB", "-5,00 USD", 1))
	if p.Rows[0].TransactionAmount == nil || p.Rows[0].TransactionAmount.Currency != "USD" || p.Rows[0].TransactionAmount.AmountMinor != -500 {
		t.Fatal("explicit original currency lost")
	}
	p = ParsePlain(strings.Replace(plainSample, "Дата и время операции | Дата и время списания", "Дата и время списания | Дата и время операции", 1))
	if p.ParseStatus != "parse_error" {
		t.Fatal("swapped headers silently accepted")
	}
}
