package routereval

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/QahNaaRlN/grace/internal/assistant"
	"github.com/QahNaaRlN/grace/internal/jeveval"
)

type Observation struct {
	CaseID               string                 `json:"caseId"`
	Language             string                 `json:"language"`
	Tags                 []string               `json:"tags"`
	ExpectedRoute        assistant.RouteChoice  `json:"expectedRoute"`
	ActualRoute          assistant.RouteChoice  `json:"actualRoute,omitempty"`
	ExpectedArgumentMode assistant.ArgumentMode `json:"expectedArgumentMode"`
	ActualArgumentMode   assistant.ArgumentMode `json:"actualArgumentMode,omitempty"`
	ExpectedDecision     Decision               `json:"expectedDecision"`
	ActualDecision       Decision               `json:"actualDecision,omitempty"`
	RouteConfidence      float64                `json:"routeConfidence,omitempty"`
	ArgumentConfidence   float64                `json:"argumentConfidence,omitempty"`
	EffectiveConfidence  float64                `json:"effectiveConfidence,omitempty"`
	ActualModel          string                 `json:"actualModel,omitempty"`
	LatencyMilliseconds  int64                  `json:"latencyMilliseconds"`
	ErrorClass           string                 `json:"errorClass,omitempty"`
	ErrorMessage         string                 `json:"errorMessage,omitempty"`
}

type Metrics struct {
	Total                    int      `json:"total"`
	Completed                int      `json:"completed"`
	Errors                   int      `json:"errors"`
	Exact                    int      `json:"exact"`
	Eligible                 int      `json:"eligibleAtThreshold"`
	SafeCoverage             float64  `json:"safeDecisionCoverage"`
	WrongAutoExecutions      int      `json:"wrongAutoExecutions"`
	UnsafeModifierExecutions int      `json:"unsafeModifierExecutions"`
	UnsafeWriteExecutions    int      `json:"unsafeWriteExecutions"`
	HighConfidenceMismatches []string `json:"highConfidenceMismatchCaseIds"`
	Pass                     bool     `json:"pass"`
}

type Report struct {
	DatasetVersion     string             `json:"datasetVersion"`
	QuestionSetVersion string             `json:"questionSetVersion"`
	RequestedModel     string             `json:"requestedModel"`
	ActualModels       []string           `json:"actualModels"`
	GeneratedAt        time.Time          `json:"generatedAt"`
	Observations       []Observation      `json:"observations"`
	MetricsByLanguage  map[string]Metrics `json:"metricsByLanguage"`
	GatePassed         bool               `json:"gatePassed"`
}

func Evaluate(ctx context.Context, dataset Dataset, model string, provider jeveval.Provider, now func() time.Time) (Report, error) {
	if err := ValidateDataset(dataset); err != nil {
		return Report{}, err
	}
	if err := jeveval.ValidateModelID(model); err != nil {
		return Report{}, err
	}
	if provider == nil {
		return Report{}, errors.New("provider is required")
	}
	if now == nil {
		now = time.Now
	}
	report := Report{DatasetVersion: DatasetVersion, QuestionSetVersion: assistant.QuestionSetVersion, RequestedModel: model, GeneratedAt: now().UTC(), Observations: make([]Observation, 0, len(dataset.Cases))}
	models := map[string]bool{}
	for _, evaluationCase := range dataset.Cases {
		observation := evaluateCase(ctx, evaluationCase, model, provider)
		report.Observations = append(report.Observations, observation)
		if observation.ActualModel != "" {
			models[observation.ActualModel] = true
		}
	}
	for actualModel := range models {
		report.ActualModels = append(report.ActualModels, actualModel)
	}
	sort.Strings(report.ActualModels)
	report.MetricsByLanguage = calculateMetrics(report.Observations)
	report.GatePassed = report.MetricsByLanguage["ru"].Pass && report.MetricsByLanguage["en"].Pass
	return report, nil
}

func evaluateCase(ctx context.Context, evaluationCase Case, model string, provider jeveval.Provider) Observation {
	observation := Observation{CaseID: evaluationCase.ID, Language: evaluationCase.Language, Tags: evaluationCase.Tags, ExpectedRoute: evaluationCase.ExpectedRoute, ExpectedArgumentMode: evaluationCase.ExpectedArgumentMode, ExpectedDecision: evaluationCase.ExpectedDecision}
	questions := map[string]jeveval.Question{}
	for id, question := range assistant.RoutingQuestions() {
		questions[id] = jeveval.Question{Type: question.Type, Instructions: question.Instructions, Criteria: question.Criteria}
	}
	request := jeveval.Request{State: jeveval.State{Utterance: evaluationCase.Utterance, Language: evaluationCase.Language, ReferenceTime: evaluationCase.ReferenceTime}, Model: model, Questions: questions}
	started := time.Now()
	response, err := provider.Evaluate(ctx, request)
	observation.LatencyMilliseconds = time.Since(started).Milliseconds()
	if err != nil {
		observation.ErrorClass = jeveval.ProviderErrorClass(err)
		observation.ErrorMessage = err.Error()
		return observation
	}
	if err := jeveval.ValidateResponse(request, response); err != nil {
		observation.ErrorClass = jeveval.ErrorProtocol
		observation.ErrorMessage = err.Error()
		return observation
	}
	observation.ActualModel = response.Model
	route := response.Answers[assistant.QuestionRoute]
	mode := response.Answers[assistant.QuestionArguments]
	observation.ActualRoute = assistant.RouteChoice(route.Choice)
	observation.ActualArgumentMode = assistant.ArgumentMode(mode.Choice)
	observation.RouteConfidence = route.Confidence
	observation.ArgumentConfidence = mode.Confidence
	observation.EffectiveConfidence = route.Confidence
	if observation.ActualRoute != assistant.RouteClarify && observation.ActualRoute != assistant.RouteUnsupported && mode.Confidence < observation.EffectiveConfidence {
		observation.EffectiveConfidence = mode.Confidence
	}
	applyProductionDecision(ctx, evaluationCase, &observation)
	return observation
}

func applyProductionDecision(ctx context.Context, evaluationCase Case, observation *Observation) {
	observation.EffectiveConfidence = observation.RouteConfidence
	terminalRoute := observation.ActualRoute == assistant.RouteClarify || observation.ActualRoute == assistant.RouteUnsupported
	if !terminalRoute && observation.ArgumentConfidence < observation.EffectiveConfidence {
		observation.EffectiveConfidence = observation.ArgumentConfidence
	}
	parsed, _ := time.Parse(time.RFC3339, evaluationCase.ReferenceTime)
	router := assistant.NewRouter(fixedProvider{judgment: assistant.RoutingJudgment{Route: observation.ActualRoute, RouteConfidence: observation.RouteConfidence, Arguments: observation.ActualArgumentMode, ArgumentConfidence: observation.ArgumentConfidence}}, successfulCaller{})
	result := router.Query(ctx, assistant.Query{Utterance: evaluationCase.Utterance, Language: evaluationCase.Language, ReferenceTime: parsed})
	observation.ActualDecision = Decision(result.Status)
	if result.Status == assistant.StatusUnavailable {
		observation.ErrorClass = "composition_error"
		observation.ErrorMessage = result.ErrorCode
	} else if observation.ErrorClass == "composition_error" {
		observation.ErrorClass = ""
		observation.ErrorMessage = ""
	}
}

type fixedProvider struct{ judgment assistant.RoutingJudgment }

func (provider fixedProvider) Judge(context.Context, assistant.Query) (assistant.RoutingJudgment, error) {
	return provider.judgment, nil
}

type successfulCaller struct{}

func (successfulCaller) Call(context.Context, string, json.RawMessage) (assistant.ToolResult, error) {
	return assistant.ToolResult{Name: "evaluation", Data: map[string]any{}}, nil
}

func calculateMetrics(observations []Observation) map[string]Metrics {
	result := map[string]Metrics{}
	for _, language := range []string{"ru", "en"} {
		metric := Metrics{}
		for _, observation := range observations {
			if observation.Language != language {
				continue
			}
			metric.Total++
			if observation.ErrorClass != "" {
				metric.Errors++
				continue
			}
			metric.Completed++
			argumentExact := observation.ActualArgumentMode == observation.ExpectedArgumentMode
			if observation.ActualRoute == assistant.RouteClarify || observation.ActualRoute == assistant.RouteUnsupported {
				argumentExact = true
			}
			exact := observation.ActualRoute == observation.ExpectedRoute && argumentExact && observation.ActualDecision == observation.ExpectedDecision
			if exact {
				metric.Exact++
			}
			if observation.EffectiveConfidence >= assistant.ConfidenceThreshold {
				if exact {
					metric.Eligible++
				} else {
					metric.HighConfidenceMismatches = append(metric.HighConfidenceMismatches, observation.CaseID)
				}
			}
			if observation.ActualDecision == DecisionCompleted && !exact {
				metric.WrongAutoExecutions++
			}
			if observation.ActualDecision == DecisionCompleted && (hasTag(observation.Tags, "explicit-modifier") || hasTag(observation.Tags, "named-income-category")) {
				metric.UnsafeModifierExecutions++
			}
			if observation.ActualDecision == DecisionCompleted && hasTag(observation.Tags, "unsupported-write") {
				metric.UnsafeWriteExecutions++
			}
		}
		if metric.Completed > 0 {
			metric.SafeCoverage = float64(metric.Eligible) / float64(metric.Completed)
		}
		metric.Pass = metric.Errors == 0 && metric.WrongAutoExecutions == 0 && metric.UnsafeModifierExecutions == 0 && metric.UnsafeWriteExecutions == 0 && len(metric.HighConfidenceMismatches) == 0 && (language != "ru" || metric.SafeCoverage >= .65)
		result[language] = metric
	}
	return result
}

func hasTag(tags []string, wanted string) bool {
	for _, tag := range tags {
		if tag == wanted {
			return true
		}
	}
	return false
}

func Summary(report Report) string {
	ru := report.MetricsByLanguage["ru"]
	return fmt.Sprintf("dataset=%s questions=%s model=%s ru=%d/%d coverage@0.70=%.4f wrong_auto=%d errors=%d gate_passed=%t", report.DatasetVersion, report.QuestionSetVersion, report.RequestedModel, ru.Exact, ru.Completed, ru.SafeCoverage, ru.WrongAutoExecutions, ru.Errors, report.GatePassed)
}

func Reanalyze(dataset Dataset, report Report) (Report, error) {
	if err := ValidateDataset(dataset); err != nil {
		return Report{}, err
	}
	if report.DatasetVersion != DatasetVersion || report.QuestionSetVersion != assistant.QuestionSetVersion {
		return Report{}, errors.New("report versions do not match v2")
	}
	if len(report.Observations) != len(dataset.Cases) {
		return Report{}, errors.New("report observation count does not match dataset")
	}
	cases := map[string]Case{}
	for _, evaluationCase := range dataset.Cases {
		cases[evaluationCase.ID] = evaluationCase
	}
	seen := map[string]bool{}
	for index := range report.Observations {
		observation := &report.Observations[index]
		evaluationCase, ok := cases[observation.CaseID]
		if !ok || seen[observation.CaseID] {
			return Report{}, fmt.Errorf("unknown or duplicate observation %q", observation.CaseID)
		}
		seen[observation.CaseID] = true
		observation.Language = evaluationCase.Language
		observation.Tags = evaluationCase.Tags
		observation.ExpectedRoute = evaluationCase.ExpectedRoute
		observation.ExpectedArgumentMode = evaluationCase.ExpectedArgumentMode
		observation.ExpectedDecision = evaluationCase.ExpectedDecision
		if observation.ErrorClass == "" || observation.ErrorClass == "composition_error" {
			applyProductionDecision(context.Background(), evaluationCase, observation)
		}
	}
	report.MetricsByLanguage = calculateMetrics(report.Observations)
	report.GatePassed = report.MetricsByLanguage["ru"].Pass && report.MetricsByLanguage["en"].Pass
	return report, nil
}
