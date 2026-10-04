package analytics

import "time"

type PeriodSummary struct {
	From       time.Time               `json:"from"`
	To         time.Time               `json:"to"`
	Currencies []CurrencyPeriodSummary `json:"currencies"`
}

type CurrencyPeriodSummary struct {
	Currency                    string                   `json:"currency"`
	OpeningBalanceMinor         int64                    `json:"openingBalanceMinor"`
	ClosingBalanceMinor         int64                    `json:"closingBalanceMinor"`
	IncomeMinor                 int64                    `json:"incomeMinor"`
	PurchasesMinor              int64                    `json:"purchasesMinor"`
	RefundsMinor                int64                    `json:"refundsMinor"`
	DebtPaymentsMinor           int64                    `json:"debtPaymentsMinor"`
	NetExternalCashFlowMinor    int64                    `json:"netExternalCashFlowMinor"`
	InternalTransferVolumeMinor int64                    `json:"internalTransferVolumeMinor"`
	ExpenseCategories           []ExpenseCategorySummary `json:"expenseCategories"`
}

type ExpenseCategorySummary struct {
	CategoryID      *string `json:"categoryId"`
	CategoryName    string  `json:"categoryName"`
	PurchasesMinor  int64   `json:"purchasesMinor"`
	RefundsMinor    int64   `json:"refundsMinor"`
	NetExpenseMinor int64   `json:"netExpenseMinor"`
}

type CashPosition struct {
	AsOf       string                 `json:"asOf"`
	Through    string                 `json:"through"`
	Currencies []CurrencyCashPosition `json:"currencies"`
}

type CurrencyCashPosition struct {
	Currency                       string                 `json:"currency"`
	LiquidBalanceMinor             int64                  `json:"liquidBalanceMinor"`
	OverdueObligationsMinor        int64                  `json:"overdueObligationsMinor"`
	UpcomingObligationsMinor       int64                  `json:"upcomingObligationsMinor"`
	ObligationsThroughMinor        int64                  `json:"obligationsThroughMinor"`
	AvailableAfterObligationsMinor int64                  `json:"availableAfterObligationsMinor"`
	ScheduledPayments              []CashScheduledPayment `json:"scheduledPayments"`
}

type CashScheduledPayment struct {
	ScheduledPaymentID string `json:"scheduledPaymentId"`
	ObligationID       string `json:"obligationId"`
	ObligationName     string `json:"obligationName"`
	DueOn              string `json:"dueOn"`
	AmountMinor        int64  `json:"amountMinor"`
	PaidMinor          int64  `json:"paidMinor"`
	RemainingMinor     int64  `json:"remainingMinor"`
	Timing             string `json:"timing"`
}
