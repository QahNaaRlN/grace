package assistant

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

const (
	ProductionModel     = "jev-1.13.0"
	QuestionSetVersion  = "v2"
	ConfidenceThreshold = 0.70
)

type RouteChoice string

const (
	RouteGetAccounts          RouteChoice = "get_accounts"
	RouteGetPeriodSummary     RouteChoice = "get_period_summary"
	RouteGetCashPosition      RouteChoice = "get_cash_position"
	RouteGetObligations       RouteChoice = "get_obligations"
	RouteGetReceipts          RouteChoice = "get_receipts"
	RouteGetPeriodSinceIncome RouteChoice = "get_period_since_income"
	RouteClarify              RouteChoice = "clarify"
	RouteUnsupported          RouteChoice = "unsupported"
)

type ArgumentMode string

const (
	ArgumentsNone            ArgumentMode = "none"
	ArgumentsReferenceTime   ArgumentMode = "reference_time"
	ArgumentsDefaults        ArgumentMode = "defaults"
	ArgumentsNeedsResolution ArgumentMode = "needs_resolution"
	ArgumentsMissing         ArgumentMode = "missing"
	ArgumentsNotApplicable   ArgumentMode = "not_applicable"
)

type Query struct {
	Utterance     string    `json:"utterance"`
	Language      string    `json:"language"`
	ReferenceTime time.Time `json:"-"`
}

type RoutingJudgment struct {
	RequestedModel        string
	ActualModel           string
	Route                 RouteChoice
	RouteProbabilities    map[string]float64
	RouteConfidence       float64
	Arguments             ArgumentMode
	ArgumentProbabilities map[string]float64
	ArgumentConfidence    float64
	InputTokens           int
	OutputTokens          int
	Latency               time.Duration
}

type RoutingProvider interface {
	Judge(context.Context, Query) (RoutingJudgment, error)
}

type ToolCaller interface {
	Call(context.Context, string, json.RawMessage) (ToolResult, error)
}

type QueryStatus string

const (
	StatusCompleted   QueryStatus = "completed"
	StatusClarify     QueryStatus = "clarify"
	StatusUnsupported QueryStatus = "unsupported"
	StatusUnavailable QueryStatus = "unavailable"
)

type QueryResult struct {
	Status            QueryStatus `json:"status"`
	Route             RouteChoice `json:"route,omitempty"`
	Confidence        *float64    `json:"confidence,omitempty"`
	ClarificationCode string      `json:"clarificationCode,omitempty"`
	RequiredArguments []string    `json:"requiredArguments,omitempty"`
	Data              *ToolResult `json:"data,omitempty"`
	ErrorCode         string      `json:"errorCode,omitempty"`
}

type Router struct {
	provider RoutingProvider
	executor ToolCaller
	timeout  time.Duration
}

func NewRouter(provider RoutingProvider, executor ToolCaller) *Router {
	return &Router{provider: provider, executor: executor, timeout: 5 * time.Second}
}

func (router *Router) Query(ctx context.Context, query Query) QueryResult {
	if router == nil || router.provider == nil || router.executor == nil {
		return unavailable("assistant_unavailable")
	}
	ctx, cancel := context.WithTimeout(ctx, router.timeout)
	defer cancel()
	judgment, err := router.provider.Judge(ctx, query)
	if err != nil {
		return unavailable("assistant_unavailable")
	}
	if err := validateJudgment(judgment); err != nil {
		return unavailable("assistant_unavailable")
	}

	effective := judgment.RouteConfidence
	if judgment.Route != RouteClarify && judgment.Route != RouteUnsupported {
		effective = min(effective, judgment.ArgumentConfidence)
	}
	if effective < ConfidenceThreshold {
		return clarify(judgment.Route, effective, "low_confidence", nil)
	}
	if judgment.Route == RouteClarify {
		return clarify(judgment.Route, effective, "clarification_required", nil)
	}
	if judgment.Route == RouteUnsupported {
		return QueryResult{Status: StatusUnsupported, Route: judgment.Route, Confidence: number(effective)}
	}

	arguments, code, required, ok := compose(query, judgment)
	if !ok {
		return clarify(judgment.Route, effective, code, required)
	}
	result, err := router.executor.Call(ctx, string(judgment.Route), arguments)
	if err != nil {
		if toolCode, ok := ErrorCodeOf(err); ok && toolCode == CodeAnchorNotFound {
			return clarify(judgment.Route, effective, "no_income_anchor", nil)
		}
		return unavailable("dependency_unavailable")
	}
	return QueryResult{Status: StatusCompleted, Route: judgment.Route, Confidence: number(effective), Data: &result}
}

func compose(query Query, judgment RoutingJudgment) (json.RawMessage, string, []string, bool) {
	if judgment.Arguments == ArgumentsNeedsResolution || judgment.Arguments == ArgumentsMissing {
		if judgment.Route == RouteGetPeriodSinceIncome && judgment.Arguments == ArgumentsNeedsResolution {
			return nil, "income_category_required", []string{"categoryId"}, false
		}
		return nil, "arguments_required", requiredFor(judgment.Route), false
	}
	switch judgment.Route {
	case RouteGetAccounts:
		if judgment.Arguments == ArgumentsNone {
			return json.RawMessage(`{}`), "", nil, true
		}
	case RouteGetObligations:
		if judgment.Arguments == ArgumentsReferenceTime {
			return mustJSON(map[string]string{"asOf": query.ReferenceTime.Format("2006-01-02")}), "", nil, true
		}
	case RouteGetReceipts:
		if judgment.Arguments == ArgumentsDefaults {
			return json.RawMessage(`{}`), "", nil, true
		}
	case RouteGetPeriodSinceIncome:
		if judgment.Arguments == ArgumentsReferenceTime {
			return mustJSON(map[string]string{"asOf": query.ReferenceTime.Format(time.RFC3339)}), "", nil, true
		}
	case RouteGetPeriodSummary, RouteGetCashPosition:
		return nil, "arguments_required", requiredFor(judgment.Route), false
	}
	return nil, "composition_mismatch", nil, false
}

func requiredFor(route RouteChoice) []string {
	switch route {
	case RouteGetPeriodSummary:
		return []string{"from", "to"}
	case RouteGetCashPosition:
		return []string{"through"}
	case RouteGetPeriodSinceIncome:
		return []string{"categoryId"}
	case RouteGetObligations:
		return []string{"asOf"}
	case RouteGetReceipts:
		return []string{"limit"}
	default:
		return nil
	}
}

func ValidateQuery(utterance, language, referenceTime string) (Query, error) {
	utterance = strings.TrimSpace(utterance)
	if len(utterance) == 0 || len([]byte(utterance)) > 2000 {
		return Query{}, errors.New("utterance must contain 1 to 2000 UTF-8 bytes")
	}
	if language != "ru" && language != "en" {
		return Query{}, errors.New("language must be ru or en")
	}
	parsed, err := time.Parse(time.RFC3339, referenceTime)
	if err != nil || !hasExplicitOffset(referenceTime) {
		return Query{}, errors.New("referenceTime must be RFC 3339 with an explicit UTC offset")
	}
	return Query{Utterance: utterance, Language: language, ReferenceTime: parsed}, nil
}

func hasExplicitOffset(value string) bool {
	if strings.HasSuffix(value, "Z") {
		return true
	}
	t := strings.LastIndex(value, "T")
	return t >= 0 && (strings.LastIndex(value, "+") > t || strings.LastIndex(value, "-") > t)
}

func validateJudgment(j RoutingJudgment) error {
	validRoute := map[RouteChoice]bool{RouteGetAccounts: true, RouteGetPeriodSummary: true, RouteGetCashPosition: true, RouteGetObligations: true, RouteGetReceipts: true, RouteGetPeriodSinceIncome: true, RouteClarify: true, RouteUnsupported: true}
	validArguments := map[ArgumentMode]bool{ArgumentsNone: true, ArgumentsReferenceTime: true, ArgumentsDefaults: true, ArgumentsNeedsResolution: true, ArgumentsMissing: true, ArgumentsNotApplicable: true}
	if !validRoute[j.Route] || !validArguments[j.Arguments] || j.RouteConfidence < 0 || j.RouteConfidence > 1 || j.ArgumentConfidence < 0 || j.ArgumentConfidence > 1 {
		return fmt.Errorf("invalid routing judgment")
	}
	return nil
}

func clarify(route RouteChoice, confidence float64, code string, required []string) QueryResult {
	return QueryResult{Status: StatusClarify, Route: route, Confidence: number(confidence), ClarificationCode: code, RequiredArguments: required}
}

func unavailable(code string) QueryResult {
	return QueryResult{Status: StatusUnavailable, ErrorCode: code}
}
func number(value float64) *float64 { return &value }
func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
func mustJSON(value any) json.RawMessage { data, _ := json.Marshal(value); return data }
