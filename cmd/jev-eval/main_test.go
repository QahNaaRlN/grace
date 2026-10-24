package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/QahNaaRlN/grace/internal/jeveval"
)

func TestValidateCommandChecksFixtureOffline(t *testing.T) {
	if err := validateCommand([]string{"--dataset", "../../testdata/jev-evaluation/v1/cases.json"}); err != nil {
		t.Fatalf("validateCommand() error = %v", err)
	}
}

func TestRunCommandRejectsAliasBeforeProviderUse(t *testing.T) {
	err := runCommand([]string{
		"--dataset", "../../testdata/jev-evaluation/v1/cases.json",
		"--model", "jev-latest",
		"--report", filepath.Join(t.TempDir(), "report.json"),
	})
	if err == nil || !strings.Contains(err.Error(), "versioned") {
		t.Fatalf("runCommand() error = %v", err)
	}
}

func TestRetiredDirectRunRejectsBeforeCredentialRead(t *testing.T) {
	t.Setenv("GRACE_JEV_OFFLINE_FIXTURE_KEY", "")
	err := runCommand([]string{
		"--dataset", "../../testdata/jev-evaluation/v1/cases.json",
		"--model", "jev-1.13.0",
		"--report", filepath.Join(t.TempDir(), "report.json"),
	})
	if err == nil || !strings.Contains(err.Error(), "direct_typesafe_retired_use_openrouter") {
		t.Fatalf("runCommand() error = %v", err)
	}
}

func TestValidateReportPathRejectsRepositoryPath(t *testing.T) {
	path, err := filepath.Abs("report.json")
	if err != nil {
		t.Fatal(err)
	}
	if err := validateReportPath(path); err == nil {
		t.Fatal("expected repository path rejection")
	}
}

func TestRunCommandWritesReportAgainstFakeHTTPProvider(t *testing.T) {
	dataset, err := jeveval.LoadDataset("../../testdata/jev-evaluation/v1/cases.json")
	if err != nil {
		t.Fatal(err)
	}
	expectedByUtterance := make(map[string]jeveval.EvaluationCase, len(dataset.Cases))
	for _, evaluationCase := range dataset.Cases {
		expectedByUtterance[evaluationCase.Utterance] = evaluationCase
	}
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Header.Get("Authorization") != "Bearer test-key" {
			writer.WriteHeader(http.StatusUnauthorized)
			return
		}
		var providerRequest jeveval.Request
		if err := json.NewDecoder(request.Body).Decode(&providerRequest); err != nil {
			t.Error(err)
			writer.WriteHeader(http.StatusBadRequest)
			return
		}
		expected := expectedByUtterance[providerRequest.State.Utterance]
		choices := map[string]string{
			"tool": string(expected.ExpectedTool), "period_shape": string(expected.ExpectedPeriodShape),
			"argument_presence": string(expected.ExpectedArgumentPresence),
		}
		answers := make(map[string]jeveval.ChoiceAnswer, len(providerRequest.Questions))
		for id, question := range providerRequest.Questions {
			probabilities := make(map[string]float64, len(question.Criteria))
			for option := range question.Criteria {
				probabilities[option] = 0
			}
			probabilities[choices[id]] = 1
			answers[id] = jeveval.ChoiceAnswer{Type: "choice", Choice: choices[id], Probabilities: probabilities, Confidence: 0.9}
		}
		_ = json.NewEncoder(writer).Encode(jeveval.Response{
			Model: "jev-1.13.0", Answers: answers, Usage: jeveval.Usage{InputTokens: 100, OutputTokens: 20},
		})
	}))
	defer server.Close()

	t.Setenv("GRACE_JEV_OFFLINE_FIXTURE_KEY", "test-key")
	reportPath := filepath.Join(t.TempDir(), "report.json")
	err = runCommand([]string{
		"--dataset", "../../testdata/jev-evaluation/v1/cases.json",
		"--model", "jev-1.13.0", "--report", reportPath, "--endpoint", server.URL,
	})
	if err != nil {
		t.Fatalf("runCommand() error = %v", err)
	}
	content, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatal(err)
	}
	var report jeveval.Report
	if err := json.Unmarshal(content, &report); err != nil {
		t.Fatal(err)
	}
	if report.Metrics["ru"].ToolAccuracy != 1 || report.Metrics["en"].ToolAccuracy != 1 {
		t.Fatalf("report metrics = %#v", report.Metrics)
	}
	reanalyzedPath := filepath.Join(t.TempDir(), "reanalyzed.json")
	if err := analyzeCommand([]string{
		"--dataset", "../../testdata/jev-evaluation/v1/cases.json",
		"--input-report", reportPath, "--report", reanalyzedPath,
	}); err != nil {
		t.Fatalf("analyzeCommand() error = %v", err)
	}
	if _, err := os.Stat(reanalyzedPath); err != nil {
		t.Fatalf("reanalyzed report: %v", err)
	}
}
