// S1.15 fixed saved P6 reader. No provider, meter or database clients.
package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/QahNaaRlN/grace/internal/pdfextract"
	"github.com/QahNaaRlN/grace/internal/statementdata"
	"github.com/QahNaaRlN/grace/internal/statementqualification"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"syscall"
	"time"
	"unicode/utf8"
)

const root = "/opt/grace/shared"
const basename = "statement-comparison-s114-comparison-20261021"
const limit = 524288

var unavailable = errors.New("diagnostic_unavailable")

type publicContext struct {
	Name        string                 `json:"name"`
	FixtureHash string                 `json:"fixtureHash"`
	Document    statementdata.Document `json:"document"`
}

// Validate bounds and duplicates before typed decoding, including Go's case-insensitive keys.
func strict(raw []byte, out any) error {
	if len(raw) > limit || !utf8.Valid(raw) {
		return unavailable
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	var walk func(int) error
	walk = func(depth int) error {
		if depth > 20 {
			return unavailable
		}
		t, e := dec.Token()
		if e != nil {
			return unavailable
		}
		if s, ok := t.(string); ok && len(s) > 10000 {
			return unavailable
		}
		if d, ok := t.(json.Delim); ok {
			n := 0
			seen := map[string]bool{}
			for dec.More() {
				n++
				if n > 5000 {
					return unavailable
				}
				if d == '{' {
					k, e := dec.Token()
					s, ok := k.(string)
					if e != nil || !ok || len(s) > 128 || seen[strings.ToLower(s)] {
						return unavailable
					}
					seen[strings.ToLower(s)] = true
				}
				if e := walk(depth + 1); e != nil {
					return unavailable
				}
			}
			end, e := dec.Token()
			if e != nil || !(d == '{' && end == json.Delim('}') || d == '[' && end == json.Delim(']')) {
				return unavailable
			}
		}
		return nil
	}
	if walk(0) != nil {
		return unavailable
	}
	if _, e := dec.Token(); e != io.EOF {
		return unavailable
	}
	dec = json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	var value any
	if dec.Decode(&value) != nil || !exactKeys(value, reflect.TypeOf(out).Elem()) {
		return unavailable
	}
	dec = json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if dec.Decode(out) != nil {
		return unavailable
	}
	return nil
}
func input(r io.Reader, out any) error {
	raw, e := io.ReadAll(io.LimitReader(r, limit+1))
	if e != nil {
		return unavailable
	}
	return strict(raw, out)
}

type snapshot struct {
	Data []byte
	Info os.FileInfo
	Hash [32]byte
}

func readPrivate(path string, max int) (snapshot, error) {
	fd, e := syscall.Open(path, syscall.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0)
	if e != nil {
		return snapshot{}, unavailable
	}
	f := os.NewFile(uintptr(fd), path)
	defer f.Close()
	st, e := f.Stat()
	if e != nil || !st.Mode().IsRegular() || st.Mode().Perm() != 0600 || st.Size() > int64(max) {
		return snapshot{}, unavailable
	}
	raw, e := io.ReadAll(io.LimitReader(f, int64(max+1)))
	if e != nil || len(raw) > max {
		return snapshot{}, unavailable
	}
	return snapshot{raw, st, sha256.Sum256(raw)}, nil
}
func unchanged(a, b snapshot) bool {
	return os.SameFile(a.Info, b.Info) && a.Info.Size() == b.Info.Size() && a.Info.Mode() == b.Info.Mode() && a.Info.ModTime() == b.Info.ModTime() && a.Hash == b.Hash
}

func samplesFor(c []publicContext) ([]statementqualification.Sample, error) {
	if len(c) != 3 {
		return nil, unavailable
	}
	samples := []statementqualification.Sample{}
	for i, n := range statementqualification.ScreeningNames() {
		if c[i].Name != n || c[i].FixtureHash != statementqualification.FixtureHash(n) || c[i].Document.Extractor != statementdata.ExtractorVersion || c[i].Document.Validate() != nil {
			return nil, unavailable
		}
		samples = append(samples, statementqualification.Sample{Name: n, Document: c[i].Document, SHA256: c[i].FixtureHash})
	}
	return samples, nil
}
func project(raw []byte, c []publicContext) (statementqualification.SavedDiagnosis, error) {
	var e statementqualification.ObservedComparisonReport
	if strict(raw, &e) != nil {
		return statementqualification.SavedDiagnosis{}, unavailable
	}
	samples, err := samplesFor(c)
	if err != nil {
		return statementqualification.SavedDiagnosis{}, unavailable
	}
	return statementqualification.DiagnoseSavedProposals(e, samples, time.Now().UTC())
}
func readReport(dir string, c []publicContext) (statementqualification.SavedDiagnosis, error) {
	return readReportUsing(dir, c, readPrivate)
}

// The production caller always supplies readPrivate; the seam lets tests mutate
// the real files between initial reads and final integrity checks deterministically.
func readReportUsing(dir string, c []publicContext, read func(string, int) (snapshot, error)) (statementqualification.SavedDiagnosis, error) {
	markerPath, reportPath := filepath.Join(dir, basename+".started"), filepath.Join(dir, basename+".json")
	marker, err := read(markerPath, 128)
	if err != nil || string(marker.Data) != statementqualification.SavedDiagnosisSource+"\n" {
		return statementqualification.SavedDiagnosis{}, unavailable
	}
	report, err := read(reportPath, limit)
	if err != nil {
		return statementqualification.SavedDiagnosis{}, unavailable
	}
	out, err := project(report.Data, c)
	if err != nil {
		return statementqualification.SavedDiagnosis{}, unavailable
	}
	ma, err := read(markerPath, 128)
	if err != nil || !unchanged(marker, ma) {
		return statementqualification.SavedDiagnosis{}, unavailable
	}
	ra, err := read(reportPath, limit)
	if err != nil || !unchanged(report, ra) {
		return statementqualification.SavedDiagnosis{}, unavailable
	}
	out.ArtifactsUnchanged = true
	return out, nil
}
func emit(out statementqualification.SavedDiagnosis) error {
	if statementqualification.ValidateSavedDiagnosis(out) != nil {
		return unavailable
	}
	raw, err := json.Marshal(out)
	if err != nil || len(raw)+1 > 65536 {
		return unavailable
	}
	_, err = os.Stdout.Write(append(raw, '\n'))
	return err
}
func run() error {
	if len(os.Args) == 4 && os.Args[1] == "prepare" {
		helper, err := pdfextract.NewClient(os.Args[3])
		if err != nil {
			return unavailable
		}
		ctx, cancel := context.WithTimeout(context.Background(), 40*time.Second)
		defer cancel()
		contexts := []publicContext{}
		for _, name := range statementqualification.ScreeningNames() {
			raw, err := os.ReadFile(filepath.Join(os.Args[2], name+".pdf"))
			if err != nil {
				return unavailable
			}
			hash := sha256.Sum256(raw)
			digest := hex.EncodeToString(hash[:])
			if digest != statementqualification.FixtureHash(name) {
				return unavailable
			}
			d, err := helper.Extract(ctx, raw)
			if err != nil {
				return unavailable
			}
			contexts = append(contexts, publicContext{name, digest, d})
		}
		if _, err := samplesFor(contexts); err != nil {
			return unavailable
		}
		raw, err := json.Marshal(contexts)
		if err != nil || len(raw) > limit {
			return unavailable
		}
		_, err = os.Stdout.Write(append(raw, '\n'))
		return err
	}
	if len(os.Args) == 2 && os.Args[1] == "validate" {
		raw, err := io.ReadAll(io.LimitReader(os.Stdin, 65537))
		if err != nil || len(raw) > 65536 {
			return unavailable
		}
		var out statementqualification.SavedDiagnosis
		if strict(raw, &out) != nil {
			return unavailable
		}
		return emit(out)
	}
	if len(os.Args) != 1 {
		return unavailable
	}
	var c []publicContext
	if input(os.Stdin, &c) != nil {
		return unavailable
	}
	out, err := readReport(root, c)
	if err != nil {
		return unavailable
	}
	return emit(out)
}
func main() {
	if run() != nil {
		fmt.Println(`{"error":"diagnostic_unavailable"}`)
		os.Exit(1)
	}
}
