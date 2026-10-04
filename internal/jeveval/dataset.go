package jeveval

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
)

const (
	minimumRussianPerTool   = 5
	minimumRussianPerPeriod = 5
	minimumEnglishPerTool   = 1
)

var sensitivePatterns = []struct {
	name    string
	pattern *regexp.Regexp
}{
	{name: "email", pattern: regexp.MustCompile(`(?i)\b[A-Z0-9._%+-]+@[A-Z0-9.-]+\.[A-Z]{2,}\b`)},
	{name: "long numeric identifier", pattern: regexp.MustCompile(`\b\d{9,}\b`)},
	{name: "authorization header", pattern: regexp.MustCompile(`(?i)authorization\s*:`)},
	{name: "API key", pattern: regexp.MustCompile(`(?i)typesafe_api_key|bearer\s+[a-z0-9._-]+`)},
	{name: "Russian tax identifier", pattern: regexp.MustCompile(`(?i)\bинн\b`)},
	{name: "fiscal identifier", pattern: regexp.MustCompile(`(?i)фискальн(?:ый|ого|ому|ым|ом|ая|ой|ую|ые|ых)\s+(?:номер|накопитель|документ)`)},
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
	if len(dataset.Cases) == 0 {
		return errors.New("dataset must contain cases")
	}

	seen := make(map[string]struct{}, len(dataset.Cases))
	toolCounts := map[string]map[Tool]int{"ru": {}, "en": {}}
	periodCounts := make(map[PeriodShape]int)
	for index, evaluationCase := range dataset.Cases {
		prefix := fmt.Sprintf("case %d", index+1)
		if strings.TrimSpace(evaluationCase.ID) == "" {
			return fmt.Errorf("%s: id is required", prefix)
		}
		if _, exists := seen[evaluationCase.ID]; exists {
			return fmt.Errorf("%s: duplicate id %q", prefix, evaluationCase.ID)
		}
		seen[evaluationCase.ID] = struct{}{}
		prefix = fmt.Sprintf("case %q", evaluationCase.ID)

		if evaluationCase.Language != "ru" && evaluationCase.Language != "en" {
			return fmt.Errorf("%s: language must be ru or en", prefix)
		}
		if strings.TrimSpace(evaluationCase.Utterance) == "" {
			return fmt.Errorf("%s: utterance is required", prefix)
		}
		for _, sensitive := range sensitivePatterns {
			if sensitive.pattern.MatchString(evaluationCase.Utterance) {
				return fmt.Errorf("%s: utterance contains prohibited %s pattern", prefix, sensitive.name)
			}
		}
		if evaluationCase.ReferenceDate != "" {
			if _, err := time.Parse("2006-01-02", evaluationCase.ReferenceDate); err != nil {
				return fmt.Errorf("%s: referenceDate must use YYYY-MM-DD", prefix)
			}
		}
		if _, ok := validTools[evaluationCase.ExpectedTool]; !ok {
			return fmt.Errorf("%s: unknown expectedTool %q", prefix, evaluationCase.ExpectedTool)
		}
		if _, ok := validPeriodShapes[evaluationCase.ExpectedPeriodShape]; !ok {
			return fmt.Errorf("%s: unknown expectedPeriodShape %q", prefix, evaluationCase.ExpectedPeriodShape)
		}
		if _, ok := validArgumentPresence[evaluationCase.ExpectedArgumentPresence]; !ok {
			return fmt.Errorf("%s: unknown expectedArgumentPresence %q", prefix, evaluationCase.ExpectedArgumentPresence)
		}
		if len(evaluationCase.Tags) == 0 {
			return fmt.Errorf("%s: at least one tag is required", prefix)
		}
		if err := validateExpectedCombination(evaluationCase); err != nil {
			return fmt.Errorf("%s: %w", prefix, err)
		}
		toolCounts[evaluationCase.Language][evaluationCase.ExpectedTool]++
		if evaluationCase.Language == "ru" {
			periodCounts[evaluationCase.ExpectedPeriodShape]++
		}
	}

	for tool := range validTools {
		if toolCounts["ru"][tool] < minimumRussianPerTool {
			return fmt.Errorf("dataset needs at least %d Russian cases for %s", minimumRussianPerTool, tool)
		}
		if toolCounts["en"][tool] < minimumEnglishPerTool {
			return fmt.Errorf("dataset needs at least %d English case for %s", minimumEnglishPerTool, tool)
		}
	}
	for shape := range validPeriodShapes {
		if periodCounts[shape] < minimumRussianPerPeriod {
			return fmt.Errorf("dataset needs at least %d Russian cases for period shape %s", minimumRussianPerPeriod, shape)
		}
	}
	return nil
}

func validateExpectedCombination(evaluationCase EvaluationCase) error {
	periodSensitive := evaluationCase.ExpectedTool == ToolGetPeriodSummary || evaluationCase.ExpectedTool == ToolGetCashPosition
	if periodSensitive {
		if evaluationCase.ExpectedPeriodShape == PeriodMissing || evaluationCase.ExpectedPeriodShape == PeriodNotApplicable {
			return errors.New("executable period-sensitive route must name a usable period")
		}
		if evaluationCase.ExpectedArgumentPresence != ArgumentsSufficient {
			return errors.New("executable period-sensitive route must have sufficient arguments")
		}
		return nil
	}
	if evaluationCase.ExpectedTool == ToolClarify {
		if evaluationCase.ExpectedPeriodShape != PeriodMissing || evaluationCase.ExpectedArgumentPresence != ArgumentsMissing {
			return errors.New("clarify cases must represent a missing period and missing arguments")
		}
		return nil
	}
	if evaluationCase.ExpectedPeriodShape != PeriodNotApplicable || evaluationCase.ExpectedArgumentPresence != ArgumentsNotApplicable {
		return errors.New("non-period route must use not_applicable period and arguments")
	}
	return nil
}
