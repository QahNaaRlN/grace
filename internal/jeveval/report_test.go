package jeveval

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteReportCreatesPrivateFileAndRefusesOverwrite(t *testing.T) {
	path := filepath.Join(t.TempDir(), "report.json")
	report := Report{DatasetVersion: "v1", QuestionSetVersion: "v1", RequestedModel: "jev-1.13.0"}
	if err := WriteReport(path, report); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Fatalf("mode = %o, want 600", got)
	}
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(content), "TYPESAFE_API_KEY") || strings.Contains(strings.ToLower(string(content)), "authorization") {
		t.Fatal("report contains a secret field")
	}
	if err := WriteReport(path, report); err == nil {
		t.Fatal("expected overwrite refusal")
	}
}
