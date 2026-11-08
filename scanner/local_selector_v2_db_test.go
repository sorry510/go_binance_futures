package scanner

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"go_binance_futures/models"
	"go_binance_futures/service/tradefills"

	"github.com/beego/beego/v2/client/orm"
	_ "github.com/mattn/go-sqlite3"
)

var smartLocalV2DBOnce sync.Once
var smartLocalV2DBSeq atomic.Int64

func setupSmartLocalV2DB(t *testing.T) orm.Ormer {
	t.Helper()
	alias := fmt.Sprintf("smart_local_v2_%d", smartLocalV2DBSeq.Add(1))
	smartLocalV2DBOnce.Do(func() {
		_ = orm.RegisterDriver("sqlite3", orm.DRSqlite)
		orm.RegisterModel(new(models.Symbols), new(models.FuturesManagedOrder), new(models.TestStrategyResults))
	})
	if err := orm.RegisterDataBase(alias, "sqlite3", fmt.Sprintf("file:%s?mode=memory&cache=shared", alias)); err != nil {
		t.Fatal(err)
	}
	if err := orm.RunSyncdb(alias, true, false); err != nil {
		t.Fatal(err)
	}
	o := orm.NewOrmUsingDB(alias)
	if _, err := o.Raw(tradefills.SchemaDDL).Exec(); err != nil {
		t.Fatal(err)
	}
	return o
}

func insertManagedCooldownFixture(t *testing.T, o orm.Ormer, id int64, account, symbol, intent string, filled float64, at int64) {
	t.Helper()
	row := &models.FuturesManagedOrder{
		ID: id, AccountID: account, Owner: "auto_strategy", Symbol: symbol,
		PositionSide: "LONG", Intent: intent, ClientOrderID: fmt.Sprintf("cooldown-%d-%s-%s", id, account, symbol),
		RequestedQty: 1, FilledQty: filled, OrderType: "MARKET",
		Status: "filled", CreatedAt: at, UpdatedAt: at,
	}
	if _, err := o.Insert(row); err != nil {
		t.Fatal(err)
	}
	if filled > 0 {
		idstr := fmt.Sprintf("%d", id)
		_, err := o.Raw(`INSERT INTO futures_trade_fills
        (market,account_id,symbol,order_id,trade_id,client_order_id,intent,
        position_side,qty,price,commission,commission_asset,realized_pnl,pnl_asset,trade_time,rest_verified)
        VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
			tradefills.MarketUSDM, account, symbol, idstr, idstr, row.ClientOrderID, intent,
			"LONG", fmt.Sprintf("%.8f", filled), "100", "0", "USDT", "0", "USDT", at, 1).Exec()
		if err != nil {
			t.Fatal(err)
		}
	}
}

func TestRecentClosedSymbolsUsesManagedFillsNotLegacyHistory(t *testing.T) {
	o := setupSmartLocalV2DB(t)
	now := time.Now().UnixMilli()
	insertManagedCooldownFixture(t, o, 1, "main", "RECENTUSDT", "close", 1, now-60_000)
	insertManagedCooldownFixture(t, o, 2, "main", "OLDUSDT", "close", 1, now-10*60_000)
	insertManagedCooldownFixture(t, o, 3, "main", "OPENUSDT", "open", 1, now-60_000)
	insertManagedCooldownFixture(t, o, 4, "main", "ACKONLYUSDT", "close", 0, now-10_000)
	insertManagedCooldownFixture(t, o, 5, "main", "PARTCANCELUSDT", "stop_loss", 0.3, now-20_000)

	insertManagedCooldownFixture(t, o, 6, "main", "LATEFILLUSDT", "close", 1, now-20*60_000)
	insertManagedCooldownFixture(t, o, 7, "main", "REFRESHEDUSDT", "close", 1, now-10_000)
	// Immutable trade times must win in BOTH directions, independent of
	// Managed updated_at (e.g. late WS/REST insert versus later status refresh).
	if _, err := o.Raw("UPDATE futures_trade_fills SET trade_time=? WHERE symbol='LATEFILLUSDT'", now-10_000).Exec(); err != nil {
		t.Fatal(err)
	}
	if _, err := o.Raw("UPDATE futures_trade_fills SET trade_time=? WHERE symbol='REFRESHEDUSDT'", now-60*60_000).Exec(); err != nil {
		t.Fatal(err)
	}

	got, err := recentClosedSymbolsWithOrm(context.Background(), o, 5)
	if err != nil {
		t.Fatal(err)
	}
	if !got["RECENTUSDT"] {
		t.Fatalf("recent close missing: %+v", got)
	}
	if got["OLDUSDT"] {
		t.Fatalf("old close must not trigger cooldown: %+v", got)
	}
	if got["OPENUSDT"] {
		t.Fatalf("open order must not trigger cooldown: %+v", got)
	}
	if got["ACKONLYUSDT"] || !got["PARTCANCELUSDT"] {
		t.Fatalf("ACK-only must not cool down, real partial protective fill must cool down: %+v", got)
	}
	if !got["LATEFILLUSDT"] || got["REFRESHEDUSDT"] {
		t.Fatalf("trade_time cooldown broken by mutable Managed timestamp: %+v", got)
	}
}

func TestRecentTestClosedSymbolsUsesTestResults(t *testing.T) {
	o := setupSmartLocalV2DB(t)
	now := time.Now().UnixMilli()
	rows := []*models.TestStrategyResults{
		{ID: 1, Symbol: "RECENTUSDT", ClosePrice: "1.1", UpdateTime: now - 60_000},
		{ID: 2, Symbol: "OLDUSDT", ClosePrice: "1.2", UpdateTime: now - 10*60_000},
		{ID: 3, Symbol: "OPENUSDT", ClosePrice: "0", UpdateTime: now - 60_000},
	}
	for _, row := range rows {
		if _, err := o.Insert(row); err != nil {
			t.Fatal(err)
		}
	}
	got, err := recentTestClosedSymbolsWithOrm(context.Background(), o, 5)
	if err != nil {
		t.Fatal(err)
	}
	if !got["RECENTUSDT"] {
		t.Fatalf("recent test close missing: %+v", got)
	}
	if got["OLDUSDT"] {
		t.Fatalf("old test close must not trigger cooldown: %+v", got)
	}
	if got["OPENUSDT"] {
		t.Fatalf("open test position must not trigger cooldown: %+v", got)
	}
}

func TestSmartLocalV2DBEndToEndAndFailClosed(t *testing.T) {
	o := setupSmartLocalV2DB(t)
	now := time.Now().UnixMilli()

	rows := []*models.Symbols{
		smartSymbol("AAAUSDT", 1, 8, 120_000_000, 220_000, now),
		smartSymbol("BBBUSDT", 1, 6, 150_000_000, 180_000, now),
		smartSymbol("STALEUSDT", 1, 5, 500_000_000, 500_000, now-60_000),
		smartSymbol("DISABLEDUSDT", 0, 5, 500_000_000, 500_000, now),
	}
	for i, row := range rows {
		row.ID = int64(i + 1)
		if _, err := o.Insert(row); err != nil {
			t.Fatal(err)
		}
	}
	insertManagedCooldownFixture(t, o, 1, "main", "BBBUSDT", "close", 1, now-30_000)

	result, err := smartLocalV2WithOrm(context.Background(), o, SmartLocalV2ModeTrade, SmartLocalV2Options{
		Limit:           3,
		IncludeExcluded: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Selector != "smart_local_v2" || result.Meta["rest_api_used"] != false {
		t.Fatalf("unexpected result meta: selector=%s meta=%+v", result.Selector, result.Meta)
	}
	if len(result.Candidates) != 1 || result.Candidates[0].Symbol != "AAAUSDT" {
		t.Fatalf("candidates=%+v want AAAUSDT only", result.Candidates)
	}
	excluded := map[string]string{}
	for _, item := range result.Excluded {
		excluded[item.Symbol] = item.Reason
	}
	if excluded["BBBUSDT"] != "最近交易冷却中" {
		t.Fatalf("BBBUSDT exclusion=%q", excluded["BBBUSDT"])
	}
	if excluded["STALEUSDT"] != "本地行情数据过旧" {
		t.Fatalf("STALEUSDT exclusion=%q", excluded["STALEUSDT"])
	}
	if excluded["DISABLEDUSDT"] != "币种未启用" {
		t.Fatalf("DISABLEDUSDT exclusion=%q", excluded["DISABLEDUSDT"])
	}

	// Make every enabled symbol stale and verify DB path fails closed.
	for _, symbol := range []string{"AAAUSDT", "BBBUSDT", "STALEUSDT"} {
		if _, err := o.QueryTable(new(models.Symbols)).Filter("Symbol", symbol).Update(orm.Params{
			"UpdateTime": now - 120_000,
		}); err != nil {
			t.Fatal(err)
		}
	}
	staleResult, err := smartLocalV2WithOrm(context.Background(), o, SmartLocalV2ModeTrade, SmartLocalV2Options{Limit: 5})
	if err != nil {
		t.Fatal(err)
	}
	if len(staleResult.Candidates) != 0 {
		t.Fatalf("all-stale selector must fail closed: %+v", staleResult.Candidates)
	}
}

func TestSmartLocalV2TradeAndTestCooldownSourcesAreIndependent(t *testing.T) {
	o := setupSmartLocalV2DB(t)
	now := time.Now().UnixMilli()
	rows := []*models.Symbols{
		smartSymbol("REALCOOLUSDT", 1, 8, 100_000_000, 100_000, now),
		smartSymbol("TESTCOOLUSDT", 1, -8, 100_000_000, 100_000, now),
		smartSymbol("FREEUSDT", 1, 4, 100_000_000, 100_000, now),
	}
	for i, row := range rows {
		row.ID = int64(i + 1)
		if _, err := o.Insert(row); err != nil {
			t.Fatal(err)
		}
	}
	insertManagedCooldownFixture(t, o, 1, "main", "REALCOOLUSDT", "close", 1, now-30_000)
	if _, err := o.Insert(&models.TestStrategyResults{
		ID: 1, Symbol: "TESTCOOLUSDT", ClosePrice: "1", UpdateTime: now - 30_000,
	}); err != nil {
		t.Fatal(err)
	}

	trade, err := smartLocalV2WithOrm(context.Background(), o, SmartLocalV2ModeTrade, SmartLocalV2Options{
		Limit: 30, IncludeExcluded: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	testMode, err := smartLocalV2WithOrm(context.Background(), o, SmartLocalV2ModeTest, SmartLocalV2Options{
		Limit: 30, IncludeExcluded: true,
	})
	if err != nil {
		t.Fatal(err)
	}

	tradeSymbols := map[string]bool{}
	for _, item := range trade.Candidates {
		tradeSymbols[item.Symbol] = true
	}
	testSymbols := map[string]bool{}
	for _, item := range testMode.Candidates {
		testSymbols[item.Symbol] = true
	}
	if tradeSymbols["REALCOOLUSDT"] {
		t.Fatalf("trade mode must exclude real cooldown symbol: %+v", trade.Candidates)
	}
	if !tradeSymbols["TESTCOOLUSDT"] {
		t.Fatalf("trade mode must ignore test-only cooldown: %+v", trade.Candidates)
	}
	if testSymbols["TESTCOOLUSDT"] {
		t.Fatalf("test mode must exclude test cooldown symbol: %+v", testMode.Candidates)
	}
	if !testSymbols["REALCOOLUSDT"] {
		t.Fatalf("test mode must ignore real-only cooldown: %+v", testMode.Candidates)
	}
}

// Stage 3 Lead cooldown uses only its own closed trade history; a main close
// never silently suppresses Lead's Top60 candidate and vice versa.
func TestStage3CooldownByAccount(t *testing.T) {
	o := setupSmartLocalV2DB(t)
	now := time.Now().UnixMilli()
	for i, symbol := range []string{"MAINCOOLUSDT", "LEADCOOLUSDT", "FREEUSDT"} {
		s := smartSymbol(symbol, 1, 8, 100_000_000, 100_000, now)
		s.ID = int64(i + 1)
		if _, err := o.Insert(s); err != nil {
			t.Fatal(err)
		}
	}
	for i, account := range []string{"main", "lead"} {
		symbol := []string{"MAINCOOLUSDT", "LEADCOOLUSDT"}[i]
		insertManagedCooldownFixture(t, o, int64(i+1), account, symbol, "close", 1, now-10_000)
	}
	for _, tc := range []struct{ account, excluded, other string }{
		{"main", "MAINCOOLUSDT", "LEADCOOLUSDT"},
		{"lead", "LEADCOOLUSDT", "MAINCOOLUSDT"},
	} {
		got, err := recentClosedSymbolsForAccountWithOrm(context.Background(), o, 5, tc.account)
		if err != nil {
			t.Fatal(err)
		}
		if !got[tc.excluded] || got[tc.other] {
			t.Fatalf("account=%s cooldown=%v", tc.account, got)
		}
	}
}
