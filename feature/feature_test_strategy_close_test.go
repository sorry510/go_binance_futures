package feature

import (
	"fmt"
	"math"
	"strings"
	"testing"

	"go_binance_futures/models"

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
		{100, 1, "ROI > 1", strings.Repeat("a", 129)},
	} {
		if ok, err := finalizeTestStrategyClose(orm.NewOrm(), row, tc.price, tc.profit, tc.code, tc.name, "close_long"); err == nil || ok {
			t.Fatalf("invalid close accepted: %+v ok=%v err=%v", tc, ok, err)
		}
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
