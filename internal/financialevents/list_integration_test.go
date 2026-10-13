//go:build integration

package financialevents

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/QahNaaRlN/grace/internal/platform/access"
	"github.com/jackc/pgx/v5/pgxpool"
)

// SPEC-0043: cursor paging without gaps or duplicates, server filters and
// search, whole-day totals across pages, transfers excluded, owner isolation.
func TestListPageCursorFiltersAndDayTotals(t *testing.T) {
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
	var owner, other, card, cash, category string
	for _, dest := range []*string{&owner, &other} {
		if err = pool.QueryRow(bg, "INSERT INTO users(display_name,role) VALUES('Synthetic list','member') RETURNING id::text").Scan(dest); err != nil {
			t.Fatal(err)
		}
	}
	for _, spec := range []struct {
		dest *string
		name string
	}{{&card, "Карта"}, {&cash, "Наличные"}} {
		if err = pool.QueryRow(bg, "INSERT INTO accounts(owner_id,name,type,currency,opening_balance_minor,balance_reconciled) VALUES($1,$2,'debit','RUB',0,true) RETURNING id::text", owner, spec.name).Scan(spec.dest); err != nil {
			t.Fatal(err)
		}
	}
	if err = pool.QueryRow(bg, "SELECT id::text FROM categories WHERE owner_id IS NULL AND kind='expense' LIMIT 1").Scan(&category); err != nil {
		t.Fatal(err)
	}
	ctx := access.WithPrincipal(bg, access.Principal{ID: owner, Role: "member"})
	otherCtx := access.WithPrincipal(bg, access.Principal{ID: other, Role: "member"})
	repo := NewRepository(pool)
	day1, _ := time.Parse(time.RFC3339, "2026-10-01T12:00:00+05:00")
	day2, _ := time.Parse(time.RFC3339, "2026-10-02T12:00:00+05:00")
	create := func(command Command) {
		t.Helper()
		command.RequestHash = strings.Repeat("d", 64)
		if _, _, err := repo.Create(ctx, command); err != nil {
			t.Fatal(err)
		}
	}
	// Seven purchases on day 2 (same instant: ties are broken by created_at and id).
	for i := 0; i < 7; i++ {
		create(Command{Kind: "purchase", CategoryID: &category, AccountID: card, AmountMinor: 100, OccurredAt: day2, Merchant: fmt.Sprintf("Магазин %d", i), IdempotencyKey: fmt.Sprintf("p2-%d", i)})
	}
	create(Command{Kind: "income_received", AccountID: card, AmountMinor: 50000, OccurredAt: day2, Description: "Зарплата_октябрь", IdempotencyKey: "income"})
	create(Command{Kind: "internal_transfer", AccountID: card, DestinationAccountID: cash, AmountMinor: 3000, OccurredAt: day2, IdempotencyKey: "transfer"})
	create(Command{Kind: "purchase", CategoryID: &category, AccountID: cash, AmountMinor: 250, OccurredAt: day1, Merchant: "Кофейня", IdempotencyKey: "p1"})

	seen := map[string]bool{}
	cursor := ""
	pages := 0
	var firstTotals []DayTotal
	for {
		page, err := repo.ListPage(ctx, ListFilter{Limit: 4, Cursor: cursor})
		if err != nil {
			t.Fatal(err)
		}
		if pages == 0 {
			firstTotals = page.DayTotals
		}
		pages++
		for _, item := range page.Items {
			if seen[item.ID] {
				t.Fatalf("duplicate %s across pages", item.ID)
			}
			seen[item.ID] = true
		}
		if page.NextCursor == nil {
			break
		}
		cursor = *page.NextCursor
	}
	if len(seen) != 10 || pages != 3 {
		t.Fatalf("paged %d events in %d pages, want 10 in 3", len(seen), pages)
	}
	// The first page holds only part of day 2, but its total covers the whole day.
	if len(firstTotals) != 1 || firstTotals[0].Date != "2026-10-02" || firstTotals[0].OutflowMinor != 700 || firstTotals[0].InflowMinor != 50000 {
		t.Fatalf("day 2 totals must cover the whole day and exclude the transfer: %+v", firstTotals)
	}
	cashPage, err := repo.ListPage(ctx, ListFilter{Limit: 50, AccountID: cash})
	if err != nil || len(cashPage.Items) != 2 {
		t.Fatalf("account filter: %d items %v", len(cashPage.Items), err)
	}
	for _, total := range cashPage.DayTotals {
		if total.Date == "2026-10-01" && total.OutflowMinor != 250 {
			t.Fatalf("account-scoped totals: %+v", cashPage.DayTotals)
		}
	}
	search, err := repo.ListPage(ctx, ListFilter{Limit: 50, Query: "кофей"})
	if err != nil || len(search.Items) != 1 || search.Items[0].Merchant != "Кофейня" {
		t.Fatalf("search: %+v %v", search.Items, err)
	}
	// "м_г" would match «Магазин» if _ were a wildcard.
	literal, err := repo.ListPage(ctx, ListFilter{Limit: 50, Query: "м_г"})
	if err != nil || len(literal.Items) != 0 {
		t.Fatal("LIKE wildcards in the query must be literal", len(literal.Items), err)
	}
	kind, err := repo.ListPage(ctx, ListFilter{Limit: 50, Kind: "income_received"})
	if err != nil || len(kind.Items) != 1 || kind.Items[0].CategoryIcon != nil {
		t.Fatalf("kind filter: %+v %v", kind.Items, err)
	}
	foreign, err := repo.ListPage(otherCtx, ListFilter{Limit: 50})
	if err != nil || len(foreign.Items) != 0 {
		t.Fatal("events leaked to another owner", err)
	}
	if _, err := repo.ListPage(ctx, ListFilter{Limit: 10, Cursor: "not-a-cursor"}); err != ErrInvalidCursor {
		t.Fatal("invalid cursor accepted", err)
	}
}
