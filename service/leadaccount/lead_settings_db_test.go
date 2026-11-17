package leadaccount

import (
	"github.com/beego/beego/v2/client/orm"
	"testing"
)

// Test-only SQLite schema; never sync or touch a deployment database.
func prepareLeadSettingsTestDB(t *testing.T) {
	t.Helper()
	prepareLeadExecutionDB(t)
	o := orm.NewOrm()
	if _, err := o.Raw(LeadTradeSettingsDDL).Exec(); err != nil {
		t.Fatal(err)
	}
	if _, err := o.Raw("DELETE FROM lead_trade_settings").Exec(); err != nil {
		t.Fatal(err)
	}
}
