// statement-preflight validates local extraction and planning only. It never
// connects to an AI provider, writes to a database, or prints banking text.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/QahNaaRlN/grace/internal/pdfextract"
	"github.com/QahNaaRlN/grace/internal/statementdata"
	"github.com/QahNaaRlN/grace/internal/statementextract"
)

func main() {
	pdf := flag.String("pdf", "", "Local PDF path (never sent to an AI provider)")
	socket := flag.String("socket", "/run/grace-pdf/extractor.sock", "Isolated helper socket")
	flag.Parse()
	if *pdf == "" {
		fmt.Fprintln(os.Stderr, "-pdf is required")
		os.Exit(2)
	}
	info, e := os.Stat(*pdf)
	if e != nil || info.Size() > statementdata.MaxPDFBytes || info.Size() < 5 {
		fmt.Fprintln(os.Stderr, "local PDF unavailable or outside limits")
		os.Exit(1)
	}
	raw, e := os.ReadFile(*pdf)
	if e != nil {
		fmt.Fprintln(os.Stderr, "local PDF unavailable")
		os.Exit(1)
	}
	client, e := pdfextract.NewClient(*socket)
	if e != nil {
		fmt.Fprintln(os.Stderr, "isolated helper unavailable")
		os.Exit(1)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 35*time.Second)
	defer cancel()
	d, e := client.Extract(ctx, raw)
	if e != nil {
		_ = json.NewEncoder(os.Stdout).Encode(map[string]any{"status": e.Error(), "providerCalls": 0})
		os.Exit(1)
	}
	plan, e := statementextract.BuildPlan(d, statementextract.Policy{Model: "google/gemini-2.5-flash", Provider: "google-vertex/eu", ProviderName: "Google", InputMicrosPerMillion: 300000, OutputMicrosPerMillion: 2500000}, nil)
	if e != nil {
		_ = json.NewEncoder(os.Stdout).Encode(map[string]any{"status": e.Error(), "pages": len(d.Pages), "spans": len(d.Spans), "providerCalls": 0})
		os.Exit(1)
	}
	max := 0
	for _, c := range plan.Chunks {
		if n := statementextract.InputBytes(c); n > max {
			max = n
		}
	}
	_ = json.NewEncoder(os.Stdout).Encode(map[string]any{"status": "local_plan_valid", "pages": len(d.Pages), "spans": len(d.Spans), "plannedCalls": len(plan.Chunks), "maxMessagesSchemaBytes": max, "maximumMicrosAtApprovedCeilings": plan.MaximumMicros, "providerCalls": 0})
}
