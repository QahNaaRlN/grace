//go:build integration

package imports

import (
	"os"
	"testing"
	"time"
)

func TestStatementVisualSourceExpiry(t *testing.T) {
	ctx, _, repo := statementSetup(t)
	raw, e := os.ReadFile("../statementextract/testdata/direct-page-reading/cards/page-1.png")
	if e != nil {
		t.Fatal(e)
	}
	source, _, e := repo.UploadStatementImage(ctx, raw, accountID, "synthetic-expiry.png", "visual-expiry")
	if e != nil {
		t.Fatal(e)
	}
	if len(source.Pages) != 1 || source.ReadingMode != "image_review" {
		t.Fatal("missing image source")
	}
	repo.now = func() time.Time { return time.Now().Add(8 * 24 * time.Hour) }
	expired, e := repo.Statement(ctx, source.ID)
	if e != nil || expired.Status != "expired" || expired.DeletedAt == nil || len(expired.Pages) != 0 || len(expired.Rendered.Pages) != 0 || expired.Layout != nil {
		t.Fatal("expired source retained full state", e)
	}
	if _, e = repo.StatementPage(ctx, source.ID, 1); e == nil {
		t.Fatal("expired page visible")
	}
	if _, _, e = repo.StatementOriginal(ctx, source.ID); e == nil {
		t.Fatal("expired original visible")
	}
}
