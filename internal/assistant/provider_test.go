package assistant

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func validProviderResponse(route RouteChoice, mode ArgumentMode) providerResponse {
	answers := map[string]choiceAnswer{}
	for id, question := range RoutingQuestions() {
		choice := string(route)
		if id == QuestionArguments {
			choice = string(mode)
		}
		probabilities := make(map[string]float64, len(question.Criteria))
		for option := range question.Criteria {
			probabilities[option] = 0
		}
		probabilities[choice] = 1
		answers[id] = choiceAnswer{Type: "choice", Choice: choice, Probabilities: probabilities, Confidence: .91}
	}
	return providerResponse{Model: ProductionModel, Answers: answers, Usage: providerUsage{InputTokens: 10, OutputTokens: 2}}
}

func TestTypeSafeProviderSendsMinimalStateAndRetriesOnce(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		attempts++
		if request.Header.Get("Authorization") != "Bearer test-key" {
			t.Error("missing bearer token")
		}
		var payload map[string]any
		if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
			t.Fatal(err)
		}
		state := payload["state"].(map[string]any)
		if len(state) != 3 || state["utterance"] != "тест" || state["language"] != "ru" {
			t.Fatalf("unexpected state: %#v", state)
		}
		if attempts == 1 {
			writer.WriteHeader(http.StatusTooManyRequests)
			return
		}
		_ = json.NewEncoder(writer).Encode(validProviderResponse(RouteGetAccounts, ArgumentsNone))
	}))
	defer server.Close()
	provider, _ := NewTypeSafeProvider("test-key", server.Client())
	provider.endpoint = server.URL
	provider.sleep = func(context.Context, time.Duration) error { return nil }
	judgment, err := provider.Judge(context.Background(), testQuery())
	if err != nil || attempts != 2 || judgment.Route != RouteGetAccounts || judgment.Arguments != ArgumentsNone {
		t.Fatalf("unexpected judgment=%#v attempts=%d err=%v", judgment, attempts, err)
	}
	if judgment.RequestedModel != ProductionModel || judgment.ActualModel != ProductionModel || judgment.InputTokens != 10 || judgment.OutputTokens != 2 || judgment.Latency < 0 {
		t.Fatalf("missing judgment metadata: %#v", judgment)
	}
}

func TestTypeSafeProviderRejectsUnsafeResponsesAndStatuses(t *testing.T) {
	tests := []struct {
		name     string
		status   int
		mutate   func(*providerResponse)
		expected ProviderErrorCode
	}{
		{name: "auth", status: http.StatusUnauthorized, expected: ProviderAuthentication},
		{name: "request", status: http.StatusUnprocessableEntity, expected: ProviderRequest},
		{name: "rate limit", status: http.StatusTooManyRequests, expected: ProviderRateLimit},
		{name: "overload", status: 529, expected: ProviderOverload},
		{name: "model mismatch", status: http.StatusOK, mutate: func(response *providerResponse) { response.Model = "jev-1.12.0" }, expected: ProviderProtocol},
		{name: "unknown choice", status: http.StatusOK, mutate: func(response *providerResponse) {
			answer := response.Answers[QuestionRoute]
			answer.Choice = "write_everything"
			response.Answers[QuestionRoute] = answer
		}, expected: ProviderProtocol},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
				writer.WriteHeader(test.status)
				if test.status == http.StatusOK {
					response := validProviderResponse(RouteGetAccounts, ArgumentsNone)
					if test.mutate != nil {
						test.mutate(&response)
					}
					_ = json.NewEncoder(writer).Encode(response)
				}
			}))
			defer server.Close()
			provider, _ := NewTypeSafeProvider("key", server.Client())
			provider.endpoint = server.URL
			provider.sleep = func(context.Context, time.Duration) error { return nil }
			_, err := provider.Judge(context.Background(), testQuery())
			var typed *ProviderError
			if !errors.As(err, &typed) || typed.Code != test.expected {
				t.Fatalf("expected %s, got %v", test.expected, err)
			}
		})
	}
}

type doerFunc func(*http.Request) (*http.Response, error)

func (function doerFunc) Do(request *http.Request) (*http.Response, error) { return function(request) }

func TestTypeSafeProviderHonorsCancellation(t *testing.T) {
	provider, _ := NewTypeSafeProvider("key", doerFunc(func(request *http.Request) (*http.Response, error) {
		<-request.Context().Done()
		return nil, request.Context().Err()
	}))
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := provider.Judge(ctx, testQuery())
	var typed *ProviderError
	if !errors.As(err, &typed) || typed.Code != ProviderTimeout {
		t.Fatalf("expected timeout-safe cancellation, got %v", err)
	}
}

func TestTypeSafeProviderClassifiesTimeoutAndTransport(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected ProviderErrorCode
	}{
		{name: "timeout", err: context.DeadlineExceeded, expected: ProviderTimeout},
		{name: "transport", err: errors.New("network details"), expected: ProviderTransport},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			provider, _ := NewTypeSafeProvider("key", doerFunc(func(*http.Request) (*http.Response, error) { return nil, test.err }))
			_, err := provider.Judge(context.Background(), testQuery())
			var typed *ProviderError
			if !errors.As(err, &typed) || typed.Code != test.expected || err.Error() != string(test.expected) {
				t.Fatalf("expected safe %s, got %v", test.expected, err)
			}
		})
	}
}
