package main

import (
	"regexp"

	"github.com/QahNaaRlN/grace/internal/statementdata"
	"github.com/QahNaaRlN/grace/internal/statementqualification"
)

// Only source locator IDs from our fixed synthetic corpus are exported. Never
// include literal values, service reasons, policy, billing, generations or usage.
func safeDiagnostic(report statementqualification.Report) map[string]any {
	locator := regexp.MustCompile(`^p[0-9]+-b[0-9]+-l[0-9]+-w[0-9]+$`)
	ids := func(in statementdata.Selection) []string {
		out := []string{}
		for _, s := range in {
			if locator.MatchString(s) {
				out = append(out, s)
			}
		}
		return out
	}
	traces := []any{}
	for _, trace := range report.Traces {
		if trace.Name != "columns" && trace.Name != "cards" {
			continue
		}
		parts := []any{}
		for _, part := range trace.Parts {
			labels := map[string]any{}
			for _, role := range []string{"openingBookBalance", "closingBookBalance", "availableBalanceObserved", "debitTurnover", "creditTurnover"} {
				labels[role] = ids(part.Header.Labels[role])
			}
			parts = append(parts, map[string]any{"opening": ids(part.Header.OpeningBookBalance), "closing": ids(part.Header.ClosingBookBalance), "available": ids(part.Header.AvailableBalanceObserved), "debit": ids(part.Header.DebitTurnover), "credit": ids(part.Header.CreditTurnover), "currency": ids(part.Header.Currency), "labels": labels})
		}
		traces = append(traces, map[string]any{"name": trace.Name, "headers": parts})
	}
	return map[string]any{"syntheticHeaderLocators": traces}
}
