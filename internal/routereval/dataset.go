package routereval

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/QahNaaRlN/grace/internal/assistant"
)

const DatasetVersion = "v2"

type Decision string

const (
	DecisionCompleted   Decision = "completed"
	DecisionClarify     Decision = "clarify"
	DecisionUnsupported Decision = "unsupported"
)

type Dataset struct {
	Version string `json:"version"`
	Cases   []Case `json:"cases"`
}

type Case struct {
	ID                   string                 `json:"id"`
	Language             string                 `json:"language"`
	Utterance            string                 `json:"utterance"`
	ReferenceTime        string                 `json:"referenceTime"`
	ExpectedRoute        assistant.RouteChoice  `json:"expectedRoute"`
	ExpectedArgumentMode assistant.ArgumentMode `json:"expectedArgumentMode"`
	ExpectedDecision     Decision               `json:"expectedDecision"`
	Tags                 []string               `json:"tags"`
}

var prohibited = []*regexp.Regexp{
	regexp.MustCompile(`(?i)\b[A-Z0-9._%+-]+@[A-Z0-9.-]+\.[A-Z]{2,}\b`),
	regexp.MustCompile(`\b\d{9,}\b`),
	regexp.MustCompile(`(?i)authorization\s*:|typesafe_api_key|bearer\s+[a-z0-9._-]+|\bинн\b`),
}

func LoadDataset(path string) (Dataset, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Dataset{}, fmt.Errorf("read dataset: %w", err)
	}
	return ParseDataset(data)
}

func ParseDataset(data []byte) (Dataset, error) {
	var dataset Dataset
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&dataset); err != nil {
		return Dataset{}, fmt.Errorf("decode dataset: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return Dataset{}, errors.New("decode dataset: expected one JSON document")
	}
	if err := ValidateDataset(dataset); err != nil {
		return Dataset{}, err
	}
	return dataset, nil
}

func ValidateDataset(dataset Dataset) error {
	if dataset.Version != DatasetVersion {
		return fmt.Errorf("dataset version must be %q", DatasetVersion)
	}
	validRoutes := map[assistant.RouteChoice]bool{assistant.RouteGetAccounts: true, assistant.RouteGetPeriodSummary: true, assistant.RouteGetCashPosition: true, assistant.RouteGetObligations: true, assistant.RouteGetReceipts: true, assistant.RouteGetPeriodSinceIncome: true, assistant.RouteClarify: true, assistant.RouteUnsupported: true}
	validModes := map[assistant.ArgumentMode]bool{assistant.ArgumentsNone: true, assistant.ArgumentsReferenceTime: true, assistant.ArgumentsDefaults: true, assistant.ArgumentsNeedsResolution: true, assistant.ArgumentsMissing: true, assistant.ArgumentsNotApplicable: true}
	validDecisions := map[Decision]bool{DecisionCompleted: true, DecisionClarify: true, DecisionUnsupported: true}
	seen := map[string]bool{}
	counts := map[string]map[assistant.RouteChoice]int{"ru": {}, "en": {}}
	ruExecutable := 0
	tags := map[string]bool{}
	for index, evaluationCase := range dataset.Cases {
		prefix := fmt.Sprintf("case %d", index+1)
		if strings.TrimSpace(evaluationCase.ID) == "" || seen[evaluationCase.ID] {
			return fmt.Errorf("%s: id is empty or duplicate", prefix)
		}
		seen[evaluationCase.ID] = true
		if evaluationCase.Language != "ru" && evaluationCase.Language != "en" {
			return fmt.Errorf("case %q: language must be ru or en", evaluationCase.ID)
		}
		if strings.TrimSpace(evaluationCase.Utterance) == "" {
			return fmt.Errorf("case %q: utterance is required", evaluationCase.ID)
		}
		for _, pattern := range prohibited {
			if pattern.MatchString(evaluationCase.Utterance) {
				return fmt.Errorf("case %q: utterance contains prohibited data", evaluationCase.ID)
			}
		}
		if _, err := time.Parse(time.RFC3339, evaluationCase.ReferenceTime); err != nil {
			return fmt.Errorf("case %q: referenceTime must be RFC 3339", evaluationCase.ID)
		}
		if !validRoutes[evaluationCase.ExpectedRoute] || !validModes[evaluationCase.ExpectedArgumentMode] || !validDecisions[evaluationCase.ExpectedDecision] {
			return fmt.Errorf("case %q: invalid expected outcome", evaluationCase.ID)
		}
		if len(evaluationCase.Tags) == 0 {
			return fmt.Errorf("case %q: tags are required", evaluationCase.ID)
		}
		for _, tag := range evaluationCase.Tags {
			tags[tag] = true
		}
		if err := validateCombination(evaluationCase); err != nil {
			return fmt.Errorf("case %q: %w", evaluationCase.ID, err)
		}
		counts[evaluationCase.Language][evaluationCase.ExpectedRoute]++
		if evaluationCase.Language == "ru" && evaluationCase.ExpectedDecision == DecisionCompleted {
			ruExecutable++
		}
	}
	for route := range validRoutes {
		if counts["ru"][route] < 8 {
			return fmt.Errorf("dataset needs at least 8 Russian cases for %s", route)
		}
		if counts["en"][route] < 1 {
			return fmt.Errorf("dataset needs at least 1 English case for %s", route)
		}
	}
	if ruExecutable < 20 {
		return errors.New("dataset needs at least 20 Russian auto-executable cases")
	}
	for _, tag := range []string{"default-safe", "explicit-modifier", "missing-argument", "named-income-category", "unsupported-write", "ambiguous-route"} {
		if !tags[tag] {
			return fmt.Errorf("dataset is missing required tag %q", tag)
		}
	}
	return nil
}

func validateCombination(evaluationCase Case) error {
	executable := map[assistant.RouteChoice]assistant.ArgumentMode{
		assistant.RouteGetAccounts:          assistant.ArgumentsNone,
		assistant.RouteGetObligations:       assistant.ArgumentsReferenceTime,
		assistant.RouteGetReceipts:          assistant.ArgumentsDefaults,
		assistant.RouteGetPeriodSinceIncome: assistant.ArgumentsReferenceTime,
	}
	if mode, ok := executable[evaluationCase.ExpectedRoute]; ok && evaluationCase.ExpectedArgumentMode == mode {
		if evaluationCase.ExpectedDecision != DecisionCompleted {
			return errors.New("safe route/mode must be completed")
		}
		return nil
	}
	if evaluationCase.ExpectedRoute == assistant.RouteUnsupported && evaluationCase.ExpectedArgumentMode == assistant.ArgumentsNotApplicable {
		if evaluationCase.ExpectedDecision != DecisionUnsupported {
			return errors.New("unsupported route must be unsupported")
		}
		return nil
	}
	if evaluationCase.ExpectedDecision != DecisionClarify {
		return errors.New("non-executable combination must clarify")
	}
	return nil
}
