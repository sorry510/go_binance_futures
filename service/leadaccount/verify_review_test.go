package leadaccount

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	binanceapi "go_binance_futures/feature/api/binance"
)

type deadlineLeadReader struct{ *fakeReader }

func (r *deadlineLeadReader) LeadTraderStatus(ctx context.Context) (binanceapi.LeadTraderStatus, error) {
	<-ctx.Done()
	return binanceapi.LeadTraderStatus{}, ctx.Err()
}

func TestStage41ReviewDeadlineReturnsError(t *testing.T) {
	store := newTestEncryptedStore(t)
	fake := validFakeReader()
	block := false
	m := newManager(store, func(Credentials) (AccountReader, error) {
		if block {
			return &deadlineLeadReader{fake}, nil
		}
		return fake, nil
	})
	if err := m.SaveCredentials(Credentials{APIKey: "test-key", APISecret: "test-secret"}); err != nil {
		t.Fatal(err)
	}
	first, err := m.VerifyReadOnly(context.Background())
	if err != nil || !first.ReadOnlyChecksPassed {
		t.Fatalf("initial verify: %v %+v", err, first)
	}
	block = true
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Millisecond)
	defer cancel()
	report, err := m.VerifyReadOnly(ctx)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("missing deadline error: %v", err)
	}
	if report.ReadOnlyChecksPassed || report.TradingReady || !hasReason(report.BlockingReasons, "verification_deadline_exceeded") {
		t.Fatalf("timeout returned success: %+v", report)
	}
	if saved := m.Status(); saved.ReadOnlyChecksPassed {
		t.Fatalf("saved stale success: %+v", saved)
	}
}
func TestStage41ReviewAlreadyCanceledSkipsNetwork(t *testing.T) {
	store := newTestEncryptedStore(t)
	reader := validFakeReader()
	m := newManager(store, func(Credentials) (AccountReader, error) { return reader, nil })
	if err := m.SaveCredentials(Credentials{APIKey: "key", APISecret: "secret"}); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	report, err := m.VerifyReadOnly(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected cancellation: %v", err)
	}
	if len(reader.calls) != 0 {
		t.Fatalf("unexpected remote calls: %v", reader.calls)
	}
	if report.ReadOnlyChecksPassed || !hasReason(report.BlockingReasons, "verification_canceled") {
		t.Fatalf("cancel result: %+v", report)
	}
}
func TestStage41ReviewLateReaderMustNotPass(t *testing.T) {
	store := newTestEncryptedStore(t)
	reader := validFakeReader()
	reader.statusStarted = make(chan struct{})
	reader.statusRelease = make(chan struct{})
	m := newManager(store, func(Credentials) (AccountReader, error) { return reader, nil })
	if err := m.SaveCredentials(Credentials{APIKey: "key", APISecret: "secret"}); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	out := make(chan error, 1)
	go func() {
		report, err := m.VerifyReadOnly(ctx)
		if report.ReadOnlyChecksPassed || !hasReason(report.BlockingReasons, "verification_canceled") {
			out <- errors.New("late reply incorrectly passed")
			return
		}
		out <- err
	}()
	<-reader.statusStarted
	cancel()
	close(reader.statusRelease)
	if err := <-out; !errors.Is(err, context.Canceled) {
		t.Fatalf("context not returned: %v", err)
	}
}
func TestStage41ReviewVerificationReasonNoRawData(t *testing.T) {
	r := verifyReader(context.Background(), validFakeReader())
	if !r.ReadOnlyChecksPassed || r.TradingReady || !hasReason(r.BlockingReasons, "stage7_live_authorization_required") {
		t.Fatalf("validation result unexpected: %+v", r)
	}
	if strings.Contains(strings.Join(r.BlockingReasons, ","), "secret") {
		t.Fatal("raw credentials in status")
	}
}
