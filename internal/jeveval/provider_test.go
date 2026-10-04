package jeveval

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestTypeSafeProviderSendsAuthenticatedRequestAndValidatesResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if got := request.Header.Get("Authorization"); got != "Bearer test-secret" {
			t.Errorf("Authorization = %q", got)
		}
		var providerRequest Request
		if err := json.NewDecoder(request.Body).Decode(&providerRequest); err != nil {
			t.Fatal(err)
		}
		writeProviderResponse(t, writer, providerRequest, map[string]string{
			questionTool: string(ToolGetAccounts), questionPeriod: string(PeriodNotApplicable),
			questionArguments: string(ArgumentsNotApplicable),
		})
	}))
	defer server.Close()

	provider, err := NewTypeSafeProvider("test-secret", server.URL, server.Client())
	if err != nil {
		t.Fatal(err)
	}
	request := requestFor(EvaluationCase{Utterance: "Покажи счета", Language: "ru"}, "jev-1.13.0")
	response, err := provider.Evaluate(context.Background(), request)
	if err != nil {
		t.Fatalf("Evaluate() error = %v", err)
	}
	if response.Model != "jev-1.13.0" {
		t.Fatalf("model = %q", response.Model)
	}
}

func TestTypeSafeProviderRetriesRateLimitWithoutLeakingSecret(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if calls.Add(1) == 1 {
			writer.Header().Set("Retry-After", "0")
			writer.WriteHeader(http.StatusTooManyRequests)
			_, _ = writer.Write([]byte(`{"error":"Bearer test-secret"}`))
			return
		}
		var providerRequest Request
		_ = json.NewDecoder(request.Body).Decode(&providerRequest)
		writeProviderResponse(t, writer, providerRequest, map[string]string{
			questionTool: string(ToolGetAccounts), questionPeriod: string(PeriodNotApplicable),
			questionArguments: string(ArgumentsNotApplicable),
		})
	}))
	defer server.Close()

	provider, err := NewTypeSafeProvider("test-secret", server.URL, server.Client())
	if err != nil {
		t.Fatal(err)
	}
	provider.sleep = func(context.Context, time.Duration) error { return nil }
	_, err = provider.Evaluate(context.Background(), requestFor(EvaluationCase{Utterance: "accounts", Language: "en"}, "jev-1.13.0"))
	if err != nil {
		t.Fatalf("Evaluate() error = %v", err)
	}
	if got := calls.Load(); got != 2 {
		t.Fatalf("calls = %d, want 2", got)
	}
}

func TestTypeSafeProviderClassifiesAndSanitizesErrors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusUnauthorized)
		_, _ = writer.Write([]byte(`{"error":"Bearer secret-value"}`))
	}))
	defer server.Close()
	provider, _ := NewTypeSafeProvider("secret-value", server.URL, server.Client())
	_, err := provider.Evaluate(context.Background(), requestFor(EvaluationCase{Utterance: "accounts", Language: "en"}, "jev-1.13.0"))
	if got := ProviderErrorClass(err); got != ErrorAuthentication {
		t.Fatalf("error class = %q", got)
	}
	if strings.Contains(err.Error(), "secret-value") {
		t.Fatalf("error leaked secret: %v", err)
	}
}

func TestNewTypeSafeProviderRequiresAPIKey(t *testing.T) {
	if _, err := NewTypeSafeProvider("", "", nil); err == nil {
		t.Fatal("expected missing API key error")
	}
}

func TestValidateResponseRejectsMissingAndInvalidAnswers(t *testing.T) {
	request := requestFor(EvaluationCase{Utterance: "accounts", Language: "en"}, "jev-1.13.0")
	response := validResponse(request, map[string]string{
		questionTool: string(ToolGetAccounts), questionPeriod: string(PeriodNotApplicable),
		questionArguments: string(ArgumentsNotApplicable),
	})
	delete(response.Answers, questionPeriod)
	if err := ValidateResponse(request, response); err == nil {
		t.Fatal("expected missing answer error")
	}

	response = validResponse(request, map[string]string{
		questionTool: string(ToolGetAccounts), questionPeriod: string(PeriodNotApplicable),
		questionArguments: string(ArgumentsNotApplicable),
	})
	response.Answers[questionTool] = ChoiceAnswer{Type: "choice", Choice: "invented", Probabilities: map[string]float64{"invented": 1}, Confidence: 1}
	if err := ValidateResponse(request, response); err == nil {
		t.Fatal("expected unknown choice error")
	}
}

func writeProviderResponse(t *testing.T, writer http.ResponseWriter, request Request, choices map[string]string) {
	t.Helper()
	writer.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(writer).Encode(validResponse(request, choices)); err != nil {
		t.Fatal(err)
	}
}

func validResponse(request Request, choices map[string]string) Response {
	answers := make(map[string]ChoiceAnswer, len(request.Questions))
	for id, question := range request.Questions {
		probabilities := make(map[string]float64, len(question.Criteria))
		choice := choices[id]
		for option := range question.Criteria {
			probabilities[option] = 0
		}
		probabilities[choice] = 1
		answers[id] = ChoiceAnswer{Type: "choice", Choice: choice, Probabilities: probabilities, Confidence: 0.9}
	}
	return Response{Model: "jev-1.13.0", Answers: answers, Usage: Usage{InputTokens: 100, OutputTokens: 20}}
}
