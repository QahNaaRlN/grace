//go:build integration

package documentqualification

import (
	"context"
	"encoding/json"
	"github.com/QahNaaRlN/grace/internal/documentdata"
	"github.com/QahNaaRlN/grace/internal/pdfextract"
	"os"
	"path/filepath"
	"testing"
)

func TestQActualRendererCorpus(t *testing.T) {
	socket := os.Getenv("TEST_PDF_EXTRACTOR_SOCKET")
	if socket == "" {
		t.Skip("isolated helper required")
	}
	helper, e := pdfextract.NewClient(socket)
	if e != nil {
		t.Fatal(e)
	}
	root := filepath.Join("..", "..")
	c, e := Prepare(context.Background(), root, helper)
	if e != nil {
		t.Fatal(e)
	}
	if output := os.Getenv("EXPORT_Q_CORPUS"); output != "" {
		if e := os.WriteFile(filepath.Join(filepath.Dir(output), "receipt-pdf.png"), c.Cases[6].Document.Pages[0].Data, 0644); e != nil {
			t.Fatal(e)
		}
		raw, _ := json.Marshal(c)
		if e = os.WriteFile(output, append(raw, '\n'), 0644); e != nil {
			t.Fatal(e)
		}
		return
	}
	saved, e := Load(root)
	if e != nil || saved.Hash() != c.Hash() {
		if e == nil {
			for i, item := range c.Cases {
				t.Logf("case=%s actual=%s expected=%s", item.ID, documentdata.Digest(item), documentdata.Digest(saved.Cases[i]))
			}
		}
		t.Fatal("committed corpus differs from actual helper", e)
	}
}
