package coinmexecution

import (
	"testing"
)

func TestCM7RuntimeSummaryIsReadOnlyAndNoSignedDAPI(t *testing.T) {
	db := cm5Fixture(t)
	summary, err := ReadRuntimeSummary(db)
	if err != nil {
		t.Fatal(err)
	}
	if !summary.ReadOnly || summary.TradeActionsAvailable ||
		summary.PrivateWSVerified || summary.WalletVerified ||
		!summary.GlobalEntryEnabled || summary.EnabledPerpetualSymbols != 1 ||
		summary.ManagedCyclesNeedingWork != 0 || summary.OrdersNeedingReconciliation != 0 {
		t.Fatalf("wrong CM7 initial local-only status: %+v", summary)
	}
	if _, err = db.Raw("UPDATE config SET coin_m_quant_enable=0").Exec(); err != nil {
		t.Fatal(err)
	}
	if _, err = db.Raw("UPDATE delivery_symbols SET enable=0").Exec(); err != nil {
		t.Fatal(err)
	}
	off, err := ReadRuntimeSummary(db)
	if err != nil || off.GlobalEntryEnabled || off.EnabledPerpetualSymbols != 0 {
		t.Fatalf("OFF switches not reflected: %+v %v", off, err)
	}
	_, err = db.Raw("INSERT INTO coin_m_managed_positions (symbol,position_side,margin_asset,contract_size,entry_price,opened_contracts,remaining_contracts,realized_pnl,status,created_at,updated_at,closed_at) VALUES('BTCUSD_PERP','LONG','BTC',100,50000,10,10,0,'OPEN',1,1,0)").Exec()
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Raw("INSERT INTO coin_m_managed_orders (position_id,symbol,position_side,intent,side,order_type,client_order_id,requested_contracts,filled_contracts,status,created_at,updated_at) VALUES(1,'BTCUSD_PERP','LONG','OPEN','BUY','MARKET','cm7pendingfill',10,0,'AWAITING_FILLS',1,1)").Exec()
	if err != nil {
		t.Fatal(err)
	}
	active, err := ReadRuntimeSummary(db)
	if err != nil || active.GlobalEntryEnabled || active.EnabledPerpetualSymbols != 0 ||
		active.ManagedCyclesNeedingWork != 1 || active.OrdersNeedingReconciliation != 1 ||
		!active.ReadOnly || active.PrivateWSVerified {
		t.Fatalf("OFF must still report pending owned recovery, never claim online account: %+v %v", active, err)
	}
}
func TestCM7RuntimeSummaryMissingConfigFailsClosed(t *testing.T) {
	db := cm5Fixture(t)
	if _, err := db.Raw("DROP TABLE config").Exec(); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadRuntimeSummary(db); err == nil {
		t.Fatal("absent CM global config incorrectly represented as disabled, no failure")
	}
}
