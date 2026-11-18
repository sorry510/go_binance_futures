package coinmquant

import (
	"fmt"
	"github.com/beego/beego/v2/client/orm"
	"go_binance_futures/models"
	"strings"
	"testing"
)

func cm4Fixture(t *testing.T) orm.Ormer {
	t.Helper()
	o := fixture(t)
	ddls := []string{
		"CREATE TABLE coin_m_managed_positions (id INTEGER PRIMARY KEY AUTOINCREMENT, symbol VARCHAR(32), position_side VARCHAR(8), margin_asset VARCHAR(16), contract_size DECIMAL(36,18), entry_price DECIMAL(36,18), opened_contracts DECIMAL(36,18), remaining_contracts DECIMAL(36,18), realized_pnl DECIMAL(36,18), status VARCHAR(32), created_at BIGINT, updated_at BIGINT, closed_at BIGINT)",
		"CREATE TABLE coin_m_managed_orders (id INTEGER PRIMARY KEY AUTOINCREMENT, position_id BIGINT, symbol VARCHAR(32), position_side VARCHAR(8), intent VARCHAR(16), side VARCHAR(4), order_type VARCHAR(32), client_order_id VARCHAR(64) UNIQUE, exchange_order_id VARCHAR(64), requested_contracts DECIMAL(36,18), filled_contracts DECIMAL(36,18), status VARCHAR(32), created_at BIGINT, updated_at BIGINT)",
		"CREATE TABLE coin_m_trade_fills (id INTEGER PRIMARY KEY AUTOINCREMENT, position_id BIGINT, order_id BIGINT, symbol VARCHAR(32), position_side VARCHAR(8), side VARCHAR(4), exchange_order_id VARCHAR(64), trade_id VARCHAR(64), contracts DECIMAL(36,18), price DECIMAL(36,18), realized_pnl DECIMAL(36,18), pnl_asset VARCHAR(32), commission DECIMAL(36,18), commission_asset VARCHAR(32), trade_time BIGINT, UNIQUE(symbol,exchange_order_id,trade_id))",
	}
	for _, ddl := range ddls {
		if _, err := o.Raw(ddl).Exec(); err != nil {
			t.Fatal(err)
		}
	}
	return o
}
func cm4Position(t *testing.T, o orm.Ormer, id int, symbol, side, asset string, size, entry, open, remaining, realized, status string) {
	t.Helper()
	_, err := o.Raw("INSERT INTO coin_m_managed_positions (id,symbol,position_side,margin_asset,contract_size,entry_price,opened_contracts,remaining_contracts,realized_pnl,status,created_at,updated_at,closed_at) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?)",
		id, symbol, side, asset, size, entry, open, remaining, realized, status, 1000, 1000, 0).Exec()
	if err != nil {
		t.Fatal(err)
	}
}
func cm4Fill(t *testing.T, o orm.Ormer, pos, ord int, symbol, side, asset, intent, qty, realized, fee, feeAsset string) {
	t.Helper()
	exchangeID := fmt.Sprint(ord * 100)
	tradeID := fmt.Sprint(ord * 1000)
	_, err := o.Raw("INSERT INTO coin_m_managed_orders (id,position_id,symbol,position_side,intent,side,order_type,client_order_id,exchange_order_id,requested_contracts,filled_contracts,status,created_at,updated_at) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?)",
		ord, pos, symbol, side, intent, func() string {
			if (side == "LONG" && intent == "OPEN") || (side == "SHORT" && intent == "CLOSE") {
				return "BUY"
			}
			return "SELL"
		}(), "MARKET", fmt.Sprintf("cm4-%d", ord), exchangeID, qty, qty, "FILLED", 1000, 1000).Exec()
	if err != nil {
		t.Fatal(err)
	}
	orderSide := "SELL"
	if (side == "LONG" && intent == "OPEN") || (side == "SHORT" && intent == "CLOSE") {
		orderSide = "BUY"
	}
	fillPrice := "55000"
	if symbol == "ETHUSD_PERP" {
		if intent == "CLOSE" {
			fillPrice = "1800"
		} else {
			fillPrice = "2000"
		}
	}
	if symbol == "BTCUSD_PERP" && intent == "OPEN" {
		fillPrice = "50000"
	}
	_, err = o.Raw("INSERT INTO coin_m_trade_fills (position_id,order_id,symbol,position_side,side,exchange_order_id,trade_id,contracts,price,realized_pnl,pnl_asset,commission,commission_asset,trade_time) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?)",
		pos, ord, symbol, side, orderSide, exchangeID, tradeID, qty, fillPrice, realized, asset, fee, feeAsset, 1000).Exec()
	if err != nil {
		t.Fatal(err)
	}
}
func cm4Marks() map[string]Mark {
	return map[string]Mark{
		"BTCUSD_PERP": {Symbol: "BTCUSD_PERP", Price: "55000", Time: 1700000000000},
		"ETHUSD_PERP": {Symbol: "ETHUSD_PERP", Price: "1800", Time: 1700000000000},
	}
}
func TestCM4FilteredAssetTotalsAndCycleReconciliation(t *testing.T) {
	o := cm4Fixture(t)
	cm4Position(t, o, 1, "BTCUSD_PERP", "LONG", "BTC", "100", "50000", "10", "5", "0.000909090909090909", "OPEN")
	cm4Fill(t, o, 1, 11, "BTCUSD_PERP", "LONG", "BTC", "OPEN", "10", "0", "0.00003", "BTC")
	cm4Fill(t, o, 1, 12, "BTCUSD_PERP", "LONG", "BTC", "CLOSE", "5", "0.000909090909090909", "0.1", "USDT")
	cm4Position(t, o, 2, "ETHUSD_PERP", "SHORT", "ETH", "10", "2000", "4", "0", "0.002222222222222222", "CLOSED")
	cm4Fill(t, o, 2, 21, "ETHUSD_PERP", "SHORT", "ETH", "OPEN", "4", "0", "0.00005", "ETH")
	cm4Fill(t, o, 2, 22, "ETHUSD_PERP", "SHORT", "ETH", "CLOSE", "4", "0.002222222222222222", "0.00002", "ETH")
	page, err := PositionSummaries(o, LedgerFilter{Page: 1, Limit: 1}, cm4Marks())
	if err != nil {
		t.Fatal(err)
	}
	if page.Total != 2 || len(page.List) != 1 || len(page.Totals) != 2 {
		t.Fatalf("pagination/filtered totals must use all positions: %+v", page)
	}
	btc := page.List[0]
	// ID descending: ETH first. Check closed cycle and no invented historical USD conversion.
	if btc.ID != 2 || btc.PnLStatus != "verified" || btc.RealizedPnL != "0.002222222222222222" ||
		btc.UnrealizedPnL == nil || *btc.UnrealizedPnL != "0" || btc.TotalUSD == nil || !strings.HasPrefix(*btc.TotalUSD, "3.999") {
		t.Fatalf("closed ETH verified cycle incorrect: %+v", btc)
	}
	btcFiltered, err := PositionSummaries(o, LedgerFilter{Symbol: "BTCUSD_PERP", Page: 2, Limit: 1}, cm4Marks())
	if err != nil || btcFiltered.Total != 1 || len(btcFiltered.List) != 0 || len(btcFiltered.Totals) != 1 || btcFiltered.Totals[0].Asset != "BTC" {
		t.Fatalf("page 2 must have empty list while retaining filtered totals: %+v %v", btcFiltered, err)
	}
	btcPage, err := PositionSummaries(o, LedgerFilter{Symbol: "BTCUSD_PERP"}, cm4Marks())
	if err != nil {
		t.Fatal(err)
	}
	pos := btcPage.List[0]
	if pos.PnLStatus != "verified" || pos.OpenFilled != "10" || pos.CloseFilled != "5" ||
		pos.RealizedPnL != "0.000909090909090909" || pos.UnrealizedPnL == nil ||
		*pos.UnrealizedPnL != "0.000909090909090909" ||
		len(pos.FeeByAsset) != 2 || pos.FeeByAsset[0].Asset != "BTC" || pos.FeeByAsset[1].Asset != "USDT" ||
		pos.MarkPrice == nil || *pos.MarkPrice != "55000" {
		t.Fatalf("long partial close/fee currency/mark calculation incorrect: %+v", pos)
	}
	summary := btcPage.Totals[0]
	if summary.TotalPnL == nil || *summary.TotalPnL != "0.001818181818181818" ||
		summary.RealizedPnL != "0.000909090909090909" || summary.Unpriced != 0 ||
		summary.TotalUSD == nil || !strings.HasPrefix(*summary.TotalUSD, "99.999") ||
		len(summary.Fees) != 2 || summary.FundingIncluded {
		t.Fatalf("asset total must not mix fees/funding and USD settlement: %+v", summary)
	}
	ethFiltered, err := PositionSummaries(o, LedgerFilter{Symbol: "ETHUSD_PERP"}, nil)
	if err != nil || len(ethFiltered.Totals) != 1 || ethFiltered.Totals[0].Asset != "ETH" {
		t.Fatalf("ETH-only filter polluted by BTC: %+v %v", ethFiltered, err)
	}
}
func TestCM4MissingMarkAndFillIntegrityFailClosed(t *testing.T) {
	o := cm4Fixture(t)
	cm4Position(t, o, 1, "BTCUSD_PERP", "LONG", "BTC", "100", "50000", "10", "10", "0", "OPEN")
	cm4Fill(t, o, 1, 11, "BTCUSD_PERP", "LONG", "BTC", "OPEN", "10", "0", "0.0001", "BTC")
	page, err := PositionSummaries(o, LedgerFilter{}, nil)
	if err != nil || len(page.List) != 1 || page.List[0].PnLStatus != "unpriced" ||
		page.List[0].UnrealizedPnL != nil || page.Totals[0].TotalPnL != nil {
		t.Fatalf("missing mark must not generate zero or stale unrealized PnL: %+v %v", page, err)
	}
	wrong := map[string]Mark{"BTCUSD_PERP": {Symbol: "ETHUSD_PERP", Price: "55000", Time: 123}}
	page, err = PositionSummaries(o, LedgerFilter{}, wrong)
	if err != nil || page.List[0].UnrealizedPnL != nil {
		t.Fatalf("wrong-asset mark accepted: %+v %v", page, err)
	}
	// Malformed Fill metadata must not be included as a verified total.
	_, err = o.Raw("UPDATE coin_m_trade_fills SET pnl_asset='ETH' WHERE order_id=11").Exec()
	if err != nil {
		t.Fatal(err)
	}
	page, err = PositionSummaries(o, LedgerFilter{}, cm4Marks())
	if err != nil || page.List[0].PnLStatus != "invalid" || page.Totals[0].Invalid != 1 ||
		page.Totals[0].TotalPnL != nil {
		t.Fatalf("cross-asset Fill PnL was silently counted: %+v %v", page, err)
	}
}
func TestCM4PartialOrMissingEvidenceNeverPretendsVerified(t *testing.T) {
	o := cm4Fixture(t)
	cm4Position(t, o, 1, "BTCUSD_PERP", "LONG", "BTC", "100", "50000", "10", "5", "0", "OPEN")
	cm4Fill(t, o, 1, 11, "BTCUSD_PERP", "LONG", "BTC", "OPEN", "10", "0", "0", "BTC")
	page, err := PositionSummaries(o, LedgerFilter{}, cm4Marks())
	if err != nil || page.List[0].PnLStatus != "unverified_fills" || page.Totals[0].Invalid != 1 {
		t.Fatalf("missing close fill should block totals: %+v %v", page, err)
	}
	need, err := NeedMarks(o, LedgerFilter{Symbol: "BTCUSD_PERP"})
	if err != nil || !need {
		t.Fatalf("open position should request fresh mark: %t %v", need, err)
	}
	need, err = NeedMarks(o, LedgerFilter{Symbol: "ETHUSD_PERP"})
	if err != nil || need {
		t.Fatalf("missing position should not query DAPI: %t %v", need, err)
	}
	// Existing schema still has trade ID uniqueness without new CM-4 columns.
	if _, err := o.Raw("INSERT INTO coin_m_trade_fills (position_id,order_id,symbol,position_side,side,exchange_order_id,trade_id,contracts,price,realized_pnl,pnl_asset,commission,commission_asset,trade_time) SELECT position_id,order_id,symbol,position_side,side,exchange_order_id,trade_id,contracts,price,realized_pnl,pnl_asset,commission,commission_asset,trade_time FROM coin_m_trade_fills").Exec(); err == nil {
		t.Fatal("duplicate verified trade ID accepted")
	}
}
func TestCM4RawModelsKeepDecimalStrings(t *testing.T) {
	_ = models.CoinMManagedPosition{RealizedPnL: "0.000000000000000001"}
	n, err := InversePnL("LONG", "1", "1", "100000000", "200000000")
	if err != nil || n != "0.000000005" {
		t.Fatalf("small exact decimal inverse PnL lost precision: %q %v", n, err)
	}
}

func TestCM4PendingOpeningIsNotFakePnL(t *testing.T) {
	o := cm4Fixture(t)
	cm4Position(t, o, 1, "BTCUSD_PERP", "LONG", "BTC", "100", "0", "0", "0", "0", "PENDING_OPEN")
	need, err := NeedMarks(o, LedgerFilter{})
	if err != nil || need {
		t.Fatalf("pending empty cycle should not fetch mark: %v %v", need, err)
	}
	page, err := PositionSummaries(o, LedgerFilter{}, nil)
	if err != nil || len(page.List) != 1 || page.List[0].PnLStatus != "pending_fills" ||
		page.List[0].TotalPnL != nil || page.Totals[0].Pending != 1 ||
		page.Totals[0].TotalPnL != nil {
		t.Fatalf("pending open incorrectly classified as realized or invalid: %+v %v", page, err)
	}
}
