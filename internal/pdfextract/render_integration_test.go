//go:build integration

package pdfextract

import (
	"bytes"
	"context"
	"fmt"
	"github.com/QahNaaRlN/grace/internal/documentdata"
	"os"
	"testing"
)

func TestS56RealRenderNativeAndScan(t *testing.T) {
	socket := os.Getenv("TEST_PDF_EXTRACTOR_SOCKET")
	if socket == "" {
		t.Skip("isolated helper required")
	}
	client, e := NewClient(socket)
	if e != nil {
		t.Fatal(e)
	}
	for _, tc := range []struct {
		path string
		scan bool
	}{{"testdata/cards.pdf", false}, {"testdata/columns.pdf", false}, {"../statementextract/testdata/direct-page-reading/full32.pdf", false}, {"testdata/full32-scan.pdf", true}} {
		t.Run(tc.path, func(t *testing.T) {
			raw, e := os.ReadFile(tc.path)
			if e != nil {
				t.Fatal(e)
			}
			out, e := client.Render(context.Background(), raw)
			if e != nil {
				t.Fatal(e)
			}
			if out.Document.Validate() != nil || out.Document.SourceSHA256 != documentdata.Hash(raw) || len(out.Document.Pages) < 1 {
				t.Fatal("invalid source render")
			}
			if tc.scan {
				if out.Layout != nil || out.Document.Text != "" || len(out.Document.Pages) != 3 {
					t.Fatal("scan gained invented native text")
				}
				if _, e = client.Extract(context.Background(), raw); e == nil {
					t.Fatal("legacy extractor silently accepted scan")
				}
			} else {
				if out.Layout == nil || out.Layout.Validate() != nil || out.Document.Text == "" {
					t.Fatal("native evidence lost")
				}
			}
		})
	}
}

func TestS56RealRendererRefusesFifthPage(t *testing.T) {
	socket := os.Getenv("TEST_PDF_EXTRACTOR_SOCKET")
	if socket == "" {
		t.Skip("isolated helper required")
	}
	client, e := NewClient(socket)
	if e != nil {
		t.Fatal(e)
	}
	objects := []string{`<< /Type /Catalog /Pages 2 0 R >>`, `<< /Type /Pages /Count 5 /Kids [3 0 R 4 0 R 5 0 R 6 0 R 7 0 R] >>`}
	for i := 0; i < 5; i++ {
		objects = append(objects, `<< /Type /Page /Parent 2 0 R /MediaBox [0 0 600 800] >>`)
	}
	var pdf bytes.Buffer
	pdf.WriteString("%PDF-1.4\n")
	offsets := []int{0}
	for i, obj := range objects {
		offsets = append(offsets, pdf.Len())
		fmt.Fprintf(&pdf, "%d 0 obj\n%s\nendobj\n", i+1, obj)
	}
	xref := pdf.Len()
	fmt.Fprintf(&pdf, "xref\n0 %d\n0000000000 65535 f \n", len(offsets))
	for _, offset := range offsets[1:] {
		fmt.Fprintf(&pdf, "%010d 00000 n \n", offset)
	}
	fmt.Fprintf(&pdf, "trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", len(offsets), xref)
	if _, e = client.Render(context.Background(), pdf.Bytes()); e == nil || e.Error() != "document_limit" {
		t.Fatal("fifth page was rendered/truncated", e)
	}
}
