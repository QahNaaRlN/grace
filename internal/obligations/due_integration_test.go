//go:build integration

package obligations_test

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/QahNaaRlN/grace/internal/analytics"
	"github.com/QahNaaRlN/grace/internal/obligations"
	"github.com/QahNaaRlN/grace/internal/platform/access"
	"github.com/QahNaaRlN/grace/internal/platform/database"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// SPEC-0046: unpaid payments due within 0–3 days are «скоро» in both the cash
// position and the obligation schedule; totals stay as they were.
func TestDueSoonInCashPositionAndSchedules(t *testing.T) {
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
	var owner string
	if err = pool.QueryRow(bg, "INSERT INTO users(display_name,role) VALUES('Synthetic due soon','member') RETURNING id::text").Scan(&owner); err != nil {
		t.Fatal(err)
	}
	ctx := access.WithPrincipal(bg, access.Principal{ID: owner, Role: "member"})
	repo := obligations.NewRepository(pool)
	const today = "2026-10-14"
	dates := []string{"2026-10-13", "2026-10-14", "2026-10-15", "2026-10-17", "2026-10-18"}
	tx, err := database.ForUsers(pool).BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		t.Fatal(err)
	}
	var id string
	if err = tx.QueryRow(ctx, `INSERT INTO obligations (name, kind, currency, total_minor, idempotency_key, request_hash)
		VALUES ('Кредит', 'loan', 'RUB', 5000, 'due-soon', $1) RETURNING id::text`, strings.Repeat("a", 64)).Scan(&id); err != nil {
		t.Fatal(err)
	}
	for i, due := range dates {
		if _, err = tx.Exec(ctx, `INSERT INTO scheduled_payments (obligation_id, position, due_on, amount_minor) VALUES ($1, $2, $3, 1000)`, id, i+1, due); err != nil {
			t.Fatal(err)
		}
	}
	if err = tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	loaded, err := repo.Get(ctx, id, today)
	if err != nil {
		t.Fatal(err)
	}
	want := []struct {
		days      int
		soon, due bool
	}{{-1, false, true}, {0, true, false}, {1, true, false}, {3, true, false}, {4, false, false}}
	for i, p := range loaded.Payments {
		if p.DaysUntilDue != want[i].days || p.DueSoon != want[i].soon || p.IsOverdue != want[i].due {
			t.Fatalf("%s: days %d soon %v overdue %v", p.DueOn, p.DaysUntilDue, p.DueSoon, p.IsOverdue)
		}
	}

	cash, err := analytics.NewRepository(pool).CashPosition(ctx, today, "2026-10-31")
	if err != nil || len(cash.Currencies) != 1 {
		t.Fatalf("cash: %+v %v", cash, err)
	}
	rub := cash.Currencies[0]
	timings := []string{}
	for _, p := range rub.ScheduledPayments {
		timings = append(timings, p.Timing)
	}
	if strings.Join(timings, ",") != "overdue,due_soon,due_soon,due_soon,upcoming" || rub.ScheduledPayments[3].DaysUntilDue != 3 {
		t.Fatalf("timings: %v", timings)
	}
	// «Скоро» is still upcoming money.
	if rub.OverdueObligationsMinor != 1000 || rub.UpcomingObligationsMinor != 4000 || rub.ObligationsThroughMinor != 5000 {
		t.Fatalf("totals: %+v", rub)
	}
}
