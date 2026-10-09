package analytics

import (
	"math"
	"net/http/httptest"
	"testing"
)

func TestStatisticsCalendarBuckets(t *testing.T) {
	from, to, err := statisticsPeriod("2026-09-30", "2026-11-03", "month")
	if err != nil {
		t.Fatal(err)
	}
	buckets := statisticsBuckets(from, to, "month")
	if len(buckets) != 3 || buckets[0].From != "2026-09-30" || buckets[0].To != "2026-10-01" || buckets[2].To != "2026-11-03" {
		t.Fatalf("partial calendar buckets: %+v", buckets)
	}
	from, to, _ = statisticsPeriod("2026-10-04", "2026-10-13", "week")
	buckets = statisticsBuckets(from, to, "week")
	if len(buckets) != 3 || buckets[0].To != "2026-10-05" || buckets[1].To != "2026-10-12" {
		t.Fatalf("Monday weeks: %+v", buckets)
	}
	_, offset := from.Zone()
	if offset != 5*3600 {
		t.Fatal("calendar must use Yekaterinburg")
	}
	for _, input := range [][3]string{{"2026-02-30", "2026-03-01", "month"}, {"2026-10-01", "2026-10-01", "week"}, {"2025-01-01", "2026-01-02", "month"}, {"2026-01-01", "2026-02-01", "day"}} {
		if _, _, err = statisticsPeriod(input[0], input[1], input[2]); err == nil {
			t.Fatalf("accepted invalid %+v", input)
		}
	}
}
func TestStatisticsAmountSafety(t *testing.T) {
	if chartAmountSafe(9007199254740992) || !chartAmountSafe(9007199254740991, -9007199254740991) {
		t.Fatal("unsafe JavaScript money accepted")
	}
	if _, err := checkedOutflows(math.MaxInt64, 1); err == nil {
		t.Fatal("overflow accepted")
	}
	if sum, err := checkedOutflows(100, 30); err != nil || sum != 130 {
		t.Fatal(sum, err)
	}
	if categoryShare(math.MaxInt64, math.MaxInt64) != 10000 || categoryShare(1, 3) != 3333 || categoryShare(0, 0) != 0 {
		t.Fatal("invalid shares")
	}
}
func TestStatisticsRejectsInvalidBeforeRepository(t *testing.T) {
	handler := NewHandler(nil)
	response := httptest.NewRecorder()
	handler.Statistics(response, httptest.NewRequest("GET", "/?from=bad&to=2026-11-01", nil))
	if response.Code != 422 {
		t.Fatal(response.Code)
	}
}
