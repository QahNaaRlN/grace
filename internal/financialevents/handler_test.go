package financialevents

import "testing"

func TestHashRequestIsStable(t *testing.T) {
	request := transferRequest{
		SourceAccountID:      "source",
		DestinationAccountID: "destination",
		AmountMinor:          5000,
		Description:          "transfer",
		OccurredAt:           "2026-10-04T12:00:00Z",
	}
	first := hashRequest(request)
	second := hashRequest(request)
	if first != second {
		t.Fatalf("hashRequest() is not stable: %q != %q", first, second)
	}
	request.AmountMinor++
	if hashRequest(request) == first {
		t.Fatal("hashRequest() did not change with the payload")
	}
}
