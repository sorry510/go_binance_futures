package feature

import (
	"fmt"
	"math"
	"strings"
	"testing"

	"go_binance_futures/models"
	strategyservice "go_binance_futures/service/strategy"

	"github.com/beego/beego/v2/client/orm"
)

func newCloseTestResult(t *testing.T) *models.TestStrategyResults {
	t.Helper()
	setupFeatureTestORM(t)
	row := &models.TestStrategyResults{
		Symbol: "TESTCLOSEUSDT", PositionSide: "LONG",
		Price: "100", PositionAmt: "1", ClosePrice: "0",
		CloseProfit: "0", Strategy: "[]",
	}
	id, err := orm.NewOrm().Insert(row)
	if err != nil {
		t.Fatal(err)
	}
	row.ID = id
	return row
}

func TestFinalizeTestStrategyCloseOnceOnlyAndFields(t *testing.T) {
	row := newCloseTestResult(t)
	o := orm.NewOrm()
	ok, err := finalizeTestStrategyClose(o, row, 110, 9.25, "ROI > 3", "ROI close", "close_long")
	if err != nil || !ok {
		t.Fatalf("first close must persist: ok=%v err=%v", ok, err)
	}
	var dbRow models.TestStrategyResults
	if err := o.QueryTable(new(models.TestStrategyResults)).Filter("id", row.ID).One(&dbRow); err != nil {
		t.Fatal(err)
	}
	if dbRow.ClosePrice != "110" || dbRow.CloseProfit != "9.250" ||
		dbRow.CloseStrategy != "ROI > 3" || dbRow.CloseStrategyName != "ROI close" ||
		dbRow.CloseStrategyType != "close_long" || dbRow.CloseStrategyHash == "" || dbRow.UpdateTime <= 0 {
		t.Fatalf("close metadata not persisted: %+v", dbRow)
	}
	if dbRow.Symbol != "TESTCLOSEUSDT" || dbRow.Price != "100" || dbRow.PositionAmt != "1" || dbRow.Strategy != "[]" {
		t.Fatalf("closing unexpectedly modified the entry snapshot: %+v", dbRow)
	}
	ok, err = finalizeTestStrategyClose(o, row, 120, 20, "ROI > 1", "another", "close_long")
	if err != nil || ok {
		t.Fatalf("duplicate close must be ignored: ok=%v err=%v", ok, err)
	}
	var persisted models.TestStrategyResults
	if err := o.QueryTable(new(models.TestStrategyResults)).Filter("id", row.ID).One(&persisted); err != nil {
		t.Fatal(err)
	}
	if persisted.ClosePrice != dbRow.ClosePrice || persisted.CloseProfit != dbRow.CloseProfit || persisted.CloseStrategyHash != dbRow.CloseStrategyHash {
		t.Fatalf("duplicate close changed persisted result: first=%+v second=%+v", dbRow, persisted)
	}
}

func TestFinalizeTestStrategyCloseRejectsInvalidInputsWithoutUpdating(t *testing.T) {
	row := newCloseTestResult(t)
	for _, tc := range []struct {
		price, profit float64
		code, name    string
	}{
		{0, 1, "ROI > 1", "close"},
		{-1, 1, "ROI > 1", "close"},
		{math.NaN(), 1, "ROI > 1", "close"},
		{100, math.Inf(1), "ROI > 1", "close"},
		{100, 1, "", "close"},
	} {
		if ok, err := finalizeTestStrategyClose(orm.NewOrm(), row, tc.price, tc.profit, tc.code, tc.name, "close_long"); err == nil || ok {
			t.Fatalf("invalid close accepted: %+v ok=%v err=%v", tc, ok, err)
		}
	}
	if ok, err := finalizeTestStrategyClose(orm.NewOrm(), row, 100, 1, "ROI > 1", "close", strings.Repeat("a", 33)); err == nil || ok {
		t.Fatalf("invalid close type accepted: ok=%v err=%v", ok, err)
	}
	var persisted models.TestStrategyResults
	if err := orm.NewOrm().QueryTable(new(models.TestStrategyResults)).Filter("id", row.ID).One(&persisted); err != nil {
		t.Fatal(err)
	}
	if persisted.ClosePrice != "0" {
		t.Fatalf("invalid close modified unclosed row: %+v", persisted)
	}
}

func TestFinalizeTestStrategyCloseDBFailureDoesNotReportSuccess(t *testing.T) {
	row := newCloseTestResult(t)
	o := orm.NewOrm()
	// A DB-side write failure must not make a simulated position appear
	// closed in memory, or authorize the calling code to send a notification.
	triggerName := fmt.Sprintf("test_close_block_%d", row.ID)
	triggerSQL := fmt.Sprintf(
		"CREATE TRIGGER %s BEFORE UPDATE ON test_strategy_results "+
			"WHEN OLD.id = %d BEGIN SELECT RAISE(FAIL, 'simulated database write failure'); END",
		triggerName, row.ID,
	)
	if _, err := o.Raw(triggerSQL).Exec(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if _, err := o.Raw("DROP TRIGGER IF EXISTS " + triggerName).Exec(); err != nil {
			t.Errorf("drop test-only trigger: %v", err)
		}
	})

	updated, err := finalizeTestStrategyClose(o, row, 112, 12.5, "ROI > 1", "close", "close_long")
	if err == nil || updated {
		t.Fatalf("failed database write must not report success: updated=%v err=%v", updated, err)
	}
	if row.ClosePrice != "0" || row.CloseProfit != "0" {
		t.Fatalf("failed write mutated the simulated position: %+v", row)
	}
	var stored models.TestStrategyResults
	if err := o.QueryTable(new(models.TestStrategyResults)).Filter("id", row.ID).One(&stored); err != nil {
		t.Fatal(err)
	}
	if stored.ClosePrice != "0" {
		t.Fatalf("failed write unexpectedly settled the simulated position: %+v", stored)
	}
	if _, err := o.Raw("DROP TRIGGER " + triggerName).Exec(); err != nil {
		t.Fatal(err)
	}

	// A subsequent successful retry should settle exactly once.
	updated, err = finalizeTestStrategyClose(o, row, 112, 12.5, "ROI > 1", "close", "close_long")
	if err != nil || !updated {
		t.Fatalf("retry after database recovery must succeed: updated=%v err=%v", updated, err)
	}
}

func TestFinalizeTestStrategyCloseTruncatesNameWithoutLosingIdentity(t *testing.T) {
	for _, tc := range []struct {
		name string
		code string
	}{
		{strings.Repeat("a", 150), "ROI > 3"},
		{strings.Repeat("止损策略", 45), "ROI < -6"},
	} {
		t.Run(fmt.Sprintf("runes_%d", len([]rune(tc.name))), func(t *testing.T) {
			row := newCloseTestResult(t)
			snapshot := fmt.Sprintf(`[{"name":%q,"type":"close_long","code":%q,"enable":true}]`, tc.name, tc.code)
			row.Strategy = snapshot
			o := orm.NewOrm()
			if _, err := o.Update(row, "Strategy"); err != nil {
				t.Fatal(err)
			}
			updated, err := finalizeTestStrategyClose(o, row, 111, 11.1, tc.code, tc.name, "close_long")
			if err != nil || !updated {
				t.Fatalf("long close name must not block closing: updated=%v err=%v", updated, err)
			}
			var saved models.TestStrategyResults
			if err := o.QueryTable(new(models.TestStrategyResults)).Filter("id", row.ID).One(&saved); err != nil {
				t.Fatal(err)
			}
			expectedName := string([]rune(tc.name)[:128])
			if saved.CloseStrategyName != expectedName || len([]rune(saved.CloseStrategyName)) != 128 {
				t.Fatalf("incorrect truncated close name: %q", saved.CloseStrategyName)
			}
			if saved.ClosePrice != "111" || saved.CloseStrategy != tc.code ||
				saved.CloseStrategyHash != strategyservice.RuleHash(tc.code) ||
				saved.Strategy != snapshot || row.CloseStrategyName != expectedName {
				t.Fatalf("close identity or original strategy snapshot changed: %+v", saved)
			}
			again, err := finalizeTestStrategyClose(o, row, 112, 12.2, tc.code, "different", "close_long")
			if err != nil || again {
				t.Fatalf("second close must be ignored: updated=%v err=%v", again, err)
			}
			var unchanged models.TestStrategyResults
			if err := o.QueryTable(new(models.TestStrategyResults)).Filter("id", row.ID).One(&unchanged); err != nil {
				t.Fatal(err)
			}
			if unchanged.ClosePrice != saved.ClosePrice || unchanged.CloseStrategyName != saved.CloseStrategyName {
				t.Fatalf("duplicate close modified stored row: %+v", unchanged)
			}
		})
	}
}
