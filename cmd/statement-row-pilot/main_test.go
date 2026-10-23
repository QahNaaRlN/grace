package main

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestRowPilotCLI(t *testing.T) {
	bin := filepath.Join(t.TempDir(), "row-pilot")
	sha := strings.Repeat("a", 40)
	build := exec.Command("go", "build", "-ldflags", "-X main.sourceSHA="+sha, "-o", bin, ".")
	if out, e := build.CombinedOutput(); e != nil {
		t.Fatal(e, string(out))
	}
	for _, mode := range []string{"default", "wrong_source", "no_execute", "unknown_mode", "args", "unavailable", "private", "refusal"} {
		t.Run(mode, func(t *testing.T) {
			args := []string{}
			input := ""
			switch mode {
			case "wrong_source":
				args = []string{"-mode=run", "-authorized-source=" + strings.Repeat("b", 40), "-execute-approved-pilot=true"}
			case "no_execute":
				args = []string{"-mode=run", "-authorized-source=" + sha}
			case "unknown_mode":
				args = []string{"-mode=unknown", "-authorized-source=" + sha}
			case "args":
				args = []string{"-PRIVATE"}
			case "unavailable", "private", "refusal":
				args = []string{"-mode=validate-public", "-authorized-source=" + sha}
				if mode == "private" {
					input = `{"PRIVATE":"raw model"}`
				}
				if mode == "refusal" {
					input = `{"error":"row_pilot_operator_owner_unavailable"}`
				}
			}
			c := exec.Command(bin, args...)
			c.Dir = t.TempDir()
			c.Env = append(os.Environ(), "AI_ENABLED=PRIVATE_INVALID", "OPENROUTER_API_KEY=", "DATABASE_URL=invalid://PRIVATE")
			c.Stdin = strings.NewReader(input)
			var stderr bytes.Buffer
			c.Stderr = &stderr
			out, e := c.Output()
			if stderr.Len() != 0 || bytes.Contains(out, []byte("PRIVATE")) {
				t.Fatal("private diagnostic", string(out), stderr.String())
			}
			var v map[string]any
			if json.Unmarshal(out, &v) != nil {
				t.Fatal("not closed JSON", string(out))
			}
			if mode == "default" {
				if e != nil || v["executionAuthorized"] != false || v["maximumPOST"] != float64(12) || v["maximumMetadataGET"] != float64(39) {
					t.Fatal(v, e)
				}
				return
			}
			if e == nil {
				t.Fatal("refusal exited success")
			}
			if mode == "no_execute" && v["error"] != "row_pilot_authorization_missing" {
				t.Fatal("read fixture/config before flag", v)
			}
			if mode == "private" || mode == "unavailable" || mode == "refusal" {
				if v["recordedCalls"] != nil || v["actualMicros"] != nil || v["settlementRecorded"] != nil || v["decision"] != "INCONCLUSIVE" || v["replayVerified"] != false {
					t.Fatal("unknown state", v)
				}
			}
		})
	}
}
