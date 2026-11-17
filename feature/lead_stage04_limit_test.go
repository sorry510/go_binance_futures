package feature

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/adshao/go-binance/v2/futures"
	binance "go_binance_futures/feature/api/binance"
	"go_binance_futures/models"
)

// The 10/30/60 checks test one specific cost property, not exchange latency:
// coin filtering and repeated hot-path whitelist lookups must not perform an
// additional SAPI call after the cache has been populated.
func TestLead04Whitelist10_30_60NoSAPIInSharedSelection(t *testing.T) {
	for _, count := range []int{10, 30, 60} {
		t.Run(fmt.Sprintf("%d", count), func(t *testing.T) {
			account, err := binance.NewAccountClient(binance.LeadAccountID,
				futures.NewClient("stage04-fake-lead", "stage04-fake-secret"))
			if err != nil {
				t.Fatal(err)
			}
			var sapiRequests atomic.Int32
			cache, err := binance.NewLeadSymbolCache(account,
				func(context.Context) ([]binance.LeadTradingSymbol, error) {
					sapiRequests.Add(1)
					whitelist := make([]binance.LeadTradingSymbol, count)
					for i := 0; i < count; i++ {
						whitelist[i] = binance.LeadTradingSymbol{
							Symbol: fmt.Sprintf("COIN%02dUSDT", i), QuoteAsset: "USDT",
						}
					}
					return whitelist, nil
				})
			if err != nil {
				t.Fatal(err)
			}
			if err := cache.Refresh(context.Background()); err != nil {
				t.Fatal(err)
			}
			all := make([]*models.Symbols, count+10)
			for i := range all {
				all[i] = &models.Symbols{Symbol: fmt.Sprintf("COIN%02dUSDT", i)}
			}
			var samples []time.Duration
			for scan := 0; scan < 100; scan++ {
				start := time.Now()
				allowed, err := cache.Snapshot()
				if err != nil {
					t.Fatal(err)
				}
				filtered := leadEligibleUniverse(all, allowed)
				samples = append(samples, time.Since(start))
				if len(filtered) != count {
					t.Fatalf("eligible coins = %d, want %d", len(filtered), count)
				}
				if err := cache.Refresh(context.Background()); err != nil {
					t.Fatal(err)
				}
			}
			if sapiRequests.Load() != 1 {
				t.Fatalf("100 hotpath selections triggered %d SAPI calls, expected 1", sapiRequests.Load())
			}
			// These are LOCAL deterministic CPU timings, not REST or
			// 2-second full strategy scan latency, and not production P95.
			t.Logf("local_whitelist_filter only: coins=%d rounds=100 sapi_requests=%d max_cpu_us=%d",
				count, sapiRequests.Load(), maxLead04Duration(samples).Microseconds())
		})
	}
}

func maxLead04Duration(samples []time.Duration) time.Duration {
	var max time.Duration
	for _, d := range samples {
		if d > max {
			max = d
		}
	}
	return max
}

func TestLead04NoMainCredentialFallbackOnCacheFailure(t *testing.T) {
	main, err := binance.NewAccountClient(binance.MainAccountID,
		futures.NewClient("main-fake", "secret"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := binance.NewLeadSymbolCache(main, nil); err == nil {
		t.Fatal("Main client accepted as Lead SAPI cache")
	}
}
