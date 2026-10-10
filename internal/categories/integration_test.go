//go:build integration

package categories

import (
	"context"
	"errors"
	"fmt"
	"github.com/QahNaaRlN/grace/internal/financialevents"
	"github.com/QahNaaRlN/grace/internal/platform/access"
	"github.com/QahNaaRlN/grace/internal/platform/database"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

func TestPersonalCategoriesAndFilteredHistory(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL required")
	}
	bg := context.Background()
	pool, err := pgxpool.New(bg, url)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	var owner, other, account string
	for _, dest := range []*string{&owner, &other} {
		if err = pool.QueryRow(bg, "INSERT INTO users(display_name,role) VALUES('Synthetic categories','member') RETURNING id::text").Scan(dest); err != nil {
			t.Fatal(err)
		}
	}
	ctx := access.WithPrincipal(bg, access.Principal{ID: owner, Role: "member"})
	otherCtx := access.WithPrincipal(bg, access.Principal{ID: other, Role: "member"})
	repo := NewRepository(pool)
	cat, err := repo.Save(ctx, "", "Спорт")
	if err != nil || !cat.Editable {
		t.Fatalf("create: %+v %v", cat, err)
	}
	otherCat, err := repo.Save(otherCtx, "", "Спорт")
	if err != nil {
		t.Fatal("same name must be independent", err)
	}
	if _, err = repo.Save(otherCtx, cat.ID, "Чужая"); !errors.Is(err, pgx.ErrNoRows) {
		t.Fatal("foreign rename allowed", err)
	}
	var global string
	if err = pool.QueryRow(bg, "SELECT id::text FROM categories WHERE owner_id IS NULL LIMIT 1").Scan(&global); err != nil {
		t.Fatal(err)
	}
	if _, err = repo.Save(ctx, global, "Changed"); !errors.Is(err, pgx.ErrNoRows) {
		t.Fatal("global rename allowed", err)
	}
	if _, err = repo.Save(ctx, "", "спорт"); err == nil {
		t.Fatal("case-insensitive duplicate allowed")
	}
	list, err := repo.List(ctx)
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range list {
		if item.ID == otherCat.ID {
			t.Fatal("foreign category leaked")
		}
	}
	handler := NewHandler(repo)
	for _, name := range []string{"   ", strings.Repeat("я", 81), "line\nline"} {
		request := httptest.NewRequest("POST", "/categories", strings.NewReader(fmt.Sprintf(`{"name":%q}`, name))).WithContext(ctx)
		response := httptest.NewRecorder()
		handler.Create(response, request)
		if response.Code != 422 {
			t.Fatalf("invalid name: %d", response.Code)
		}
	}
	if err = pool.QueryRow(bg, "INSERT INTO accounts(owner_id,name,type,currency,opening_balance_minor,balance_reconciled) VALUES($1,'Synthetic','debit','RUB',0,true) RETURNING id::text", owner).Scan(&account); err != nil {
		t.Fatal(err)
	}
	events := financialevents.NewRepository(pool)
	at, _ := time.Parse(time.RFC3339, "2026-09-01T00:00:00+05:00")
	event, _, err := events.Create(ctx, financialevents.Command{Kind: "purchase", CategoryID: &cat.ID, AccountID: account, AmountMinor: 15000, OccurredAt: at, IdempotencyKey: "personal-category", RequestHash: strings.Repeat("a", 64)})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = repo.Save(ctx, cat.ID, "Тренировки"); err != nil {
		t.Fatal(err)
	}
	// More than 100 newer records ensure filtering happens before LIMIT.
	for i := 0; i < 105; i++ {
		if _, _, err = events.Create(ctx, financialevents.Command{Kind: "purchase", CategoryID: &global, AccountID: account, AmountMinor: 1, OccurredAt: at.Add(time.Hour), IdempotencyKey: fmt.Sprintf("newer-%d", i), RequestHash: strings.Repeat("b", 64)}); err != nil {
			t.Fatal(err)
		}
	}
	filtered, err := events.ListFiltered(ctx, 100, cat.ID, "2026-09-01", "2026-09-02")
	if err != nil || len(filtered) != 1 || filtered[0].ID != event.ID || *filtered[0].CategoryName != "Тренировки" {
		t.Fatalf("filtered history: %+v %v", filtered, err)
	}
	excluded, err := events.ListFiltered(ctx, 100, cat.ID, "2026-08-31", "2026-09-01")
	if err != nil || len(excluded) != 0 {
		t.Fatal("exclusive local boundary failed", err)
	}
	foreign, err := events.ListFiltered(otherCtx, 100, cat.ID, "", "")
	if err != nil || len(foreign) != 0 {
		t.Fatal("foreign events leaked", err)
	}
	// Direct DB writes cannot bypass the relationship invariant.
	tx, err := database.ForUsers(pool).BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		t.Fatal(err)
	}
	_, err = tx.Exec(ctx, "UPDATE financial_events SET category_id=$2 WHERE id=$1", event.ID, otherCat.ID)
	tx.Rollback(ctx)
	if err == nil {
		t.Fatal("foreign category reference allowed")
	}
	var minor int64
	if err = pool.QueryRow(bg, "SELECT amount_minor FROM postings WHERE financial_event_id=$1", event.ID).Scan(&minor); err != nil || minor != 15000 {
		t.Fatal("money changed by rename", minor, err)
	}
}
