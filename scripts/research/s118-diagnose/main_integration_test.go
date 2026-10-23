//go:build integration

package main

import (
	"context"
	"github.com/QahNaaRlN/grace/internal/pdfextract"
	"github.com/QahNaaRlN/grace/internal/statementqualification"
	"os"
	"testing"
)

func TestS118VerifierRealHelperContext(t *testing.T) {
	socket := os.Getenv("TEST_PDF_EXTRACTOR_SOCKET")
	if socket == "" {
		t.Skip("pinned helper required")
	}
	helper, err := pdfextract.NewClient(socket)
	if err != nil {
		t.Fatal(err)
	}
	contexts := []publicContext{}
	for _, name := range statementqualification.ScreeningNames() {
		raw, err := os.ReadFile("../../../internal/pdfextract/testdata/" + name + ".pdf")
		if err != nil {
			t.Fatal(err)
		}
		d, err := helper.Extract(context.Background(), raw)
		if err != nil {
			t.Fatal(err)
		}
		contexts = append(contexts, publicContext{name, statementqualification.FixtureHash(name), d})
	}
	if _, err := samplesFor(contexts); err != nil {
		t.Fatal(err)
	}
	contexts[0].FixtureHash = "PRIVATE"
	if _, err := samplesFor(contexts); err == nil {
		t.Fatal("hash mismatch")
	}
}
