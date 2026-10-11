// Package mysqlretry retries independent, idempotent SQL statements that
// InnoDB has already rolled back after a transient deadlock/lock wait timeout.
// Never use this around a multi-statement transaction or a non-idempotent order.
package mysqlretry

import (
	"context"
	"errors"
	"time"

	"github.com/go-sql-driver/mysql"
)

const MaxAttempts = 4

// IsRetryable reports lock-cycle (1213) and lock-wait timeout (1205) errors.
// Callers must ensure each attempt is a fresh, standalone SQL statement.
func IsRetryable(err error) bool {
	var sqlErr *mysql.MySQLError
	return errors.As(err, &sqlErr) && (sqlErr.Number == 1213 || sqlErr.Number == 1205)
}

// Do returns the underlying SQL error if attempts are exhausted. Retrying
// only an independent idempotent statement prevents duplicate side-effects.
func Do(ctx context.Context, exec func() error) error {
	if ctx == nil {
		ctx = context.Background()
	}
	for attempt := 0; attempt < MaxAttempts; attempt++ {
		if err := ctx.Err(); err != nil {
			return err
		}
		err := exec()
		if err == nil {
			return nil
		}
		if !IsRetryable(err) || attempt == MaxAttempts-1 {
			return err
		}
		pause := time.Duration(25<<attempt) * time.Millisecond
		timer := time.NewTimer(pause)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
	return nil
}
