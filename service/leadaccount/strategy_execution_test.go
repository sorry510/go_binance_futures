package leadaccount

import (
	"context"
	"errors"
	"testing"

	"github.com/adshao/go-binance/v2/futures"
	binance "go_binance_futures/feature/api/binance"
	"go_binance_futures/models"
	"go_binance_futures/service/futuresownership"
)

func TestLead032StrategyRiskInputMustBeExact(t *testing.T) {
	coin := &models.Symbols{
		Symbol: "BTCUSDT", Type: "USDT", Leverage: 4,
		MarginType: "CROSSED", Usdt: "25",
	}
	check, err := leadStrategyRiskCheck(coin, "LONG", 1, 100, "MARKET")
	if err != nil || check.AccountID != binance.LeadAccountID ||
		check.Symbol != "BTCUSDT" || check.Leverage != 4 ||
		check.ConfiguredUSDT != 25 || check.OrderType != "MARKET" ||
		check.Quantity != 1 || check.ReferencePrice != 100 {
		t.Fatalf("Lead source risk conversion mismatch: %+v %v", check, err)
	}
	for _, tc := range []struct {
		name, side, orderType string
		coin                  *models.Symbols
		qty, price            float64
	}{
		{"nil_coin", "LONG", "MARKET", nil, 1, 100},
		{"wrong_side", "BOTH", "MARKET", coin, 1, 100},
		{"wrong_type", "LONG", "STOP_MARKET", coin, 1, 100},
		{"zero_quantity", "LONG", "MARKET", coin, 0, 100},
		{"zero_price", "LONG", "MARKET", coin, 1, 0},
		{"empty_symbol", "LONG", "MARKET", &models.Symbols{Usdt: "25", Leverage: 4, MarginType: "CROSSED"}, 1, 100},
		{"bad_leverage", "LONG", "MARKET", &models.Symbols{Symbol: "BTCUSDT", Usdt: "25", Leverage: 0, MarginType: "CROSSED"}, 1, 100},
		{"unknown_usdt", "LONG", "MARKET", &models.Symbols{Symbol: "BTCUSDT", Usdt: "abc", Leverage: 4, MarginType: "CROSSED"}, 1, 100},
		{"negative_usdt", "LONG", "MARKET", &models.Symbols{Symbol: "BTCUSDT", Usdt: "-1", Leverage: 4, MarginType: "CROSSED"}, 1, 100},
		{"missing_margin", "LONG", "MARKET", &models.Symbols{Symbol: "BTCUSDT", Usdt: "25", Leverage: 4}, 1, 100},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := leadStrategyRiskCheck(tc.coin, tc.side, tc.qty, tc.price, tc.orderType); !errors.Is(err, ErrLeadStrategyInput) {
				t.Fatalf("invalid strategy risk input accepted: %v", err)
			}
		})
	}
}

func TestLead032RealExecutorIdentityAndDurableGateStayLocked(t *testing.T) {
	client, err := binance.NewAccountClient(binance.LeadAccountID, futures.NewClient("lead032-key", "secret"))
	if err != nil {
		t.Fatal(err)
	}
	cache, err := binance.NewLeadSymbolCache(client, nil)
	if err != nil {
		t.Fatal(err)
	}
	a, err := NewLeadExecutionAdapter(client, cache)
	if err != nil {
		t.Fatal(err)
	}
	foreign, err := binance.NewAccountClient(binance.LeadAccountID, futures.NewClient("lead032-other", "secret"))
	if err != nil {
		t.Fatal(err)
	}
	main, err := binance.NewAccountClient(binance.MainAccountID, futures.NewClient("main-key", "secret"))
	if err != nil {
		t.Fatal(err)
	}
	if !a.BoundTo(client) || a.BoundTo(foreign) || a.BoundTo(main) || a.BoundTo(nil) {
		t.Fatal("Lead executor bound to wrong credentials")
	}
	coin := &models.Symbols{Symbol: "BTCUSDT", Leverage: 4, Usdt: "25", MarginType: "CROSSED"}
	if err := a.PreflightStrategyOpen(context.Background(), coin, "LONG", 1, 100, "MARKET"); !errors.Is(err, ErrLeadOpenBlocked) {
		t.Fatalf("Lead preflight bypassed risk/durable gate: %v", err)
	}
	request := futuresownership.OrderRequest{
		Owner: futuresownership.OwnerAutoStrategy, Symbol: "BTCUSDT", PositionSide: "LONG",
		Side: "BUY", OrderType: "MARKET", Intent: futuresownership.IntentOpen,
		Quantity: 1, SourceRef: "auto_strategy:BTCUSDT",
	}
	if _, err := a.ExecuteStrategyOpen(context.Background(), request, coin, 100); !errors.Is(err, ErrLeadOpenBlocked) {
		t.Fatalf("Lead actual order adapter bypassed hard gate: %v", err)
	}
	request.Symbol = "ETHUSDT"
	if _, err := a.ExecuteStrategyOpen(context.Background(), request, coin, 100); !errors.Is(err, ErrLeadStrategyInput) {
		t.Fatalf("mismatched submitted symbol allowed: %v", err)
	}
	request.Symbol = "BTCUSDT"
	request.Owner = "manual"
	if _, err := a.ExecuteStrategyOpen(context.Background(), request, coin, 100); !errors.Is(err, ErrLeadStrategyInput) {
		t.Fatalf("manual owner accepted: %v", err)
	}
}
