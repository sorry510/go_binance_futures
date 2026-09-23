package scanner

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"go_binance_futures/models"

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
		orm.RegisterModel(new(models.Symbols), new(models.Order), new(models.TestStrategyResults))
	})
	if err := orm.RegisterDataBase(alias, "sqlite3", fmt.Sprintf("file:%s?mode=memory&cache=shared", alias)); err != nil {
		t.Fatal(err)
	}
	if err := orm.RunSyncdb(alias, true, false); err != nil {
		t.Fatal(err)
	}
	return orm.NewOrmUsingDB(alias)
}

func TestRecentClosedSymbolsUsesLocalOrderTable(t *testing.T) {
	o := setupSmartLocalV2DB(t)
	now := time.Now().UnixMilli()
	rows := []*models.Order{
		{ID: 1, Symbol: "RECENTUSDT", Side: "close", UpdateTime: now - 60_000},
		{ID: 2, Symbol: "OLDUSDT", Side: "close", UpdateTime: now - 10*60_000},
		{ID: 3, Symbol: "OPENUSDT", Side: "open", UpdateTime: now - 60_000},
	}
	for _, row := range rows {
		if _, err := o.Insert(row); err != nil {
			t.Fatal(err)
		}
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
	if _, err := o.Insert(&models.Order{ID: 1, Symbol: "BBBUSDT", Side: "close", UpdateTime: now - 30_000}); err != nil {
		t.Fatal(err)
	}

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
	if _, err := o.Insert(&models.Order{
		ID: 1, Symbol: "REALCOOLUSDT", Side: "close", UpdateTime: now - 30_000,
	}); err != nil {
		t.Fatal(err)
	}
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
