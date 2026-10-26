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

	"github.com/QahNaaRlN/grace/internal/statementextract"
)

// ModelPolicy is an operator-verified endpoint and price snapshot. No automatic
// model aliases or fallback may silently change the experiment's cost/privacy.
type ModelPolicy struct {
	Selection              *ModelSelection `json:"selection,omitempty"`
	RequestFeeMicros       int64           `json:"requestFeeMicros,omitempty"`
	ImageFeeMicros         int64           `json:"imageFeeMicros,omitempty"`
	Model                  string          `json:"model"`
	Provider               string          `json:"provider"`
	ProviderName           string          `json:"providerName,omitempty"`
	InputMicrosPerMillion  int64           `json:"inputMicrosPerMillion"`
	OutputMicrosPerMillion int64           `json:"outputMicrosPerMillion"`
	ImageTokenUpperBound   int             `json:"imageTokenUpperBound"`
	VerifiedAt             time.Time       `json:"verifiedAt"`
	Vision                 bool            `json:"vision"`
}

func (p ModelPolicy) DisplayProvider() string {
	if p.ProviderName != "" {
		return p.ProviderName
	}
	return p.Provider
}

type routerMetadata struct {
	Requested string `json:"requested"`
	Strategy  string `json:"strategy"`
	Attempt   int    `json:"attempt"`
	IsBYOK    bool   `json:"is_byok"`
	Endpoints struct {
		Available []struct {
			Provider string `json:"provider"`
			Model    string `json:"model"`
			Selected bool   `json:"selected"`
		} `json:"available"`
	} `json:"endpoints"`
	Pipeline []json.RawMessage `json:"pipeline"`
}

func (p ModelPolicy) Valid(now time.Time) bool {
	return p.Model != "" && !strings.ContainsAny(p.Model, ": \r\n") && p.Provider != "" &&
		p.InputMicrosPerMillion > 0 && p.OutputMicrosPerMillion > 0 && p.RequestFeeMicros >= 0 && p.RequestFeeMicros <= 1000000 && p.ImageFeeMicros >= 0 && p.ImageFeeMicros <= 1000000 &&
		p.InputMicrosPerMillion <= 100_000_000 && p.OutputMicrosPerMillion <= 100_000_000 &&
		!p.VerifiedAt.IsZero() && !p.VerifiedAt.After(now) && now.Sub(p.VerifiedAt) <= 24*time.Hour &&
		(!p.Vision || p.ImageTokenUpperBound > 0 && p.ImageTokenUpperBound < 10000)
}

func (p ModelPolicy) ReservationMicros(inputBound, outputBound int) (int64, error) {
	maxInput, maxOutput := 10000, 2000
	if p.Selection != nil && documentProfile(p.Selection.Profile) {
		maxInput, maxOutput = 30000, 16000
	}
	if !p.Valid(time.Now()) || inputBound < 1 || inputBound > maxInput || outputBound < 1 || outputBound > maxOutput {
		return 0, errors.New("price_unknown")
	}
	// Prices are bounded above, so these products cannot overflow int64.
	n := p.InputMicrosPerMillion*int64(inputBound) + p.OutputMicrosPerMillion*int64(outputBound)
	images := 0
	if p.Selection != nil {
		images = p.Selection.MaxImages
	}
	return (n+999999)/1000000 + p.RequestFeeMicros + int64(images)*p.ImageFeeMicros, nil
}

type LLMMessage struct {
	Role    string `json:"role"`
	Content any    `json:"content"`
}
type LLMRequest struct {
	comparisonProof      *statementextract.ComparisonIdentityProof
	screeningProof       *statementextract.ScreeningIdentityProof
	DisableReasoning     bool
	Policy               ModelPolicy
	Messages             []LLMMessage
	Schema               json.RawMessage
	InputTokenUpperBound int
	MaxOutputTokens      int
}

// ProviderProtocolError exposes only a fixed diagnostic code, never input/output.
type ProviderProtocolError struct{ Reason string }

func (e *ProviderProtocolError) Error() string          { return "provider_protocol" }
func (e *ProviderProtocolError) ProtocolReason() string { return e.Reason }

type LLMResponse struct {
	NotSent             bool
	ComparisonRoute     *statementextract.ComparisonRouteEvidence `json:"-"`
	ScreeningDiagnostic *statementextract.ScreeningDiagnostic     `json:"-"`
	BillingKnown        bool
	ID                  string
	Model               string
	Provider            string
	Content             json.RawMessage
	PromptTokens        int
	CompletionTokens    int
	CostMicros          int64
}
type ReasoningProvider interface {
	Complete(context.Context, LLMRequest) (LLMResponse, error)
}
type OpenRouterProvider struct {
	apiKey            string
	endpoint          string
	client            HTTPDoer
	statementDocument bool
	screeningDocument bool
	groups            *ModelGroups
	readingDocument   bool
	comparisonID      string
}

func NewOpenRouterProvider(key string, client HTTPDoer) (*OpenRouterProvider, error) {
	if strings.TrimSpace(key) == "" || strings.ContainsAny(key, "\r\n") {
		return nil, errors.New("ai_disabled")
	}
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	}
	return &OpenRouterProvider{apiKey: key, endpoint: "https://openrouter.ai/api/v1/chat/completions", client: client}, nil
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

func (p *OpenRouterProvider) Complete(ctx context.Context, in LLMRequest) (result LLMResponse, resultErr error) {
	sent := false
	defer func() {
		if p.groups != nil && resultErr != nil && !sent {
			result.NotSent = true
			result.BillingKnown = true
			result.ID = "not_sent"
			result.CostMicros = 0
		}
	}()
	if in.Policy.Selection != nil && in.Policy.Selection.Profile == PlainDocumentReading && (p.groups == nil || len(in.Schema) != 0) {
		return LLMResponse{}, errors.New("provider_request")
	}
	if e := p.checkGroupRequest(in); e != nil {
		return LLMResponse{}, e
	}
	diagnostic := statementextract.ScreeningDiagnostic{Code: "request"}
	var comparisonRoute *statementextract.ComparisonRouteEvidence
	defer func() {
		if !p.screeningDocument && p.comparisonID == "" {
			return
		}
		diagnostic = diagnostic.Sanitize()
		result.ScreeningDiagnostic = &diagnostic
		result.ComparisonRoute = comparisonRoute
		if resultErr != nil {
			resultErr = &screeningDiagnosticError{cause: resultErr, detail: diagnostic}
		}
	}()
	if p.screeningDocument {
		observed := time.Now().UTC()
		diagnostic.Route = &statementextract.ScreeningRouteEvidence{Proof: in.screeningProof, ObservedAt: observed, SelectedModelClass: "not_observed"}
		pin := statementextract.ScreeningPolicy()
		if in.screeningProof == nil || !in.screeningProof.ValidAt(observed) || in.Policy.Model != pin.Model || in.Policy.Provider != pin.Provider || in.Policy.DisplayProvider() != pin.ProviderName || in.Policy.InputMicrosPerMillion != pin.InputMicrosPerMillion || in.Policy.OutputMicrosPerMillion != pin.OutputMicrosPerMillion {
			return LLMResponse{}, errors.New("provider_request")
		}
	}
	if p.comparisonID != "" {
		now := time.Now().UTC()
		c, ok := statementextract.ComparisonCandidateByID(p.comparisonID)
		if !ok || p.screeningDocument || in.comparisonProof == nil || !in.comparisonProof.ValidAt(now) || in.comparisonProof.Candidate != c || in.Policy.Model != c.Policy.Model || in.Policy.Provider != c.Policy.Provider || in.Policy.DisplayProvider() != c.Policy.ProviderName || in.Policy.InputMicrosPerMillion != c.Policy.InputMicrosPerMillion || in.Policy.OutputMicrosPerMillion != c.Policy.OutputMicrosPerMillion {
			return LLMResponse{}, errors.New("provider_request")
		}
		proof := *in.comparisonProof
		comparisonRoute = &statementextract.ComparisonRouteEvidence{Proof: &proof, ObservedAt: now, SelectedModelClass: "not_observed"}
	}
	if _, err := p.reservation(in.Policy, in.InputTokenUpperBound, in.MaxOutputTokens); err != nil {
		return LLMResponse{}, err
	}
	if len(in.Messages) == 0 || len(in.Messages) > 12 || !json.Valid(in.Schema) && !(in.Policy.Selection != nil && in.Policy.Selection.Profile == PlainDocumentReading && len(in.Schema) == 0) {
		return LLMResponse{}, errors.New("provider_request")
	}
	if p.statementDocument {
		if !in.DisableReasoning || len(in.Messages) != 2 {
			return LLMResponse{}, errors.New("provider_request")
		}
		for i, message := range in.Messages {
			role := []string{"system", "user"}[i]
			if _, ok := message.Content.(string); !ok || message.Role != role {
				return LLMResponse{}, errors.New("provider_request")
			}
		}
	}
	encodedMessages, err := json.Marshal(in.Messages)
	if err != nil || len(encodedMessages) > 12<<20 {
		return LLMResponse{}, errors.New("provider_request")
	}
	body := map[string]any{"model": in.Policy.Model, "messages": in.Messages, "max_tokens": in.MaxOutputTokens, "stream": false,
		"provider":        map[string]any{"only": []string{in.Policy.Provider}, "allow_fallbacks": false, "require_parameters": true, "data_collection": "deny", "zdr": true, "max_price": map[string]any{"prompt": microsPrice(in.Policy.InputMicrosPerMillion), "completion": microsPrice(in.Policy.OutputMicrosPerMillion)}},
		"response_format": map[string]any{"type": "json_schema", "json_schema": map[string]any{"name": "grace_response", "strict": true, "schema": in.Schema}}}
	if in.Policy.Selection != nil && in.Policy.Selection.Profile == PlainDocumentReading {
		delete(body, "response_format")
	}
	if in.DisableReasoning {
		body["reasoning"] = map[string]any{"enabled": false}
	}
	if in.Policy.Vision {
		// Image ceiling in dollars per image; reservation also includes every image token.
		body["provider"].(map[string]any)["max_price"].(map[string]any)["image"] = json.Number(big.NewRat(in.Policy.InputMicrosPerMillion*int64(in.Policy.ImageTokenUpperBound), 1_000_000_000_000).FloatString(12))
	}
	if p.groups != nil {
		caps := body["provider"].(map[string]any)["max_price"].(map[string]any)
		caps["request"] = json.Number(big.NewRat(in.Policy.RequestFeeMicros, 1000000).FloatString(6))
		caps["image"] = json.Number(big.NewRat(in.Policy.ImageFeeMicros, 1000000).FloatString(6))
	}
	payload, err := json.Marshal(body)
	if err != nil || (p.statementDocument && len(payload) > statementMaxEnvelopeBytes) {
		return LLMResponse{}, errors.New("provider_request")
	}
	timeout := 30 * time.Second
	if p.readingDocument {
		timeout = 180 * time.Second
	}
	if p.statementDocument {
		timeout = statementRequestTimeout()
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.endpoint, bytes.NewReader(payload))
	if err != nil {
		return LLMResponse{}, errors.New("provider_request")
	}
	req.Header.Set("Authorization", "Bearer "+p.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-OpenRouter-Metadata", "enabled")
	diagnostic.Code = "transport"
	sent = true
	response, err := p.client.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			diagnostic.Code = "timeout"
			return LLMResponse{}, errors.New("provider_timeout")
		}
		return LLMResponse{}, errors.New("provider_unavailable")
	}
	defer response.Body.Close()
	diagnostic.HTTPStatus = response.StatusCode
	diagnostic.ContentType = screeningContentType(response.Header.Get("Content-Type"))
	// Do not reflect provider error bodies (which may echo personal input) in logs/UI.
	if response.StatusCode != http.StatusOK {
		diagnostic.Code = "http_other"
		if response.StatusCode >= 400 && response.StatusCode < 500 {
			diagnostic.Code = "http_4xx"
		}
		if response.StatusCode >= 500 && response.StatusCode < 600 {
			diagnostic.Code = "http_5xx"
		}
		if response.StatusCode == http.StatusBadRequest || response.StatusCode == http.StatusUnprocessableEntity {
			return LLMResponse{}, errors.New("provider_request")
		}
		return LLMResponse{}, errors.New("provider_unavailable")
	}
	raw, err := io.ReadAll(io.LimitReader(response.Body, (1<<20)+1))
	if err != nil {
		diagnostic.Code = "read"
		return LLMResponse{}, errors.New("provider_protocol")
	}
	if len(raw) > 1<<20 {
		diagnostic.Code = "oversize"
		return LLMResponse{}, errors.New("provider_protocol")
	}
	if p.groups != nil && len(p.apiKey) >= 16 && bytes.Contains(raw, []byte(p.apiKey)) {
		return LLMResponse{}, errors.New("provider_protocol")
	}
	var out struct {
		ID       string          `json:"id"`
		Model    string          `json:"model"`
		Provider string          `json:"provider"`
		Metadata *routerMetadata `json:"openrouter_metadata"`
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
	diagnostic.Code = "envelope"
	decoded := json.Unmarshal(raw, &out) == nil
	if decoded {
		if diagnostic.Route != nil {
			diagnostic.Route.SelectedModelClass = selectedModelClass(out.Metadata)
		}
		diagnostic.CorrelationID = out.ID
		if comparisonRoute != nil {
			comparisonRoute.SelectedModelClass = comparisonSelectedClass(out.Metadata, in.comparisonProof)
		}
	}
	if !decoded || out.ID == "" || (out.Model != in.Policy.Model && !(p.groups != nil && in.Policy.Selection != nil && out.Model == in.Policy.Selection.Canonical) && !(p.comparisonID != "" && in.comparisonProof != nil && out.Model == in.comparisonProof.Candidate.Canonical)) {
		return LLMResponse{}, &ProviderProtocolError{Reason: "envelope"}
	}
	// Routing slugs and response display names are distinct API fields. The
	// expected name is verified by the operator, never inferred from a slug.
	if out.Metadata == nil {
		diagnostic.Code = "metadata_missing"
		return LLMResponse{}, errors.New("provider_protocol")
	}
	for _, guard := range []struct {
		bad  bool
		code string
	}{
		{out.Metadata.Requested != in.Policy.Model, "requested"},
		{out.Metadata.Strategy != "direct", "strategy"},
		{out.Metadata.Attempt != 1, "attempt"},
		{out.Metadata.IsBYOK, "byok"},
		{len(out.Metadata.Pipeline) > 0, "pipeline"},
	} {
		if guard.bad {
			diagnostic.Code = guard.code
			return LLMResponse{}, errors.New("provider_protocol")
		}
	}
	selected := 0
	actualProvider := ""
	for _, endpoint := range out.Metadata.Endpoints.Available {
		if endpoint.Selected {
			selected++
			if endpoint.Model != in.Policy.Model && !(p.groups != nil && in.Policy.Selection != nil && endpoint.Model == in.Policy.Selection.Canonical) && !(p.screeningDocument && in.screeningProof != nil && endpoint.Model == in.screeningProof.Canonical) && !(p.comparisonID != "" && in.comparisonProof != nil && endpoint.Model == in.comparisonProof.Candidate.Canonical) {
				diagnostic.Code = "selected_model"
				return LLMResponse{}, errors.New("provider_protocol")
			}
			if !strings.EqualFold(endpoint.Provider, in.Policy.DisplayProvider()) {
				diagnostic.Code = "provider"
				return LLMResponse{}, errors.New("provider_protocol")
			}
			actualProvider = endpoint.Provider
		}
	}
	if selected != 1 {
		diagnostic.Code = "count"
		return LLMResponse{}, errors.New("provider_protocol")
	}
	if out.Provider != "" && !strings.EqualFold(out.Provider, actualProvider) {
		diagnostic.Code = "display_provider"
		return LLMResponse{}, errors.New("provider_protocol")
	}
	diagnostic.Code = "cost"
	cost, err := costMicros(string(out.Usage.Cost))
	if err != nil {
		return LLMResponse{}, err
	}
	if out.Usage.Prompt < 1 || out.Usage.Completion < 0 {
		diagnostic.Code = "usage"
		return LLMResponse{}, &ProviderProtocolError{Reason: "usage"}
	}
	// Billing is gateway metadata, independent of whether the model produced
	// usable JSON. Retain only validated route/cost/ID on a content failure.
	billed := LLMResponse{BillingKnown: true, ID: out.ID, Model: out.Model, Provider: actualProvider, PromptTokens: out.Usage.Prompt, CompletionTokens: out.Usage.Completion, CostMicros: cost}
	if p.groups != nil {
		bound, _ := in.Policy.ReservationMicros(in.InputTokenUpperBound, in.MaxOutputTokens)
		if cost > bound {
			return billed, &ProviderProtocolError{Reason: "cost_bound"}
		}
	}
	if out.Usage.Completion == 0 {
		diagnostic.Code = "usage"
		return billed, &ProviderProtocolError{Reason: "usage"}
	}
	if out.Usage.Prompt > in.InputTokenUpperBound || out.Usage.Completion > in.MaxOutputTokens {
		diagnostic.Code = "token_bound"
		return billed, &ProviderProtocolError{Reason: "token_bound"}
	}
	if len(out.Choices) != 1 {
		diagnostic.Code = "choices"
		return billed, &ProviderProtocolError{Reason: "choices"}
	}
	if len(out.Choices[0].Message.ToolCalls) > 0 {
		diagnostic.Code = "tools"
		return billed, &ProviderProtocolError{Reason: "tools"}
	}
	if out.Choices[0].FinishReason != "stop" {
		reason := "finish_other"
		if out.Choices[0].FinishReason == "length" {
			reason = "finish_length"
		}
		diagnostic.Code = reason
		return billed, &ProviderProtocolError{Reason: reason}
	}
	if in.Policy.Selection != nil && in.Policy.Selection.Profile == PlainDocumentReading {
		if strings.TrimSpace(out.Choices[0].Message.Content) == "" {
			return billed, &ProviderProtocolError{Reason: "content"}
		}
		billed.Content = json.RawMessage(out.Choices[0].Message.Content)
		return billed, nil
	}
	if !json.Valid([]byte(out.Choices[0].Message.Content)) {
		diagnostic.Code = "content"
		return billed, &ProviderProtocolError{Reason: "content"}
	}
	diagnostic.Code = "unknown"
	billed.Content = json.RawMessage(out.Choices[0].Message.Content)
	return billed, nil
}

func microsPrice(n int64) json.Number {
	return json.Number(new(big.Rat).SetFrac64(n, 1000000).FloatString(6))
}
