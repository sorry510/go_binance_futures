package binance

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/adshao/go-binance/v2"
	"github.com/beego/beego/v2/client/orm"
	_ "github.com/mattn/go-sqlite3"
	"github.com/shopspring/decimal"
	"go_binance_futures/service/spotexecution"
	"go_binance_futures/service/spotledger"
)

// Live TESTNET only: disabled unless SPOT_QUANT_TESTNET_REAL_E2E=1 is set
// explicitly. Uses a separate sqlite DB under t.TempDir. Absolutely NO
// Mainnet API keys or user database connections are used.
func spotLiveTestLedger(t *testing.T) (*spotledger.Ledger, orm.Ormer) {
	t.Helper()
	_ = orm.RegisterDriver("sqlite3", orm.DRSqlite)
	alias := fmt.Sprintf("sq_testnet_0674_%d", time.Now().UnixNano())
	if err := orm.RegisterDataBase(alias, "sqlite3", filepath.Join(t.TempDir(), "spot_testnet_e2e.db")); err != nil {
		t.Fatal(err)
	}
	db := orm.NewOrmUsingDB(alias)
	for _, sql := range []string{
		`CREATE TABLE config(id INTEGER PRIMARY KEY,spot_quant_enable TINYINT NOT NULL)`,
		`INSERT INTO config(id,spot_quant_enable) VALUES(1,1)`,
		`CREATE TABLE spot_symbols(id INTEGER PRIMARY KEY,symbol VARCHAR(32) UNIQUE NOT NULL,spot_quant_enabled TINYINT NOT NULL)`,
		`INSERT INTO spot_symbols(id,symbol,spot_quant_enabled) VALUES(1,'BTCUSDT',1)`,
		`CREATE TABLE spot_managed_positions (
      id INTEGER PRIMARY KEY AUTOINCREMENT,symbol VARCHAR(32) NOT NULL,
      base_asset VARCHAR(32) NOT NULL,quote_asset VARCHAR(32) NOT NULL,
      bought_qty DECIMAL(36,18) NOT NULL DEFAULT 0,
      remaining_qty DECIMAL(36,18) NOT NULL DEFAULT 0,
      avg_cost_quote DECIMAL(36,18) NOT NULL DEFAULT 0,
      realized_pnl DECIMAL(36,18) NOT NULL DEFAULT 0,
      status VARCHAR(32) NOT NULL,created_at BIGINT NOT NULL,
      updated_at BIGINT NOT NULL,closed_at BIGINT NOT NULL DEFAULT 0
  )`,
		`CREATE TABLE spot_managed_orders (
      id INTEGER PRIMARY KEY AUTOINCREMENT,position_id BIGINT NOT NULL,
      symbol VARCHAR(32) NOT NULL,side VARCHAR(4) NOT NULL,
      order_type VARCHAR(24) NOT NULL,client_order_id VARCHAR(64) NOT NULL UNIQUE,
      exchange_order_id VARCHAR(64) NULL,
      requested_qty DECIMAL(36,18) NOT NULL,filled_qty DECIMAL(36,18) NOT NULL DEFAULT 0,
      status VARCHAR(32) NOT NULL,created_at BIGINT NOT NULL,updated_at BIGINT NOT NULL,
      CONSTRAINT uex UNIQUE(symbol,exchange_order_id)
  )`,
		`CREATE TABLE spot_trade_fills (
      id INTEGER PRIMARY KEY AUTOINCREMENT,position_id BIGINT NOT NULL,
      order_id BIGINT NOT NULL,exchange_order_id VARCHAR(64) NOT NULL,
      trade_id VARCHAR(64) NOT NULL,symbol VARCHAR(32) NOT NULL,
      side VARCHAR(4) NOT NULL,qty DECIMAL(36,18) NOT NULL,
      price DECIMAL(36,18) NOT NULL,quote_qty DECIMAL(36,18) NOT NULL,
      commission DECIMAL(36,18) NOT NULL,commission_asset VARCHAR(32) NOT NULL,
      trade_time BIGINT NOT NULL,CONSTRAINT ufill UNIQUE(symbol,exchange_order_id,trade_id)
  )`,
	} {
		if _, err := db.Raw(sql).Exec(); err != nil {
			t.Fatal(err)
		}
	}
	ledger, err := spotledger.New(db, "sqlite")
	if err != nil {
		t.Fatal(err)
	}
	return ledger, db
}

func TestSpotQuantLiveTestnetBUYSELLAndWS(t *testing.T) {
	if os.Getenv("SPOT_QUANT_TESTNET_REAL_E2E") != "1" {
		t.Skip("Testnet order mutation requires explicit SPOT_QUANT_TESTNET_REAL_E2E=1")
	}
	// Always create a read-only client from the THREE Spot Testnet fields.
	readOnly, err := NewSpotQuantTestnetAdapterFromConfig()
	if err != nil {
		t.Fatal(err)
	}
	if readOnly == nil || readOnly.c == nil || readOnly.c.BaseURL != spotQuantTestnetRESTURL || readOnly.allowOrderMutation {
		t.Fatal("unsafe Spot Testnet client; no order allowed")
	}
	// This order-capable instance exists ONLY in this opt-in test process.
	ex, err := NewSpotQuantAdapterWithClient(readOnly.c, true)
	if err != nil {
		t.Fatal(err)
	}
	if ex.c.BaseURL != spotQuantTestnetRESTURL || !ex.allowOrderMutation {
		t.Fatal("Spot order mutation is not exclusively pinned to Testnet")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Second)
	defer cancel()
	if _, err := ex.c.NewSetServerTimeService().Do(ctx); err != nil {
		t.Fatalf("Testnet time GET: %v", err)
	}
	rules, err := ex.Rules(ctx, "BTCUSDT")
	if err != nil {
		t.Fatalf("Testnet trading rules: %v", err)
	}
	price, err := ex.Price(ctx, "BTCUSDT")
	if err != nil {
		t.Fatal(err)
	}
	freeQuote, err := ex.Free(ctx, "USDT")
	if err != nil {
		t.Fatal(err)
	}
	budget := decimal.NewFromInt(20)
	if !rules.Trading || !rules.MarketAllowed || rules.Base != "BTC" || rules.Quote != "USDT" ||
		decimal.RequireFromString(freeQuote).LessThan(decimal.NewFromInt(30)) {
		t.Fatalf("Testnet BTC/USDT trading rules or mock-fund balance insufficient; no order posted")
	}
	step, err := decimal.NewFromString(rules.Step)
	if err != nil || !step.IsPositive() {
		t.Fatal("missing Spot Testnet step")
	}
	if decimal.RequireFromString(price).IsZero() {
		t.Fatal("invalid Testnet BTCUSDT price")
	}
	ledger, db := spotLiveTestLedger(t)
	p, err := spotexecution.NewProcessor(ex, spotexecution.DBGate{DB: db}, ledger)
	if err != nil {
		t.Fatal(err)
	}

	// Subscribe using the ACTUAL project's signed Testnet WS user stream.
	streamCtx, stopStream := context.WithCancel(context.Background())
	streamReady := make(chan struct{}, 1)
	streamResults := make(chan error, 1)
	events := make(chan binance.WsOrderUpdate, 64)
	go func() {
		streamResults <- ex.runSpotQuantUserStream(streamCtx, p,
			func() {
				select {
				case streamReady <- struct{}{}:
				default:
				}
			},
			func(e binance.WsOrderUpdate) {
				select {
				case events <- e:
				default:
				}
			})
	}()
	defer func() {
		stopStream()
		select {
		case <-streamResults:
		case <-time.After(3 * time.Second):
		}
	}()
	select {
	case <-streamReady:
		t.Log("SPOT_TESTNET_SIGNED_WS_SUBSCRIPTION_ACK=PASS")
	case e := <-streamResults:
		t.Fatalf("signed Spot Testnet WS subscription failed BEFORE any order: %v", e)
	case <-time.After(20 * time.Second):
		t.Fatal("Spot Testnet WS subscription did not ACK; no orders sent")
	}
	// At most one BUY/SELL, with unique IDs. On unknown outcomes the
	// Processor performs read-only reconciliation, never repeat a POST.
	ts := time.Now().UnixNano()
	buyCID := fmt.Sprintf("sqTBuy%x", ts)
	sellCID := fmt.Sprintf("sqTSell%x", ts)
	t.Logf("TESTNET_ORDER_INTENT symbol=BTCUSDT max_budget_USDT=%s buy_client=%s", budget, buyCID)
	buy, err := p.ExecuteBuy(ctx, "BTCUSDT", budget.String(), buyCID)
	if err != nil {
		t.Fatalf("TESTNET_BUY_ERROR client=%s err=%v; do not repeat Testnet POST without checking Binance order", buyCID, err)
	}
	t.Logf("TESTNET_BUY_LOCAL status=%s position=%d requested_qty=%s client=%s", buy.Status, buy.PositionID, buy.RequestedQty, buyCID)
	// Market FILLED may be acknowledged before Testnet myTrades becomes visible.
	for i := 0; i < 4; i++ {
		snapshot, e := ledger.Accounting(buy.PositionID, "")
		if e == nil && snapshot.Position.Status == "OPEN" {
			break
		}
		if _, e = p.Reconcile(ctx, buy.ID); e != nil && i == 3 {
			t.Fatalf("TESTNET_BUY_RECONCILIATION_REQUIRED client=%s: %v", buyCID, e)
		}
		time.Sleep(time.Second)
	}
	open, err := ledger.Accounting(buy.PositionID, price)
	if err != nil || open.Position.Status != "OPEN" {
		t.Fatalf("real Testnet BUY Fill not yet fully reconciled; client=%s status=%s err=%v", buyCID, open.Position.Status, err)
	}
	buyFills, err := ledger.ListFills(buy.PositionID)
	if err != nil || len(buyFills) == 0 {
		t.Fatalf("real Testnet BUY has no myTrades Fill: %v", err)
	}
	t.Logf("TESTNET_BUY_VERIFIED fills=%d owned_qty=%s fee_assets=%v", len(buyFills), open.Position.RemainingQty, open.UnpricedFees)

	// First try the full real Processor close, no manual-wallet asset claims.
	sell, err := p.ExecuteSell(ctx, buy.PositionID, sellCID)
	if err != nil && errors.Is(err, spotexecution.ErrNeedsReview) &&
		strings.Contains(err.Error(), "strategy residual cannot be fully sold") {
		// Base-asset commission leaves untradeable dust. On Testnet ONLY,
		// submit a single safe floor-stepped SELL strictly BELOW verified
		// strategy-owned inventory; never use existing manual wallet BTC.
		owned := decimal.RequireFromString(open.Position.RemainingQty)
		floor := owned.Div(step).Floor().Mul(step)
		minN, _ := decimal.NewFromString(rules.MinNotional)
		nowPrice, e := ex.Price(ctx, "BTCUSDT")
		if e != nil || !floor.IsPositive() || floor.GreaterThan(owned) ||
			floor.Mul(decimal.RequireFromString(nowPrice)).LessThan(minN) {
			t.Fatalf("cannot safely close Testnet base-fee dust: owned=%s floor=%s: %v", owned, floor, e)
		}
		t.Logf("TESTNET_BASE_FEE_DUST: strategy_owned=%s sell_floor=%s; manual wallet is NOT sold", owned, floor)
		local, duplicate, e := ledger.CreateOrder(spotledger.OrderIntent{
			PositionID: buy.PositionID, Symbol: "BTCUSDT", Side: "SELL", Type: "MARKET",
			ClientOrderID: sellCID, RequestedQty: floor.String(),
		})
		if e != nil || duplicate {
			t.Fatalf("Testnet dust SELL order intent error=%v duplicate=%v", e, duplicate)
		}
		_, e = ex.Place(ctx, spotexecution.Request{
			Symbol: "BTCUSDT", Side: "SELL", Type: "MARKET", ClientOrderID: sellCID, Quantity: floor.String(),
		})
		if e != nil {
			t.Fatalf("TESTNET_DUST_SELL_UNCERTAIN client=%s err=%v; never repeat POST", sellCID, e)
		}
		sell, err = p.Reconcile(ctx, local.ID)
	}
	if err != nil {
		t.Fatalf("TESTNET_SELL_ERROR client=%s err=%v; no duplicate SELL should be posted", sellCID, err)
	}
	t.Logf("TESTNET_SELL_LOCAL status=%s requested_qty=%s client=%s", sell.Status, sell.RequestedQty, sellCID)
	for i := 0; i < 4; i++ {
		if _, err = p.Reconcile(ctx, sell.ID); err == nil {
			break
		}
		time.Sleep(time.Second)
	}
	closed, err := ledger.Accounting(buy.PositionID, price)
	if err != nil {
		t.Fatalf("Testnet final Fill projection: %v", err)
	}
	fills, err := ledger.ListFills(buy.PositionID)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("TESTNET_END_TO_END_RESULT local_status=%s remaining_qty=%s filled_events=%d", closed.Position.Status, closed.Position.RemainingQty, len(fills))
	if len(fills) < 2 {
		t.Fatal("missing real BUY/SELL myTrades")
	}
	if closed.Position.Status != "CLOSED" {
		t.Logf("TESTNET_DUST_REMAINS: status=%s owned_remaining=%s, no manual BTC used", closed.Position.Status, closed.Position.RemainingQty)
	}
	// Compare actual WS push events from signed live subscription for both
	// unique Testnet client orders. Do not retry any order if a WS push lags.
	seenBuy, seenSell := false, false
	deadline := time.NewTimer(20 * time.Second)
	defer deadline.Stop()
	for !(seenBuy && seenSell) {
		select {
		case event := <-events:
			id := event.ClientOrderId
			if event.OrigCustomOrderId != "" {
				id = event.OrigCustomOrderId
			}
			if id == buyCID {
				seenBuy = true
			}
			if id == sellCID {
				seenSell = true
			}
		case streamErr := <-streamResults:
			t.Fatalf("Testnet WS disconnected during E2E: %v", streamErr)
		case <-deadline.C:
			t.Fatalf("Testnet orders executed but signed WS executionReport incomplete: buy_seen=%t sell_seen=%t", seenBuy, seenSell)
		}
	}
	t.Logf("SPOT_TESTNET_WS_EXECUTION_REPORTS=PASS buy_seen=%t sell_seen=%t", seenBuy, seenSell)
}

func TestSpotQuantTestnetReadBackExecutedOrders(t *testing.T) {
	raw := os.Getenv("SPOT_QUANT_TESTNET_VERIFY_CLIENT_IDS")
	if raw == "" {
		t.Skip("read-only reconciliation requires explicit client IDs")
	}
	ids := strings.Split(raw, ",")
	if len(ids) != 2 || !strings.HasPrefix(ids[0], "sqTBuy") || !strings.HasPrefix(ids[1], "sqTSell") {
		t.Fatal("expected precisely one Spot Testnet BUY and SELL client ID")
	}
	a, err := NewSpotQuantTestnetAdapterFromConfig()
	if err != nil {
		t.Fatal(err)
	}
	if a.allowOrderMutation || a.c.BaseURL != spotQuantTestnetRESTURL {
		t.Fatal("readback must use read-only Spot Testnet client")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	if _, err := a.c.NewSetServerTimeService().Do(ctx); err != nil {
		t.Fatal(err)
	}
	for i, id := range ids {
		wantSide := "BUY"
		if i == 1 {
			wantSide = "SELL"
		}
		remote, err := a.FindOrder(ctx, "BTCUSDT", id)
		if err != nil || remote == nil {
			t.Fatalf("remote %s readback: %v", wantSide, err)
		}
		if remote.Side != wantSide || remote.Status != "FILLED" || remote.ClientOrderID != id {
			t.Fatalf("remote %s identity/status mismatch: %+v", wantSide, remote)
		}
		trades, err := a.Trades(ctx, "BTCUSDT", remote.ExchangeOrderID)
		if err != nil || len(trades) == 0 {
			t.Fatalf("remote %s myTrades not confirmed: %v", wantSide, err)
		}
		sumQty, sumQuote := decimal.Zero, decimal.Zero
		fees := map[string]decimal.Decimal{}
		for _, tr := range trades {
			if tr.Side != wantSide || tr.Symbol != "BTCUSDT" || tr.OrderID != remote.ExchangeOrderID {
				t.Fatalf("remote %s trade ownership mismatch", wantSide)
			}
			sumQty = sumQty.Add(decimal.RequireFromString(tr.Qty))
			sumQuote = sumQuote.Add(decimal.RequireFromString(tr.QuoteQty))
			fee := decimal.RequireFromString(tr.Fee)
			fees[tr.FeeAsset] = fees[tr.FeeAsset].Add(fee)
		}
		if !sumQty.Equal(decimal.RequireFromString(remote.ExecutedQty)) {
			t.Fatalf("remote %s fills sum %s != executed %s", wantSide, sumQty, remote.ExecutedQty)
		}
		t.Logf("TESTNET_READBACK_%s: orderID=%s status=%s qty=%s quote=%s fills=%d fees=%v",
			wantSide, remote.ExchangeOrderID, remote.Status, sumQty, sumQuote, len(trades), fees)
	}
}

func TestSpotQuantTestnetMarketPreflight(t *testing.T) {
	if os.Getenv("SPOT_QUANT_TESTNET_MARKET_PREFLIGHT") != "1" {
		t.Skip("read-only Testnet market preflight requires explicit opt-in")
	}
	a, err := NewSpotQuantTestnetAdapterFromConfig()
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	ticker, err := a.MarketSnapshot(ctx, "BTCUSDT")
	if err != nil {
		t.Fatalf("Spot Testnet 24h ticker not usable: %v", err)
	}
	if ticker.Symbol != "BTCUSDT" || ticker.Price == "" || ticker.Open == "" || ticker.High == "" || ticker.Low == "" {
		t.Fatalf("missing Spot Testnet ticker fields: %+v", ticker)
	}
	klines, err := a.MarketKlines(ctx, "BTCUSDT", "5m", 10)
	if err != nil {
		t.Fatalf("Spot Testnet 5m Kline unavailable: %v", err)
	}
	if len(klines) < 5 || klines[len(klines)-1].OpenTime <= 0 {
		t.Fatal("Spot Testnet Kline history has insufficient bars")
	}
	threeDay, err := a.MarketKlines(ctx, "BTCUSDT", "3d", 2)
	if err != nil || len(threeDay) == 0 {
		t.Fatalf("Spot Testnet 3d official opening missing: %v", err)
	}
	t.Logf("TESTNET_MARKET_PREFLIGHT_OK ticker=%s now_price=%s 5m_bars=%d 3d_open_utc=%s",
		ticker.Symbol, ticker.Price, len(klines), time.UnixMilli(threeDay[len(threeDay)-1].OpenTime).UTC().Format(time.RFC3339))
}
