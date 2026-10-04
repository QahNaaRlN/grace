package routereval

import (
	"context"
	"testing"
	"time"

	"github.com/QahNaaRlN/grace/internal/assistant"
	"github.com/QahNaaRlN/grace/internal/jeveval"
)

type corpusProvider struct {
	byUtterance map[string]Case
	sabotage    string
}

func (provider corpusProvider) Evaluate(_ context.Context, request jeveval.Request) (jeveval.Response, error) {
	evaluationCase := provider.byUtterance[request.State.Utterance]
	route, mode := string(evaluationCase.ExpectedRoute), string(evaluationCase.ExpectedArgumentMode)
	if evaluationCase.ID == provider.sabotage {
		route, mode = string(assistant.RouteGetAccounts), string(assistant.ArgumentsNone)
	}
	answers := map[string]jeveval.ChoiceAnswer{}
	for id, question := range request.Questions {
		choice := route
		if id == assistant.QuestionArguments {
			choice = mode
		}
		probabilities := map[string]float64{}
		for option := range question.Criteria {
			probabilities[option] = 0
		}
		probabilities[choice] = 1
		answers[id] = jeveval.ChoiceAnswer{Type: "choice", Choice: choice, Probabilities: probabilities, Confidence: .95}
	}
	return jeveval.Response{Model: request.Model, Answers: answers}, nil
}

func TestV2CorpusPassesWithExactJudgments(t *testing.T) {
	dataset, err := LoadDataset("../../testdata/jev-evaluation/v2/cases.json")
	if err != nil {
		t.Fatal(err)
	}
	byUtterance := map[string]Case{}
	for _, evaluationCase := range dataset.Cases {
		byUtterance[evaluationCase.Utterance] = evaluationCase
	}
	report, err := Evaluate(context.Background(), dataset, assistant.ProductionModel, corpusProvider{byUtterance: byUtterance}, func() time.Time { return time.Unix(0, 0) })
	if err != nil || !report.GatePassed || report.MetricsByLanguage["ru"].Exact != 64 || report.MetricsByLanguage["ru"].SafeCoverage != 1 {
		t.Fatalf("unexpected report: %#v %v", report.MetricsByLanguage, err)
	}
}

func TestV2GateRejectsWrongAutoExecution(t *testing.T) {
	dataset, err := LoadDataset("../../testdata/jev-evaluation/v2/cases.json")
	if err != nil {
		t.Fatal(err)
	}
	byUtterance := map[string]Case{}
	for _, evaluationCase := range dataset.Cases {
		byUtterance[evaluationCase.Utterance] = evaluationCase
	}
	report, err := Evaluate(context.Background(), dataset, assistant.ProductionModel, corpusProvider{byUtterance: byUtterance, sabotage: "ru-unsupported-write-1"}, time.Now)
	if err != nil || report.GatePassed || report.MetricsByLanguage["ru"].WrongAutoExecutions != 1 || report.MetricsByLanguage["ru"].UnsafeWriteExecutions != 1 {
		t.Fatalf("wrong auto-execution was not rejected: %#v %v", report.MetricsByLanguage["ru"], err)
	}
}
