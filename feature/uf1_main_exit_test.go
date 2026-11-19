package feature

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"reflect"
	"sync/atomic"
	"testing"
	"time"

	"github.com/adshao/go-binance/v2/futures"
	"github.com/beego/beego/v2/client/orm"
	_ "github.com/mattn/go-sqlite3"

	"go_binance_futures/feature/api/binance"
	"go_binance_futures/feature/strategy"
	"go_binance_futures/models"
	"go_binance_futures/notify"
	"go_binance_futures/service/futuresownership"
	"go_binance_futures/types"
)

var uf1FixtureSequence atomic.Uint64

func uf1LocalOwnershipDB(t *testing.T) orm.Ormer {
	t.Helper()
	_ = orm.RegisterDriver("sqlite3", orm.DRSqlite)
	alias := fmt.Sprintf("uf1_main_ownership_%d", uf1FixtureSequence.Add(1))
	if err := orm.RegisterDataBase(alias, "sqlite3", filepath.Join(t.TempDir(), "ownership.db")); err != nil {
		t.Fatal(err)
	}
	db := orm.NewOrmUsingDB(alias)
	for _, sql := range []string{
		`CREATE TABLE futures_managed_positions(id INTEGER PRIMARY KEY,account_id TEXT NOT NULL,owner TEXT NOT NULL,status TEXT NOT NULL,managed_qty REAL NOT NULL)`,
		`CREATE TABLE futures_managed_orders(id INTEGER PRIMARY KEY,account_id TEXT NOT NULL,owner TEXT NOT NULL,status TEXT NOT NULL)`,
	} {
		if _, err := db.Raw(sql).Exec(); err != nil {
			t.Fatal(err)
		}
	}
	return db
}

func TestUF1MainOFFOnlyWakesWithMainOwnedWork(t *testing.T) {
	db := uf1LocalOwnershipDB(t)
	check := func(want bool) {
		t.Helper()
		found, err := mainAutoStrategyWorkNeeded(db)
		if err != nil || found != want {
			t.Fatalf("Main owned work found=%t want=%t err=%v", found, want, err)
		}
	}
	check(false)
	for _, sql := range []string{
		`INSERT INTO futures_managed_positions(account_id,owner,status,managed_qty) VALUES('lead','auto_strategy','active',1)`,
		`INSERT INTO futures_managed_positions(account_id,owner,status,managed_qty) VALUES('main','notice_auto_order','active',1)`,
		`INSERT INTO futures_managed_positions(account_id,owner,status,managed_qty) VALUES('main','auto_strategy','closed',0)`,
		`INSERT INTO futures_managed_orders(account_id,owner,status) VALUES('lead','auto_strategy','pending')`,
		`INSERT INTO futures_managed_orders(account_id,owner,status) VALUES('main','notice_auto_order','submitted')`,
		`INSERT INTO futures_managed_orders(account_id,owner,status) VALUES('main','auto_strategy','filled')`,
	} {
		if _, err := db.Raw(sql).Exec(); err != nil {
			t.Fatal(err)
		}
	}
	check(false)
	// Active-looking rows with zero contracts are not executable Managed
	// positions; they must not cause 2-second authenticated account polling.
	if _, err := db.Raw("INSERT INTO futures_managed_positions(account_id,owner,status,managed_qty) VALUES('main','auto_strategy','suspended',0)").Exec(); err != nil {
		t.Fatal(err)
	}
	check(false)
	for _, status := range []string{"pending", "submitted", "partially_filled", "reconcile_required"} {
		if _, err := db.Raw("INSERT INTO futures_managed_orders(account_id,owner,status) VALUES('main','auto_strategy',?)", status).Exec(); err != nil {
			t.Fatal(err)
		}
		check(true)
		if _, err := db.Raw("DELETE FROM futures_managed_orders WHERE account_id='main' AND owner='auto_strategy' AND status=?", status).Exec(); err != nil {
			t.Fatal(err)
		}
		check(false)
	}
	for _, status := range []string{"active", "suspended", "reconcile_required"} {
		if _, err := db.Raw("INSERT INTO futures_managed_positions(account_id,owner,status,managed_qty) VALUES('main','auto_strategy',?,1)", status).Exec(); err != nil {
			t.Fatal(err)
		}
		check(true)
		if _, err := db.Raw("DELETE FROM futures_managed_positions WHERE account_id='main' AND owner='auto_strategy' AND status=?", status).Exec(); err != nil {
			t.Fatal(err)
		}
		check(false)
	}
}

func TestUF1MainOFFDBFailureDoesNotInferSafeNoWork(t *testing.T) {
	db := uf1LocalOwnershipDB(t)
	if _, err := db.Raw("DROP TABLE futures_managed_positions").Exec(); err != nil {
		t.Fatal(err)
	}
	if _, err := mainAutoStrategyWorkNeeded(db); err == nil {
		t.Fatal("missing ownership ledger treated as confirmed idle")
	}
	if needed, err := mainTradeCycleAllowed(false, func() (bool, error) { return mainAutoStrategyWorkNeeded(db) }); err == nil || needed {
		t.Fatalf("OFF with unavailable DB must never authorize account execution: %t %v", needed, err)
	}
	runs := 0
	on, err := mainTradeCycleAllowed(true, func() (bool, error) { runs++; return false, nil })
	if err != nil || !on || runs != 0 {
		t.Fatalf("ON must retain original Main scheduler: run=%v err=%v queries=%d", on, err, runs)
	}
}

func uf1MockMainRunner(t *testing.T, side string, allowNewOpens bool) (*accountTradeRunner, *int, *int, *int, *int) {
	t.Helper()
	config := &models.Config{FutureEnable: 0, FutureMaxCount: 5, LossMaxCount: 5, FutureAllowLong: 1, FutureAllowShort: 1, FutureOrderType: "MARKET"}
	coin := &models.Symbols{Symbol: "BTCUSDT", TickSize: "0.1", StepSize: "0.01", Usdt: "100", Leverage: 4, Profit: "8", Loss: "6"}
	quantity := "1"
	if side == "SHORT" {
		quantity = "-1"
	}
	active := types.FuturesPosition{
		Symbol: "BTCUSDT", Side: side, Amount: quantity, EntryPrice: "100", MarkPrice: "100",
		UnrealizedProfit: "4", Leverage: 4,
	}
	opens, closes, selects, cancels := new(int), new(int), new(int), new(int)
	r := &accountTradeRunner{
		AccountID: binance.MainAccountID, Config: config, AllowNewOpens: allowNewOpens,
		LoadSymbols: func() ([]*models.Symbols, error) { return []*models.Symbols{coin}, nil },
		SelectCoins: func(*models.Config, []*models.Symbols) ([]*models.Symbols, error) {
			*selects++
			return []*models.Symbols{coin}, nil
		},
		ReadPositions:  func(context.Context) ([]types.FuturesPosition, error) { return []types.FuturesPosition{active}, nil },
		ReadOpenOrders: func(context.Context) ([]types.FuturesOrder, error) { return []types.FuturesOrder{}, nil },
		SyncPositions: func([]types.FuturesPosition) ([]ownedTradePosition, error) {
			return []ownedTradePosition{{Position: active, Owner: futuresownership.OwnerAutoStrategy, SourceRef: "uf1-verified-main"}}, nil
		},
		CancelExpired: func(int64) { *cancels++ },
		EvaluateEntry: func(*models.Symbols, []types.FuturesPosition) strategy.OpenResult {
			return strategy.OpenResult{CanLong: true, CanShort: true}
		},
		Depth:        func(context.Context, string, int) (float64, float64, error) { return 100, 100, nil },
		EnsureConfig: func(context.Context, *models.Symbols) error { return nil },
		SubmitOpen: func(*tradeCycleAccountSnapshot, string, float64, float64, futures.SideType, futures.PositionSideType, futures.OrderType, string) (*futures.CreateOrderResponse, error) {
			*opens++
			return &futures.CreateOrderResponse{OrderID: 999}, nil
		},
		SubmitClose: func(owner, source, symbol string, qty float64, posSide futures.PositionSideType) (*futures.CreateOrderResponse, error) {
			if owner != futuresownership.OwnerAutoStrategy || source != "uf1-verified-main" || symbol != "BTCUSDT" ||
				string(posSide) != side || qty < 0.99 || qty > 1.01 {
				t.Fatalf("wrong Managed exit ownership: %s %s %s %.6f %s", owner, source, symbol, qty, posSide)
			}
			*closes++
			return &futures.CreateOrderResponse{OrderID: 123}, nil
		},
		NotifyOpen:   func(notify.FuturesOrderParams) {},
		NotifyClose:  func(notify.FuturesOrderParams) {},
		EvaluateExit: func(strategy.CloseParams, float64, float64) tradeExitReason { return tradeExitAuto },
		Sleep:        func(time.Duration) {},
	}
	return r, opens, closes, selects, cancels
}

func TestUF1MainOFFLongShortExitBeforeSelection(t *testing.T) {
	for _, side := range []string{"LONG", "SHORT"} {
		t.Run(side, func(t *testing.T) {
			r, opens, closes, selects, cancels := uf1MockMainRunner(t, side, false)
			r.SelectCoins = func(*models.Config, []*models.Symbols) ([]*models.Symbols, error) {
				*selects++
				return nil, errors.New("candidate service outage")
			}
			runAccountTradeCycle(r)
			if *opens != 0 || *closes != 1 || *selects != 0 || *cancels != 1 {
				t.Fatalf("OFF should exit only, not evaluate candidate: open=%d close=%d selection=%d cleanup=%d", *opens, *closes, *selects, *cancels)
			}
		})
	}
}

func TestUF1MainONSelectionFailsButExitStillRuns(t *testing.T) {
	r, opens, closes, selects, _ := uf1MockMainRunner(t, "LONG", true)
	r.SelectCoins = func(*models.Config, []*models.Symbols) ([]*models.Symbols, error) {
		*selects++
		return nil, errors.New("candidate scan error")
	}
	runAccountTradeCycle(r)
	if *closes != 1 || *opens != 0 || *selects != 1 {
		t.Fatalf("ON select error must not strand exit: closes=%d opens=%d scans=%d", *closes, *opens, *selects)
	}
}

func TestUF1MainOFFHoldsWhenNoExitSignal(t *testing.T) {
	r, opens, closes, selects, _ := uf1MockMainRunner(t, "SHORT", false)
	r.EvaluateExit = func(strategy.CloseParams, float64, float64) tradeExitReason { return tradeExitHold }
	runAccountTradeCycle(r)
	if *closes != 0 || *opens != 0 || *selects != 0 {
		t.Fatalf("OFF no exit signal must hold, not open/close: %d %d %d", *opens, *closes, *selects)
	}
}

func TestUF1MainOFFUnownedOrEmptyCannotTrade(t *testing.T) {
	r, opens, closes, selects, _ := uf1MockMainRunner(t, "LONG", false)
	r.SyncPositions = func([]types.FuturesPosition) ([]ownedTradePosition, error) { return nil, nil }
	runAccountTradeCycle(r)
	if *opens != 0 || *closes != 0 || *selects != 0 {
		t.Fatalf("unknown/external position was claimed or traded: open=%d close=%d select=%d", *opens, *closes, *selects)
	}
}

func TestUF1MainONSelectionStillRunsAfterExit(t *testing.T) {
	r, _, closes, selects, _ := uf1MockMainRunner(t, "SHORT", true)
	r.SelectCoins = func(*models.Config, []*models.Symbols) ([]*models.Symbols, error) {
		*selects++
		return []*models.Symbols{}, nil
	}
	runAccountTradeCycle(r)
	if *closes != 1 || *selects != 1 {
		t.Fatalf("ON did not preserve close and subsequent candidate pass: close=%d select=%d", *closes, *selects)
	}
}

// Owner-specific running modes must remain explicit: the OFF-only recovery
// exception is not permission to process NewCoinRush/FundingRate positions.
func TestUF1MainOFFUsesAutoStrategyOnlySync(t *testing.T) {
	for _, tc := range []struct {
		enabled      int
		wantAutoOnly bool
	}{
		{0, true}, {1, false},
	} {
		runner := mainTradeRunner(&models.Config{FutureEnable: tc.enabled})
		got := reflect.ValueOf(runner.SyncPositions).Pointer()
		expected := reflect.ValueOf(syncStrategyExitPositions).Pointer()
		if tc.wantAutoOnly {
			expected = reflect.ValueOf(syncAutoStrategyOwnership).Pointer()
		}
		if got != expected {
			t.Fatalf("Main FutureEnable=%d sync scope changed: auto only=%t", tc.enabled, tc.wantAutoOnly)
		}
	}
}

func TestUF1MainOFFMissingSymbolCannotBlindlyClose(t *testing.T) {
	r, opens, closes, selects, _ := uf1MockMainRunner(t, "LONG", false)
	r.LoadSymbols = func() ([]*models.Symbols, error) {
		return []*models.Symbols{{Symbol: "ETHUSDT", Profit: "8", Loss: "6"}}, nil
	}
	runAccountTradeCycle(r)
	if *opens != 0 || *closes != 0 || *selects != 0 {
		t.Fatalf("missing symbol config was guessed for automatic close: open=%d close=%d selected=%d", *opens, *closes, *selects)
	}
}
