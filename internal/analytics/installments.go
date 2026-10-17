package analytics

import (
	"context"
	"github.com/QahNaaRlN/grace/internal/installments"
	"github.com/jackc/pgx/v5"
	"time"
)

func installmentRepaid(ctx context.Context, db periodQuerier, from, to time.Time, granularity string) (map[string]map[string]int64, error) {
	bucket := `''`
	args := []any{from, to}
	if granularity != "" {
		bucket = `to_char(date_trunc($3::text,COALESCE(e.occurred_at,e.occurred_date::timestamp AT TIME ZONE 'Asia/Yekaterinburg') AT TIME ZONE 'Asia/Yekaterinburg'),'YYYY-MM-DD')`
		args = append(args, granularity)
	}
	rows, e := db.Query(ctx, `SELECT p.currency,`+bucket+`,sum(x.amount_minor)::bigint FROM installment_payment_allocations x
  JOIN financial_events e ON e.id=x.event_id JOIN postings p ON p.financial_event_id=e.id AND p.direction='debit'
  WHERE x.active AND e.voided_at IS NULL AND COALESCE(e.occurred_at,e.occurred_date::timestamp AT TIME ZONE 'Asia/Yekaterinburg')>=$1
  AND COALESCE(e.occurred_at,e.occurred_date::timestamp AT TIME ZONE 'Asia/Yekaterinburg')<$2 GROUP BY p.currency,2`, args...)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := map[string]map[string]int64{}
	for rows.Next() {
		var currency, b string
		var amount int64
		if e = rows.Scan(&currency, &b, &amount); e != nil {
			return nil, e
		}
		if out[currency] == nil {
			out[currency] = map[string]int64{}
		}
		out[currency][b] = amount
	}
	return out, rows.Err()
}
func addInstallmentSummary(ctx context.Context, db periodQuerier, from, to time.Time, items map[string]*CurrencyPeriodSummary) error {
	amounts, e := installmentRepaid(ctx, db, from, to, "")
	if e != nil {
		return e
	}
	for currency, v := range amounts {
		if items[currency] != nil {
			items[currency].InstallmentPrincipalRepaidMinor = v[""]
		}
	}
	return nil
}
func addInstallmentBuckets(ctx context.Context, db periodQuerier, from, to time.Time, granularity string, result *Statistics) error {
	amounts, e := installmentRepaid(ctx, db, from, to, granularity)
	if e != nil {
		return e
	}
	for n := range result.Currencies {
		c := &result.Currencies[n]
		for j := range c.Series {
			b := &c.Series[j]
			t, _ := time.Parse("2006-01-02", b.From)
			b.InstallmentPrincipalRepaidMinor = amounts[c.Currency][bucketStart(t, granularity).Format("2006-01-02")]
		}
	}
	return nil
}
func addCardRequirements(ctx context.Context, tx pgx.Tx, on, through string, items map[string]*CurrencyCashPosition) error {
	gaps, e := tx.Query(ctx, `SELECT DISTINCT o.currency FROM installment_plans i JOIN obligations o ON o.id=i.obligation_id WHERE i.historical AND i.opened_on<=$1::date AND i.observed_on>$1::date`, on)
	if e != nil {
		return e
	}
	for gaps.Next() {
		var currency string
		if e = gaps.Scan(&currency); e != nil {
			gaps.Close()
			return e
		}
		item := items[currency]
		if item == nil {
			item = &CurrencyCashPosition{Currency: currency, ScheduledPayments: []CashScheduledPayment{}}
			items[currency] = item
		}
		item.CoverageIncomplete = true
		item.CoverageReasons = append(item.CoverageReasons, "История рассрочки на эту дату не подтверждена.")
	}
	e = gaps.Err()
	gaps.Close()
	if e != nil {
		return e
	}

	rows, e := tx.Query(ctx, `SELECT DISTINCT i.account_id::text,o.currency,o.name FROM installment_plans i JOIN accounts o ON o.id=i.account_id WHERE i.mode='credit_card' AND i.observed_on<=$1::date`, on)
	if e != nil {
		return e
	}
	type card struct{ id, currency, name string }
	cards := []card{}
	for rows.Next() {
		var c card
		if e = rows.Scan(&c.id, &c.currency, &c.name); e != nil {
			rows.Close()
			return e
		}
		cards = append(cards, c)
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return e
	}
	for _, c := range cards {
		item := items[c.currency]
		if item == nil {
			item = &CurrencyCashPosition{Currency: c.currency, ScheduledPayments: []CashScheduledPayment{}}
			items[c.currency] = item
		}
		req, e := installments.CardRequirement(ctx, tx, c.id, on)
		if e != nil {
			return e
		}
		if req == nil {
			continue
		}
		amount, due, complete := installments.CanonicalRequirement(req)
		if !complete || through > due {
			item.CoverageIncomplete = true
			item.CoverageReasons = append(item.CoverageReasons, "Требования карты «"+c.name+"» не подтверждены на весь период.")
		}
		if complete && due <= through && amount > 0 {
			item.ScheduledPayments = append(item.ScheduledPayments, CashScheduledPayment{ScheduledPaymentID: "card:" + c.id, ObligationID: "card:" + c.id, ObligationName: c.name, DueOn: due, AmountMinor: amount, RemainingMinor: amount, Timing: "upcoming"})
			item.UpcomingObligationsMinor += amount
		}
	}
	return nil
}

// Opening balances of historical debt accounts are known only from observation,
// so an earlier aggregate balance must remain unknown rather than invented.
func installmentHistoryCoverage(ctx context.Context, db periodQuerier, from, to time.Time, items map[string]*CurrencyPeriodSummary) error {
	rows, e := db.Query(ctx, `SELECT o.currency,max(i.observed_on::timestamp AT TIME ZONE 'Asia/Yekaterinburg') FROM installment_plans i JOIN obligations o ON o.id=i.obligation_id WHERE i.mode='standalone' AND i.historical GROUP BY o.currency`)
	if e != nil {
		return e
	}
	defer rows.Close()
	for rows.Next() {
		var currency string
		var latest time.Time
		if e = rows.Scan(&currency, &latest); e != nil {
			return e
		}
		if item := items[currency]; item != nil {
			if from.Before(latest) {
				item.OpeningBalanceMinor = nil
			}
			if to.Before(latest) {
				item.ClosingBalanceMinor = nil
			}
		}
	}
	return rows.Err()
}
