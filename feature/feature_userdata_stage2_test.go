package feature

import (
	"testing"

	"github.com/beego/beego/v2/client/orm"
)

// Regression for Gate 2: the Main WS startup cleanup must not delete the
// Lead account's working orders or positions, even with matching symbols.
func TestStage2DeleteOldUserDataKeepsLeadMirror(t *testing.T) {
	setupFeatureTestORM(t)
	o := orm.NewOrm()
	symbol := "STAGE2CHECKUSDT"
	for _, account := range []string{"main", "lead"} {
		if _, err := o.Raw("INSERT INTO futures_positions(account_id,symbol,side,amount) VALUES(?,?,?,?)", account, symbol, "LONG", "1").Exec(); err != nil {
			t.Fatal(err)
		}
	}
	orders := []struct {
		account, id, status string
	}{
		{"main", "90101", "NEW"},
		{"main", "90102", "PARTIALLY_FILLED"},
		{"main", "90103", "FILLED"},
		{"lead", "90201", "NEW"},
		{"lead", "90202", "PARTIALLY_FILLED"},
		{"lead", "90203", "FILLED"},
	}
	for _, order := range orders {
		if _, err := o.Raw("INSERT INTO futures_orders(account_id,symbol,order_id,status) VALUES(?,?,?,?)", order.account, symbol, order.id, order.status).Exec(); err != nil {
			t.Fatal(err)
		}
	}
	t.Cleanup(func() {
		_, _ = o.Raw("DELETE FROM futures_positions WHERE symbol=?", symbol).Exec()
		_, _ = o.Raw("DELETE FROM futures_orders WHERE symbol=?", symbol).Exec()
	})
	count := func(table, condition string) int64 {
		t.Helper()
		var n int64
		if err := o.Raw("SELECT COUNT(*) FROM "+table+" WHERE symbol=? AND "+condition, symbol).QueryRow(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}
	if got := count("futures_orders", "account_id='lead'"); got != 3 {
		t.Fatalf("fixture has %d lead orders", got)
	}
	deleteOldUserData()
	checks := []struct {
		table, where string
		want         int64
	}{
		{"futures_positions", "account_id='main'", 0},
		{"futures_positions", "account_id='lead'", 1},
		{"futures_orders", "account_id='main' AND status='NEW'", 0},
		{"futures_orders", "account_id='main' AND status='PARTIALLY_FILLED'", 0},
		{"futures_orders", "account_id='main' AND status='FILLED'", 1},
		{"futures_orders", "account_id='lead' AND status='NEW'", 1},
		{"futures_orders", "account_id='lead' AND status='PARTIALLY_FILLED'", 1},
		{"futures_orders", "account_id='lead' AND status='FILLED'", 1},
	}
	for _, test := range checks {
		if got := count(test.table, test.where); got != test.want {
			t.Errorf("main WS cleanup changed %s where %s: rows=%d want=%d", test.table, test.where, got, test.want)
		}
	}
}
