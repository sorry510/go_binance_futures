package spotstrategy

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/beego/beego/v2/client/orm"
)

// S3 exercises production accounting with signed-Fill test records in an
// isolated SQLite database. This is not a separate Mock trading product, and
// no Binance request or live account is used during tests.
func TestS3VerifiedBuyHoldingPartialFeeAndIdempotency(t *testing.T) {
	e, f := prepareS2(t)
	buy := mustPrepareSubmitS2(t, e, "s3-real-buy-accounting")
	f.Lock()
	f.trades = []Trade{
		tradeS2(1, "2", "100", "200", "0.01", "BTC"),
		tradeS2(2, "2", "120", "240", "0.2", "USDT"),
	}
	f.order.Status = "CANCELED"
	f.order.ExecutedQty = "4"
	f.order.ExecutedQuoteQty = "440"
	f.Unlock()
	canceled, err := e.Reconcile(context.Background(), buy.ID)
	if err != nil || canceled.Status != StateCanceled {
		t.Fatalf("read signed trade fixture: %+v %v", canceled, err)
	}
	if _, err = e.ApplyVerifiedFills(context.Background(), buy.ID); !errors.Is(err, ErrReconciliation) {
		t.Fatalf("unverified S2 mock fill was applied to inventory: %v", err)
	}
	if countTableS2(t, "spot_strategy_holdings") != 0 {
		t.Fatal("created inventory from unverified fill")
	}
	db, err := orm.GetDB("default")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec("UPDATE spot_strategy_fills SET rest_verified=1 WHERE client_order_id=?", buy.ClientOrderID); err != nil {
		t.Fatal(err)
	}
	h, err := e.ApplyVerifiedFills(context.Background(), buy.ID)
	if err != nil {
		t.Fatalf("verified BUY settlement: %v", err)
	}
	if h.ID < 1 || h.Status != "active" || h.OwnedQty != "3.99" || h.AvgCostQuote != "110.32581453634085213" {
		// 440.2 / 3.99 = 110.325814536340852130
		t.Fatalf("actual base/quote fee inventory: %+v", h)
	}
	hAgain, err := e.ApplyVerifiedFills(context.Background(), buy.ID)
	if err != nil || hAgain.ID != h.ID || hAgain.OwnedQty != "3.99" {
		t.Fatalf("replayed Fill duplicated inventory: %+v %v", hAgain, err)
	}
	if _, err = e.Prepare(context.Background(), requestS2("s3-no-add-to-active")); !errors.Is(err, ErrReconciliation) {
		t.Fatalf("BUY added to an active cycle: %v", err)
	}
	report, err := e.HoldingReport(context.Background(), h.ID, "125")
	if err != nil || !report.Complete || report.Fees["BTC"] != "0.01" || report.Fees["USDT"] != "0.2" ||
		report.CostBasisQuote != "440.2" {
		t.Fatalf("quote cost and fee report %+v %v", report, err)
	}
}

func TestS3SellBelongsToHoldingNotAccountFree(t *testing.T) {
	e, f := prepareS2(t)
	buy := mustPrepareSubmitS2(t, e, "s3-exit-source")
	f.Lock()
	f.trades = []Trade{tradeS2(7, "4", "100", "400", "0.01", "BNB")}
	f.order.Status = "CANCELED"
	f.order.ExecutedQty = "4"
	f.order.ExecutedQuoteQty = "400"
	f.Unlock()
	if _, err := e.Reconcile(context.Background(), buy.ID); err != nil {
		t.Fatal(err)
	}
	db, _ := orm.GetDB("default")
	if _, err := db.Exec("UPDATE spot_strategy_fills SET rest_verified=1"); err != nil {
		t.Fatal(err)
	}
	h, err := e.ApplyVerifiedFills(context.Background(), buy.ID)
	if err != nil || h.OwnedQty != "4" {
		t.Fatalf("holding %+v %v", h, err)
	}
	f.baseFree = "1" // strategy owns 4, physical free is now insufficient
	req := requestS2("s3-close")
	req.Side = "SELL"
	req.Quantity = "4"
	req.Type = "LIMIT"
	req.Price = "125"
	if _, err = e.PrepareExit(context.Background(), req); !errors.Is(err, ErrReconciliation) {
		t.Fatalf("physical Spot free below strategy inventory must block SELL: %v", err)
	}
	f.baseFree = "20" // 20 physical BTC, but only 4 belong to the strategy
	sell, err := e.PrepareExit(context.Background(), req)
	if err != nil {
		t.Fatalf("prepare only strategy SELL: %v", err)
	}
	if sell.RequestedQty != "4" || sell.ReservedQty != "4" {
		t.Fatalf("used account-wide 20 BTC: %+v", sell)
	}
	if _, err = e.PrepareExit(context.Background(), req); err == nil {
		t.Fatal("duplicate SELL reserved")
	}
	// Simulate the real REST snapshot/status/Fill data as an isolated fixture.
	if _, err = db.Exec(`UPDATE spot_strategy_orders SET status='partially_filled',exchange_order_id='902',
 filled_qty='1',executed_quote_qty='125' WHERE id=?`, sell.ID); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UnixMilli()
	_, err = db.Exec(`INSERT INTO spot_strategy_fills
 (account_id,symbol,order_id,trade_id,holding_id,client_order_id,side,qty,price,quote_qty,commission,commission_asset,trade_time,rest_verified)
 VALUES (?,?,?,?,NULL,?,'SELL','1','125','125','0.125','USDT',?,1)`,
		AccountMain, req.Symbol, "902", "88", sell.ClientOrderID, now)
	if err != nil {
		t.Fatal(err)
	}
	partial, err := e.ApplyVerifiedFills(context.Background(), sell.ID)
	if err != nil || partial.Status != "active" || partial.OwnedQty != "3" {
		t.Fatalf("partial close %+v %v", partial, err)
	}
	if _, err = db.Exec(`INSERT INTO spot_strategy_fills
 (account_id,symbol,order_id,trade_id,holding_id,client_order_id,side,qty,price,quote_qty,commission,commission_asset,trade_time,rest_verified)
 VALUES (?,?,?,?,NULL,?,'SELL','3','125','375','0.375','USDT',?,1)`,
		AccountMain, req.Symbol, "902", "89", sell.ClientOrderID, now+1); err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec("UPDATE spot_strategy_orders SET status='filled',pending_slot=NULL WHERE id=?", sell.ID); err != nil {
		t.Fatal(err)
	}
	closed, err := e.ApplyVerifiedFills(context.Background(), sell.ID)
	if err != nil || closed.Status != "closed" || closed.OwnedQty != "0" {
		t.Fatalf("full holding close %+v %v", closed, err)
	}
	summary, err := e.HoldingReport(context.Background(), h.ID, "")
	if err != nil || summary.RealizedGrossQuote != "100" || summary.Complete || summary.IncompleteReason != "third_asset_fee_requires_rate_at_trade_time" {
		t.Fatalf("report should keep BNB fee incomplete: %+v %v", summary, err)
	}
	if summary.RealizedNetQuote != "" {
		t.Fatalf("third fee converted into fake USDT: %+v", summary)
	}
	if _, err = e.GetActiveHolding(context.Background(), req.Symbol); !errors.Is(err, ErrUnmanagedOrder) {
		t.Fatalf("closed Holding still active %v", err)
	}
	// The next LONG must be a genuinely new inventory cycle, not a reused
	// holding_id or a merge with the previous BUY/SELL history.
	next, err := e.Prepare(context.Background(), requestS2("s3-next-new-cycle"))
	if err != nil {
		t.Fatalf("new strategy cycle failed after verified close: %v", err)
	}
	if _, err = db.Exec(`UPDATE spot_strategy_orders SET status='canceled',exchange_order_id='903',
	 filled_qty='2',executed_quote_qty='200',reserved_qty='0',pending_slot=NULL WHERE id=?`, next.ID); err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec(`INSERT INTO spot_strategy_fills
	 (account_id,symbol,order_id,trade_id,holding_id,client_order_id,side,qty,price,quote_qty,commission,commission_asset,trade_time,rest_verified)
	 VALUES (?,?,?,?,NULL,?,'BUY','2','100','200','0','USDT',?,1)`,
		AccountMain, next.Symbol, "903", "99", next.ClientOrderID, time.Now().UnixMilli()); err != nil {
		t.Fatal(err)
	}
	another, err := e.ApplyVerifiedFills(context.Background(), next.ID)
	if err != nil || another.ID == h.ID || another.Status != "active" || another.OwnedQty != "2" {
		t.Fatalf("separate new BUY period: old=%+v new=%+v err=%v", h, another, err)
	}
}

func TestS3ActiveSlotCannotSellOtherOwnerOrMissingFree(t *testing.T) {
	e, f := prepareS2(t)
	req := requestS2("bad-sell")
	req.Side = "SELL"
	req.Price = "100"
	if _, err := e.PrepareExit(context.Background(), req); !errors.Is(err, ErrUnmanagedOrder) {
		t.Fatalf("sold without strategy Holding: %v", err)
	}
	f.unmanaged = []OrderSnapshot{{Symbol: "BTCUSDT", Status: "NEW", OrderID: "999"}}
	if _, err := e.PrepareExit(context.Background(), req); err == nil {
		t.Fatal("unowned open orders were ignored")
	}
}

func TestS3ClosedSignalDeterminismAndDirection(t *testing.T) {
	raw := `[{"name":"enter","type":"long","enable":true,"code":"NowPrice > 100"},{"name":"exit","type":"close_long","enable":true,"code":"NowPrice > 100"}]`
	env := map[string]interface{}{"NowPrice": 110.0}
	a, err := EvaluateClosedSignal("BTCUSDT", raw, "{}", 1760000000000, env, false)
	if err != nil || a.Direction != "LONG" || a.SignalKey == "" {
		t.Fatalf("LONG %+v %v", a, err)
	}
	b, err := EvaluateClosedSignal("BTCUSDT", raw, "{}", 1760000000000, env, false)
	if err != nil || a.SignalKey != b.SignalKey {
		t.Fatal("same closed Kline changed client signal ID")
	}
	c, err := EvaluateClosedSignal("BTCUSDT", raw, "{}", 1760000060000, env, false)
	if err != nil || c.SignalKey == a.SignalKey {
		t.Fatal("distinct closed Kline reused SignalKey")
	}
	exit, err := EvaluateClosedSignal("BTCUSDT", raw, "{}", 1760000000000, env, true)
	if err != nil || exit.Direction != "CLOSE_LONG" || exit.SignalKey == a.SignalKey {
		t.Fatalf("CLOSE_LONG %+v %v", exit, err)
	}
}

func TestS3DefaultLiveAuthorizationDenied(t *testing.T) {
	t.Setenv(liveApprovalVariable, "")
	if err := AuthorizeLiveMode(); !errors.Is(err, ErrLiveWriteDisabled) {
		t.Fatalf("live gate open by default: %v", err)
	}
	db, err := orm.GetDB("default")
	if err != nil {
		t.Fatal(err)
	}
	_, err = NewLiveEngine(db, newFakeS2())
	if !errors.Is(err, ErrLiveWriteDisabled) {
		t.Fatalf("untrusted Fake accepted as signed REST: %v", err)
	}
}

func TestS3ForgedHoldingWithoutSignedFillsCannotSell(t *testing.T) {
	e, f := prepareS2(t)
	f.baseFree = "1000"
	db, err := orm.GetDB("default")
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`INSERT INTO spot_strategy_holdings
 (account_id,owner,symbol,base_asset,quote_asset,owned_qty,avg_cost_quote,status,active_slot,created_at,updated_at)
 VALUES ('main','auto_strategy','BTCUSDT','BTC','USDT','10','100','active','main:auto_strategy:BTCUSDT',1,1)`)
	if err != nil {
		t.Fatal(err)
	}
	req := requestS2("forged-sell")
	req.Side = "SELL"
	req.Price = "120"
	if _, err = e.PrepareExit(context.Background(), req); !errors.Is(err, ErrReconciliation) {
		t.Fatalf("forged Holding row without verified linked Fills allowed SELL: %v", err)
	}
	if countTableS2(t, "spot_strategy_orders") != 0 {
		t.Fatal("unowned position generated an order")
	}
}
func TestS3SellDustBelowExchangeMinimumStaysOwned(t *testing.T) {
	e, f := prepareS2(t)
	buy := mustPrepareSubmitS2(t, e, "s3-small-buy")
	f.Lock()
	f.trades = []Trade{tradeS2(121, "0.01", "100", "1", "0", "USDT")}
	f.order.Status = "PARTIALLY_FILLED"
	f.order.ExecutedQty = "0.01"
	f.order.ExecutedQuoteQty = "1"
	f.Unlock()
	if _, err := e.Reconcile(context.Background(), buy.ID); err != nil {
		t.Fatal(err)
	}
	db, _ := orm.GetDB("default")
	if _, err := db.Exec("UPDATE spot_strategy_fills SET rest_verified=1"); err != nil {
		t.Fatal(err)
	}
	h, err := e.ApplyVerifiedFills(context.Background(), buy.ID)
	if err != nil || h.OwnedQty != "0.01" {
		t.Fatalf("buy tiny verified fill: %+v %v", h, err)
	}
	f.baseFree = "0.01"
	// Terminal notional 1 USDT is below the symbol's minNotional=10 and
	// minQty=1; preserving real dust beats artificially rounding to 1 BTC.
	req := requestS2("s3-dust")
	req.Side = "SELL"
	req.Quantity = ""
	req.Price = "100"
	if _, err = e.PrepareExit(context.Background(), req); err == nil {
		t.Fatal("illegal dust SELL was created")
	}
	current, err := e.GetActiveHolding(context.Background(), "BTCUSDT")
	if err != nil || current.OwnedQty != "0.01" {
		t.Fatalf("dust inventory disappeared: %+v %v", current, err)
	}
}

func TestS3ClosedCandleHundredConcurrentTriggersSingleIntent(t *testing.T) {
	e, _ := prepareS2(t)
	db, err := orm.GetDB("default")
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(8)
	const count = 100
	var wg sync.WaitGroup
	success := make(chan struct{}, count)
	for i := 0; i < count; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			req := requestS2("s3-unchanged-closed-candle")
			if _, err := e.Prepare(context.Background(), req); err == nil {
				success <- struct{}{}
			}
		}()
	}
	wg.Wait()
	close(success)
	if len(success) != 1 {
		t.Fatalf("concurrent triggers produced %d orders, want one", len(success))
	}
	if n := countTableS2(t, "spot_strategy_orders"); n != 1 {
		t.Fatalf("repeated signal orders=%d", n)
	}
}
