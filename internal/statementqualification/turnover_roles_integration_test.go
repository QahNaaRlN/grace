//go:build integration

package statementqualification

import (
	"context"
	"os"
	"testing"

	"github.com/QahNaaRlN/grace/internal/pdfextract"
)

func TestTurnoverRolesRealHelper(t *testing.T) {
	socket := os.Getenv("TEST_PDF_EXTRACTOR_SOCKET")
	if socket == "" {
		t.Skip("isolated pinned helper required")
	}
	helper, e := pdfextract.NewClient(socket)
	if e != nil {
		t.Fatal(e)
	}
	raw, e := os.ReadFile("../pdfextract/testdata/turnover-roles-ru.pdf")
	if e != nil {
		t.Fatal(e)
	}
	d, e := helper.Extract(context.Background(), raw)
	if e != nil {
		t.Fatal(e)
	}
	assertTurnoverFixture(t, d)
}
