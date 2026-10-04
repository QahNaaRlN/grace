package analytics

import (
	"net/http/httptest"
	"testing"
)

func TestRequestPeriod(t *testing.T) {
	request := httptest.NewRequest("GET", "/?from=2026-10-01T00:00:00%2B05:00&to=2026-11-01T00:00:00%2B05:00", nil)
	if _, _, ok := requestPeriod(httptest.NewRecorder(), request); !ok {
		t.Fatal("expected valid explicit-offset period")
	}
	for _, query := range []string{
		"?from=2026-10-01T00:00:00&to=2026-11-01T00:00:00Z",
		"?from=2026-11-01T00:00:00Z&to=2026-10-01T00:00:00Z",
		"?from=2020-01-01T00:00:00Z&to=2026-01-02T00:00:00Z",
	} {
		if _, _, ok := requestPeriod(httptest.NewRecorder(), httptest.NewRequest("GET", "/"+query, nil)); ok {
			t.Fatalf("expected %s to be rejected", query)
		}
	}
}

func TestRequestCashHorizon(t *testing.T) {
	request := httptest.NewRequest("GET", "/?asOf=2026-10-04&through=2027-01-01", nil)
	if asOf, through, ok := requestCashHorizon(httptest.NewRecorder(), request); !ok || asOf != "2026-10-04" || through != "2027-01-01" {
		t.Fatalf("unexpected parsed horizon: %s %s %t", asOf, through, ok)
	}
	for _, query := range []string{
		"?asOf=2026-10-04",
		"?asOf=2026-10-04&through=2026-02-30",
		"?asOf=2026-10-04&through=2026-10-03",
		"?asOf=2020-01-01&through=2026-01-02",
	} {
		if _, _, ok := requestCashHorizon(httptest.NewRecorder(), httptest.NewRequest("GET", "/"+query, nil)); ok {
			t.Fatalf("expected %s to be rejected", query)
		}
	}
}
