package installments

import (
	"context"
	"fmt"
	"github.com/QahNaaRlN/grace/internal/financialevents"
	"github.com/jackc/pgx/v5"
)

func readPlan(ctx context.Context, tx pgx.Tx, id, on string) (Plan, error) {
	p := Plan{Payments: []Payment{}}
	err := tx.QueryRow(ctx, `SELECT i.id::text,i.obligation_id::text,o.name,i.mode,i.provider,i.contract_number,o.currency,i.account_id::text,
  i.purchase_event_id::text,i.opened_on::text,i.observed_on::text,i.principal_minor,i.initial_remaining_minor,i.historical,i.reminders,i.revision,
  CASE WHEN a.balance_reconciled THEN -(a.opening_balance_minor+COALESCE((SELECT sum(CASE po.direction WHEN 'credit' THEN po.amount_minor ELSE -po.amount_minor END)
   FROM postings po JOIN financial_events e ON e.id=po.financial_event_id WHERE po.account_id=a.id AND COALESCE(e.occurred_date,(e.occurred_at AT TIME ZONE 'Asia/Yekaterinburg')::date)<=$2::date),0)) ELSE NULL END
  FROM installment_plans i JOIN obligations o ON o.id=i.obligation_id JOIN accounts a ON a.id=i.account_id WHERE i.id::text=$1 AND i.observed_on<=$2::date`, id, on).
		Scan(&p.ID, &p.ObligationID, &p.Name, &p.Mode, &p.Provider, &p.ContractNumber, &p.Currency, &p.AccountID, &p.PurchaseEventID, &p.OpenedOn, &p.ObservedOn, &p.PrincipalMinor, &p.InitialRemainingMinor, &p.Historical, &p.Reminders, &p.Revision, &p.AccountDebtMinor)
	if err != nil {
		return p, err
	}
	rows, err := tx.Query(ctx, `SELECT sp.id::text,sp.position,sp.due_on::text,sp.amount_minor,COALESCE(sum(x.amount_minor) FILTER(WHERE x.active AND COALESCE(e.occurred_date,(e.occurred_at AT TIME ZONE 'Asia/Yekaterinburg')::date)<=$2::date),0)::bigint
  FROM scheduled_payments sp LEFT JOIN installment_payment_allocations x ON x.payment_id=sp.id LEFT JOIN financial_events e ON e.id=x.event_id
  WHERE sp.obligation_id=$1::uuid AND sp.plan_active GROUP BY sp.id ORDER BY sp.position`, p.ObligationID, on)
	if err != nil {
		return p, err
	}
	defer rows.Close()
	for rows.Next() {
		var v Payment
		if err = rows.Scan(&v.ID, &v.Position, &v.DueOn, &v.AmountMinor, &v.PaidMinor); err != nil {
			return p, err
		}
		v.RemainingMinor = v.AmountMinor - v.PaidMinor
		p.PaidMinor += v.PaidMinor
		p.RemainingMinor += v.RemainingMinor
		p.Payments = append(p.Payments, v)
	}
	if p.Mode == "standalone" && p.AccountDebtMinor != nil {
		d := *p.AccountDebtMinor - p.RemainingMinor
		p.DifferenceMinor = &d
	}
	if err = rows.Err(); err != nil {
		return p, err
	}
	rows.Close()
	p.Drafts = []Draft{}
	draftRows, err := tx.Query(ctx, `SELECT id::text,revision,plan_revision,payments,applied FROM installment_schedule_drafts WHERE plan_id=$1::uuid AND NOT applied ORDER BY created_at DESC,id`, p.ID)
	if err != nil {
		return p, err
	}
	defer draftRows.Close()
	for draftRows.Next() {
		var d Draft
		if err = draftRows.Scan(&d.ID, &d.Revision, &d.PlanRevision, &d.Payments, &d.Applied); err != nil {
			return p, err
		}
		p.Drafts = append(p.Drafts, d)
	}
	return p, draftRows.Err()
}

func createPlan(ctx context.Context, tx pgx.Tx, c CreateRequest, hash, key string) (Plan, error) {
	var err error
	if c.AccountID == "" {
		opening := int64(0)
		if c.Historical {
			opening = -c.InitialRemainingMinor
		}
		err = tx.QueryRow(ctx, `INSERT INTO accounts(name,type,currency,opening_balance_minor,balance_reconciled) VALUES($1,'installment',$2,$3,true) RETURNING id::text`, c.Name, c.Currency, opening).Scan(&c.AccountID)
		if err != nil {
			return Plan{}, err
		}
	}
	var currency, kind string
	var opening *int64
	var reconciled bool
	err = tx.QueryRow(ctx, `SELECT currency,type::text,opening_balance_minor,balance_reconciled FROM accounts WHERE id::text=$1 FOR UPDATE`, c.AccountID).Scan(&currency, &kind, &opening, &reconciled)
	if err != nil {
		return Plan{}, err
	}
	if currency != c.Currency || (c.Mode == "standalone" && kind != "installment") || (c.Mode == "credit_card" && kind != "credit") {
		return Plan{}, ErrInvalid
	}
	if c.Mode == "standalone" {
		expected := int64(0)
		if c.Historical {
			expected = -c.InitialRemainingMinor
		}
		if !reconciled || opening == nil || *opening != expected {
			return Plan{}, ErrInvalid
		}
	}
	if c.Purchase != nil {
		cat := c.Purchase.CategoryID
		event, _, e := financialevents.CreateInTx(ctx, tx, financialevents.Command{Kind: "purchase", AccountID: c.AccountID, CategoryID: &cat, AmountMinor: c.PrincipalMinor, OccurredDate: c.OpenedOn, Merchant: c.Purchase.Merchant, Description: c.Purchase.Description, IdempotencyKey: "ip-purchase-" + hashOf(key), RequestHash: hash}, "manual")
		if e != nil {
			return Plan{}, e
		}
		c.PurchaseEventID = event.ID
	}
	var obligationID string
	err = tx.QueryRow(ctx, `INSERT INTO obligations(name,kind,currency,total_minor,idempotency_key,request_hash) VALUES($1,'installment',$2,$3,$4,$5) RETURNING id::text`, c.Name, c.Currency, c.InitialRemainingMinor, "ip-obligation-"+hashOf(key), hash).Scan(&obligationID)
	if err != nil {
		return Plan{}, err
	}
	var planID string
	err = tx.QueryRow(ctx, `INSERT INTO installment_plans(obligation_id,account_id,purchase_event_id,mode,provider,contract_number,opened_on,observed_on,principal_minor,initial_remaining_minor,historical,reminders)
  VALUES($1::uuid,$2::uuid,NULLIF($3,'')::uuid,$4,$5,$6,$7::date,$8::date,$9,$10,$11,$12) RETURNING id::text`, obligationID, c.AccountID, c.PurchaseEventID, c.Mode, c.Provider, c.ContractNumber, c.OpenedOn, c.ObservedOn, c.PrincipalMinor, c.InitialRemainingMinor, c.Historical, c.Reminders).Scan(&planID)
	if err != nil {
		return Plan{}, err
	}
	rowIDs := []string{}
	for n, p := range c.Payments {
		var id string
		err = tx.QueryRow(ctx, `INSERT INTO scheduled_payments(obligation_id,position,due_on,amount_minor) VALUES($1::uuid,$2,$3::date,$4) RETURNING id::text`, obligationID, n+1, p.DueOn, p.AmountMinor).Scan(&id)
		if err != nil {
			return Plan{}, err
		}
		rowIDs = append(rowIDs, id)
	}
	if first := c.FirstPayment; first != nil {
		event, _, e := financialevents.CreateInTx(ctx, tx, financialevents.Command{Kind: "internal_transfer", AccountID: first.SourceAccountID, DestinationAccountID: c.AccountID, AmountMinor: first.AmountMinor, OccurredDate: first.On, Description: "Взнос по рассрочке: " + c.Name, IdempotencyKey: "ip-first-" + hashOf(key), RequestHash: hash}, "manual")
		if e != nil {
			return Plan{}, e
		}
		left := first.AmountMinor
		items := []Allocation{}
		for n, row := range c.Payments {
			amount := min(left, row.AmountMinor)
			if amount > 0 {
				items = append(items, Allocation{rowIDs[n], amount})
				left -= amount
			}
		}
		if _, err = saveAllocation(ctx, tx, event.ID, AllocationRequest{Items: items, Reason: "Первый взнос"}); err != nil {
			return Plan{}, err
		}
	}
	return readPlan(ctx, tx, planID, Today())
}

func readAllocation(ctx context.Context, tx pgx.Tx, eventID string) (AllocationView, error) {
	a := AllocationView{EventID: eventID, Items: []Allocation{}, History: []AllocationHistory{}}
	err := tx.QueryRow(ctx, `SELECT p.account_id::text,p.currency,p.amount_minor,COALESCE(b.revision,0) FROM financial_events e JOIN postings p ON p.financial_event_id=e.id AND p.direction='credit'
  LEFT JOIN installment_allocation_batches b ON b.event_id=e.id WHERE e.id::text=$1 AND e.kind='internal_transfer' AND e.voided_at IS NULL`, eventID).Scan(&a.AccountID, &a.Currency, &a.CreditMinor, &a.Revision)
	if err != nil {
		return a, err
	}
	a.UnallocatedMinor = a.CreditMinor
	rows, err := tx.Query(ctx, `SELECT payment_id::text,amount_minor FROM installment_payment_allocations WHERE event_id::text=$1 AND active ORDER BY payment_id`, eventID)
	if err != nil {
		return a, err
	}
	for rows.Next() {
		var v Allocation
		if err = rows.Scan(&v.PaymentID, &v.AmountMinor); err != nil {
			rows.Close()
			return a, err
		}
		a.Items = append(a.Items, v)
		a.UnallocatedMinor -= v.AmountMinor
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return a, err
	}
	rows, err = tx.Query(ctx, `SELECT previous,current,reason,changed_at FROM installment_allocation_revisions WHERE event_id::text=$1 ORDER BY changed_at,id`, eventID)
	if err != nil {
		return a, err
	}
	defer rows.Close()
	for rows.Next() {
		var v AllocationHistory
		if err = rows.Scan(&v.Previous, &v.Current, &v.Reason, &v.ChangedAt); err != nil {
			return a, err
		}
		a.History = append(a.History, v)
	}
	return a, rows.Err()
}
func saveAllocation(ctx context.Context, tx pgx.Tx, eventID string, c AllocationRequest) (AllocationView, error) {
	a, err := readAllocation(ctx, tx, eventID)
	if err != nil {
		return a, err
	}
	if c.Revision != a.Revision || len([]rune(c.Reason)) < 1 || len([]rune(c.Reason)) > 500 {
		return a, ErrConflict
	}
	if len(c.Items) > 600 {
		return a, ErrInvalid
	}
	sum := int64(0)
	seen := map[string]bool{}
	for _, v := range c.Items {
		if v.PaymentID == "" || seen[v.PaymentID] || v.AmountMinor <= 0 || v.AmountMinor > a.CreditMinor-sum {
			return a, ErrInvalid
		}
		seen[v.PaymentID] = true
		sum += v.AmountMinor
	}
	_, err = tx.Exec(ctx, `INSERT INTO installment_allocation_batches(event_id,revision,reason) VALUES($1::uuid,1,$2)
  ON CONFLICT(event_id) DO UPDATE SET revision=installment_allocation_batches.revision+1,reason=EXCLUDED.reason`, eventID, c.Reason)
	if err != nil {
		return a, err
	}
	if _, err = tx.Exec(ctx, `UPDATE installment_payment_allocations SET active=false WHERE event_id::text=$1 AND active`, eventID); err != nil {
		return a, err
	}
	for _, v := range c.Items {
		_, err = tx.Exec(ctx, `INSERT INTO installment_payment_allocations(event_id,payment_id,amount_minor) VALUES($1::uuid,$2::uuid,$3)
  ON CONFLICT(event_id,payment_id) DO UPDATE SET amount_minor=EXCLUDED.amount_minor,active=true`, eventID, v.PaymentID, v.AmountMinor)
		if err != nil {
			return a, fmt.Errorf("allocate transfer: %w", err)
		}
	}
	return readAllocation(ctx, tx, eventID)
}
