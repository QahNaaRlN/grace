//go:build integration

package assistanttest

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/QahNaaRlN/grace/internal/accounts"
	"github.com/QahNaaRlN/grace/internal/analytics"
	"github.com/QahNaaRlN/grace/internal/assistant"
	"github.com/QahNaaRlN/grace/internal/obligations"
	"github.com/QahNaaRlN/grace/internal/platform/access"
	"github.com/QahNaaRlN/grace/internal/platform/database"
	"github.com/QahNaaRlN/grace/internal/receipts"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestReadToolsAgainstPostgreSQL(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Fatal("TEST_DATABASE_URL is required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	ctx = access.WithPrincipal(ctx, access.Principal{ID: access.BootstrapOwnerID, Role: "admin"})
	pool, err := database.Open(ctx, databaseURL)
	if err != nil {
		t.Fatalf("open integration database: %v", err)
	}
	defer pool.Close()

	executor := assistant.NewExecutor(
		accounts.NewRepository(pool),
		analytics.NewRepository(pool),
		obligations.NewRepository(pool),
		receipts.NewRepository(pool),
	)
	before := financialRowCounts(t, ctx, pool)

	accountCall, err := executor.Call(ctx, "get_accounts", json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("get_accounts: %v", err)
	}
	accountResult := accountCall.Data.(assistant.AccountsResult)
	if len(accountResult.Items) != 1 || accountResult.Items[0].Currency != "RUB" || accountResult.Items[0].BalanceMinor != 12500000 {
		t.Fatalf("unexpected accounts result: %#v", accountResult)
	}

	periodCall, err := executor.Call(ctx, "get_period_summary", json.RawMessage(`{"from":"2026-10-01T00:00:00+05:00","to":"2026-11-01T00:00:00+05:00"}`))
	if err != nil {
		t.Fatalf("get_period_summary: %v", err)
	}
	period := periodCall.Data.(analytics.PeriodSummary)
	if len(period.Currencies) != 1 || period.Currencies[0].Currency != "RUB" || period.Currencies[0].OpeningBalanceMinor != 12500000 || period.Currencies[0].ClosingBalanceMinor != 12500000 {
		t.Fatalf("unexpected period summary: %#v", period)
	}

	cashCall, err := executor.Call(ctx, "get_cash_position", json.RawMessage(`{"asOf":"2026-10-04","through":"2026-11-01"}`))
	if err != nil {
		t.Fatalf("get_cash_position: %v", err)
	}
	cash := cashCall.Data.(analytics.CashPosition)
	if len(cash.Currencies) != 1 || cash.Currencies[0].LiquidBalanceMinor != 12500000 || cash.Currencies[0].AvailableAfterObligationsMinor != 12500000 {
		t.Fatalf("unexpected cash position: %#v", cash)
	}

	obligationCall, err := executor.Call(ctx, "get_obligations", json.RawMessage(`{"asOf":"2026-10-04"}`))
	if err != nil {
		t.Fatalf("get_obligations: %v", err)
	}
	if obligationsResult := obligationCall.Data.(assistant.ObligationsResult); obligationsResult.AsOf != "2026-10-04" || len(obligationsResult.Items) != 0 {
		t.Fatalf("unexpected obligations result: %#v", obligationsResult)
	}

	receiptCall, err := executor.Call(ctx, "get_receipts", json.RawMessage(`{"limit":10}`))
	if err != nil {
		t.Fatalf("get_receipts: %v", err)
	}
	if receiptsResult := receiptCall.Data.(assistant.ReceiptsResult); len(receiptsResult.Items) != 0 {
		t.Fatalf("unexpected receipts result: %#v", receiptsResult)
	}

	for _, result := range []assistant.ToolResult{accountCall, periodCall, cashCall, obligationCall, receiptCall} {
		if payload, err := json.Marshal(result); err != nil || !json.Valid(payload) {
			t.Fatalf("tool result must be valid JSON: %s %v", payload, err)
		}
	}
	after := financialRowCounts(t, ctx, pool)
	if before != after {
		t.Fatalf("read tools changed financial records: before=%s after=%s", before, after)
	}
}

func financialRowCounts(t *testing.T, ctx context.Context, pool *pgxpool.Pool) string {
	t.Helper()
	var counts string
	err := pool.QueryRow(ctx, `
		SELECT concat_ws(',',
			(SELECT count(*) FROM accounts),
			(SELECT count(*) FROM financial_events),
			(SELECT count(*) FROM postings),
			(SELECT count(*) FROM receipts),
			(SELECT count(*) FROM receipt_items),
			(SELECT count(*) FROM obligations),
			(SELECT count(*) FROM scheduled_payments),
			(SELECT count(*) FROM scheduled_payment_settlements)
		)`).Scan(&counts)
	if err != nil {
		t.Fatalf("count financial rows: %v", err)
	}
	return counts
}
