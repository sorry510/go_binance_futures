package binance

import (
	"errors"
	"strconv"
	"testing"
	"time"

	"github.com/adshao/go-binance/v2/common"
)

func TestFuturesKlineRequestWeight(t *testing.T) {
	tests := []struct {
		limit int
		want  int
	}{
		{1, 1},
		{99, 1},
		{100, 2},
		{150, 2},
		{499, 2},
		{500, 5},
		{1000, 5},
		{1001, 10},
		{1500, 10},
	}
	for _, tt := range tests {
		if got := futuresKlineRequestWeight(tt.limit); got != tt.want {
			t.Fatalf("limit=%d weight=%d want=%d", tt.limit, got, tt.want)
		}
	}
}

func TestLiveKlineCacheTTL(t *testing.T) {
	tests := map[string]time.Duration{
		"1m":  time.Second,
		"5m":  2 * time.Second,
		"30m": 5 * time.Second,
		"1h":  10 * time.Second,
		"4h":  10 * time.Second,
	}
	for interval, want := range tests {
		if got := liveKlineCacheTTL(interval); got != want {
			t.Fatalf("interval=%s ttl=%s want=%s", interval, got, want)
		}
	}
}

func TestNoteFuturesMarketDataErrorUsesBanUntil(t *testing.T) {
	// Prevent the production alert goroutine from sending external notifications
	// while this unit test exercises cooldown parsing.
	futuresRateLimitAlertMu.Lock()
	oldAlertActive := futuresRateLimitAlertActive
	futuresRateLimitAlertActive = true
	futuresRateLimitAlertMu.Unlock()
	defer func() {
		futuresRateLimitAlertMu.Lock()
		futuresRateLimitAlertActive = oldAlertActive
		futuresRateLimitAlertMu.Unlock()
	}()

	futuresMarketDataMu.Lock()
	oldCooldown := futuresMarketDataCooldownUntil
	futuresMarketDataCooldownUntil = time.Time{}
	futuresMarketDataMu.Unlock()
	defer func() {
		futuresMarketDataMu.Lock()
		futuresMarketDataCooldownUntil = oldCooldown
		futuresMarketDataMu.Unlock()
	}()

	banUntil := time.Now().Add(3 * time.Minute).Truncate(time.Millisecond)
	apiErr := &common.APIError{
		Code:    -1003,
		Message: "Way too many requests; IP(127.0.0.1) banned until " + strconv.FormatInt(banUntil.UnixMilli(), 10) + ".",
	}
	noteFuturesAPIError(apiErr)

	futuresMarketDataMu.Lock()
	got := futuresMarketDataCooldownUntil
	futuresMarketDataMu.Unlock()
	want := banUntil.Add(2 * time.Second)
	if got.Before(want) {
		t.Fatalf("cooldown=%s want >=%s", got, want)
	}

	before := got
	noteFuturesAPIError(errors.New("ordinary error"))
	futuresMarketDataMu.Lock()
	after := futuresMarketDataCooldownUntil
	futuresMarketDataMu.Unlock()
	if !after.Equal(before) {
		t.Fatalf("ordinary error changed cooldown: before=%s after=%s", before, after)
	}
}

func TestRunFuturesRateLimitAlertSequenceSendsExactlyThree(t *testing.T) {
	var got []int
	runFuturesRateLimitAlertSequence(3, time.Millisecond, func(sequence int) {
		got = append(got, sequence)
	})

	want := []int{1, 2, 3}
	if len(got) != len(want) {
		t.Fatalf("send count=%d want=%d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("sequence[%d]=%d want=%d", i, got[i], want[i])
		}
	}
}
