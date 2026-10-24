package main

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/QahNaaRlN/grace/internal/statementextract"
	"github.com/QahNaaRlN/grace/internal/statementqualification"
)

const testSource = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

func commandCorpus(t *testing.T) statementqualification.DirectCorpus {
	t.Helper()
	c, e := statementqualification.LoadDirectCorpus(filepath.Join("..", ".."))
	if e != nil {
		t.Fatal(e)
	}
	return c
}
func configured(t *testing.T) ([]string, string, statementqualification.DirectCorpus) {
	t.Helper()
	sourceSHA = testSource
	t.Cleanup(func() { sourceSHA = "UNBOUND" })
	c := commandCorpus(t)
	path := filepath.Join(t.TempDir(), "allowance.json")
	a := allowance{Source: testSource, Approval: "offline-fake", ManifestSHA256: c.ManifestSHA256, CeilingMicros: statementextract.DirectCampaignCeilingMicros, State: "available", Endpoint: "https://api.openai.com/v1/responses", PricingConfirmed: true, FundingConfirmed: true, AccountModelAccessConfirmed: true, RegionPrivacyConfirmed: true, NoAdditionalFeesConfirmed: true}
	raw, _ := json.Marshal(a)
	if e := os.WriteFile(path, raw, 0600); e != nil {
		t.Fatal(e)
	}
	return []string{"--mode", "run", "--root", "../..", "--execute-approved-reading", "--authorized-source", testSource, "--approval-id", "offline-fake", "--allowance", path}, path, c
}
func fakeWire(r statementextract.DirectReading) []byte {
	answer, _ := json.Marshal(r)
	wire, _ := json.Marshal(map[string]any{"model": statementextract.DirectModel, "status": "completed", "service_tier": "default", "output": []any{map[string]any{"type": "message", "role": "assistant", "status": "completed", "content": []any{map[string]any{"type": "output_text", "text": string(answer)}}}}, "usage": map[string]int{"input_tokens": 1000, "output_tokens": 2000, "total_tokens": 3000}})
	return wire
}
func neverDependencies(t *testing.T) dependencies {
	t.Helper()
	return dependencies{key: func(string) string { t.Fatal("key read before gate"); return "" }, call: func(context.Context, statementextract.DirectRequest, []string, string) statementextract.DirectObservation {
		t.Fatal("external call before gate")
		return statementextract.DirectObservation{}
	}}
}
func TestCheckWithHostileEnvironmentNoKeysOrCommands(t *testing.T) {
	t.Setenv("HOME", "/does-not-exist/s122")
	t.Setenv(keyEnvironment, "FORBIDDEN_REAL_ENV_READ")
	spy := t.TempDir()
	mark := filepath.Join(spy, "invoked")
	for _, name := range []string{"ssh", "docker", "curl", "git", "python3", "env", "node"} {
		if e := os.WriteFile(filepath.Join(spy, name), []byte("#!/bin/sh\necho invoked > '"+mark+"'\nexit 99\n"), 0700); e != nil {
			t.Fatal(e)
		}
	}
	t.Setenv("PATH", spy)
	var b bytes.Buffer
	if e := command([]string{"--root", "../.."}, &b, neverDependencies(t)); e != nil {
		t.Fatal(e)
	}
	if !strings.Contains(b.String(), `"calls":0`) || strings.Contains(b.String(), "FORBIDDEN_REAL_ENV_READ") {
		t.Fatal(b.String())
	}
	if _, e := os.Stat(mark); !os.IsNotExist(e) {
		t.Fatal("external command invoked")
	}
}
func TestRunGateRefusalsConsumeWithoutKey(t *testing.T) {
	for _, kind := range []string{"execute", "source", "approval", "funding", "manifest", "symlink", "permissions"} {
		t.Run(kind, func(t *testing.T) {
			args, path, _ := configured(t)
			switch kind {
			case "execute":
				args[4] = "--execute-approved-reading=false"
			case "source":
				args[6] = strings.Repeat("b", 40)
			case "approval":
				args[8] = "invalid approval"
			case "funding":
				raw, _ := os.ReadFile(path)
				raw = bytes.Replace(raw, []byte(`"fundingConfirmed":true`), []byte(`"fundingConfirmed":false`), 1)
				os.WriteFile(path, raw, 0600)
			case "manifest":
				raw, _ := os.ReadFile(path)
				raw = bytes.Replace(raw, []byte(`"manifestSHA256":"`), []byte(`"manifestSHA256":"wrong`), 1)
				os.WriteFile(path, raw, 0600)
			case "permissions":
				os.Chmod(path, 0644)
			case "symlink":
				target := path + ".real"
				os.Rename(path, target)
				os.Symlink(target, path)
			}
			var b bytes.Buffer
			if e := command(args, &b, neverDependencies(t)); e == nil {
				t.Fatal("refusal missing")
			}
			if _, e := os.Stat(path + ".consumed"); e != nil {
				t.Fatal("refusal did not consume", e)
			}
			if e := command(args, &b, neverDependencies(t)); e == nil || e.Error() != "one_shot_unavailable" {
				t.Fatalf("retry possible %v", e)
			}
		})
	}
}
func TestFakeCampaignReservationReplayAndNoRetry(t *testing.T) {
	for _, kind := range []string{"positive", "semantic", "unknown_usage", "timeout", "missing_key"} {
		t.Run(kind, func(t *testing.T) {
			args, path, c := configured(t)
			var keys, calls atomic.Int32
			deps := dependencies{key: func(name string) string {
				keys.Add(1)
				if name != keyEnvironment {
					t.Fatal("unexpected key name")
				}
				raw, e := os.ReadFile(path + ".consumed")
				if e != nil || !bytes.Contains(raw, []byte(`"state": "reserved"`)) || !bytes.Contains(raw, []byte(`"reservedMicros": 10400000`)) {
					t.Fatal("key read before durable full reservation", e, string(raw))
				}
				if kind == "missing_key" {
					return ""
				}
				return "FAKE_NEVER_A_REAL_KEY"
			}, call: func(ctx context.Context, r statementextract.DirectRequest, models []string, key string) statementextract.DirectObservation {
				i := int(calls.Add(1)) - 1
				if key != "FAKE_NEVER_A_REAL_KEY" {
					t.Fatal("fake key")
				}
				if _, ok := ctx.Deadline(); !ok {
					t.Fatal("campaign deadline missing")
				}
				// A second process must fail even while the first campaign is running.
				var b bytes.Buffer
				if e := command(args, &b, neverDependencies(t)); e == nil || e.Error() != "one_shot_unavailable" {
					t.Fatal("concurrent one-shot not enforced", e)
				}
				reading := c.Cases[i%4].Golden
				if kind == "semantic" {
					raw, _ := json.Marshal(reading)
					reading, _ = statementextract.DecodeDirectReading(raw)
					reading.Rows[0].Fields["accountAmount"].Value = "-1234.57"
				}
				if kind == "timeout" {
					return statementextract.DirectObservation{State: "inconclusive", Reason: "transport_unknown_usage", DurationMS: 180000}
				}
				raw := fakeWire(reading)
				if kind == "unknown_usage" {
					var m map[string]any
					json.Unmarshal(raw, &m)
					delete(m, "usage")
					raw, _ = json.Marshal(m)
				}
				o := statementextract.ObserveDirectResponse(raw, r, models)
				o.Response = raw
				o.DurationMS = 123
				return o
			}}
			var b bytes.Buffer
			e := command(args, &b, deps)
			raw, readErr := os.ReadFile(path + ".consumed")
			if readErr != nil {
				t.Fatal(readErr)
			}
			if bytes.Contains(raw, []byte("FAKE_NEVER_A_REAL_KEY")) {
				t.Fatal("key leaked into receipt")
			}
			var rec receipt
			if json.Unmarshal(raw, &rec) != nil {
				t.Fatal("receipt JSON")
			}
			if keys.Load() != 1 {
				t.Fatal("key fetch count")
			}
			if kind == "missing_key" {
				if e == nil || calls.Load() != 0 || rec.State != "refused_before_calls" {
					t.Fatal("missing key gate")
				}
				return
			}
			if e != nil {
				t.Fatal(e)
			}
			wantCalls := int32(1)
			if kind == "positive" {
				wantCalls = 8
			}
			if calls.Load() != wantCalls {
				t.Fatalf("calls=%d", calls.Load())
			}
			wantDecision := "INCONCLUSIVE"
			if kind == "positive" {
				wantDecision = "KEEP_FOR_FURTHER_REVIEW"
			}
			if kind == "semantic" {
				wantDecision = "REJECT"
			}
			if rec.Report == nil || rec.Report.Decision != wantDecision || rec.Report.ImportReady {
				t.Fatal("decision/guard")
			}
			if (kind == "unknown_usage" || kind == "timeout") && rec.State != "uncertain" {
				t.Fatal("unknown cost released")
			}
			reportRaw, _ := json.Marshal(rec.Report)
			if _, e = statementqualification.VerifyDirectReport(c, reportRaw, testSource, "offline-fake"); e != nil {
				t.Fatal("saved independent replay", e)
			}
			if e = command(args, &b, neverDependencies(t)); e == nil || e.Error() != "one_shot_unavailable" {
				t.Fatal("repeat allowed")
			}
		})
	}
}
