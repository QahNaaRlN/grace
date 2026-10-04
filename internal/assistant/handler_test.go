package assistant

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

type fakeQueryRunner struct {
	calls  int
	query  Query
	result QueryResult
}

func (fake *fakeQueryRunner) Query(_ context.Context, query Query) QueryResult {
	fake.calls++
	fake.query = query
	return fake.result
}

func TestHandlerValidatesAndMapsStatus(t *testing.T) {
	runner := &fakeQueryRunner{result: QueryResult{Status: StatusUnavailable, ErrorCode: "assistant_unavailable"}}
	handler := NewHandler(runner)
	recorder := httptest.NewRecorder()
	handler.Query(recorder, httptest.NewRequest(http.MethodPost, "/api/v1/assistant/query", bytes.NewBufferString(`{"utterance":"Покажи счета","language":"ru","referenceTime":"2026-10-05T12:00:00+05:00"}`)))
	if recorder.Code != http.StatusServiceUnavailable || runner.calls != 1 || runner.query.Utterance != "Покажи счета" {
		t.Fatalf("unexpected response=%d calls=%d", recorder.Code, runner.calls)
	}
	var body map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil || body["errorCode"] != "assistant_unavailable" {
		t.Fatalf("unexpected body: %s", recorder.Body.String())
	}

	for _, payload := range []string{`{}`, `{"utterance":"x","language":"ru","referenceTime":"bad"}`, `{"utterance":"x","language":"ru","referenceTime":"2026-10-05T12:00:00Z","extra":true}`} {
		recorder = httptest.NewRecorder()
		handler.Query(recorder, httptest.NewRequest(http.MethodPost, "/api/v1/assistant/query", bytes.NewBufferString(payload)))
		if recorder.Code != http.StatusUnprocessableEntity {
			t.Fatalf("expected 422 for %s, got %d", payload, recorder.Code)
		}
	}
	if runner.calls != 1 {
		t.Fatal("invalid request reached router")
	}
}
