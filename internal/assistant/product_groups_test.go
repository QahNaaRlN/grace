package assistant

import "testing"

func TestProductGroupNoMatchThresholdAndEvidence(t *testing.T) {
	groups := map[string]string{"milk": "Молоко", "bread": "Хлеб"}
	answer := choiceAnswer{Type: "choice", Choice: "milk", Confidence: 0.93, Probabilities: map[string]float64{"milk": 0.96, "bread": 0.02, "unknown": 0.02}}
	result, err := ValidateProductGroup(answer, groups, 0.90)
	if err != nil || result == nil || result.Group != "milk" || result.QuestionVersion == "" {
		t.Fatal(result, err)
	}
	result, err = ValidateProductGroup(answer, groups, 0.95)
	if err != nil || result != nil {
		t.Fatal("threshold ignored", result, err)
	}
	answer.Choice = "unknown"
	answer.Probabilities = map[string]float64{"milk": 0.02, "bread": 0.02, "unknown": 0.96}
	result, err = ValidateProductGroup(answer, groups, 0.90)
	if err != nil || result != nil {
		t.Fatal("no-match guessed", result, err)
	}
	answer.Choice = "invented"
	if _, err = ValidateProductGroup(answer, groups, 0.9); err == nil {
		t.Fatal("invented group accepted")
	}
}
