package mysqlretry

import (
	"context"
	"errors"
	"testing"

	"github.com/go-sql-driver/mysql"
)

func TestOnlyInnoDBTransientErrorsAreRetried(t *testing.T) {
	for _, code := range []uint16{1213, 1205} {
		t.Run((&mysql.MySQLError{Number: code}).Error(), func(t *testing.T) {
			attempts := 0
			err := Do(context.Background(), func() error {
				attempts++
				if attempts <= 2 {
					return &mysql.MySQLError{Number: code, Message: "transient"}
				}
				return nil
			})
			if err != nil || attempts != 3 {
				t.Fatalf("attempts=%d err=%v", attempts, err)
			}
		})
	}
	attempts := 0
	if err := Do(context.Background(), func() error {
		attempts++
		return &mysql.MySQLError{Number: 1062, Message: "duplicate"}
	}); err == nil || attempts != 1 {
		t.Fatalf("non-retryable SQL error attempts=%d err=%v", attempts, err)
	}
}
func TestDeadlockRetriesBoundedAndCanceled(t *testing.T) {
	attempts := 0
	err := Do(context.Background(), func() error {
		attempts++
		return &mysql.MySQLError{Number: 1213}
	})
	if attempts != MaxAttempts || !IsRetryable(err) {
		t.Fatalf("attempts=%d err=%v", attempts, err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	attempts = 0
	err = Do(ctx, func() error { attempts++; return errors.New("unexpected") })
	if !errors.Is(err, context.Canceled) || attempts != 0 {
		t.Fatalf("context canceled attempts=%d err=%v", attempts, err)
	}
	if IsRetryable(errors.New("Deadlock found 1213")) {
		t.Fatal("plain text error must not be retried")
	}
}
