package feature

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/adshao/go-binance/v2/futures"
	"go_binance_futures/feature/api/binance"
	"go_binance_futures/feature/strategy"
	"go_binance_futures/models"
	"go_binance_futures/notify"
	"go_binance_futures/types"
)

func uf2EntryConfigFixture() (*models.Symbols, *models.Config) {
	return &models.Symbols{
			ID: 10, Symbol: "BTCUSDT", Enable: 1, Leverage: 4, MarginType: "CROSSED",
			Usdt: "100", Profit: "8", Loss: "6", TickSize: "0.1", StepSize: "0.01",
			Strategy: "[]", Technology: "{}", Type: "USDT", StrategyTemplateID: 2,
		}, &models.Config{
			FutureEnable: 1, FutureAllowLong: 1, FutureAllowShort: 1,
			FutureMaxCount: 5, LossMaxCount: 5, FutureOrderType: "MARKET",
		}
}
func TestUF2MainConfigureFailureFailsClosedAndNeverReadsStaleSnapshot(t *testing.T) {
	for _, scenario := range []string{"margin_failed", "leverage_failed", "api_429", "timeout", "permission_denied"} {
		t.Run(scenario, func(t *testing.T) {
			coin, cfg := uf2EntryConfigFixture()
			sentinel := errors.New(scenario)
			configCalls, dbReads := 0, 0
			err := ensureMainOpenConfigurationWith(context.Background(), coin, cfg,
				func(context.Context, *models.Symbols) error { configCalls++; return sentinel },
				func(context.Context, string) (*models.Symbols, *models.Config, error) {
					dbReads++
					return coin, cfg, nil
				})
			if !errors.Is(err, sentinel) || configCalls != 1 || dbReads != 0 {
				t.Fatalf("ignored Binance %s error or read a stale state: config=%d reads=%d err=%v", scenario, configCalls, dbReads, err)
			}
		})
	}
}
func TestUF2MainFinalConfigSnapshotRejectsAllTradeCriticalEdits(t *testing.T) {
	variants := []struct {
		name   string
		modify func(*models.Symbols, *models.Config)
	}{
		{"global_OFF", func(_ *models.Symbols, c *models.Config) { c.FutureEnable = 0 }},
		{"symbol_OFF", func(s *models.Symbols, _ *models.Config) { s.Enable = 0 }},
		{"margin_type", func(s *models.Symbols, _ *models.Config) { s.MarginType = "ISOLATED" }},
		{"leverage", func(s *models.Symbols, _ *models.Config) { s.Leverage = 5 }},
		{"budget", func(s *models.Symbols, _ *models.Config) { s.Usdt = "150" }},
		{"tick", func(s *models.Symbols, _ *models.Config) { s.TickSize = "0.01" }},
		{"step", func(s *models.Symbols, _ *models.Config) { s.StepSize = "0.001" }},
		{"strategy", func(s *models.Symbols, _ *models.Config) { s.Strategy = `[{"name":"edited"}]` }},
		{"technology", func(s *models.Symbols, _ *models.Config) { s.Technology = `{"ma":[]}` }},
		{"profit", func(s *models.Symbols, _ *models.Config) { s.Profit = "9" }},
		{"loss", func(s *models.Symbols, _ *models.Config) { s.Loss = "7" }},
		{"type", func(s *models.Symbols, _ *models.Config) { s.Type = "USDC" }},
		{"kline", func(s *models.Symbols, _ *models.Config) { s.KlineInterval = "5m" }},
		{"template", func(s *models.Symbols, _ *models.Config) { s.StrategyTemplateID = 3 }},
		{"symbol_id", func(s *models.Symbols, _ *models.Config) { s.ID = 11 }},
		{"long_OFF", func(_ *models.Symbols, c *models.Config) { c.FutureAllowLong = 0 }},
		{"short_OFF", func(_ *models.Symbols, c *models.Config) { c.FutureAllowShort = 0 }},
		{"order_type", func(_ *models.Symbols, c *models.Config) { c.FutureOrderType = "LIMIT" }},
		{"max_positions", func(_ *models.Symbols, c *models.Config) { c.FutureMaxCount = 4 }},
		{"max_loss_positions", func(_ *models.Symbols, c *models.Config) { c.LossMaxCount = 4 }},
	}
	for _, scenario := range variants {
		t.Run(scenario.name, func(t *testing.T) {
			original, originalConfig := uf2EntryConfigFixture()
			latest := *original
			latestConfig := *originalConfig
			invoked := 0
			err := ensureMainOpenConfigurationWith(context.Background(), original, originalConfig,
				func(_ context.Context, _ *models.Symbols) error {
					invoked++
					scenario.modify(&latest, &latestConfig)
					return nil
				},
				func(_ context.Context, symbol string) (*models.Symbols, *models.Config, error) {
					if symbol != "BTCUSDT" {
						t.Fatal("read wrong Main contract")
					}
					return &latest, &latestConfig, nil
				})
			if err == nil || invoked != 1 {
				t.Fatalf("configuration edited during Binance request but accepted: %v calls=%d", err, invoked)
			}
		})
	}
}
func TestUF2MainConfigPositiveCacheOrValidMarketUpdatesStillAllow(t *testing.T) {
	coin, cfg := uf2EntryConfigFixture()
	latest := *coin
	latest.Close = "55123"
	latest.UpdateTime = time.Now().UnixMilli()
	latest.QuoteVolume = 9000000
	current := *cfg
	configCalls, reads := 0, 0
	err := ensureMainOpenConfigurationWith(context.Background(), coin, cfg,
		func(context.Context, *models.Symbols) error { configCalls++; return nil },
		func(context.Context, string) (*models.Symbols, *models.Config, error) {
			reads++
			return &latest, &current, nil
		})
	if err != nil || configCalls != 1 || reads != 1 {
		t.Fatalf("valid success or WS price refresh rejected: %v config=%d reads=%d", err, configCalls, reads)
	}
}
func TestUF2MainInvalidConfigStopsBeforeRPC(t *testing.T) {
	for _, scenario := range []struct {
		name string
		edit func(*models.Symbols, *models.Config)
	}{
		{"global_off", func(_ *models.Symbols, c *models.Config) { c.FutureEnable = 0 }},
		{"symbol_off", func(s *models.Symbols, _ *models.Config) { s.Enable = 0 }},
		{"invalid_margin", func(s *models.Symbols, _ *models.Config) { s.MarginType = "BAD" }},
		{"invalid_leverage", func(s *models.Symbols, _ *models.Config) { s.Leverage = 0 }},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			coin, cfg := uf2EntryConfigFixture()
			scenario.edit(coin, cfg)
			calls := 0
			err := ensureMainOpenConfigurationWith(context.Background(), coin, cfg,
				func(context.Context, *models.Symbols) error { calls++; return nil },
				func(context.Context, string) (*models.Symbols, *models.Config, error) { calls++; return coin, cfg, nil })
			if err == nil || calls != 0 {
				t.Fatalf("invalid preflight sent Binance RPC: %v calls=%d", err, calls)
			}
		})
	}
}
func TestUF2MainConfigSnapshotUnavailableBlocks(t *testing.T) {
	coin, cfg := uf2EntryConfigFixture()
	sentinel := errors.New("local database unavailable")
	err := ensureMainOpenConfigurationWith(context.Background(), coin, cfg,
		func(context.Context, *models.Symbols) error { return nil },
		func(context.Context, string) (*models.Symbols, *models.Config, error) { return nil, nil, sentinel })
	if !errors.Is(err, sentinel) {
		t.Fatalf("Main ignored failed fresh symbol read: %v", err)
	}
}
func TestUF2UpdateSymbolTradeInfoInvalidModeAndLeverageNoDAPI(t *testing.T) {
	for _, margin := range []string{"", "INVALID", "CROSS", "ISOLATED "} {
		if err := UpdateSymbolTradeInfoContext(context.Background(), &models.Symbols{Symbol: "BTCUSDT", Leverage: 4, MarginType: margin}); err == nil {
			t.Fatalf("invalid margin mode %q treated as ISOLATED", margin)
		}
	}
	for _, leverage := range []int64{0, -1} {
		if err := UpdateSymbolTradeInfoContext(context.Background(), &models.Symbols{Symbol: "BTCUSDT", Leverage: leverage, MarginType: "ISOLATED"}); err == nil {
			t.Fatalf("leverage %d allowed real Binance request", leverage)
		}
	}
	if err := UpdateSymbolTradeInfoContext(context.Background(), nil); err == nil {
		t.Fatal("nil Main symbol allowed")
	}
}
func uf2MainMockOpenRunner(side string) (*accountTradeRunner, *int, *int, *int) {
	coin, cfg := uf2EntryConfigFixture()
	opens, configs, closes := new(int), new(int), new(int)
	r := &accountTradeRunner{
		AccountID: binance.MainAccountID, Config: cfg, AllowNewOpens: true,
		LoadSymbols: func() ([]*models.Symbols, error) { return []*models.Symbols{coin}, nil },
		SelectCoins: func(*models.Config, []*models.Symbols) ([]*models.Symbols, error) {
			return []*models.Symbols{coin}, nil
		},
		ReadPositions:  func(context.Context) ([]types.FuturesPosition, error) { return nil, nil },
		ReadOpenOrders: func(context.Context) ([]types.FuturesOrder, error) { return nil, nil },
		SyncPositions:  func([]types.FuturesPosition) ([]ownedTradePosition, error) { return nil, nil },
		CancelExpired:  func(int64) {},
		EvaluateEntry: func(*models.Symbols, []types.FuturesPosition) strategy.OpenResult {
			return strategy.OpenResult{CanLong: side == "LONG", CanShort: side == "SHORT"}
		},
		Depth: func(context.Context, string, int) (float64, float64, error) { return 100, 100, nil },
		EnsureConfig: func(context.Context, *models.Symbols) error {
			*configs++
			return fmt.Errorf("Main futures %s RPC denied", side)
		},
		SubmitOpen: func(*tradeCycleAccountSnapshot, string, float64, float64, futures.SideType, futures.PositionSideType, futures.OrderType, string) (*futures.CreateOrderResponse, error) {
			*opens++
			return &futures.CreateOrderResponse{OrderID: 100}, nil
		},
		SubmitClose: func(string, string, string, float64, futures.PositionSideType) (*futures.CreateOrderResponse, error) {
			*closes++
			return &futures.CreateOrderResponse{OrderID: 200}, nil
		},
		NotifyOpen:   func(notify.FuturesOrderParams) {},
		NotifyClose:  func(notify.FuturesOrderParams) {},
		EvaluateExit: func(strategy.CloseParams, float64, float64) tradeExitReason { return tradeExitHold },
		Sleep:        func(time.Duration) {},
	}
	return r, opens, configs, closes
}
func TestUF2MainLongShortRPCFailureNeverSubmitsOpen(t *testing.T) {
	for _, side := range []string{"LONG", "SHORT"} {
		t.Run(side, func(t *testing.T) {
			r, opens, configs, closes := uf2MainMockOpenRunner(side)
			runAccountTradeCycle(r)
			if *configs != 1 || *opens != 0 || *closes != 0 {
				t.Fatalf("%s ignored failed config RPC: configure=%d open=%d close=%d", side, *configs, *opens, *closes)
			}
		})
	}
}
func TestUF2FailedEntryConfigDoesNotBlockManagedExit(t *testing.T) {
	r, opens, closes, selected, _ := uf1MockMainRunner(t, "LONG", true)
	r.EnsureConfig = func(context.Context, *models.Symbols) error { return errors.New("Binance margin mode unavailable") }
	runAccountTradeCycle(r)
	if *closes != 1 || *opens != 0 || *selected != 1 {
		t.Fatalf("Main Managed CLOSE incorrectly depends on successful OPEN config: close=%d open=%d scan=%d", *closes, *opens, *selected)
	}
}
func TestUF2MainNoBlindSuccessWithCanceledContext(t *testing.T) {
	coin, cfg := uf2EntryConfigFixture()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	calls := 0
	err := ensureMainOpenConfigurationWith(ctx, coin, cfg,
		func(context.Context, *models.Symbols) error { calls++; return nil },
		func(context.Context, string) (*models.Symbols, *models.Config, error) { calls++; return coin, cfg, nil })
	if !errors.Is(err, context.Canceled) || calls != 0 {
		t.Fatalf("canceled Main preflight contacted Binance or DB: %v calls=%d", err, calls)
	}
}
func TestUF2MainErrorMessagesStayScoped(t *testing.T) {
	coin, cfg := uf2EntryConfigFixture()
	err := ensureMainOpenConfigurationWith(context.Background(), coin, cfg,
		func(context.Context, *models.Symbols) error { return errors.New("leverage change denied") },
		func(context.Context, string) (*models.Symbols, *models.Config, error) { return coin, cfg, nil })
	if err == nil || !strings.Contains(err.Error(), "Main futures") {
		t.Fatalf("Main configuration error missing context: %v", err)
	}
}

// Ensure that the actual Main runner, not just test-injected helpers,
// propagates a local preflight failure to runAccountTradeCycle.
func TestUF2ProductionMainRunnerDoesNotDiscardPreflightError(t *testing.T) {
	coin, cfg := uf2EntryConfigFixture()
	coin.MarginType = "INVALID"
	runner := mainTradeRunner(cfg)
	if err := runner.EnsureConfig(context.Background(), coin); err == nil {
		t.Fatal("real Main EnsureConfig ignored invalid margin type")
	}
	coin.MarginType = "CROSSED"
	coin.Leverage = 0
	if err := runner.EnsureConfig(context.Background(), coin); err == nil {
		t.Fatal("real Main EnsureConfig ignored invalid leverage")
	}
}
