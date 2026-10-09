package assistant

import (
	"encoding/json"
	"math"
	"strings"
	"testing"
	"time"

	"github.com/QahNaaRlN/grace/internal/analytics"
)

func TestAnalysisUsesExactCoreFactsAndCalendarPeriod(t *testing.T) {
	from, to, err := CalendarMonth("2026-09")
	if err != nil || from.Format(time.RFC3339) != "2026-09-01T00:00:00+05:00" || to.Format(time.RFC3339) != "2026-10-01T00:00:00+05:00" {
		t.Fatal(from, to, err)
	}
	facts := BuildAnalysisFacts("2026-09", analytics.PeriodSummary{From: from, To: to, Currencies: []analytics.CurrencyPeriodSummary{{Currency: "RUB", PurchasesMinor: math.MaxInt64, ExpenseCategories: []analytics.ExpenseCategorySummary{{CategoryName: "Synthetic food", PurchasesMinor: math.MaxInt64, NetExpenseMinor: math.MaxInt64}}}}}, time.Now())
	if facts.Facts[0].ValueMinor != "9223372036854775807" || *facts.Facts[5].ShareBasisPoints != 10000 || facts.Coverage != "confirmed_only_coverage_unknown" {
		t.Fatal(facts)
	}
	if _, err = AnalysisRequest(policyFixture(), "Synthetic monthly analysis", facts); err != nil {
		t.Fatal(err)
	}
}

func TestAnalysisResponseRejectsProseWithoutLosingFactsOrBilling(t *testing.T) {
	facts := AnalysisFacts{Coverage: "confirmed_only_coverage_unknown", Facts: []AnalysisFact{{ID: "known", ValueMinor: "12345", Currency: "RUB"}}}
	for _, raw := range []string{
		`{"interpretations":[{"kind":"assumption","text":"Расходы выросли на 50%","factIds":["known"]}]}`,
		`{"interpretations":[{"kind":"suggestion","text":"Проверьте данные","factIds":["invented"]}]}`,
		`{"interpretations":[{"kind":"suggestion","text":"Первая строка\nВторая строка","factIds":["known"]}]}`,
	} {
		answer := AnalysisResponse(LLMResponse{Model: "fixture", Provider: "fixture", CostMicros: 158, Content: json.RawMessage(raw)}, facts)
		if answer.Interpretations == nil || len(answer.Interpretations) != 0 || answer.Facts.Facts[0].ValueMinor != "12345" || answer.CostMicros != "158" || !strings.Contains(answer.Notice, "не прошло проверку") || !strings.Contains(answer.Notice, "Полнота") {
			t.Fatal(answer)
		}
	}
	answer := AnalysisResponse(LLMResponse{Content: json.RawMessage(`{"interpretations":[{"kind":"suggestion","text":"Проверьте полноту истории","factIds":["known"]}]}`)}, facts)
	if len(answer.Interpretations) != 1 || strings.Contains(answer.Notice, "не прошло проверку") {
		t.Fatal(answer)
	}
}
func TestAnalysisRejectsInventedNumbersReferencesAndAdditionalFields(t *testing.T) {
	facts := AnalysisFacts{Facts: []AnalysisFact{{ID: "known", ValueMinor: "12345", Currency: "RUB"}}}
	for _, raw := range []string{
		`{"interpretations":[{"kind":"assumption","text":"Расходы выросли на 50%","factIds":["known"]}]}`,
		`{"interpretations":[{"kind":"suggestion","text":"Проверь категорию","factIds":["invented"]}]}`,
		`{"interpretations":[{"kind":"fact","text":"Итог вычислен моделью","factIds":["known"]}]}`,
		`{"interpretations":[],"tool":"pay"}`,
		`{"interpretations":[]} {"interpretations":[]}`,
	} {
		if _, err := ValidateInterpretations(json.RawMessage(raw), facts); err == nil {
			t.Fatal("unsupported claim accepted", raw)
		}
	}
	raw := json.RawMessage(`{"interpretations":[{"kind":"suggestion","text":"Проверь, какие покупки входят в эту категорию.","factIds":["known"]}]}`)
	if _, err := ValidateInterpretations(raw, facts); err != nil {
		t.Fatal(err)
	}
	for _, month := range []string{"2026-9", "0000-01", "2026-13", "2026-09-01"} {
		if _, _, err := CalendarMonth(month); err == nil {
			t.Fatal(month)
		}
	}
}
