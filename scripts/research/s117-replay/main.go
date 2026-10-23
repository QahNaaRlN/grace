// S1.17 offline artifact replay. Only pinned public PDFs and a local helper socket.
package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/QahNaaRlN/grace/internal/statementdata"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/QahNaaRlN/grace/internal/pdfextract"
	"github.com/QahNaaRlN/grace/internal/statementqualification"
)

func run(args []string) error {
	if len(args) != 3 || args[0] != "-" || !filepath.IsAbs(args[1]) {
		return fmt.Errorf("arguments")
	}
	f := os.Stdin
	raw, e := io.ReadAll(io.LimitReader(f, statementqualification.ComparisonArtifactLimit+1))
	if e != nil || len(raw) > statementqualification.ComparisonArtifactLimit {
		return fmt.Errorf("artifact")
	}
	_, observed, e := statementqualification.DecodeS117ComparisonArtifact(bytes.NewReader(raw))
	if e != nil || observed == nil {
		return fmt.Errorf("artifact")
	}
	results := []statementqualification.ComparisonReplay{}
	helper, e := pdfextract.NewClient(args[2])
	if e != nil {
		return e
	}
	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Second)
	defer cancel()
	for _, name := range statementqualification.ScreeningNames() {
		file, e := os.Open(filepath.Join(args[1], name+".pdf"))
		if e != nil {
			return e
		}
		pdf, e := io.ReadAll(io.LimitReader(file, statementdata.MaxPDFBytes+1))
		file.Close()
		if len(pdf) > statementdata.MaxPDFBytes {
			return fmt.Errorf("fixture")
		}
		if e != nil {
			return e
		}
		hash := sha256.Sum256(pdf)
		if hex.EncodeToString(hash[:]) != statementqualification.FixtureHash(name) {
			return fmt.Errorf("fixture")
		}
		doc, e := helper.Extract(ctx, pdf)
		if e != nil {
			return e
		}
		i := len(results)
		for _, index := range []int{i, i + 1} {
			results = append(results, statementqualification.ReplayComparisonObservation(observed, index, doc, time.Now().UTC()))
		}
	}
	return json.NewEncoder(os.Stdout).Encode(struct {
		ProviderCalls int                                       `json:"providerCalls"`
		Results       []statementqualification.ComparisonReplay `json:"results"`
	}{0, results})
}
func main() {
	if run(os.Args[1:]) != nil {
		fmt.Println(`{"verified":false,"status":"UNVERIFIED","providerCalls":0}`)
		os.Exit(1)
	}
}
