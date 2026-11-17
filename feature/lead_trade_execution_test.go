package feature

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/adshao/go-binance/v2/futures"
	binance "go_binance_futures/feature/api/binance"
	"go_binance_futures/feature/strategy"
	"go_binance_futures/models"
	"go_binance_futures/service/futuresownership"
	"go_binance_futures/service/leadaccount"
	"go_binance_futures/types"
)

type stage032GatedPort struct {
	account                         *binance.AccountClient
	preflights, opens, closes       int
	lastRef                         float64
	cancels                         int
	lastOpen, lastClose             futuresownership.OrderRequest
	openRequests                    []futuresownership.OrderRequest
	preflightErr, openErr, closeErr error
}

func (p *stage032GatedPort) CancelExpiredStrategyOpens(_ context.Context, timeout int64) error {
	if timeout > 0 {
		p.cancels++
	}
	return nil
}

func (p *stage032GatedPort) BoundTo(account *binance.AccountClient) bool {
	return p != nil && p.account == account
}
func (p *stage032GatedPort) PreflightStrategyOpen(_ context.Context, _ *models.Symbols, _ string, _, price float64, _ string) error {
	p.preflights++
	p.lastRef = price
	return p.preflightErr
}
func (p *stage032GatedPort) ExecuteStrategyOpen(_ context.Context, req futuresownership.OrderRequest, _ *models.Symbols, reference float64) (futuresownership.ExchangeOrder, error) {
	p.opens++
	p.lastOpen = req
	p.openRequests = append(p.openRequests, req)
	p.lastRef = reference
	return futuresownership.ExchangeOrder{ExchangeOrderID: "12345", ClientOrderID: "lead_test_open", Status: "NEW"}, p.openErr
}
func (p *stage032GatedPort) ExecuteManagedClose(_ context.Context, req futuresownership.OrderRequest) (futuresownership.ExchangeOrder, error) {
	p.closes++
	p.lastClose = req
	return futuresownership.ExchangeOrder{ExchangeOrderID: "23456", ClientOrderID: "lead_test_close", Status: "NEW"}, p.closeErr
}

func stage032Hooks(t *testing.T, orderType string) (*accountTradeRunner, *stage032GatedPort, *models.Symbols, *leadTradeIO) {
	t.Helper()
	account, err := binance.NewAccountClient(binance.LeadAccountID, futures.NewClient("lead-fake", "test-secret"))
	if err != nil {
		t.Fatal(err)
	}
	symbols, err := binance.NewLeadSymbolCache(account, func(context.Context) ([]binance.LeadTradingSymbol, error) {
		return []binance.LeadTradingSymbol{{Symbol: "BTCUSDT", QuoteAsset: "USDT"}}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := symbols.Refresh(context.Background()); err != nil {
		t.Fatal(err)
	}
	io := &leadTradeIO{account: account, ownership: futuresownership.Service{AccountID: "lead"}}
	port := &stage032GatedPort{account: account}
	r := &accountTradeRunner{AccountID: binance.LeadAccountID, Config: &models.Config{FutureOrderType: orderType},
		LeadSymbols: symbols, AllowNewOpens: true}
	if err := bindLeadOrderHooks(r, io, port); err != nil {
		t.Fatal(err)
	}
	coin := &models.Symbols{Symbol: "BTCUSDT", Leverage: 4, MarginType: "CROSSED", Usdt: "50"}
	return r, port, coin, io
}

func TestLead032MarketPreflightSingleUseAndSourceRef(t *testing.T) {
	r, port, coin, _ := stage032Hooks(t, "MARKET")
	ctx := context.Background()
	if err := r.PreflightOpen(ctx, coin, futures.PositionSideTypeLong, 2, 100); err != nil {
		t.Fatal(err)
	}
	snapshot := newTradeCycleAccountSnapshot(nil, nil)
	got, err := r.SubmitOpen(snapshot, "BTCUSDT", 2, 0, futures.SideTypeBuy, futures.PositionSideTypeLong, futures.OrderTypeMarket, "strategy_hash")
	if err != nil || got.OrderID != 12345 || port.opens != 1 || port.lastRef != 100 ||
		port.lastOpen.Price != 0 || port.lastOpen.Quantity != 2 ||
		port.lastOpen.Owner != futuresownership.OwnerAutoStrategy ||
		port.lastOpen.Intent != futuresownership.IntentOpen ||
		port.lastOpen.PositionSide != "LONG" || port.lastOpen.Side != "BUY" ||
		port.lastOpen.SourceRef == "" {
		t.Fatalf("wrong bound market order: %+v %+v %v", port.lastOpen, got, err)
	}
	if snapshot.AccountSlotCount() != 1 {
		t.Fatalf("accepted order not reserved in current cycle: %d", snapshot.AccountSlotCount())
	}
	if _, err := r.SubmitOpen(snapshot, "BTCUSDT", 2, 0, futures.SideTypeBuy, futures.PositionSideTypeLong, futures.OrderTypeMarket, "strategy_hash"); !errors.Is(err, ErrLeadStrategyTicket) || port.opens != 1 {
		t.Fatalf("reused preflight: %v opens=%d", err, port.opens)
	}
}

func TestLead032PriceQuantitySideMustMatchExactPreflight(t *testing.T) {
	for _, tc := range []struct {
		name, typ       string
		quantity, price float64
		side            futures.SideType
		positionSide    futures.PositionSideType
	}{
		{"qty_changed", "MARKET", 3, 0, futures.SideTypeBuy, futures.PositionSideTypeLong},
		{"market_nonzero_price", "MARKET", 2, 100, futures.SideTypeBuy, futures.PositionSideTypeLong},
		{"wrong_side", "MARKET", 2, 0, futures.SideTypeSell, futures.PositionSideTypeLong},
		{"wrong_position_side", "MARKET", 2, 0, futures.SideTypeBuy, futures.PositionSideTypeShort},
		{"limit_changed_price", "LIMIT", 2, 101, futures.SideTypeBuy, futures.PositionSideTypeLong},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r, port, coin, _ := stage032Hooks(t, tc.typ)
			if err := r.PreflightOpen(context.Background(), coin, futures.PositionSideTypeLong, 2, 100); err != nil {
				t.Fatal(err)
			}
			if _, err := r.SubmitOpen(newTradeCycleAccountSnapshot(nil, nil), "BTCUSDT", tc.quantity, tc.price, tc.side, tc.positionSide, futures.OrderType(tc.typ), "rule"); !errors.Is(err, ErrLeadStrategyTicket) {
				t.Fatalf("mismatched request accepted: %v", err)
			}
			if port.opens != 0 {
				t.Fatal("mismatched order reached broker")
			}
		})
	}
	r, port, coin, _ := stage032Hooks(t, "LIMIT")
	if err := r.PreflightOpen(context.Background(), coin, futures.PositionSideTypeShort, 1, 88); err != nil {
		t.Fatal(err)
	}
	got, err := r.SubmitOpen(newTradeCycleAccountSnapshot(nil, nil), "BTCUSDT", 1, 88, futures.SideTypeSell, futures.PositionSideTypeShort, futures.OrderTypeLimit, "short_rule")
	if err != nil || got.OrderID != 12345 || port.lastRef != 88 || port.lastOpen.Side != "SELL" ||
		port.lastOpen.PositionSide != "SHORT" || port.lastOpen.Price != 88 {
		t.Fatalf("valid short limit rejected or altered: %+v %+v %v", port.lastOpen, got, err)
	}
}

func TestLead032FailedPreflightClearsTicketAndRejectedBrokerDoesNotReserve(t *testing.T) {
	r, port, coin, _ := stage032Hooks(t, "MARKET")
	if err := r.PreflightOpen(context.Background(), coin, futures.PositionSideTypeLong, 1, 100); err != nil {
		t.Fatal(err)
	}
	port.preflightErr = context.DeadlineExceeded
	if err := r.PreflightOpen(context.Background(), coin, futures.PositionSideTypeLong, 1, 100); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatal(err)
	}
	if _, err := r.SubmitOpen(newTradeCycleAccountSnapshot(nil, nil), "BTCUSDT", 1, 0, futures.SideTypeBuy, futures.PositionSideTypeLong, futures.OrderTypeMarket, "x"); !errors.Is(err, ErrLeadStrategyTicket) || port.opens != 0 {
		t.Fatalf("stale ticket survived rejected preflight: %v", err)
	}
	port.preflightErr = nil
	if err := r.PreflightOpen(context.Background(), coin, futures.PositionSideTypeLong, 1, 100); err != nil {
		t.Fatal(err)
	}
	port.openErr = context.DeadlineExceeded
	snapshot := newTradeCycleAccountSnapshot(nil, nil)
	if _, err := r.SubmitOpen(snapshot, "BTCUSDT", 1, 0, futures.SideTypeBuy, futures.PositionSideTypeLong, futures.OrderTypeMarket, "x"); !errors.Is(err, context.DeadlineExceeded) || snapshot.AccountSlotCount() != 0 {
		t.Fatalf("unconfirmed broker receipt counted as opening: %v", err)
	}
	if _, err := r.SubmitOpen(snapshot, "BTCUSDT", 1, 0, futures.SideTypeBuy, futures.PositionSideTypeLong, futures.OrderTypeMarket, "x"); !errors.Is(err, ErrLeadStrategyTicket) || port.opens != 1 {
		t.Fatalf("uncertain submit automatically retried: %v", err)
	}
}

func TestLead032PausedOrExpiredWhitelistDoesNotBlockManagedClose(t *testing.T) {
	r, port, _, _ := stage032Hooks(t, "MARKET")
	r.AllowNewOpens = false
	r.LeadSymbols.Invalidate()
	if err := r.PreflightOpen(context.Background(), &models.Symbols{Symbol: "BTCUSDT"}, futures.PositionSideTypeLong, 1, 100); !errors.Is(err, ErrLeadStrategyTicket) {
		t.Fatalf("new open allowed with paused or invalidated whitelist: %v", err)
	}
	resp, err := r.SubmitClose(futuresownership.OwnerAutoStrategy, "managed_cycle", "BTCUSDT", 1, futures.PositionSideTypeShort)
	if err != nil || resp.OrderID != 23456 || port.closes != 1 ||
		port.lastClose.Side != "BUY" || port.lastClose.PositionSide != "SHORT" ||
		port.lastClose.Owner != futuresownership.OwnerAutoStrategy ||
		port.lastClose.Intent != futuresownership.IntentClose || port.lastClose.OrderType != "MARKET" {
		t.Fatalf("paused managed short close blocked or changed: %+v %+v %v", port.lastClose, resp, err)
	}
	if _, err := r.SubmitClose("manual", "x", "BTCUSDT", 1, futures.PositionSideTypeLong); !errors.Is(err, ErrLeadStrategyTicket) {
		t.Fatalf("manual inventory may be sold by strategy: %v", err)
	}
	if _, err := r.SubmitClose(futuresownership.OwnerAutoStrategy, "x", "BTCUSDT", 0, futures.PositionSideTypeLong); !errors.Is(err, ErrLeadStrategyTicket) {
		t.Fatalf("zero close quantity reached broker: %v", err)
	}
	if port.closes != 1 {
		t.Fatal("unsafe close reached broker")
	}
}

func TestLead032PreflightInvalidInputClearsOldTicket(t *testing.T) {
	r, port, coin, _ := stage032Hooks(t, "MARKET")
	if err := r.PreflightOpen(context.Background(), coin, futures.PositionSideTypeLong, 1, 100); err != nil {
		t.Fatal(err)
	}
	if err := r.PreflightOpen(context.Background(), nil, futures.PositionSideTypeLong, 1, 100); !errors.Is(err, ErrLeadStrategyTicket) {
		t.Fatalf("invalid preflight unexpectedly accepted: %v", err)
	}
	if _, err := r.SubmitOpen(newTradeCycleAccountSnapshot(nil, nil), "BTCUSDT", 1, 0, futures.SideTypeBuy, futures.PositionSideTypeLong, futures.OrderTypeMarket, "x"); !errors.Is(err, ErrLeadStrategyTicket) {
		t.Fatalf("stale ticket survived invalid input: %v", err)
	}
	if port.opens != 0 {
		t.Fatal("invalid request reached broker")
	}
}

func TestLead032CanceledPreflightRevokesPreviousTicket(t *testing.T) {
	r, port, coin, _ := stage032Hooks(t, "MARKET")
	if err := r.PreflightOpen(context.Background(), coin, futures.PositionSideTypeLong, 1, 100); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := r.PreflightOpen(ctx, coin, futures.PositionSideTypeLong, 1, 100); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled preflight ignored: %v", err)
	}
	if _, err := r.SubmitOpen(newTradeCycleAccountSnapshot(nil, nil), "BTCUSDT", 1, 0, futures.SideTypeBuy, futures.PositionSideTypeLong, futures.OrderTypeMarket, "x"); !errors.Is(err, ErrLeadStrategyTicket) {
		t.Fatalf("stale preflight survived cancellation: %v", err)
	}
	if port.opens != 0 {
		t.Fatal("canceled request reached order broker")
	}
}

func TestLead032RejectedReceiptCannotBeReportedAsAccepted(t *testing.T) {
	for _, input := range []futuresownership.ExchangeOrder{
		{ExchangeOrderID: "123", ClientOrderID: "main_invalid", Status: "NEW"},
		{ExchangeOrderID: "123", ClientOrderID: "lead_abc", Status: "REJECTED"},
		{ExchangeOrderID: "0", ClientOrderID: "lead_abc", Status: "FILLED"},
		{ExchangeOrderID: "nan", ClientOrderID: "lead_abc", Status: "NEW"},
	} {
		if got, err := leadExchangeReceipt(input); !errors.Is(err, ErrLeadStrategyTicket) || got != nil {
			t.Fatalf("invalid exchange receipt accepted: %+v %+v %v", input, got, err)
		}
	}
}

func TestLead032BindingAndProductionExecutionRemainLocked(t *testing.T) {
	r, port, coin, io := stage032Hooks(t, "MARKET")
	if err := r.validate(); err == nil {
		t.Fatal("binding execution hooks accidentally enabled production Lead scheduling")
	}
	main, err := binance.NewAccountClient(binance.MainAccountID, futures.NewClient("main", "secret"))
	if err != nil {
		t.Fatal(err)
	}
	foreign := &leadTradeIO{account: main, ownership: futuresownership.Service{AccountID: "lead"}}
	if err := bindLeadOrderHooks(r, foreign, port); err == nil {
		t.Fatal("Main credentials bound to Lead broker")
	}
	other, err := binance.NewAccountClient(binance.LeadAccountID, futures.NewClient("other-lead", "other"))
	if err != nil {
		t.Fatal(err)
	}
	stale := &leadTradeIO{account: other, ownership: futuresownership.Service{AccountID: "lead"}}
	if err := bindLeadOrderHooks(r, stale, port); err == nil {
		t.Fatal("wrong Lead credentials bound to broker")
	}
	cache, err := binance.NewLeadSymbolCache(other, func(context.Context) ([]binance.LeadTradingSymbol, error) {
		return []binance.LeadTradingSymbol{{Symbol: "BTCUSDT", QuoteAsset: "USDT"}}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	r.LeadSymbols = cache
	if err := bindLeadOrderHooks(r, io, port); err == nil {
		t.Fatal("different Lead whitelist credentials accepted")
	}
	// Real Lead adapter is wired, but without explicit Stage 7 authorization
	// neither its preflight nor managed close can reach the exchange.
	r.LeadSymbols, err = binance.NewLeadSymbolCache(io.account, nil)
	if err != nil {
		t.Fatal(err)
	}
	adapter, err := leadaccount.NewLeadExecutionAdapter(io.account, r.LeadSymbols)
	if err != nil {
		t.Fatal(err)
	}
	if err := bindLeadOrderHooks(r, io, adapter); err != nil {
		t.Fatal(err)
	}
	if err := r.PreflightOpen(context.Background(), coin, futures.PositionSideTypeLong, 1, 100); err == nil {
		t.Fatal("real adapter preflight passed despite production hard gate")
	}
	if _, err := r.SubmitClose(futuresownership.OwnerAutoStrategy, "x", "BTCUSDT", 1, futures.PositionSideTypeLong); err == nil {
		t.Fatal("real adapter submitted managed close without authorization")
	}
}

// Exercise the existing shared auto-trade loop, NOT a new simulation runner.
// The Stage 3 permit exists only in _test.go; no production task is started.
func lead032SharedFixture(r *accountTradeRunner, coin *models.Symbols) {
	r.Mock = true
	r.leadMockPermit = stage3TestOnlyPermit{}
	r.Config.FutureEnable = 1
	r.Config.FutureAllowLong = 1
	r.Config.FutureAllowShort = 1
	r.Config.FutureMaxCount = 5
	r.Config.LossMaxCount = 5
	coin.TickSize = "0.1"
	coin.StepSize = "0.01"
	coin.Type = "USDT"
	coin.Profit = "8"
	coin.Loss = "6"
	r.LoadSymbols = func() ([]*models.Symbols, error) { return []*models.Symbols{coin}, nil }
	r.SelectCoins = func(*models.Config, []*models.Symbols) ([]*models.Symbols, error) {
		return []*models.Symbols{coin}, nil
	}
	r.ReadPositions = func(context.Context) ([]types.FuturesPosition, error) { return nil, nil }
	r.ReadOpenOrders = func(context.Context) ([]types.FuturesOrder, error) { return nil, nil }
	r.SyncPositions = func([]types.FuturesPosition) ([]ownedTradePosition, error) { return nil, nil }
	r.Depth = func(context.Context, string, int) (float64, float64, error) { return 100, 100, nil }
	r.Sleep = func(time.Duration) {}
	r.EvaluateEntry = func(*models.Symbols, []types.FuturesPosition) strategy.OpenResult {
		return strategy.OpenResult{CanLong: true, CanShort: true, LongStrategyHash: "rule-long", ShortStrategyHash: "rule-short"}
	}
}

func TestLead032FullSharedCycleUsesBoundAccountExecutorBothDirections(t *testing.T) {
	r, port, coin, _ := stage032Hooks(t, "MARKET")
	lead032SharedFixture(r, coin)
	if err := r.validate(); err != nil {
		t.Fatal(err)
	}
	runAccountTradeCycle(r)
	if port.preflights != 2 || port.opens != 2 || len(port.openRequests) != 2 {
		t.Fatalf("shared runner did not reach both gated directions: preflight=%d opens=%d", port.preflights, port.opens)
	}
	long := port.openRequests[0]
	short := port.openRequests[1]
	if long.Symbol != "BTCUSDT" || long.Side != "BUY" || long.PositionSide != "LONG" ||
		long.Quantity != 2 || long.Price != 0 || long.Intent != futuresownership.IntentOpen ||
		short.Symbol != "BTCUSDT" || short.Side != "SELL" || short.PositionSide != "SHORT" ||
		short.Quantity != 2 || short.Price != 0 || short.Intent != futuresownership.IntentOpen {
		t.Fatalf("shared LONG/SHORT order data mismatch: long=%+v short=%+v", long, short)
	}
}

func TestLead032FullSharedCyclePausedStillClosesBothManagedLegs(t *testing.T) {
	r, port, coin, _ := stage032Hooks(t, "MARKET")
	lead032SharedFixture(r, coin)
	r.AllowNewOpens = false
	r.LeadSymbols.Invalidate()
	positions := []types.FuturesPosition{
		{Symbol: "BTCUSDT", Side: "LONG", Amount: "1", Leverage: 4, MarkPrice: "100", UnrealizedProfit: "10"},
		{Symbol: "BTCUSDT", Side: "SHORT", Amount: "-1", Leverage: 4, MarkPrice: "100", UnrealizedProfit: "-10"},
	}
	r.ReadPositions = func(context.Context) ([]types.FuturesPosition, error) { return positions, nil }
	r.SyncPositions = func([]types.FuturesPosition) ([]ownedTradePosition, error) {
		return []ownedTradePosition{
			{Position: positions[0], Owner: futuresownership.OwnerAutoStrategy, SourceRef: "owned_long"},
			{Position: positions[1], Owner: futuresownership.OwnerAutoStrategy, SourceRef: "owned_short"},
		}, nil
	}
	r.EvaluateExit = func(strategy.CloseParams, float64, float64) tradeExitReason { return tradeExitProfit }
	if err := r.validate(); err != nil {
		t.Fatal(err)
	}
	runAccountTradeCycle(r)
	if port.opens != 0 || port.closes != 2 {
		t.Fatalf("paused shared runner dropped exits or allowed opens: opens=%d closes=%d", port.opens, port.closes)
	}
	if port.lastClose.SourceRef != "owned_short" || port.lastClose.Side != "BUY" ||
		port.lastClose.PositionSide != "SHORT" || port.lastClose.Quantity != 1 {
		t.Fatalf("managed short exit lost ownership: %+v", port.lastClose)
	}
}

func TestLead03TimeoutCancellationUsesBoundLeadPort(t *testing.T) {
	r, port, _, _ := stage032Hooks(t, "MARKET")
	r.CancelExpired(0)
	if port.cancels != 0 {
		t.Fatal("invalid timeout sent a cancellation")
	}
	r.CancelExpired(60)
	if port.cancels != 1 {
		t.Fatalf("Lead timeout did not reach account-bound executor: %d", port.cancels)
	}
}
