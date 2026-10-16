package commitments

import (
	"context"
	"github.com/QahNaaRlN/grace/internal/auth"
	"github.com/QahNaaRlN/grace/internal/creditcards"
	"github.com/QahNaaRlN/grace/internal/platform/access"
	"github.com/QahNaaRlN/grace/internal/platform/database"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"log"
	"time"
)

type ReminderSender interface {
	Ready() bool
	SendPaymentReminder(context.Context, string, string, string) error
}
type EmailQuota interface {
	ReserveEmail(context.Context, string, string, int) error
}
type ReminderWorker struct {
	pool      *pgxpool.Pool
	tenants   *database.TenantPool
	sender    ReminderSender
	quota     EmailQuota
	budget    int
	publicURL string
}

func NewReminderWorker(pool *pgxpool.Pool, sender ReminderSender, quota EmailQuota, budget int, url string) *ReminderWorker {
	return &ReminderWorker{pool, database.ForUsers(pool), sender, quota, budget, url}
}
func (w *ReminderWorker) Run(ctx context.Context) {
	if !w.sender.Ready() || w.budget < 1 {
		return
	}
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		if err := w.Tick(ctx, time.Now()); err != nil && ctx.Err() == nil {
			log.Print("Grace payment reminder processing unavailable")
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

// Only the privileged identity lookup runs outside the tenant role. Each financial
// read and outbox transition is scoped to the verified, enabled owner.
func (w *ReminderWorker) Tick(ctx context.Context, now time.Time) error {
	local := now.UTC().Add(5 * time.Hour)
	if local.Hour() < 9 {
		return nil
	}
	rows, err := w.pool.Query(ctx, `SELECT u.id::text,i.subject FROM users u JOIN auth_identities i ON i.user_id=u.id AND i.issuer=$1 WHERE NOT u.disabled`, auth.EmailIssuer)
	if err != nil {
		return err
	}
	type recipient struct{ id, email string }
	owners := []recipient{}
	for rows.Next() {
		var o recipient
		if err = rows.Scan(&o.id, &o.email); err != nil {
			rows.Close()
			return err
		}
		owners = append(owners, o)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	for _, o := range owners {
		tenantCtx := access.WithPrincipal(ctx, access.Principal{ID: o.id})
		if err = w.process(tenantCtx, o.email, local.Format("2006-01-02")); err != nil {
			return err
		}
	}
	return nil
}
func (w *ReminderWorker) process(ctx context.Context, email, today string) error {
	tx, err := w.tenants.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	// A committed dispatch is never retried after a crash or ambiguous SMTP result.
	_, err = tx.Exec(ctx, `UPDATE payment_reminders SET status='uncertain' WHERE status='dispatching' AND dispatched_at<now()-interval '5 minutes'`)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `UPDATE payment_reminders SET status='cancelled' WHERE status='pending' AND due_on-lead_days<$1::date`, today)
	if err != nil {
		return err
	}
	rows, err := tx.Query(ctx, `SELECT id::text,starts_on::text,period,COALESCE(ends_on::text,'') FROM subscriptions WHERE reminders AND NOT paused`)
	if err != nil {
		return err
	}
	type candidate struct {
		kind, id, due string
		lead          int
	}
	candidates := []candidate{}
	for rows.Next() {
		var id, start, period, end string
		if err = rows.Scan(&id, &start, &period, &end); err != nil {
			rows.Close()
			return err
		}
		day, _ := time.Parse("2006-01-02", today)
		for _, lead := range []int{3, 1} {
			target := day.AddDate(0, 0, lead).Format("2006-01-02")
			due := NextDate(start, period, end, target, false)
			if due != nil && *due == target {
				candidates = append(candidates, candidate{"subscription", id, target, lead})
			}
		}
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	rows, err = tx.Query(ctx, `SELECT sp.id::text,sp.due_on::text,(sp.due_on-$1::date)::int FROM scheduled_payments sp JOIN loan_details l ON l.obligation_id=sp.obligation_id WHERE l.reminders AND sp.plan_active AND sp.due_on-$1::date IN(1,3) AND sp.amount_minor>COALESCE((SELECT sum(s.amount_minor) FROM scheduled_payment_settlements s WHERE s.scheduled_payment_id=sp.id),0)`, today)
	if err != nil {
		return err
	}
	for rows.Next() {
		c := candidate{kind: "loan"}
		if err = rows.Scan(&c.id, &c.due, &c.lead); err != nil {
			rows.Close()
			return err
		}
		candidates = append(candidates, c)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	cards, err := creditCardCandidates(ctx, tx, today)
	if err != nil {
		return err
	}
	for _, c := range cards {
		candidates = append(candidates, candidate{"credit_card", c.id, c.due, c.lead})
	}
	for _, c := range candidates {
		if _, err = tx.Exec(ctx, `INSERT INTO payment_reminders(source_kind,source_id,due_on,lead_days) VALUES($1,$2,$3,$4) ON CONFLICT DO NOTHING`, c.kind, c.id, c.due, c.lead); err != nil {
			return err
		}
	}
	var id, kind, source, due string
	var lead int
	err = tx.QueryRow(ctx, `SELECT id::text,source_kind,source_id::text,due_on::text,lead_days FROM payment_reminders WHERE status='pending' AND due_on-lead_days=$1::date ORDER BY due_on,id FOR UPDATE SKIP LOCKED LIMIT 1`, today).Scan(&id, &kind, &source, &due, &lead)
	if err == pgx.ErrNoRows {
		return tx.Commit(ctx)
	}
	if err != nil {
		return err
	}
	active := false
	for _, c := range candidates {
		if c.kind == kind && c.id == source && c.due == due && c.lead == lead {
			active = true
			break
		}
	}
	if !active {
		_, err = tx.Exec(ctx, `UPDATE payment_reminders SET status='cancelled' WHERE id::text=$1`, id)
		if err != nil {
			return err
		}
		return tx.Commit(ctx)
	}
	// Keep part of the same persisted monthly allowance available for sign-in,
	// verification and password recovery even if reminders exhaust their share.
	reservedForAuth := w.budget / 3
	if reservedForAuth == 0 {
		reservedForAuth = 1
	}
	if reservedForAuth > 500 {
		reservedForAuth = 500
	}
	if err = w.quota.ReserveEmail(ctx, email, "payment-reminder", w.budget-reservedForAuth); err != nil {
		if err == auth.ErrRateLimited {
			return tx.Commit(ctx)
		}
		return err
	}
	if _, err = tx.Exec(ctx, `UPDATE payment_reminders SET status='dispatching',dispatched_at=now() WHERE id::text=$1`, id); err != nil {
		return err
	}
	if err = tx.Commit(ctx); err != nil {
		return err
	}
	checkTx, err := w.tenants.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer checkTx.Rollback(context.WithoutCancel(ctx))
	active, err = reminderStillActive(ctx, checkTx, kind, source, due)
	if err != nil {
		return err
	}
	// Revalidate the enabled identity immediately before handing the message to SMTP.
	principal, _ := access.Current(ctx)
	var verified bool
	err = w.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM users u JOIN auth_identities i ON i.user_id=u.id WHERE u.id=$1 AND NOT u.disabled AND i.issuer=$2 AND i.subject=$3)`, principal.ID, auth.EmailIssuer, email).Scan(&verified)
	status := "cancelled"
	if err == nil && verified && active {
		status = "sent"
		link := w.publicURL + "/payments"
		if kind == "subscription" {
			link = w.publicURL + "/subscriptions"
		}
		if kind == "credit_card" {
			link = w.publicURL + "/accounts"
		}
		if w.sender.SendPaymentReminder(ctx, email, due, link) != nil {
			status = "uncertain"
		}
	}
	if err != nil {
		status = "uncertain"
	}
	finishCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancel()
	var finished string
	finishErr := w.tenants.QueryRow(finishCtx, `UPDATE payment_reminders SET status=$2,sent_at=CASE WHEN $2='sent' THEN now() END WHERE id::text=$1 AND status='dispatching' RETURNING id::text`, id, status).Scan(&finished)
	return finishErr
}

type cardReminder struct {
	id, due string
	lead    int
}

// creditCardCandidates finds cards with reminders on whose grace or minimum
// payment is due in 3 or 1 days and still unpaid (SPEC-0051).
func creditCardCandidates(ctx context.Context, tx pgx.Tx, today string) ([]cardReminder, error) {
	rows, err := tx.Query(ctx, `SELECT s.account_id::text FROM credit_card_settings s JOIN accounts a ON a.id=s.account_id WHERE s.reminders AND a.type='credit'`)
	if err != nil {
		return nil, err
	}
	ids := []string{}
	for rows.Next() {
		var id string
		if err = rows.Scan(&id); err != nil {
			rows.Close()
			return nil, err
		}
		ids = append(ids, id)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, err
	}
	day, err := creditcards.ParseDay(today)
	if err != nil {
		return nil, err
	}
	out := []cardReminder{}
	for _, id := range ids {
		dates, err := creditcards.DueDates(ctx, tx, id, day)
		if err != nil {
			return nil, err
		}
		for _, due := range dates {
			for _, lead := range []int{3, 1} {
				if creditcards.FormatDay(day.AddDate(0, 0, lead)) == due {
					out = append(out, cardReminder{id, due, lead})
				}
			}
		}
	}
	return out, nil
}

func reminderStillActive(ctx context.Context, tx pgx.Tx, kind, source, due string) (bool, error) {
	if kind == "credit_card" {
		var reminders bool
		err := tx.QueryRow(ctx, `SELECT reminders FROM credit_card_settings WHERE account_id::text=$1 FOR UPDATE`, source).Scan(&reminders)
		if err == pgx.ErrNoRows || (err == nil && !reminders) {
			return false, nil
		}
		if err != nil {
			return false, err
		}
		dates, err := creditcards.DueDates(ctx, tx, source, creditcards.Today())
		if err != nil {
			return false, err
		}
		for _, d := range dates {
			if d == due {
				return true, nil
			}
		}
		return false, nil
	}
	if kind == "subscription" {
		var start, period, end string
		err := tx.QueryRow(ctx, `SELECT starts_on::text,period,COALESCE(ends_on::text,'') FROM subscriptions WHERE id::text=$1 AND reminders AND NOT paused FOR UPDATE`, source).Scan(&start, &period, &end)
		if err == pgx.ErrNoRows {
			return false, nil
		}
		if err != nil {
			return false, err
		}
		next := NextDate(start, period, end, due, false)
		return next != nil && *next == due, nil
	}
	var amount, paid int64
	err := tx.QueryRow(ctx, `SELECT sp.amount_minor,COALESCE((SELECT sum(s.amount_minor) FROM scheduled_payment_settlements s WHERE s.scheduled_payment_id=sp.id),0) FROM scheduled_payments sp JOIN obligations o ON o.id=sp.obligation_id JOIN loan_details l ON l.obligation_id=o.id WHERE sp.id::text=$1 AND sp.due_on=$2::date AND sp.plan_active AND l.reminders FOR UPDATE OF o,sp,l`, source, due).Scan(&amount, &paid)
	if err == pgx.ErrNoRows {
		return false, nil
	}
	if err == nil {
		err = tx.QueryRow(ctx, `SELECT COALESCE(sum(amount_minor),0) FROM scheduled_payment_settlements WHERE scheduled_payment_id::text=$1`, source).Scan(&paid)
	}
	return amount > paid, err
}
