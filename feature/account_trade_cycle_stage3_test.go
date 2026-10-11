package feature

import (
	"context"
	"fmt"
	"go_binance_futures/feature/strategy/line"
	"reflect"
	"testing"
	"time"

	"go_binance_futures/feature/api/binance"
	"go_binance_futures/feature/strategy"
	"go_binance_futures/models"
	"go_binance_futures/notify"
	"go_binance_futures/service/binanceapiusage"
	"go_binance_futures/types"

	"github.com/adshao/go-binance/v2/futures"
)

// This type is intentionally compiled only into tests. Production code
// has no way to create a valid Stage 3 Lead runner, irrespective of Mock=true.
type stage3TestOnlyPermit struct{}

func (stage3TestOnlyPermit) stage3MockOnly() {}

// Build from explicit test hooks, never from mainTradeRunner or its real
// broker, history writer, WS position source or side-effecting API closures.
func leadMockTradeRunner(cfg *models.Config, symbols *binance.LeadSymbolCache, fake *accountTradeRunner) (*accountTradeRunner, error) {
	if cfg == nil || symbols == nil || fake == nil {
		return nil, fmt.Errorf("lead mock test configuration/hooks required")
	}
	if fake.AccountID != "" || fake.Mock || fake.leadMockPermit != nil {
		return nil, fmt.Errorf("Lead test fixtures cannot reuse an initialized account runner")
	}
	r := *fake
	r.AccountID = binance.LeadAccountID
	r.Config = cfg
	r.Mock = true
	r.leadMockPermit = stage3TestOnlyPermit{}
	r.LeadSymbols = symbols
	r.SelectCoins = func(c *models.Config, all []*models.Symbols) ([]*models.Symbols, error) {
		return selectLeadTradeCoins(c, all, symbols)
	}
	r.EvaluateEntry = func(coin *models.Symbols, positions []types.FuturesPosition) strategy.OpenResult {
		return (line.TradeLineCustom{}).GetCanLongOrShortWithPositions(strategy.OpenParams{Symbols: coin}, positions)
	}
	r.EvaluateExit = evaluateSharedTradeExit
	if err := r.validate(); err != nil {
		return nil, err
	}
	return &r, nil
}

func stage3MockRunner(t *testing.T, allowed []string) (*accountTradeRunner, *int, *int, *int) {
	t.Helper()
	client, err := binance.NewAccountClient(binance.LeadAccountID, futures.NewClient("fake", "fake"))
	if err != nil {
		t.Fatal(err)
	}
	cache, err := binance.NewLeadSymbolCache(client, func(context.Context) ([]binance.LeadTradingSymbol, error) {
		symbols := make([]binance.LeadTradingSymbol, 0, len(allowed))
		for _, sym := range allowed {
			symbols = append(symbols, binance.LeadTradingSymbol{Symbol: sym, QuoteAsset: "USDT"})
		}
		return symbols, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := cache.Refresh(context.Background()); err != nil {
		t.Fatal(err)
	}
	cfg := &models.Config{FutureEnable: 1, FutureMaxCount: 5, LossMaxCount: 5, FutureAllowLong: 1, FutureAllowShort: 1, FutureOrderType: "MARKET"}
	coins := []*models.Symbols{
		{Symbol: "BTCUSDT", TickSize: "0.1", StepSize: "0.01", Usdt: "100", Leverage: 4, Profit: "8", Loss: "6"},
		{Symbol: "ETHUSDT", TickSize: "0.1", StepSize: "0.01", Usdt: "100", Leverage: 4, Profit: "8", Loss: "6"},
	}
	openCount, closeCount, historyCount := new(int), new(int), new(int)
	fake := &accountTradeRunner{
		AllowNewOpens:  true,
		LoadSymbols:    func() ([]*models.Symbols, error) { return coins, nil },
		ReadPositions:  func(context.Context) ([]types.FuturesPosition, error) { return []types.FuturesPosition{}, nil },
		ReadOpenOrders: func(context.Context) ([]types.FuturesOrder, error) { return []types.FuturesOrder{}, nil },
		SyncPositions:  func([]types.FuturesPosition) ([]ownedTradePosition, error) { return nil, nil },
		CancelExpired:  func(int64) {},
		Depth:          func(context.Context, string, int) (float64, float64, error) { return 100, 100, nil },
		EnsureConfig:   func(context.Context, *models.Symbols) error { return nil },
		PreflightOpen:  func(context.Context, *models.Symbols, futures.PositionSideType, float64, float64) error { return nil },
		SubmitOpen: func(snap *tradeCycleAccountSnapshot, symbol string, quantity, price float64, side futures.SideType, pSide futures.PositionSideType, typ futures.OrderType, hash string) (*futures.CreateOrderResponse, error) {
			*openCount++
			order := &futures.CreateOrderResponse{OrderID: int64(*openCount)}
			snap.RecordPendingOpen(symbol, side, pSide, order.OrderID)
			return order, nil
		},
		SubmitClose: func(string, string, string, float64, futures.PositionSideType) (*futures.CreateOrderResponse, error) {
			*closeCount++
			return &futures.CreateOrderResponse{OrderID: 100}, nil
		},
		RecordOpen:  func(string, float64, string, string, int64, int64) { *historyCount++ },
		RecordClose: func(types.FuturesPosition, float64, float64, string, int64, *models.Config) { *historyCount++ },
		NotifyOpen:  func(notify.FuturesOrderParams) {},
		NotifyClose: func(notify.FuturesOrderParams) {},
		Sleep:       func(time.Duration) {},
	}
	r, err := leadMockTradeRunner(cfg, cache, fake)
	if err != nil {
		t.Fatal(err)
	}
	// Inject a static mock selector/evaluator, without Binance REST or database.
	// The production Lead selector has a dedicated whitelist-first regression.
	r.SelectCoins = func(*models.Config, []*models.Symbols) ([]*models.Symbols, error) { return coins, nil }
	r.EvaluateEntry = func(*models.Symbols, []types.FuturesPosition) strategy.OpenResult {
		return strategy.OpenResult{CanLong: true}
	}
	return r, openCount, closeCount, historyCount
}
func TestStage3LeadMockNeverOpensOutsideWhitelist(t *testing.T) {
	r, opens, closes, history := stage3MockRunner(t, []string{"BTCUSDT"})
	runAccountTradeCycle(r)
	if *opens != 1 || *closes != 0 || *history != 1 {
		t.Fatalf("opens=%d closes=%d history=%d", *opens, *closes, *history)
	}
	// Cached whitelist is required again immediately before submission.
	r.LeadSymbols.Invalidate()
	runAccountTradeCycle(r)
	if *opens != 1 {
		t.Fatalf("invalidated whitelist permitted open: %d", *opens)
	}
}
func TestStage3LeadPausedStillReconciles(t *testing.T) {
	r, opens, _, _ := stage3MockRunner(t, []string{"BTCUSDT"})
	r.AllowNewOpens = false
	syncs, cancels := 0, 0
	r.SyncPositions = func([]types.FuturesPosition) ([]ownedTradePosition, error) { syncs++; return nil, nil }
	r.CancelExpired = func(int64) { cancels++ }
	runAccountTradeCycle(r)
	if syncs != 1 || cancels != 1 || *opens != 0 {
		t.Fatalf("sync=%d cancel=%d opens=%d", syncs, cancels, *opens)
	}
}
func TestStage3LeadUsesOwnPositionsForOpeningSlots(t *testing.T) {
	r, opens, _, _ := stage3MockRunner(t, []string{"BTCUSDT", "ETHUSDT"})
	r.ReadPositions = func(context.Context) ([]types.FuturesPosition, error) {
		return []types.FuturesPosition{{Symbol: "BTCUSDT", Side: "LONG", Amount: "1"}}, nil
	}
	runAccountTradeCycle(r)
	if *opens != 1 {
		t.Fatalf("lead existing BTC slot failed to prevent duplicate opening: %d", *opens)
	}
}
func TestStage3LeadCannotUseIncompleteOrLiveRunner(t *testing.T) {
	if err := (&accountTradeRunner{AccountID: binance.LeadAccountID, Config: &models.Config{}}).validate(); err == nil {
		t.Fatal("lead runner missing mock/whitelist accepted")
	}
}

// Golden input fixture locks down the former Main StartTrade sequencing and
// MARKET/LIMIT price/quantity semantics through the shared runner. It never
// invokes the real Binance client or writes the database.
func TestStage3MainMarketEntryGolden(t *testing.T) {
	r, _, _, _ := stage3MockRunner(t, []string{"BTCUSDT"})
	r.AccountID = binance.MainAccountID
	r.Mock = false
	r.leadMockPermit = nil
	r.LeadSymbols = nil
	r.Config.FutureMaxCount = 3
	r.SelectCoins = func(_ *models.Config, all []*models.Symbols) ([]*models.Symbols, error) { return all[:1], nil }
	r.EvaluateEntry = func(*models.Symbols, []types.FuturesPosition) strategy.OpenResult {
		return strategy.OpenResult{CanLong: true, CanShort: true, LongStrategyHash: "long-hash", ShortStrategyHash: "short-hash"}
	}
	type intent struct {
		symbol, side, positionSide, orderType, hash string
		quantity, price                             float64
	}
	var actual []intent
	r.SubmitOpen = func(snapshot *tradeCycleAccountSnapshot, symbol string, qty, price float64, side futures.SideType, ps futures.PositionSideType, typ futures.OrderType, hash string) (*futures.CreateOrderResponse, error) {
		actual = append(actual, intent{symbol, string(side), string(ps), string(typ), hash, qty, price})
		id := int64(len(actual))
		snapshot.RecordPendingOpen(symbol, side, ps, id)
		return &futures.CreateOrderResponse{OrderID: id}, nil
	}
	var recorded []string
	r.RecordOpen = func(_ string, qty float64, avg, positionSide string, _ int64, _ int64) {
		recorded = append(recorded, positionSide+":"+avg)
		if qty != 4 {
			t.Errorf("quantity=%v want 4", qty)
		}
	}
	var sleeps []time.Duration
	r.Sleep = func(d time.Duration) { sleeps = append(sleeps, d) }
	runAccountTradeCycle(r)
	expected := []intent{
		{"BTCUSDT", "BUY", "LONG", "MARKET", "long-hash", 4, 0},
		{"BTCUSDT", "SELL", "SHORT", "MARKET", "short-hash", 4, 0},
	}
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("golden intents mismatch:\ngot=%+v\nwant=%+v", actual, expected)
	}
	if !reflect.DeepEqual(recorded, []string{"LONG:100.1", "SHORT:99.9"}) {
		t.Fatalf("golden market history price mismatch: %v", recorded)
	}
	if !reflect.DeepEqual(sleeps, []time.Duration{30 * time.Second}) {
		t.Fatalf("cooldown=%v", sleeps)
	}
}
func TestStage3LeadCloseDoesNotDependOnWhitelist(t *testing.T) {
	r, opens, closes, history := stage3MockRunner(t, []string{"BTCUSDT"})
	r.AllowNewOpens = false
	r.ReadPositions = func(context.Context) ([]types.FuturesPosition, error) {
		return []types.FuturesPosition{{Symbol: "BTCUSDT", Side: "LONG", Amount: "2", Leverage: 4, MarkPrice: "100", UnrealizedProfit: "-20"}}, nil
	}
	r.SyncPositions = func(positions []types.FuturesPosition) ([]ownedTradePosition, error) {
		return []ownedTradePosition{{Position: positions[0], Owner: "auto_strategy", SourceRef: "auto_strategy:BTCUSDT"}}, nil
	}
	r.EvaluateExit = func(strategy.CloseParams, float64, float64) tradeExitReason { return tradeExitAuto }
	r.LeadSymbols.Invalidate()
	runAccountTradeCycle(r)
	if *opens != 0 || *closes != 1 || *history != 1 {
		t.Fatalf("expired whitelist blocked managed close, opens=%d closes=%d history=%d", *opens, *closes, *history)
	}
}

// CODE_REVIEW_Stage3 PR-Stage3-1: the LIMIT request and history price must
// remain exactly the rounded depth price (no MARKET 1.0012/0.9988 adjustment).
func TestStage3MainLimitOrderGolden(t *testing.T) {
	r, _, _, _ := stage3MockRunner(t, []string{"BTCUSDT"})
	r.AccountID = binance.MainAccountID
	r.Mock = false
	r.leadMockPermit = nil
	r.LeadSymbols = nil
	r.Config.FutureOrderType = "LIMIT"
	r.Config.FutureMaxCount = 3
	r.SelectCoins = func(_ *models.Config, coins []*models.Symbols) ([]*models.Symbols, error) { return coins[:1], nil }
	r.EvaluateEntry = func(*models.Symbols, []types.FuturesPosition) strategy.OpenResult {
		return strategy.OpenResult{CanLong: true, CanShort: true, LongStrategyHash: "long-hash", ShortStrategyHash: "short-hash"}
	}
	type intent struct {
		symbol       string
		side         futures.SideType
		positionSide futures.PositionSideType
		orderType    futures.OrderType
		qty, price   float64
		hash         string
	}
	var got []intent
	r.SubmitOpen = func(snap *tradeCycleAccountSnapshot, symbol string, qty, price float64, side futures.SideType, pos futures.PositionSideType, kind futures.OrderType, hash string) (*futures.CreateOrderResponse, error) {
		got = append(got, intent{symbol, side, pos, kind, qty, price, hash})
		id := int64(len(got))
		snap.RecordPendingOpen(symbol, side, pos, id)
		return &futures.CreateOrderResponse{OrderID: id}, nil
	}
	var history []string
	r.RecordOpen = func(_ string, qty float64, price, pos string, _ int64, _ int64) {
		if qty != 4 {
			t.Errorf("quantity=%g", qty)
		}
		history = append(history, pos+":"+price)
	}
	var sleep []time.Duration
	r.Sleep = func(d time.Duration) { sleep = append(sleep, d) }
	runAccountTradeCycle(r)
	want := []intent{
		{"BTCUSDT", futures.SideTypeBuy, futures.PositionSideTypeLong, futures.OrderTypeLimit, 4, 100, "long-hash"},
		{"BTCUSDT", futures.SideTypeSell, futures.PositionSideTypeShort, futures.OrderTypeLimit, 4, 100, "short-hash"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("LIMIT trade drift\ngot=%+v\nwant=%+v", got, want)
	}
	if !reflect.DeepEqual(history, []string{"LONG:100", "SHORT:100"}) {
		t.Fatalf("LIMIT history price drift: %v", history)
	}
	if !reflect.DeepEqual(sleep, []time.Duration{30 * time.Second}) {
		t.Fatalf("LIMIT cooldown drift: %v", sleep)
	}
}

// CODE_REVIEW_Stage3 PR-Stage3-2: changing one direction flag must not
// suppress the other side. Both cases use a real shared trade-cycle branch.
func TestStage3MainDirectionSwitches(t *testing.T) {
	for _, tc := range []struct {
		name                  string
		allowLong, allowShort int
		want                  []futures.PositionSideType
	}{
		{"short_only", 0, 1, []futures.PositionSideType{futures.PositionSideTypeShort}},
		{"long_only", 1, 0, []futures.PositionSideType{futures.PositionSideTypeLong}},
		{"both_disabled", 0, 0, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r, _, _, _ := stage3MockRunner(t, []string{"BTCUSDT"})
			r.AccountID = binance.MainAccountID
			r.Mock = false
			r.leadMockPermit = nil
			r.LeadSymbols = nil
			r.Config.FutureAllowLong = tc.allowLong
			r.Config.FutureAllowShort = tc.allowShort
			r.SelectCoins = func(_ *models.Config, coins []*models.Symbols) ([]*models.Symbols, error) { return coins[:1], nil }
			r.EvaluateEntry = func(*models.Symbols, []types.FuturesPosition) strategy.OpenResult {
				return strategy.OpenResult{CanLong: true, CanShort: true}
			}
			var got []futures.PositionSideType
			r.SubmitOpen = func(snap *tradeCycleAccountSnapshot, sym string, qty, price float64, side futures.SideType, ps futures.PositionSideType, typ futures.OrderType, hash string) (*futures.CreateOrderResponse, error) {
				got = append(got, ps)
				id := int64(len(got))
				snap.RecordPendingOpen(sym, side, ps, id)
				return &futures.CreateOrderResponse{OrderID: id}, nil
			}
			runAccountTradeCycle(r)
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("direction gate got=%v want=%v", got, tc.want)
			}
		})
	}
}

// CODE_REVIEW_Stage3 PR-Stage3-3: when no Lead whitelist is ready, the real
// selector returns an empty non-error result, permitting exit/reconcile.
func TestStage3LeadUnavailableWhitelistDoesNotAbortExitCycle(t *testing.T) {
	r, opens, _, _ := stage3MockRunner(t, []string{"BTCUSDT"})
	r.SelectCoins = func(cfg *models.Config, all []*models.Symbols) ([]*models.Symbols, error) {
		return selectLeadTradeCoins(cfg, all, r.LeadSymbols)
	}
	r.LeadSymbols.Invalidate()
	selected, err := r.SelectCoins(r.Config, []*models.Symbols{{Symbol: "BTCUSDT"}})
	if err != nil || selected == nil || len(selected) != 0 {
		t.Fatalf("cache not ready: selected=%v err=%v", selected, err)
	}
	syncCount, cancelCount := 0, 0
	r.SyncPositions = func([]types.FuturesPosition) ([]ownedTradePosition, error) { syncCount++; return nil, nil }
	r.CancelExpired = func(int64) { cancelCount++ }
	runAccountTradeCycle(r)
	if *opens != 0 || syncCount != 1 || cancelCount != 1 {
		t.Fatalf("invalid cache aborted account maintenance: opens=%d sync=%d cancel=%d", *opens, syncCount, cancelCount)
	}
}

// Protection against the exact factory misuse identified in F3.
func TestStage3LeadMockFactoryRejectsMainRunner(t *testing.T) {
	r, _, _, _ := stage3MockRunner(t, []string{"BTCUSDT"})
	if _, err := leadMockTradeRunner(r.Config, r.LeadSymbols, mainTradeRunner(r.Config)); err == nil {
		t.Fatal("main's real broker and IO hooks must never be reused by a Lead fixture")
	}
	copyOfLead := *r
	if _, err := leadMockTradeRunner(r.Config, r.LeadSymbols, &copyOfLead); err == nil {
		t.Fatal("already-initialized Lead fixture must not be recycled")
	}
	r.leadMockPermit = nil
	if err := r.validate(); err == nil {
		t.Fatal("Mock=true without test-only permit cannot authorize Lead execution")
	}
}

// CODE_REVIEW_Stage3 F2: source attribution is explicitly per account in
// the common runner. Main keeps the legacy source label unchanged.
func TestStage3TradeCycleAttributionPerAccount(t *testing.T) {
	for _, tc := range []struct {
		name string
		id   binance.AccountID
		want string
	}{
		{"main", binance.MainAccountID, "start_trade"},
		{"lead", binance.LeadAccountID, "lead_trading"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r, _, _, _ := stage3MockRunner(t, []string{"BTCUSDT"})
			r.AccountID = tc.id
			if tc.id == binance.MainAccountID {
				r.Mock = false
				r.leadMockPermit = nil
				r.LeadSymbols = nil
			}
			r.AllowNewOpens = false
			got := ""
			r.ReadPositions = func(ctx context.Context) ([]types.FuturesPosition, error) {
				got = binanceapiusage.SourceFromContext(ctx)
				return []types.FuturesPosition{}, nil
			}
			runAccountTradeCycle(r)
			if got != tc.want {
				t.Fatalf("source=%q want=%q", got, tc.want)
			}
		})
	}
}
