package assistant

import (
	"errors"
	"fmt"
)

type ErrorCode string

const (
	CodeUnknownTool       ErrorCode = "unknown_tool"
	CodeInvalidArguments  ErrorCode = "invalid_arguments"
	CodeDependencyFailure ErrorCode = "dependency_failure"
	CodeCanceled          ErrorCode = "canceled"
)

type ToolError struct {
	Code ErrorCode
	Err  error
}

func (err *ToolError) Error() string {
	if err.Err == nil {
		return string(err.Code)
	}
	return fmt.Sprintf("%s: %v", err.Code, err.Err)
}

func (err *ToolError) Unwrap() error { return err.Err }

func ErrorCodeOf(err error) (ErrorCode, bool) {
	var toolErr *ToolError
	if !errors.As(err, &toolErr) {
		return "", false
	}
	return toolErr.Code, true
}
