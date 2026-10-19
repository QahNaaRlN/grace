//go:build integration

package statementextract

import (
	"context"
	"fmt"
	"os"
	"sort"
	"testing"

	"github.com/QahNaaRlN/grace/internal/pdfextract"
	"github.com/QahNaaRlN/grace/internal/statementdata"
)

func TestStatementDocumentLayouts(t *testing.T) {
	socket := os.Getenv("TEST_PDF_EXTRACTOR_SOCKET")
	if socket == "" {
		t.Skip("isolated real helper required")
	}
	client, e := pdfextract.NewClient(socket)
	if e != nil {
		t.Fatal(e)
	}
	for _, name := range []string{"columns", "cards"} {
		raw, e := os.ReadFile("../pdfextract/testdata/" + name + ".pdf")
		if e != nil {
			t.Fatal(e)
		}
		base, e := client.Extract(context.Background(), raw)
		if e != nil {
			t.Fatal(e)
		}
		// Full records are selected from the generator's known printed geometry on
		// the original source, before mutations. This oracle is not sent to a model.
		rows := [][]string{}
		ys := []float64{}
		for i, text := range []string{"19.10.2026", "21.10.2026"} {
			var anchor statementdata.Span
			for _, s := range base.Spans {
				if s.Text == text {
					anchor = s
				}
			}
			if anchor.ID == "" {
				t.Fatal("missing public anchor")
			}
			ys = append(ys, anchor.Box.YMin)
			height := 20.0
			if name == "cards" {
				height = 80
				if i == 1 {
					height = 60
				}
			}
			ids := []string{}
			for _, s := range base.Spans {
				if s.Page == anchor.Page && s.Box.YMin >= anchor.Box.YMin-0.01 && s.Box.YMin <= anchor.Box.YMin+height+0.01 {
					ids = append(ids, s.ID)
				}
			}
			rows = append(rows, ids)
		}
		variants := []string{"prefix_0", "prefix_4", "prefix_8", "prefix_12", "prefix_16", "prefix_24", "prefix_32", "prefix_64", "translate", "scale", "wide_gap", "continuation", "columns", "reordered"}
		for _, variant := range variants {
			t.Run(name+"/"+variant, func(t *testing.T) {
				d := base
				d.Pages = append([]statementdata.Page{}, base.Pages...)
				d.Spans = append([]statementdata.Span{}, base.Spans...)
				first, second := map[string]bool{}, map[string]bool{}
				for _, id := range rows[0] {
					first[id] = true
				}
				for _, id := range rows[1] {
					second[id] = true
				}
				padding := 0
				fmt.Sscanf(variant, "prefix_%d", &padding)
				prefix := []statementdata.Span{}
				for i := 0; i < padding; i++ {
					x := float64(1 + i%50*10)
					prefix = append(prefix, statementdata.Span{ID: fmt.Sprintf("p1-b90000-l1-w%d", i+1), Page: 1, Block: 90000, Line: 1, Word: i + 1, Box: statementdata.Box{XMin: x, YMin: 2, XMax: x + 8, YMax: 7}, Text: "SYNTHETIC"})
				}
				d.Spans = append(prefix, d.Spans...)
				if variant == "continuation" {
					d.Pages = append(d.Pages, statementdata.Page{Number: len(d.Pages) + 1, Width: 600, Height: 800})
				}
				for i := range d.Spans {
					s := &d.Spans[i]
					switch variant {
					case "translate":
						s.Box.YMin += 3
						s.Box.YMax += 3
					case "scale":
						s.Box.XMin *= 1.5
						s.Box.XMax *= 1.5
						s.Box.YMin *= 1.5
						s.Box.YMax *= 1.5
					case "wide_gap":
						if first[s.ID] && s.Box.YMin > ys[0]+5 {
							s.Box.YMin += 35
							s.Box.YMax += 35
						}
					case "continuation":
						if first[s.ID] && s.Box.YMin > ys[0]+5 {
							s.Page = len(d.Pages)
							s.Box.YMin -= ys[0]
							s.Box.YMax -= ys[0]
						}
					case "columns":
						if second[s.ID] {
							s.Page = 1
							s.Box.XMin += 220
							s.Box.XMax += 220
							s.Box.YMin -= ys[1] - ys[0]
							s.Box.YMax -= ys[1] - ys[0]
						}
					}
				}
				for i := range d.Pages {
					switch variant {
					case "translate":
						d.Pages[i].Height += 3
					case "scale":
						d.Pages[i].Width *= 1.5
						d.Pages[i].Height *= 1.5
					case "columns":
						d.Pages[i].Width += 220
					}
				}
				if variant == "reordered" {
					for l, r := 0, len(d.Spans)-1; l < r; l, r = l+1, r-1 {
						d.Spans[l], d.Spans[r] = d.Spans[r], d.Spans[l]
					}
				}
				sort.SliceStable(d.Spans, func(i, j int) bool { return d.Spans[i].Page < d.Spans[j].Page })
				p, e := BuildPlan(d, testPolicy(), nil)
				if e != nil || !validPlan(p) || len(p.Chunks) != 1 {
					t.Fatal("document layout refused or split", e)
				}
				seen := map[string]int{}
				for _, s := range p.Chunks[0].Spans {
					seen[p.Aliases[s.ID]]++
				}
				if len(seen) != len(d.Spans) {
					t.Fatal("source omitted")
				}
				for _, s := range d.Spans {
					if seen[s.ID] != 1 {
						t.Fatal("source lost or duplicated")
					}
				}
				for _, row := range rows {
					for _, id := range row {
						if seen[id] != 1 {
							t.Fatal("full record lost")
						}
					}
				}
				repeat, e := BuildPlan(d, testPolicy(), nil)
				if e != nil || Hash(p) != Hash(repeat) {
					t.Fatal("nondeterministic plan")
				}
			})
		}
	}
}
