//go:build ignore

// Offline public-fixture diagnosis; never linked into application binaries.
package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"reflect"
	"runtime"

	"github.com/QahNaaRlN/grace/internal/pdfextract"
	"github.com/QahNaaRlN/grace/internal/statementextract"
)

const candidateInstruction = " Preserve all printed fragments of each transaction description in source order, including standalone punctuation separators such as plus or minus signs inside that description. Numeric lexical eligibility is not a financial role: a separator printed within description text belongs to that row.description, not service or a monetary field. Do not include neighboring amounts, dates, currency tokens or page titles in the description."

func must(err error) {
	if err != nil {
		panic(err)
	}
}
func main() {
	client, err := pdfextract.NewClient("/run/grace-pdf/extractor.sock")
	must(err)
	raw, err := os.ReadFile("docs/research/statement-quality/s11-smoke-2026-10-19-locators.json")
	must(err)
	var trace struct {
		Traces []struct {
			Name  string `json:"name"`
			Parts []struct {
				Rows []struct {
					Description []string `json:"description"`
				} `json:"rows"`
				Service [][]string `json:"service"`
			} `json:"parts"`
		} `json:"syntheticFieldLocators"`
	}
	must(json.Unmarshal(raw, &trace))
	if len(trace.Traces) != 1 || trace.Traces[0].Name != "columns" || len(trace.Traces[0].Parts) != 1 || len(trace.Traces[0].Parts[0].Rows) != 2 {
		panic("unexpected diagnosis")
	}
	cases := []any{}
	for _, name := range []string{"columns", "cards"} {
		pdf, err := os.ReadFile("internal/pdfextract/testdata/" + name + ".pdf")
		must(err)
		d, err := client.Extract(context.Background(), pdf)
		must(err)
		before := d.Hash()
		p, err := statementextract.BuildPlan(d, statementextract.Policy{Model: "google/gemini-2.5-flash", Provider: "google-vertex/eu", ProviderName: "Google", InputMicrosPerMillion: 300000, OutputMicrosPerMillion: 2500000}, nil)
		must(err)
		msgs := statementextract.Messages(p.Chunks[0])
		msgs[0]["content"] += candidateInstruction
		b, err := json.Marshal(msgs)
		must(err)
		size := len(b) + len(statementextract.RequestSchema(p.Chunks[0]))
		if size > statementextract.MaxInputBytes || d.Hash() != before {
			panic("bounds/source changed")
		}
		out := map[string]any{"fixture": name, "spans": len(d.Spans), "currentInputBytes": statementextract.InputBytes(p.Chunks[0]), "candidateInputBytes": size, "fitsAccepted90k": true, "sourceUnchanged": true}
		if name == "columns" {
			matches := []bool{}
			missing := []string{}
			part := trace.Traces[0].Parts[0]
			for i, anchor := range []string{"19.10.2026", "21.10.2026"} {
				page := 0
				y := 0.0
				for _, s := range d.Spans {
					if s.Text == anchor {
						page = s.Page
						y = s.Box.YMin
					}
				}
				expected := []string{}
				for _, s := range d.Spans {
					if s.Page == page && math.Abs(s.Box.YMin-y-20) <= 1 {
						expected = append(expected, s.ID)
						found := false
						for _, id := range part.Rows[i].Description {
							if id == s.ID {
								found = true
							}
						}
						if !found {
							if s.Text != "-" {
								panic("diagnosed omission is not public separator")
							}
							missing = append(missing, s.ID)
						}
					}
				}
				matches = append(matches, reflect.DeepEqual(expected, part.Rows[i].Description))
			}
			if !matches[0] || matches[1] || len(missing) != 1 || missing[0] != "p2-b8-l1-w3" {
				panic("source evidence changed")
			}
			inService := false
			for _, group := range part.Service {
				for _, id := range group {
					if id == missing[0] {
						inService = true
					}
				}
			}
			if !inService {
				panic("missing separator not in service")
			}
			out["descriptionMatchesSource"] = matches
			out["missingDescriptionRefs"] = missing
			out["missingSeparatorInService"] = true
		}
		cases = append(cases, out)
	}
	hashes := map[string]string{}
	for _, path := range []string{"scripts/research/statement-description-replay.go", "scripts/research/probe-statement-description.sh", "docs/research/statement-quality/s11-smoke-2026-10-19-locators.json", "internal/statementextract/plan.go", "internal/statementextract/schema.go", "internal/statementextract/lexical.go", "Dockerfile.pdf-extractor", "internal/pdfextract/testdata/columns.pdf", "internal/pdfextract/testdata/cards.pdf"} {
		b, err := os.ReadFile(path)
		must(err)
		h := sha256.Sum256(b)
		hashes[path] = hex.EncodeToString(h[:])
	}
	must(json.NewEncoder(os.Stdout).Encode(map[string]any{"version": "statement-description-replay.offline.v1", "baselineRun": "39255669019", "goVersion": runtime.Version(), "sourceHashes": hashes, "providerCalls": 0, "databaseWrites": 0, "candidateInstruction": candidateInstruction, "cases": cases, "limitation": "Exact public-source diagnosis and candidate byte bounds only; candidate model behavior UNVERIFIED. No repaired mapping is scored as a model response."}))
	fmt.Fprint(os.Stderr, "S1.1 public-source description diagnosis and candidate bounds PASS; zero provider calls\n")
}
