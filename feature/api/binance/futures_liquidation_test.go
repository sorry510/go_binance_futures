package binance

import (
	"database/sql/driver"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"go_binance_futures/models"

	"github.com/beego/beego/v2/client/orm"
	_ "github.com/mattn/go-sqlite3"
)

func TestCleanupOldFuturesLiquidationOrdersDeletesInBatches(t *testing.T) {
	if err := orm.RegisterDriver("sqlite3", orm.DRSqlite); err != nil {
		t.Fatal(err)
	}
	if err := orm.RegisterDataBase("default", "sqlite3", filepath.Join(t.TempDir(), "test.db")); err != nil {
		t.Fatal(err)
	}
	orm.RegisterModel(new(models.FuturesLiquidationOrder))
	if err := orm.RunSyncdb("default", false, false); err != nil {
		t.Fatal(err)
	}

	o := orm.NewOrm()
	oldOrders := make([]models.FuturesLiquidationOrder, futuresLiquidationOrderCleanupBatchSize+3)
	oldEventTime := time.Now().AddDate(0, 0, -futuresLiquidationOrderKeepDays-1).UnixMilli()
	for i := range oldOrders {
		oldOrders[i].Symbol = "BTCUSDT"
		oldOrders[i].EventTime = oldEventTime
	}
	if _, err := o.InsertMulti(futuresLiquidationOrderCleanupBatchSize, oldOrders); err != nil {
		t.Fatal(err)
	}
	if _, err := o.Insert(&models.FuturesLiquidationOrder{
		Symbol:    "ETHUSDT",
		EventTime: time.Now().UnixMilli(),
	}); err != nil {
		t.Fatal(err)
	}

	deleted, err := CleanupOldFuturesLiquidationOrders(futuresLiquidationOrderKeepDays)
	if err != nil {
		t.Fatal(err)
	}
	if deleted != int64(len(oldOrders)) {
		t.Fatalf("deleted %d orders, want %d", deleted, len(oldOrders))
	}

	remaining, err := o.QueryTable(new(models.FuturesLiquidationOrder)).Count()
	if err != nil {
		t.Fatal(err)
	}
	if remaining != 1 {
		t.Fatalf("remaining %d orders, want 1", remaining)
	}
}

func TestRetryFuturesLiquidationOrderCleanupRetriesBadConnection(t *testing.T) {
	var attempts int
	var waits int
	cleanup := func(int) (int64, error) {
		attempts++
		if attempts < futuresLiquidationOrderCleanupMaxRetries {
			return int64(attempts), driver.ErrBadConn
		}
		return int64(attempts), nil
	}

	deleted, err := retryFuturesLiquidationOrderCleanup(cleanup, futuresLiquidationOrderKeepDays, func(time.Duration) {
		waits++
	})
	if err != nil {
		t.Fatal(err)
	}
	if attempts != futuresLiquidationOrderCleanupMaxRetries {
		t.Fatalf("cleanup attempts %d, want %d", attempts, futuresLiquidationOrderCleanupMaxRetries)
	}
	if waits != futuresLiquidationOrderCleanupMaxRetries-1 {
		t.Fatalf("retry waits %d, want %d", waits, futuresLiquidationOrderCleanupMaxRetries-1)
	}
	if deleted != 6 {
		t.Fatalf("deleted %d orders, want 6", deleted)
	}
}

func TestRetryFuturesLiquidationOrderCleanupDoesNotRetryOtherErrors(t *testing.T) {
	wantErr := errors.New("delete failed")
	var attempts int

	deleted, err := retryFuturesLiquidationOrderCleanup(func(int) (int64, error) {
		attempts++
		return 2, wantErr
	}, futuresLiquidationOrderKeepDays, func(time.Duration) {
		t.Fatal("unexpected retry wait")
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("error %v, want %v", err, wantErr)
	}
	if attempts != 1 {
		t.Fatalf("cleanup attempts %d, want 1", attempts)
	}
	if deleted != 2 {
		t.Fatalf("deleted %d orders, want 2", deleted)
	}
}
