package feature

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"

	"go_binance_futures/feature/strategy/line"
	"go_binance_futures/models"
	"go_binance_futures/service/spotexecution"
	"go_binance_futures/service/spotledger"

	"github.com/adshao/go-binance/v2/futures"
	"github.com/beego/beego/v2/client/orm"
	"github.com/shopspring/decimal"
)

// The only remote endpoint in these tests is this in-memory Spot fake.
// The runner uses the REAL 06-4 Processor, 06-3 SQL Ledger, 06-5 ReadModel,
// DBGate and 06-7-1 Expr evaluator. There is no Binance client/network.
type spot0673Exchange struct {
	price, freeQuote, freeBase string
	orders                     map[string]spotexecution.RemoteOrder
	trades                     map[string][]spotexecution.Trade
	places, finds, tradeReads  int
	sequence                   int64
	mode                       string // full, partial, before_accept, accepted_unknown, missing_trades
	failLookupOnce             bool
	feeAsset, feeAmount        string
}

func newSpot0673Exchange() *spot0673Exchange {
	return &spot0673Exchange{
		price: "100", freeQuote: "2000", freeBase: "50",
		orders:    map[string]spotexecution.RemoteOrder{},
		trades:    map[string][]spotexecution.Trade{},
		mode:      "full",
		feeAmount: "0",
	}
}
func (e *spot0673Exchange) Rules(_ context.Context, symbol string) (spotexecution.Rules, error) {
	if symbol != "BTCUSDT" && symbol != "WBTCUSDT" {
		return spotexecution.Rules{}, fmt.Errorf("unexpected Spot symbol %s", symbol)
	}
	return spotexecution.Rules{
		Symbol: symbol, Base: strings.TrimSuffix(symbol, "USDT"), Quote: "USDT", Trading: true, MarketAllowed: true,
		Step: "0.001", MinQty: "0.01", MaxQty: "100", MinNotional: "10", MaxNotional: "100000",
	}, nil
}
func (e *spot0673Exchange) Price(_ context.Context, symbol string) (string, error) {
	if symbol != "BTCUSDT" && symbol != "WBTCUSDT" {
		return "", fmt.Errorf("incorrect Spot symbol")
	}
	return e.price, nil
}
func (e *spot0673Exchange) Free(_ context.Context, asset string) (string, error) {
	switch asset {
	case "BTC", "WBTC":
		return e.freeBase, nil // includes manual assets, never sell all of it
	case "USDT":
		return e.freeQuote, nil
	default:
		return "", fmt.Errorf("unrecognized fake free asset %s", asset)
	}
}
func (e *spot0673Exchange) FindOrder(_ context.Context, symbol, client string) (*spotexecution.RemoteOrder, error) {
	e.finds++
	if symbol != "BTCUSDT" && symbol != "WBTCUSDT" {
		return nil, fmt.Errorf("foreign Spot order")
	}
	if e.failLookupOnce {
		e.failLookupOnce = false
		return nil, fmt.Errorf("simulated REST timeout")
	}
	order, ok := e.orders[client]
	if !ok {
		return nil, nil
	} // authoritative exchange -2013, never a timeout
	if order.Symbol != symbol {
		return nil, fmt.Errorf("Fake exchange client id is owned by a different Spot symbol")
	}
	return &order, nil
}
func (e *spot0673Exchange) makeTrade(o spotexecution.RemoteOrder, qty string) spotexecution.Trade {
	e.sequence++
	price := decimal.RequireFromString(e.price)
	quantity := decimal.RequireFromString(qty)
	return spotexecution.Trade{
		Symbol: o.Symbol, Side: o.Side, OrderID: o.ExchangeOrderID,
		TradeID: fmt.Sprintf("%d", 80000+e.sequence), Qty: qty,
		Price: e.price, QuoteQty: quantity.Mul(price).String(),
		Fee: e.feeAmount, FeeAsset: e.feeAsset, Time: 100000 + e.sequence,
	}
}
func (e *spot0673Exchange) Place(_ context.Context, request spotexecution.Request) (spotexecution.RemoteOrder, error) {
	e.places++
	if request.Type != "MARKET" || (request.Side != "BUY" && request.Side != "SELL") {
		return spotexecution.RemoteOrder{}, fmt.Errorf("unexpected fake exchange request")
	}
	if _, exists := e.orders[request.ClientOrderID]; exists {
		return spotexecution.RemoteOrder{}, fmt.Errorf("duplicate remote Spot POST")
	}
	if e.mode == "before_accept" {
		return spotexecution.RemoteOrder{}, fmt.Errorf("simulated ambiguous timeout before acceptance")
	}
	e.sequence++
	order := spotexecution.RemoteOrder{
		Symbol: request.Symbol, Side: request.Side, Type: request.Type,
		ClientOrderID: request.ClientOrderID, ExchangeOrderID: fmt.Sprintf("%d", 70000+e.sequence),
		OrigQty: request.Quantity, Status: "FILLED", ExecutedQty: request.Quantity,
	}
	switch e.mode {
	case "partial":
		order.Status = "PARTIALLY_FILLED"
		first := decimal.RequireFromString(request.Quantity).Div(decimal.NewFromInt(2)).String()
		order.ExecutedQty = first
		e.trades[order.ExchangeOrderID] = []spotexecution.Trade{e.makeTrade(order, first)}
	case "missing_trades":
		e.trades[order.ExchangeOrderID] = nil
	default:
		e.trades[order.ExchangeOrderID] = []spotexecution.Trade{e.makeTrade(order, request.Quantity)}
	}
	e.orders[request.ClientOrderID] = order
	if e.mode == "accepted_unknown" {
		e.failLookupOnce = true
		return spotexecution.RemoteOrder{}, fmt.Errorf("simulated SDK timeout after accepted POST")
	}
	return order, nil
}
func (e *spot0673Exchange) Cancel(_ context.Context, symbol, client string) (spotexecution.RemoteOrder, error) {
	if symbol != "BTCUSDT" && symbol != "WBTCUSDT" {
		return spotexecution.RemoteOrder{}, fmt.Errorf("wrong symbol")
	}
	order, ok := e.orders[client]
	if !ok {
		return spotexecution.RemoteOrder{}, spotexecution.ErrNotFound
	}
	if order.Symbol != symbol {
		return spotexecution.RemoteOrder{}, fmt.Errorf("Fake cancel symbol mismatch")
	}
	order.Status = "CANCELED"
	e.orders[client] = order
	return order, nil
}
func (e *spot0673Exchange) Trades(_ context.Context, symbol, orderID string) ([]spotexecution.Trade, error) {
	e.tradeReads++
	if symbol != "BTCUSDT" && symbol != "WBTCUSDT" {
		return nil, fmt.Errorf("wrong symbol")
	}
	for _, row := range e.trades[orderID] {
		if row.Symbol != symbol {
			return nil, fmt.Errorf("Fake trade belongs to a different Spot symbol")
		}
	}
	return append([]spotexecution.Trade(nil), e.trades[orderID]...), nil
}
func (e *spot0673Exchange) completePartial() error {
	for id, o := range e.orders {
		if o.Status != "PARTIALLY_FILLED" {
			continue
		}
		remaining := decimal.RequireFromString(o.OrigQty).Sub(decimal.RequireFromString(o.ExecutedQty))
		if !remaining.IsPositive() {
			return fmt.Errorf("invalid fake partial remainder")
		}
		fee, asset := e.feeAmount, e.feeAsset
		e.feeAmount, e.feeAsset = "0", ""
		e.trades[o.ExchangeOrderID] = append(e.trades[o.ExchangeOrderID], e.makeTrade(o, remaining.String()))
		e.feeAmount, e.feeAsset = fee, asset
		o.Status = "FILLED"
		o.ExecutedQty = o.OrigQty
		e.orders[id] = o
		return nil
	}
	return fmt.Errorf("no partial fake remote order")
}
func (e *spot0673Exchange) publishMissingTrades() error {
	for id, o := range e.orders {
		if o.Status != "FILLED" || len(e.trades[o.ExchangeOrderID]) > 0 {
			continue
		}
		e.trades[o.ExchangeOrderID] = []spotexecution.Trade{e.makeTrade(o, o.OrigQty)}
		e.orders[id] = o
		return nil
	}
	return fmt.Errorf("no missing fake trade")
}

func spot0673Fixture(t *testing.T) (*SpotQuantRunner, *spot0673Exchange, *spotexecution.Processor, orm.Ormer) {
	t.Helper()
	r, _, _, db := spotRunnerFixture(t)
	ex := newSpot0673Exchange()
	processor, err := spotexecution.NewProcessor(ex, spotexecution.DBGate{DB: db}, r.ledger)
	if err != nil {
		t.Fatal(err)
	}
	r.exec = processor
	// Actual 06-7-1 expression compiler/ROI behavior, not a fake BUY/SELL
	// evaluator; Spot ticker values are injected, not fetched from Binance.
	r.strategy = line.SpotRuleEvaluator{
		LoadEnvironment: func(c models.SpotSymbols) (map[string]interface{}, error) {
			price, err := strconv.ParseFloat(ex.price, 64)
			if err != nil {
				return nil, err
			}
			return map[string]interface{}{"NowPrice": price, "NowSymbolClose": price, "NowTime": time.Now().UnixMilli()}, nil
		},
		LoadKlines: func(_, _ string, _ int) ([]*futures.Kline, error) { return nil, fmt.Errorf("fallback not enabled") },
	}
	return r, ex, processor, db
}

func spot0673Enable(t *testing.T, db orm.Ormer) {
	t.Helper()
	spotRunnerSetSwitches(t, db, 1, 1)
}
func spot0673Tick(t *testing.T, r *SpotQuantRunner) {
	t.Helper()
	if err := r.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestSpotQuant0673RealProcessorBUYSELLFillReadModelAndNextCycle(t *testing.T) {
	r, ex, _, db := spot0673Fixture(t)
	ctx := context.Background()
	for i := 0; i < 5; i++ {
		spot0673Tick(t, r)
	}
	if ex.places != 0 {
		t.Fatal("global disabled submitted an order")
	}
	spot0673Enable(t, db)
	spot0673Tick(t, r)
	if ex.places != 1 {
		t.Fatalf("first real Processor BUY count=%d", ex.places)
	}
	buyID := spotQuantClientID("BTCUSDT", "BUY", 0, "1m", spotCandleOpen(r.clock(), "1m"))
	buy, err := r.ledger.FindOrderByClientID(buyID)
	if err != nil || buy.Side != "BUY" || buy.Status != "FILLED" {
		t.Fatalf("actual Processor BUY: %+v %v", buy, err)
	}
	snap, err := r.ledger.Accounting(buy.PositionID, "100")
	if err != nil || snap.Position.Status != "OPEN" || !snap.FullyPriced {
		t.Fatalf("verified BUY inventory: %+v %v", snap, err)
	}
	buyQty := decimal.RequireFromString(buy.RequestedQty)
	if !decimal.RequireFromString(snap.Position.RemainingQty).Equal(buyQty) {
		t.Fatal("BUY Fill lost inventory")
	}
	for i := 0; i < 20; i++ {
		spot0673Tick(t, r)
	}
	if ex.places != 1 {
		t.Fatalf("repeated 2s polls re-BUYed: %d", ex.places)
	}
	// 50 manually held BTC are in the wallet, but may NEVER be sold.
	ex.price = "110"
	if _, err = db.Raw("UPDATE spot_symbols SET close='110',updateTime=? WHERE symbol='BTCUSDT'", time.Now().UnixMilli()).Exec(); err != nil {
		t.Fatal(err)
	}
	spot0673Tick(t, r)
	if ex.places != 2 {
		t.Fatalf("missing Processor SELL, places=%d", ex.places)
	}
	var sell spotexecution.RemoteOrder
	for _, order := range ex.orders {
		if order.Side == "SELL" {
			sell = order
		}
	}
	if sell.Side != "SELL" || !decimal.RequireFromString(sell.OrigQty).Equal(buyQty) {
		t.Fatalf("SELL sold manual wallet or rounded inventory: buy=%s sell=%+v", buyQty, sell)
	}
	read, err := spotledger.NewSpotReadService(db, "sqlite")
	if err != nil {
		t.Fatal(err)
	}
	closed, err := read.Position(buy.PositionID)
	if err != nil || closed.Position.Status != "CLOSED" || closed.TotalPnL == "" || !closed.FullyPriced {
		t.Fatalf("06-5 readmodel missed real Processor close: %+v %v", closed, err)
	}
	if !decimal.RequireFromString(closed.Position.RemainingQty).IsZero() {
		t.Fatalf("real verified Fill not fully closed: %+v", closed.Position)
	}
	orders, err := read.Orders(spotledger.OrderFilter{PositionID: buy.PositionID, Page: 1, Limit: 10})
	if err != nil || orders.Total != 2 {
		t.Fatalf("BUY/SELL pair missing in read model: %+v %v", orders, err)
	}
	for i := 0; i < 20; i++ {
		spot0673Tick(t, r)
	}
	if ex.places != 2 {
		t.Fatalf("same 1m candle produced new BUY/SELL: %d", ex.places)
	}
	oldClock := r.clock()
	r.clock = func() time.Time { return oldClock.Add(time.Minute) }
	spot0673Tick(t, r)
	if ex.places != 3 {
		t.Fatalf("subsequent candle must allow a NEW verified position: places=%d", ex.places)
	}
	var sentinel string
	if err = db.Raw("SELECT marker FROM futures_managed_positions WHERE id=1").QueryRow(&sentinel); err != nil || sentinel != "untouched" {
		t.Fatalf("Fake Processor touched Futures records: %q %v", sentinel, err)
	}
	_ = ctx
}

func TestSpotQuant0673PartialFillRequiresRESTRecoveryBeforeSELL(t *testing.T) {
	r, ex, _, db := spot0673Fixture(t)
	spot0673Enable(t, db)
	ex.mode = "partial"
	spot0673Tick(t, r)
	if ex.places != 1 {
		t.Fatalf("partial BUY POST count=%d", ex.places)
	}
	active, exists, err := r.activeSpotPosition("BTCUSDT")
	if err != nil || !exists {
		t.Fatalf("missing verified partial inventory: %+v %v", active, err)
	}
	orders, err := r.ledger.ListOrders(active.ID)
	if err != nil || len(orders) != 1 || orders[0].Status != "PARTIALLY_FILLED" {
		t.Fatalf("partial order missing: %+v %v", orders, err)
	}
	partial, err := r.ledger.ListFills(active.ID)
	if err != nil || len(partial) != 1 {
		t.Fatalf("partial fake myTrades not persisted: %v %d", err, len(partial))
	}
	// The remote BUY is still only partially executed at the original
	// $100 price. Do not move price to $110 until all BUY fills settle.
	for i := 0; i < 5; i++ {
		spot0673Tick(t, r)
	}
	if ex.places != 1 {
		t.Fatalf("partial BUY triggered duplicate POST: %d", ex.places)
	}
	if err := ex.completePartial(); err != nil {
		t.Fatal(err)
	}
	ex.price = "110"
	ex.mode = "full"
	spot0673Tick(t, r) // first get all myTrades, no new SELL in recovery cycle
	if ex.places != 1 {
		t.Fatal("recovery cycle incorrectly sold inventory")
	}
	recovered, err := r.ledger.ListFills(active.ID)
	if err != nil || len(recovered) != 2 {
		t.Fatalf("late myTrades not imported: %v %d", err, len(recovered))
	}
	spot0673Tick(t, r)
	if ex.places != 2 {
		after, _, stateErr := r.activeSpotPosition("BTCUSDT")
		localOrders, ordersErr := r.ledger.ListOrders(active.ID)
		t.Fatalf("fully verified Fill not sold: places=%d active=%+v err=%v orders=%+v ordersErr=%v", ex.places, after, stateErr, localOrders, ordersErr)
	}
	closed, err := r.ledger.Accounting(active.ID, "110")
	if err != nil || closed.Position.Status != "CLOSED" {
		t.Fatalf("partial->final->SELL not closed: %+v %v", closed.Position, err)
	}
}

func TestSpotQuant0673AcceptedUnknownRestartAndNeverDuplicatePOST(t *testing.T) {
	r, ex, _, db := spot0673Fixture(t)
	spot0673Enable(t, db)
	ex.mode = "accepted_unknown"
	err := r.Run(context.Background())
	if !errors.Is(err, spotexecution.ErrUncertain) || ex.places != 1 {
		t.Fatalf("exchange accepted but SDK timeout must be ambiguous: %v places=%d", err, ex.places)
	}
	// Recreate REAL Processor and REAL Runner against the SAME SQLite Ledger.
	secondProcessor, err := spotexecution.NewProcessor(ex, spotexecution.DBGate{DB: db}, r.ledger)
	if err != nil {
		t.Fatal(err)
	}
	second, err := NewSpotQuantRunner(db, r.ledger, r.strategy, secondProcessor)
	if err != nil {
		t.Fatal(err)
	}
	second.clock = r.clock
	ex.mode = "full"
	spot0673Tick(t, second) // read-only reconciliation by persisted client ID
	if ex.places != 1 {
		t.Fatalf("restart re-POSTed ambiguous BUY: %d", ex.places)
	}
	for i := 0; i < 10; i++ {
		spot0673Tick(t, second)
	}
	if ex.places != 1 {
		t.Fatal("restart after accepted ambiguous BUY re-POSTed")
	}
	ex.price = "110"
	spot0673Tick(t, second)
	if ex.places != 2 {
		t.Fatalf("recovered accepted BUY not eligible for verified SELL: %d", ex.places)
	}
}

func TestSpotQuant0673AuthoritativeAbsentUnknownStaysBlocked(t *testing.T) {
	r, ex, _, db := spot0673Fixture(t)
	spot0673Enable(t, db)
	ex.mode = "before_accept"
	err := r.Run(context.Background())
	if !errors.Is(err, spotexecution.ErrUncertain) || ex.places != 1 {
		t.Fatalf("not accepted ambiguous BUY must remain unresolved: err=%v count=%d", err, ex.places)
	}
	ex.mode = "full"
	for i := 0; i < 10; i++ {
		next := r.clock().Add(time.Minute)
		r.clock = func() time.Time { return next }
		err = r.Run(context.Background())
		if !errors.Is(err, spotexecution.ErrNotFound) {
			t.Fatalf("an authoritative -2013 must require operator confirmation: %v", err)
		}
	}
	if ex.places != 1 {
		t.Fatalf("unknown order accidentally retried with a new client ID: %d", ex.places)
	}
}

func TestSpotQuant0673RemoteFILLEDWithoutMyTradesRecoversLateFill(t *testing.T) {
	r, ex, _, db := spot0673Fixture(t)
	spot0673Enable(t, db)
	ex.mode = "missing_trades"
	err := r.Run(context.Background())
	if !errors.Is(err, spotexecution.ErrNeedsReview) || ex.places != 1 {
		t.Fatalf("missing myTrades must fail closed, not trust FILLED ACK: err=%v places=%d", err, ex.places)
	}
	pending, err := r.ledger.ListUnsettledOrders("BTCUSDT")
	if err != nil || len(pending) != 1 {
		t.Fatalf("FILLED order with missing local fills must remain recoverable: %+v err=%v", pending, err)
	}
	// Underlying exchange Fill becomes available later: no second POST.
	if err := ex.publishMissingTrades(); err != nil {
		t.Fatal(err)
	}
	ex.mode = "full"
	spot0673Tick(t, r) // import late real myTrades; defer decisions until next cycle
	if ex.places != 1 {
		t.Fatalf("late myTrades recovery placed a duplicate BUY: %d", ex.places)
	}
	first, exists, err := r.activeSpotPosition("BTCUSDT")
	if err != nil || !exists || first.Status != "OPEN" {
		t.Fatalf("late trade did not complete inventory: %+v exists=%v err=%v", first, exists, err)
	}
	ex.price = "110"
	spot0673Tick(t, r)
	if ex.places != 2 {
		t.Fatalf("late Fill never became eligible for SELL: %d", ex.places)
	}
}

func TestSpotQuant0673KnownBNBFeeVsBaseFeeDustFailClosed(t *testing.T) {
	t.Run("known third asset BNB fee", func(t *testing.T) {
		r, ex, _, db := spot0673Fixture(t)
		spot0673Enable(t, db)
		ex.feeAsset, ex.feeAmount = "BNB", "0.0001"
		spot0673Tick(t, r)
		if ex.places != 1 {
			t.Fatal("BUY with legitimate BNB commission rejected")
		}
		pos, exists, err := r.activeSpotPosition("BTCUSDT")
		if err != nil || !exists || pos.Status != "OPEN" {
			t.Fatalf("BNB commission made Fill inventory unverifiable: %+v %v", pos, err)
		}
		view, err := r.ledger.Accounting(pos.ID, "100")
		if err != nil || view.FullyPriced || view.UnpricedFees["BNB"] == "" {
			t.Fatalf("BNB third asset should be unpriced but recorded: %+v %v", view, err)
		}
		ex.price = "110"
		spot0673Tick(t, r)
		if ex.places != 2 {
			t.Fatalf("legitimate known BNB fee must not permanently block SELL: places=%d", ex.places)
		}
		closed, err := r.ledger.Accounting(pos.ID, "110")
		if err != nil || closed.Position.Status != "CLOSED" || closed.FullyPriced {
			t.Fatalf("closed BNB fee position must not claim complete net PnL: %+v %v", closed, err)
		}
	})
	t.Run("base asset dust must not pretend fully closed", func(t *testing.T) {
		r, ex, _, db := spot0673Fixture(t)
		spot0673Enable(t, db)
		ex.feeAsset, ex.feeAmount = "BTC", "0.0005"
		spot0673Tick(t, r)
		if ex.places != 1 {
			t.Fatal("legitimate base fee BUY blocked")
		}
		pos, exists, err := r.activeSpotPosition("BTCUSDT")
		if err != nil || !exists {
			t.Fatalf("base fee position missing: %v", err)
		}
		remaining := decimal.RequireFromString(pos.RemainingQty)
		if remaining.Equal(decimal.RequireFromString("0.99")) {
			t.Fatal("BUY base fee not deducted from inventory")
		}
		ex.price = "110"
		err = r.Run(context.Background())
		if !errors.Is(err, spotexecution.ErrNeedsReview) || ex.places != 1 {
			t.Fatalf("dust SELL must fail closed, not touch manual wallet: %v places=%d", err, ex.places)
		}
		again, err := r.ledger.Accounting(pos.ID, "110")
		if err != nil || again.Position.Status != "OPEN" {
			t.Fatalf("dust was silently marked CLOSED: %+v %v", again.Position, err)
		}
	})
}

func TestSpotQuant0673DisabledGateAndWalletFundsCannotOverrideFillOwnership(t *testing.T) {
	r, ex, processor, db := spot0673Fixture(t)
	spot0673Enable(t, db)
	ex.freeQuote = "3" // despite enabled switches, quote budget is insufficient
	err := r.Run(context.Background())
	if err == nil || ex.places != 0 {
		t.Fatalf("insufficient quote wrongly submitted Spot BUY: err=%v places=%d", err, ex.places)
	}
	ex.freeQuote = "2000"
	spot0673Tick(t, r)
	if ex.places != 1 {
		t.Fatal("BUY not restored after valid balance")
	}
	pos, exists, err := r.activeSpotPosition("BTCUSDT")
	if err != nil || !exists {
		t.Fatal("BUY must form inventory")
	}
	ex.price = "110"
	ex.freeBase = "0.01" // strategy owns 0.99; never sell a different amount
	err = r.Run(context.Background())
	if !errors.Is(err, spotexecution.ErrNeedsReview) || ex.places != 1 {
		t.Fatalf("insufficient free base should block SELL: %v places=%d", err, ex.places)
	}
	// Keep cancellation/recovery controls available when global switch OFF.
	spotRunnerSetSwitches(t, db, 0, 1)
	for i := 0; i < 5; i++ {
		spot0673Tick(t, r)
	}
	if ex.places != 1 {
		t.Fatalf("global OFF submitted new trade: %d", ex.places)
	}
	orders, err := r.ledger.ListOrders(pos.ID)
	if err != nil || len(orders) != 1 {
		t.Fatalf("BUY evidence lost after OFF: %+v %v", orders, err)
	}
	if _, err := processor.Reconcile(context.Background(), orders[0].ID); err != nil {
		t.Fatalf("OFF must not prevent read-only recovery: %v", err)
	}
}

func TestSpotQuant0673RealProcessorFiveMinuteAndHourSignalDedup(t *testing.T) {
	for _, test := range []struct {
		interval string
		minutes  int
	}{
		{"5m", 5}, {"1h", 60},
	} {
		t.Run(test.interval, func(t *testing.T) {
			r, ex, _, db := spot0673Fixture(t)
			spot0673Enable(t, db)
			tech := fmt.Sprintf(`{"ema":[{"name":"ma_driver","enable":true,"kline_interval":"%s","period":5}]}`, test.interval)
			if _, err := db.Raw("UPDATE spot_symbols SET technology=? WHERE symbol='BTCUSDT'", tech).Exec(); err != nil {
				t.Fatal(err)
			}
			base := time.Date(2026, 11, 12, 12, 0, 0, 0, time.UTC)
			r.clock = func() time.Time { return base }
			spot0673Tick(t, r) // actual Processor BUY + confirmed Fill
			ex.price = "110"
			r.clock = func() time.Time { return base.Add(time.Minute) }
			spot0673Tick(t, r) // actual Processor SELL + confirmed Fill
			if ex.places != 2 {
				t.Fatalf("%s initial Processor BUY/SELL count=%d", test.interval, ex.places)
			}
			// Edit the rule text, but leave the primary interval unchanged:
			// an existing same-candle BUY must still prevent another POST.
			edited := `[{"name":"edited","type":"long","enable":true,"code":"NowPrice > 80"},
				{"name":"exit","type":"close_long","enable":true,"code":"ROI >= 8"}]`
			if _, err := db.Raw("UPDATE spot_symbols SET strategy=? WHERE symbol='BTCUSDT'", edited).Exec(); err != nil {
				t.Fatal(err)
			}
			for minute := 2; minute < test.minutes; minute++ {
				now := base.Add(time.Duration(minute) * time.Minute)
				r.clock = func() time.Time { return now }
				spot0673Tick(t, r)
			}
			if ex.places != 2 {
				t.Fatalf("%s repeated real Processor BUY/SELL in same Kline: %d", test.interval, ex.places)
			}
			next := base.Add(time.Duration(test.minutes) * time.Minute)
			r.clock = func() time.Time { return next }
			spot0673Tick(t, r)
			if ex.places != 3 {
				t.Fatalf("%s next candle failed to start next cycle: %d", test.interval, ex.places)
			}
		})
	}
}

func TestSpotQuant0673WSDuplicateFillAndRestReconcileRemainIdempotent(t *testing.T) {
	r, ex, processor, db := spot0673Fixture(t)
	spot0673Enable(t, db)
	ex.mode = "partial"
	spot0673Tick(t, r)
	active, exists, err := r.activeSpotPosition("BTCUSDT")
	if err != nil || !exists {
		t.Fatalf("partial BUY must establish managed cycle: %+v %v", active, err)
	}
	orders, err := r.ledger.ListOrders(active.ID)
	if err != nil || len(orders) != 1 {
		t.Fatalf("missing pending BUY: %+v %v", orders, err)
	}
	o := ex.orders[orders[0].ClientOrderID]
	first := ex.trades[o.ExchangeOrderID][0]
	report := spotexecution.ExecutionReport{
		Symbol: first.Symbol, Side: "BUY", ClientOrderID: orders[0].ClientOrderID,
		ExchangeOrderID: o.ExchangeOrderID, Status: "PARTIALLY_FILLED",
		ExecutionType: "TRADE", TradeID: first.TradeID,
		LastQty: first.Qty, LastPrice: first.Price, LastQuoteQty: first.QuoteQty,
		Fee: "0", TradeTime: first.Time,
	}
	for i := 0; i < 3; i++ {
		managed, err := processor.HandleExecutionReport(context.Background(), report)
		if err != nil || !managed {
			t.Fatalf("duplicate managed WS fill was not idempotent: %v managed=%v", err, managed)
		}
	}
	fills, err := r.ledger.ListFills(active.ID)
	if err != nil || len(fills) != 1 {
		t.Fatalf("duplicate WS event inserted duplicate immutable Fill: %v count=%d", err, len(fills))
	}
	if err := ex.completePartial(); err != nil {
		t.Fatal(err)
	}
	second := ex.trades[o.ExchangeOrderID][1]
	report.Status = "FILLED"
	report.TradeID, report.LastQty = second.TradeID, second.Qty
	report.LastPrice, report.LastQuoteQty = second.Price, second.QuoteQty
	report.TradeTime = second.Time
	if managed, err := processor.HandleExecutionReport(context.Background(), report); err != nil || !managed {
		t.Fatalf("final WS Fill not accepted: %v managed=%v", err, managed)
	}
	if _, err := processor.Reconcile(context.Background(), orders[0].ID); err != nil {
		t.Fatalf("REST after WS filled order must remain idempotent: %v", err)
	}
	fills, err = r.ledger.ListFills(active.ID)
	if err != nil || len(fills) != 2 {
		t.Fatalf("WS and REST double-counted fills: %d err=%v", len(fills), err)
	}
	ex.mode = "full"
	ex.price = "110"
	spot0673Tick(t, r)
	if ex.places != 2 {
		t.Fatalf("confirmed WS Fill did not permit real Processor SELL: %d", ex.places)
	}
}

func TestSpotQuant0673SwitchingIndicatorTimeframeCannotReBUYPreviousSignal(t *testing.T) {
	const target = "BTCUSDT"
	base := time.Date(2026, 11, 12, 12, 0, 0, 0, time.UTC)
	t.Run("prior one hour candle still blocks later one minute strategy", func(t *testing.T) {
		r, ex, _, db := spot0673Fixture(t)
		spot0673Enable(t, db)
		if _, err := db.Raw("UPDATE spot_symbols SET technology=? WHERE symbol=?",
			`{"ema":[{"name":"ema_1h","enable":true,"kline_interval":"1h","period":5}]}`, target).Exec(); err != nil {
			t.Fatal(err)
		}
		r.clock = func() time.Time { return base.Add(time.Minute) }
		spot0673Tick(t, r) // BUY with an 1h event identity
		ex.price = "110"
		r.clock = func() time.Time { return base.Add(2 * time.Minute) }
		spot0673Tick(t, r) // SELL
		if ex.places != 2 {
			t.Fatalf("seed BUY/SELL missing: %d", ex.places)
		}
		if _, err := db.Raw("UPDATE spot_symbols SET technology='{}' WHERE symbol=?", target).Exec(); err != nil {
			t.Fatal(err)
		}
		r.clock = func() time.Time { return base.Add(3 * time.Minute) }
		spot0673Tick(t, r)
		if ex.places != 2 {
			t.Fatalf("switch 1h->1m repeated BUY in same previously used 1h candle: %d", ex.places)
		}
	})
	t.Run("prior one minute BUY inside new one hour candle blocks second BUY", func(t *testing.T) {
		r, ex, _, db := spot0673Fixture(t)
		spot0673Enable(t, db)
		r.clock = func() time.Time { return base.Add(time.Minute) }
		spot0673Tick(t, r) // BUY under 1m event
		ex.price = "110"
		spot0673Tick(t, r) // SELL
		if ex.places != 2 {
			t.Fatalf("seed BUY/SELL missing: %d", ex.places)
		}
		if _, err := db.Raw("UPDATE spot_symbols SET technology=? WHERE symbol=?",
			`{"ema":[{"name":"ema_1h","enable":true,"kline_interval":"1h","period":5}]}`, target).Exec(); err != nil {
			t.Fatal(err)
		}
		r.clock = func() time.Time { return base.Add(3 * time.Minute) }
		spot0673Tick(t, r)
		if ex.places != 2 {
			t.Fatalf("switch 1m->1h repeated BUY within same new 1h event: %d", ex.places)
		}
	})
}

// Review 06-7-3 F1: keep ledger's REAL created_at completely untouched,
// while the strategy intentionally runs against a historical event clock.
// Previous mixed-clock SQL could silently send a second BUY in 1m -> 1h.
func TestReview0673SeparatedLedgerClockCannotRepeatHistoricalSignal(t *testing.T) {
	r, ex, _, db := spot0673Fixture(t)
	spot0673Enable(t, db)
	base := time.Date(2025, 11, 12, 12, 0, 0, 0, time.UTC)
	r.clock = func() time.Time { return base.Add(time.Minute) }
	spot0673Tick(t, r) // actual Processor and Fill BUY, wall-clock DB timestamp
	ex.price = "110"
	spot0673Tick(t, r) // same minute: actual Processor SELL
	if ex.places != 2 {
		t.Fatalf("seeded BUY/SELL = %d, expected 2", ex.places)
	}
	var actualCreated int64
	if err := db.Raw("SELECT created_at FROM spot_managed_orders WHERE symbol='BTCUSDT' AND side='BUY'").QueryRow(&actualCreated); err != nil {
		t.Fatal(err)
	}
	if actualCreated <= base.Add(3*time.Minute).UnixMilli() {
		t.Fatalf("regression fixture failed to separate clocks: order=%d event=%d", actualCreated, base.UnixMilli())
	}
	tech := `{"ema":[{"name":"ema_1h","enable":true,"kline_interval":"1h","period":5}]}`
	if _, err := db.Raw("UPDATE spot_symbols SET technology=? WHERE symbol='BTCUSDT'", tech).Exec(); err != nil {
		t.Fatal(err)
	}
	r.clock = func() time.Time { return base.Add(3 * time.Minute) }
	spot0673Tick(t, r)
	if ex.places != 2 {
		t.Fatalf("different clocks duplicated BUY inside 1h Kline: %d", ex.places)
	}
	// No change to the stored wall-clock timestamps, including on restart.
	secondProcessor, err := spotexecution.NewProcessor(ex, spotexecution.DBGate{DB: db}, r.ledger)
	if err != nil {
		t.Fatal(err)
	}
	second, err := NewSpotQuantRunner(db, r.ledger, r.strategy, secondProcessor)
	if err != nil {
		t.Fatal(err)
	}
	second.clock = r.clock
	spot0673Tick(t, second)
	if ex.places != 2 {
		t.Fatalf("restart re-BUYed separated clock: %d", ex.places)
	}
	second.clock = func() time.Time { return base.Add(time.Hour) }
	spot0673Tick(t, second)
	if ex.places != 3 {
		t.Fatalf("next genuine 1h candle should allow a new cycle: %d", ex.places)
	}
}

func TestReview0673OverfilledFILLEDRaisesReviewInsteadOfSilentRetry(t *testing.T) {
	r, ex, _, db := spot0673Fixture(t)
	spot0673Enable(t, db)
	spot0673Tick(t, r)
	if ex.places != 1 {
		t.Fatal("missing initial BUY")
	}
	if _, err := db.Raw("UPDATE spot_managed_orders SET filled_qty='2' WHERE symbol='BTCUSDT' AND side='BUY' AND status='FILLED'").Exec(); err != nil {
		t.Fatal(err)
	}
	pending, err := r.ledger.ListUnsettledOrders("BTCUSDT")
	if err != nil || len(pending) != 1 {
		t.Fatalf("overfilled FILLED must be detected: %+v %v", pending, err)
	}
	for i := 0; i < 3; i++ {
		err = r.Run(context.Background())
		if !errors.Is(err, spotledger.ErrNeedsReview) || ex.places != 1 {
			t.Fatalf("overfill silent/duplicate trade at pass %d: err=%v places=%d", i, err, ex.places)
		}
	}
	if !strings.Contains(r.lastErr["BTCUSDT"], "exceeds requested quantity") {
		t.Fatalf("operator-visible review error absent: %+v", r.lastErr)
	}
	if _, err := db.Raw("UPDATE spot_managed_orders SET filled_qty=requested_qty WHERE symbol='BTCUSDT' AND side='BUY'").Exec(); err != nil {
		t.Fatal(err)
	}
	spot0673Tick(t, r)
	if len(r.lastErr) != 0 || ex.places != 1 {
		t.Fatalf("review state not recoverable: errors=%+v places=%d", r.lastErr, ex.places)
	}
}

func TestReview0673ClockRollbackCannotBypassExistingBUY(t *testing.T) {
	r, ex, _, db := spot0673Fixture(t)
	spot0673Enable(t, db)
	base := time.Date(2025, 11, 12, 12, 0, 0, 0, time.UTC)
	r.clock = func() time.Time { return base.Add(3 * time.Minute) }
	spot0673Tick(t, r)
	ex.price = "110"
	spot0673Tick(t, r)
	if ex.places != 2 {
		t.Fatal("failed to seed first completed cycle")
	}
	r.clock = func() time.Time { return base.Add(-time.Minute) }
	err := r.Run(context.Background())
	if !errors.Is(err, spotledger.ErrNeedsReview) || ex.places != 2 {
		t.Fatalf("rolled-back event clock re-BUYed: err=%v places=%d", err, ex.places)
	}
}

func TestReview0673LegacyHashedClientIDsFailClosedBeforeAutomaticBUY(t *testing.T) {
	r, ex, _, db := spot0673Fixture(t)
	spot0673Enable(t, db)
	// A previous v3 client ID is not reversible. Never silently assume
	// it belonged to a different candle and submit another strategy BUY.
	if _, err := db.Raw(`INSERT INTO spot_managed_orders(
		position_id,symbol,side,order_type,client_order_id,exchange_order_id,
		requested_qty,filled_qty,status,created_at,updated_at)
		VALUES(999,'BTCUSDT','BUY','MARKET','sq_legacy_unreadable',NULL,
		'1','1','FILLED',1000,1000)`).Exec(); err != nil {
		t.Fatal(err)
	}
	err := r.Run(context.Background())
	if !errors.Is(err, spotledger.ErrNeedsReview) || ex.places != 0 {
		t.Fatalf("non-decodable old strategy BUY must not be ignored: err=%v places=%d", err, ex.places)
	}
}

func TestReview0673TwoSymbolsThroughRealProcessorStayExactAndIndependent(t *testing.T) {
	r, ex, _, db := spot0673Fixture(t)
	spot0673Enable(t, db)
	if _, err := db.Raw(`INSERT INTO spot_symbols(
		id,symbol,spot_quant_enabled,usdt,profit,loss,strategy,technology,type,updateTime)
		VALUES (2,'WBTCUSDT',1,'100','8','6',?,'{}','USDT',?)`,
		runnerLongRules, time.Now().UnixMilli()).Exec(); err != nil {
		t.Fatal(err)
	}
	spot0673Tick(t, r)
	if ex.places != 2 {
		t.Fatalf("two enabled symbols must create two exact Spot BUYs, got %d", ex.places)
	}
	for _, symbol := range []string{"BTCUSDT", "WBTCUSDT"} {
		page, err := r.ledger.ListPositions(spotledger.PositionFilter{Symbol: symbol, Page: 1, Limit: 10})
		if err != nil || page.Total != 1 || len(page.List) != 1 || page.List[0].Symbol != symbol {
			t.Fatalf("cross-symbol position leak for %s: %+v %v", symbol, page, err)
		}
		snap, err := r.ledger.Accounting(page.List[0].ID, "100")
		if err != nil || snap.Position.Status != "OPEN" {
			t.Fatalf("cross-symbol accounting invalid %s: %+v %v", symbol, snap, err)
		}
	}
	// Disable WBTC only. BTC may close, but WBTC remains owned and OPEN.
	if _, err := db.Raw("UPDATE spot_symbols SET spot_quant_enabled=0 WHERE symbol='WBTCUSDT'").Exec(); err != nil {
		t.Fatal(err)
	}
	ex.price = "110"
	spot0673Tick(t, r)
	if ex.places != 3 {
		t.Fatalf("disabled WBTC was auto-sold: places=%d", ex.places)
	}
	for _, o := range ex.orders {
		if o.Side == "SELL" && o.Symbol != "BTCUSDT" {
			t.Fatalf("wrong coin auto-sold: %+v", o)
		}
	}
	pending, err := r.ledger.ListPositions(spotledger.PositionFilter{Symbol: "WBTCUSDT", Page: 1, Limit: 10})
	if err != nil || len(pending.List) != 1 {
		t.Fatalf("WBTC position disappeared: %+v %v", pending, err)
	}
	snap, err := r.ledger.Accounting(pending.List[0].ID, "110")
	if err != nil || snap.Position.Status != "OPEN" {
		t.Fatalf("disabled WBTC unexpectedly closed: %+v %v", snap, err)
	}
	if _, err := db.Raw("UPDATE spot_symbols SET spot_quant_enabled=1 WHERE symbol='WBTCUSDT'").Exec(); err != nil {
		t.Fatal(err)
	}
	spot0673Tick(t, r)
	if ex.places != 4 {
		t.Fatalf("re-enabled WBTC failed independent SELL: places=%d", ex.places)
	}
	final, err := r.ledger.Accounting(pending.List[0].ID, "110")
	if err != nil || final.Position.Status != "CLOSED" {
		t.Fatalf("WBTC verified close failed: %+v %v", final, err)
	}
}
