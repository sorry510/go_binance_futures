package spot

import (
	"testing"
	"time"

	sdk "github.com/adshao/go-binance/v2"
)

func TestS3WorkerDefaultOffNoPrivateExchangeIO(t *testing.T) {
	t.Setenv("SPOT_STRATEGY_LIVE_TRADING_APPROVAL", "")
	// No account DB, exchange API or ticker is available/needed here.
	RunRealSpotStrategies()
}

func TestS3ClosedCandleSelectsOnlyFinalizedBar(t *testing.T) {
	now := time.Now().UnixMilli()
	closed := &sdk.Kline{OpenTime: now - 120000, CloseTime: now - 60000, Close: "100", Open: "99"}
	forming := &sdk.Kline{OpenTime: now - 30000, CloseTime: now + 30000, Close: "200"}
	result, err := lastClosedSpotKline([]*sdk.Kline{closed, forming}, now)
	if err != nil || result != closed {
		t.Fatalf("unclosed Kline selected: %+v %v", result, err)
	}
	if _, err = lastClosedSpotKline([]*sdk.Kline{forming}, now); err == nil {
		t.Fatal("formed partial Kline treated as closed")
	}
	old := &sdk.Kline{OpenTime: now - 600000, CloseTime: now - 400000, Close: "50"}
	if _, err = lastClosedSpotKline([]*sdk.Kline{old}, now); err == nil {
		t.Fatal("old stale Kline permitted live signal")
	}
}

func TestS3F6WorkerPanicIsIsolated(t *testing.T) {
	didRun := false
	runSpotStrategySafely(func() { didRun = true; panic("sensitive-transport-failure") })
	if !didRun {
		t.Fatal("worker callback was not executed")
	}
	// If panic escaped, the Go test process would crash and never reach here.
}
