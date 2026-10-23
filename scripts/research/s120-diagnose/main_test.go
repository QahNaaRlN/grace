package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/QahNaaRlN/grace/internal/statementdata"
	q "github.com/QahNaaRlN/grace/internal/statementqualification"
)

var fixtureDir string

func TestMain(m *testing.M) {
	readerSource = strings.Repeat("1", 40)
	dir, e := os.MkdirTemp("", "grace-s120-synthetic-")
	if e != nil {
		os.Exit(2)
	}
	fixtureDir = dir
	cmd := exec.Command("go", "test", "./internal/statementqualification", "-run", "^TestClockDiagnosisActualPipeline$", "-count=1")
	cmd.Dir = "../../.."
	cmd.Env = append(os.Environ(), "TEST_S120_OUTPUT="+dir)
	if b, e := cmd.CombinedOutput(); e != nil {
		os.Stderr.Write(b)
		os.RemoveAll(dir)
		os.Exit(2)
	}
	code := m.Run()
	os.RemoveAll(dir)
	os.Exit(code)
}
func fixture(t *testing.T) ([]byte, []byte, statementdata.Document) {
	t.Helper()
	r, e := os.ReadFile(filepath.Join(fixtureDir, "report.json"))
	if e != nil {
		t.Fatal(e)
	}
	c, e := os.ReadFile(filepath.Join(fixtureDir, "context.json"))
	if e != nil {
		t.Fatal(e)
	}
	d, e := publicDocument(c)
	if e != nil {
		t.Fatal(e)
	}
	return r, c, d
}
func files(t *testing.T, raw []byte) (string, string, string) {
	t.Helper()
	dir := t.TempDir()
	mp, rp := filepath.Join(dir, basename+".started"), filepath.Join(dir, basename+".json")
	if os.WriteFile(mp, []byte(q.ClockProducerSource+"\n"), 0600) != nil || os.WriteFile(rp, raw, 0600) != nil {
		t.Fatal("files")
	}
	return dir, mp, rp
}
func TestS120FixedStableReader(t *testing.T) {
	raw, _, d := fixture(t)
	dir, mp, rp := files(t, raw)
	m, _ := readPrivate(mp, 41)
	r, _ := readPrivate(rp, q.ClockPilotReportLimit)
	out, e := readReport(dir, d, readPrivate)
	if e != nil || out.Status != "VERIFIED" || out.Accounting.ActualMicros != 60 {
		t.Fatal(e)
	}
	ma, _ := readPrivate(mp, 41)
	ra, _ := readPrivate(rp, q.ClockPilotReportLimit)
	if !unchanged(m, ma) || !unchanged(r, ra) {
		t.Fatal("read modified files")
	}
	b, _ := json.Marshal(out)
	if bytes.Contains(b, []byte("PRIVATE")) {
		t.Fatal("leak")
	}
	for _, kind := range []string{"missing", "symlink", "fifo", "directory", "mode", "oversize", "marker_source", "marker_oversize", "partial"} {
		t.Run(kind, func(t *testing.T) {
			dir, mp, rp := files(t, raw)
			switch kind {
			case "missing":
				os.Remove(rp)
			case "symlink":
				os.Remove(rp)
				os.Symlink(mp, rp)
			case "fifo":
				os.Remove(rp)
				if syscall.Mkfifo(rp, 0600) != nil {
					t.Fatal("fifo")
				}
			case "directory":
				os.Remove(rp)
				os.Mkdir(rp, 0600)
			case "mode":
				os.Chmod(rp, 0644)
			case "oversize":
				os.WriteFile(rp, bytes.Repeat([]byte("x"), q.ClockPilotReportLimit+1), 0600)
			case "marker_source":
				os.WriteFile(mp, []byte(strings.Repeat("f", 40)+"\n"), 0600)
			case "marker_oversize":
				os.WriteFile(mp, []byte(strings.Repeat("f", 42)), 0600)
			case "partial":
				os.WriteFile(rp, raw[:len(raw)/2], 0600)
			}
			start := time.Now()
			if _, e := readReport(dir, d, readPrivate); e == nil {
				t.Fatal("unsafe read")
			}
			if time.Since(start) > time.Second {
				t.Fatal("blocked special file")
			}
		})
	}
}
func TestS120DetectsMutationAndReplacement(t *testing.T) {
	raw, _, d := fixture(t)
	for _, kind := range []string{"bytes", "mtime", "mode", "replace", "symlink", "marker", "remove"} {
		t.Run(kind, func(t *testing.T) {
			dir, mp, rp := files(t, raw)
			calls := 0
			read := func(p string, n int) (snapshot, error) {
				s, e := readPrivate(p, n)
				calls++
				if calls == 2 {
					var mutation error
					switch kind {
					case "bytes":
						mutation = os.WriteFile(rp, append(append([]byte{}, raw...), ' '), 0600)
					case "mtime":
						mutation = os.Chtimes(rp, time.Unix(100, 0), time.Unix(100, 0))
					case "mode":
						mutation = os.Chmod(rp, 0644)
					case "replace":
						next := rp + ".next"
						mutation = os.WriteFile(next, raw, 0600)
						if mutation == nil {
							mutation = os.Rename(next, rp)
						}
					case "symlink":
						mutation = os.Remove(rp)
						if mutation == nil {
							mutation = os.Symlink(mp, rp)
						}
					case "marker":
						mutation = os.WriteFile(mp, []byte(strings.Repeat("f", 40)+"\n"), 0600)
					case "remove":
						mutation = os.Remove(rp)
					}
					if mutation != nil {
						t.Fatal(mutation)
					}
				}
				return s, e
			}
			if _, e := readReport(dir, d, read); e == nil || q.ClockDiagnosisReason(e) != "snapshot_changed" {
				t.Fatal("unstable snapshot accepted", e)
			}
		})
	}
}
func TestS120DefaultAuthorizationAndClosedErrors(t *testing.T) {
	_, c, _ := fixture(t)
	calls := 0
	read := func(statementdata.Document) (q.ClockDiagnosis, error) {
		calls++
		return q.ClockDiagnosis{}, errors.New("snapshot_unavailable")
	}
	check, e := run(nil, strings.NewReader("PRIVATE"), read)
	if e != nil || calls != 0 || bytes.Contains(check, []byte("PRIVATE")) {
		t.Fatal("default touched report")
	}
	for _, args := range [][]string{{"-mode=read"}, {"-mode=read", "-authorized-source=" + strings.Repeat("f", 40)}, {"-path=/tmp/PRIVATE"}, {"-mode=validate"}, {"-producer-source=x"}} {
		if _, e := run(args, bytes.NewReader(c), read); e == nil || calls != 0 {
			t.Fatal("unauthorized private read")
		}
	}
	args := []string{"-mode=read", "-authorized-source=" + readerSource}
	if _, e := run(args, strings.NewReader("PRIVATE"), read); q.ClockDiagnosisReason(e) != "context_mismatch" || calls != 0 {
		t.Fatal("untrusted context")
	}
	if _, e := run(args, bytes.NewReader(c), read); q.ClockDiagnosisReason(e) != "snapshot_unavailable" || calls != 1 {
		t.Fatal("cause lost")
	}
	failureBytes := failure(errors.New("snapshot_changed"))
	if q.ValidateClockOutput(failureBytes, readerSource) != nil {
		t.Fatal("failure shape")
	}
	validated, e := run([]string{"-mode=validate", "-authorized-source=" + readerSource}, bytes.NewReader(failureBytes), read)
	if e != nil || !bytes.Equal(validated, failureBytes) || calls != 1 {
		t.Fatal("validate touched private files")
	}
	for _, err := range []error{errors.New("PRIVATE"), errors.New("snapshot_unavailable"), errors.New("output_mismatch")} {
		b := failure(err)
		if bytes.Contains(b, []byte("PRIVATE")) || q.ValidateClockOutput(b, readerSource) != nil {
			t.Fatal("failure leaked")
		}
	}
	if _, e := run(args, strings.NewReader(strings.Repeat("x", contextLimit+1)), read); q.ClockDiagnosisReason(e) != "diagnosis_limit_exceeded" {
		t.Fatal("input bound")
	}
}
