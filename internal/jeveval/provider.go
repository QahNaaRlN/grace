package jeveval

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const DefaultTypeSafeEndpoint = "https://api.typesafe.ai/v1/systemone"

const (
	ErrorAuthentication = "provider_authentication_error"
	ErrorRequest        = "provider_request_error"
	ErrorRateLimit      = "provider_rate_limit_error"
	ErrorOverload       = "provider_overload_error"
	ErrorProtocol       = "provider_protocol_error"
	ErrorTimeout        = "provider_timeout"
	ErrorTransport      = "provider_transport_error"
)

type ProviderError struct {
	Class  string
	Status int
	Err    error
}

func (err *ProviderError) Error() string {
	if err.Status != 0 {
		return fmt.Sprintf("%s: TypeSafe returned HTTP %d", err.Class, err.Status)
	}
	if err.Err != nil {
		return fmt.Sprintf("%s: %v", err.Class, err.Err)
	}
	return err.Class
}

func (err *ProviderError) Unwrap() error { return err.Err }

func ProviderErrorClass(err error) string {
	var providerErr *ProviderError
	if errors.As(err, &providerErr) {
		return providerErr.Class
	}
	return ErrorTransport
}

type httpDoer interface {
	Do(*http.Request) (*http.Response, error)
}

type TypeSafeProvider struct {
	apiKey      string
	endpoint    string
	httpClient  httpDoer
	maxAttempts int
	sleep       func(context.Context, time.Duration) error
}

func NewTypeSafeProvider(apiKey, endpoint string, httpClient *http.Client) (*TypeSafeProvider, error) {
	if strings.TrimSpace(apiKey) == "" {
		return nil, errors.New("TYPESAFE_API_KEY is required")
	}
	if endpoint == "" {
		endpoint = DefaultTypeSafeEndpoint
	}
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second}
	}
	return &TypeSafeProvider{
		apiKey: apiKey, endpoint: endpoint, httpClient: httpClient, maxAttempts: 3,
		sleep: sleepContext,
	}, nil
}

func (provider *TypeSafeProvider) Evaluate(ctx context.Context, request Request) (Response, error) {
	payload, err := json.Marshal(request)
	if err != nil {
		return Response{}, &ProviderError{Class: ErrorRequest, Err: errors.New("cannot encode request")}
	}

	for attempt := 1; attempt <= provider.maxAttempts; attempt++ {
		httpRequest, err := http.NewRequestWithContext(ctx, http.MethodPost, provider.endpoint, bytes.NewReader(payload))
		if err != nil {
			return Response{}, &ProviderError{Class: ErrorRequest, Err: errors.New("cannot create request")}
		}
		httpRequest.Header.Set("Authorization", "Bearer "+provider.apiKey)
		httpRequest.Header.Set("Content-Type", "application/json")

		httpResponse, err := provider.httpClient.Do(httpRequest)
		if err != nil {
			if errors.Is(ctx.Err(), context.DeadlineExceeded) || errors.Is(err, context.DeadlineExceeded) {
				return Response{}, &ProviderError{Class: ErrorTimeout, Err: errors.New("request deadline exceeded")}
			}
			if errors.Is(ctx.Err(), context.Canceled) || errors.Is(err, context.Canceled) {
				return Response{}, &ProviderError{Class: ErrorTimeout, Err: errors.New("request canceled")}
			}
			return Response{}, &ProviderError{Class: ErrorTransport, Err: errors.New("request failed")}
		}

		response, retryAfter, err := decodeHTTPResponse(httpResponse, request)
		if err == nil {
			return response, nil
		}
		var providerErr *ProviderError
		if !errors.As(err, &providerErr) || (providerErr.Status != http.StatusTooManyRequests && providerErr.Status != 529) || attempt == provider.maxAttempts {
			return Response{}, err
		}
		if retryAfter <= 0 {
			retryAfter = time.Duration(attempt) * 100 * time.Millisecond
		}
		if retryAfter > 2*time.Second {
			retryAfter = 2 * time.Second
		}
		if err := provider.sleep(ctx, retryAfter); err != nil {
			return Response{}, &ProviderError{Class: ErrorTimeout, Err: errors.New("retry canceled")}
		}
	}
	return Response{}, &ProviderError{Class: ErrorTransport, Err: errors.New("request attempts exhausted")}
}

func decodeHTTPResponse(httpResponse *http.Response, request Request) (Response, time.Duration, error) {
	defer httpResponse.Body.Close()
	if httpResponse.StatusCode != http.StatusOK {
		_, _ = io.Copy(io.Discard, io.LimitReader(httpResponse.Body, 64<<10))
		class := ErrorTransport
		switch httpResponse.StatusCode {
		case http.StatusUnauthorized:
			class = ErrorAuthentication
		case http.StatusUnprocessableEntity:
			class = ErrorRequest
		case http.StatusTooManyRequests:
			class = ErrorRateLimit
		case 529:
			class = ErrorOverload
		}
		return Response{}, parseRetryAfter(httpResponse.Header.Get("Retry-After")), &ProviderError{Class: class, Status: httpResponse.StatusCode}
	}

	var response Response
	decoder := json.NewDecoder(io.LimitReader(httpResponse.Body, 4<<20))
	if err := decoder.Decode(&response); err != nil {
		return Response{}, 0, &ProviderError{Class: ErrorProtocol, Err: errors.New("invalid JSON response")}
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return Response{}, 0, &ProviderError{Class: ErrorProtocol, Err: errors.New("response contains trailing JSON")}
	}
	if err := ValidateResponse(request, response); err != nil {
		return Response{}, 0, &ProviderError{Class: ErrorProtocol, Err: err}
	}
	return response, 0, nil
}

func ValidateResponse(request Request, response Response) error {
	if response.Model == "" {
		return errors.New("response model is required")
	}
	if len(response.Answers) != len(request.Questions) {
		return errors.New("response must contain exactly one answer per question")
	}
	for id, question := range request.Questions {
		answer, ok := response.Answers[id]
		if !ok {
			return fmt.Errorf("answer %q is missing", id)
		}
		if answer.Type != question.Type || answer.Type != "choice" {
			return fmt.Errorf("answer %q has unexpected type %q", id, answer.Type)
		}
		if _, ok := question.Criteria[answer.Choice]; !ok {
			return fmt.Errorf("answer %q selected unknown option %q", id, answer.Choice)
		}
		if answer.Confidence < 0 || answer.Confidence > 1 {
			return fmt.Errorf("answer %q confidence must be between 0 and 1", id)
		}
		if len(answer.Probabilities) != len(question.Criteria) {
			return fmt.Errorf("answer %q probabilities do not cover every option", id)
		}
		total := 0.0
		for option := range question.Criteria {
			probability, ok := answer.Probabilities[option]
			if !ok || probability < 0 || probability > 1 {
				return fmt.Errorf("answer %q has invalid probability for %q", id, option)
			}
			total += probability
		}
		if total < 0.99 || total > 1.01 {
			return fmt.Errorf("answer %q probabilities must sum to one", id)
		}
	}
	return nil
}

func parseRetryAfter(value string) time.Duration {
	if seconds, err := strconv.Atoi(strings.TrimSpace(value)); err == nil && seconds >= 0 {
		return time.Duration(seconds) * time.Second
	}
	if retryAt, err := http.ParseTime(value); err == nil {
		if duration := time.Until(retryAt); duration > 0 {
			return duration
		}
	}
	return 0
}

func sleepContext(ctx context.Context, duration time.Duration) error {
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
