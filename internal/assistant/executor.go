package assistant

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/QahNaaRlN/grace/internal/accounts"
	"github.com/QahNaaRlN/grace/internal/analytics"
	"github.com/QahNaaRlN/grace/internal/obligations"
	"github.com/QahNaaRlN/grace/internal/receipts"
)

type AccountsReader interface {
	List(context.Context) ([]accounts.Account, error)
}

type AnalyticsReader interface {
	PeriodSummary(context.Context, time.Time, time.Time) (analytics.PeriodSummary, error)
	CashPosition(context.Context, string, string) (analytics.CashPosition, error)
	IncomeRelativePeriod(context.Context, time.Time, *string) (analytics.IncomeRelativePeriod, error)
}

type ObligationsReader interface {
	List(context.Context, string, int) ([]obligations.Obligation, error)
}

type ReceiptsReader interface {
	List(context.Context, int) ([]receipts.Receipt, error)
}

type ToolResult struct {
	Name string `json:"name"`
	Data any    `json:"data"`
}

type AccountsResult struct {
	Items []accounts.Account `json:"items"`
}

type ObligationsResult struct {
	AsOf  string                   `json:"asOf"`
	Items []obligations.Obligation `json:"items"`
}

type ReceiptsResult struct {
	Items []receipts.Receipt `json:"items"`
}

type Executor struct {
	accounts    AccountsReader
	analytics   AnalyticsReader
	obligations ObligationsReader
	receipts    ReceiptsReader
	now         func() time.Time
}

func NewExecutor(accountsReader AccountsReader, analyticsReader AnalyticsReader, obligationsReader ObligationsReader, receiptsReader ReceiptsReader) *Executor {
	return &Executor{
		accounts: accountsReader, analytics: analyticsReader,
		obligations: obligationsReader, receipts: receiptsReader,
		now: time.Now,
	}
}

func (executor *Executor) Call(ctx context.Context, name string, arguments json.RawMessage) (ToolResult, error) {
	switch name {
	case "get_accounts":
		var input struct{}
		if err := decodeArguments(arguments, &input); err != nil {
			return ToolResult{}, invalidArguments(err)
		}
		items, err := executor.accounts.List(ctx)
		if err != nil {
			return ToolResult{}, dependencyError(ctx, err)
		}
		return ToolResult{Name: name, Data: AccountsResult{Items: items}}, nil

	case "get_period_summary":
		var input struct {
			From string `json:"from"`
			To   string `json:"to"`
		}
		if err := decodeArguments(arguments, &input); err != nil {
			return ToolResult{}, invalidArguments(err)
		}
		from, to, err := analytics.ParsePeriod(input.From, input.To)
		if err != nil {
			return ToolResult{}, invalidArguments(err)
		}
		result, err := executor.analytics.PeriodSummary(ctx, from, to)
		if err != nil {
			return ToolResult{}, dependencyError(ctx, err)
		}
		return ToolResult{Name: name, Data: result}, nil

	case "get_cash_position":
		var input struct {
			AsOf    string `json:"asOf"`
			Through string `json:"through"`
		}
		if err := decodeArguments(arguments, &input); err != nil {
			return ToolResult{}, invalidArguments(err)
		}
		asOf, through, err := analytics.ParseCashHorizon(input.AsOf, input.Through, executor.now())
		if err != nil {
			return ToolResult{}, invalidArguments(err)
		}
		result, err := executor.analytics.CashPosition(ctx, asOf, through)
		if err != nil {
			return ToolResult{}, dependencyError(ctx, err)
		}
		return ToolResult{Name: name, Data: result}, nil

	case "get_obligations":
		var input struct {
			AsOf string `json:"asOf"`
		}
		if err := decodeArguments(arguments, &input); err != nil {
			return ToolResult{}, invalidArguments(err)
		}
		asOf, err := obligations.ParseAsOf(input.AsOf, executor.now())
		if err != nil {
			return ToolResult{}, invalidArguments(err)
		}
		items, err := executor.obligations.List(ctx, asOf, 100)
		if err != nil {
			return ToolResult{}, dependencyError(ctx, err)
		}
		return ToolResult{Name: name, Data: ObligationsResult{AsOf: asOf, Items: items}}, nil

	case "get_receipts":
		var input struct {
			Limit *int `json:"limit"`
		}
		if err := decodeArguments(arguments, &input); err != nil {
			return ToolResult{}, invalidArguments(err)
		}
		limit := 100
		if input.Limit != nil {
			limit = *input.Limit
		}
		if limit < 1 || limit > 100 {
			return ToolResult{}, invalidArguments(fmt.Errorf("limit must be between 1 and 100"))
		}
		items, err := executor.receipts.List(ctx, limit)
		if err != nil {
			return ToolResult{}, dependencyError(ctx, err)
		}
		return ToolResult{Name: name, Data: ReceiptsResult{Items: items}}, nil

	case "get_period_since_income":
		var input struct {
			AsOf       string          `json:"asOf"`
			CategoryID json.RawMessage `json:"categoryId"`
		}
		if err := decodeArguments(arguments, &input); err != nil {
			return ToolResult{}, invalidArguments(err)
		}
		var categoryIDRaw string
		if input.CategoryID != nil {
			if bytes.Equal(bytes.TrimSpace(input.CategoryID), []byte("null")) {
				return ToolResult{}, invalidArguments(fmt.Errorf("categoryId must be a UUID string"))
			}
			if err := json.Unmarshal(input.CategoryID, &categoryIDRaw); err != nil {
				return ToolResult{}, invalidArguments(fmt.Errorf("categoryId must be a UUID string"))
			}
		}
		asOf, categoryID, err := analytics.ParseIncomeAnchorQuery(input.AsOf, categoryIDRaw)
		if err != nil {
			return ToolResult{}, invalidArguments(err)
		}
		result, err := executor.analytics.IncomeRelativePeriod(ctx, asOf, categoryID)
		if err != nil {
			return ToolResult{}, dependencyError(ctx, err)
		}
		return ToolResult{Name: name, Data: result}, nil
	default:
		return ToolResult{}, &ToolError{Code: CodeUnknownTool, Err: fmt.Errorf("tool %q is not registered", name)}
	}
}

func decodeArguments(arguments json.RawMessage, destination any) error {
	var object map[string]json.RawMessage
	if len(arguments) == 0 || json.Unmarshal(arguments, &object) != nil || object == nil {
		return errors.New("arguments must be one JSON object")
	}
	decoder := json.NewDecoder(bytes.NewReader(arguments))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return errors.New("arguments must contain one JSON object")
	}
	return nil
}

func invalidArguments(err error) error {
	return &ToolError{Code: CodeInvalidArguments, Err: err}
}

func dependencyError(ctx context.Context, err error) error {
	if errors.Is(err, analytics.ErrTimePrecision) {
		return &ToolError{Code: CodeTimePrecision, Err: err}
	}
	if errors.Is(ctx.Err(), context.Canceled) || errors.Is(ctx.Err(), context.DeadlineExceeded) ||
		errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return &ToolError{Code: CodeCanceled, Err: err}
	}
	if code, ok := analytics.IncomeAnchorErrorCodeOf(err); ok {
		switch code {
		case analytics.CodeIncomeTimeUnknown:
			return &ToolError{Code: CodeTimePrecision, Err: err}
		case analytics.CodeInvalidAnchorCategory:
			return &ToolError{Code: CodeInvalidAnchorCategory, Err: err}
		case analytics.CodeAnchorNotFound:
			return &ToolError{Code: CodeAnchorNotFound, Err: err}
		}
	}
	return &ToolError{Code: CodeDependencyFailure, Err: err}
}
