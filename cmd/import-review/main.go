// import-review is an offline operator tool. It has no database/provider wiring.
package main

import (
	"encoding/json"
	"flag"
	"github.com/QahNaaRlN/grace/internal/importreview"
	"io"
	"os"
)

func main() { os.Exit(run(os.Args[1:], os.Stdout, os.Stderr)) }
func run(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("grace-import-review", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	input := flags.String("input", "", "versioned snapshot JSON")
	output := flags.String("output-dir", "", "new private operator directory")
	if flags.Parse(args) != nil || flags.NArg() != 0 || *input == "" || *output == "" {
		return fail(stderr, "invalid_arguments", 2)
	}
	f, err := os.Open(*input)
	if err != nil {
		return fail(stderr, "input_unavailable", 1)
	}
	s, err := importreview.Load(f)
	closeErr := f.Close()
	if err != nil || closeErr != nil {
		return fail(stderr, "invalid_snapshot", 1)
	}
	r, err := importreview.Review(s)
	if err != nil {
		return fail(stderr, err.Error(), 1)
	}
	if err := importreview.Write(*output, r); err != nil {
		return fail(stderr, err.Error(), 1)
	}
	issues := len(r.Issues)
	for _, record := range r.Records {
		issues += len(record.Issues)
	}
	if json.NewEncoder(stdout).Encode(map[string]any{"status": "reviewed", "records": len(r.Records), "candidates": len(r.Candidates), "issues": issues, "published": false}) != nil {
		return 1
	}
	return 0
}
func fail(w io.Writer, code string, exit int) int {
	_ = json.NewEncoder(w).Encode(map[string]string{"error": code})
	return exit
}
