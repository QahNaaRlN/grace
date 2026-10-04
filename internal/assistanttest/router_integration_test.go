//go:build integration

package assistanttest

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/QahNaaRlN/grace/internal/accounts"
	"github.com/QahNaaRlN/grace/internal/analytics"
	"github.com/QahNaaRlN/grace/internal/assistant"
	"github.com/QahNaaRlN/grace/internal/financialevents"
	"github.com/QahNaaRlN/grace/internal/obligations"
	"github.com/QahNaaRlN/grace/internal/platform/database"
	"github.com/QahNaaRlN/grace/internal/receipts"
	"os"
)

type integrationProvider struct{ judgment assistant.RoutingJudgment }

func (provider *integrationProvider) Judge(context.Context, assistant.Query) (assistant.RoutingJudgment, error) {
	return provider.judgment, nil
}

func TestReadOnlyRouterAgainstPostgreSQL(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Fatal("TEST_DATABASE_URL is required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	pool, err := database.Open(ctx, databaseURL)
	if err != nil {
		t.Fatalf("open integration database: %v", err)
	}
	defer pool.Close()

	categoryID := "20000000-0000-0000-0000-000000000004"
	eventRepository := financialevents.NewRepository(pool)
	_, created, err := eventRepository.Create(ctx, financialevents.Command{
		Kind: "income_received", IdempotencyKey: "spec-0008-income", RequestHash: strings.Repeat("8", 64),
		AccountID: "10000000-0000-0000-0000-000000000001", CategoryID: &categoryID, AmountMinor: 100000,
		Merchant: "Integration fixture", Description: "SPEC-0008", OccurredAt: time.Date(2026, 10, 4, 8, 0, 0, 0, time.UTC),
	})
	if err != nil || !created {
		t.Fatalf("create income fixture: created=%v err=%v", created, err)
	}

	executor := assistant.NewExecutor(accounts.NewRepository(pool), analytics.NewRepository(pool), obligations.NewRepository(pool), receipts.NewRepository(pool))
	provider := &integrationProvider{}
	router := assistant.NewRouter(provider, executor)
	query := assistant.Query{Utterance: "integration", Language: "ru", ReferenceTime: time.Date(2026, 10, 5, 12, 0, 0, 0, time.FixedZone("YEKT", 5*60*60))}
	before := financialRowCounts(t, ctx, pool)
	tests := []assistant.RoutingJudgment{
		{Route: assistant.RouteGetAccounts, RouteConfidence: .99, Arguments: assistant.ArgumentsNone, ArgumentConfidence: .99},
		{Route: assistant.RouteGetObligations, RouteConfidence: .99, Arguments: assistant.ArgumentsReferenceTime, ArgumentConfidence: .99},
		{Route: assistant.RouteGetReceipts, RouteConfidence: .99, Arguments: assistant.ArgumentsDefaults, ArgumentConfidence: .99},
		{Route: assistant.RouteGetPeriodSinceIncome, RouteConfidence: .99, Arguments: assistant.ArgumentsReferenceTime, ArgumentConfidence: .99},
	}
	for _, judgment := range tests {
		provider.judgment = judgment
		result := router.Query(ctx, query)
		if result.Status != assistant.StatusCompleted || result.Data == nil || result.Data.Name != string(judgment.Route) {
			t.Fatalf("route %s failed: %#v", judgment.Route, result)
		}
	}
	after := financialRowCounts(t, ctx, pool)
	if before != after {
		t.Fatalf("router reads changed financial rows: before=%s after=%s", before, after)
	}
}
