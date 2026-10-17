package analytics

import (
	"context"
	"errors"
	"fmt"
	"math"
	"math/big"
	"net/http"
	"sort"
	"time"

	"github.com/QahNaaRlN/grace/internal/platform/web"
	"github.com/jackc/pgx/v5"
)

type Statistics struct {
	From        string               `json:"from"`
	To          string               `json:"to"`
	Granularity string               `json:"granularity"`
	Coverage    string               `json:"coverage"`
	Currencies  []StatisticsCurrency `json:"currencies"`
}
type StatisticsCurrency struct {
	Currency   string                `json:"currency"`
	Summary    CurrencyPeriodSummary `json:"summary"`
	Categories []StatisticsCategory  `json:"categories"`
	Series     []StatisticsBucket    `json:"series"`
}
type StatisticsCategory struct {
	ExpenseCategorySummary
	ShareBasisPoints int64 `json:"shareBasisPoints"`
}
type StatisticsBucket struct {
	InstallmentPrincipalRepaidMinor int64  `json:"installmentPrincipalRepaidMinor"`
	From                            string `json:"from"`
	To                              string `json:"to"`
	IncomeMinor                     int64  `json:"incomeMinor"`
	PurchasesMinor                  int64  `json:"purchasesMinor"`
	RefundsMinor                    int64  `json:"refundsMinor"`
	DebtPaymentsMinor               int64  `json:"debtPaymentsMinor"`
	DebtSplit
	OutflowsMinor int64 `json:"outflowsMinor"`
}

func statisticsPeriod(fromRaw, toRaw, granularity string) (time.Time, time.Time, error) {
	zone, err := time.LoadLocation("Asia/Yekaterinburg")
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	from, e1 := time.ParseInLocation(dateLayout, fromRaw, zone)
	to, e2 := time.ParseInLocation(dateLayout, toRaw, zone)
	if e1 != nil || e2 != nil || from.Format(dateLayout) != fromRaw || to.Format(dateLayout) != toRaw || !from.Before(to) || to.After(from.AddDate(1, 0, 0)) || (granularity != "week" && granularity != "month") {
		return time.Time{}, time.Time{}, ErrInvalidPeriod
	}
	return from, to, nil
}
func bucketStart(t time.Time, granularity string) time.Time {
	if granularity == "month" {
		return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, t.Location())
	}
	return t.AddDate(0, 0, -(int(t.Weekday())+6)%7)
}
func statisticsBuckets(from, to time.Time, granularity string) []StatisticsBucket {
	result := make([]StatisticsBucket, 0)
	for cursor := bucketStart(from, granularity); cursor.Before(to); {
		next := cursor.AddDate(0, 0, 7)
		if granularity == "month" {
			next = cursor.AddDate(0, 1, 0)
		}
		left, right := cursor, next
		if left.Before(from) {
			left = from
		}
		if right.After(to) {
			right = to
		}
		result = append(result, StatisticsBucket{From: left.Format(dateLayout), To: right.Format(dateLayout)})
		cursor = next
	}
	return result
}
func checkedOutflows(purchases, debt int64) (int64, error) {
	if purchases < 0 || debt < 0 || purchases > math.MaxInt64-debt {
		return 0, errors.New("statistics amount overflow")
	}
	return purchases + debt, nil
}
func categoryShare(purchases, total int64) int64 {
	if total <= 0 {
		return 0
	}
	n := new(big.Int).Mul(big.NewInt(purchases), big.NewInt(10000))
	return n.Quo(n, big.NewInt(total)).Int64()
}

func chartAmountSafe(amounts ...int64) bool {
	for _, amount := range amounts {
		if amount > 9007199254740991 || amount < -9007199254740991 {
			return false
		}
	}
	return true
}

func (repo *Repository) Statistics(ctx context.Context, from, to time.Time, granularity string) (Statistics, error) {
	tx, err := repo.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return Statistics{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	result, err := statisticsSnapshot(ctx, tx, from, to, granularity)
	if err != nil {
		return Statistics{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return Statistics{}, err
	}
	return result, nil
}
func statisticsSnapshot(ctx context.Context, db periodQuerier, from, to time.Time, granularity string) (Statistics, error) {
	summary, err := periodSummary(ctx, db, from, to)
	if err != nil {
		return Statistics{}, err
	}
	result := Statistics{From: from.Format(dateLayout), To: to.Format(dateLayout), Granularity: granularity, Coverage: "recorded_only", Currencies: make([]StatisticsCurrency, 0, len(summary.Currencies))}
	byCurrency := make(map[string]int)
	for _, item := range summary.Currencies {
		if !chartAmountSafe(item.IncomeMinor, item.PurchasesMinor, item.RefundsMinor, item.DebtPaymentsMinor, item.NetExternalCashFlowMinor, item.InternalTransferVolumeMinor) || (item.OpeningBalanceMinor != nil && !chartAmountSafe(*item.OpeningBalanceMinor)) || (item.ClosingBalanceMinor != nil && !chartAmountSafe(*item.ClosingBalanceMinor)) {
			return Statistics{}, errors.New("statistics exceeds client precision")
		}
		categories := make([]StatisticsCategory, 0, len(item.ExpenseCategories))
		for _, category := range item.ExpenseCategories {
			if !chartAmountSafe(category.PurchasesMinor, category.RefundsMinor, category.NetExpenseMinor) {
				return Statistics{}, errors.New("statistics exceeds client precision")
			}
			categories = append(categories, StatisticsCategory{category, categoryShare(category.PurchasesMinor, item.PurchasesMinor)})
		}
		sort.SliceStable(categories, func(i, j int) bool { return categories[i].PurchasesMinor > categories[j].PurchasesMinor })
		byCurrency[item.Currency] = len(result.Currencies)
		result.Currencies = append(result.Currencies, StatisticsCurrency{Currency: item.Currency, Summary: item, Categories: categories, Series: statisticsBuckets(from, to, granularity)})
	}
	rows, err := db.Query(ctx, `SELECT p.currency,
   to_char(date_trunc($3::text, COALESCE(e.occurred_at,e.occurred_date::timestamp AT TIME ZONE 'Asia/Yekaterinburg') AT TIME ZONE 'Asia/Yekaterinburg'),'YYYY-MM-DD'),
   COALESCE(SUM(p.amount_minor) FILTER(WHERE e.kind='income_received' AND p.direction='credit'),0)::bigint,
   COALESCE(SUM(p.amount_minor) FILTER(WHERE e.kind='purchase' AND p.direction='debit'),0)::bigint,
   COALESCE(SUM(p.amount_minor) FILTER(WHERE e.kind='refund' AND p.direction='credit'),0)::bigint,
   COALESCE(SUM(p.amount_minor) FILTER(WHERE e.kind='debt_payment' AND p.direction='debit'),0)::bigint
  FROM financial_events e JOIN postings p ON p.financial_event_id=e.id
  WHERE COALESCE(e.occurred_at,e.occurred_date::timestamp AT TIME ZONE 'Asia/Yekaterinburg') >= $1
    AND COALESCE(e.occurred_at,e.occurred_date::timestamp AT TIME ZONE 'Asia/Yekaterinburg') < $2
    AND e.kind IN ('income_received','purchase','refund','debt_payment')
  GROUP BY p.currency,2 ORDER BY p.currency,2`, from, to, granularity)
	if err != nil {
		return Statistics{}, fmt.Errorf("query statistics series: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var currency, day string
		var value StatisticsBucket
		if err = rows.Scan(&currency, &day, &value.IncomeMinor, &value.PurchasesMinor, &value.RefundsMinor, &value.DebtPaymentsMinor); err != nil {
			return Statistics{}, err
		}
		value.OutflowsMinor, err = checkedOutflows(value.PurchasesMinor, value.DebtPaymentsMinor)
		if err != nil {
			return Statistics{}, err
		}
		if !chartAmountSafe(value.IncomeMinor, value.PurchasesMinor, value.RefundsMinor, value.DebtPaymentsMinor, value.OutflowsMinor) {
			return Statistics{}, errors.New("statistics exceeds client precision")
		}
		index, ok := byCurrency[currency]
		if !ok {
			return Statistics{}, errors.New("statistics currency mismatch")
		}
		series := result.Currencies[index].Series
		matched := false
		for i := range series {
			left, _ := time.ParseInLocation(dateLayout, series[i].From, from.Location())
			if bucketStart(left, granularity).Format(dateLayout) == day {
				value.From, value.To = series[i].From, series[i].To
				series[i] = value
				matched = true
				break
			}
		}
		if !matched {
			return Statistics{}, errors.New("statistics bucket mismatch")
		}
	}
	if err = rows.Err(); err != nil {
		return Statistics{}, err
	}
	rows.Close()
	splits, err := debtSplits(ctx, db, from, to, granularity)
	if err != nil {
		return Statistics{}, err
	}
	for currency, buckets := range splits {
		index, ok := byCurrency[currency]
		if !ok {
			return Statistics{}, errors.New("statistics currency mismatch")
		}
		series := result.Currencies[index].Series
		for i := range series {
			left, _ := time.ParseInLocation(dateLayout, series[i].From, from.Location())
			if split, ok := buckets[bucketStart(left, granularity).Format(dateLayout)]; ok {
				series[i].DebtSplit = split
			}
		}
	}
	if err := addInstallmentBuckets(ctx, db, from, to, granularity, &result); err != nil {
		return Statistics{}, err
	}
	return result, nil
}
func (handler *Handler) Statistics(w http.ResponseWriter, r *http.Request) {
	granularity := r.URL.Query().Get("granularity")
	if granularity == "" {
		granularity = "month"
	}
	from, to, err := statisticsPeriod(r.URL.Query().Get("from"), r.URL.Query().Get("to"), granularity)
	if err != nil {
		web.WriteError(w, 422, "validation_error", "Выберите календарный период до одного года и группировку week или month.")
		return
	}
	result, err := handler.repo.Statistics(r.Context(), from, to, granularity)
	if err != nil {
		web.WriteError(w, 500, "internal_error", "Не удалось рассчитать статистику.")
		return
	}
	web.WriteJSON(w, 200, result)
}
