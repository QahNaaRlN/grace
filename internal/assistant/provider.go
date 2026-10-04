package assistant

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

const typeSafeEndpoint = "https://api.typesafe.ai/v1/systemone"

type ProviderErrorCode string

const (
	ProviderAuthentication ProviderErrorCode = "authentication"
	ProviderRequest        ProviderErrorCode = "request"
	ProviderRateLimit      ProviderErrorCode = "rate_limit"
	ProviderOverload       ProviderErrorCode = "overload"
	ProviderProtocol       ProviderErrorCode = "protocol"
	ProviderTimeout        ProviderErrorCode = "timeout"
	ProviderTransport      ProviderErrorCode = "transport"
)

type ProviderError struct {
	Code ProviderErrorCode
	Err  error
}

func (err *ProviderError) Error() string { return string(err.Code) }
func (err *ProviderError) Unwrap() error { return err.Err }

type providerState struct {
	Utterance     string `json:"utterance"`
	Language      string `json:"language"`
	ReferenceTime string `json:"reference_time"`
}

type providerRequest struct {
	State     providerState              `json:"state"`
	Model     string                     `json:"model"`
	Questions map[string]RoutingQuestion `json:"questions"`
}

type choiceAnswer struct {
	Type          string             `json:"type"`
	Choice        string             `json:"choice"`
	Probabilities map[string]float64 `json:"probabilities"`
	Confidence    float64            `json:"confidence"`
}

type providerResponse struct {
	Model   string                  `json:"model"`
	Answers map[string]choiceAnswer `json:"answers"`
	Usage   providerUsage           `json:"usage"`
}

type providerUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

type HTTPDoer interface {
	Do(*http.Request) (*http.Response, error)
}

type TypeSafeProvider struct {
	apiKey   string
	endpoint string
	client   HTTPDoer
	sleep    func(context.Context, time.Duration) error
}

func NewTypeSafeProvider(apiKey string, client HTTPDoer) (*TypeSafeProvider, error) {
	if strings.TrimSpace(apiKey) == "" {
		return nil, errors.New("TYPESAFE_API_KEY is required")
	}
	if client == nil {
		client = &http.Client{}
	}
	return &TypeSafeProvider{apiKey: apiKey, endpoint: typeSafeEndpoint, client: client, sleep: sleepWithContext}, nil
}

func (provider *TypeSafeProvider) Judge(ctx context.Context, query Query) (RoutingJudgment, error) {
	started := time.Now()
	request := providerRequest{
		State:     providerState{Utterance: query.Utterance, Language: query.Language, ReferenceTime: query.ReferenceTime.Format(time.RFC3339)},
		Model:     ProductionModel,
		Questions: RoutingQuestions(),
	}
	payload, err := json.Marshal(request)
	if err != nil {
		return RoutingJudgment{}, providerError(ProviderRequest, err)
	}
	for attempt := 1; attempt <= 2; attempt++ {
		httpRequest, err := http.NewRequestWithContext(ctx, http.MethodPost, provider.endpoint, bytes.NewReader(payload))
		if err != nil {
			return RoutingJudgment{}, providerError(ProviderRequest, err)
		}
		httpRequest.Header.Set("Authorization", "Bearer "+provider.apiKey)
		httpRequest.Header.Set("Content-Type", "application/json")
		httpResponse, err := provider.client.Do(httpRequest)
		if err != nil {
			if errors.Is(ctx.Err(), context.DeadlineExceeded) || errors.Is(err, context.DeadlineExceeded) {
				return RoutingJudgment{}, providerError(ProviderTimeout, err)
			}
			if errors.Is(ctx.Err(), context.Canceled) || errors.Is(err, context.Canceled) {
				return RoutingJudgment{}, providerError(ProviderTimeout, err)
			}
			return RoutingJudgment{}, providerError(ProviderTransport, err)
		}
		response, retry, err := decodeProviderResponse(httpResponse, request)
		if err == nil {
			return RoutingJudgment{
				RequestedModel: ProductionModel, ActualModel: response.Model,
				Route: RouteChoice(response.Answers[QuestionRoute].Choice), RouteProbabilities: response.Answers[QuestionRoute].Probabilities, RouteConfidence: response.Answers[QuestionRoute].Confidence,
				Arguments: ArgumentMode(response.Answers[QuestionArguments].Choice), ArgumentProbabilities: response.Answers[QuestionArguments].Probabilities, ArgumentConfidence: response.Answers[QuestionArguments].Confidence,
				InputTokens: response.Usage.InputTokens, OutputTokens: response.Usage.OutputTokens, Latency: time.Since(started),
			}, nil
		}
		var typed *ProviderError
		if !errors.As(err, &typed) || (typed.Code != ProviderRateLimit && typed.Code != ProviderOverload) || attempt == 2 {
			return RoutingJudgment{}, err
		}
		if retry <= 0 {
			retry = 100 * time.Millisecond
		}
		if retry > 2*time.Second {
			retry = 2 * time.Second
		}
		if err := provider.sleep(ctx, retry); err != nil {
			return RoutingJudgment{}, providerError(ProviderTimeout, err)
		}
	}
	return RoutingJudgment{}, providerError(ProviderTransport, errors.New("attempts exhausted"))
}

func decodeProviderResponse(httpResponse *http.Response, request providerRequest) (providerResponse, time.Duration, error) {
	defer httpResponse.Body.Close()
	if httpResponse.StatusCode != http.StatusOK {
		_, _ = io.Copy(io.Discard, io.LimitReader(httpResponse.Body, 64<<10))
		code := ProviderTransport
		switch httpResponse.StatusCode {
		case http.StatusUnauthorized:
			code = ProviderAuthentication
		case http.StatusUnprocessableEntity:
			code = ProviderRequest
		case http.StatusTooManyRequests:
			code = ProviderRateLimit
		case 529:
			code = ProviderOverload
		}
		return providerResponse{}, retryAfter(httpResponse.Header.Get("Retry-After")), providerError(code, fmt.Errorf("HTTP %d", httpResponse.StatusCode))
	}
	var response providerResponse
	const maximumResponseBytes = 4 << 20
	body, err := io.ReadAll(io.LimitReader(httpResponse.Body, maximumResponseBytes+1))
	if err != nil || len(body) > maximumResponseBytes {
		return providerResponse{}, 0, providerError(ProviderProtocol, errors.New("invalid response size"))
	}
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&response); err != nil {
		return providerResponse{}, 0, providerError(ProviderProtocol, errors.New("invalid response"))
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return providerResponse{}, 0, providerError(ProviderProtocol, errors.New("trailing response"))
	}
	if err := validateProviderResponse(request, response); err != nil {
		return providerResponse{}, 0, providerError(ProviderProtocol, err)
	}
	return response, 0, nil
}

func validateProviderResponse(request providerRequest, response providerResponse) error {
	if response.Model != request.Model {
		return errors.New("model mismatch")
	}
	if response.Usage.InputTokens < 0 || response.Usage.OutputTokens < 0 {
		return errors.New("invalid usage")
	}
	if len(response.Answers) != len(request.Questions) {
		return errors.New("unexpected answer count")
	}
	for id, question := range request.Questions {
		answer, ok := response.Answers[id]
		if !ok || answer.Type != "choice" || answer.Type != question.Type {
			return fmt.Errorf("invalid answer %s", id)
		}
		if _, ok := question.Criteria[answer.Choice]; !ok {
			return fmt.Errorf("unknown choice for %s", id)
		}
		if answer.Confidence < 0 || answer.Confidence > 1 {
			return fmt.Errorf("invalid confidence for %s", id)
		}
		if len(answer.Probabilities) != len(question.Criteria) {
			return fmt.Errorf("incomplete probabilities for %s", id)
		}
		total := 0.0
		for option := range question.Criteria {
			probability, ok := answer.Probabilities[option]
			if !ok || probability < 0 || probability > 1 {
				return fmt.Errorf("invalid probability for %s", id)
			}
			total += probability
		}
		if total < .99 || total > 1.01 {
			return fmt.Errorf("probabilities do not sum to one for %s", id)
		}
	}
	return nil
}

func providerError(code ProviderErrorCode, err error) error {
	return &ProviderError{Code: code, Err: err}
}

func retryAfter(value string) time.Duration {
	if seconds, err := strconv.Atoi(strings.TrimSpace(value)); err == nil && seconds >= 0 {
		return time.Duration(seconds) * time.Second
	}
	if at, err := http.ParseTime(value); err == nil {
		if wait := time.Until(at); wait > 0 {
			return wait
		}
	}
	return 0
}

func sleepWithContext(ctx context.Context, duration time.Duration) error {
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
