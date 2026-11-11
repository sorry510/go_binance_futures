package webnotification

import (
	"fmt"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"

	"go_binance_futures/models"

	"github.com/beego/beego/v2/client/orm"
	_ "github.com/mattn/go-sqlite3"
)

var notificationDefaultOnce sync.Once
var notificationDefaultErr error
var notificationMigrationSeq atomic.Uint64

func TestPublishWithOptionsPersistsStructuredLiquidationFields(t *testing.T) {
	const alias = "default"
	if err := orm.RegisterDriver("sqlite3", orm.DRSqlite); err != nil {
		t.Fatal(err)
	}
	notificationDefaultOnce.Do(func() {
		notificationDefaultErr = orm.RegisterDataBase(alias, "sqlite3", "file:s510_webnotification_default?mode=memory&cache=shared")
		if notificationDefaultErr == nil {
			orm.RegisterModel(new(models.Notification))
		}
	})
	if notificationDefaultErr != nil {
		t.Fatal(notificationDefaultErr)
	}
	if err := orm.RunSyncdb(alias, true, false); err != nil {
		t.Fatal(err)
	}

	notification, err := PublishWithOptions("futures_liquidation", "## BTCUSDT large liquidation aggregate alert", PublishOptions{
		Level:             "warning",
		EventType:         "futures_liquidation_aggregate",
		Symbol:            "BTCUSDT",
		LiquidationSide:   "long",
		AggregateNotional: 6_000_000,
		OrderCount:        3,
		WindowStart:       1_700_000_000_000,
		WindowEnd:         1_700_000_030_000,
	})
	if err != nil {
		t.Fatal(err)
	}

	o := orm.NewOrmUsingDB(alias)
	stored := &models.Notification{ID: notification.ID}
	if err := o.Read(stored); err != nil {
		t.Fatal(err)
	}
	if stored.EventType != "futures_liquidation_aggregate" || stored.Symbol != "BTCUSDT" ||
		stored.LiquidationSide != "long" || stored.AggregateNotional != 6_000_000 ||
		stored.OrderCount != 3 || stored.WindowStart == 0 || stored.WindowEnd == 0 {
		t.Fatalf("unexpected structured notification: %+v", stored)
	}

	migrationAlias := fmt.Sprintf("webnotification_existing_schema_%d", notificationMigrationSeq.Add(1))
	if err := orm.RegisterDataBase(migrationAlias, "sqlite3", filepath.Join(t.TempDir(), "existing.db")); err != nil {
		t.Fatal(err)
	}
	migrationOrm := orm.NewOrmUsingDB(migrationAlias)
	_, err = migrationOrm.Raw(`CREATE TABLE notifications (
		id integer NOT NULL PRIMARY KEY AUTOINCREMENT,
		title varchar(255) NOT NULL,
		content text NOT NULL,
		module varchar(64) NOT NULL,
		level varchar(20) NOT NULL DEFAULT 'info',
		is_read integer NOT NULL DEFAULT 0,
		create_time bigint NOT NULL,
		read_time bigint NOT NULL DEFAULT 0
	)`).Exec()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := migrationOrm.Raw(`INSERT INTO notifications
		(title, content, module, level, is_read, create_time, read_time)
		VALUES ('existing', 'existing content', 'system', 'info', 0, 1, 0)`).Exec(); err != nil {
		t.Fatal(err)
	}
	if err := orm.RunSyncdb(migrationAlias, false, false); err != nil {
		t.Fatal(err)
	}
	var existing models.Notification
	if err := migrationOrm.QueryTable(new(models.Notification)).Filter("title", "existing").One(&existing); err != nil {
		t.Fatal(err)
	}
	if existing.EventType != "" || existing.AggregateNotional != 0 {
		t.Fatalf("existing notification should retain empty structured fields: %+v", existing)
	}
}
