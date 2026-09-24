package binance

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"go_binance_futures/service/binanceapiusage"

	"github.com/adshao/go-binance/v2/futures"
)

func TestAllPositionsCacheAndInvalidation(t *testing.T) {
	InvalidateAccountReadCache()
	t.Cleanup(InvalidateAccountReadCache)
	binanceapiusage.Default().ResetForTest()
	var calls atomic.Int32
	loader := func(context.Context) ([]*futures.PositionRisk, error) {
		calls.Add(1)
		return []*futures.PositionRisk{{Symbol: "BTCUSDT", PositionAmt: "1"}}, nil
	}
	ctx := binanceapiusage.WithSource(context.Background(), "test")

	first, err := getAllPositionsCached(ctx, loader)
	if err != nil {
		t.Fatal(err)
	}
	second, err := getAllPositionsCached(ctx, loader)
	if err != nil {
		t.Fatal(err)
	}
	if calls.Load() != 1 {
		t.Fatalf("loader calls=%d want=1", calls.Load())
	}
	if len(first) != 1 || len(second) != 1 {
		t.Fatalf("unexpected rows first=%+v second=%+v", first, second)
	}
	first[0].PositionAmt = "99"
	if second[0].PositionAmt != "1" {
		t.Fatalf("cache result was not cloned: %+v", second[0])
	}

	InvalidateAccountReadCache()
	if _, err := getAllPositionsCached(ctx, loader); err != nil {
		t.Fatal(err)
	}
	if calls.Load() != 2 {
		t.Fatalf("loader calls after invalidation=%d want=2", calls.Load())
	}
}

func TestAllPositionsCacheEmptySnapshot(t *testing.T) {
	InvalidateAccountReadCache()
	t.Cleanup(InvalidateAccountReadCache)
	binanceapiusage.Default().ResetForTest()

	var calls atomic.Int32
	loader := func(context.Context) ([]*futures.PositionRisk, error) {
		calls.Add(1)
		return []*futures.PositionRisk{}, nil
	}
	ctx := binanceapiusage.WithSource(context.Background(), "test")

	for i := 0; i < 3; i++ {
		rows, err := getAllPositionsCached(ctx, loader)
		if err != nil {
			t.Fatal(err)
		}
		if len(rows) != 0 {
			t.Fatalf("unexpected rows=%+v", rows)
		}
	}
	if calls.Load() != 1 {
		t.Fatalf("empty position snapshot was not cached: loader calls=%d want=1", calls.Load())
	}
}

func TestAllOpenOrdersSingleflightCoalescesConcurrentLoads(t *testing.T) {
	InvalidateAccountReadCache()
	t.Cleanup(InvalidateAccountReadCache)
	binanceapiusage.Default().ResetForTest()
	var calls atomic.Int32
	started := make(chan struct{})
	release := make(chan struct{})
	loader := func(context.Context) ([]*futures.Order, error) {
		if calls.Add(1) == 1 {
			close(started)
		}
		<-release
		return []*futures.Order{{Symbol: "BTCUSDT", OrderID: 1}}, nil
	}
	ctx := binanceapiusage.WithSource(context.Background(), "test")
	var wg sync.WaitGroup
	errs := make(chan error, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := getAllOpenOrdersCached(ctx, loader)
			errs <- err
		}()
	}
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("loader did not start")
	}
	close(release)
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	if calls.Load() != 1 {
		t.Fatalf("loader calls=%d want=1", calls.Load())
	}
}

func TestFreshPositionsBypassCacheAndReplaceSnapshot(t *testing.T) {
	InvalidateAccountReadCache()
	t.Cleanup(InvalidateAccountReadCache)

	var cachedCalls atomic.Int32
	cachedLoader := func(context.Context) ([]*futures.PositionRisk, error) {
		cachedCalls.Add(1)
		return []*futures.PositionRisk{{Symbol: "BTCUSDT", PositionAmt: "1"}}, nil
	}
	if _, err := getAllPositionsCached(context.Background(), cachedLoader); err != nil {
		t.Fatal(err)
	}

	var freshCalls atomic.Int32
	fresh, err := getAllPositionsFresh(context.Background(), func(context.Context) ([]*futures.PositionRisk, error) {
		freshCalls.Add(1)
		return []*futures.PositionRisk{{Symbol: "BTCUSDT", PositionAmt: "2"}}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if freshCalls.Load() != 1 || len(fresh) != 1 || fresh[0].PositionAmt != "2" {
		t.Fatalf("fresh result calls=%d rows=%+v", freshCalls.Load(), fresh)
	}

	var unexpected atomic.Int32
	after, err := getAllPositionsCached(context.Background(), func(context.Context) ([]*futures.PositionRisk, error) {
		unexpected.Add(1)
		return []*futures.PositionRisk{{Symbol: "BTCUSDT", PositionAmt: "3"}}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if unexpected.Load() != 0 || len(after) != 1 || after[0].PositionAmt != "2" {
		t.Fatalf("fresh snapshot was not retained: loader_calls=%d rows=%+v", unexpected.Load(), after)
	}
}

func TestInvalidationPreventsOlderInFlightPositionLoadFromRepopulatingCache(t *testing.T) {
	InvalidateAccountReadCache()
	t.Cleanup(InvalidateAccountReadCache)

	started := make(chan struct{})
	release := make(chan struct{})
	done := make(chan struct{})
	go func() {
		defer close(done)
		_, _ = getAllPositionsCached(context.Background(), func(context.Context) ([]*futures.PositionRisk, error) {
			close(started)
			<-release
			return []*futures.PositionRisk{{Symbol: "BTCUSDT", PositionAmt: "old"}}, nil
		})
	}()
	<-started

	InvalidateAccountReadCache()
	close(release)
	<-done

	var calls atomic.Int32
	rows, err := getAllPositionsCached(context.Background(), func(context.Context) ([]*futures.PositionRisk, error) {
		calls.Add(1)
		return []*futures.PositionRisk{{Symbol: "BTCUSDT", PositionAmt: "new"}}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if calls.Load() != 1 || len(rows) != 1 || rows[0].PositionAmt != "new" {
		t.Fatalf("stale in-flight load repopulated cache: calls=%d rows=%+v", calls.Load(), rows)
	}
}
