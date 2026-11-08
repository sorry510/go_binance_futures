package tradefills

import (
	"math/big"
	"path/filepath"
	"sync"
	"testing"

	"go_binance_futures/models"

	"github.com/adshao/go-binance/v2/futures"
	"github.com/beego/beego/v2/client/orm"
	_ "github.com/mattn/go-sqlite3"
)

var reviewRegisterModelsOnce sync.Once

func TestStage012ImmutableDedupAlgoAndAccountScope(t *testing.T) {
	_ = orm.RegisterDriver("sqlite3", orm.DRSqlite)
	if err := orm.RegisterDataBase("default", "sqlite3", filepath.Join(t.TempDir(), "fills.db")); err != nil {
		t.Fatal(err)
	}
	reviewRegisterModelsOnce.Do(func() { orm.RegisterModel(new(models.FuturesManagedPosition), new(models.FuturesManagedOrder)) })
	if err := orm.RunSyncdb("default", false, false); err != nil {
		t.Fatal(err)
	}
	o := orm.NewOrm()
	const create = SchemaDDL
	if _, err := o.Raw(create).Exec(); err != nil {
		t.Fatal(err)
	}
	ws := futures.WsOrderTradeUpdate{
		Symbol: "BTCUSDT", ID: 777, TradeID: 888, ClientOrderID: "child_order",
		ExecutionType: futures.OrderExecutionType("TRADE"),
		PositionSide:  futures.PositionSideTypeLong,
		LastFilledQty: "0.5", LastFilledPrice: "100", Commission: "0.02",
		CommissionAsset: "USDT", RealizedPnL: "0", TradeTime: 1800000000000,
	}
	raw, ok, err := FromWS("main", ws)
	if err != nil || !ok {
		t.Fatalf("WS conversion: %+v %v %v", raw, ok, err)
	}
	for i := 0; i < 100; i++ {
		if err := Insert(o, raw); err != nil {
			t.Fatalf("duplicate %d: %v", i, err)
		}
	}
	var count int
	if err := o.Raw("SELECT COUNT(*) FROM futures_trade_fills").QueryRow(&count); err != nil || count != 1 {
		t.Fatalf("100 duplicate events created %d fills: %v", count, err)
	}
	// Identical exchange order/trade IDs are account-scoped.
	lead := raw
	lead.AccountID = "lead"
	if err := Insert(o, lead); err != nil {
		t.Fatal(err)
	}
	if err := o.Raw("SELECT COUNT(*) FROM futures_trade_fills").QueryRow(&count); err != nil || count != 2 {
		t.Fatalf("cross-account collision: count=%d err=%v", count, err)
	}
	// No matching managed order: price/time/symbol never claim a cycle.
	var orphan int
	if err := o.Raw("SELECT COUNT(*) FROM futures_trade_fills WHERE position_id IS NULL").QueryRow(&orphan); err != nil || orphan != 2 {
		t.Fatalf("unexpected guessed position: %d %v", orphan, err)
	}
	pos := models.FuturesManagedPosition{AccountID: "main", Owner: "agent_trade", Symbol: "BTCUSDT", PositionSide: "LONG", ManagedQty: 0.5, Status: "active", CreatedAt: 1, UpdatedAt: 1}
	pid, err := o.Insert(&pos)
	if err != nil {
		t.Fatal(err)
	}
	child := "777"
	algo := models.FuturesManagedOrder{AccountID: "main", Owner: "agent_trade", Symbol: "BTCUSDT", PositionSide: "LONG", Intent: "take_profit",
		ClientOrderID: "algo_parent", ExchangeOrderID: "123456", ActualOrderID: &child, PositionID: &pid,
		RequestedQty: 0.5, FilledQty: 0.5, Status: "filled", OrderType: "TAKE_PROFIT_MARKET", CreatedAt: 1, UpdatedAt: 1}
	if _, err = o.Insert(&algo); err != nil {
		t.Fatal(err)
	}
	if err = Attach(o, algo); err != nil {
		t.Fatal(err)
	}
	var linked int64
	if err = o.Raw("SELECT position_id FROM futures_trade_fills WHERE account_id='main' AND order_id='777'").QueryRow(&linked); err != nil || linked != pid {
		t.Fatalf("Algo actualOrderId not attached to intended cycle: %d %v", linked, err)
	}
	// The same trade on Lead must remain unattributed.
	if err = o.Raw("SELECT COUNT(*) FROM futures_trade_fills WHERE account_id='lead' AND position_id IS NOT NULL").QueryRow(&count); err != nil || count != 0 {
		t.Fatalf("lead contamination: %d %v", count, err)
	}
	qty, err := Quantity(o, algo)
	if err != nil || qty.Cmp(newRat("1/2")) != 0 {
		t.Fatalf("quantity=%v err=%v", qty, err)
	}
	rest := &futures.AccountTrade{ID: 888, OrderID: 777, Symbol: "BTCUSDT", Quantity: "0.5",
		Price: "100.000", Commission: "0.02000", CommissionAsset: "USDT", RealizedPnl: "0", Time: 1800000000000, PositionSide: futures.PositionSideTypeLong}
	canonical, err := FromREST("main", rest)
	if err != nil {
		t.Fatal(err)
	}
	if err = Insert(o, canonical); err != nil {
		t.Fatalf("REST echo should dedup WS: %v", err)
	}
	rest.Commission = "0.03"
	conflict, _ := FromREST("main", rest)
	if err = Insert(o, conflict); err == nil {
		t.Fatal("inconsistent REST fee must not replace WS fact")
	}
	ws.LastFilledQty = "0"
	if _, accepted, err := FromWS("main", ws); accepted || err != nil {
		t.Fatalf("zero fill should be ignored: %v %v", accepted, err)
	}
	ws.ExecutionType = futures.OrderExecutionType("CANCELED")
	ws.LastFilledQty = "0.5"
	if _, accepted, err := FromWS("main", ws); accepted || err != nil {
		t.Fatalf("CANCELED is not a trade: %v %v", accepted, err)
	}
	ws.ExecutionType = futures.OrderExecutionType("TRADE")
	ws.Commission = ""
	if provisional, accepted, err := FromWS("main", ws); !accepted || err != nil || provisional.Commission != "0" || provisional.RESTVerified {
		t.Fatalf("zero-fee WS must remain provisional: %+v accepted=%v err=%v", provisional, accepted, err)
	}
}

func TestStageReviewRESTAuthoritativeAndMissingWSCommission(t *testing.T) {
	_ = orm.RegisterDriver("sqlite3", orm.DRSqlite)
	reviewRegisterModelsOnce.Do(func() { orm.RegisterModel(new(models.FuturesManagedPosition), new(models.FuturesManagedOrder)) })
	const alias = "stage_review_rest_confirmed"
	if err := orm.RegisterDataBase(alias, "sqlite3", filepath.Join(t.TempDir(), "review-rest.db")); err != nil {
		t.Fatal(err)
	}
	if err := orm.RunSyncdb(alias, false, false); err != nil {
		t.Fatal(err)
	}
	o := orm.NewOrmUsingDB(alias)
	if _, err := o.Raw(SchemaDDL).Exec(); err != nil {
		t.Fatal(err)
	}
	ws := futures.WsOrderTradeUpdate{Symbol: "BTCUSDT", ID: 777100, TradeID: 99101, ClientOrderID: "review_ws",
		ExecutionType: futures.OrderExecutionType("TRADE"), PositionSide: futures.PositionSideTypeLong,
		LastFilledQty: "1", LastFilledPrice: "100", Commission: "", CommissionAsset: "",
		RealizedPnL: "0", TradeTime: 1800000001000}
	provisional, accepted, err := FromWS("main", ws)
	if !accepted || err != nil || provisional.RESTVerified || provisional.Commission != "0" || provisional.CommissionAsset != "" {
		t.Fatalf("WS without fee asset should remain a provisional fill: %+v %v", provisional, err)
	}
	if err = Insert(o, provisional); err != nil {
		t.Fatal(err)
	}
	rest := &futures.AccountTrade{ID: 99101, OrderID: 777100, Symbol: "BTCUSDT",
		Quantity: "1.000", Price: "100.000", Commission: "0.42", CommissionAsset: "USDT",
		RealizedPnl: "-0.12", Time: 1800000001050, PositionSide: futures.PositionSideTypeLong}
	canonical, err := FromREST("main", rest)
	if err != nil {
		t.Fatal(err)
	}
	if err = Insert(o, canonical); err != nil {
		t.Fatalf("different WS fee/T should reconcile to REST: %v", err)
	}
	var fee, asset, pnl string
	var ts int64
	var verified int
	if err = o.Raw("SELECT CAST(commission AS TEXT),commission_asset,CAST(realized_pnl AS TEXT),trade_time,rest_verified FROM futures_trade_fills WHERE trade_id='99101'").
		QueryRow(&fee, &asset, &pnl, &ts, &verified); err != nil {
		t.Fatal(err)
	}
	if verified != 1 || fee != "0.42" || asset != "USDT" || pnl != "-0.12" || ts != rest.Time {
		t.Fatalf("REST authority lost: fee=%s asset=%s pnl=%s t=%d verified=%d", fee, asset, pnl, ts, verified)
	}
	if err = Insert(o, provisional); err != nil {
		t.Fatalf("late stale WS must not override REST: %v", err)
	}
	rest.Quantity = "1.1"
	changed, err := FromREST("main", rest)
	if err != nil {
		t.Fatal(err)
	}
	if err = Insert(o, changed); err == nil {
		t.Fatal("REST quantity conflict must fail closed")
	}
}
func TestStageReviewDECIMAL3618RejectsNineteenIntegerDigits(t *testing.T) {
	for _, input := range []string{"1000000000000000000", "9999999999999999999", "0.1234567890123456789"} {
		if _, err := decimal(input, true); err == nil {
			t.Fatalf("out-of-range %q accepted", input)
		}
	}
	if _, err := decimal("999999999999999999.123456789012345678", true); err != nil {
		t.Fatalf("max valid DECIMAL(36,18) rejected: %v", err)
	}
}

func newRat(s string) *big.Rat {
	r, ok := new(big.Rat).SetString(s)
	if !ok {
		panic(s)
	}
	return r
}

// REST itself may omit the commission asset for an EXACT zero fee.
func TestStageR2RESTZeroFeeMissingAssetIsAuthoritative(t *testing.T) {
	trade := &futures.AccountTrade{ID: 28001, OrderID: 28002, Symbol: "BTCUSDT",
		Quantity: "1", Price: "100", Commission: "0", CommissionAsset: "",
		RealizedPnl: "0", Time: 1800000000000, PositionSide: futures.PositionSideTypeLong}
	fill, err := FromREST("main", trade)
	if err != nil || !fill.RESTVerified || fill.CommissionAsset != "" {
		t.Fatalf("official zero commission not accepted: %+v %v", fill, err)
	}
	trade.Commission = "0.001"
	if _, err = FromREST("main", trade); err == nil {
		t.Fatal("unknown nonzero fee currency should fail closed")
	}
}
