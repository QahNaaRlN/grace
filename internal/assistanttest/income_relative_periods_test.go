//go:build integration

package assistanttest

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/QahNaaRlN/grace/internal/accounts"
	"github.com/QahNaaRlN/grace/internal/analytics"
	"github.com/QahNaaRlN/grace/internal/assistant"
	"github.com/QahNaaRlN/grace/internal/financialevents"
	"github.com/QahNaaRlN/grace/internal/obligations"
	"github.com/QahNaaRlN/grace/internal/platform/access"
	"github.com/QahNaaRlN/grace/internal/platform/database"
	"github.com/QahNaaRlN/grace/internal/receipts"
)

func TestIncomeRelativePeriodsAgainstPostgreSQL(t *testing.T) {
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

	const (
		accountID   = "10000000-0000-0000-0000-000000000001"
		salaryID    = "20000000-0000-0000-0000-000000000004"
		advanceID   = "20000000-0000-0000-0000-000000000005"
		oldIncomeID = "20000000-0000-0000-0000-000000000006"
		expenseID   = "20000000-0000-0000-0000-000000000001"
	)
	if _, err := pool.Exec(ctx, `
		INSERT INTO categories (id, name, kind) VALUES
			($1, 'Аванс', 'income'),
			($2, 'Старый доход', 'income')`, advanceID, oldIncomeID); err != nil {
		t.Fatalf("create income categories: %v", err)
	}

	events := financialevents.NewRepository(pool)
	salaryCategory, advanceCategory, expenseCategory := salaryID, advanceID, expenseID
	createEvent := func(sequence int, kind string, categoryID *string, amount int64, occurredAt time.Time) financialevents.Event {
		t.Helper()
		event, created, err := events.Create(ctx, financialevents.Command{
			Kind: kind, IdempotencyKey: fmt.Sprintf("spec-0007-%d", sequence),
			RequestHash: strings.Repeat(fmt.Sprintf("%x", sequence%16), 64),
			AccountID:   accountID, CategoryID: categoryID, AmountMinor: amount,
			Merchant: "Integration fixture", Description: "SPEC-0007",
			OccurredAt: occurredAt,
		})
		if err != nil || !created {
			t.Fatalf("create event %d: created=%v err=%v", sequence, created, err)
		}
		createdAt := time.Date(2026, 1, 1, 0, 0, sequence, 0, time.UTC)
		if _, err := pool.Exec(ctx, "UPDATE financial_events SET created_at = $1 WHERE id = $2", createdAt, event.ID); err != nil {
			t.Fatalf("stabilize event %d creation time: %v", sequence, err)
		}
		return event
	}

	createEvent(1, "income_received", &salaryCategory, 100_000, time.Date(2026, 9, 25, 8, 0, 0, 0, time.UTC))
	createEvent(2, "income_received", &advanceCategory, 40_000, time.Date(2026, 10, 5, 8, 0, 0, 0, time.UTC))
	latestAdvance := createEvent(3, "income_received", &advanceCategory, 10_000, time.Date(2026, 10, 5, 8, 0, 0, 0, time.UTC))
	createEvent(4, "purchase", &expenseCategory, 12_500, time.Date(2026, 10, 6, 9, 0, 0, 0, time.UTC))
	createEvent(5, "income_received", &salaryCategory, 999_000, time.Date(2026, 10, 20, 8, 0, 0, 0, time.UTC))
	uncategorized := createEvent(6, "income_received", nil, 5_000, time.Date(2026, 10, 7, 8, 0, 0, 0, time.UTC))
	oldIncomeCategory := oldIncomeID
	createEvent(7, "income_received", &oldIncomeCategory, 1_000, time.Date(2020, 10, 1, 8, 0, 0, 0, time.UTC))

	analyticsRepo := analytics.NewRepository(pool)
	executor := assistant.NewExecutor(
		accounts.NewRepository(pool), analyticsRepo,
		obligations.NewRepository(pool), receipts.NewRepository(pool),
	)
	before := financialRowCounts(t, ctx, pool)

	call, err := executor.Call(ctx, "get_period_since_income", json.RawMessage(`{
		"asOf":"2026-10-10T12:00:00Z",
		"categoryId":"20000000-0000-0000-0000-000000000005"
	}`))
	if err != nil {
		t.Fatalf("get_period_since_income: %v", err)
	}
	result := call.Data.(analytics.IncomeRelativePeriod)
	if result.Anchor.EventID != latestAdvance.ID || result.Anchor.CategoryID == nil || *result.Anchor.CategoryID != advanceID ||
		result.Anchor.CategoryName == nil || *result.Anchor.CategoryName != "Аванс" || len(result.Anchor.Postings) != 1 ||
		result.Anchor.Postings[0].AmountMinor != 10_000 || result.Anchor.Postings[0].Currency != "RUB" {
		t.Fatalf("unexpected income anchor: %#v", result.Anchor)
	}
	if !result.Summary.From.Equal(result.Anchor.OccurredAt) || !result.Summary.To.Equal(time.Date(2026, 10, 10, 12, 0, 0, 0, time.UTC)) {
		t.Fatalf("unexpected summary boundaries: %#v", result.Summary)
	}
	if len(result.Summary.Currencies) != 1 || result.Summary.Currencies[0].IncomeMinor != 55_000 ||
		result.Summary.Currencies[0].PurchasesMinor != 12_500 || result.Summary.Currencies[0].NetExternalCashFlowMinor != 42_500 {
		t.Fatalf("anchor must be included in canonical summary: %#v", result.Summary)
	}

	withoutCategory, err := analyticsRepo.IncomeRelativePeriod(ctx, time.Date(2026, 10, 10, 12, 0, 0, 0, time.UTC), nil)
	if err != nil || withoutCategory.Anchor.EventID != uncategorized.ID || withoutCategory.Anchor.CategoryID != nil {
		t.Fatalf("unexpected unfiltered anchor: %#v %v", withoutCategory.Anchor, err)
	}
	atExclusiveBoundary, err := analyticsRepo.IncomeRelativePeriod(ctx, uncategorized.OccurredAt, nil)
	if err != nil || atExclusiveBoundary.Anchor.EventID != latestAdvance.ID {
		t.Fatalf("asOf must exclude an anchor at the boundary: %#v %v", atExclusiveBoundary.Anchor, err)
	}

	_, err = analyticsRepo.IncomeRelativePeriod(ctx, time.Date(2026, 10, 10, 12, 0, 0, 0, time.UTC), &expenseCategory)
	assertAnchorCode(t, err, analytics.CodeInvalidAnchorCategory)
	unknownCategory := "20000000-0000-0000-0000-000000000099"
	_, err = analyticsRepo.IncomeRelativePeriod(ctx, time.Date(2026, 10, 10, 12, 0, 0, 0, time.UTC), &unknownCategory)
	assertAnchorCode(t, err, analytics.CodeInvalidAnchorCategory)
	_, err = analyticsRepo.IncomeRelativePeriod(ctx, time.Date(2026, 10, 10, 12, 0, 0, 0, time.UTC), &oldIncomeCategory)
	assertAnchorCode(t, err, analytics.CodeAnchorNotFound)

	after := financialRowCounts(t, ctx, pool)
	if before != after {
		t.Fatalf("income-relative reads changed financial records: before=%s after=%s", before, after)
	}
}

func assertAnchorCode(t *testing.T, err error, expected analytics.IncomeAnchorErrorCode) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected %s", expected)
	}
	var anchorErr *analytics.IncomeAnchorError
	if !errors.As(err, &anchorErr) || anchorErr.Code != expected {
		t.Fatalf("expected %s, got %v", expected, err)
	}
}
