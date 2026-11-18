package coinmquant

import (
	"strings"
	"testing"
)

func TestCM4InversePnLExamples(t *testing.T) {
	cases := []struct{ side, n, c, entry, exit, want string }{
		{"LONG", "10", "100", "50000", "55000", "0.001818181818181818"},
		{"LONG", "10", "100", "50000", "45000", "-0.002222222222222222"},
		{"SHORT", "10", "100", "50000", "45000", "0.002222222222222222"},
		{"SHORT", "10", "100", "50000", "55000", "-0.001818181818181818"},
		{"LONG", "3", "10", "100", "100", "0"},
		{"SHORT", "1", "100", "0.1", "0.2", "-500"},
		{"LONG", "1", "100", "0.1", "0.2", "500"},
		{"LONG", "1", "100", "50000", "50001", "0.000000039999200016"},
	}
	for _, tc := range cases {
		got, err := InversePnL(tc.side, tc.n, tc.c, tc.entry, tc.exit)
		if err != nil || got != tc.want {
			t.Errorf("inverse PnL %s %s*%s %s->%s = %s (want %s) err=%v",
				tc.side, tc.n, tc.c, tc.entry, tc.exit, got, tc.want, err)
		}
	}
}
func TestCM4InversePnLRejectInvalidInputs(t *testing.T) {
	for _, input := range [][5]string{
		{"BOTH", "1", "100", "50000", "55000"},
		{"LONG", "0", "100", "50000", "55000"},
		{"SHORT", "-2", "100", "50000", "55000"},
		{"LONG", "2", "0", "50000", "55000"},
		{"LONG", "1", "100", "0", "55000"},
		{"LONG", "1", "100", "50000", "0"},
		{"LONG", "1e9", "100", "50000", "55000"},
		{"LONG", "NaN", "100", "50000", "55000"},
		{"LONG", "1", "100", "50000", "abc"},
	} {
		if p, err := InversePnL(input[0], input[1], input[2], input[3], input[4]); err == nil {
			t.Errorf("accepted inverse PnL invalid input %+v output=%s", input, p)
		}
	}
}
func TestCM4StrictMarkAndSettlementConversion(t *testing.T) {
	for _, tc := range []struct {
		symbol, asset string
		ok            bool
	}{
		{"BTCUSD_PERP", "BTC", true}, {"ETHUSD_PERP", "ETH", true}, {"BTCUSD_PERP", "USDT", false},
		{"BTCUSD_261225", "BTC", false}, {"BTCUSDT", "BTC", false},
	} {
		if got := settledAssetForSymbol(tc.symbol, tc.asset); got != tc.ok {
			t.Errorf("%s %s match=%t", tc.symbol, tc.asset, got)
		}
	}
	if err := (Mark{Symbol: "BTCUSD_PERP", Price: "0", Time: 1}).Check("BTCUSD_PERP"); err == nil {
		t.Fatal("zero mark accepted")
	}
	if err := (Mark{Symbol: "ETHUSD_PERP", Price: "55000", Time: 1}).Check("BTCUSD_PERP"); err == nil {
		t.Fatal("cross-asset mark accepted")
	}
	if !strings.Contains((Mark{Symbol: "BTCUSD_261225", Price: "50000", Time: 1}).Check("BTCUSD_261225").Error(), "mark") {
		t.Fatal("delivery contract mark price must be rejected")
	}
}
