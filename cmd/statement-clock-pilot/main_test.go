package main

import (
	"flag"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestClockCLIHelper(t *testing.T) {
	if os.Getenv("CLOCK_TEST_CHILD") != "1" {
		return
	}
	sourceSHA = os.Getenv("CLOCK_TEST_SOURCE")
	flag.CommandLine = flag.NewFlagSet("clock-test", flag.ExitOnError)
	args := []string{"clock"}
	args = append(args, strings.Fields(os.Getenv("CLOCK_TEST_ARGS"))...)
	os.Args = args
	main()
	os.Exit(0)
}
func TestClockCLIClosedRefusalAndDefault(t *testing.T) {
	sha := strings.Repeat("b", 40)
	dir := "/tmp/grace-statement-clock-s119-" + sha
	if e := os.Mkdir(dir, 0700); e != nil {
		t.Fatal(e)
	}
	defer os.RemoveAll(dir)
	raw, e := os.ReadFile("../../internal/statementextract/testdata/s119-cards-document.json")
	if e != nil {
		t.Fatal(e)
	}
	if os.WriteFile(filepath.Join(dir, "cards-document.json"), raw, 0600) != nil {
		t.Fatal("fixture")
	}
	for _, tc := range []struct {
		args, input, want string
		ok                bool
	}{
		{"", "", `"activationAuthorized":false`, true},
		{"-mode=run -authorized-source=wrong", "", "clock_source_not_authorized", false},
		{"-mode=summary -authorized-source=" + sha, `{"error":"clock_global_ai_unavailable"}`, "clock_global_ai_unavailable", false},
		{"-mode=summary -authorized-source=" + sha, `{"error":"PRIVATE"}`, "clock_report_invalid", false},
	} {
		cmd := exec.Command(os.Args[0], "-test.run=^TestClockCLIHelper$")
		cmd.Env = append(os.Environ(), "CLOCK_TEST_CHILD=1", "CLOCK_TEST_SOURCE="+sha, "CLOCK_TEST_ARGS="+tc.args)
		cmd.Stdin = strings.NewReader(tc.input)
		out, e := cmd.CombinedOutput()
		if (e == nil) != tc.ok || !strings.Contains(string(out), tc.want) || strings.Contains(string(out), "PRIVATE") {
			t.Fatal(tc.args, string(out), e)
		}
	}
}
