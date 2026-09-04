package toolruntime

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
)

type ErrorType string

const (
	ErrorInvalidInput  ErrorType = "invalid_input"
	ErrorNotFound      ErrorType = "not_found"
	ErrorRateLimit     ErrorType = "rate_limit"
	ErrorTimeout       ErrorType = "timeout"
	ErrorUpstream      ErrorType = "upstream"
	ErrorStale         ErrorType = "stale"
	ErrorPartial       ErrorType = "partial"
	ErrorPermission    ErrorType = "permission"
	ErrorInternal      ErrorType = "internal"
	ErrorInputRequired ErrorType = "input_required"
)

type Error struct {
	Type    ErrorType
	Tool    string
	Message string
	Cause   error
}

func (err *Error) Error() string {
	if err == nil {
		return ""
	}
	if err.Message != "" {
		return err.Message
	}
	if err.Cause != nil {
		return err.Cause.Error()
	}
	return string(err.Type)
}
func (err *Error) Unwrap() error {
	if err == nil {
		return nil
	}
	return err.Cause
}

func newError(kind ErrorType, tool string, cause error, format string, args ...any) *Error {
	return &Error{Type: kind, Tool: tool, Cause: cause, Message: fmt.Sprintf(format, args...)}
}

func TypeOf(err error) ErrorType {
	var classified *Error
	if errors.As(err, &classified) {
		return classified.Type
	}
	return classifyToolError(err)
}

func classifyToolError(err error) ErrorType {
	if err == nil {
		return ""
	}
	var inputRequired interface{ InputRequired() bool }
	if errors.As(err, &inputRequired) && inputRequired.InputRequired() {
		return ErrorInputRequired
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return ErrorTimeout
	}
	if errors.Is(err, context.Canceled) {
		return ErrorTimeout
	}
	var networkErr net.Error
	if errors.As(err, &networkErr) {
		if networkErr.Timeout() {
			return ErrorTimeout
		}
		return ErrorUpstream
	}
	message := strings.ToLower(err.Error())
	switch {
	case strings.Contains(message, "429"), strings.Contains(message, "rate limit"), strings.Contains(message, "too many requests"):
		return ErrorRateLimit
	case strings.Contains(message, "not found"), strings.Contains(message, "no row"), strings.Contains(message, "does not exist"):
		return ErrorNotFound
	case strings.Contains(message, "timeout"), strings.Contains(message, "deadline exceeded"):
		return ErrorTimeout
	case strings.Contains(message, "connection reset"), strings.Contains(message, "connection refused"), strings.Contains(message, "bad gateway"), strings.Contains(message, "service unavailable"):
		return ErrorUpstream
	default:
		return ErrorInternal
	}
}
