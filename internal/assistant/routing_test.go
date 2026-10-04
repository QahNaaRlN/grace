package assistant

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"testing"
	"time"
)

type fakeRoutingProvider struct {
	judgment RoutingJudgment
	err      error
	calls    int
}

func (fake *fakeRoutingProvider) Judge(_ context.Context, _ Query) (RoutingJudgment, error) {
	fake.calls++
	return fake.judgment, fake.err
}

type fakeToolCaller struct {
	calls     int
	name      string
	arguments json.RawMessage
	result    ToolResult
	err       error
}

func (fake *fakeToolCaller) Call(_ context.Context, name string, arguments json.RawMessage) (ToolResult, error) {
	fake.calls++
	fake.name = name
	fake.arguments = append(json.RawMessage(nil), arguments...)
	return fake.result, fake.err
}

func testQuery() Query {
	return Query{Utterance: "тест", Language: "ru", ReferenceTime: time.Date(2026, 10, 5, 12, 30, 0, 0, time.FixedZone("YEKT", 5*60*60))}
}

func TestRouterExecutesOnlyAcceptedCompatibilityPairs(t *testing.T) {
	tests := []struct {
		name      string
		route     RouteChoice
		mode      ArgumentMode
		arguments string
	}{
		{name: "accounts", route: RouteGetAccounts, mode: ArgumentsNone, arguments: `{}`},
		{name: "obligations", route: RouteGetObligations, mode: ArgumentsReferenceTime, arguments: `{"asOf":"2026-10-05"}`},
		{name: "receipts", route: RouteGetReceipts, mode: ArgumentsDefaults, arguments: `{}`},
		{name: "income period", route: RouteGetPeriodSinceIncome, mode: ArgumentsReferenceTime, arguments: `{"asOf":"2026-10-05T12:30:00+05:00"}`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			provider := &fakeRoutingProvider{judgment: RoutingJudgment{Route: test.route, RouteConfidence: .91, Arguments: test.mode, ArgumentConfidence: .83}}
			caller := &fakeToolCaller{result: ToolResult{Name: string(test.route), Data: map[string]any{"items": []any{}}}}
			result := NewRouter(provider, caller).Query(context.Background(), testQuery())
			if result.Status != StatusCompleted || caller.calls != 1 || caller.name != string(test.route) || string(caller.arguments) != test.arguments {
				t.Fatalf("unexpected result=%#v caller=%#v", result, caller)
			}
			if result.Confidence == nil || *result.Confidence != .83 {
				t.Fatalf("effective confidence must be minimum: %#v", result.Confidence)
			}
		})
	}
}

func TestRouterFallbacksNeverCallExecutor(t *testing.T) {
	tests := []struct {
		name     string
		judgment RoutingJudgment
		status   QueryStatus
		code     string
		required []string
	}{
		{name: "low confidence", judgment: RoutingJudgment{Route: RouteGetAccounts, RouteConfidence: .95, Arguments: ArgumentsNone, ArgumentConfidence: .69}, status: StatusClarify, code: "low_confidence"},
		{name: "period", judgment: RoutingJudgment{Route: RouteGetPeriodSummary, RouteConfidence: .9, Arguments: ArgumentsNeedsResolution, ArgumentConfidence: .9}, status: StatusClarify, code: "arguments_required", required: []string{"from", "to"}},
		{name: "cash", judgment: RoutingJudgment{Route: RouteGetCashPosition, RouteConfidence: .9, Arguments: ArgumentsMissing, ArgumentConfidence: .9}, status: StatusClarify, code: "arguments_required", required: []string{"through"}},
		{name: "named income", judgment: RoutingJudgment{Route: RouteGetPeriodSinceIncome, RouteConfidence: .9, Arguments: ArgumentsNeedsResolution, ArgumentConfidence: .9}, status: StatusClarify, code: "income_category_required", required: []string{"categoryId"}},
		{name: "modifier", judgment: RoutingJudgment{Route: RouteGetReceipts, RouteConfidence: .9, Arguments: ArgumentsNeedsResolution, ArgumentConfidence: .9}, status: StatusClarify, code: "arguments_required", required: []string{"limit"}},
		{name: "composition", judgment: RoutingJudgment{Route: RouteGetAccounts, RouteConfidence: .9, Arguments: ArgumentsDefaults, ArgumentConfidence: .8}, status: StatusClarify, code: "composition_mismatch"},
		{name: "unsupported", judgment: RoutingJudgment{Route: RouteUnsupported, RouteConfidence: .9, Arguments: ArgumentsNotApplicable, ArgumentConfidence: .1}, status: StatusUnsupported},
		{name: "unsupported ignores speculative arguments", judgment: RoutingJudgment{Route: RouteUnsupported, RouteConfidence: .9, Arguments: ArgumentsNeedsResolution, ArgumentConfidence: .1}, status: StatusUnsupported},
		{name: "clarify ignores speculative arguments", judgment: RoutingJudgment{Route: RouteClarify, RouteConfidence: .9, Arguments: ArgumentsReferenceTime, ArgumentConfidence: .1}, status: StatusClarify, code: "clarification_required"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			caller := &fakeToolCaller{}
			result := NewRouter(&fakeRoutingProvider{judgment: test.judgment}, caller).Query(context.Background(), testQuery())
			if result.Status != test.status || result.ClarificationCode != test.code || !reflect.DeepEqual(result.RequiredArguments, test.required) || caller.calls != 0 {
				t.Fatalf("unexpected result=%#v calls=%d", result, caller.calls)
			}
		})
	}
}

func TestRouterErrorsAreSafe(t *testing.T) {
	caller := &fakeToolCaller{}
	if result := NewRouter(&fakeRoutingProvider{err: errors.New("secret provider body")}, caller).Query(context.Background(), testQuery()); result.Status != StatusUnavailable || result.ErrorCode != "assistant_unavailable" || caller.calls != 0 {
		t.Fatalf("unexpected provider failure: %#v", result)
	}
	provider := &fakeRoutingProvider{judgment: RoutingJudgment{Route: RouteGetAccounts, RouteConfidence: .9, Arguments: ArgumentsNone, ArgumentConfidence: .9}}
	caller.err = &ToolError{Code: CodeDependencyFailure, Err: errors.New("database details")}
	if result := NewRouter(provider, caller).Query(context.Background(), testQuery()); result.Status != StatusUnavailable || result.ErrorCode != "dependency_unavailable" || result.Data != nil {
		t.Fatalf("unexpected dependency failure: %#v", result)
	}
	caller.err = &ToolError{Code: CodeAnchorNotFound, Err: errors.New("missing")}
	provider.judgment.Route, provider.judgment.Arguments = RouteGetPeriodSinceIncome, ArgumentsReferenceTime
	if result := NewRouter(provider, caller).Query(context.Background(), testQuery()); result.Status != StatusClarify || result.ClarificationCode != "no_income_anchor" {
		t.Fatalf("unexpected anchor result: %#v", result)
	}
}

func TestValidateQuery(t *testing.T) {
	query, err := ValidateQuery("  Покажи счета  ", "ru", "2026-10-05T12:00:00+05:00")
	if err != nil || query.Utterance != "Покажи счета" || query.ReferenceTime.Location() == time.UTC {
		t.Fatalf("unexpected query: %#v %v", query, err)
	}
	for _, input := range []struct{ utterance, language, reference string }{{"", "ru", "2026-10-05T12:00:00Z"}, {"ok", "de", "2026-10-05T12:00:00Z"}, {"ok", "ru", "2026-10-05T12:00:00"}} {
		if _, err := ValidateQuery(input.utterance, input.language, input.reference); err == nil {
			t.Fatalf("expected validation failure for %#v", input)
		}
	}
}

func TestQuestionSetV3KeepsClosedArgumentModes(t *testing.T) {
	if QuestionSetVersion != "v3" {
		t.Fatalf("unexpected question set version %q", QuestionSetVersion)
	}
	question := RoutingQuestions()[QuestionArguments]
	want := []ArgumentMode{ArgumentsNone, ArgumentsReferenceTime, ArgumentsDefaults, ArgumentsNeedsResolution, ArgumentsMissing, ArgumentsNotApplicable}
	if len(question.Criteria) != len(want) {
		t.Fatalf("unexpected criteria count: %d", len(question.Criteria))
	}
	for _, mode := range want {
		if _, ok := question.Criteria[string(mode)]; !ok {
			t.Fatalf("missing argument mode %s", mode)
		}
	}
}
