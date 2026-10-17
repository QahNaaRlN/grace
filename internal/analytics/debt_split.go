package analytics

import (
	"context"
	"fmt"
	"time"

	"github.com/QahNaaRlN/grace/internal/loans"
)

// DebtSplit divides debt payments into loan principal, interest and other
// charges (the cost of credit) and money whose split is not known (SPEC-0053).
type DebtSplit struct {
	DebtPrincipalMinor   int64 `json:"debtPrincipalMinor"`
	DebtInterestMinor    int64 `json:"debtInterestMinor"`
	DebtOtherMinor       int64 `json:"debtOtherMinor"`
	DebtUnallocatedMinor int64 `json:"debtUnallocatedMinor"`
}

// debtSplits returns the split of debt payments in [from, to) by currency and,
// when granularity is set, by the start day of each bucket ("" otherwise).
func debtSplits(ctx context.Context, db periodQuerier, from, to time.Time, granularity string) (map[string]map[string]DebtSplit, error) {
	bucket := `''`
	args := []any{from, to}
	if granularity != "" {
		bucket = `to_char(date_trunc($3::text, COALESCE(e.occurred_at,e.occurred_date::timestamp AT TIME ZONE 'Asia/Yekaterinburg') AT TIME ZONE 'Asia/Yekaterinburg'),'YYYY-MM-DD')`
		args = append(args, granularity)
	}
	rows, err := db.Query(ctx, `SELECT e.id::text, p.currency, `+bucket+`, p.amount_minor
		FROM financial_events e JOIN postings p ON p.financial_event_id = e.id
		WHERE e.kind = 'debt_payment' AND p.direction = 'debit'
		  AND COALESCE(e.occurred_at,e.occurred_date::timestamp AT TIME ZONE 'Asia/Yekaterinburg') >= $1
		  AND COALESCE(e.occurred_at,e.occurred_date::timestamp AT TIME ZONE 'Asia/Yekaterinburg') < $2`, args...)
	if err != nil {
		return nil, fmt.Errorf("query debt payments: %w", err)
	}
	type payment struct {
		id, currency, bucket string
		amount               int64
	}
	var payments []payment
	ids := []string{}
	for rows.Next() {
		var p payment
		if err := rows.Scan(&p.id, &p.currency, &p.bucket, &p.amount); err != nil {
			rows.Close()
			return nil, fmt.Errorf("scan debt payment: %w", err)
		}
		payments = append(payments, p)
		ids = append(ids, p.id)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, fmt.Errorf("read debt payments: %w", err)
	}
	splits, err := loans.EventSplits(ctx, db, ids)
	if err != nil {
		return nil, fmt.Errorf("split debt payments: %w", err)
	}
	out := map[string]map[string]DebtSplit{}
	for _, p := range payments {
		if out[p.currency] == nil {
			out[p.currency] = map[string]DebtSplit{}
		}
		s := out[p.currency][p.bucket]
		known := splits[p.id]
		s.DebtPrincipalMinor += known.PrincipalMinor
		s.DebtInterestMinor += known.InterestMinor
		s.DebtOtherMinor += known.OtherMinor
		s.DebtUnallocatedMinor += p.amount - known.Total()
		out[p.currency][p.bucket] = s
	}
	return out, nil
}
