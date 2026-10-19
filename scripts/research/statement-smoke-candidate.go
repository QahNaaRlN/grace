//go:build ignore

// Offline candidate study, never linked into an application binary.
package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"os"
	"reflect"
	"regexp"
	"runtime"
	"sort"
	"strconv"

	"github.com/QahNaaRlN/grace/internal/pdfextract"
	"github.com/QahNaaRlN/grace/internal/statementextract"
)

var numeric = regexp.MustCompile(`^[+−-]?(?:[0-9]+|[0-9]{1,3}(?:[ \x{00A0}\x{202F}][0-9]{3})+)(?:[.,][0-9]{1,2})?$|^[+−-]$`)

type rowRefs struct {
	AccountAmount []string `json:"accountAmount"`
	Description   []string `json:"description"`
}
type diagnostic struct {
	Traces []struct {
		Name  string `json:"name"`
		Parts []struct {
			Rows []rowRefs `json:"rows"`
		} `json:"parts"`
	} `json:"syntheticFieldLocators"`
}

func must(e error) {
	if e != nil {
		panic(e)
	}
}
func equal(a, b []string) bool {
	a = append([]string{}, a...)
	b = append([]string{}, b...)
	sort.Strings(a)
	sort.Strings(b)
	return reflect.DeepEqual(a, b)
}
func main() {
	socket := flag.String("socket", "/run/grace-pdf/extractor.sock", "isolated helper")
	flag.Parse()
	client, e := pdfextract.NewClient(*socket)
	must(e)
	raw, e := os.ReadFile("docs/research/statement-quality/owner-smoke-2026-10-19-locators.json")
	must(e)
	var trace diagnostic
	must(json.Unmarshal(raw, &trace))
	reports := []any{}
	for _, name := range []string{"columns", "cards"} {
		pdf, e := os.ReadFile("internal/pdfextract/testdata/" + name + ".pdf")
		must(e)
		d, e := client.Extract(context.Background(), pdf)
		must(e)
		p, e := statementextract.BuildPlan(d, statementextract.Policy{Model: "google/gemini-2.5-flash", Provider: "google-vertex/eu", ProviderName: "Google", InputMicrosPerMillion: 300000, OutputMicrosPerMillion: 2500000}, nil)
		must(e)
		var schema map[string]any
		must(json.Unmarshal(statementextract.RequestSchema(p.Chunks[0]), &schema))
		oldEnum := schema["$defs"].(map[string]any)["moneySelection"].(map[string]any)["items"].(map[string]any)["enum"].([]any)
		oldCandidates := map[string]bool{}
		for _, alias := range oldEnum {
			oldCandidates[p.Aliases[alias.(string)]] = true
		}
		sourceNumeric := map[string]bool{}
		for _, s := range d.Spans {
			sourceNumeric[s.ID] = numeric.MatchString(s.Text)
		}
		// Simulate only proposed wire construction; current product files/versions
		// are unchanged. Aliases follow delivered source order rather than ID sort.
		proposed := p.Chunks[0]
		proposed.Spans = append([]statementextract.RedactedSpan{}, proposed.Spans...)
		candidates := []string{}
		seen := map[string]bool{}
		for i := range proposed.Spans {
			oldID := p.Aliases[proposed.Spans[i].ID]
			alias := "s" + strconv.FormatInt(int64(i), 36)
			if seen[oldID] {
				panic("source loss/duplication")
			}
			seen[oldID] = true
			s := &proposed.Spans[i]
			if s.Text == "[text_"+s.ID+"]" {
				prefix := "text"
				if sourceNumeric[oldID] {
					prefix = "number"
				}
				s.Text = "[" + prefix + "_" + alias + "]"
			}
			s.ID = alias
			if sourceNumeric[oldID] {
				candidates = append(candidates, alias)
			}
		}
		if len(seen) != len(d.Spans) {
			panic("source loss")
		}
		money := schema["$defs"].(map[string]any)["moneySelection"].(map[string]any)
		money["items"].(map[string]any)["enum"] = candidates
		money["description"] = "Only printed numeric/sign tokens or number-class placeholders. Plain text placeholders are never monetary candidates. Original source values remain hidden and Go validates every selection."
		msgs := statementextract.Messages(proposed)
		msgs[0]["content"] += " Redacted number placeholders disclose lexical class only, never a value or a financial role. Text placeholders cannot be amounts. Aliases follow source order but never encode meaning. Preserve the printed record grouping across dates, amount, currency and descriptions; page titles and headings are service, not merchant descriptions. Never create a transaction solely to consume residual text. Uncertain fields remain empty and visible."
		messageBytes, _ := json.Marshal(msgs)
		schemaBytes, _ := json.Marshal(schema)
		out := map[string]any{"fixture": name, "spans": len(d.Spans), "baselineInputBytes": statementextract.InputBytes(p.Chunks[0]), "candidateInputBytes": len(messageBytes) + len(schemaBytes), "baselineMoneyCandidates": len(oldCandidates), "candidateMoneyCandidates": len(candidates), "exactOnceSourceCoverage": true, "fitsAccepted90k": len(messageBytes)+len(schemaBytes) <= 90000, "modelCalls": 0}
		if name == "columns" {
			expected := [][]string{}
			for _, anchorText := range []string{"19.10.2026", "21.10.2026"} {
				page := 0
				y := 0.0
				for _, s := range d.Spans {
					if s.Text == anchorText {
						page = s.Page
						y = s.Box.YMin
					}
				}
				ids := []string{}
				for _, s := range d.Spans {
					if s.Page == page && s.Box.YMin >= y+19 && s.Box.YMin <= y+21 {
						ids = append(ids, s.ID)
					}
				}
				expected = append(expected, ids)
			}
			badMoney := []string{}
			badDescriptions := []int{}
			observedRows := 0
			for _, t := range trace.Traces {
				if t.Name != name {
					continue
				}
				for _, part := range t.Parts {
					observedRows += len(part.Rows)
					for i, row := range part.Rows {
						if i < len(expected) && !equal(row.Description, expected[i]) {
							badDescriptions = append(badDescriptions, i+1)
						}
						for _, id := range row.AccountAmount {
							if oldCandidates[id] && !sourceNumeric[id] {
								badMoney = append(badMoney, id)
							}
						}
					}
				}
			}
			out["observedRows"] = observedRows
			out["expectedRows"] = 2
			out["plainTextSelectedAsAccountAmount"] = badMoney
			out["wrongDescriptionRows"] = badDescriptions
			if len(badMoney) != 3 || len(badDescriptions) != 2 || badDescriptions[0] != 1 || badDescriptions[1] != 2 {
				panic("diagnosis source evidence changed")
			}
		}
		reports = append(reports, out)
	}
	hashes := map[string]string{}
	for _, path := range []string{"scripts/research/statement-smoke-candidate.go", "scripts/research/probe-statement-smoke-candidate.sh", "docs/research/statement-quality/owner-smoke-2026-10-19-locators.json", "Dockerfile.pdf-extractor", "internal/pdfextract/testdata/columns.pdf", "internal/pdfextract/testdata/cards.pdf"} {
		raw, e := os.ReadFile(path)
		must(e)
		h := sha256.Sum256(raw)
		hashes[path] = hex.EncodeToString(h[:])
	}
	must(json.NewEncoder(os.Stdout).Encode(map[string]any{"version": "statement-owner-candidate.offline.v1", "sourceHashes": hashes, "goVersion": runtime.Version(), "baselineRun": "39248364426", "productionUnchanged": true, "providerCalls": 0, "cases": reports, "limitation": "Shape/coverage evidence only; candidate semantic quality remains UNVERIFIED"}))
}
