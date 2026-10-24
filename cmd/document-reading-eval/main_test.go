package main

import (
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"github.com/QahNaaRlN/grace/internal/assistant"
	q "github.com/QahNaaRlN/grace/internal/documentqualification"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestQDefaultAndRefusalsNeverReadSecretsOrCallRun(t *testing.T) {
	keyReads, runs := 0, 0
	deps := dependencies{executable: func() (string, error) { return "/tmp/unapproved/reader", nil }, credentials: func() (credentials, error) { keyReads++; return credentials{}, errors.New("unavailable") }, run: func(context.Context, q.Corpus, q.Manifest, assistant.ExperimentGrant, credentials) (q.Report, error) {
		runs++
		return q.Report{}, nil
	}}
	var out bytes.Buffer
	if e := command([]string{"-root=../.."}, strings.NewReader(""), &out, deps); e != nil || !strings.Contains(out.String(), `"candidate":null`) {
		t.Fatal(e, out.String())
	}
	old := sourceSHA
	sourceSHA = strings.Repeat("a", 40)
	defer func() { sourceSHA = old }()
	for _, args := range [][]string{{"-mode=run"}, {"-mode=run", "-authorized-source=" + sourceSHA}, {"-mode=run", "-authorized-source=" + strings.Repeat("b", 40), "-execute-approved-reading=true"}, {"-mode=run", "-authorized-source=" + sourceSHA, "-execute-approved-reading=true", "-approval-id=" + strings.Repeat("b", 40)}, {"-mode=invalid"}} {
		out.Reset()
		if e := command(args, strings.NewReader(""), &out, deps); e == nil {
			t.Fatal("unsafe arguments accepted", args)
		}
	}
	if keyReads != 0 || runs != 0 {
		t.Fatal("default/refusal touched execution", keyReads, runs)
	}
}
func TestQUnselectedCandidateAndManifestMismatchRefuseBeforeSecrets(t *testing.T) {
	c, e := q.Load("../..")
	if e != nil {
		t.Fatal(e)
	}
	unique := sha1.Sum([]byte(t.TempDir()))
	old := sourceSHA
	sourceSHA = hex.EncodeToString(unique[:])
	defer func() { sourceSHA = old }()
	dir := "/tmp/grace-document-reading-q-" + sourceSHA
	if e = q.WriteBundle(dir, c, q.NewManifest(c, sourceSHA, nil)); e != nil {
		t.Fatal(e)
	}
	defer os.RemoveAll(dir)
	calls := 0
	deps := dependencies{executable: func() (string, error) { return filepath.Join(dir, "reader"), nil }, credentials: func() (credentials, error) { calls++; return credentials{}, nil }}
	args := []string{"-mode=run", "-authorized-source=" + sourceSHA, "-execute-approved-reading=true", "-approval-id=" + strings.Repeat("b", 40), "-approved-manifest=" + strings.Repeat("0", 64)}
	if e = command(args, strings.NewReader(""), &bytes.Buffer{}, deps); e == nil || calls != 0 {
		t.Fatal("candidate/manifest gate touched credentials", e, calls)
	}
}
