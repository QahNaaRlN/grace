package assistant

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"math/big"
	"net/http"
	"regexp"
	"strings"
	"time"
)

// ModelPolicy is an operator-verified endpoint and price snapshot. No automatic
// model aliases or fallback may silently change the experiment's cost/privacy.
type ModelPolicy struct {
	Model                  string    `json:"model"`
	Provider               string    `json:"provider"`
	InputMicrosPerMillion  int64     `json:"inputMicrosPerMillion"`
	OutputMicrosPerMillion int64     `json:"outputMicrosPerMillion"`
	ImageTokenUpperBound   int       `json:"imageTokenUpperBound"`
	VerifiedAt             time.Time `json:"verifiedAt"`
	Vision                 bool      `json:"vision"`
}

func (p ModelPolicy) Valid(now time.Time) bool {
	return p.Model != "" && !strings.ContainsAny(p.Model, ": \r\n") && p.Provider != "" &&
		p.InputMicrosPerMillion > 0 && p.OutputMicrosPerMillion > 0 &&
		p.InputMicrosPerMillion <= 100_000_000 && p.OutputMicrosPerMillion <= 100_000_000 &&
		!p.VerifiedAt.IsZero() && !p.VerifiedAt.After(now) && now.Sub(p.VerifiedAt) <= 24*time.Hour &&
		(!p.Vision || p.ImageTokenUpperBound > 0 && p.ImageTokenUpperBound < 10000)
}

func (p ModelPolicy) ReservationMicros(inputBound, outputBound int) (int64, error) {
	if !p.Valid(time.Now()) || inputBound < 1 || inputBound > 10000 || outputBound < 1 || outputBound > 2000 {
		return 0, errors.New("price_unknown")
	}
	// Prices are bounded above, so these products cannot overflow int64.
	n := p.InputMicrosPerMillion*int64(inputBound) + p.OutputMicrosPerMillion*int64(outputBound)
	return (n + 999999) / 1000000, nil
}

type LLMMessage struct {
	Role    string `json:"role"`
	Content any    `json:"content"`
}
type LLMRequest struct {
	Policy               ModelPolicy
	Messages             []LLMMessage
	Schema               json.RawMessage
	InputTokenUpperBound int
	MaxOutputTokens      int
}
type LLMResponse struct {
	ID               string
	Model            string
	Provider         string
	Content          json.RawMessage
	PromptTokens     int
	CompletionTokens int
	CostMicros       int64
}
type ReasoningProvider interface {
	Complete(context.Context, LLMRequest) (LLMResponse, error)
}
type OpenRouterProvider struct {
	apiKey   string
	endpoint string
	client   HTTPDoer
}

func NewOpenRouterProvider(key string, client HTTPDoer) (*OpenRouterProvider, error) {
	if strings.TrimSpace(key) == "" || strings.ContainsAny(key, "\r\n") {
		return nil, errors.New("ai_disabled")
	}
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	}
	return &OpenRouterProvider{key, "https://openrouter.ai/api/v1/chat/completions", client}, nil
}

var decimalCost = regexp.MustCompile(`^(?:0|[1-9][0-9]*)(?:\.[0-9]+)?(?:[eE][+-]?[0-9]{1,2})?$`)

func costMicros(value string) (int64, error) {
	if len(value) > 64 || !decimalCost.MatchString(value) {
		return 0, errors.New("price_unknown")
	}
	r, ok := new(big.Rat).SetString(value)
	if !ok || r.Sign() < 0 {
		return 0, errors.New("price_unknown")
	}
	r.Mul(r, big.NewRat(1000000, 1))
	n, rem := new(big.Int), new(big.Int)
	n.QuoRem(r.Num(), r.Denom(), rem)
	if rem.Sign() > 0 {
		n.Add(n, big.NewInt(1))
	}
	if !n.IsInt64() {
		return 0, errors.New("price_unknown")
	}
	return n.Int64(), nil
}

func (p *OpenRouterProvider) Complete(ctx context.Context, in LLMRequest) (LLMResponse, error) {
	if _, err := in.Policy.ReservationMicros(in.InputTokenUpperBound, in.MaxOutputTokens); err != nil {
		return LLMResponse{}, err
	}
	if len(in.Messages) == 0 || len(in.Messages) > 12 || !json.Valid(in.Schema) {
		return LLMResponse{}, errors.New("provider_request")
	}
	encodedMessages, err := json.Marshal(in.Messages)
	if err != nil || len(encodedMessages) > 12<<20 {
		return LLMResponse{}, errors.New("provider_request")
	}
	body := map[string]any{"model": in.Policy.Model, "messages": in.Messages, "max_tokens": in.MaxOutputTokens, "stream": false,
		"provider":        map[string]any{"only": []string{in.Policy.Provider}, "allow_fallbacks": false, "require_parameters": true, "data_collection": "deny", "zdr": true},
		"response_format": map[string]any{"type": "json_schema", "json_schema": map[string]any{"name": "grace_response", "strict": true, "schema": in.Schema}}}
	payload, err := json.Marshal(body)
	if err != nil {
		return LLMResponse{}, errors.New("provider_request")
	}
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.endpoint, bytes.NewReader(payload))
	if err != nil {
		return LLMResponse{}, errors.New("provider_request")
	}
	req.Header.Set("Authorization", "Bearer "+p.apiKey)
	req.Header.Set("Content-Type", "application/json")
	response, err := p.client.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return LLMResponse{}, errors.New("provider_timeout")
		}
		return LLMResponse{}, errors.New("provider_unavailable")
	}
	defer response.Body.Close()
	// Do not reflect provider error bodies (which may echo personal input) in logs/UI.
	if response.StatusCode != http.StatusOK {
		return LLMResponse{}, errors.New("provider_unavailable")
	}
	raw, err := io.ReadAll(io.LimitReader(response.Body, (1<<20)+1))
	if err != nil || len(raw) > 1<<20 {
		return LLMResponse{}, errors.New("provider_protocol")
	}
	var out struct {
		ID       string `json:"id"`
		Model    string `json:"model"`
		Provider string `json:"provider"`
		Choices  []struct {
			FinishReason string `json:"finish_reason"`
			Message      struct {
				Content   string            `json:"content"`
				ToolCalls []json.RawMessage `json:"tool_calls"`
			} `json:"message"`
		} `json:"choices"`
		Usage struct {
			Prompt     int         `json:"prompt_tokens"`
			Completion int         `json:"completion_tokens"`
			Cost       json.Number `json:"cost"`
		} `json:"usage"`
	}
	if json.Unmarshal(raw, &out) != nil || out.ID == "" || out.Model != in.Policy.Model || !strings.EqualFold(out.Provider, in.Policy.Provider) ||
		len(out.Choices) != 1 || len(out.Choices[0].Message.ToolCalls) > 0 || out.Choices[0].FinishReason != "stop" || !json.Valid([]byte(out.Choices[0].Message.Content)) ||
		out.Usage.Prompt < 1 || out.Usage.Prompt > in.InputTokenUpperBound || out.Usage.Completion < 1 || out.Usage.Completion > in.MaxOutputTokens {
		return LLMResponse{}, errors.New("provider_protocol")
	}
	cost, err := costMicros(string(out.Usage.Cost))
	if err != nil {
		return LLMResponse{}, err
	}
	return LLMResponse{out.ID, out.Model, out.Provider, json.RawMessage(out.Choices[0].Message.Content), out.Usage.Prompt, out.Usage.Completion, cost}, nil
}
