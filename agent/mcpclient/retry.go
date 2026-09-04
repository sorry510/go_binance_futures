package mcpclient

import (
	"context"
	"errors"
	"net"
	"strings"
	"time"
)

const (
	mcpReadAttemptTimeout = 30 * time.Second
	mcpRetryBackoff       = 150 * time.Millisecond
)

func withMCPReadRetry[T any](ctx context.Context, call func(context.Context) (T, error)) (T, error) {
	var zero T
	var lastErr error
	for attempt := 0; attempt < 2; attempt++ {
		if err := ctx.Err(); err != nil {
			return zero, err
		}
		attemptCtx, cancel := context.WithTimeout(ctx, mcpReadAttemptTimeout)
		value, err := call(attemptCtx)
		cancel()
		if err == nil {
			return value, nil
		}
		lastErr = err
		if attempt == 1 || !retryableMCPTransportError(ctx, err) {
			break
		}
		select {
		case <-ctx.Done():
			return zero, ctx.Err()
		case <-time.After(mcpRetryBackoff):
		}
	}
	return zero, lastErr
}

func retryableMCPTransportError(ctx context.Context, err error) bool {
	if err == nil || ctx.Err() != nil {
		return false
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	var networkErr net.Error
	if errors.As(err, &networkErr) {
		return networkErr.Timeout() || networkErr.Temporary()
	}
	message := strings.ToLower(err.Error())
	for _, marker := range []string{"connection reset", "unexpected eof", "connection refused", "bad gateway", "service unavailable", "gateway timeout", "status 502", "status 503", "status 504"} {
		if strings.Contains(message, marker) {
			return true
		}
	}
	return false
}
