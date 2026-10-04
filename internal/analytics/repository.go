package analytics

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct{ pool *pgxpool.Pool }

func NewRepository(pool *pgxpool.Pool) *Repository { return &Repository{pool: pool} }

func (repo *Repository) PeriodSummary(ctx context.Context, from, to time.Time) (PeriodSummary, error) {
	tx, err := repo.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return PeriodSummary{}, fmt.Errorf("begin period summary: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	byCurrency := make(map[string]*CurrencyPeriodSummary)
	rows, err := tx.Query(ctx, `
		WITH account_opening AS (
			SELECT currency, SUM(opening_balance_minor)::bigint AS opening_minor
			FROM accounts GROUP BY currency
		), posting_effects AS (
			SELECT p.currency,
				COALESCE(SUM(CASE WHEN e.occurred_at < $1 THEN
					CASE p.direction WHEN 'credit' THEN p.amount_minor ELSE -p.amount_minor END
				ELSE 0 END), 0)::bigint AS before_from,
				COALESCE(SUM(CASE WHEN e.occurred_at < $2 THEN
					CASE p.direction WHEN 'credit' THEN p.amount_minor ELSE -p.amount_minor END
				ELSE 0 END), 0)::bigint AS before_to
			FROM postings p JOIN financial_events e ON e.id = p.financial_event_id
			GROUP BY p.currency
		), currencies AS (
			SELECT currency FROM account_opening UNION SELECT currency FROM posting_effects
		)
		SELECT c.currency,
			(COALESCE(a.opening_minor, 0) + COALESCE(p.before_from, 0))::bigint,
			(COALESCE(a.opening_minor, 0) + COALESCE(p.before_to, 0))::bigint
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

	rows, err = tx.Query(ctx, `
		SELECT p.currency,
			COALESCE(SUM(p.amount_minor) FILTER (WHERE e.kind = 'income_received' AND p.direction = 'credit'), 0)::bigint,
			COALESCE(SUM(p.amount_minor) FILTER (WHERE e.kind = 'purchase' AND p.direction = 'debit'), 0)::bigint,
			COALESCE(SUM(p.amount_minor) FILTER (WHERE e.kind = 'refund' AND p.direction = 'credit'), 0)::bigint,
			COALESCE(SUM(p.amount_minor) FILTER (WHERE e.kind = 'debt_payment' AND p.direction = 'debit'), 0)::bigint,
			COALESCE(SUM(p.amount_minor) FILTER (WHERE e.kind = 'internal_transfer' AND p.direction = 'debit'), 0)::bigint
		FROM financial_events e JOIN postings p ON p.financial_event_id = e.id
		WHERE e.occurred_at >= $1 AND e.occurred_at < $2
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

	rows, err = tx.Query(ctx, `
		SELECT p.currency, e.category_id::text, COALESCE(c.name, 'Без категории'),
			COALESCE(SUM(p.amount_minor) FILTER (WHERE e.kind = 'purchase'), 0)::bigint,
			COALESCE(SUM(p.amount_minor) FILTER (WHERE e.kind = 'refund'), 0)::bigint
		FROM financial_events e
		JOIN postings p ON p.financial_event_id = e.id
		LEFT JOIN categories c ON c.id = e.category_id
		WHERE e.occurred_at >= $1 AND e.occurred_at < $2
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

	if err := tx.Commit(ctx); err != nil {
		return PeriodSummary{}, fmt.Errorf("commit period summary snapshot: %w", err)
	}
	currencies := make([]CurrencyPeriodSummary, 0, len(byCurrency))
	keys := sortedKeys(byCurrency)
	for _, currency := range keys {
		currencies = append(currencies, *byCurrency[currency])
	}
	return PeriodSummary{From: from, To: to, Currencies: currencies}, nil
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
			(SUM(a.opening_balance_minor) + COALESCE(SUM(pe.effect_minor), 0))::bigint
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
			COALESCE(SUM(s.amount_minor), 0)::bigint AS paid_minor,
			(sp.amount_minor - COALESCE(SUM(s.amount_minor), 0))::bigint AS remaining_minor
		FROM scheduled_payments sp
		JOIN obligations o ON o.id = sp.obligation_id
		LEFT JOIN scheduled_payment_settlements s ON s.scheduled_payment_id = sp.id
		WHERE sp.due_on <= $1::date
		GROUP BY o.currency, sp.id, o.id, o.name, sp.due_on, sp.amount_minor
		HAVING sp.amount_minor - COALESCE(SUM(s.amount_minor), 0) > 0
		ORDER BY o.currency, sp.due_on, sp.position, sp.id`, through)
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
			item = &CurrencyCashPosition{Currency: currency, ScheduledPayments: make([]CashScheduledPayment, 0)}
			byCurrency[currency] = item
		}
		if payment.DueOn < asOf {
			payment.Timing = "overdue"
			item.OverdueObligationsMinor += payment.RemainingMinor
		} else {
			payment.Timing = "upcoming"
			item.UpcomingObligationsMinor += payment.RemainingMinor
		}
		item.ScheduledPayments = append(item.ScheduledPayments, payment)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return CashPosition{}, fmt.Errorf("read outstanding obligations: %w", err)
	}
	rows.Close()

	if err := tx.Commit(ctx); err != nil {
		return CashPosition{}, fmt.Errorf("commit cash position snapshot: %w", err)
	}
	currencies := make([]CurrencyCashPosition, 0, len(byCurrency))
	for _, currency := range sortedKeys(byCurrency) {
		item := byCurrency[currency]
		item.ObligationsThroughMinor = item.OverdueObligationsMinor + item.UpcomingObligationsMinor
		item.AvailableAfterObligationsMinor = item.LiquidBalanceMinor - item.ObligationsThroughMinor
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
