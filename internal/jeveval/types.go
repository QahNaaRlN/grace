package jeveval

import (
	"context"
	"time"
)

const (
	DatasetVersion     = "v1"
	QuestionSetVersion = "v1"
)

type Tool string

const (
	ToolGetAccounts      Tool = "get_accounts"
	ToolGetPeriodSummary Tool = "get_period_summary"
	ToolGetCashPosition  Tool = "get_cash_position"
	ToolGetObligations   Tool = "get_obligations"
	ToolGetReceipts      Tool = "get_receipts"
	ToolClarify          Tool = "clarify"
	ToolUnsupported      Tool = "unsupported"
)

var validTools = map[Tool]struct{}{
	ToolGetAccounts: {}, ToolGetPeriodSummary: {}, ToolGetCashPosition: {},
	ToolGetObligations: {}, ToolGetReceipts: {}, ToolClarify: {}, ToolUnsupported: {},
}

type PeriodShape string

const (
	PeriodNotApplicable PeriodShape = "not_applicable"
	PeriodCalendar      PeriodShape = "calendar_period"
	PeriodRelative      PeriodShape = "relative_period"
	PeriodExplicitRange PeriodShape = "explicit_range"
	PeriodMissing       PeriodShape = "missing_period"
)

var validPeriodShapes = map[PeriodShape]struct{}{
	PeriodNotApplicable: {}, PeriodCalendar: {}, PeriodRelative: {},
	PeriodExplicitRange: {}, PeriodMissing: {},
}

type ArgumentPresence string

const (
	ArgumentsSufficient    ArgumentPresence = "sufficient"
	ArgumentsMissing       ArgumentPresence = "missing"
	ArgumentsNotApplicable ArgumentPresence = "not_applicable"
)

var validArgumentPresence = map[ArgumentPresence]struct{}{
	ArgumentsSufficient: {}, ArgumentsMissing: {}, ArgumentsNotApplicable: {},
}

type Dataset struct {
	Version string           `json:"version"`
	Cases   []EvaluationCase `json:"cases"`
}

type EvaluationCase struct {
	ID                       string           `json:"id"`
	Language                 string           `json:"language"`
	Utterance                string           `json:"utterance"`
	ReferenceDate            string           `json:"referenceDate,omitempty"`
	ExpectedTool             Tool             `json:"expectedTool"`
	ExpectedPeriodShape      PeriodShape      `json:"expectedPeriodShape"`
	ExpectedArgumentPresence ArgumentPresence `json:"expectedArgumentPresence"`
	Tags                     []string         `json:"tags"`
}

type State struct {
	Utterance     string `json:"utterance"`
	Language      string `json:"language"`
	ReferenceDate string `json:"reference_date,omitempty"`
	ReferenceTime string `json:"reference_time,omitempty"`
}

type Question struct {
	Type         string            `json:"type"`
	Instructions any               `json:"instructions"`
	Criteria     map[string]string `json:"criteria"`
}

type Request struct {
	State     State               `json:"state"`
	Model     string              `json:"model"`
	Questions map[string]Question `json:"questions"`
}

type ChoiceAnswer struct {
	Type          string             `json:"type"`
	Choice        string             `json:"choice"`
	Probabilities map[string]float64 `json:"probabilities"`
	Confidence    float64            `json:"confidence"`
}

type Usage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

type Response struct {
	Model   string                  `json:"model"`
	Answers map[string]ChoiceAnswer `json:"answers"`
	Usage   Usage                   `json:"usage"`
}

type Provider interface {
	Evaluate(context.Context, Request) (Response, error)
}

type EvaluationObservation struct {
	CaseID                    string                  `json:"caseId"`
	Language                  string                  `json:"language"`
	ExpectedTool              Tool                    `json:"expectedTool"`
	PredictedTool             Tool                    `json:"predictedTool,omitempty"`
	RawToolChoice             Tool                    `json:"rawToolChoice,omitempty"`
	ExpectedPeriodShape       PeriodShape             `json:"expectedPeriodShape"`
	PredictedPeriodShape      PeriodShape             `json:"predictedPeriodShape,omitempty"`
	ExpectedArgumentPresence  ArgumentPresence        `json:"expectedArgumentPresence"`
	PredictedArgumentPresence ArgumentPresence        `json:"predictedArgumentPresence,omitempty"`
	Answers                   map[string]ChoiceAnswer `json:"answers,omitempty"`
	ActualModel               string                  `json:"actualModel,omitempty"`
	Usage                     Usage                   `json:"usage"`
	LatencyMilliseconds       int64                   `json:"latencyMilliseconds"`
	ErrorClass                string                  `json:"errorClass,omitempty"`
	ErrorMessage              string                  `json:"errorMessage,omitempty"`
	CompositionNote           string                  `json:"compositionNote,omitempty"`
}

type Report struct {
	DatasetVersion     string                  `json:"datasetVersion"`
	QuestionSetVersion string                  `json:"questionSetVersion"`
	RequestedModel     string                  `json:"requestedModel"`
	ActualModels       []string                `json:"actualModels"`
	GeneratedAt        time.Time               `json:"generatedAt"`
	AnalyzedAt         time.Time               `json:"analyzedAt"`
	Observations       []EvaluationObservation `json:"observations"`
	Metrics            map[string]Metrics      `json:"metricsByLanguage"`
}

type Metrics struct {
	Total                    int                       `json:"total"`
	Completed                int                       `json:"completed"`
	ProviderErrors           int                       `json:"providerErrors"`
	CompositionErrors        int                       `json:"compositionErrors"`
	ToolExact                int                       `json:"toolExact"`
	PeriodExact              int                       `json:"periodExact"`
	ArgumentPresenceExact    int                       `json:"argumentPresenceExact"`
	ToolAccuracy             float64                   `json:"toolAccuracy"`
	PeriodAccuracy           float64                   `json:"periodAccuracy"`
	ArgumentPresenceAccuracy float64                   `json:"argumentPresenceAccuracy"`
	ClarificationRate        float64                   `json:"clarificationRate"`
	Confusion                map[string]map[string]int `json:"toolConfusion"`
	ConfidenceBuckets        []ConfidenceBucket        `json:"confidenceBuckets"`
	Thresholds               []ThresholdMetrics        `json:"thresholds"`
	HighConfidenceErrors     []string                  `json:"highConfidenceErrorCaseIds"`
}

type ConfidenceBucket struct {
	Label    string  `json:"label"`
	Total    int     `json:"total"`
	Correct  int     `json:"correct"`
	Accuracy float64 `json:"accuracy"`
}

type ThresholdMetrics struct {
	Threshold float64 `json:"threshold"`
	Eligible  int     `json:"eligible"`
	Incorrect int     `json:"incorrect"`
	Coverage  float64 `json:"coverage"`
	ErrorRate float64 `json:"errorRate"`
}
