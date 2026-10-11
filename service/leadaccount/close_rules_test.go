package leadaccount

import (
	"context"
	"errors"
	"testing"

	"go_binance_futures/service/futuresownership"
)

func TestStage44CloseExchangeFilterCases(t *testing.T) {
	_, symbol, _, _ := stage43RuleCase()
	cases := []struct {
		name, orderType                string
		qty, price, trigger, cap, live float64
		blocked                        bool
	}{
		{"valid_market", "MARKET", 0.5, 0, 0, 1, 1, false},
		{"valid_limit", "LIMIT", 1, 100.50, 0, 1, 1, false},
		{"valid_stop", "STOP_MARKET", 1, 0, 95.50, 1, 1, false},
		{"valid_tp", "TAKE_PROFIT_MARKET", 1, 0, 105.50, 1, 1, false},
		{"qty_step", "MARKET", 0.501, 0, 0, 1, 1, true},
		{"qty_above_cap", "MARKET", 1.01, 0, 0, 1, 1.01, true},
		{"price_tick", "LIMIT", 1, 100.505, 0, 1, 1, true},
		{"stop_tick", "STOP_MARKET", 1, 0, 95.505, 1, 1, true},
		{"stop_missing", "STOP_MARKET", 1, 0, 0, 1, 1, true},
		{"market_with_price", "MARKET", 1, 1, 0, 1, 1, true},
		{"partial_too_small", "MARKET", 0.01, 0, 0, 1, 1, true},
		{"full_small_allowed", "MARKET", 0.01, 0, 0, 0.01, 0.01, false},
		{"managed_less_than_live_cannot_exempt", "MARKET", 0.01, 0, 0, 0.01, 1, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := futuresownership.OrderRequest{Symbol: "BTCUSDT", OrderType: tc.orderType, Quantity: tc.qty, Price: tc.price, StopPrice: tc.trigger}
			err := ValidateLeadCloseRules(req, symbol, 100, tc.cap, tc.live)
			if (err != nil) != tc.blocked {
				t.Fatalf("block=%v, err=%v", tc.blocked, err)
			}
		})
	}
	t.Run("missing_filters", func(t *testing.T) {
		symbol.Filters = nil
		if err := ValidateLeadCloseRules(futuresownership.OrderRequest{Symbol: "BTCUSDT", OrderType: "MARKET", Quantity: 1}, symbol, 100, 1, 1); err == nil {
			t.Fatal("missing filters allowed")
		}
	})
}

func TestStage44CloseCannotSendInvalidFiltersToBroker(t *testing.T) {
	a, broker, _ := stage43OwnedFixture(t, "LONG", 1)
	req := stage43CloseRequest("LONG", "LIMIT", 1)
	req.Price = 100.505
	if _, err := a.ExecuteManagedClose(context.Background(), req); !errors.Is(err, ErrLeadOpenBlocked) {
		t.Fatalf("invalid price accepted: %v", err)
	}
	req.OrderType = "STOP_MARKET"
	req.Price = 0
	req.StopPrice = 0
	if _, err := a.ExecuteManagedClose(context.Background(), req); !errors.Is(err, ErrLeadOpenBlocked) {
		t.Fatalf("missing stop accepted: %v", err)
	}
	if broker.submits != 0 || a.pendingReconcile {
		t.Fatalf("invalid close mutated state/submitted: submits=%d pending=%v", broker.submits, a.pendingReconcile)
	}
}
