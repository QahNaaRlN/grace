//go:build unix

package pdfextract

import (
	"context"
	"errors"
	"github.com/QahNaaRlN/grace/internal/documentdata"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

func (Poppler) Render(ctx context.Context, path string) ([][]byte, error) {
	info, e := runBounded(ctx, exec.CommandContext(ctx, "/usr/bin/pdfinfo", path))
	if e != nil {
		return nil, e
	}
	count := 0
	for _, line := range strings.Split(string(info), "\n") {
		if strings.HasPrefix(line, "Pages:") {
			count, _ = strconv.Atoi(strings.TrimSpace(strings.TrimPrefix(line, "Pages:")))
		}
		if strings.HasPrefix(line, "Encrypted:") && !strings.Contains(line, "no") {
			return nil, errors.New("unsupported_pdf")
		}
	}
	if count < 1 || count > documentdata.MaxPages {
		return nil, errors.New("document_limit")
	}
	prefix := filepath.Join(filepath.Dir(path), "page")
	// pdfinfo already bounded the complete document; no silent first-page subset.
	_, e = runBounded(ctx, exec.CommandContext(ctx, "/usr/bin/pdftoppm", "-png", "-scale-to", "1800", path, prefix))
	if e != nil {
		return nil, e
	}
	names, e := filepath.Glob(prefix + "-*.png")
	if e != nil || len(names) != count {
		return nil, errors.New("unsupported_pdf")
	}
	pages := [][]byte{}
	total := int64(0)
	for _, name := range names {
		st, e := os.Lstat(name)
		if e != nil || !st.Mode().IsRegular() {
			return nil, errors.New("unsupported_pdf")
		}
		total += st.Size()
		if total > documentdata.MaxPageBytes {
			return nil, errors.New("document_limit")
		}
		b, e := os.ReadFile(name)
		if e != nil {
			return nil, e
		}
		pages = append(pages, b)
	}
	return pages, nil
}
