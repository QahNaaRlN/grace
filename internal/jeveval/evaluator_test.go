package jeveval

import (
	"context"
	"testing"
	"time"
)

type fakeProvider struct {
	evaluate func(Request) (Response, error)
}

func (provider fakeProvider) Evaluate(_ context.Context, request Request) (Response, error) {
	return provider.evaluate(request)
}

func TestEvaluateProducesPerfectSeparatedMetricsWithDeterministicFake(t *testing.T) {
	dataset, err := LoadDataset(fixturePath)
	if err != nil {
		t.Fatal(err)
	}
	expectedByUtterance := make(map[string]EvaluationCase, len(dataset.Cases))
	for _, evaluationCase := range dataset.Cases {
		expectedByUtterance[evaluationCase.Utterance] = evaluationCase
	}
	provider := fakeProvider{evaluate: func(request Request) (Response, error) {
		expected := expectedByUtterance[request.State.Utterance]
		return validResponse(request, map[string]string{
			questionTool: string(expected.ExpectedTool), questionPeriod: string(expected.ExpectedPeriodShape),
			questionArguments: string(expected.ExpectedArgumentPresence),
		}), nil
	}}

	report, err := Evaluate(context.Background(), dataset, "jev-1.13.0", provider, func() time.Time {
		return time.Date(2026, 10, 4, 12, 0, 0, 0, time.UTC)
	})
	if err != nil {
		t.Fatal(err)
	}
	if report.Metrics["ru"].ToolAccuracy != 1 || report.Metrics["en"].ToolAccuracy != 1 {
		t.Fatalf("tool metrics = %#v", report.Metrics)
	}
	if report.Metrics["ru"].Total != 42 || report.Metrics["en"].Total != 7 {
		t.Fatalf("language totals = ru:%d en:%d", report.Metrics["ru"].Total, report.Metrics["en"].Total)
	}
	if len(report.ActualModels) != 1 || report.ActualModels[0] != "jev-1.13.0" {
		t.Fatalf("actual models = %v", report.ActualModels)
	}
	if got := report.Metrics["ru"].Thresholds[4].Coverage; got != 1 {
		t.Fatalf("coverage at 0.90 = %f, want 1", got)
	}
	if got := report.Metrics["ru"].Thresholds[5].Coverage; got != 0 {
		t.Fatalf("coverage at 0.95 = %f, want 0", got)
	}
	if got := report.Metrics["ru"].ConfidenceBuckets[3].Total; got != 42 {
		t.Fatalf("high-confidence bucket total = %d, want 42", got)
	}
	if len(report.Metrics["ru"].HighConfidenceErrors) != 0 {
		t.Fatalf("high-confidence errors = %v", report.Metrics["ru"].HighConfidenceErrors)
	}
}

func TestEvaluateSeparatesProviderErrors(t *testing.T) {
	dataset, err := LoadDataset(fixturePath)
	if err != nil {
		t.Fatal(err)
	}
	provider := fakeProvider{evaluate: func(Request) (Response, error) {
		return Response{}, &ProviderError{Class: ErrorOverload, Status: 529}
	}}
	report, err := Evaluate(context.Background(), dataset, "jev-1.13.0", provider, time.Now)
	if err != nil {
		t.Fatal(err)
	}
	if report.Metrics["ru"].ProviderErrors != 42 || report.Metrics["ru"].Completed != 0 {
		t.Fatalf("Russian metrics = %#v", report.Metrics["ru"])
	}
	if report.Observations[0].ErrorClass != ErrorOverload {
		t.Fatalf("error class = %q", report.Observations[0].ErrorClass)
	}
}

func TestEvaluateCaseNormalizesMissingPeriodToClarify(t *testing.T) {
	evaluationCase := EvaluationCase{
		ID: "case", Language: "ru", Utterance: "Сколько я потратил?",
		ExpectedTool: ToolClarify, ExpectedPeriodShape: PeriodMissing,
		ExpectedArgumentPresence: ArgumentsMissing, Tags: []string{"test"},
	}
	provider := fakeProvider{evaluate: func(request Request) (Response, error) {
		return validResponse(request, map[string]string{
			questionTool: string(ToolGetPeriodSummary), questionPeriod: string(PeriodMissing),
			questionArguments: string(ArgumentsMissing),
		}), nil
	}}
	observation := evaluateCase(context.Background(), evaluationCase, "jev-1.13.0", provider)
	if observation.PredictedTool != ToolClarify || observation.RawToolChoice != ToolGetPeriodSummary {
		t.Fatalf("observation = %#v", observation)
	}
	if observation.ErrorClass != "" || observation.CompositionNote == "" {
		t.Fatalf("composition result = %#v", observation)
	}
}

func TestValidateModelIDRejectsAliases(t *testing.T) {
	for _, model := range []string{"", "jev-latest", "jev-preview", "other-1.0.0"} {
		if err := ValidateModelID(model); err == nil {
			t.Errorf("ValidateModelID(%q) succeeded", model)
		}
	}
	if err := ValidateModelID("jev-1.13.0"); err != nil {
		t.Fatalf("versioned model rejected: %v", err)
	}
}

func TestEvaluateRequiresProvider(t *testing.T) {
	dataset, err := LoadDataset(fixturePath)
	if err != nil {
		t.Fatal(err)
	}
	_, err = Evaluate(context.Background(), dataset, "jev-1.13.0", nil, time.Now)
	if err == nil {
		t.Fatal("expected provider error")
	}
}
