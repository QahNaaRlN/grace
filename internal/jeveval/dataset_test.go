package jeveval

import (
	"strings"
	"testing"
)

const fixturePath = "../../testdata/jev-evaluation/v1/cases.json"

func TestFixtureDatasetIsValidAndComplete(t *testing.T) {
	dataset, err := LoadDataset(fixturePath)
	if err != nil {
		t.Fatalf("LoadDataset() error = %v", err)
	}
	if got, want := len(dataset.Cases), 49; got != want {
		t.Fatalf("case count = %d, want %d", got, want)
	}
}

func TestParseDatasetRejectsUnknownFields(t *testing.T) {
	_, err := ParseDataset([]byte(`{"version":"v1","cases":[],"unknown":true}`))
	if err == nil || !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("ParseDataset() error = %v, want unknown field", err)
	}
}

func TestValidateDatasetRejectsInvalidCases(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*Dataset)
		want   string
	}{
		{
			name: "duplicate id",
			mutate: func(dataset *Dataset) {
				dataset.Cases[1].ID = dataset.Cases[0].ID
			},
			want: "duplicate id",
		},
		{
			name: "empty utterance",
			mutate: func(dataset *Dataset) {
				dataset.Cases[0].Utterance = "  "
			},
			want: "utterance is required",
		},
		{
			name: "unknown tool",
			mutate: func(dataset *Dataset) {
				dataset.Cases[0].ExpectedTool = "invented_tool"
			},
			want: "unknown expectedTool",
		},
		{
			name: "invalid combination",
			mutate: func(dataset *Dataset) {
				dataset.Cases[0].ExpectedPeriodShape = PeriodCalendar
			},
			want: "non-period route",
		},
		{
			name: "sensitive long identifier",
			mutate: func(dataset *Dataset) {
				dataset.Cases[0].Utterance = "Счёт 12345678901234567890"
			},
			want: "prohibited long numeric identifier",
		},
		{
			name: "secret",
			mutate: func(dataset *Dataset) {
				dataset.Cases[0].Utterance = "Authorization: Bearer secret-value"
			},
			want: "prohibited authorization header",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			dataset, err := LoadDataset(fixturePath)
			if err != nil {
				t.Fatal(err)
			}
			test.mutate(&dataset)
			err = ValidateDataset(dataset)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("ValidateDataset() error = %v, want containing %q", err, test.want)
			}
		})
	}
}

func TestQuestionsAreVersionedClosedChoices(t *testing.T) {
	if QuestionSetVersion == "" {
		t.Fatal("QuestionSetVersion must not be empty")
	}
	questions := Questions()
	if got, want := len(questions), 3; got != want {
		t.Fatalf("question count = %d, want %d", got, want)
	}
	for id, question := range questions {
		if question.Type != "choice" {
			t.Errorf("question %s type = %q, want choice", id, question.Type)
		}
		if len(question.Criteria) < 3 {
			t.Errorf("question %s has too few criteria", id)
		}
	}
	for tool := range validTools {
		if _, ok := questions[questionTool].Criteria[string(tool)]; !ok {
			t.Errorf("tool question lacks %s", tool)
		}
	}
}
