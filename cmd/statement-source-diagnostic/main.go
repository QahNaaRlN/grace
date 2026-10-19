// Temporary read-only diagnostic for shipped synthetic sources only. No AI calls.
package main

import (
	"context"
	"encoding/json"
	"github.com/QahNaaRlN/grace/internal/pdfextract"
	"github.com/QahNaaRlN/grace/internal/statementdata"
	"github.com/QahNaaRlN/grace/internal/statementextract"
	"github.com/QahNaaRlN/grace/internal/statementqualification"
	"io"
	"os"
	"regexp"
	"strings"
	"time"
)

func main() {
	var report statementqualification.Report
	if json.NewDecoder(io.LimitReader(os.Stdin, 262145)).Decode(&report) != nil || report.Version != "statement.synthetic.v1" {
		os.Exit(1)
	}
	client, err := pdfextract.NewClient(os.Getenv("GRACE_PDF_EXTRACTOR_SOCKET"))
	if err != nil {
		os.Exit(1)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	output := []any{}
	for _, trace := range report.Traces {
		if trace.Name != "columns" && trace.Name != "cards" {
			os.Exit(1)
		}
		raw, err := os.ReadFile("/usr/share/grace/statement-qualification/" + trace.Name + ".pdf")
		if err != nil {
			os.Exit(1)
		}
		d, err := client.Extract(ctx, raw)
		if err != nil {
			os.Exit(1)
		}
		plan, err := statementextract.BuildPlan(d, report.Policy, nil)
		if err != nil {
			os.Exit(1)
		}
		p, err := statementextract.MergeProposals(plan, trace.Parts)
		if err != nil {
			os.Exit(1)
		}
		pages := []int{}
		for _, page := range d.Pages {
			pages = append(pages, page.Number)
		}
		r, err := statementdata.ValidateProposal(d, p, pages, true, "")
		if err != nil {
			os.Exit(1)
		}
		used := map[string]bool{}
		evidence := append([]statementdata.Evidence{}, r.Header.Evidence...)
		for _, row := range r.Rows {
			evidence = append(evidence, row.Evidence...)
		}
		for _, e := range evidence {
			for _, s := range e.Spans {
				used[s.ID] = true
			}
		}
		for _, g := range p.Service {
			for _, id := range g.Spans {
				used[id] = true
			}
		}
		missing := []any{}
		locator := regexp.MustCompile(`^p[0-9]+-b[0-9]+-l[0-9]+-w[0-9]+$`)
		for _, s := range d.Spans {
			if !used[s.ID] && locator.MatchString(s.ID) {
				kind := "text"
				if _, err := statementdata.Currency(s.Text); err == nil {
					kind = "currency"
				}
				if regexp.MustCompile(`^[+−0-9.,:-]+$`).MatchString(s.Text) {
					kind = "numeric_or_date"
				}
				if strings.Contains(strings.ToLower(s.Text), "balance") {
					kind = "balance_heading"
				}
				missing = append(missing, map[string]string{"locator": s.ID, "kind": kind})
			}
		}
		output = append(output, map[string]any{"name": trace.Name, "unmapped": missing})
	}
	json.NewEncoder(os.Stdout).Encode(map[string]any{"syntheticSourceDiagnostic": output})
}
