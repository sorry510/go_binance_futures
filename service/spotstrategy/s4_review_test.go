package spotstrategy

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/beego/beego/v2/client/orm"
	"go_binance_futures/models"
)

func TestS4F3LiveDisplayComesFromExecutionGate(t *testing.T) {
	setupSpotS1(t)
	t.Setenv(liveApprovalVariable, liveApprovalValue)
	for _, mode := range []string{ModeLegacy, ModeOff, ModeStrategy} {
		execution, approved, reason := LiveStatus(mode)
		if approved || execution != "live_disabled" || reason == "" {
			t.Fatalf("display diverged from funds release gate mode=%s approval=%v execution=%s reason=%s", mode, approved, execution, reason)
		}
	}
	if LiveOrdersApproved() {
		t.Fatal("release gate unexpectedly enabled by environment alone")
	}
	// Account and mode status have independent endpoints. A symbol row must
	// never carry a per-account authorization decision.
	setting, err := LoadSetting("BTCUSDT")
	if err != nil || setting.Symbol != "BTCUSDT" {
		t.Fatalf("independent symbol read failed: %+v %v", setting, err)
	}
}

func TestS4F4ReferencePriceFreshnessAndVerifiedOnlyEstimate(t *testing.T) {
	setupSpotS1(t)
	db, err := orm.GetDB("default")
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UnixMilli()
	_, err = db.Exec("UPDATE spot_symbols SET close='120',lastUpdateTime=? WHERE symbol='BTCUSDT'", now)
	if err != nil {
		t.Fatal(err)
	}
	res, err := db.Exec(`INSERT INTO spot_strategy_holdings
 (account_id,owner,symbol,base_asset,quote_asset,owned_qty,avg_cost_quote,status,active_slot,created_at,updated_at)
 VALUES ('main','auto_strategy','BTCUSDT','BTC','USDT','2','100','active','main:auto_strategy:BTCUSDT',?,?)`, now, now)
	if err != nil {
		t.Fatal(err)
	}
	hid, err := res.LastInsertId()
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`INSERT INTO spot_strategy_fills
 (account_id,symbol,order_id,trade_id,holding_id,client_order_id,side,qty,price,quote_qty,commission,commission_asset,trade_time,rest_verified)
 VALUES ('main','BTCUSDT','900','91',?,'s4-ref','BUY','2','100','200','0','USDT',?,1)`, hid, now)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	got, err := ListHoldings(ctx, db, PageRequest{})
	if err != nil || len(got.Items) != 1 {
		t.Fatalf("fresh holding %+v %v", got, err)
	}
	h := got.Items[0]
	if !h.Complete || !h.ReferencePriceFresh || h.ReferencePriceQuote != "120" ||
		h.ReferencePriceTime != now || h.UnrealizedEstimateQuote != "40" {
		t.Fatalf("verified Holding mark must be 40 only with fresh local BTCUSDT price: %+v", h)
	}
	_, err = db.Exec("UPDATE spot_symbols SET lastUpdateTime=? WHERE symbol='BTCUSDT'", now-120000)
	if err != nil {
		t.Fatal(err)
	}
	got, err = ListHoldings(ctx, db, PageRequest{})
	if err != nil {
		t.Fatal(err)
	}
	h = got.Items[0]
	if h.ReferencePriceFresh || h.ReferencePriceQuote != "" || h.UnrealizedEstimateQuote != "" {
		t.Fatalf("stale quote produced fake profit: %+v", h)
	}
	_, err = db.Exec("UPDATE spot_symbols SET type='USDC',lastUpdateTime=? WHERE symbol='BTCUSDT'", now)
	if err != nil {
		t.Fatal(err)
	}
	got, err = ListHoldings(ctx, db, PageRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if got.Items[0].ReferencePriceFresh || got.Items[0].UnrealizedEstimateQuote != "" {
		t.Fatalf("wrong quote currency cannot price a USDT holding: %+v", got.Items[0])
	}
	_, err = db.Exec("UPDATE spot_symbols SET type='USDT' WHERE symbol='BTCUSDT'")
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec("UPDATE spot_strategy_fills SET rest_verified=0 WHERE holding_id=?", hid)
	if err != nil {
		t.Fatal(err)
	}
	got, err = ListHoldings(ctx, db, PageRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if got.Items[0].Complete || got.Items[0].UnrealizedEstimateQuote != "" || got.Items[0].OwnedQty != "" {
		t.Fatalf("unverified fills used in valuation: %+v", got.Items[0])
	}
}

func TestS4F5ReadOnlyEngineCannotSettle(t *testing.T) {
	setupSpotS1(t)
	db, err := orm.GetDB("default")
	if err != nil {
		t.Fatal(err)
	}
	e := NewReadOnlyEngine(db, nil)
	_, err = e.ApplyVerifiedFills(context.Background(), 1)
	if !errors.Is(err, ErrLiveWriteDisabled) {
		t.Fatalf("nil client should fail before dereference: %v", err)
	}
}

func TestS4F6UnsupportedOrInconsistentQuotesNotEditable(t *testing.T) {
	setupSpotS1(t)
	o := orm.NewOrm()
	for _, v := range []models.SpotSymbols{
		{Symbol: "OLDABC", Type: "BUSD", Technology: "{}", Strategy: "[]", StrategyBudgetQuote: "0"},
		{Symbol: "WRONGUSDT", Type: "USDC", Technology: "{}", Strategy: "[]", StrategyBudgetQuote: "0"},
		{Symbol: "ETHUSDC", Type: "USDC", Technology: "{}", Strategy: "[]", StrategyBudgetQuote: "0"},
	} {
		if _, err := o.Insert(&v); err != nil {
			t.Fatal(err)
		}
	}
	db, err := orm.GetDB("default")
	if err != nil {
		t.Fatal(err)
	}
	got, err := ListSettings(context.Background(), db, PageRequest{})
	if err != nil || got.Total != 2 || len(got.Items) != 2 {
		t.Fatalf("editable symbol filter got %+v err=%v", got, err)
	}
	for _, item := range got.Items {
		if item.Symbol != "BTCUSDT" && item.Symbol != "ETHUSDC" {
			t.Fatalf("unsupported quote exposed as editable: %+v", item)
		}
	}
	if _, err := ListHoldings(context.Background(), db, PageRequest{Status: "incomplete"}); !errors.Is(err, ErrInvalidS4Filter) {
		t.Fatalf("incomplete is completeness flag, never DB status: %v", err)
	}
}

// F1: The actual S2/S3 Mock runtime path can store local order/trade records,
// but cannot attest a Binance signed REST response. A Holding can only appear
// in isolated tests that explicitly inject verified evidence; this test
// permanently protects the truthful S4 empty-state expectation.
func TestS4F1RuntimeUnverifiedTradeNeverCreatesHolding(t *testing.T) {
	e, f := prepareS2(t)
	buy := mustPrepareSubmitS2(t, e, "s4-r1-no-verified-source")
	f.Lock()
	f.trades = []Trade{tradeS2(91, "1", "100", "100", "0", "USDT")}
	f.order.Status = "CANCELED"
	f.order.ExecutedQty = "1"
	f.order.ExecutedQuoteQty = "100"
	f.Unlock()
	got, err := e.Reconcile(context.Background(), buy.ID)
	if err != nil || got.FilledQty != "1" {
		t.Fatalf("local trade reconciliation failed %+v %v", got, err)
	}
	db, err := orm.GetDB("default")
	if err != nil {
		t.Fatal(err)
	}
	var verified int
	if err = db.QueryRow("SELECT COUNT(*) FROM spot_strategy_fills WHERE rest_verified=1").Scan(&verified); err != nil {
		t.Fatal(err)
	}
	if verified != 0 {
		t.Fatalf("Mock path forged %d verified fills", verified)
	}
	if _, err = e.RecoverAndSettle(context.Background(), 10); err != nil {
		t.Fatalf("read-only recovery %v", err)
	}
	list, err := ListHoldings(context.Background(), db, PageRequest{})
	if err != nil || list.Total != 0 || len(list.Items) != 0 {
		t.Fatalf("unauthenticated Fill created a Holding: %+v %v", list, err)
	}
}
