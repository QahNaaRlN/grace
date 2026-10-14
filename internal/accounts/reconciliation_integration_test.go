//go:build integration

package accounts_test

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/QahNaaRlN/grace/internal/accounts"
	"github.com/QahNaaRlN/grace/internal/analytics"
	"github.com/QahNaaRlN/grace/internal/financialevents"
	"github.com/QahNaaRlN/grace/internal/platform/access"
	"github.com/QahNaaRlN/grace/internal/platform/database"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func ptr[T any](value T) *T { return &value }

// SPEC-0040: details and cards are editable, a reconciliation sets the
// opening of an unreconciled account, books a difference as a balance
// adjustment that analytics ignores, and is append-only, idempotent and
// guarded by its preview hash.
func TestAccountEditingAndReconciliation(t *testing.T) {
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
	var owner, other, food string
	for _, dest := range []*string{&owner, &other} {
		if err = pool.QueryRow(bg, "INSERT INTO users(display_name,role) VALUES('Synthetic reconciliation','member') RETURNING id::text").Scan(dest); err != nil {
			t.Fatal(err)
		}
	}
	if err = pool.QueryRow(bg, "SELECT id::text FROM categories WHERE owner_id IS NULL AND kind='expense' LIMIT 1").Scan(&food); err != nil {
		t.Fatal(err)
	}
	ctx := access.WithPrincipal(bg, access.Principal{ID: owner, Role: "member"})
	otherCtx := access.WithPrincipal(bg, access.Principal{ID: other, Role: "member"})
	repo := accounts.NewRepository(pool)
	events := financialevents.NewRepository(pool)
	analyticsRepo := analytics.NewRepository(pool)
	find := func(id string) accounts.Account {
		t.Helper()
		items, err := repo.List(ctx)
		if err != nil {
			t.Fatal(err)
		}
		for _, item := range items {
			if item.ID == id {
				return item
			}
		}
		t.Fatalf("account %s not listed", id)
		return accounts.Account{}
	}

	// An unreconciled card account with history.
	card, err := repo.Create(ctx, accounts.CreateRequest{Name: "Сбербанк — счёт 3021", Type: "debit", Currency: "RUB", BalanceReconciled: ptr(false),
		Card: &accounts.CardRequest{Name: "Карта", Bank: "Сбербанк", Last4: "3021"}})
	if err != nil {
		t.Fatal(err)
	}
	at := func(s string) time.Time { v, _ := time.Parse(time.RFC3339, s); return v }
	if _, _, err = events.Create(ctx, financialevents.Command{Kind: "income_received", AccountID: card.ID, AmountMinor: 500000, OccurredAt: at("2026-10-01T10:00:00+05:00"), IdempotencyKey: "r-income", RequestHash: strings.Repeat("a", 64)}); err != nil {
		t.Fatal(err)
	}
	if _, _, err = events.Create(ctx, financialevents.Command{Kind: "purchase", CategoryID: &food, AccountID: card.ID, AmountMinor: 120000, OccurredAt: at("2026-10-02T10:00:00+05:00"), IdempotencyKey: "r-buy", RequestHash: strings.Repeat("b", 64)}); err != nil {
		t.Fatal(err)
	}
	// A date-only purchase on the reconciliation day counts (decision 4).
	if _, _, err = events.Create(ctx, financialevents.Command{Kind: "purchase", CategoryID: &food, AccountID: card.ID, AmountMinor: 30000, OccurredDate: "2026-10-05", IdempotencyKey: "r-dateonly", RequestHash: strings.Repeat("c", 64)}); err != nil {
		t.Fatal(err)
	}
	// A later purchase is outside the moment.
	if _, _, err = events.Create(ctx, financialevents.Command{Kind: "purchase", CategoryID: &food, AccountID: card.ID, AmountMinor: 10000, OccurredAt: at("2026-10-06T09:00:00+05:00"), IdempotencyKey: "r-later", RequestHash: strings.Repeat("d", 64)}); err != nil {
		t.Fatal(err)
	}

	// AC1: details and cards.
	current := find(card.ID)
	if current.Revision != 1 || !current.HasOperations || current.LastReconciliation != nil {
		t.Fatalf("fresh account: %+v", current)
	}
	if _, err = repo.Update(ctx, card.ID, accounts.Update{Revision: 1, Type: ptr("credit")}); !errors.Is(err, accounts.ErrHasOperations) {
		t.Fatal("type changed with operations", err)
	}
	renamed, err := repo.Update(ctx, card.ID, accounts.Update{Revision: 1, Name: ptr("Сбер, зарплатный")})
	if err != nil || renamed.Name != "Сбер, зарплатный" || renamed.Revision != 2 {
		t.Fatalf("rename: %+v %v", renamed, err)
	}
	if _, err = repo.Update(ctx, card.ID, accounts.Update{Revision: 1, Name: ptr("X")}); !errors.Is(err, accounts.ErrRevisionConflict) {
		t.Fatal("stale revision accepted", err)
	}
	withCard, err := repo.AddCard(ctx, card.ID, accounts.CardChange{Name: ptr("Виртуальная"), Bank: ptr("Сбербанк"), Last4: ptr("7788")})
	if err != nil || len(withCard.PaymentInstruments) != 2 {
		t.Fatalf("add card: %+v %v", withCard, err)
	}
	first := withCard.PaymentInstruments[0].ID
	disabled, err := repo.UpdateCard(ctx, card.ID, first, accounts.CardChange{Last4: ptr("3022"), Disabled: ptr(true)})
	if err != nil {
		t.Fatal(err)
	}
	last := disabled.PaymentInstruments[len(disabled.PaymentInstruments)-1]
	if last.ID != first || last.Last4 != "3022" || last.DisabledAt == nil || disabled.PaymentInstruments[0].DisabledAt != nil {
		t.Fatalf("disabled card must be kept last: %+v", disabled.PaymentInstruments)
	}
	if _, err = repo.UpdateCard(ctx, card.ID, first, accounts.CardChange{Last4: ptr("12")}); !errors.Is(err, accounts.ErrInvalid) {
		t.Fatal("bad last4 accepted", err)
	}
	empty, err := repo.Create(ctx, accounts.CreateRequest{Name: "Пустой", Type: "debit", Currency: "RUB", OpeningBalanceMinor: ptr[int64](0)})
	if err != nil {
		t.Fatal(err)
	}
	if moved, err := repo.Update(ctx, empty.ID, accounts.Update{Revision: 1, Type: ptr("cash")}); err != nil || moved.Type != "cash" {
		t.Fatal("type of an account without operations must change", err)
	}
	if _, err = repo.AddCard(ctx, empty.ID, accounts.CardChange{Name: ptr("К"), Bank: ptr("Б"), Last4: ptr("1111")}); !errors.Is(err, accounts.ErrInvalid) {
		t.Fatal("card on a cash account accepted", err)
	}

	// AC2: unreconciled → opening derived, account reconciled.
	moment := at("2026-10-05T20:00:00+05:00")
	if _, err = repo.PreviewReconciliation(ctx, card.ID, time.Now().Add(time.Hour), 0); !errors.Is(err, accounts.ErrInvalid) {
		t.Fatal("future moment accepted", err)
	}
	preview, err := repo.PreviewReconciliation(ctx, card.ID, moment, 1000000)
	if err != nil || preview.Outcome != "opening_set" || preview.ComputedMinor != 350000 || *preview.ProposedOpeningMinor != 650000 {
		t.Fatalf("opening preview: %+v %v", preview, err)
	}
	cashBefore, _ := analyticsRepo.CashPosition(ctx, "2026-10-14", "2026-10-31")
	first1, account, err := repo.Reconcile(ctx, card.ID, "rec-1", accounts.ReconcileRequest{AsOf: moment, StatedMinor: 1000000, PreviewHash: preview.PreviewHash})
	if err != nil || first1.Outcome != "opening_set" || !account.BalanceReconciled || *account.BalanceMinor != 990000 {
		t.Fatalf("opening reconcile: %+v %+v %v", first1, account, err)
	}
	if account.LastReconciliation == nil || account.LastReconciliation.Stale || account.LastReconciliation.StatedMinor != 1000000 {
		t.Fatalf("last reconciliation: %+v", account.LastReconciliation)
	}
	cashAfter, _ := analyticsRepo.CashPosition(ctx, "2026-10-14", "2026-10-31")
	if before, after := cashBefore.Currencies[0].LiquidBalanceMinor, cashAfter.Currencies[0].LiquidBalanceMinor; after == nil || (before != nil && *before == *after) {
		t.Fatalf("liquid balance must become known: %v → %v", before, after)
	}
	replay, _, err := repo.Reconcile(ctx, card.ID, "rec-1", accounts.ReconcileRequest{AsOf: moment, StatedMinor: 1000000, PreviewHash: "anything"})
	if err != nil || replay.ID != first1.ID {
		t.Fatal("idempotent replay must return the stored reconciliation", err)
	}

	// AC3: reconciled with a difference → balance adjustment, not in analytics.
	moment2 := at("2026-10-07T12:00:00+05:00")
	stale, err := repo.PreviewReconciliation(ctx, card.ID, moment2, 975000)
	if err != nil || stale.Outcome != "adjusted" || stale.ComputedMinor != 990000 || stale.DifferenceMinor != -15000 {
		t.Fatalf("difference preview: %+v %v", stale, err)
	}
	if _, _, err = events.Create(ctx, financialevents.Command{Kind: "purchase", CategoryID: &food, AccountID: card.ID, AmountMinor: 5000, OccurredAt: at("2026-10-07T08:00:00+05:00"), IdempotencyKey: "r-between", RequestHash: strings.Repeat("e", 64)}); err != nil {
		t.Fatal(err)
	}
	if _, _, err = repo.Reconcile(ctx, card.ID, "rec-2", accounts.ReconcileRequest{AsOf: moment2, StatedMinor: 975000, PreviewHash: stale.PreviewHash}); !errors.Is(err, accounts.ErrStalePreview) {
		t.Fatal("stale preview accepted", err)
	}
	fresh, _ := repo.PreviewReconciliation(ctx, card.ID, moment2, 975000)
	summaryBefore, _ := analyticsRepo.PeriodSummary(ctx, at("2026-10-01T00:00:00+05:00"), at("2026-11-01T00:00:00+05:00"))
	adjusted, account, err := repo.Reconcile(ctx, card.ID, "rec-2", accounts.ReconcileRequest{AsOf: moment2, StatedMinor: 975000, PreviewHash: fresh.PreviewHash})
	if err != nil || adjusted.Outcome != "adjusted" || adjusted.DifferenceMinor != -10000 || adjusted.AdjustmentEventID == nil || *account.BalanceMinor != 975000 {
		t.Fatalf("adjust: %+v %+v %v", adjusted, account, err)
	}
	summaryAfter, _ := analyticsRepo.PeriodSummary(ctx, at("2026-10-01T00:00:00+05:00"), at("2026-11-01T00:00:00+05:00"))
	flows := func(c analytics.CurrencyPeriodSummary) [5]int64 {
		return [5]int64{c.IncomeMinor, c.PurchasesMinor, c.RefundsMinor, c.DebtPaymentsMinor, c.NetExternalCashFlowMinor}
	}
	if flows(summaryBefore.Currencies[0]) != flows(summaryAfter.Currencies[0]) {
		t.Fatalf("adjustment counted in analytics: %+v → %+v", summaryBefore.Currencies[0], summaryAfter.Currencies[0])
	}
	var linked *financialevents.LinkError
	if _, err = events.Void(ctx, *adjusted.AdjustmentEventID, 1, "ошибка", "void-adj"); !errors.As(err, &linked) || linked.Links[0] != "сверка остатка" {
		t.Fatal("adjustment of a reconciliation voided", err)
	}
	history, err := repo.Reconciliations(ctx, card.ID)
	if err != nil || len(history) != 2 || history[0].ID != adjusted.ID {
		t.Fatalf("history: %+v %v", history, err)
	}

	// Decision 2 (SPEC-0041): moving an operation into the reconciled period
	// marks the reconciliation stale.
	later, _, err := events.Create(ctx, financialevents.Command{Kind: "purchase", CategoryID: &food, AccountID: card.ID, AmountMinor: 2000, OccurredAt: at("2026-10-08T09:00:00+05:00"), IdempotencyKey: "r-move", RequestHash: strings.Repeat("f", 64)})
	if err != nil {
		t.Fatal(err)
	}
	if find(card.ID).LastReconciliation.Stale {
		t.Fatal("operation after the moment made the reconciliation stale")
	}
	if _, err = events.Correct(ctx, later.ID, financialevents.Correction{Revision: 1, IdempotencyKey: "move-1", OccurredAt: ptr(at("2026-10-06T09:00:00+05:00"))}); err != nil {
		t.Fatal(err)
	}
	if !find(card.ID).LastReconciliation.Stale {
		t.Fatal("reconciliation not stale after a date moved into its period")
	}

	// AC5: append-only and owner isolation.
	tx, err := database.ForUsers(pool).BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		t.Fatal(err)
	}
	_, updateErr := tx.Exec(ctx, "UPDATE account_reconciliations SET stated_minor=0")
	_ = tx.Rollback(ctx)
	tx, _ = database.ForUsers(pool).BeginTx(ctx, pgx.TxOptions{})
	_, deleteErr := tx.Exec(ctx, "DELETE FROM account_reconciliations")
	_ = tx.Rollback(ctx)
	if updateErr == nil || deleteErr == nil {
		t.Fatal("reconciliations are not append-only", updateErr, deleteErr)
	}
	if _, err = repo.PreviewReconciliation(otherCtx, card.ID, moment, 0); !errors.Is(err, accounts.ErrNotFound) {
		t.Fatal("foreign preview allowed", err)
	}
	if items, _ := repo.Reconciliations(otherCtx, card.ID); len(items) != 0 {
		t.Fatal("foreign reconciliations visible")
	}
	if _, err = repo.Update(otherCtx, card.ID, accounts.Update{Revision: 2, Name: ptr("чужой")}); !errors.Is(err, accounts.ErrNotFound) {
		t.Fatal("foreign edit allowed", err)
	}
}
