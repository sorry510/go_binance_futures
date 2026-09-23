package feature

import (
	"testing"

	"go_binance_futures/feature/strategy/coin"
)

func TestGetCoinStrategySmartLocalV2(t *testing.T) {
	strategy := GetCoinStrategy("smart_local_v2")
	if _, ok := strategy.(coin.SmartLocalV2); !ok {
		t.Fatalf("GetCoinStrategy(smart_local_v2) returned %T", strategy)
	}
}
