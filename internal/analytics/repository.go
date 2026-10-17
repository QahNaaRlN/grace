package analytics

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/QahNaaRlN/grace/internal/obligations"
	"github.com/QahNaaRlN/grace/internal/platform/database"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct{ pool *database.TenantPool }

func NewRepository(pool *pgxpool.Pool) *Repository { return &Repository{pool: database.ForUsers(pool)} }

func (repo *Repository) PeriodSummary(ctx context.Context, from, to time.Time) (PeriodSummary, error) {
	tx, err := repo.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return PeriodSummary{}, fmt.Errorf("begin period summary: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	result, err := periodSummary(ctx, tx, from, to)
	if err != nil {
		return PeriodSummary{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return PeriodSummary{}, fmt.Errorf("commit period summary snapshot: %w", err)
	}
	return result, nil
}

type periodQuerier interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
}

func periodSummary(ctx context.Context, db periodQuerier, from, to time.Time) (PeriodSummary, error) {
	if err := checkPeriodPrecision(ctx, db, from, to); err != nil {
		return PeriodSummary{}, err
	}
	byCurrency := make(map[string]*CurrencyPeriodSummary)
	rows, err := db.Query(ctx, `
		WITH account_opening AS (
			SELECT currency, SUM(opening_balance_minor)::bigint AS opening_minor, BOOL_AND(balance_reconciled) AS reconciled
			FROM accounts GROUP BY currency
		), posting_effects AS (
			SELECT p.currency,
				COALESCE(SUM(CASE WHEN COALESCE(e.occurred_at,e.occurred_date::timestamp AT TIME ZONE 'Asia/Yekaterinburg') < $1 THEN
					CASE p.direction WHEN 'credit' THEN p.amount_minor ELSE -p.amount_minor END
				ELSE 0 END), 0)::bigint AS before_from,
				COALESCE(SUM(CASE WHEN COALESCE(e.occurred_at,e.occurred_date::timestamp AT TIME ZONE 'Asia/Yekaterinburg') < $2 THEN
					CASE p.direction WHEN 'credit' THEN p.amount_minor ELSE -p.amount_minor END
				ELSE 0 END), 0)::bigint AS before_to
			FROM postings p JOIN financial_events e ON e.id = p.financial_event_id
			GROUP BY p.currency
		), currencies AS (
			SELECT currency FROM account_opening UNION SELECT currency FROM posting_effects
		)
		SELECT c.currency,
			CASE WHEN a.reconciled THEN (a.opening_minor + COALESCE(p.before_from, 0))::bigint ELSE NULL END,
			CASE WHEN a.reconciled THEN (a.opening_minor + COALESCE(p.before_to, 0))::bigint ELSE NULL END
		FROM currencies c
		LEFT JOIN account_opening a USING (currency)
		LEFT JOIN posting_effects p USING (currency)
		ORDER BY c.currency`, from, to)
	if err != nil {
		return PeriodSummary{}, fmt.Errorf("query period balances: %w", err)
	}
	for rows.Next() {
		item := CurrencyPeriodSummary{ExpenseCategories: make([]ExpenseCategorySummary, 0)}
		if err := rows.Scan(&item.Currency, &item.OpeningBalanceMinor, &item.ClosingBalanceMinor); err != nil {
			rows.Close()
			return PeriodSummary{}, fmt.Errorf("scan period balances: %w", err)
		}
		byCurrency[item.Currency] = &item
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return PeriodSummary{}, fmt.Errorf("read period balances: %w", err)
	}
	rows.Close()

	rows, err = db.Query(ctx, `
		SELECT p.currency,
			COALESCE(SUM(p.amount_minor) FILTER (WHERE e.kind = 'income_received' AND p.direction = 'credit'), 0)::bigint,
			COALESCE(SUM(p.amount_minor) FILTER (WHERE e.kind = 'purchase' AND p.direction = 'debit'), 0)::bigint,
			COALESCE(SUM(p.amount_minor) FILTER (WHERE e.kind = 'refund' AND p.direction = 'credit'), 0)::bigint,
			COALESCE(SUM(p.amount_minor) FILTER (WHERE e.kind = 'debt_payment' AND p.direction = 'debit'), 0)::bigint,
			COALESCE(SUM(p.amount_minor) FILTER (WHERE e.kind = 'internal_transfer' AND p.direction = 'debit'), 0)::bigint
		FROM financial_events e JOIN postings p ON p.financial_event_id = e.id
		WHERE COALESCE(e.occurred_at,e.occurred_date::timestamp AT TIME ZONE 'Asia/Yekaterinburg') >= $1 AND COALESCE(e.occurred_at,e.occurred_date::timestamp AT TIME ZONE 'Asia/Yekaterinburg') < $2
		GROUP BY p.currency ORDER BY p.currency`, from, to)
	if err != nil {
		return PeriodSummary{}, fmt.Errorf("query period flows: %w", err)
	}
	for rows.Next() {
		var currency string
		var income, purchases, refunds, debtPayments, transfers int64
		if err := rows.Scan(&currency, &income, &purchases, &refunds, &debtPayments, &transfers); err != nil {
			rows.Close()
			return PeriodSummary{}, fmt.Errorf("scan period flows: %w", err)
		}
		item := byCurrency[currency]
		if item == nil {
			item = &CurrencyPeriodSummary{Currency: currency, ExpenseCategories: make([]ExpenseCategorySummary, 0)}
			byCurrency[currency] = item
		}
		item.IncomeMinor = income
		item.PurchasesMinor = purchases
		item.RefundsMinor = refunds
		item.DebtPaymentsMinor = debtPayments
		item.InternalTransferVolumeMinor = transfers
		item.NetExternalCashFlowMinor = income + refunds - purchases - debtPayments
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return PeriodSummary{}, fmt.Errorf("read period flows: %w", err)
	}
	rows.Close()

	if err := installmentHistoryCoverage(ctx, db, from, to, byCurrency); err != nil {
		return PeriodSummary{}, err
	}
	if err := addInstallmentSummary(ctx, db, from, to, byCurrency); err != nil {
		return PeriodSummary{}, err
	}
	rows, err = db.Query(ctx, `
		SELECT p.currency, e.category_id::text, COALESCE(c.name, 'Без категории'),
			COALESCE(SUM(p.amount_minor) FILTER (WHERE e.kind = 'purchase'), 0)::bigint,
			COALESCE(SUM(p.amount_minor) FILTER (WHERE e.kind = 'refund'), 0)::bigint
		FROM financial_events e
		JOIN postings p ON p.financial_event_id = e.id
		LEFT JOIN categories c ON c.id = e.category_id
		WHERE COALESCE(e.occurred_at,e.occurred_date::timestamp AT TIME ZONE 'Asia/Yekaterinburg') >= $1 AND COALESCE(e.occurred_at,e.occurred_date::timestamp AT TIME ZONE 'Asia/Yekaterinburg') < $2
			AND e.kind IN ('purchase', 'refund')
		GROUP BY p.currency, e.category_id, c.name
		ORDER BY p.currency, COALESCE(c.name, 'Без категории'), e.category_id`, from, to)
	if err != nil {
		return PeriodSummary{}, fmt.Errorf("query period categories: %w", err)
	}
	for rows.Next() {
		var currency string
		var category ExpenseCategorySummary
		if err := rows.Scan(&currency, &category.CategoryID, &category.CategoryName, &category.PurchasesMinor, &category.RefundsMinor); err != nil {
			rows.Close()
			return PeriodSummary{}, fmt.Errorf("scan period categories: %w", err)
		}
		category.NetExpenseMinor = category.PurchasesMinor - category.RefundsMinor
		item := byCurrency[currency]
		if item == nil {
			item = &CurrencyPeriodSummary{Currency: currency, ExpenseCategories: make([]ExpenseCategorySummary, 0)}
			byCurrency[currency] = item
		}
		item.ExpenseCategories = append(item.ExpenseCategories, category)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return PeriodSummary{}, fmt.Errorf("read period categories: %w", err)
	}
	rows.Close()

	splits, err := debtSplits(ctx, db, from, to, "")
	if err != nil {
		return PeriodSummary{}, err
	}
	for currency, buckets := range splits {
		if item := byCurrency[currency]; item != nil {
			item.DebtSplit = buckets[""]
		}
	}

	currencies := make([]CurrencyPeriodSummary, 0, len(byCurrency))
	keys := sortedKeys(byCurrency)
	for _, currency := range keys {
		currencies = append(currencies, *byCurrency[currency])
	}
	return PeriodSummary{From: from, To: to, Currencies: currencies}, nil
}

func (repo *Repository) IncomeRelativePeriod(ctx context.Context, asOf time.Time, categoryID *string) (IncomeRelativePeriod, error) {
	tx, err := repo.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return IncomeRelativePeriod{}, fmt.Errorf("begin income relative period: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if categoryID != nil {
		var kind string
		err := tx.QueryRow(ctx, "SELECT kind::text FROM categories WHERE id = $1", *categoryID).Scan(&kind)
		if errors.Is(err, pgx.ErrNoRows) || (err == nil && kind != "income") {
			return IncomeRelativePeriod{}, &IncomeAnchorError{Code: CodeInvalidAnchorCategory, Err: errors.New("category must exist and have income kind")}
		}
		if err != nil {
			return IncomeRelativePeriod{}, fmt.Errorf("validate income anchor category: %w", err)
		}
	}

	lowerBound := asOf.AddDate(-5, 0, 0)
	anchor := IncomeAnchor{Postings: make([]IncomeAnchorPosting, 0)}
	var categoryArgument any
	if categoryID != nil {
		categoryArgument = *categoryID
	}
	err = tx.QueryRow(ctx, `
		SELECT e.id::text, e.category_id::text, c.name, e.occurred_at
		FROM financial_events e
		LEFT JOIN categories c ON c.id = e.category_id
		WHERE e.kind = 'income_received'
			AND e.occurred_at >= $1 AND COALESCE(e.occurred_at,e.occurred_date::timestamp AT TIME ZONE 'Asia/Yekaterinburg') < $2
			AND ($3::uuid IS NULL OR e.category_id = $3::uuid)
		ORDER BY e.occurred_at DESC, e.created_at DESC, e.id DESC
		LIMIT 1`, lowerBound, asOf, categoryArgument).Scan(
		&anchor.EventID, &anchor.CategoryID, &anchor.CategoryName, &anchor.OccurredAt,
	)
	hasExactAnchor := err == nil
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return IncomeRelativePeriod{}, fmt.Errorf("find income anchor: %w", err)
	}
	anchorDay := historyDay(lowerBound)
	if hasExactAnchor {
		anchorDay = historyDay(anchor.OccurredAt)
	}
	var uncertainIncome bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM financial_events WHERE kind='income_received'
	AND occurred_date >= $1::date AND occurred_date <= $2::date AND occurred_date >= $3::date
	AND ($4::uuid IS NULL OR category_id=$4::uuid))`, historyDay(lowerBound), historyDay(asOf), anchorDay, categoryArgument).Scan(&uncertainIncome); err != nil {
		return IncomeRelativePeriod{}, err
	}
	if uncertainIncome {
		return IncomeRelativePeriod{}, &IncomeAnchorError{Code: CodeIncomeTimeUnknown}
	}
	if !hasExactAnchor {
		return IncomeRelativePeriod{}, &IncomeAnchorError{Code: CodeAnchorNotFound}
	}

	rows, err := tx.Query(ctx, `
		SELECT p.account_id::text, a.name, p.amount_minor, p.currency
		FROM postings p
		JOIN accounts a ON a.id = p.account_id
		WHERE p.financial_event_id = $1 AND p.direction = 'credit'
		ORDER BY p.created_at, p.id`, anchor.EventID)
	if err != nil {
		return IncomeRelativePeriod{}, fmt.Errorf("query income anchor postings: %w", err)
	}
	for rows.Next() {
		var posting IncomeAnchorPosting
		if err := rows.Scan(&posting.AccountID, &posting.AccountName, &posting.AmountMinor, &posting.Currency); err != nil {
			rows.Close()
			return IncomeRelativePeriod{}, fmt.Errorf("scan income anchor posting: %w", err)
		}
		anchor.Postings = append(anchor.Postings, posting)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return IncomeRelativePeriod{}, fmt.Errorf("read income anchor postings: %w", err)
	}
	rows.Close()

	summary, err := periodSummary(ctx, tx, anchor.OccurredAt, asOf)
	if err != nil {
		return IncomeRelativePeriod{}, fmt.Errorf("summarize income relative period: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return IncomeRelativePeriod{}, fmt.Errorf("commit income relative period snapshot: %w", err)
	}
	return IncomeRelativePeriod{Anchor: anchor, Summary: summary}, nil
}

func (repo *Repository) CashPosition(ctx context.Context, asOf, through string) (CashPosition, error) {
	tx, err := repo.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return CashPosition{}, fmt.Errorf("begin cash position: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	byCurrency := make(map[string]*CurrencyCashPosition)
	rows, err := tx.Query(ctx, `
		SELECT a.currency,
			CASE WHEN BOOL_AND(a.balance_reconciled) THEN (SUM(a.opening_balance_minor) + COALESCE(SUM(pe.effect_minor), 0))::bigint ELSE NULL END
		FROM accounts a
		LEFT JOIN (
			SELECT p.account_id, SUM(CASE p.direction WHEN 'credit' THEN p.amount_minor ELSE -p.amount_minor END) AS effect_minor
			FROM postings p GROUP BY p.account_id
		) pe ON pe.account_id = a.id
		WHERE a.type IN ('debit', 'cash')
		GROUP BY a.currency ORDER BY a.currency`)
	if err != nil {
		return CashPosition{}, fmt.Errorf("query liquid balances: %w", err)
	}
	for rows.Next() {
		item := CurrencyCashPosition{ScheduledPayments: make([]CashScheduledPayment, 0)}
		if err := rows.Scan(&item.Currency, &item.LiquidBalanceMinor); err != nil {
			rows.Close()
			return CashPosition{}, fmt.Errorf("scan liquid balances: %w", err)
		}
		byCurrency[item.Currency] = &item
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return CashPosition{}, fmt.Errorf("read liquid balances: %w", err)
	}
	rows.Close()

	rows, err = tx.Query(ctx, `
		SELECT o.currency, sp.id::text, o.id::text, o.name, sp.due_on::text,
			sp.amount_minor,
			(COALESCE(SUM(s.amount_minor), 0)+COALESCE((SELECT sum(x.amount_minor) FROM installment_payment_allocations x JOIN financial_events e ON e.id=x.event_id WHERE x.payment_id=sp.id AND x.active AND COALESCE(e.occurred_date,(e.occurred_at AT TIME ZONE 'Asia/Yekaterinburg')::date)<=$2::date),0))::bigint AS paid_minor,
			(sp.amount_minor - (COALESCE(SUM(s.amount_minor), 0)+COALESCE((SELECT sum(x.amount_minor) FROM installment_payment_allocations x JOIN financial_events e ON e.id=x.event_id WHERE x.payment_id=sp.id AND x.active AND COALESCE(e.occurred_date,(e.occurred_at AT TIME ZONE 'Asia/Yekaterinburg')::date)<=$2::date),0)))::bigint AS remaining_minor
		FROM scheduled_payments sp
		JOIN obligations o ON o.id = sp.obligation_id
		LEFT JOIN scheduled_payment_settlements s ON s.scheduled_payment_id = sp.id
		WHERE sp.due_on <= $1::date AND sp.plan_active AND NOT EXISTS(SELECT 1 FROM installment_plans i WHERE i.obligation_id=o.id AND (i.mode='credit_card' OR i.observed_on>$2::date))
		GROUP BY o.currency, sp.id, o.id, o.name, sp.due_on, sp.amount_minor
		HAVING sp.amount_minor - (COALESCE(SUM(s.amount_minor), 0)+COALESCE((SELECT sum(x.amount_minor) FROM installment_payment_allocations x JOIN financial_events e ON e.id=x.event_id WHERE x.payment_id=sp.id AND x.active AND COALESCE(e.occurred_date,(e.occurred_at AT TIME ZONE 'Asia/Yekaterinburg')::date)<=$2::date),0)) > 0
		ORDER BY o.currency, sp.due_on, sp.position, sp.id`, through, asOf)
	if err != nil {
		return CashPosition{}, fmt.Errorf("query outstanding obligations: %w", err)
	}
	for rows.Next() {
		var currency string
		var payment CashScheduledPayment
		if err := rows.Scan(&currency, &payment.ScheduledPaymentID, &payment.ObligationID, &payment.ObligationName,
			&payment.DueOn, &payment.AmountMinor, &payment.PaidMinor, &payment.RemainingMinor); err != nil {
			rows.Close()
			return CashPosition{}, fmt.Errorf("scan outstanding obligations: %w", err)
		}
		item := byCurrency[currency]
		if item == nil {
			zero := int64(0)
			item = &CurrencyCashPosition{Currency: currency, LiquidBalanceMinor: &zero, ScheduledPayments: make([]CashScheduledPayment, 0)}
			byCurrency[currency] = item
		}
		payment.DaysUntilDue = obligations.DueIn(asOf, payment.DueOn)
		if payment.DueOn < asOf {
			payment.Timing = "overdue"
			item.OverdueObligationsMinor += payment.RemainingMinor
		} else {
			// «Скоро» is still upcoming money; only the label changes.
			payment.Timing = "upcoming"
			if obligations.DueSoon(payment.DaysUntilDue, payment.RemainingMinor) {
				payment.Timing = "due_soon"
			}
			item.UpcomingObligationsMinor += payment.RemainingMinor
		}
		item.ScheduledPayments = append(item.ScheduledPayments, payment)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return CashPosition{}, fmt.Errorf("read outstanding obligations: %w", err)
	}
	rows.Close()

	if err := addCardRequirements(ctx, tx, asOf, through, byCurrency); err != nil {
		return CashPosition{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return CashPosition{}, fmt.Errorf("commit cash position snapshot: %w", err)
	}
	currencies := make([]CurrencyCashPosition, 0, len(byCurrency))
	for _, currency := range sortedKeys(byCurrency) {
		item := byCurrency[currency]
		item.ObligationsThroughMinor = item.OverdueObligationsMinor + item.UpcomingObligationsMinor
		if item.LiquidBalanceMinor != nil && !item.CoverageIncomplete {
			available := *item.LiquidBalanceMinor - item.ObligationsThroughMinor
			item.AvailableAfterObligationsMinor = &available
		}
		currencies = append(currencies, *item)
	}
	return CashPosition{AsOf: asOf, Through: through, Currencies: currencies}, nil
}

func sortedKeys[T any](items map[string]*T) []string {
	keys := make([]string, 0, len(items))
	for key := range items {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
