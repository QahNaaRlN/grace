// Offline study only: fixed public PDFs, no model, database or financial writer.
package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/QahNaaRlN/grace/internal/pdfextract"
	"github.com/QahNaaRlN/grace/internal/statementdata"
	"github.com/QahNaaRlN/grace/internal/statementextract"
)

type RowProbe struct {
	Row        int   `json:"row"`
	CoreChunks []int `json:"coreChunks"`
	CoreAtomic bool  `json:"coreAtomic"`
}
type Case struct {
	Name              string     `json:"name"`
	Variant           string     `json:"variant"`
	SourceSHA         string     `json:"sourceSHA256"`
	Spans             int        `json:"spans"`
	Chunks            int        `json:"chunks"`
	MaxInput          int        `json:"maxMessagesSchemaBytes"`
	SourceExactlyOnce bool       `json:"sourceExactlyOnce"`
	Rows              []RowProbe `json:"rows"`
	Error             string     `json:"error,omitempty"`
}

func main() {
	fixtures := flag.String("fixtures", "/fixtures", "Fixed synthetic PDF fixture directory")
	socket := flag.String("socket", "/run/grace-pdf/extractor.sock", "Isolated helper socket")
	flag.Parse()
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	client, err := pdfextract.NewClient(*socket)
	if err != nil {
		panic("isolated helper unavailable")
	}
	cases := []Case{}
	for _, name := range []string{"columns", "cards"} {
		raw, err := os.ReadFile(filepath.Join(*fixtures, name+".pdf"))
		if err != nil {
			panic("synthetic fixture unavailable")
		}
		digest := sha256.Sum256(raw)
		doc, err := client.Extract(ctx, raw)
		if err != nil {
			panic("synthetic extraction failed")
		}
		rows := [][]string{{"19.10.2026", "20.10.2026", "-1234.56", "-15.00"}, {"21.10.2026", "22.10.2026", "+2000.00"}}
		if name == "cards" {
			rows[0] = append(rows[0], "10:30", "11:00")
			rows[1] = append(rows[1], "12:15", "13:20")
		}
		rowIDs := [][]string{}
		for _, texts := range rows {
			ids := []string{}
			for _, text := range texts {
				found := []string{}
				for _, s := range doc.Spans {
					if s.Text == text {
						found = append(found, s.ID)
					}
				}
				if len(found) != 1 {
					panic("synthetic golden core selector is not unique")
				}
				ids = append(ids, found[0])
			}
			rowIDs = append(rowIDs, ids)
		}
		for _, padding := range []int{0, 4, 8, 12, 16, 24, 32, 64} {
			d := doc
			d.Spans = append([]statementdata.Span{}, doc.Spans...)
			prefix := []statementdata.Span{}
			for i := 0; i < padding; i++ {
				x := float64(1 + i%50*10)
				prefix = append(prefix, statementdata.Span{ID: fmt.Sprintf("p1-b90000-l1-w%d", i+1), Page: 1, Block: 90000, Line: 1, Word: i + 1, Box: statementdata.Box{XMin: x, YMin: 2, XMax: x + 8, YMax: 7}, Text: "SYNTHETIC"})
			}
			d.Spans = append(prefix, d.Spans...)
			out := Case{Name: name, Variant: fmt.Sprintf("layout_prefix_%d", padding), SourceSHA: hex.EncodeToString(digest[:]), Spans: len(d.Spans), Rows: []RowProbe{}}
			plan, err := statementextract.BuildPlan(d, statementextract.Policy{Model: "google/gemini-2.5-flash", Provider: "google-vertex/eu", ProviderName: "Google", InputMicrosPerMillion: 300000, OutputMicrosPerMillion: 2500000}, nil)
			if err != nil {
				out.Error = err.Error()
				cases = append(cases, out)
				continue
			}
			out.Chunks = len(plan.Chunks)
			seen := map[string]int{}
			chunkIDs := []map[string]bool{}
			for _, chunk := range plan.Chunks {
				ids := map[string]bool{}
				for _, s := range chunk.Spans {
					seen[s.ID]++
					ids[s.ID] = true
				}
				chunkIDs = append(chunkIDs, ids)
				if n := statementextract.InputBytes(chunk); n > out.MaxInput {
					out.MaxInput = n
				}
			}
			out.SourceExactlyOnce = len(seen) == len(d.Spans)
			for _, n := range seen {
				if n != 1 {
					out.SourceExactlyOnce = false
				}
			}
			if !out.SourceExactlyOnce || out.MaxInput > 9000 {
				panic("planner lost or duplicated source or exceeded declared byte bound")
			}
			for row, ids := range rowIDs {
				r := RowProbe{Row: row + 1, CoreChunks: []int{}}
				for c, source := range chunkIDs {
					n := 0
					for _, id := range ids {
						if source[id] {
							n++
						}
					}
					if n > 0 {
						r.CoreChunks = append(r.CoreChunks, c+1)
					}
					if n == len(ids) {
						r.CoreAtomic = true
					}
				}
				out.Rows = append(out.Rows, r)
			}
			cases = append(cases, out)
		}
	}
	json.NewEncoder(os.Stdout).Encode(map[string]any{"studyVersion": "statement-chunks.v1", "promptVersion": statementextract.PromptVersion, "extractorVersion": statementdata.ExtractorVersion, "providerCalls": 0, "databaseWrites": 0, "cases": cases})
}
