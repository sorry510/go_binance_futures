package feature

import (
	"go_binance_futures/models"
	"strings"
	"testing"
)

func TestEmptySmartSelectionExplainsDisabledSymbols(t *testing.T) {
	all := []*models.Symbols{
		{Symbol: "BTCUSDT", Type: "USDT", Enable: 0},
		{Symbol: "ETHUSDT", Type: "USDT", Enable: 0},
	}
	description, _ := describeEmptySmartSelection(all)
	if !strings.Contains(description, "enable=0") {
		t.Fatalf("missing diagnostic: %s", description)
	}
	all[0].Enable = 1
	description, _ = describeEmptySmartSelection(all)
	if strings.Contains(description, "all futures symbols") {
		t.Fatalf("incorrect all-disabled diagnosis: %s", description)
	}
}
func TestEmptySmartSelectionDoesNotEnableSymbols(t *testing.T) {
	all := []*models.Symbols{{Symbol: "BTCUSDT", Type: "USDT", Enable: 0}}
	_, _ = describeEmptySmartSelection(all)
	if all[0].Enable != 0 {
		t.Fatal("diagnosis must not change trade permission")
	}
}
