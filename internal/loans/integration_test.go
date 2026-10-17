//go:build integration

package loans_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/QahNaaRlN/grace/internal/accounts"
	"github.com/QahNaaRlN/grace/internal/analytics"
	"github.com/QahNaaRlN/grace/internal/commitments"
	"github.com/QahNaaRlN/grace/internal/financialevents"
	"github.com/QahNaaRlN/grace/internal/loans"
	"github.com/QahNaaRlN/grace/internal/obligations"
	"github.com/QahNaaRlN/grace/internal/platform/access"
	"github.com/QahNaaRlN/grace/internal/platform/database"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func ptr[T any](v T) *T { return &v }

type loanResponse struct {
	Revision int               `json:"revision"`
	Debt     *loans.DebtStatus `json:"debt"`
	Drafts   []struct {
		ID       string `json:"id"`
		Revision int    `json:"revision"`
	} `json:"drafts"`
}

// SPEC-0053 on a synthetic annuity loan: parts from the bank and from Grace's
// calculation, one charge split into a row and a prepayment, a separate
// prepayment, owner allocations with history, bank snapshots, statistics, void
// protection and owner isolation.
func TestLoanPrincipalInterestAndDebt(t *testing.T) {
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
	var owner, other string
	for _, dest := range []*string{&owner, &other} {
		if err = pool.QueryRow(bg, "INSERT INTO users(display_name,role) VALUES('Synthetic loan owner','member') RETURNING id::text").Scan(dest); err != nil {
			t.Fatal(err)
		}
	}
	var food string
	if err = pool.QueryRow(bg, "SELECT id::text FROM categories WHERE owner_id IS NULL AND kind='expense' LIMIT 1").Scan(&food); err != nil {
		t.Fatal(err)
	}
	ctx := access.WithPrincipal(bg, access.Principal{ID: owner, Role: "member"})
	otherCtx := access.WithPrincipal(bg, access.Principal{ID: other, Role: "member"})
	salary, err := accounts.NewRepository(pool).Create(ctx, accounts.CreateRequest{Name: "Synthetic salary", Type: "debit", Currency: "RUB", OpeningBalanceMinor: ptr(int64(50_000_000))})
	if err != nil {
		t.Fatal(err)
	}
	events := financialevents.NewRepository(pool)
	n := 0
	pay := func(kind, on string, amount int64) financialevents.Event {
		t.Helper()
		n++
		c := financialevents.Command{Kind: kind, AccountID: salary.ID, AmountMinor: amount, OccurredDate: on,
			IdempotencyKey: owner + "-loan-event-" + string(rune('a'+n)), RequestHash: strings.Repeat(string(rune('a'+n%20)), 64)}
		if kind == "purchase" {
			c.CategoryID = &food
		}
		e, _, err := events.Create(ctx, c)
		if err != nil {
			t.Fatal(err)
		}
		return e
	}

	router := chi.NewRouter()
	oh := obligations.NewHandler(obligations.NewRepository(pool))
	router.Post("/obligations", oh.Create)
	router.Get("/obligations/{obligationId}", oh.Get)
	router.Post("/obligations/{obligationId}/scheduled-payments/{paymentId}/settlements", oh.CreateSettlement)
	commitments.NewHandler(pool).Routes(router)
	loans.NewHandler(pool).Routes(router)
	call := func(c context.Context, method, path, key string, body any, out any) int {
		t.Helper()
		var reader *bytes.Reader
		if body != nil {
			data, _ := json.Marshal(body)
			reader = bytes.NewReader(data)
		} else {
			reader = bytes.NewReader(nil)
		}
		request := httptest.NewRequest(method, path, reader).WithContext(c)
		if key != "" {
			request.Header.Set("Idempotency-Key", key)
		}
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, request)
		if out != nil {
			_ = json.Unmarshal(recorder.Body.Bytes(), out)
		}
		if recorder.Code >= 400 {
			t.Logf("%s %s → %d %s", method, path, recorder.Code, recorder.Body)
		}
		return recorder.Code
	}

	// A synthetic 12-month annuity: 120 000 ₽ at 12%, 10 661,85 ₽ a month. The
	// bank schedule gives parts for the first two rows only.
	payments := make([]map[string]any, 12)
	for i := range payments {
		payments[i] = map[string]any{"dueOn": time.Date(2026, time.Month(2+i), 15, 0, 0, 0, 0, time.UTC).Format("2006-01-02"), "amountMinor": 1_066_185}
	}
	payments[0]["principalMinor"], payments[0]["interestMinor"], payments[0]["otherMinor"] = 943_884, 122_301, 0
	payments[1]["principalMinor"], payments[1]["interestMinor"], payments[1]["otherMinor"] = 964_408, 101_777, 0
	var loan obligations.Obligation
	if code := call(ctx, "POST", "/obligations", "loan-1", map[string]any{"name": "Synthetic loan", "kind": "loan", "currency": "RUB", "totalMinor": 12 * 1_066_185, "payments": payments}, &loan); code != 201 {
		t.Fatalf("create loan: %d", code)
	}
	if p := loan.Payments[0]; p.PrincipalMinor == nil || *p.PrincipalMinor != 943_884 || *p.PartsSource != "bank" || loan.Payments[2].PrincipalMinor != nil {
		t.Fatalf("parts of created rows: %+v", loan.Payments[:3])
	}
	installment := map[string]any{"name": "Synthetic installment", "kind": "installment", "currency": "RUB", "totalMinor": 1000,
		"payments": []map[string]any{{"dueOn": "2026-02-01", "amountMinor": 1000, "principalMinor": 1000, "interestMinor": 0, "otherMinor": 0}}}
	if code := call(ctx, "POST", "/obligations", "installment-1", installment, nil); code != 422 {
		t.Fatalf("parts on an installment must be rejected: %d", code)
	}
	base := "/loans/" + loan.ID
	var details loanResponse
	if code := call(ctx, "PATCH", base, "", map[string]any{"lender": "Synthetic bank", "agreementNumber": "", "agreementOn": "2026-01-15",
		"principalMinor": 12_000_000, "annualRateBasisPoints": 1200, "repaymentType": "annuity", "reminders": false, "revision": 0}, &details); code != 200 {
		t.Fatalf("save loan details: %d", code)
	}

	// Grace fills the rest of the rows by the annuity formula; the owner applies it.
	var draft commitments.ScheduleDraft
	if code := call(ctx, "POST", base+"/schedule-drafts", "", nil, &draft); code != 201 {
		t.Fatalf("copy schedule: %d", code)
	}
	var calc commitments.PartsCalculation
	if code := call(ctx, "POST", base+"/schedule-drafts/"+draft.ID+"/calculate-parts", "", map[string]any{"revision": draft.Revision}, &calc); code != 200 {
		t.Fatalf("calculate parts: %d", code)
	}
	if calc.PrincipalDifferenceMinor != 0 || *calc.Draft.Payments[0].PrincipalMinor != 943_884 || *calc.Draft.Payments[5].PartsSource != loans.SourceCalculated {
		t.Fatalf("calculation: %+v", calc)
	}
	for i, row := range calc.Rows {
		if row.Warning != nil {
			t.Fatalf("row %d warning %s", i, *row.Warning)
		}
	}
	if code := call(ctx, "POST", base+"/schedule-drafts/"+draft.ID+"/apply", "", map[string]any{"revision": calc.Draft.Revision, "confirm": true}, nil); code != 200 {
		t.Fatalf("apply schedule: %d", code)
	}
	call(ctx, "GET", "/obligations/"+loan.ID+"?asOf=2026-02-01", "", nil, &loan)
	if loan.Payments[11].PrincipalMinor == nil || *loan.Payments[11].PartsSource != "calculated" {
		t.Fatalf("applied parts: %+v", loan.Payments[11])
	}

	// February is paid by the schedule; March's charge pays the row and repays 10 000 ₽ early.
	february := pay("debt_payment", "2026-02-15", 1_066_185)
	march := pay("debt_payment", "2026-03-15", 2_066_185)
	settle := func(row int, event financialevents.Event, key string, prepay bool) int {
		return call(ctx, "POST", "/obligations/"+loan.ID+"/scheduled-payments/"+loan.Payments[row].ID+"/settlements", key,
			map[string]any{"financialEventId": event.ID, "prepayRemainder": prepay}, nil)
	}
	if code := settle(0, february, "settle-feb", false); code != 201 {
		t.Fatalf("settle February: %d", code)
	}
	if code := settle(1, march, "settle-mar-plain", false); code != 422 {
		t.Fatalf("an amount above the row must need confirmation: %d", code)
	}
	if code := settle(1, march, "settle-mar", true); code != 201 {
		t.Fatalf("settle March with the remainder as a prepayment: %d", code)
	}
	if code := settle(1, march, "settle-mar", true); code != 200 {
		t.Fatalf("idempotent settlement: %d", code)
	}

	// A separate early repayment with the owner's split.
	early := pay("debt_payment", "2026-03-20", 500_000)
	prepayment := map[string]any{"financialEventId": early.ID, "principalMinor": 490_000, "interestMinor": 10_000, "otherMinor": 0}
	var debt loans.DebtStatus
	if code := call(ctx, "POST", base+"/prepayments", "early-1", prepayment, &debt); code != 201 {
		t.Fatalf("prepayment: %d", code)
	}
	if code := call(ctx, "POST", base+"/prepayments", "early-1", prepayment, nil); code != 200 {
		t.Fatalf("idempotent prepayment: %d", code)
	}
	if code := call(ctx, "POST", base+"/prepayments", "early-1", map[string]any{"financialEventId": early.ID}, nil); code != 409 {
		t.Fatalf("changed payload with the same key: %d", code)
	}
	if code := call(ctx, "POST", base+"/prepayments", "early-2", map[string]any{"financialEventId": march.ID}, nil); code != 409 {
		t.Fatalf("an operation linked to a row cannot be linked again: %d", code)
	}
	purchase := pay("purchase", "2026-03-21", 100_000)
	if code := call(ctx, "POST", base+"/prepayments", "early-3", map[string]any{"financialEventId": purchase.ID}, nil); code != 422 {
		t.Fatalf("a purchase is not a debt payment: %d", code)
	}
	if code := call(ctx, "POST", base+"/prepayments", "early-4", map[string]any{"financialEventId": pay("debt_payment", "2026-03-22", 1000).ID, "principalMinor": 900}, nil); code != 422 {
		t.Fatalf("incomplete parts: %d", code)
	}

	// Outstanding principal: 120 000 − 9 438,84 − 9 644,08 − 10 000 − 4 900.
	var got loanResponse
	call(ctx, "GET", base+"?on=2026-03-31", "", nil, &got)
	d := got.Debt
	if d.Basis != loans.BasisIssued || d.PrincipalOutstandingMinor == nil || *d.PrincipalOutstandingMinor != 12_000_000-943_884-964_408-1_000_000-490_000 {
		t.Fatalf("outstanding: %+v", d)
	}
	if d.Paid.PrincipalMinor != 943_884+964_408+1_000_000+490_000 || d.Paid.InterestMinor != 122_301+101_777+10_000 || d.Paid.UnknownMinor != 0 {
		t.Fatalf("paid: %+v", d.Paid)
	}
	if !d.ScheduleOutdated || len(d.Prepayments) != 2 {
		t.Fatalf("after prepayments the schedule is outdated: %+v", d)
	}

	// The bank shows 8 ₽ more than Grace on 20.03; later values start from it.
	var snapshot loans.DebtStatus
	body := map[string]any{"takenOn": "2026-03-20", "principalMinor": 8_601_700, "payoffMinor": 8_650_000, "note": ""}
	if code := call(ctx, "POST", base+"/debt-snapshots", "snap-1", body, &snapshot); code != 201 {
		t.Fatalf("snapshot: %d", code)
	}
	call(ctx, "GET", base+"?on=2026-03-31", "", nil, &got)
	d = got.Debt
	if d.Basis != loans.BasisSnapshot || *d.PrincipalOutstandingMinor != 8_601_700 || *d.GracePrincipalOnSnapshot != 8_601_708 || *d.DiscrepancyMinor != 8 {
		t.Fatalf("snapshot comparison: %+v", d)
	}
	if code := call(ctx, "POST", base+"/debt-snapshots", "snap-2", body, nil); code != 409 {
		t.Fatalf("second snapshot on the same day: %d", code)
	}
	if code := call(ctx, "POST", base+"/debt-snapshots", "snap-3", map[string]any{"takenOn": loans.FormatDay(loans.Today().AddDate(0, 0, 2)), "principalMinor": 1}, nil); code != 422 {
		t.Fatalf("snapshot in the future: %d", code)
	}
	body["principalMinor"], body["revision"] = 8_601_708, d.LatestSnapshot.Revision
	if code := call(ctx, "PATCH", base+"/debt-snapshots/"+d.LatestSnapshot.ID, "", body, nil); code != 200 {
		t.Fatalf("edit snapshot: %d", code)
	}
	if code := call(ctx, "PATCH", base+"/debt-snapshots/"+d.LatestSnapshot.ID, "", body, nil); code != 409 {
		t.Fatalf("stale snapshot revision: %d", code)
	}

	// The owner's split replaces the default and keeps history; reset returns it.
	var allocation loans.AllocationView
	call(ctx, "GET", "/financial-events/"+february.ID+"/loan-allocation", "", nil, &allocation)
	if allocation.Schedule == nil || allocation.Schedule.Source != loans.AllocationSchedule || *allocation.Schedule.Parts != (loans.Parts{PrincipalMinor: 943_884, InterestMinor: 122_301}) {
		t.Fatalf("default allocation: %+v", allocation.Schedule)
	}
	call(ctx, "GET", "/financial-events/"+march.ID+"/loan-allocation", "", nil, &allocation)
	if allocation.AmountMinor != 2_066_185 || allocation.Prepayment == nil || allocation.Prepayment.AmountMinor != 1_000_000 {
		t.Fatalf("split operation: %+v", allocation)
	}
	override := map[string]any{"principalMinor": 900_000, "interestMinor": 166_185, "otherMinor": 0, "revision": 0}
	if code := call(ctx, "PUT", "/financial-events/"+february.ID+"/loan-allocation", "", override, &allocation); code != 200 || allocation.Schedule.Source != loans.AllocationOwner {
		t.Fatalf("owner allocation: %d %+v", code, allocation.Schedule)
	}
	if code := call(ctx, "PUT", "/financial-events/"+february.ID+"/loan-allocation", "", override, nil); code != 409 {
		t.Fatalf("stale allocation revision: %d", code)
	}
	override["revision"], override["interestMinor"] = 1, 1
	if code := call(ctx, "PUT", "/financial-events/"+february.ID+"/loan-allocation", "", override, nil); code != 422 {
		t.Fatalf("parts that do not add up: %d", code)
	}
	if code := call(ctx, "DELETE", "/financial-events/"+february.ID+"/loan-allocation?revision=1", "", nil, &allocation); code != 200 || allocation.Schedule.Source != loans.AllocationSchedule {
		t.Fatalf("reset allocation: %d %+v", code, allocation.Schedule)
	}
	var revisions int
	tenants := database.ForUsers(pool)
	if err = tenants.QueryRow(ctx, "SELECT count(*) FROM loan_allocation_revisions").Scan(&revisions); err != nil || revisions != 4 {
		t.Fatalf("allocation history %d %v", revisions, err)
	}
	if code := call(ctx, "GET", "/financial-events/"+purchase.ID+"/loan-allocation", "", nil, nil); code != 404 {
		t.Fatalf("operation without a loan: %d", code)
	}

	// Statistics split the payments; an unlinked debt payment stays unallocated.
	pay("debt_payment", "2026-03-25", 300_000)
	repo := analytics.NewRepository(pool)
	from, to := time.Date(2026, 2, 1, 0, 0, 0, 0, time.FixedZone("UTC+5", 5*3600)), time.Date(2026, 4, 1, 0, 0, 0, 0, time.FixedZone("UTC+5", 5*3600))
	summary, err := repo.PeriodSummary(ctx, from, to)
	if err != nil || len(summary.Currencies) != 1 {
		t.Fatalf("summary %+v %v", summary, err)
	}
	s := summary.Currencies[0]
	if s.DebtPrincipalMinor != 943_884+964_408+1_000_000+490_000 || s.DebtInterestMinor != 122_301+101_777+10_000 || s.DebtUnallocatedMinor != 300_000+1000 ||
		s.DebtPrincipalMinor+s.DebtInterestMinor+s.DebtOtherMinor+s.DebtUnallocatedMinor != s.DebtPaymentsMinor {
		t.Fatalf("summary split %+v", s)
	}
	statistics, err := repo.Statistics(ctx, from, to, "month")
	if err != nil {
		t.Fatal(err)
	}
	for _, bucket := range statistics.Currencies[0].Series {
		if bucket.DebtPrincipalMinor+bucket.DebtInterestMinor+bucket.DebtOtherMinor+bucket.DebtUnallocatedMinor != bucket.DebtPaymentsMinor {
			t.Fatalf("bucket split %+v", bucket)
		}
	}
	if march := statistics.Currencies[0].Series[1]; march.DebtInterestMinor != 101_777+10_000 {
		t.Fatalf("March interest %+v", march)
	}

	// Linked prepayments protect their operation; other owners see nothing.
	if _, err = events.Void(ctx, early.ID, early.Revision, "synthetic", "void-early"); !errors.Is(err, financialevents.ErrLinked) {
		t.Fatalf("void of a prepaid operation: %v", err)
	}
	if code := call(otherCtx, "GET", base, "", nil, nil); code != 409 {
		t.Fatalf("other owner reads the loan: %d", code)
	}
	if code := call(otherCtx, "GET", "/financial-events/"+february.ID+"/loan-allocation", "", nil, nil); code != 404 {
		t.Fatalf("other owner reads the allocation: %d", code)
	}
	if code := call(otherCtx, "POST", base+"/debt-snapshots", "other-snap", map[string]any{"takenOn": "2026-03-01", "principalMinor": 1}, nil); code != 409 {
		t.Fatalf("other owner writes a snapshot: %d", code)
	}
}
