package jeveval

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"time"
)

var versionedModelPattern = regexp.MustCompile(`^jev-[0-9]+\.[0-9]+\.[0-9]+$`)

var thresholdCandidates = []float64{0.50, 0.60, 0.70, 0.80, 0.90, 0.95}

func ValidateModelID(model string) error {
	if !versionedModelPattern.MatchString(model) {
		return fmt.Errorf("model must be an explicit versioned Jev id such as jev-1.13.0")
	}
	return nil
}

func Evaluate(ctx context.Context, dataset Dataset, model string, provider Provider, now func() time.Time) (Report, error) {
	if err := ValidateDataset(dataset); err != nil {
		return Report{}, err
	}
	if err := ValidateModelID(model); err != nil {
		return Report{}, err
	}
	if provider == nil {
		return Report{}, errors.New("provider is required")
	}
	if now == nil {
		now = time.Now
	}

	report := Report{
		DatasetVersion: DatasetVersion, QuestionSetVersion: QuestionSetVersion,
		RequestedModel: model, GeneratedAt: now().UTC(),
		Observations: make([]EvaluationObservation, 0, len(dataset.Cases)),
	}
	models := make(map[string]struct{})
	for _, evaluationCase := range dataset.Cases {
		observation := evaluateCase(ctx, evaluationCase, model, provider)
		report.Observations = append(report.Observations, observation)
		if observation.ActualModel != "" {
			models[observation.ActualModel] = struct{}{}
		}
	}
	for modelID := range models {
		report.ActualModels = append(report.ActualModels, modelID)
	}
	sort.Strings(report.ActualModels)
	report.Metrics = calculateMetrics(report.Observations)
	return report, nil
}

func evaluateCase(ctx context.Context, evaluationCase EvaluationCase, model string, provider Provider) EvaluationObservation {
	observation := EvaluationObservation{
		CaseID: evaluationCase.ID, Language: evaluationCase.Language,
		ExpectedTool:             evaluationCase.ExpectedTool,
		ExpectedPeriodShape:      evaluationCase.ExpectedPeriodShape,
		ExpectedArgumentPresence: evaluationCase.ExpectedArgumentPresence,
	}
	request := requestFor(evaluationCase, model)
	started := time.Now()
	response, err := provider.Evaluate(ctx, request)
	observation.LatencyMilliseconds = time.Since(started).Milliseconds()
	if err != nil {
		observation.ErrorClass = ProviderErrorClass(err)
		observation.ErrorMessage = err.Error()
		return observation
	}
	if err := ValidateResponse(request, response); err != nil {
		observation.ErrorClass = ErrorProtocol
		observation.ErrorMessage = err.Error()
		return observation
	}

	observation.Answers = response.Answers
	observation.ActualModel = response.Model
	observation.Usage = response.Usage
	observation.RawToolChoice = Tool(response.Answers[questionTool].Choice)
	observation.PredictedTool = observation.RawToolChoice
	observation.PredictedPeriodShape = PeriodShape(response.Answers[questionPeriod].Choice)
	observation.PredictedArgumentPresence = ArgumentPresence(response.Answers[questionArguments].Choice)

	if observation.RawToolChoice == ToolGetPeriodSummary || observation.RawToolChoice == ToolGetCashPosition {
		switch observation.PredictedArgumentPresence {
		case ArgumentsMissing:
			observation.PredictedTool = ToolClarify
			observation.CompositionNote = "period-sensitive route normalized to clarify because temporal arguments are missing"
		case ArgumentsSufficient:
		default:
			observation.ErrorClass = "composition_error"
			observation.ErrorMessage = "period-sensitive route cannot have not_applicable arguments"
		}
	} else if observation.PredictedArgumentPresence != ArgumentsNotApplicable && observation.RawToolChoice != ToolClarify {
		observation.ErrorClass = "composition_error"
		observation.ErrorMessage = "non-period route must have not_applicable arguments"
	}
	return observation
}

func calculateMetrics(observations []EvaluationObservation) map[string]Metrics {
	metrics := make(map[string]Metrics)
	for _, language := range []string{"ru", "en"} {
		var selected []EvaluationObservation
		for _, observation := range observations {
			if observation.Language == language {
				selected = append(selected, observation)
			}
		}
		metrics[language] = metricsFor(selected)
	}
	return metrics
}

func metricsFor(observations []EvaluationObservation) Metrics {
	metrics := Metrics{Total: len(observations), Confusion: make(map[string]map[string]int)}
	buckets := []struct {
		label     string
		minimum   float64
		maximum   float64
		inclusive bool
	}{
		{label: "[0.00,0.50)", minimum: 0, maximum: 0.50},
		{label: "[0.50,0.70)", minimum: 0.50, maximum: 0.70},
		{label: "[0.70,0.90)", minimum: 0.70, maximum: 0.90},
		{label: "[0.90,1.00]", minimum: 0.90, maximum: 1.00, inclusive: true},
	}
	metrics.ConfidenceBuckets = make([]ConfidenceBucket, len(buckets))
	for index, bucket := range buckets {
		metrics.ConfidenceBuckets[index].Label = bucket.label
	}
	clarifications := 0
	for _, observation := range observations {
		if observation.ErrorClass != "" {
			if observation.ErrorClass == "composition_error" {
				metrics.CompositionErrors++
			} else {
				metrics.ProviderErrors++
			}
			continue
		}
		metrics.Completed++
		if metrics.Confusion[string(observation.ExpectedTool)] == nil {
			metrics.Confusion[string(observation.ExpectedTool)] = make(map[string]int)
		}
		metrics.Confusion[string(observation.ExpectedTool)][string(observation.PredictedTool)]++
		if observation.PredictedTool == observation.ExpectedTool {
			metrics.ToolExact++
		}
		if observation.PredictedPeriodShape == observation.ExpectedPeriodShape {
			metrics.PeriodExact++
		}
		if observation.PredictedArgumentPresence == observation.ExpectedArgumentPresence {
			metrics.ArgumentPresenceExact++
		}
		if observation.PredictedTool == ToolClarify {
			clarifications++
		}
		confidence := routeConfidence(observation)
		for index, bucket := range buckets {
			if confidence >= bucket.minimum && (confidence < bucket.maximum || bucket.inclusive && confidence <= bucket.maximum) {
				metrics.ConfidenceBuckets[index].Total++
				if observation.PredictedTool == observation.ExpectedTool {
					metrics.ConfidenceBuckets[index].Correct++
				}
				break
			}
		}
		if confidence >= 0.90 && observation.PredictedTool != observation.ExpectedTool {
			metrics.HighConfidenceErrors = append(metrics.HighConfidenceErrors, observation.CaseID)
		}
	}
	if metrics.Completed > 0 {
		denominator := float64(metrics.Completed)
		metrics.ToolAccuracy = float64(metrics.ToolExact) / denominator
		metrics.PeriodAccuracy = float64(metrics.PeriodExact) / denominator
		metrics.ArgumentPresenceAccuracy = float64(metrics.ArgumentPresenceExact) / denominator
		metrics.ClarificationRate = float64(clarifications) / denominator
	}
	for index := range metrics.ConfidenceBuckets {
		bucket := &metrics.ConfidenceBuckets[index]
		if bucket.Total > 0 {
			bucket.Accuracy = float64(bucket.Correct) / float64(bucket.Total)
		}
	}
	for _, threshold := range thresholdCandidates {
		thresholdMetric := ThresholdMetrics{Threshold: threshold}
		for _, observation := range observations {
			if observation.ErrorClass != "" {
				continue
			}
			confidence := routeConfidence(observation)
			if confidence >= threshold {
				thresholdMetric.Eligible++
				if observation.PredictedTool != observation.ExpectedTool {
					thresholdMetric.Incorrect++
				}
			}
		}
		if metrics.Completed > 0 {
			thresholdMetric.Coverage = float64(thresholdMetric.Eligible) / float64(metrics.Completed)
		}
		if thresholdMetric.Eligible > 0 {
			thresholdMetric.ErrorRate = float64(thresholdMetric.Incorrect) / float64(thresholdMetric.Eligible)
		}
		metrics.Thresholds = append(metrics.Thresholds, thresholdMetric)
	}
	return metrics
}

func routeConfidence(observation EvaluationObservation) float64 {
	confidence := observation.Answers[questionTool].Confidence
	if observation.RawToolChoice == ToolGetPeriodSummary || observation.RawToolChoice == ToolGetCashPosition || observation.RawToolChoice == ToolClarify {
		if candidate := observation.Answers[questionArguments].Confidence; candidate < confidence {
			confidence = candidate
		}
	}
	return confidence
}
