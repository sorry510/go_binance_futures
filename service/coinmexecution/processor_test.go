package coinmexecution

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/adshao/go-binance/v2/delivery"
	"github.com/beego/beego/v2/client/orm"
	_ "github.com/mattn/go-sqlite3"
	"go_binance_futures/feature"
	"go_binance_futures/feature/api/binance"
	"go_binance_futures/models"
)

var fixtureOnce sync.Once
var fixtureCount atomic.Uint64

func cm5Fixture(t *testing.T) orm.Ormer {
	t.Helper()
	_ = orm.RegisterDriver("sqlite3", orm.DRSqlite)
	if _, err := orm.GetDB("default"); err != nil {
		if e := orm.RegisterDataBase("default", "sqlite3", "file:coinmexecution_default?mode=memory&cache=shared"); e != nil {
			t.Fatal(e)
		}
	}
	fixtureOnce.Do(func() { orm.RegisterModel(new(models.DeliverySymbols)) })
	alias := fmt.Sprintf("coinm5_%d", fixtureCount.Add(1))
	if err := orm.RegisterDataBase(alias, "sqlite3", filepath.Join(t.TempDir(), "coinm5.db")); err != nil {
		t.Fatal(err)
	}
	if err := orm.RunSyncdb(alias, false, false); err != nil {
		t.Fatal(err)
	}
	db := orm.NewOrmUsingDB(alias)
	for _, sql := range []string{
		"ALTER TABLE delivery_symbols ADD COLUMN budget_usd DECIMAL(36,18) NOT NULL DEFAULT 0",
		"CREATE TABLE config (id INTEGER PRIMARY KEY,coin_m_quant_enable INTEGER NOT NULL DEFAULT 0)",
		"INSERT INTO config(id,coin_m_quant_enable) VALUES(1,1)",
		"CREATE TABLE coin_m_managed_positions (id INTEGER PRIMARY KEY AUTOINCREMENT, symbol VARCHAR(32) NOT NULL, position_side VARCHAR(8) NOT NULL, margin_asset VARCHAR(32) NOT NULL, contract_size DECIMAL(36,18) NOT NULL DEFAULT 0, entry_price DECIMAL(36,18) NOT NULL DEFAULT 0, opened_contracts DECIMAL(36,18) NOT NULL DEFAULT 0, remaining_contracts DECIMAL(36,18) NOT NULL DEFAULT 0, realized_pnl DECIMAL(36,18) NOT NULL DEFAULT 0, status VARCHAR(32) NOT NULL, created_at BIGINT NOT NULL, updated_at BIGINT NOT NULL, closed_at BIGINT NOT NULL DEFAULT 0)",
		"CREATE TABLE coin_m_managed_orders (id INTEGER PRIMARY KEY AUTOINCREMENT, position_id BIGINT NOT NULL, symbol VARCHAR(32) NOT NULL, position_side VARCHAR(8) NOT NULL, intent VARCHAR(16) NOT NULL, side VARCHAR(4) NOT NULL, order_type VARCHAR(32) NOT NULL, client_order_id VARCHAR(64) NOT NULL UNIQUE, exchange_order_id VARCHAR(64) NULL, requested_contracts DECIMAL(36,18) NOT NULL DEFAULT 0, filled_contracts DECIMAL(36,18) NOT NULL DEFAULT 0, status VARCHAR(32) NOT NULL, created_at BIGINT NOT NULL, updated_at BIGINT NOT NULL)",
		"CREATE UNIQUE INDEX uq_cm5_order_exchange ON coin_m_managed_orders(symbol,exchange_order_id)",
		"CREATE TABLE coin_m_trade_fills (id INTEGER PRIMARY KEY AUTOINCREMENT, position_id BIGINT NOT NULL, order_id BIGINT NOT NULL, symbol VARCHAR(32) NOT NULL, position_side VARCHAR(8) NOT NULL, side VARCHAR(4) NOT NULL, exchange_order_id VARCHAR(64) NOT NULL, trade_id VARCHAR(64) NOT NULL, contracts DECIMAL(36,18) NOT NULL, price DECIMAL(36,18) NOT NULL, realized_pnl DECIMAL(36,18) NOT NULL DEFAULT 0, pnl_asset VARCHAR(32) NOT NULL, commission DECIMAL(36,18) NOT NULL DEFAULT 0, commission_asset VARCHAR(32) NOT NULL, trade_time BIGINT NOT NULL)",
		"CREATE UNIQUE INDEX uq_cm5_fill_trade ON coin_m_trade_fills(symbol,exchange_order_id,trade_id)",
	} {
		if _, e := db.Raw(sql).Exec(); e != nil {
			t.Fatal(sql, e)
		}
	}
	row := &models.DeliverySymbols{Symbol: "BTCUSD_PERP", ContractType: "PERPETUAL", ContractSize: 100,
		MarginAsset: "BTC", Enable: 1, Leverage: 4, MarginType: "CROSSED",
		Strategy:   `[{"name":"test","type":"long","enable":true,"code":"Close>Open"},{"name":"test","type":"short","enable":true,"code":"Close<Open"}]`,
		Technology: "{}", TickSize: "0.1", StepSize: "1", Profit: "8", Loss: "6"}
	id, err := db.Insert(row)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = db.Raw("UPDATE delivery_symbols SET budget_usd='250' WHERE id=?", id).Exec(); err != nil {
		t.Fatal(err)
	}
	return db
}

type mockCM5 struct {
	ready, hedge  bool
	spec          feature.CoinMPerpetualSpec
	qty           map[string]string
	orders        map[string]*OrderAck
	trades        map[int64][]binance.CoinMTrade
	openAlgos     []binance.CoinMAlgoOrderResponse
	closedAlgos   []binance.CoinMAlgoOrderResponse
	calls         []binance.CoinMTradeOrder
	failCreate    bool
	failQuery     bool
	failTrades    bool
	nextID        int64
	algoChildID   int64
	algoTriggerID string
	algoState     string
}

func newMockCM5() *mockCM5 {
	return &mockCM5{ready: true, hedge: true, qty: map[string]string{"LONG": "0", "SHORT": "0"},
		orders: map[string]*OrderAck{}, trades: map[int64][]binance.CoinMTrade{}, nextID: 9000,
		spec: feature.CoinMPerpetualSpec{Symbol: "BTCUSD_PERP", ContractType: "PERPETUAL",
			ContractSize: 100, MarginAsset: "BTC", MinQty: "1", MaxQty: "100000", StepSize: "1",
			MarketMinQty: "1", MarketMaxQty: "100000", MarketStepSize: "1",
			TickSize: "0.1", MinPrice: "0", MaxPrice: "1000000"}}
}
func (m *mockCM5) Ready() bool                        { return m.ready }
func (m *mockCM5) Mode(context.Context) (bool, error) { return m.hedge, nil }
func (m *mockCM5) Spec(context.Context, string) (feature.CoinMPerpetualSpec, error) {
	return m.spec, nil
}
func (m *mockCM5) Position(_ context.Context, _ string, side string) (string, error) {
	return m.qty[side], nil
}
func (m *mockCM5) TradeSettings(context.Context, string, string, int64, string) error { return nil }
func (m *mockCM5) HasOpenOrders(context.Context, string) (bool, error)                { return false, nil }
func (m *mockCM5) Mark(context.Context, string) (string, error)                       { return "50000", nil }
func (m *mockCM5) Available(context.Context, string) (string, error)                  { return "1.0", nil }
func (m *mockCM5) Create(_ context.Context, r binance.CoinMTradeOrder) (*OrderAck, error) {
	m.calls = append(m.calls, r)
	m.nextID++
	ack := &OrderAck{ClientOrderID: r.ClientOrderID, ExchangeOrderID: m.nextID,
		Symbol: r.Symbol, Side: r.Side, PositionSide: r.PositionSide, Status: "NEW", FilledContracts: "0"}
	m.orders[r.ClientOrderID] = ack
	if m.failCreate {
		return nil, errors.New("simulated transport timeout after exchange accepted request")
	}
	return ack, nil
}
func (m *mockCM5) Query(_ context.Context, _ string, id string) (*OrderAck, error) {
	if m.failQuery {
		return nil, errors.New("simulated 429 / uncertain query")
	}
	return m.orders[id], nil
}
func (m *mockCM5) Cancel(_ context.Context, _ string, id string) (*OrderAck, error) {
	ack := m.orders[id]
	if ack == nil {
		return nil, errors.New("remote order not found")
	}
	copy := *ack
	copy.Status = "CANCELED"
	m.orders[id] = &copy
	return &copy, nil
}
func (m *mockCM5) Trades(_ context.Context, _ string, id int64) ([]binance.CoinMTrade, error) {
	if m.failTrades {
		return nil, errors.New("simulated REST Fill outage")
	}
	return m.trades[id], nil
}
func (m *mockCM5) Algo(_ context.Context, r binance.CoinMAlgoOrderRequest) (*AlgoAck, error) {
	m.nextID++
	a := binance.CoinMAlgoOrderResponse{AlgoID: m.nextID, ClientAlgoID: r.ClientAlgoID,
		Symbol: r.Symbol, PositionSide: r.PositionSide, Side: r.Side, OrderType: r.Type, AlgoStatus: "NEW"}
	m.openAlgos = append(m.openAlgos, a)
	return &AlgoAck{ClientAlgoID: a.ClientAlgoID, AlgoID: a.AlgoID, Symbol: a.Symbol,
		Side: a.Side, PositionSide: a.PositionSide, Status: a.AlgoStatus}, nil
}
func (m *mockCM5) CancelAlgo(_ context.Context, _ string, id string) error {
	for n, a := range m.openAlgos {
		if a.ClientAlgoID == id {
			a.AlgoStatus = "CANCELED"
			m.closedAlgos = append(m.closedAlgos, a)
			m.openAlgos = append(m.openAlgos[:n], m.openAlgos[n+1:]...)
			return nil
		}
	}
	return errors.New("unknown managed Algo")
}
func (m *mockCM5) OpenAlgos(context.Context, string) ([]binance.CoinMAlgoOrderResponse, error) {
	return m.openAlgos, nil
}
func sampleTrade(ack *OrderAck, tradeID int64, qty, price, pnl string) binance.CoinMTrade {
	return binance.CoinMTrade{Symbol: ack.Symbol, OrderID: ack.ExchangeOrderID, TradeID: tradeID,
		Side: ack.Side, PositionSide: ack.PositionSide, Price: price, Quantity: qty,
		RealizedPnL: pnl, MarginAsset: "BTC", Commission: "0.0001", CommissionAsset: "BTC",
		Time: time.Now().UnixMilli()}
}
func expectCM5(t *testing.T, db orm.Ormer, sql string, want int64) {
	t.Helper()
	var n int64
	if err := db.Raw(sql).QueryRow(&n); err != nil || n != want {
		t.Fatalf("query %s count=%d want=%d err=%v", sql, n, want, err)
	}
}

func (m *mockCM5) QueryAlgo(_ context.Context, symbol, id string) (binance.CoinMAlgoDetail, error) {
	for _, a := range append(append([]binance.CoinMAlgoOrderResponse{}, m.openAlgos...), m.closedAlgos...) {
		if a.Symbol == symbol && a.ClientAlgoID == id {
			actualID := json.RawMessage("0")
			status := a.AlgoStatus
			if m.algoState != "" && (m.algoTriggerID == "" || m.algoTriggerID == id) {
				status = m.algoState
				if m.algoChildID > 0 {
					actualID = json.RawMessage(strconv.FormatInt(m.algoChildID, 10))
				}
			}
			return binance.CoinMAlgoDetail{AlgoID: a.AlgoID, ClientAlgoID: a.ClientAlgoID,
				Symbol: a.Symbol, Side: a.Side, PositionSide: a.PositionSide,
				OrderType: a.OrderType, AlgoStatus: status, ActualOrderID: actualID}, nil
		}
	}
	return binance.CoinMAlgoDetail{}, errors.New("mock Algo not found")
}
func (m *mockCM5) OrderByID(_ context.Context, symbol string, orderID int64) (*OrderAck, error) {
	for _, o := range m.orders {
		if o.Symbol == symbol && o.ExchangeOrderID == orderID {
			return o, nil
		}
	}
	return nil, errors.New("mock child order not found")
}

var _ Gateway = (*mockCM5)(nil)
var _ = delivery.OrderStatusTypeNew
var _ = strings.Contains

func TestCM5LongOpenPartialCloseAndDeduplicatedVerifiedFills(t *testing.T) {
	db := cm5Fixture(t)
	api := newMockCM5()
	p := NewProcessor(db, api)
	ctx := context.Background()
	opened, err := p.Open(ctx, OpenRequest{Symbol: "BTCUSD_PERP", PositionSide: "LONG", ClientOrderID: "cm5open1001", Type: "MARKET"})
	if err != nil {
		t.Fatal(err)
	}
	if opened.Intent != "OPEN" || opened.RequestedContracts != "10" || opened.Side != "BUY" ||
		opened.Status != "NEW" || opened.ExchangeOrderID == "" {
		t.Fatalf("unexpected owned entry: %+v", opened)
	}
	if len(api.calls) != 1 {
		t.Fatal("missing normal DAPI MARKET mock call")
	}
	if _, err = p.Open(ctx, OpenRequest{Symbol: "BTCUSD_PERP", PositionSide: "LONG", ClientOrderID: "cm5open1001", Type: "MARKET"}); err != nil || len(api.calls) != 1 {
		t.Fatalf("duplicate client ID submitted a second order: %v count=%d", err, len(api.calls))
	}
	api.qty["LONG"] = "10"
	ack := api.orders["cm5open1001"]
	fill := sampleTrade(ack, 101, "10", "50000", "0")
	ok, err := p.ApplyFill(fill)
	if err != nil || !ok {
		t.Fatalf("valid opening Fill failed: %t %v", ok, err)
	}
	if ok, err = p.ApplyFill(fill); err != nil || ok {
		t.Fatalf("duplicate trade ID was applied twice: %t %v", ok, err)
	}
	pos, err := loadPosition(db, opened.PositionID)
	if err != nil || pos.Status != "OPEN" || pos.OpenedContracts != "10" || pos.RemainingContracts != "10" || pos.EntryPrice != "50000" {
		t.Fatalf("opening position not computed from actual Fill: %+v %v", pos, err)
	}
	closeOrder, err := p.Close(ctx, CloseRequest{PositionID: pos.ID, ClientOrderID: "cm5close1001", Type: "MARKET"})
	if err != nil {
		t.Fatal(err)
	}
	if closeOrder.Side != "SELL" || closeOrder.PositionSide != "LONG" || closeOrder.RequestedContracts != "10" {
		t.Fatalf("LONG close must sell owned contracts: %+v", closeOrder)
	}
	ackClose := api.orders["cm5close1001"]
	partial := sampleTrade(ackClose, 201, "4", "55000", "0.0002")
	if ok, err = p.ApplyFill(partial); err != nil || !ok {
		t.Fatalf("partial close failed: %t %v", ok, err)
	}
	pos, err = loadPosition(db, pos.ID)
	if err != nil || pos.RemainingContracts != "6" || pos.Status != "OPEN" {
		t.Fatalf("partial close lost inventory: %+v %v", pos, err)
	}
	rest := sampleTrade(ackClose, 202, "6", "56000", "0.0003")
	if ok, err = p.ApplyFill(rest); err != nil || !ok {
		t.Fatalf("final close Fill failed: %t %v", ok, err)
	}
	pos, err = loadPosition(db, pos.ID)
	if err != nil || pos.Status != "CLOSED" || pos.RemainingContracts != "0" || pos.RealizedPnL != "0.0005" {
		t.Fatalf("close cycle summary corrupted: %+v %v", pos, err)
	}
	expectCM5(t, db, "SELECT COUNT(*) FROM coin_m_trade_fills", 3)
	// Late private WS is idempotent: it must not create a second Fill.
	event := &delivery.WsUserDataEvent{Event: delivery.UserDataEventTypeOrderTradeUpdate}
	event.OrderTradeUpdate.ClientOrderID = "cm5close1001"
	event.OrderTradeUpdate.Symbol = "BTCUSD_PERP"
	event.OrderTradeUpdate.Side = delivery.SideTypeSell
	event.OrderTradeUpdate.PositionSide = delivery.PositionSideTypeLong
	event.OrderTradeUpdate.ID = ackClose.ExchangeOrderID
	event.OrderTradeUpdate.Status = delivery.OrderStatusTypeFilled
	event.OrderTradeUpdate.ExecutionType = delivery.OrderExecutionTypeTrade
	event.OrderTradeUpdate.AccumulatedFilledQty = "10"
	event.OrderTradeUpdate.LastFilledQty = "6"
	event.OrderTradeUpdate.LastFilledPrice = "56000"
	event.OrderTradeUpdate.TradeID = 202
	event.OrderTradeUpdate.RealizedPnL = "0.0003"
	event.OrderTradeUpdate.MarginAsset = "BTC"
	event.OrderTradeUpdate.Commission = "0.0001"
	event.OrderTradeUpdate.CommissionAsset = "BTC"
	event.OrderTradeUpdate.TradeTime = rest.Time
	if err = p.OnWSOrder(event); err != nil {
		t.Fatal(err)
	}
	expectCM5(t, db, "SELECT COUNT(*) FROM coin_m_trade_fills", 3)
	// Reconcile the same trades from REST: no extra count or PnL.
	ackClose.FilledContracts = "10"
	ackClose.Status = "FILLED"
	api.trades[ackClose.ExchangeOrderID] = []binance.CoinMTrade{partial, rest}
	if _, err = p.Reconcile(ctx, "cm5close1001"); err != nil {
		t.Fatal(err)
	}
	pos, err = loadPosition(db, pos.ID)
	if err != nil || pos.RealizedPnL != "0.0005" || pos.Status != "CLOSED" {
		t.Fatalf("REST+WS double counted fills: %+v %v", pos, err)
	}
	expectCM5(t, db, "SELECT COUNT(*) FROM coin_m_trade_fills", 3)
}
func TestCM5TimeoutNeverBlindPostsAgain(t *testing.T) {
	db := cm5Fixture(t)
	api := newMockCM5()
	api.failCreate = true
	p := NewProcessor(db, api)
	ctx := context.Background()
	order, err := p.Open(ctx, OpenRequest{Symbol: "BTCUSD_PERP", PositionSide: "SHORT", ClientOrderID: "cm5timeout0001", Type: "MARKET"})
	if !errors.Is(err, ErrUncertain) || order.Status != "UNKNOWN" || len(api.calls) != 1 {
		t.Fatalf("unknown POST must be held for reconciliation: %+v %v, calls=%d", order, err, len(api.calls))
	}
	second, err := p.Open(ctx, OpenRequest{Symbol: "BTCUSD_PERP", PositionSide: "SHORT", ClientOrderID: "cm5timeout0001", Type: "MARKET"})
	if err != nil || second.ID != order.ID || len(api.calls) != 1 {
		t.Fatalf("retry blindly sent duplicate POST: %+v %v, calls=%d", second, err, len(api.calls))
	}
	api.failQuery = true
	if _, err = p.Reconcile(ctx, "cm5timeout0001"); !errors.Is(err, ErrUncertain) || len(api.calls) != 1 {
		t.Fatalf("429 must not retry a POST: %v calls=%d", err, len(api.calls))
	}
	api.failQuery = false
	if _, err = p.Reconcile(ctx, "cm5timeout0001"); err != nil {
		t.Fatalf("confirmed remote order should reconcile: %v", err)
	}
	expectCM5(t, db, "SELECT COUNT(*) FROM coin_m_managed_orders", 1)
}
func TestCM5SwitchModeAndUnmanagedGuards(t *testing.T) {
	for _, scenario := range []struct {
		name  string
		setup func(orm.Ormer, *mockCM5)
	}{
		{"global_off", func(db orm.Ormer, m *mockCM5) { _, _ = db.Raw("UPDATE config SET coin_m_quant_enable=0").Exec() }},
		{"symbol_off", func(db orm.Ormer, m *mockCM5) { _, _ = db.Raw("UPDATE delivery_symbols SET enable=0").Exec() }},
		{"private_ws_disconnected", func(db orm.Ormer, m *mockCM5) { m.ready = false }},
		{"hedge_mode_off", func(db orm.Ormer, m *mockCM5) { m.hedge = false }},
		{"unmanaged_remote_long", func(db orm.Ormer, m *mockCM5) { m.qty["LONG"] = "1" }},
		{"unmanaged_remote_short", func(db orm.Ormer, m *mockCM5) { m.qty["SHORT"] = "2" }},
		{"exchange_face_value_changed", func(db orm.Ormer, m *mockCM5) { m.spec.ContractSize = 10 }},
		{"insufficient_balance", func(db orm.Ormer, m *mockCM5) { m.spec.MarketMinQty = "11" }},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			db := cm5Fixture(t)
			m := newMockCM5()
			scenario.setup(db, m)
			p := NewProcessor(db, m)
			_, err := p.Open(context.Background(), OpenRequest{Symbol: "BTCUSD_PERP", PositionSide: "LONG", ClientOrderID: "cm5guard001", Type: "MARKET"})
			if err == nil || len(m.calls) != 0 {
				t.Fatalf("unsafe entry allowed: %v calls=%+v", err, m.calls)
			}
			expectCM5(t, db, "SELECT COUNT(*) FROM coin_m_managed_orders", 0)
		})
	}
}
func TestCM5ProtectionOwnedAndNoDuplicate(t *testing.T) {
	db := cm5Fixture(t)
	m := newMockCM5()
	p := NewProcessor(db, m)
	ctx := context.Background()
	order, err := p.Open(ctx, OpenRequest{Symbol: "BTCUSD_PERP", PositionSide: "LONG", ClientOrderID: "cm5protectentry", Type: "MARKET"})
	if err != nil {
		t.Fatal(err)
	}
	m.qty["LONG"] = "10"
	entry := sampleTrade(m.orders[order.ClientOrderID], 401, "10", "50000", "0")
	if _, err = p.ApplyFill(entry); err != nil {
		t.Fatal(err)
	}
	if _, err = p.PlaceProtection(ctx, ProtectionRequest{PositionID: order.PositionID, ClientAlgoID: "cm5badtrig", Type: "STOP_MARKET", TriggerPrice: "55000"}); err == nil {
		t.Fatal("STOP_MARKET on wrong side of mark accepted")
	}
	stop, err := p.PlaceProtection(ctx, ProtectionRequest{PositionID: order.PositionID, ClientAlgoID: "cm5protect001", Type: "STOP_MARKET", TriggerPrice: "45000"})
	if err != nil || stop.Intent != "PROTECT" || stop.Status != "ACTIVE" || stop.OrderType != "STOP_MARKET" || len(m.openAlgos) != 1 {
		t.Fatalf("managed protective Algo not persisted: %+v %v", stop, err)
	}
	_, err = p.Close(ctx, CloseRequest{PositionID: order.PositionID, ClientOrderID: "cm5closeprotected", Type: "MARKET"})
	if err == nil {
		t.Fatal("normal close should not leave active exchange protection behind")
	}
	repeat, err := p.PlaceProtection(ctx, ProtectionRequest{PositionID: order.PositionID, ClientAlgoID: "cm5protect001", Type: "STOP_MARKET", TriggerPrice: "45000"})
	if err != nil || repeat.ID != stop.ID || len(m.openAlgos) != 1 {
		t.Fatalf("duplicate algo created: %+v %v", repeat, err)
	}
	if _, err = p.CancelProtection(ctx, "cm5protect001"); err != nil {
		t.Fatal(err)
	}
	if _, err = p.Close(ctx, CloseRequest{PositionID: order.PositionID, ClientOrderID: "cm5closeafterstop", Type: "MARKET"}); err != nil {
		t.Fatalf("owned position should close after cancellation of owned protection: %v", err)
	}
}

func TestCM5ShortOpenCloseAndCrossProductFillRejected(t *testing.T) {
	db := cm5Fixture(t)
	api := newMockCM5()
	p := NewProcessor(db, api)
	ctx := context.Background()
	opening, err := p.Open(ctx, OpenRequest{Symbol: "BTCUSD_PERP", PositionSide: "SHORT", ClientOrderID: "cm5shortentry", Type: "MARKET"})
	if err != nil || opening.Side != "SELL" {
		t.Fatalf("SHORT open must SELL contracts: %+v %v", opening, err)
	}
	api.qty["SHORT"] = "10"
	ack := api.orders["cm5shortentry"]
	wrong := sampleTrade(ack, 701, "10", "50000", "0")
	wrong.PositionSide = "LONG"
	if _, err = p.ApplyFill(wrong); err == nil {
		t.Fatal("cross-hedge Fill was incorrectly claimed")
	}
	wrong = sampleTrade(ack, 702, "10", "50000", "0")
	wrong.Symbol = "ETHUSD_PERP"
	if _, err = p.ApplyFill(wrong); err == nil {
		t.Fatal("cross-contract Fill was incorrectly claimed")
	}
	expectCM5(t, db, "SELECT COUNT(*) FROM coin_m_trade_fills", 0)
	if ok, err := p.ApplyFill(sampleTrade(ack, 703, "10", "50000", "0")); err != nil || !ok {
		t.Fatalf("short open Fill: %v %v", ok, err)
	}
	pos, err := loadPosition(db, opening.PositionID)
	if err != nil || pos.Status != "OPEN" || pos.PositionSide != "SHORT" {
		t.Fatalf("short cycle state: %+v %v", pos, err)
	}
	closing, err := p.Close(ctx, CloseRequest{PositionID: opening.PositionID, ClientOrderID: "cm5shortclose", Type: "MARKET"})
	if err != nil || closing.Side != "BUY" || closing.RequestedContracts != "10" {
		t.Fatalf("SHORT close must BUY exact owned contracts: %+v %v", closing, err)
	}
	finish := sampleTrade(api.orders["cm5shortclose"], 704, "10", "45000", "0.002222222222222222")
	if ok, err := p.ApplyFill(finish); err != nil || !ok {
		t.Fatalf("short close Fill: %v %v", ok, err)
	}
	pos, err = loadPosition(db, opening.PositionID)
	if err != nil || pos.Status != "CLOSED" || pos.RealizedPnL != "0.002222222222222222" {
		t.Fatalf("verified SHORT inverse realized PnL incorrectly recorded: %+v %v", pos, err)
	}
}
func TestCM5ExchangePartialFillReconciliationRequiresCompleteEvidence(t *testing.T) {
	db := cm5Fixture(t)
	api := newMockCM5()
	p := NewProcessor(db, api)
	ctx := context.Background()
	opening, err := p.Open(ctx, OpenRequest{Symbol: "BTCUSD_PERP", PositionSide: "LONG", ClientOrderID: "cm5evidence", Type: "MARKET"})
	if err != nil {
		t.Fatal(err)
	}
	ack := api.orders[opening.ClientOrderID]
	partial := sampleTrade(ack, 801, "4", "50000", "0")
	ack.FilledContracts = "5"
	ack.Status = "PARTIALLY_FILLED"
	api.trades[ack.ExchangeOrderID] = []binance.CoinMTrade{partial}
	local, err := p.Reconcile(ctx, opening.ClientOrderID)
	if !errors.Is(err, ErrUncertain) || local.Status != "AWAITING_FILLS" {
		t.Fatalf("remote filled=5 but local Fill=4 must not reconcile: %+v %v", local, err)
	}
	expectCM5(t, db, "SELECT COUNT(*) FROM coin_m_trade_fills", 1)
	rest := sampleTrade(ack, 802, "1", "50000", "0")
	api.trades[ack.ExchangeOrderID] = []binance.CoinMTrade{partial, rest}
	local, err = p.Reconcile(ctx, opening.ClientOrderID)
	if err != nil || local.FilledContracts != "5" {
		t.Fatalf("second REST verified Fill did not repair partial state: %+v %v", local, err)
	}
	expectCM5(t, db, "SELECT COUNT(*) FROM coin_m_trade_fills", 2)
}
