//go:build integration

package receiptdrafts

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/QahNaaRlN/grace/internal/financialevents"
	"github.com/QahNaaRlN/grace/internal/platform/access"
	"github.com/QahNaaRlN/grace/internal/platform/database"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestReceiptDraftAtomicConfirmationIsolationAndCleanup(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL required")
	}
	background := context.Background()
	pool, err := pgxpool.New(background, url)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	var owner, other, account, category string
	if err = pool.QueryRow(background, "INSERT INTO users(display_name,role) VALUES('Synthetic receipt owner','member') RETURNING id::text").Scan(&owner); err != nil {
		t.Fatal(err)
	}
	if err = pool.QueryRow(background, "INSERT INTO users(display_name,role) VALUES('Synthetic other owner','member') RETURNING id::text").Scan(&other); err != nil {
		t.Fatal(err)
	}
	ctx := access.WithPrincipal(background, access.Principal{ID: owner, Role: "member"})
	foreign := access.WithPrincipal(background, access.Principal{ID: other, Role: "member"})
	tenant := database.ForUsers(pool)
	if err = tenant.QueryRow(ctx, "INSERT INTO accounts(name,type,currency) VALUES('Synthetic cash','cash','RUB') RETURNING id::text").Scan(&account); err != nil {
		t.Fatal(err)
	}
	category = "20000000-0000-0000-0000-000000000001"
	fieldsFixture := func() Fields { f := fixtureFields(account); f.CategoryID = &category; return f }
	repo := NewRepository(pool)
	store, _ := NewPhotoStore(t.TempDir())
	create := func(hash string) Draft {
		t.Helper()
		imageID, err := store.Save(syntheticPhoto(t))
		if err != nil {
			t.Fatal(err)
		}
		d, created, err := repo.Create(ctx, imageID, Hash(hash))
		if err != nil || !created {
			t.Fatal(d, err)
		}
		return d
	}
	d := create("first-photo")
	if _, err = repo.Get(foreign, d.ID); err == nil {
		t.Fatal("foreign draft visible")
	}
	d, err = repo.Edit(ctx, d.ID, Edit{d.Revision, fieldsFixture()})
	if err != nil {
		t.Fatal(err)
	}
	var count int
	if err = tenant.QueryRow(ctx, "SELECT count(*) FROM postings").Scan(&count); err != nil || count != 0 {
		t.Fatal("edit wrote ledger", count, err)
	}
	preview, err := repo.Preview(ctx, d.ID, d.Revision)
	if err != nil {
		t.Fatal(err)
	}
	in := Confirmation{preview.ID, preview.Hash}
	var wg sync.WaitGroup
	results := make(chan error, 8)
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); _, err := repo.Confirm(ctx, d.ID, "same-click", in); results <- err }()
	}
	wg.Wait()
	close(results)
	for err := range results {
		if err != nil {
			t.Fatal("concurrent confirmation", err)
		}
	}
	if err = tenant.QueryRow(ctx, "SELECT count(*) FROM postings").Scan(&count); err != nil || count != 1 {
		t.Fatal("duplicate posting", count, err)
	}
	d, err = repo.Get(ctx, d.ID)
	if err != nil || d.Status != "confirmed" || d.PhotoStatus != "pending_delete" {
		t.Fatal(d, err)
	}
	if _, err = repo.Confirm(ctx, d.ID, "same-click", Confirmation{preview.ID, strings.Repeat("f", 64)}); err == nil {
		t.Fatal("key conflict accepted")
	}
	if _, err = repo.Edit(ctx, d.ID, Edit{d.Revision, fieldsFixture()}); err == nil {
		t.Fatal("confirmed provenance mutable")
	}
	if err = repo.DeletePhoto(ctx, d, store); err != nil {
		t.Fatal(err)
	}
	if _, err = store.Read(d.ImageID); err == nil {
		t.Fatal("confirmed photo retained")
	}
	second := create("second-photo")
	f := fieldsFixture()
	second, err = repo.Edit(ctx, second.ID, Edit{second.Revision, f})
	if err != nil {
		t.Fatal(err)
	}
	match, err := repo.Preview(ctx, second.ID, second.Revision)
	if err != nil || len(match.Candidates) != 1 || len(match.Warnings) == 0 {
		t.Fatal("missing duplicate review", match, err)
	}
	if _, err = repo.Confirm(ctx, second.ID, "unreviewed", Confirmation{match.ID, match.Hash}); err == nil {
		t.Fatal("unreviewed duplicate published")
	}
	// Explicit create reason resolves candidate review, but ledger drift invalidates it.
	f.CandidateReason = "Synthetic separate purchase"
	second, err = repo.Edit(ctx, second.ID, Edit{second.Revision, f})
	if err != nil {
		t.Fatal(err)
	}
	match, err = repo.Preview(ctx, second.ID, second.Revision)
	if err != nil {
		t.Fatal(err)
	}
	command := financialevents.Command{Kind: "purchase", AccountID: account, CategoryID: &category, AmountMinor: 100, OccurredDate: "2026-09-13", IdempotencyKey: "drift", RequestHash: "drift"}
	if _, _, err = financialevents.NewRepository(pool).Create(ctx, command); err != nil {
		t.Fatal(err)
	}
	if _, err = repo.Confirm(ctx, second.ID, "stale", Confirmation{match.ID, match.Hash}); err == nil {
		t.Fatal("stale ledger preview accepted")
	}
	// Existing purchase can be detailed without a second posting.
	event, _, err := financialevents.NewRepository(pool).Create(ctx, financialevents.Command{Kind: "purchase", AccountID: account, CategoryID: &category, AmountMinor: 18000, OccurredDate: "2026-09-12", IdempotencyKey: "existing", RequestHash: "existing"})
	if err != nil {
		t.Fatal(err)
	}
	f.Action = "link_existing"
	f.ExistingEventID = event.ID
	second, err = repo.Edit(ctx, second.ID, Edit{second.Revision, f})
	if err != nil {
		t.Fatal(err)
	}
	match, err = repo.Preview(ctx, second.ID, second.Revision)
	if err != nil {
		t.Fatal(err)
	}
	linked, err := repo.Confirm(ctx, second.ID, "link", Confirmation{match.ID, match.Hash})
	if err != nil || linked.EventID == nil || *linked.EventID != event.ID {
		t.Fatal(linked, err)
	}
	if err = tenant.QueryRow(ctx, "SELECT count(*) FROM postings").Scan(&count); err != nil || count != 3 {
		t.Fatal("link created posting", count, err)
	}
	cancelled := create("cancel-photo")
	cancelled, err = repo.Cancel(ctx, cancelled.ID)
	if err != nil {
		t.Fatal(err)
	}
	photoPath, _ := store.path(cancelled.ImageID)
	if err = os.Rename(photoPath, photoPath+".backup"); err != nil {
		t.Fatal(err)
	}
	if err = os.Mkdir(photoPath, 0700); err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(photoPath, "blocked"), []byte("synthetic"), 0600); err != nil {
		t.Fatal(err)
	}
	if err = repo.Sweep(ctx, store); err == nil {
		t.Fatal("storage failure hidden")
	}
	pending, err := repo.Get(ctx, cancelled.ID)
	if err != nil || pending.PhotoStatus != "pending_delete" {
		t.Fatal("cleanup job lost", pending, err)
	}
	if err = os.Remove(filepath.Join(photoPath, "blocked")); err != nil {
		t.Fatal(err)
	}
	if err = os.Remove(photoPath); err != nil {
		t.Fatal(err)
	}
	if err = os.Rename(photoPath+".backup", photoPath); err != nil {
		t.Fatal(err)
	}
	// A new repository models a restart; durable pending jobs remain discoverable.
	if err = NewRepository(pool).Sweep(ctx, store); err != nil {
		t.Fatal(err)
	}
	if _, err = store.Read(cancelled.ImageID); err == nil {
		t.Fatal("cancelled photo retained")
	}
	expired := create("expire-photo")
	if _, err = pool.Exec(background, "UPDATE receipt_drafts SET expires_at=now()-interval '1 hour' WHERE id::text=$1", expired.ID); err != nil {
		t.Fatal(err)
	}
	if err = repo.Sweep(ctx, store); err != nil {
		t.Fatal(err)
	}
	expired, err = repo.Get(ctx, expired.ID)
	if err != nil || expired.Status != "expired" || expired.PhotoStatus != "deleted" {
		t.Fatal(expired, err)
	}
	if _, err = tenant.BeginTx(background, pgx.TxOptions{}); err == nil {
		t.Fatal("principal-less access")
	}
}
