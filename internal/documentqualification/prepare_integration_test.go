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
	// SPEC-0058 fixes missing fallback fonts. Keep the historical blank-image corpus
	// immutable and pin the two changed complete cases separately after visual review.
	repaired := map[string]string{
		"cards":   "cbf89e181c39c9c6f60607602e0dddb3597454f543c164fa0a4081cb462e922a",
		"columns": "3d25eaa437e494ad3f94b8a9ed9fbe2f3beec9741f6411f74627d7f866c8603f",
	}
	if e == nil {
		for i, item := range c.Cases {
			if expected, ok := repaired[item.ID]; ok {
				if documentdata.Digest(item) != expected || item.Document.CheckReadable() != nil {
					t.Fatalf("font-repaired case %s differs or is blank", item.ID)
				}
				saved.Cases[i] = item
			}
		}
	}

	if e != nil || saved.Hash() != c.Hash() {
		if e == nil {
			for i, item := range c.Cases {
				t.Logf("case=%s actual=%s expected=%s", item.ID, documentdata.Digest(item), documentdata.Digest(saved.Cases[i]))
			}
		}
		t.Fatal("committed corpus differs from actual helper", e)
	}
}
