package installments

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5"
	"time"
)

// Requirement is the remaining bank requirement at a checkpoint, never an
// invented amortization or an extra amount to add to the card's total debt.
type Requirement struct {
	Source           string  `json:"source"`
	Status           string  `json:"status"`
	SnapshotID       *string `json:"snapshotId"`
	TakenOn          *string `json:"takenOn"`
	PeriodFrom       *string `json:"periodFrom"`
	GraceAmountMinor *int64  `json:"graceAmountMinor"`
	GraceDueOn       *string `json:"graceDueOn"`
	MinimumMinor     *int64  `json:"minimumMinor"`
	MinimumDueOn     *string `json:"minimumDueOn"`
	MinimumInGrace   bool    `json:"minimumInGrace"`
	Reason           string  `json:"reason"`
}

func CardRequirement(ctx context.Context, tx pgx.Tx, accountID, on string) (*Requirement, error) {
	var managed bool
	e := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM installment_plans i WHERE account_id::text=$1 AND mode='credit_card' AND observed_on<=$2::date AND EXISTS(SELECT 1 FROM scheduled_payments sp WHERE sp.obligation_id=i.obligation_id AND sp.plan_active AND sp.amount_minor>COALESCE((SELECT sum(x.amount_minor) FROM installment_payment_allocations x JOIN financial_events e ON e.id=x.event_id WHERE x.payment_id=sp.id AND x.active AND COALESCE(e.occurred_date,(e.occurred_at AT TIME ZONE 'Asia/Yekaterinburg')::date)<=$2::date),0)))`, accountID, on).Scan(&managed)
	if e != nil || !managed {
		return nil, e
	}
	out := &Requirement{Source: "unknown", Status: "unknown", Reason: "Введите общее требование банка, включающее рассрочки."}
	var id, taken, from string
	var grace *int64
	var graceDue, minDue *string
	var minimum *int64
	var included bool
	var confirmed *time.Time
	e = tx.QueryRow(ctx, `SELECT id::text,taken_on::text,COALESCE(period_from::text,''),grace_amount_minor,grace_due_on::text,minimum_minor,minimum_due_on::text,minimum_in_grace,confirmed_through
  FROM credit_card_snapshots WHERE account_id::text=$1 AND taken_on<=$2::date AND installments_included ORDER BY taken_on DESC,created_at DESC,id DESC LIMIT 1`, accountID, on).
		Scan(&id, &taken, &from, &grace, &graceDue, &minimum, &minDue, &included, &confirmed)
	if errors.Is(e, pgx.ErrNoRows) {
		return out, nil
	}
	if e != nil {
		return nil, e
	}
	out.SnapshotID = &id
	out.TakenOn = &taken
	out.PeriodFrom = &from
	var changed bool
	e = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM financial_events e JOIN postings p ON p.financial_event_id=e.id
  WHERE p.account_id::text=$1 AND COALESCE(e.occurred_date,(e.occurred_at AT TIME ZONE 'Asia/Yekaterinburg')::date)<=$2::date
  AND (COALESCE(e.occurred_date,(e.occurred_at AT TIME ZONE 'Asia/Yekaterinburg')::date)>$3::date OR e.created_at>$4 OR e.voided_at>$4))
  OR EXISTS(SELECT 1 FROM installment_plans WHERE account_id::text=$1 AND updated_at>$4 AND observed_on<=$2::date)
  OR EXISTS(SELECT 1 FROM credit_card_terms WHERE account_id::text=$1 AND updated_at>$4 AND effective_from<=$2::date)`, accountID, on, taken, confirmed).Scan(&changed)
	if e != nil {
		return nil, e
	}
	if changed || grace == nil || graceDue == nil || *graceDue < on || (minDue != nil && *minDue < on) {
		out.Status = "needs_confirmation"
		out.Reason = "Требование банка устарело. Обновите данные после операций или смены периода."
		return out, nil
	}
	out.Source = "bank_confirmed"
	out.Status = "current"
	out.Reason = "Общая сумма банка уже включает взносы рассрочек."
	out.GraceAmountMinor = grace
	out.GraceDueOn = graceDue
	out.MinimumMinor = minimum
	out.MinimumDueOn = minDue
	out.MinimumInGrace = included
	return out, nil
}

// CanonicalRequirement collapses only explicitly nested, same-day requirements.
// The caller must not sum partially known or overlapping requirements.
func CanonicalRequirement(r *Requirement) (amount int64, due string, complete bool) {
	if r == nil || r.Source != "bank_confirmed" || r.GraceAmountMinor == nil || r.GraceDueOn == nil {
		return 0, "", false
	}
	if r.MinimumMinor == nil && !r.MinimumInGrace {
		return 0, "", false
	}
	amount, due = *r.GraceAmountMinor, *r.GraceDueOn
	if r.MinimumMinor != nil && *r.MinimumMinor > 0 {
		if r.MinimumDueOn == nil || *r.MinimumDueOn != due || !r.MinimumInGrace {
			return 0, "", false
		}
		amount = max(amount, *r.MinimumMinor)
	}
	return amount, due, true
}
