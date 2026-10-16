//go:build integration

package creditcards_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/QahNaaRlN/grace/internal/accounts"
	"github.com/QahNaaRlN/grace/internal/creditcards"
	"github.com/QahNaaRlN/grace/internal/financialevents"
	"github.com/QahNaaRlN/grace/internal/platform/access"
	"github.com/QahNaaRlN/grace/internal/platform/database"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func ptr[T any](v T) *T { return &v }

// SPEC-0051: terms versions with history, a statement-model status from the ledger,
// repayment as a transfer, bank snapshots and their comparison, idempotency,
// owner isolation and due dates for reminders.
func TestCreditCardTermsStatusAndSnapshots(t *testing.T) {
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
		if err = pool.QueryRow(bg, "INSERT INTO users(display_name,role) VALUES('Synthetic card owner','member') RETURNING id::text").Scan(dest); err != nil {
			t.Fatal(err)
		}
	}
	if err = pool.QueryRow(bg, "SELECT id::text FROM categories WHERE owner_id IS NULL AND kind='expense' LIMIT 1").Scan(&food); err != nil {
		t.Fatal(err)
	}
	ctx := access.WithPrincipal(bg, access.Principal{ID: owner, Role: "member"})
	otherCtx := access.WithPrincipal(bg, access.Principal{ID: other, Role: "member"})
	accountsRepo := accounts.NewRepository(pool)
	events := financialevents.NewRepository(pool)
	card, err := accountsRepo.Create(ctx, accounts.CreateRequest{Name: "Synthetic credit card", Type: "credit", Currency: "RUB", OpeningBalanceMinor: ptr(int64(0))})
	if err != nil {
		t.Fatal(err)
	}
	debit, err := accountsRepo.Create(ctx, accounts.CreateRequest{Name: "Synthetic salary", Type: "debit", Currency: "RUB", OpeningBalanceMinor: ptr(int64(5_000_000))})
	if err != nil {
		t.Fatal(err)
	}
	n := 0
	event := func(c financialevents.Command) {
		t.Helper()
		n++
		c.IdempotencyKey, c.RequestHash = "cc-"+string(rune('a'+n)), strings.Repeat(string(rune('a'+n%20)), 64)
		if _, _, err := events.Create(ctx, c); err != nil {
			t.Fatal(err)
		}
	}
	event(financialevents.Command{Kind: "purchase", CategoryID: &food, AccountID: card.ID, AmountMinor: 1_000_000, OccurredDate: "2026-09-03"})
	event(financialevents.Command{Kind: "purchase", CategoryID: &food, AccountID: card.ID, AmountMinor: 500_000, OccurredDate: "2026-09-05"})
	event(financialevents.Command{Kind: "purchase", CategoryID: &food, AccountID: card.ID, AmountMinor: 200_000, OccurredDate: "2026-09-06"})
	// Repaying the card is a transfer: debt goes down, nothing is counted twice.
	event(financialevents.Command{Kind: "internal_transfer", AccountID: debit.ID, DestinationAccountID: card.ID, AmountMinor: 600_000, OccurredDate: "2026-09-20"})

	router := chi.NewRouter()
	creditcards.NewHandler(pool).Routes(router)
	call := func(c context.Context, method, path, key string, body any) (*httptest.ResponseRecorder, creditcards.Card) {
		t.Helper()
		data, _ := json.Marshal(body)
		request := httptest.NewRequest(method, path, bytes.NewReader(data)).WithContext(c)
		if key != "" {
			request.Header.Set("Idempotency-Key", key)
		}
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, request)
		var out creditcards.Card
		_ = json.Unmarshal(recorder.Body.Bytes(), &out)
		return recorder, out
	}
	base := "/accounts/" + card.ID + "/credit-card"
	terms := map[string]any{"effectiveFrom": "2026-09-01", "limitMinor": 10_000_000, "graceModel": "statement", "statementDay": 5, "paymentDays": 25,
		"minPercentBasisPoints": 1400, "minAmountMinor": 60_000, "minRoundingMinor": 10_000, "minPercentIsCap": true, "note": "synthetic", "revision": 0}
	rec, created := call(ctx, "POST", base+"/terms", "terms-1", terms)
	if rec.Code != 201 || len(created.Terms) != 1 {
		t.Fatalf("create terms: %d %s", rec.Code, rec.Body)
	}
	if rec, _ = call(ctx, "POST", base+"/terms", "terms-1", terms); rec.Code != 200 {
		t.Fatalf("idempotent retry: %d", rec.Code)
	}
	rec, got := call(ctx, "GET", base+"?on=2026-09-25", "", nil)
	s := got.Status
	if rec.Code != 200 || s.Source != "ledger" || *s.DebtMinor != 1_100_000 || *s.AvailableMinor != 8_900_000 {
		t.Fatalf("status: %d %s", rec.Code, rec.Body)
	}
	if *s.GraceAmountMinor != 900_000 || *s.GraceDueOn != "2026-09-30" || s.GraceAtRisk {
		t.Fatalf("grace: %+v", s)
	}
	// 14% of 15 000 ₽ = 2 100 ₽, already covered by the 6 000 ₽ transfer.
	if *s.MinimumMinor != 0 || !s.MinimumIsUpperBound {
		t.Fatalf("minimum: %+v", s)
	}

	// A new version applies from its date; editing a version keeps history.
	later := map[string]any{"effectiveFrom": "2026-10-01", "limitMinor": 12_000_000, "graceModel": "statement", "statementDay": 5, "paymentDays": 25, "note": "", "revision": 0}
	if rec, _ = call(ctx, "POST", base+"/terms", "terms-2", later); rec.Code != 201 {
		t.Fatalf("second version: %d %s", rec.Code, rec.Body)
	}
	if _, got = call(ctx, "GET", base+"?on=2026-09-25", "", nil); *got.Status.AvailableMinor != 8_900_000 {
		t.Fatalf("September uses the first version: %+v", got.Status)
	}
	if _, got = call(ctx, "GET", base+"?on=2026-10-02", "", nil); *got.Status.AvailableMinor != 10_900_000 {
		t.Fatalf("October uses the second version: %+v", got.Status)
	}
	first := got.Terms[1]
	terms["limitMinor"], terms["revision"] = 11_000_000, first.Revision
	if rec, _ = call(ctx, "PATCH", base+"/terms/"+first.ID, "", terms); rec.Code != 200 {
		t.Fatalf("edit: %d %s", rec.Code, rec.Body)
	}
	if rec, _ = call(ctx, "PATCH", base+"/terms/"+first.ID, "", terms); rec.Code != 409 {
		t.Fatalf("stale revision must conflict: %d", rec.Code)
	}
	var history int
	tenants := database.ForUsers(pool)
	if err = tenants.QueryRow(ctx, "SELECT count(*) FROM credit_card_term_revisions WHERE terms_id::text=$1", first.ID).Scan(&history); err != nil || history != 1 {
		t.Fatalf("revision history %d %v", history, err)
	}

	// The bank snapshot never changes postings; the difference is shown.
	snapshot := map[string]any{"takenOn": "2026-09-25", "debtMinor": 1_150_000, "graceAmountMinor": 950_000, "graceDueOn": "2026-09-30", "note": ""}
	rec, got = call(ctx, "POST", base+"/snapshots", "snap-1", snapshot)
	if rec.Code != 201 || got.BankComparison == nil || got.BankComparison.DifferenceMinor != 50_000 || got.BankComparison.LedgerDebtMinor != 1_100_000 {
		t.Fatalf("snapshot: %d %s", rec.Code, rec.Body)
	}
	if rec, _ = call(ctx, "POST", base+"/snapshots", "snap-1", snapshot); rec.Code != 200 {
		t.Fatalf("idempotent snapshot: %d", rec.Code)
	}

	// Due dates feed the reminder worker: unpaid grace payment due on 30 Sep.
	tx, err := tenants.BeginTx(ctx, pgx.TxOptions{AccessMode: pgx.ReadOnly})
	if err != nil {
		t.Fatal(err)
	}
	day, _ := creditcards.ParseDay("2026-09-27")
	dates, err := creditcards.DueDates(ctx, tx, card.ID, day)
	tx.Rollback(ctx)
	if err != nil || len(dates) != 1 || dates[0] != "2026-09-30" {
		t.Fatalf("due dates %v %v", dates, err)
	}

	// Only credit accounts have card terms; other owners see nothing.
	if rec, _ = call(ctx, "POST", "/accounts/"+debit.ID+"/credit-card/terms", "terms-debit", terms); rec.Code != 422 {
		t.Fatalf("debit account: %d %s", rec.Code, rec.Body)
	}
	if rec, _ = call(otherCtx, "GET", base, "", nil); rec.Code != 409 {
		t.Fatalf("other owner: %d %s", rec.Code, rec.Body)
	}
	if rec, _ = call(ctx, "PUT", base+"/settings", "", map[string]any{"reminders": true}); rec.Code != 200 {
		t.Fatalf("settings: %d", rec.Code)
	}
	if _, got = call(ctx, "GET", base, "", nil); !got.Reminders {
		t.Fatal("reminders setting not saved")
	}

	// Purchases and transfers keep analytics honest: only the 17 000 ₽ is spending.
	var spent int64
	if err = tenants.QueryRow(ctx, `SELECT COALESCE(sum(p.amount_minor),0) FROM postings p JOIN financial_events e ON e.id=p.financial_event_id WHERE e.kind='purchase'`).Scan(&spent); err != nil || spent != 1_700_000 {
		t.Fatalf("spending %d %v", spent, err)
	}
}
