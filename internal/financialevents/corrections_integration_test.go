//go:build integration

package financialevents_test

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

// SPEC-0041: corrections change balances and analytics exactly once, voided
// operations stop counting everywhere, links block money changes, and the
// database itself refuses to resurrect or strip a live operation.
func TestCorrectionsAndVoidKeepMoneyConsistent(t *testing.T) {
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
	var owner, other, card, cash, food string
	for _, dest := range []*string{&owner, &other} {
		if err = pool.QueryRow(bg, "INSERT INTO users(display_name,role) VALUES('Synthetic corrections','member') RETURNING id::text").Scan(dest); err != nil {
			t.Fatal(err)
		}
	}
	for _, spec := range []struct {
		dest *string
		kind string
	}{{&card, "debit"}, {&cash, "cash"}} {
		if err = pool.QueryRow(bg, "INSERT INTO accounts(owner_id,name,type,currency,opening_balance_minor,balance_reconciled) VALUES($1,$2,$3,'RUB',0,true) RETURNING id::text", owner, spec.kind, spec.kind).Scan(spec.dest); err != nil {
			t.Fatal(err)
		}
	}
	if err = pool.QueryRow(bg, "SELECT id::text FROM categories WHERE owner_id IS NULL AND kind='expense' LIMIT 1").Scan(&food); err != nil {
		t.Fatal(err)
	}
	ctx := access.WithPrincipal(bg, access.Principal{ID: owner, Role: "member"})
	otherCtx := access.WithPrincipal(bg, access.Principal{ID: other, Role: "member"})
	events := financialevents.NewRepository(pool)
	accountRepo := accounts.NewRepository(pool)
	analyticsRepo := analytics.NewRepository(pool)
	at, _ := time.Parse(time.RFC3339, "2026-10-05T12:00:00+05:00")
	from, _ := time.Parse(time.RFC3339, "2026-10-01T00:00:00+05:00")
	to, _ := time.Parse(time.RFC3339, "2026-11-01T00:00:00+05:00")
	balance := func(id string) int64 {
		t.Helper()
		items, err := accountRepo.List(ctx)
		if err != nil {
			t.Fatal(err)
		}
		for _, item := range items {
			if item.ID == id && item.BalanceMinor != nil {
				return *item.BalanceMinor
			}
		}
		t.Fatalf("account %s not listed", id)
		return 0
	}
	summary := func() analytics.CurrencyPeriodSummary {
		t.Helper()
		value, err := analyticsRepo.PeriodSummary(ctx, from, to)
		if err != nil || len(value.Currencies) != 1 {
			t.Fatalf("summary %+v %v", value, err)
		}
		return value.Currencies[0]
	}
	purchase, _, err := events.Create(ctx, financialevents.Command{Kind: "purchase", CategoryID: &food, AccountID: card, AmountMinor: 10000, OccurredAt: at, Merchant: "Магазин", IdempotencyKey: "c-purchase", RequestHash: strings.Repeat("e", 64)})
	if err != nil {
		t.Fatal(err)
	}
	if balance(card) != -10000 || summary().PurchasesMinor != 10000 {
		t.Fatal("baseline")
	}

	// Descriptive edit, idempotent replay, stale revision.
	edited, err := events.Correct(ctx, purchase.ID, financialevents.Correction{Revision: 1, IdempotencyKey: "k1", Description: ptr("Продукты на неделю")})
	if err != nil || edited.Revision != 2 || !edited.EditedByUser || edited.Description != "Продукты на неделю" {
		t.Fatalf("describe: %+v %v", edited, err)
	}
	if replay, err := events.Correct(ctx, purchase.ID, financialevents.Correction{Revision: 1, IdempotencyKey: "k1", Description: ptr("Продукты на неделю")}); err != nil || replay.Revision != 2 {
		t.Fatal("replay must return the stored result", replay.Revision, err)
	}
	if _, err := events.Correct(ctx, purchase.ID, financialevents.Correction{Revision: 1, IdempotencyKey: "k-stale", Merchant: ptr("X")}); !errors.Is(err, financialevents.ErrRevisionConflict) {
		t.Fatal("stale revision accepted", err)
	}

	// Amount and account change the balances once.
	if _, err := events.Correct(ctx, purchase.ID, financialevents.Correction{Revision: 2, IdempotencyKey: "k2", AmountMinor: ptr[int64](25000), AccountID: &cash}); err != nil {
		t.Fatal(err)
	}
	if balance(card) != 0 || balance(cash) != -25000 || summary().PurchasesMinor != 25000 {
		t.Fatalf("after amount/account: card %d cash %d purchases %d", balance(card), balance(cash), summary().PurchasesMinor)
	}

	// Kind: a purchase recorded by mistake becomes income; category is cleared.
	asIncome, err := events.Correct(ctx, purchase.ID, financialevents.Correction{Revision: 3, IdempotencyKey: "k3", Kind: ptr("income_received")})
	if err != nil || asIncome.Kind != "income_received" || asIncome.CategoryID != nil || asIncome.Postings[0].Direction != "credit" {
		t.Fatalf("kind: %+v %v", asIncome, err)
	}
	if balance(cash) != 25000 || summary().IncomeMinor != 25000 || summary().PurchasesMinor != 0 {
		t.Fatal("income after kind change", balance(cash), summary())
	}
	if _, err := events.Correct(ctx, purchase.ID, financialevents.Correction{Revision: 4, IdempotencyKey: "k4", Kind: ptr("purchase")}); !errors.Is(err, financialevents.ErrInvalid) {
		t.Fatal("purchase without a category accepted", err)
	}
	history, err := events.History(ctx, purchase.ID)
	if err != nil || len(history) != 3 || history[0].Changes[0].Field != "kind" {
		t.Fatalf("history: %+v %v", history, err)
	}

	// Transfers: amount moves both postings, accounts must differ.
	transfer, _, err := events.Create(ctx, financialevents.Command{Kind: "internal_transfer", AccountID: card, DestinationAccountID: cash, AmountMinor: 1000, OccurredAt: at, IdempotencyKey: "c-transfer", RequestHash: strings.Repeat("f", 64)})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := events.Correct(ctx, transfer.ID, financialevents.Correction{Revision: 1, IdempotencyKey: "t1", AmountMinor: ptr[int64](4000)}); err != nil {
		t.Fatal(err)
	}
	if balance(card) != -4000 || balance(cash) != 29000 {
		t.Fatal("transfer amount", balance(card), balance(cash))
	}
	if _, err := events.Correct(ctx, transfer.ID, financialevents.Correction{Revision: 2, IdempotencyKey: "t2", DestinationAccountID: &card}); !errors.Is(err, financialevents.ErrInvalid) {
		t.Fatal("transfer into the same account accepted", err)
	}

	// Links block money changes and voiding.
	bought, _, err := events.Create(ctx, financialevents.Command{Kind: "purchase", CategoryID: &food, AccountID: card, AmountMinor: 3000, OccurredAt: at, Merchant: "Возвратный", IdempotencyKey: "c-bought", RequestHash: strings.Repeat("g", 64)})
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err = events.Create(ctx, financialevents.Command{Kind: "refund", OriginalEventID: bought.ID, AmountMinor: 1000, OccurredAt: at, IdempotencyKey: "c-refund", RequestHash: strings.Repeat("h", 64)}); err != nil {
		t.Fatal(err)
	}
	var linked *financialevents.LinkError
	if _, err := events.Correct(ctx, bought.ID, financialevents.Correction{Revision: 1, IdempotencyKey: "l1", AmountMinor: ptr[int64](5000)}); !errors.As(err, &linked) || linked.Links[0] != "возврат по этой покупке" {
		t.Fatal("amount of a refunded purchase changed", err)
	}
	if _, err := events.Void(ctx, bought.ID, 1, "ошибка", "l2"); !errors.As(err, &linked) {
		t.Fatal("refunded purchase voided", err)
	}
	if _, err := events.Correct(ctx, bought.ID, financialevents.Correction{Revision: 1, IdempotencyKey: "l3", Merchant: ptr("Возвратный магазин")}); err != nil {
		t.Fatal("description of a linked purchase must stay editable", err)
	}

	// Void: out of balances, analytics, cash position and lists; kept on record.
	before := balance(card)
	cashBefore, _ := analyticsRepo.CashPosition(ctx, "2026-10-13", "2026-10-31")
	victim, _, err := events.Create(ctx, financialevents.Command{Kind: "purchase", CategoryID: &food, AccountID: card, AmountMinor: 7000, OccurredAt: at, Merchant: "Ошибочная", IdempotencyKey: "c-victim", RequestHash: strings.Repeat("i", 64)})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := events.Void(ctx, victim.ID, 1, "", "v0"); !errors.Is(err, financialevents.ErrInvalid) {
		t.Fatal("void without a reason accepted", err)
	}
	voided, err := events.Void(ctx, victim.ID, 1, "Дубль из выписки", "v1")
	if err != nil || voided.VoidedAt == nil || *voided.VoidReason != "Дубль из выписки" || len(voided.Postings) != 1 {
		t.Fatalf("void: %+v %v", voided, err)
	}
	cashAfter, _ := analyticsRepo.CashPosition(ctx, "2026-10-13", "2026-10-31")
	if balance(card) != before || *cashAfter.Currencies[0].LiquidBalanceMinor != *cashBefore.Currencies[0].LiquidBalanceMinor {
		t.Fatal("voided money still counted", balance(card), before)
	}
	live, _ := events.ListPage(ctx, financialevents.ListFilter{Limit: 100})
	for _, item := range live.Items {
		if item.ID == victim.ID {
			t.Fatal("voided operation in the live list")
		}
	}
	gone, _ := events.ListPage(ctx, financialevents.ListFilter{Limit: 100, Voided: true})
	if len(gone.Items) != 1 || gone.Items[0].ID != victim.ID {
		t.Fatalf("voided list: %+v", gone.Items)
	}
	if _, err := events.Correct(ctx, victim.ID, financialevents.Correction{Revision: 2, IdempotencyKey: "v2", Merchant: ptr("Ещё")}); !errors.Is(err, financialevents.ErrVoided) {
		t.Fatal("voided operation edited", err)
	}

	// The database refuses what the module never does.
	tx, err := database.ForUsers(pool).BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		t.Fatal(err)
	}
	tag, err := tx.Exec(ctx, "DELETE FROM postings WHERE financial_event_id=$1", transfer.ID)
	if err != nil || tag.RowsAffected() != 0 {
		t.Fatal("postings of a live operation deletable", tag.RowsAffected(), err)
	}
	_, err = tx.Exec(ctx, "UPDATE financial_events SET voided_at=NULL, void_reason=NULL WHERE id=$1", victim.ID)
	_ = tx.Rollback(ctx)
	if err == nil {
		t.Fatal("voided operation resurrected")
	}
	if _, err := events.Correct(otherCtx, transfer.ID, financialevents.Correction{Revision: 2, IdempotencyKey: "o1", Merchant: ptr("чужое")}); !errors.Is(err, financialevents.ErrNotFound) {
		t.Fatal("foreign correction allowed", err)
	}
}
