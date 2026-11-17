package controllers

import (
	"github.com/beego/beego/v2/client/orm"
	_ "github.com/mattn/go-sqlite3"
	"go_binance_futures/service/leadaccount"
	"testing"
)

func prepareLeadControllerSettingsDB(t *testing.T) {
	t.Helper()
	_ = orm.RegisterDriver("sqlite3", orm.DRSqlite)
	if _, err := orm.GetDB("default"); err != nil {
		if err := orm.RegisterDataBase("default", "sqlite3", "file:controllers_stable_default_test?mode=memory&cache=shared"); err != nil {
			t.Fatal(err)
		}
	}
	o := orm.NewOrm()
	if _, err := o.Raw(leadaccount.LeadTradeSettingsDDL).Exec(); err != nil {
		t.Fatal(err)
	}
	if _, err := o.Raw("DELETE FROM lead_trade_settings").Exec(); err != nil {
		t.Fatal(err)
	}
}
