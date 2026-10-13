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
	cat, err := repo.Save(ctx, "", str("Спорт"), nil)
	if err != nil || !cat.Editable {
		t.Fatalf("create: %+v %v", cat, err)
	}
	otherCat, err := repo.Save(otherCtx, "", str("Спорт"), nil)
	if err != nil {
		t.Fatal("same name must be independent", err)
	}
	if _, err = repo.Save(otherCtx, cat.ID, str("Чужая"), nil); !errors.Is(err, ErrNotFound) {
		t.Fatal("foreign rename allowed", err)
	}
	var global string
	if err = pool.QueryRow(bg, "SELECT id::text FROM categories WHERE owner_id IS NULL LIMIT 1").Scan(&global); err != nil {
		t.Fatal(err)
	}
	if _, err = repo.Save(ctx, global, str("Changed"), nil); !errors.Is(err, ErrNotFound) {
		t.Fatal("global rename allowed", err)
	}
	if _, err = repo.Save(ctx, "", str("спорт"), nil); err == nil {
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
	if _, err = repo.Save(ctx, cat.ID, str("Тренировки"), nil); err != nil {
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

func str(value string) *string { return &value }

func TestCategoryIconsHidingAndDeletionWithMove(t *testing.T) {
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
	var owner, other, account, global string
	for _, dest := range []*string{&owner, &other} {
		if err = pool.QueryRow(bg, "INSERT INTO users(display_name,role) VALUES('Synthetic category management','member') RETURNING id::text").Scan(dest); err != nil {
			t.Fatal(err)
		}
	}
	if err = pool.QueryRow(bg, "SELECT id::text FROM categories WHERE owner_id IS NULL AND kind='expense' LIMIT 1").Scan(&global); err != nil {
		t.Fatal(err)
	}
	ctx := access.WithPrincipal(bg, access.Principal{ID: owner, Role: "member"})
	otherCtx := access.WithPrincipal(bg, access.Principal{ID: other, Role: "member"})
	repo := NewRepository(pool)
	gym, err := repo.Save(ctx, "", str("Зал"), str("dumbbell"))
	if err != nil || gym.Icon == nil || *gym.Icon != "dumbbell" {
		t.Fatalf("create with icon: %+v %v", gym, err)
	}
	if gym, err = repo.Save(ctx, gym.ID, nil, str("")); err != nil || gym.Icon != nil || gym.Name != "Зал" {
		t.Fatalf("clear icon keeps name: %+v %v", gym, err)
	}
	if _, err = repo.Save(ctx, gym.ID, nil, str("rocket")); err == nil {
		t.Fatal("icon outside the set accepted by the database")
	}
	sport, err := repo.Save(ctx, "", str("Спорт и здоровье"), nil)
	if err != nil {
		t.Fatal(err)
	}
	if err = pool.QueryRow(bg, "INSERT INTO accounts(owner_id,name,type,currency,opening_balance_minor,balance_reconciled) VALUES($1,'Synthetic','debit','RUB',0,true) RETURNING id::text", owner).Scan(&account); err != nil {
		t.Fatal(err)
	}
	events := financialevents.NewRepository(pool)
	at, _ := time.Parse(time.RFC3339, "2026-09-01T00:00:00+05:00")
	for i := 0; i < 3; i++ {
		if _, _, err = events.Create(ctx, financialevents.Command{Kind: "purchase", CategoryID: &gym.ID, AccountID: account, AmountMinor: 1000, OccurredAt: at, IdempotencyKey: fmt.Sprintf("gym-%d", i), RequestHash: strings.Repeat("c", 64)}); err != nil {
			t.Fatal(err)
		}
	}
	list, _ := repo.List(ctx)
	for _, item := range list {
		if item.ID == gym.ID && item.OperationCount != 3 {
			t.Fatalf("operation count %d", item.OperationCount)
		}
	}
	if _, err = repo.Delete(ctx, gym.ID, "", 3); !errors.Is(err, ErrNeedsTarget) {
		t.Fatal("deletion without a target allowed", err)
	}
	if _, err = repo.Delete(ctx, gym.ID, sport.ID, 2); !errors.Is(err, ErrCountChanged) {
		t.Fatal("stale preview count accepted", err)
	}
	if _, err = repo.Delete(ctx, gym.ID, gym.ID, 3); !errors.Is(err, ErrInvalidTarget) {
		t.Fatal("move into itself allowed", err)
	}
	if _, err = repo.Delete(otherCtx, gym.ID, global, 0); !errors.Is(err, ErrNotFound) {
		t.Fatal("foreign deletion allowed", err)
	}
	if _, err = repo.Delete(ctx, global, sport.ID, 0); !errors.Is(err, ErrNotFound) {
		t.Fatal("global deletion allowed", err)
	}
	moved, err := repo.Delete(ctx, gym.ID, sport.ID, 3)
	if err != nil || moved != 3 {
		t.Fatalf("delete with move: %d %v", moved, err)
	}
	var remaining, inTarget int
	_ = pool.QueryRow(bg, "SELECT count(*) FROM categories WHERE id=$1", gym.ID).Scan(&remaining)
	_ = pool.QueryRow(bg, "SELECT count(*) FROM financial_events WHERE category_id=$1", sport.ID).Scan(&inTarget)
	if remaining != 0 || inTarget != 3 {
		t.Fatalf("after move: category rows %d, target operations %d", remaining, inTarget)
	}
	var postings int64
	_ = pool.QueryRow(bg, "SELECT COALESCE(sum(p.amount_minor),0) FROM postings p JOIN financial_events e ON e.id=p.financial_event_id WHERE e.owner_id=$1", owner).Scan(&postings)
	if postings != 3000 {
		t.Fatal("money changed by category deletion", postings)
	}
	empty, _ := repo.Save(ctx, "", str("Пустая"), nil)
	if moved, err = repo.Delete(ctx, empty.ID, "", 0); err != nil || moved != 0 {
		t.Fatal("empty category must delete without a target", err)
	}
	hidden, err := repo.SetHidden(ctx, global, true)
	if err != nil || !hidden.Hidden {
		t.Fatalf("hide global: %+v %v", hidden, err)
	}
	otherList, _ := repo.List(otherCtx)
	for _, item := range otherList {
		if item.ID == global && item.Hidden {
			t.Fatal("hidden flag leaked to another owner")
		}
	}
	if shown, err := repo.SetHidden(ctx, global, false); err != nil || shown.Hidden {
		t.Fatalf("unhide: %+v %v", shown, err)
	}
}
