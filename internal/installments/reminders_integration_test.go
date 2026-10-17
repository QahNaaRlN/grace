//go:build integration

package installments_test

import (
	"context"
	"github.com/QahNaaRlN/grace/internal/accounts"
	"github.com/QahNaaRlN/grace/internal/auth"
	"github.com/QahNaaRlN/grace/internal/commitments"
	"github.com/QahNaaRlN/grace/internal/financialevents"
	"github.com/QahNaaRlN/grace/internal/installments"
	"github.com/QahNaaRlN/grace/internal/platform/access"
	"github.com/QahNaaRlN/grace/internal/platform/database"
	"testing"
	"time"
)

type reminderMail struct{ calls int }

func (m *reminderMail) Ready() bool { return true }
func (m *reminderMail) SendPaymentReminder(context.Context, string, string, string) error {
	m.calls++
	return nil
}

type reminderQuota struct{}

func (reminderQuota) ReserveEmail(context.Context, string, string, int) error { return nil }
func reminderIdentity(t *testing.T, f *fixture) {
	t.Helper()
	owner, _ := access.Current(f.ctx)
	if _, e := f.pool.Exec(context.Background(), `INSERT INTO auth_identities(issuer,subject,user_id) VALUES($1,$2,$3)`, auth.EmailIssuer, "synthetic-"+owner.ID+"@example.invalid", owner.ID); e != nil {
		t.Fatal(e)
	}
}
func reminderDates() (time.Time, string) {
	on, _ := time.Parse("2006-01-02", installments.Today())
	return on.Add(5 * time.Hour), on.AddDate(0, 0, 3).Format("2006-01-02")
}

func TestPaidInstallmentCancelsPendingReminder(t *testing.T) {
	f := setup(t)
	reminderIdentity(t, f)
	now, due := reminderDates()
	c := f.request()
	c.Reminders = true
	c.Payments = []installments.Row{{DueOn: due, AmountMinor: 1200000}}
	var p installments.Plan
	if f.call(f.ctx, "POST", "/installment-plans", "reminder-plan", c, &p) != 200 {
		t.Fatal("plan")
	}
	tenants := database.ForUsers(f.pool)
	if e := tenants.QueryRow(f.ctx, `INSERT INTO payment_reminders(source_kind,source_id,due_on,lead_days) VALUES('installment',$1::uuid,$2::date,3) RETURNING id::text`, p.Payments[0].ID, due).Scan(new(string)); e != nil {
		t.Fatal(e)
	}
	event := f.event(financialevents.Command{Kind: "internal_transfer", AccountID: f.debit, DestinationAccountID: p.AccountID, AmountMinor: 1200000, OccurredDate: installments.Today()})
	f.call(f.ctx, "PUT", "/financial-events/"+event.ID+"/installment-allocation", "paid", installments.AllocationRequest{Reason: "Bank confirmed", Items: []installments.Allocation{{PaymentID: p.Payments[0].ID, AmountMinor: 1200000}}}, nil)
	sender := &reminderMail{}
	worker := commitments.NewReminderWorker(f.pool, sender, reminderQuota{}, 1500, "https://grace.invalid")
	if e := worker.Tick(context.Background(), now); e != nil {
		t.Fatal(e)
	}
	var status string
	if e := tenants.QueryRow(f.ctx, `SELECT status FROM payment_reminders WHERE source_id::text=$1`, p.Payments[0].ID).Scan(&status); e != nil || status != "cancelled" {
		t.Fatalf("pending %s %v", status, e)
	}
	if sender.calls != 0 {
		t.Fatal("sent a paid installment reminder")
	}
}

func TestCardPlansShareOneReminderAndStaleSnapshotCancels(t *testing.T) {
	f := setup(t)
	reminderIdentity(t, f)
	now, due := reminderDates()
	zero := int64(0)
	account, e := accounts.NewRepository(f.pool).Create(f.ctx, accounts.CreateRequest{Name: "Synthetic reminder card", Type: "credit", Currency: "RUB", OpeningBalanceMinor: &zero})
	if e != nil {
		t.Fatal(e)
	}
	c := f.request()
	c.Mode = "credit_card"
	c.AccountID = account.ID
	c.Payments = []installments.Row{{DueOn: due, AmountMinor: 1200000}}
	c.Reminders = true
	var plan installments.Plan
	f.call(f.ctx, "POST", "/installment-plans", "card-plan-a", c, &plan)
	f.call(f.ctx, "POST", "/installment-plans", "card-plan-b", c, nil)
	base := "/accounts/" + account.ID + "/credit-card"
	snapshot := map[string]any{"takenOn": installments.Today(), "debtMinor": 2400000, "graceAmountMinor": 500000, "graceDueOn": due, "minimumMinor": 200000, "minimumDueOn": due, "note": "synthetic", "installmentsIncluded": true, "minimumInGrace": true, "confirmAllOperations": true, "periodFrom": "2026-09-01"}
	if f.call(f.ctx, "POST", base+"/snapshots", "bank", snapshot, nil) != 201 {
		t.Fatal("snapshot")
	}
	f.call(f.ctx, "PUT", base+"/settings", "", map[string]bool{"reminders": true}, nil)
	tenants := database.ForUsers(f.pool)
	// A pending reminder becomes invalid when another operation changes the bank checkpoint.
	if e = tenants.QueryRow(f.ctx, `INSERT INTO payment_reminders(source_kind,source_id,due_on,lead_days) VALUES('credit_card',$1::uuid,$2::date,3) RETURNING id::text`, account.ID, due).Scan(new(string)); e != nil {
		t.Fatal(e)
	}
	sender := &reminderMail{}
	worker := commitments.NewReminderWorker(f.pool, sender, reminderQuota{}, 1500, "https://grace.invalid")
	if e = worker.Tick(context.Background(), now); e != nil {
		t.Fatal(e)
	}
	if sender.calls != 1 {
		t.Fatalf("expected one shared card reminder, got %d", sender.calls)
	}
	nextOn := now.AddDate(0, 0, 2)
	if e = tenants.QueryRow(f.ctx, `INSERT INTO payment_reminders(source_kind,source_id,due_on,lead_days) VALUES('credit_card',$1::uuid,$2::date,1) RETURNING id::text`, account.ID, due).Scan(new(string)); e != nil {
		t.Fatal(e)
	}
	f.event(financialevents.Command{Kind: "internal_transfer", AccountID: f.debit, DestinationAccountID: account.ID, AmountMinor: 100000, OccurredDate: nextOn.Format("2006-01-02")})
	now = nextOn
	if e = worker.Tick(context.Background(), now); e != nil {
		t.Fatal(e)
	}
	var total, planReminders int
	if e = tenants.QueryRow(f.ctx, `SELECT count(*),count(*) FILTER(WHERE source_kind='installment') FROM payment_reminders`).Scan(&total, &planReminders); e != nil || total != 2 || planReminders != 0 {
		t.Fatalf("duplicate reminders %d %d %v", total, planReminders, e)
	}
	var status string
	tenants.QueryRow(f.ctx, `SELECT status FROM payment_reminders WHERE source_id::text=$1 AND lead_days=1`, account.ID).Scan(&status)
	if sender.calls != 1 || status != "cancelled" {
		t.Fatalf("stale card reminder %d %s", sender.calls, status)
	}
}
