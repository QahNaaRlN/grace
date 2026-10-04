package assistant

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/QahNaaRlN/grace/internal/accounts"
	"github.com/QahNaaRlN/grace/internal/analytics"
	"github.com/QahNaaRlN/grace/internal/obligations"
	"github.com/QahNaaRlN/grace/internal/receipts"
)

type fakeAccounts struct {
	items        []accounts.Account
	err          error
	calls        int
	contextValue any
}

func (fake *fakeAccounts) List(ctx context.Context) ([]accounts.Account, error) {
	fake.calls++
	fake.contextValue = ctx.Value(contextKey("proof"))
	return fake.items, fake.err
}

type fakeAnalytics struct {
	period        analytics.PeriodSummary
	cash          analytics.CashPosition
	incomePeriod  analytics.IncomeRelativePeriod
	err           error
	periodCalls   int
	cashCalls     int
	incomeCalls   int
	from, to      time.Time
	incomeAsOf    time.Time
	categoryID    *string
	asOf, through string
}

func (fake *fakeAnalytics) PeriodSummary(_ context.Context, from, to time.Time) (analytics.PeriodSummary, error) {
	fake.periodCalls++
	fake.from, fake.to = from, to
	return fake.period, fake.err
}

func (fake *fakeAnalytics) CashPosition(_ context.Context, asOf, through string) (analytics.CashPosition, error) {
	fake.cashCalls++
	fake.asOf, fake.through = asOf, through
	return fake.cash, fake.err
}

func (fake *fakeAnalytics) IncomeRelativePeriod(_ context.Context, asOf time.Time, categoryID *string) (analytics.IncomeRelativePeriod, error) {
	fake.incomeCalls++
	fake.incomeAsOf, fake.categoryID = asOf, categoryID
	return fake.incomePeriod, fake.err
}

type fakeObligations struct {
	items        []obligations.Obligation
	err          error
	calls, limit int
	asOf         string
}

func (fake *fakeObligations) List(_ context.Context, asOf string, limit int) ([]obligations.Obligation, error) {
	fake.calls++
	fake.asOf, fake.limit = asOf, limit
	return fake.items, fake.err
}

type fakeReceipts struct {
	items        []receipts.Receipt
	err          error
	calls, limit int
}

func (fake *fakeReceipts) List(_ context.Context, limit int) ([]receipts.Receipt, error) {
	fake.calls++
	fake.limit = limit
	return fake.items, fake.err
}

type contextKey string

func newTestExecutor() (*Executor, *fakeAccounts, *fakeAnalytics, *fakeObligations, *fakeReceipts) {
	accountReader := &fakeAccounts{items: []accounts.Account{{ID: "account-1", Currency: "RUB", BalanceMinor: 123}}}
	analyticsReader := &fakeAnalytics{
		period: analytics.PeriodSummary{Currencies: []analytics.CurrencyPeriodSummary{}},
		cash:   analytics.CashPosition{Currencies: []analytics.CurrencyCashPosition{}},
		incomePeriod: analytics.IncomeRelativePeriod{
			Anchor:  analytics.IncomeAnchor{EventID: "income-1", Postings: []analytics.IncomeAnchorPosting{}},
			Summary: analytics.PeriodSummary{Currencies: []analytics.CurrencyPeriodSummary{}},
		},
	}
	obligationReader := &fakeObligations{items: []obligations.Obligation{}}
	receiptReader := &fakeReceipts{items: []receipts.Receipt{}}
	executor := NewExecutor(accountReader, analyticsReader, obligationReader, receiptReader)
	executor.now = func() time.Time { return time.Date(2026, 10, 4, 12, 0, 0, 0, time.FixedZone("test", 5*60*60)) }
	return executor, accountReader, analyticsReader, obligationReader, receiptReader
}

func TestDefinitionsAreStableStrictAndReadOnly(t *testing.T) {
	definitions := Definitions()
	names := make([]string, 0, len(definitions))
	for _, definition := range definitions {
		names = append(names, definition.Name)
		if !definition.ReadOnly {
			t.Fatalf("tool %s must be read-only", definition.Name)
		}
		if definition.InputSchema.Schema != schemaDraft || definition.InputSchema.Type != "object" || definition.InputSchema.AdditionalProperties {
			t.Fatalf("tool %s has a non-strict schema: %#v", definition.Name, definition.InputSchema)
		}
	}
	want := []string{"get_accounts", "get_period_summary", "get_cash_position", "get_obligations", "get_receipts", "get_period_since_income"}
	if !reflect.DeepEqual(names, want) {
		t.Fatalf("unexpected definition order: %v", names)
	}
	if !reflect.DeepEqual(definitions[1].InputSchema.Required, []string{"from", "to"}) ||
		!reflect.DeepEqual(definitions[2].InputSchema.Required, []string{"through"}) ||
		!reflect.DeepEqual(definitions[5].InputSchema.Required, []string{"asOf"}) ||
		definitions[5].InputSchema.Properties["categoryId"].Format != "uuid" {
		t.Fatal("required fields do not match the accepted contract")
	}
	payload, err := json.Marshal(definitions)
	if err != nil || !json.Valid(payload) {
		t.Fatalf("definitions must serialize as JSON: %v", err)
	}
}

func TestExecutorDispatchesCanonicalDependencies(t *testing.T) {
	executor, accountReader, analyticsReader, obligationReader, receiptReader := newTestExecutor()
	ctx := context.WithValue(context.Background(), contextKey("proof"), "same-context")

	accountsResult, err := executor.Call(ctx, "get_accounts", json.RawMessage(`{}`))
	if err != nil || accountsResult.Name != "get_accounts" || accountReader.contextValue != "same-context" {
		t.Fatalf("unexpected accounts result: %#v %v", accountsResult, err)
	}
	if got := accountsResult.Data.(AccountsResult).Items[0].BalanceMinor; got != 123 {
		t.Fatalf("expected canonical balance, got %d", got)
	}

	_, err = executor.Call(ctx, "get_period_summary", json.RawMessage(`{"from":"2026-10-01T00:00:00+05:00","to":"2026-11-01T00:00:00+05:00"}`))
	if err != nil || analyticsReader.periodCalls != 1 || analyticsReader.from.Location() == time.UTC || !analyticsReader.from.Before(analyticsReader.to) {
		t.Fatalf("period was not passed to analytics: %v %#v", err, analyticsReader)
	}

	_, err = executor.Call(ctx, "get_cash_position", json.RawMessage(`{"through":"2026-11-01"}`))
	if err != nil || analyticsReader.asOf != "2026-10-04" || analyticsReader.through != "2026-11-01" {
		t.Fatalf("unexpected cash horizon: %s %s %v", analyticsReader.asOf, analyticsReader.through, err)
	}

	obligationsResult, err := executor.Call(ctx, "get_obligations", json.RawMessage(`{}`))
	if err != nil || obligationReader.asOf != "2026-10-04" || obligationReader.limit != 100 || obligationsResult.Data.(ObligationsResult).AsOf != "2026-10-04" {
		t.Fatalf("unexpected obligations call: %#v %v", obligationReader, err)
	}

	_, err = executor.Call(ctx, "get_receipts", json.RawMessage(`{}`))
	if err != nil || receiptReader.limit != 100 {
		t.Fatalf("unexpected default receipts call: %#v %v", receiptReader, err)
	}
	_, err = executor.Call(ctx, "get_receipts", json.RawMessage(`{"limit":25}`))
	if err != nil || receiptReader.limit != 25 || receiptReader.calls != 2 {
		t.Fatalf("unexpected receipts call: %#v %v", receiptReader, err)
	}

	incomeResult, err := executor.Call(ctx, "get_period_since_income", json.RawMessage(`{"asOf":"2026-10-05T12:00:00+05:00","categoryId":"20000000-0000-0000-0000-000000000004"}`))
	if err != nil || analyticsReader.incomeCalls != 1 || analyticsReader.categoryID == nil ||
		*analyticsReader.categoryID != "20000000-0000-0000-0000-000000000004" || analyticsReader.incomeAsOf.Location() == time.UTC ||
		incomeResult.Data.(analytics.IncomeRelativePeriod).Anchor.EventID != "income-1" {
		t.Fatalf("unexpected income-relative call: %#v %v", analyticsReader, err)
	}
}

func TestInvalidCallsDoNotReachDependencies(t *testing.T) {
	tests := []struct {
		name      string
		tool      string
		arguments string
		code      ErrorCode
	}{
		{name: "unknown tool", tool: "get_everything", arguments: `{}`, code: CodeUnknownTool},
		{name: "malformed", tool: "get_accounts", arguments: `{`, code: CodeInvalidArguments},
		{name: "non object", tool: "get_accounts", arguments: `[]`, code: CodeInvalidArguments},
		{name: "unknown field", tool: "get_accounts", arguments: `{"extra":true}`, code: CodeInvalidArguments},
		{name: "missing period", tool: "get_period_summary", arguments: `{"from":"2026-10-01T00:00:00Z"}`, code: CodeInvalidArguments},
		{name: "period without offset", tool: "get_period_summary", arguments: `{"from":"2026-10-01T00:00:00","to":"2026-11-01T00:00:00Z"}`, code: CodeInvalidArguments},
		{name: "reverse cash horizon", tool: "get_cash_position", arguments: `{"asOf":"2026-10-04","through":"2026-10-03"}`, code: CodeInvalidArguments},
		{name: "invalid asOf", tool: "get_obligations", arguments: `{"asOf":"2026-02-30"}`, code: CodeInvalidArguments},
		{name: "small receipt limit", tool: "get_receipts", arguments: `{"limit":0}`, code: CodeInvalidArguments},
		{name: "large receipt limit", tool: "get_receipts", arguments: `{"limit":101}`, code: CodeInvalidArguments},
		{name: "missing income asOf", tool: "get_period_since_income", arguments: `{}`, code: CodeInvalidArguments},
		{name: "income asOf without offset", tool: "get_period_since_income", arguments: `{"asOf":"2026-10-05T12:00:00"}`, code: CodeInvalidArguments},
		{name: "malformed income category", tool: "get_period_since_income", arguments: `{"asOf":"2026-10-05T12:00:00Z","categoryId":"not-a-uuid"}`, code: CodeInvalidArguments},
		{name: "null income category", tool: "get_period_since_income", arguments: `{"asOf":"2026-10-05T12:00:00Z","categoryId":null}`, code: CodeInvalidArguments},
		{name: "numeric income category", tool: "get_period_since_income", arguments: `{"asOf":"2026-10-05T12:00:00Z","categoryId":42}`, code: CodeInvalidArguments},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			executor, accountReader, analyticsReader, obligationReader, receiptReader := newTestExecutor()
			_, err := executor.Call(context.Background(), test.tool, json.RawMessage(test.arguments))
			if code, ok := ErrorCodeOf(err); !ok || code != test.code {
				t.Fatalf("expected %s, got %v", test.code, err)
			}
			if accountReader.calls+analyticsReader.periodCalls+analyticsReader.cashCalls+analyticsReader.incomeCalls+obligationReader.calls+receiptReader.calls != 0 {
				t.Fatal("invalid call reached a dependency")
			}
		})
	}
}

func TestIncomeAnchorErrorsHaveStableToolCodes(t *testing.T) {
	tests := []struct {
		name     string
		anchor   analytics.IncomeAnchorErrorCode
		expected ErrorCode
	}{
		{name: "invalid category", anchor: analytics.CodeInvalidAnchorCategory, expected: CodeInvalidAnchorCategory},
		{name: "not found", anchor: analytics.CodeAnchorNotFound, expected: CodeAnchorNotFound},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			executor, _, analyticsReader, _, _ := newTestExecutor()
			analyticsReader.err = &analytics.IncomeAnchorError{Code: test.anchor, Err: errors.New("test")}
			result, err := executor.Call(context.Background(), "get_period_since_income", json.RawMessage(`{"asOf":"2026-10-05T12:00:00Z"}`))
			if code, ok := ErrorCodeOf(err); !ok || code != test.expected || result.Data != nil {
				t.Fatalf("expected %s without data, got %#v %v", test.expected, result, err)
			}
		})
	}
}

func TestDependencyErrorsHaveStableCodes(t *testing.T) {
	executor, accountReader, _, _, _ := newTestExecutor()
	accountReader.err = errors.New("database unavailable")
	result, err := executor.Call(context.Background(), "get_accounts", json.RawMessage(`{}`))
	if code, ok := ErrorCodeOf(err); !ok || code != CodeDependencyFailure || result.Data != nil {
		t.Fatalf("expected dependency failure without data, got %#v %v", result, err)
	}

	accountReader.err = context.Canceled
	result, err = executor.Call(context.Background(), "get_accounts", json.RawMessage(`{}`))
	if code, ok := ErrorCodeOf(err); !ok || code != CodeCanceled || result.Data != nil {
		t.Fatalf("expected cancellation without data, got %#v %v", result, err)
	}
}
