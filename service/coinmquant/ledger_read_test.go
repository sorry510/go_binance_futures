package coinmquant

import (
	"testing"
)

func TestCM3LocalLedgerReadOnlyAndSymbolFilter(t *testing.T) {
	o := fixture(t)
	tables := []string{
		"CREATE TABLE coin_m_managed_positions (id INTEGER PRIMARY KEY AUTOINCREMENT, symbol VARCHAR(32), position_side VARCHAR(8), margin_asset VARCHAR(16), contract_size DECIMAL(36,18), entry_price DECIMAL(36,18), opened_contracts DECIMAL(36,18), remaining_contracts DECIMAL(36,18), realized_pnl DECIMAL(36,18), status VARCHAR(32), created_at BIGINT, updated_at BIGINT, closed_at BIGINT)",
		"CREATE TABLE coin_m_managed_orders (id INTEGER PRIMARY KEY AUTOINCREMENT, position_id BIGINT, symbol VARCHAR(32), position_side VARCHAR(8), intent VARCHAR(16), side VARCHAR(4), order_type VARCHAR(32), client_order_id VARCHAR(64), exchange_order_id VARCHAR(64), requested_contracts DECIMAL(36,18), filled_contracts DECIMAL(36,18), status VARCHAR(32), created_at BIGINT, updated_at BIGINT)",
		"CREATE TABLE coin_m_trade_fills (id INTEGER PRIMARY KEY AUTOINCREMENT, position_id BIGINT, order_id BIGINT, symbol VARCHAR(32), position_side VARCHAR(8), side VARCHAR(4), exchange_order_id VARCHAR(64), trade_id VARCHAR(64), contracts DECIMAL(36,18), price DECIMAL(36,18), realized_pnl DECIMAL(36,18), pnl_asset VARCHAR(32), commission DECIMAL(36,18), commission_asset VARCHAR(32), trade_time BIGINT)",
	}
	for _, ddl := range tables {
		if _, err := o.Raw(ddl).Exec(); err != nil {
			t.Fatal(err)
		}
	}
	_, err := o.Raw("INSERT INTO coin_m_managed_positions(id,symbol,position_side,margin_asset,contract_size,entry_price,opened_contracts,remaining_contracts,realized_pnl,status,created_at,updated_at,closed_at) VALUES(1,'BTCUSD_PERP','LONG','BTC',100,50000,10,5,0.001,'OPEN',100,101,0)").Exec()
	if err != nil {
		t.Fatal(err)
	}
	_, err = o.Raw("INSERT INTO coin_m_managed_orders(id,position_id,symbol,position_side,intent,side,order_type,client_order_id,exchange_order_id,requested_contracts,filled_contracts,status,created_at,updated_at) VALUES(2,1,'BTCUSD_PERP','LONG','OPEN','BUY','LIMIT','cm-open-2',NULL,10,0,'NEW',100,101)").Exec()
	if err != nil {
		t.Fatal(err)
	}
	_, err = o.Raw("INSERT INTO coin_m_trade_fills(id,position_id,order_id,symbol,position_side,side,exchange_order_id,trade_id,contracts,price,realized_pnl,pnl_asset,commission,commission_asset,trade_time) VALUES(3,1,2,'BTCUSD_PERP','LONG','BUY','4001','123',10,50000,0.001,'BTC',0.0001,'BTC',110)").Exec()
	if err != nil {
		t.Fatal(err)
	}
	page, err := Positions(o, LedgerFilter{Symbol: "BTCUSD_PERP", Page: 1, Limit: 20})
	if err != nil || page.Total != 1 || len(page.List) != 1 || page.List[0].ID != 1 || page.List[0].PositionSide != "LONG" || page.List[0].MarginAsset != "BTC" {
		t.Fatalf("local CM position query not isolated: %+v %v", page, err)
	}
	filtered, err := Positions(o, LedgerFilter{Symbol: "ETHUSD_PERP"})
	if err != nil || filtered.Total != 0 {
		t.Fatalf("unexpected non-matching symbol: %+v %v", filtered, err)
	}
	orders, err := Orders(o, LedgerFilter{Symbol: "BTCUSD_PERP"})
	if err != nil || orders.Total != 1 || len(orders.List) != 1 || orders.List[0].ClientOrderID != "cm-open-2" || orders.List[0].ExchangeOrderID != "" {
		t.Fatalf("pending CM order should preserve NULL as empty ID: %+v %v", orders, err)
	}
	detail, err := Detail(o, 1)
	if err != nil || detail.Position.Symbol != "BTCUSD_PERP" || len(detail.Orders) != 1 || len(detail.Fills) != 1 ||
		detail.Fills[0].TradeID != "123" || detail.Fills[0].PnLAsset != "BTC" || detail.Fills[0].CommissionAsset != "BTC" {
		t.Fatalf("COIN-M local position / fill detail incorrect: %+v %v", detail, err)
	}
	if _, err := Detail(o, 999); err == nil {
		t.Fatal("unknown local CM position unexpectedly succeeded")
	}
	if _, err := Positions(o, LedgerFilter{Status: "INVALID STATUS"}); err == nil {
		t.Fatal("bad status accepted")
	}
}
