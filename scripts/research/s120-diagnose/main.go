// S1.20 fixed saved-report reader. No provider, SQL, config or helper clients.
package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"syscall"

	"github.com/QahNaaRlN/grace/internal/statementdata"
	q "github.com/QahNaaRlN/grace/internal/statementqualification"
)

// Set by the exact-source build; neither source nor paths have runtime overrides.
var readerSource string

const root = "/opt/grace/shared"
const basename = "statement-s119-clock-pilot"
const contextLimit = 32768

type snapshot struct {
	Data []byte
	Info os.FileInfo
	Hash [32]byte
}

func unchanged(a, b snapshot) bool {
	return os.SameFile(a.Info, b.Info) && a.Info.Size() == b.Info.Size() && a.Info.Mode() == b.Info.Mode() && a.Info.ModTime().Equal(b.Info.ModTime()) && a.Hash == b.Hash
}
func readPrivate(path string, limit int) (snapshot, error) {
	fail := errors.New("snapshot_unavailable")
	fd, e := syscall.Open(path, syscall.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_NONBLOCK|syscall.O_CLOEXEC, 0)
	if e != nil {
		return snapshot{}, fail
	}
	f := os.NewFile(uintptr(fd), path)
	defer f.Close()
	before, e := f.Stat()
	if e != nil || !before.Mode().IsRegular() || before.Mode() != 0600 || before.Size() < 0 || before.Size() > int64(limit) {
		return snapshot{}, fail
	}
	raw, e := io.ReadAll(io.LimitReader(f, int64(limit+1)))
	if e != nil || len(raw) > limit {
		return snapshot{}, fail
	}
	after, e := f.Stat()
	if e != nil || !os.SameFile(before, after) || before.Size() != after.Size() || before.Mode() != after.Mode() || !before.ModTime().Equal(after.ModTime()) || int64(len(raw)) != after.Size() {
		return snapshot{}, errors.New("snapshot_changed")
	}
	return snapshot{raw, after, sha256.Sum256(raw)}, nil
}
func publicDocument(raw []byte) (statementdata.Document, error) {
	var d statementdata.Document
	hash := sha256.Sum256(raw)
	if len(raw) > contextLimit || hex.EncodeToString(hash[:]) != q.ClockContextSHA256 || json.Unmarshal(raw, &d) != nil || d.Validate() != nil {
		return d, errors.New("context_mismatch")
	}
	return d, nil
}
func readReport(dir string, d statementdata.Document, read func(string, int) (snapshot, error)) (q.ClockDiagnosis, error) {
	fail := func(code string) (q.ClockDiagnosis, error) { return q.ClockDiagnosis{}, errors.New(code) }
	mp, rp := filepath.Join(dir, basename+".started"), filepath.Join(dir, basename+".json")
	m, e := read(mp, 41)
	if e != nil {
		return q.ClockDiagnosis{}, e
	}
	if string(m.Data) != q.ClockProducerSource+"\n" {
		return fail("source_mismatch")
	}
	r, e := read(rp, q.ClockPilotReportLimit)
	if e != nil {
		return q.ClockDiagnosis{}, e
	}
	out, e := q.DiagnoseClock(r.Data, d, readerSource)
	if e != nil {
		return q.ClockDiagnosis{}, e
	}
	ma, e := read(mp, 41)
	if e != nil || !unchanged(m, ma) {
		return fail("snapshot_changed")
	}
	ra, e := read(rp, q.ClockPilotReportLimit)
	if e != nil || !unchanged(r, ra) {
		return fail("snapshot_changed")
	}
	return out, nil
}
func failure(err error) []byte {
	b, _ := json.Marshal(q.ClockDiagnosisFailure{Version: q.ClockDiagnosisVersion, ReaderSource: readerSource, Status: "UNVERIFIED", Reason: q.ClockDiagnosisReason(err)})
	return append(b, '\n')
}
func input(r io.Reader, limit int) ([]byte, error) {
	b, e := io.ReadAll(io.LimitReader(r, int64(limit+1)))
	if e != nil || len(b) > limit {
		return nil, errors.New("diagnosis_limit_exceeded")
	}
	return b, nil
}

// read is injected only by tests; the production closure fixes root and reader.
func run(args []string, in io.Reader, read func(statementdata.Document) (q.ClockDiagnosis, error)) ([]byte, error) {
	flags := flag.NewFlagSet("s120", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	mode := flags.String("mode", "check", "")
	authorized := flags.String("authorized-source", "", "")
	if flags.Parse(args) != nil || flags.NArg() != 0 || !regexp.MustCompile(`^[0-9a-f]{40}$`).MatchString(readerSource) {
		return nil, errors.New("source_mismatch")
	}
	if *mode == "check" {
		if *authorized != "" {
			return nil, errors.New("source_mismatch")
		}
		b, e := json.Marshal(struct {
			Version                   string `json:"version"`
			ReaderSource              string `json:"readerSource"`
			ProducerSource            string `json:"producerSource"`
			ProducerRun               int64  `json:"producerRun"`
			ContextSHA256             string `json:"contextSha256"`
			ProviderCalls             int    `json:"providerCalls"`
			SQLCalls                  int    `json:"sqlCalls"`
			PersistentWritesRequested int    `json:"persistentWritesRequested"`
			ActivationAuthorized      bool   `json:"activationAuthorized"`
		}{"statement.clock-diagnosis.manifest.s120.v1", readerSource, q.ClockProducerSource, q.ClockProducerRun, q.ClockContextSHA256, 0, 0, 0, false})
		return append(b, '\n'), e
	}
	if *authorized != readerSource || (*mode != "read" && *mode != "validate") {
		return nil, errors.New("source_mismatch")
	}
	if *mode == "validate" {
		b, e := input(in, q.ClockDiagnosisLimit)
		if e != nil {
			return nil, e
		}
		if q.ValidateClockOutput(b, readerSource) != nil {
			return nil, errors.New("output_mismatch")
		}
		return b, nil
	}
	b, e := input(in, contextLimit)
	if e != nil {
		return nil, e
	}
	d, e := publicDocument(b)
	if e != nil {
		return nil, e
	}
	out, e := read(d)
	if e != nil {
		return nil, e
	}
	b, e = json.Marshal(out)
	if e != nil || len(b)+1 > q.ClockDiagnosisLimit {
		return nil, errors.New("diagnosis_limit_exceeded")
	}
	if q.ValidateClockOutput(b, readerSource) != nil {
		return nil, errors.New("output_mismatch")
	}
	return append(b, '\n'), nil
}
func main() {
	b, e := run(os.Args[1:], os.Stdin, func(d statementdata.Document) (q.ClockDiagnosis, error) { return readReport(root, d, readPrivate) })
	if e != nil {
		os.Stdout.Write(failure(e))
		os.Exit(1)
	}
	os.Stdout.Write(bytes.TrimSpace(b))
	os.Stdout.Write([]byte{'\n'})
}
