//go:build integration

package main

import (
	"context"
	"github.com/QahNaaRlN/grace/internal/pdfextract"
	"github.com/QahNaaRlN/grace/internal/statementqualification"
	"os"
	"testing"
)

func TestS112RealHelperStages(t *testing.T) {
	socket := os.Getenv("TEST_PDF_EXTRACTOR_SOCKET")
	if socket == "" {
		t.Skip("isolated helper required")
	}
	helper, e := pdfextract.NewClient(socket)
	if e != nil {
		t.Fatal(e)
	}
	raw, e := os.ReadFile("../../../internal/pdfextract/testdata/columns.pdf")
	if e != nil {
		t.Fatal(e)
	}
	d, e := helper.Extract(context.Background(), raw)
	if e != nil {
		t.Fatal(e)
	}
	checkStages(t, publicContext{statementqualification.FixtureHash("columns"), d})
}
