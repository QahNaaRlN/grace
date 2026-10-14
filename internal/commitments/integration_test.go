//go:build integration

package commitments

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/QahNaaRlN/grace/internal/auth"
	"github.com/QahNaaRlN/grace/internal/categories"
	"github.com/QahNaaRlN/grace/internal/financialevents"
	"github.com/QahNaaRlN/grace/internal/obligations"
	"github.com/QahNaaRlN/grace/internal/platform/access"
	"github.com/QahNaaRlN/grace/internal/platform/database"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"mime/multipart"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type fakeMail struct {
	calls atomic.Int64
	fail  bool
}

func (m *fakeMail) Ready() bool { return true }
func (m *fakeMail) SendPaymentReminder(_ context.Context, email, due, link string) error {
	m.calls.Add(1)
	if !strings.HasPrefix(email, "synthetic-") || due == "" || !strings.HasPrefix(link, "https://grace.invalid/") {
		return errors.New("invalid mail")
	}
	if m.fail {
		return errors.New("ambiguous SMTP")
	}
	return nil
}

func TestCommitmentsIsolationScheduleAndReminderDurability(t *testing.T) {
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
	if _, err = pool.Exec(bg, `UPDATE users SET disabled=true WHERE display_name='Synthetic commitments'`); err != nil {
		t.Fatal(err)
	}
	var owner, other, account, loan string
	for _, dest := range []*string{&owner, &other} {
		if err = pool.QueryRow(bg, `INSERT INTO users(display_name,role) VALUES('Synthetic commitments','member') RETURNING id::text`).Scan(dest); err != nil {
			t.Fatal(err)
		}
	}
	ctx := access.WithPrincipal(bg, access.Principal{ID: owner})
	defer pool.Exec(bg, `UPDATE users SET disabled=true WHERE id=$1 OR id=$2`, owner, other)
	otherCtx := access.WithPrincipal(bg, access.Principal{ID: other})
	tenant := database.ForUsers(pool)
	if err = tenant.QueryRow(ctx, `INSERT INTO accounts(name,type,currency,opening_balance_minor) VALUES('Synthetic','debit','RUB',100000) RETURNING id::text`).Scan(&account); err != nil {
		t.Fatal(err)
	}
	if err = tenant.QueryRow(ctx, `INSERT INTO obligations(name,kind,currency,total_minor,idempotency_key,request_hash) VALUES('Synthetic loan','loan','RUB',2000,$1,repeat('a',64)) RETURNING id::text`, owner).Scan(&loan); err != nil {
		t.Fatal(err)
	}
	var p1, p2 string
	for i, p := range []*string{&p1, &p2} {
		if err = tenant.QueryRow(ctx, `INSERT INTO scheduled_payments(obligation_id,position,due_on,amount_minor) VALUES($1,$2,$3,1000) RETURNING id::text`, loan, i+1, fmt.Sprintf("2026-11-0%d", i+1)).Scan(p); err != nil {
			t.Fatal(err)
		}
	}
	router := chi.NewRouter()
	NewHandler(pool).Routes(router)
	call := func(ctx context.Context, method, path string, body any) *httptest.ResponseRecorder {
		data, _ := json.Marshal(body)
		req := httptest.NewRequest(method, path, bytes.NewReader(data)).WithContext(ctx)
		principal, _ := access.Current(ctx)
		req.Header.Set("Idempotency-Key", principal.ID+method+path)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		return rec
	}
	require := func(rec *httptest.ResponseRecorder, status int) {
		t.Helper()
		if rec.Code != status {
			t.Fatalf("status %d want %d: %s", rec.Code, status, rec.Body.String())
		}
	}
	var category, incomeCategory string
	if err = tenant.QueryRow(ctx, `SELECT id::text FROM categories WHERE kind='expense' LIMIT 1`).Scan(&category); err != nil {
		t.Fatal(err)
	}
	if err = tenant.QueryRow(ctx, `SELECT id::text FROM categories WHERE kind='income' LIMIT 1`).Scan(&incomeCategory); err != nil {
		t.Fatal(err)
	}
	// SPEC-0047: like a purchase, a subscription needs an expense category.
	uncategorized := Subscription{Name: "No category", AmountMinor: 100, Currency: "RUB", Period: "monthly", StartsOn: "2026-01-31"}
	require(call(ctx, "POST", "/subscriptions", uncategorized), 422)
	uncategorized.CategoryID = &incomeCategory
	require(call(ctx, "POST", "/subscriptions", uncategorized), 422)
	sub := Subscription{Name: "Synthetic subscription", AmountMinor: 1200, Currency: "RUB", Period: "monthly", StartsOn: "2026-01-31", AccountID: &account, Reminders: true, CategoryID: &category}
	rec := call(ctx, "POST", "/subscriptions", sub)
	require(rec, 201)
	if err = json.Unmarshal(rec.Body.Bytes(), &sub); err != nil {
		t.Fatal(err)
	}
	if sub.CategoryID == nil || *sub.CategoryID != category {
		t.Fatalf("category not stored: %v", sub.CategoryID)
	}
	// Identical create retries do not make another recurring charge.
	original := Subscription{Name: "Synthetic subscription", AmountMinor: 1200, Currency: "RUB", Period: "monthly", StartsOn: "2026-01-31", AccountID: &account, Reminders: true, CategoryID: &category}
	retry := call(ctx, "POST", "/subscriptions", original)
	require(retry, 200)
	var repeated Subscription
	json.Unmarshal(retry.Body.Bytes(), &repeated)
	if repeated.ID != sub.ID {
		t.Fatal("creation duplicated")
	}
	original.Name = "Conflicting create"
	require(call(ctx, "POST", "/subscriptions", original), 409)
	// Linking an old purchase returns its real history without another posting.
	purchase, _, err := financialevents.NewRepository(pool).Create(ctx, financialevents.Command{Kind: "purchase", AccountID: account, AmountMinor: 1200, OccurredAt: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC), IdempotencyKey: owner + "-purchase", RequestHash: strings.Repeat("d", 64), CategoryID: &category})
	if err != nil {
		t.Fatal(err)
	}
	link := map[string]any{"financialEventId": purchase.ID}
	require(call(ctx, "POST", "/subscriptions/"+sub.ID+"/expenses", link), 200)
	require(call(ctx, "POST", "/subscriptions/"+sub.ID+"/expenses", link), 200)
	rec = call(ctx, "GET", "/subscriptions/"+sub.ID+"/expenses", nil)
	require(rec, 200)
	var linked struct {
		Items []financialevents.Event `json:"items"`
	}
	json.Unmarshal(rec.Body.Bytes(), &linked)
	if len(linked.Items) != 1 || linked.Items[0].ID != purchase.ID {
		t.Fatal("linked history lost")
	}
	require(call(otherCtx, "GET", "/subscriptions/"+sub.ID+"/expenses", nil), 404)
	require(call(otherCtx, "PATCH", "/subscriptions/"+sub.ID, sub), 409)
	sub.Name = "Edited"
	require(call(ctx, "PATCH", "/subscriptions/"+sub.ID, sub), 200)
	require(call(ctx, "PATCH", "/subscriptions/"+sub.ID, sub), 409)
	// SPEC-0047: on update null keeps the category (older subscriptions still
	// pause); deleting a category moves its subscriptions to the target.
	var personal string
	if err = tenant.QueryRow(ctx, `INSERT INTO categories(name,kind,owner_id) VALUES('Сервисы','expense',grace_owner_id()) RETURNING id::text`).Scan(&personal); err != nil {
		t.Fatal(err)
	}
	sub.Revision++
	sub.CategoryID = &personal
	require(call(ctx, "PATCH", "/subscriptions/"+sub.ID, sub), 200)
	sub.Revision++
	sub.CategoryID = nil
	rec = call(ctx, "PATCH", "/subscriptions/"+sub.ID, sub)
	require(rec, 200)
	json.Unmarshal(rec.Body.Bytes(), &sub)
	if sub.CategoryID == nil || *sub.CategoryID != personal {
		t.Fatalf("null replaced the category: %v", sub.CategoryID)
	}
	if _, err = categories.NewRepository(pool).Delete(ctx, personal, category, 0); err != nil {
		t.Fatal(err)
	}
	var moved string
	if err = tenant.QueryRow(ctx, `SELECT category_id::text FROM subscriptions WHERE id::text=$1`, sub.ID).Scan(&moved); err != nil || moved != category {
		t.Fatalf("subscription not moved with its category: %q %v", moved, err)
	}
	// A foreign account cannot be bound to a new subscription.
	sub.CategoryID = &category
	sub.ID = ""
	require(call(otherCtx, "POST", "/subscriptions", sub), 422)
	var d ScheduleDraft
	rec = call(ctx, "POST", "/loans/"+loan+"/schedule-drafts", nil)
	require(rec, 201)
	json.Unmarshal(rec.Body.Bytes(), &d)
	d.Payments[1].AmountMinor = 1500
	rec = call(ctx, "PATCH", "/loans/"+loan+"/schedule-drafts/"+d.ID, map[string]any{"revision": d.Revision, "payments": d.Payments})
	require(rec, 200)
	json.Unmarshal(rec.Body.Bytes(), &d)
	require(call(ctx, "POST", "/loans/"+loan+"/schedule-drafts/"+d.ID+"/apply", map[string]any{"revision": d.Revision, "confirm": false}), 422)
	require(call(ctx, "POST", "/loans/"+loan+"/schedule-drafts/"+d.ID+"/apply", map[string]any{"revision": d.Revision, "confirm": true}), 200)
	result, err := obligations.NewRepository(pool).Get(ctx, loan, "2026-10-10")
	if err != nil || result.TotalMinor != 2500 || len(result.Payments) != 2 {
		t.Fatalf("active plan %+v %v", result, err)
	}
	var history int
	if err = tenant.QueryRow(ctx, `SELECT count(*) FROM scheduled_payments WHERE obligation_id=$1`, loan).Scan(&history); err != nil || history != 4 {
		t.Fatalf("history %d %v", history, err)
	}
	// Original PDFs are owner-protected immutable versions; arbitrary files rejected.
	upload := func(data []byte, who context.Context) *httptest.ResponseRecorder {
		var buf bytes.Buffer
		mw := multipart.NewWriter(&buf)
		file, _ := mw.CreateFormFile("file", "signed.pdf")
		file.Write(data)
		mw.Close()
		req := httptest.NewRequest("POST", "/loans/"+loan+"/documents", &buf).WithContext(who)
		req.Header.Set("Content-Type", mw.FormDataContentType())
		r := httptest.NewRecorder()
		router.ServeHTTP(r, req)
		return r
	}
	require(upload([]byte("not PDF"), ctx), 422)
	pdf := []byte("%PDF-1.4\nSynthetic signing document\n%%EOF")
	rec = upload(pdf, ctx)
	require(rec, 201)
	var doc Document
	json.Unmarshal(rec.Body.Bytes(), &doc)
	require(upload(pdf, otherCtx), 409)
	rec = call(ctx, "GET", "/loans/"+loan+"/documents/"+doc.ID, nil)
	require(rec, 200)
	if !bytes.Equal(rec.Body.Bytes(), pdf) {
		t.Fatal("document bytes changed")
	}
	require(call(otherCtx, "GET", "/loans/"+loan+"/documents/"+doc.ID, nil), 404)
	require(call(ctx, "PATCH", "/loans/"+loan, map[string]any{"lender": "Synthetic bank", "agreementNumber": "TEST", "agreementOn": nil, "principalMinor": nil, "annualRateBasisPoints": nil, "reminders": true, "revision": 0}), 200)
	// Only this owner has a verified address, and all mail is captured by a fake.
	email := "synthetic-" + owner + "@example.invalid"
	if _, err = pool.Exec(bg, `INSERT INTO auth_identities(issuer,subject,user_id) VALUES($1,$2,$3)`, auth.EmailIssuer, email, owner); err != nil {
		t.Fatal(err)
	}
	sender := &fakeMail{}
	worker := NewReminderWorker(pool, sender, auth.NewRepository(pool), 1500, "https://grace.invalid")
	now := time.Date(2026, 10, 29, 5, 0, 0, 0, time.UTC)
	if err = worker.Tick(bg, now); err != nil {
		t.Fatal(err)
	}
	if sender.calls.Load() != 1 {
		t.Fatalf("mail calls %d", sender.calls.Load())
	}
	if err = worker.Tick(bg, now); err != nil {
		t.Fatal(err)
	}
	if sender.calls.Load() != 1 {
		t.Fatal("duplicate mail on restart")
	}
	// Lost SMTP acknowledgement stays uncertain and is never automatically resent.
	var reminderID, status string
	err = tenant.QueryRow(ctx, `INSERT INTO payment_reminders(source_kind,source_id,due_on,lead_days,status,dispatched_at) VALUES('loan',$1,'2026-11-02',3,'dispatching',now()-interval '10 minutes') RETURNING id::text`, result.Payments[1].ID).Scan(&reminderID)
	if err != nil {
		t.Fatal(err)
	}
	if err = worker.Tick(bg, now.AddDate(0, 0, 1)); err != nil {
		t.Fatal(err)
	}
	if err = tenant.QueryRow(ctx, `SELECT status FROM payment_reminders WHERE id=$1`, reminderID).Scan(&status); err != nil || status != "uncertain" {
		t.Fatalf("uncertain %s %v", status, err)
	}
	if sender.calls.Load() != 1 {
		t.Fatal("uncertain mail retried")
	}
	// Settlement after copying invalidates the snapshot. A new copy must retain the paid row.
	rec = call(ctx, "POST", "/loans/"+loan+"/schedule-drafts", nil)
	require(rec, 201)
	json.Unmarshal(rec.Body.Bytes(), &d)
	event, _, err := financialevents.NewRepository(pool).Create(ctx, financialevents.Command{Kind: "debt_payment", AccountID: account, AmountMinor: 250, OccurredAt: time.Now(), IdempotencyKey: owner + "-settle", RequestHash: strings.Repeat("b", 64)})
	if err != nil {
		t.Fatal(err)
	}
	_, _, err = obligations.NewRepository(pool).CreateSettlement(ctx, obligations.SettlementCommand{ObligationID: loan, PaymentID: result.Payments[0].ID, FinancialEventID: event.ID, IdempotencyKey: owner + "-settle", RequestHash: strings.Repeat("c", 64)})
	if err != nil {
		t.Fatal(err)
	}
	require(call(ctx, "POST", "/loans/"+loan+"/schedule-drafts/"+d.ID+"/apply", map[string]any{"revision": d.Revision, "confirm": true}), 409)
	rec = call(ctx, "POST", "/loans/"+loan+"/schedule-drafts", nil)
	require(rec, 201)
	json.Unmarshal(rec.Body.Bytes(), &d)
	if d.Payments[0].PaidMinor != 250 {
		t.Fatal("missing settlement")
	}
	d.Payments[1].AmountMinor = 1750
	rec = call(ctx, "PATCH", "/loans/"+loan+"/schedule-drafts/"+d.ID, map[string]any{"revision": d.Revision, "payments": d.Payments})
	require(rec, 200)
	json.Unmarshal(rec.Body.Bytes(), &d)
	require(call(ctx, "POST", "/loans/"+loan+"/schedule-drafts/"+d.ID+"/apply", map[string]any{"revision": d.Revision, "confirm": true}), 200)
	final, err := obligations.NewRepository(pool).Get(ctx, loan, today())
	if err != nil || final.Payments[0].ID != result.Payments[0].ID || final.PaidMinor != 250 {
		t.Fatalf("settlement lost %+v %v", final, err)
	}
	// Competing workers may claim the same due window, but only one sends it.
	concurrent := NewReminderWorker(pool, sender, noLimitQuota{}, 1500, "https://grace.invalid")
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if tickErr := concurrent.Tick(bg, now.AddDate(0, 0, 1)); tickErr != nil {
				t.Errorf("concurrent tick: %v", tickErr)
			}
		}()
	}
	wg.Wait()
	// One subscription and the newly applied unpaid loan row are now due.
	if sender.calls.Load() != 3 {
		t.Fatalf("concurrent mail calls %d", sender.calls.Load())
	}
	var sent int
	if err = tenant.QueryRow(ctx, `SELECT count(*) FROM payment_reminders WHERE status='sent' AND due_on-lead_days='2026-10-30'::date`).Scan(&sent); err != nil || sent != 2 {
		t.Fatalf("window sent %d %v", sent, err)
	}
}

type noLimitQuota struct{}

func (noLimitQuota) ReserveEmail(context.Context, string, string, int) error { return nil }
