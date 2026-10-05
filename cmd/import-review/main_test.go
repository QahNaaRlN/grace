package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCLIErrorDoesNotEchoPrivateDataAndCreatesNoOutput(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://must-not-connect")
	t.Setenv("TYPESAFE_API_KEY", "must-not-use")
	root := t.TempDir()
	input := filepath.Join(root, "input.json")
	if e := os.WriteFile(input, []byte(`{"private":"sensitive-source-marker"}`), 0600); e != nil {
		t.Fatal(e)
	}
	var stdout, stderr bytes.Buffer
	output := filepath.Join(root, "result")
	if code := run([]string{"--input", input, "--output-dir", output}, &stdout, &stderr); code != 1 {
		t.Fatal("invalid snapshot accepted")
	}
	if stdout.Len() != 0 || strings.Contains(stderr.String(), "sensitive-source-marker") || strings.Contains(stderr.String(), input) {
		t.Fatal("sensitive error output")
	}
	if _, e := os.Stat(output); !os.IsNotExist(e) {
		t.Fatal("output created for invalid input")
	}
}
func TestCLISyntheticSnapshotRunsOffline(t *testing.T) {
	t.Setenv("HTTP_PROXY", "http://127.0.0.1:1")
	t.Setenv("HTTPS_PROXY", "http://127.0.0.1:1")
	t.Setenv("DATABASE_URL", "postgres://127.0.0.1:1/no-db")
	root := t.TempDir()
	var stdout, stderr bytes.Buffer
	input := "../../testdata/import-review/snapshot.json"
	output := filepath.Join(root, "result")
	if code := run([]string{"--input", input, "--output-dir", output}, &stdout, &stderr); code != 0 {
		t.Fatalf("code=%d stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"published":false`) || strings.Contains(stdout.String(), "synthetic purchase") {
		t.Fatal("CLI summary is not bounded")
	}
	if code := run([]string{"--input", input, "--output-dir", output}, &stdout, &stderr); code != 1 {
		t.Fatal("existing result overwritten")
	}
}
func TestCLIInvalidArguments(t *testing.T) {
	var out bytes.Buffer
	if run([]string{"--unknown", "private-marker"}, &out, &out) != 2 || strings.Contains(out.String(), "private-marker") {
		t.Fatal("argument failure leaked input")
	}
}
