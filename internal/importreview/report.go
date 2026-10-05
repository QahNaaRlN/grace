package importreview

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func escape(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "|", "&#124;")
	s = strings.ReplaceAll(s, "`", "&#96;")
	s = strings.ReplaceAll(s, "[", "&#91;")
	s = strings.ReplaceAll(s, "]", "&#93;")
	s = strings.ReplaceAll(s, "\r", " ")
	return strings.ReplaceAll(s, "\n", " ")
}
func Markdown(r Report) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# Grace import review\n\nSource: %s. Captured: %s.\n\nNo financial facts were published. Candidates require individual confirmation.\n\n", escape(r.Source.SourceID), escape(r.Source.CapturedAt))
	b.WriteString("## Domains\n\n| Domain | Rows |\n| --- | ---: |\n")
	domains := []string{}
	for d := range r.DomainCounts {
		domains = append(domains, d)
	}
	sort.Strings(domains)
	for _, d := range domains {
		fmt.Fprintf(&b, "| %s | %d |\n", d, r.DomainCounts[d])
	}
	for _, issue := range r.Issues {
		fmt.Fprintf(&b, "\n%s: %s\n", escape(issue.Code), escape(issue.Message))
	}
	byID := map[string]Record{}
	for _, record := range r.Records {
		byID[record.ID] = record
	}
	for _, domain := range domains {
		fmt.Fprintf(&b, "\n## %s\n\n| Source row | Kind | Date | Amount minor | Issues |\n| --- | --- | --- | ---: | --- |\n", domain)
		for _, record := range r.Records {
			if record.Domain != domain {
				continue
			}
			n := record.Normalized
			date, amount := "unknown", "unknown"
			if n.Date != nil {
				date = n.Date.Local
			}
			if n.AmountMinor != nil {
				amount = fmt.Sprint(*n.AmountMinor)
			}
			codes := []string{}
			for _, i := range record.Issues {
				codes = append(codes, i.Code)
			}
			fmt.Fprintf(&b, "| %s!%d | %s | %s | %s | %s |\n", escape(record.Sheet), record.Row, escape(n.Kind), escape(date), amount, escape(strings.Join(codes, ", ")))
		}
		b.WriteString("\n| Source row | Named amount | Minor units |\n| --- | --- | ---: |\n")
		for _, record := range r.Records {
			if record.Domain != domain {
				continue
			}
			keys := []string{}
			for name := range record.Normalized.Amounts {
				keys = append(keys, name)
			}
			sort.Strings(keys)
			for _, name := range keys {
				value := "unknown"
				if n := record.Normalized.Amounts[name]; n != nil {
					value = fmt.Sprint(*n)
				}
				fmt.Fprintf(&b, "| %s!%d | %s | %s |\n", escape(record.Sheet), record.Row, name, value)
			}
		}
	}
	b.WriteString("\n## Reconciliation candidates\n\n| Source row | Candidate row | Rule | Matched | Different source columns | Decision |\n| --- | --- | --- | --- | --- | --- |\n")
	for _, c := range r.Candidates {
		a, z := byID[c.RecordID], byID[c.OtherRecordID]
		fmt.Fprintf(&b, "| %s!%d | %s!%d | %s | %s | %s | %s |\n", escape(a.Sheet), a.Row, escape(z.Sheet), z.Row, c.Rule, strings.Join(c.Matched, ", "), strings.Join(c.Differences, ", "), c.Decision)
	}
	b.WriteString("\n## Issues\n\n| Coordinate | Severity | Code | Action |\n| --- | --- | --- | --- |\n")
	for _, record := range r.Records {
		for _, issue := range record.Issues {
			fmt.Fprintf(&b, "| %s | %s | %s | %s |\n", escape(issue.Coordinate), issue.Severity, issue.Code, escape(issue.Message))
		}
	}
	return b.String()
}

// Write creates a new private result directory. It never overwrites an existing
// result and removes only files it created if a write fails.
func Write(outputDir string, r Report) error {
	abs, err := filepath.Abs(outputDir)
	if err != nil {
		return fmt.Errorf("invalid_output")
	}
	parent, err := filepath.EvalSymlinks(filepath.Dir(abs))
	if err != nil {
		return fmt.Errorf("invalid_output")
	}
	for p := parent; ; p = filepath.Dir(p) {
		if _, e := os.Lstat(filepath.Join(p, ".git")); e == nil {
			return fmt.Errorf("output_inside_repository")
		}
		if filepath.Dir(p) == p {
			break
		}
	}
	abs = filepath.Join(parent, filepath.Base(abs))
	if err := os.Mkdir(abs, 0o700); err != nil {
		return fmt.Errorf("output_exists_or_unavailable")
	}
	complete := false
	created := []string{}
	defer func() {
		if !complete {
			for _, p := range created {
				_ = os.Remove(p)
			}
			_ = os.Remove(abs)
		}
	}()
	var encoded bytes.Buffer
	enc := json.NewEncoder(&encoded)
	enc.SetIndent("", "  ")
	if enc.Encode(r) != nil {
		return fmt.Errorf("encode_report")
	}
	for _, entry := range []struct {
		name string
		data []byte
	}{{"review.json", encoded.Bytes()}, {"review.md", []byte(Markdown(r))}} {
		path := filepath.Join(abs, entry.name)
		f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
		if err != nil {
			return fmt.Errorf("write_report")
		}
		created = append(created, path)
		_, writeErr := f.Write(entry.data)
		closeErr := f.Close()
		if writeErr != nil || closeErr != nil {
			return fmt.Errorf("write_report")
		}
	}
	complete = true
	return nil
}
