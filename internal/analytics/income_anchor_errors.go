package analytics

import (
	"errors"
	"fmt"
)

type IncomeAnchorErrorCode string

const (
	CodeIncomeTimeUnknown     IncomeAnchorErrorCode = "income_time_unknown"
	CodeInvalidAnchorCategory IncomeAnchorErrorCode = "invalid_anchor_category"
	CodeAnchorNotFound        IncomeAnchorErrorCode = "anchor_not_found"
)

type IncomeAnchorError struct {
	Code IncomeAnchorErrorCode
	Err  error
}

func (err *IncomeAnchorError) Error() string {
	if err.Err == nil {
		return string(err.Code)
	}
	return fmt.Sprintf("%s: %v", err.Code, err.Err)
}

func (err *IncomeAnchorError) Unwrap() error { return err.Err }

func IncomeAnchorErrorCodeOf(err error) (IncomeAnchorErrorCode, bool) {
	var anchorErr *IncomeAnchorError
	if !errors.As(err, &anchorErr) {
		return "", false
	}
	return anchorErr.Code, true
}
