package imports

import (
	"encoding/csv"
	"strings"
	"testing"

	"github.com/QahNaaRlN/grace/internal/importreview"
)

func csvFixture(delimiter rune, count int) []byte {
	var body strings.Builder
	w := csv.NewWriter(&body)
	w.Comma = delimiter
	_ = w.Write([]string{"Финансовое здоровье"})
	_ = w.Write([]string{""})
	_ = w.Write([]string{"Описание листа"})
	_ = w.Write([]string{"Дата", "Тип операции", "Описание", "Категория", "Подкатегория", "Сумма", "Источник денег", "Счёт", "Приоритет", "Регулярный", "Получатель", "Комментарий", "Документ", "Статус", "ID"})
	for i := 0; i < count; i++ {
		_ = w.Write([]string{"01.09.2026", "Расход", "Молоко, сыр; хлеб\nдве строки", "Еда", "", "762,75", "Собственные средства", "Карта", "", "Нет", "Магазин", "", "", "", ""})
	}
	w.Flush()
	return []byte(body.String())
}

func TestCSVPreservesAllRowsAndExactAmounts(t *testing.T) {
	for _, delimiter := range []rune{',', ';'} {
		raw := append([]byte{0xef, 0xbb, 0xbf}, csvFixture(delimiter, 301)...)
		s, err := csvSnapshot(raw, "Финансовое здоровье", "2026-10-07T12:00:00+05:00")
		if err != nil {
			t.Fatal(err)
		}
		report, err := importreview.Review(s)
		if err != nil || len(report.Records) != 301 {
			t.Fatal("rows lost", err, len(report.Records))
		}
		last := report.Records[300]
		if last.Row != 305 || *last.Normalized.AmountMinor != 76275 || last.Normalized.Description != "Молоко, сыр; хлеб\nдве строки" {
			t.Fatal(last)
		}
		if last.Normalized.Date.UTC != "" || last.Normalized.Date.Precision != "date" {
			t.Fatal("invented time")
		}
	}
}

func TestCSVBlankRowsAndInvalidStructure(t *testing.T) {
	raw := strings.Replace(string(csvFixture(',', 2)), "\"\"\n", "\n", 1)
	s, err := csvSnapshot([]byte(raw), "source", "2026-10-07T12:00:00Z")
	if err != nil || s.Sheets[0].Ranges[0].Rows[4].Values[0].FormattedValue != "01.09.2026" {
		t.Fatal("blank row shifted source", err)
	}
	for _, raw := range [][]byte{[]byte("Дата,Тип,Сумма\n2026-09-01,Расход,10"), []byte("\"broken"), {0xff, 0xfe}, []byte(strings.Replace(string(csvFixture(',', 1)), "Сумма", "Неизвестная колонка", 1)), []byte(strings.Replace(string(csvFixture(',', 1)), "\n01.09.2026,", "\nextra,01.09.2026,", 1))} {
		if _, err := csvSnapshot(raw, "source", "2026-10-07T12:00:00Z"); err == nil {
			t.Fatal("invalid CSV accepted")
		}
	}
}
