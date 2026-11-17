package feature

import (
	"context"
	"strings"
	"sync"
	"testing"

	"go_binance_futures/models"

	"github.com/adshao/go-binance/v2/delivery"
	"github.com/beego/beego/v2/client/orm"
	_ "github.com/mattn/go-sqlite3"
)

func coinMTestSymbol(symbol, contractType, status string, size int) delivery.Symbol {
	return delivery.Symbol{
		Symbol: symbol, ContractType: contractType, ContractStatus: status,
		QuoteAsset: "USD", MarginAsset: "BTC", ContractSize: size,
		Filters: []map[string]any{
			{"filterType": "PRICE_FILTER", "minPrice": "0.1", "maxPrice": "1000000", "tickSize": "0.1"},
			{"filterType": "LOT_SIZE", "minQty": "1", "maxQty": "100", "stepSize": "1"},
			{"filterType": "MARKET_LOT_SIZE", "minQty": "2", "maxQty": "100", "stepSize": "2"},
		},
	}
}

func TestCM2PerpetualContractFiltersAndQuantity(t *testing.T) {
	s := coinMTestSymbol("BTCUSD_PERP", "PERPETUAL", "TRADING", 100)
	spec, err := ParseCoinMPerpetualSpec(s)
	if err != nil || spec.ContractSize != 100 || spec.MarginAsset != "BTC" || spec.StepSize != "1" ||
		spec.MarketStepSize != "2" || spec.MinQty != "1" {
		t.Fatalf("wrong BTCUSD_PERP metadata %+v error %v", spec, err)
	}
	for _, tt := range []struct {
		quantity, price   string
		market, wantValid bool
	}{
		{"10", "", true, true},
		{"11", "", true, false}, // market step=2, even though LIMIT step=1
		{"11", "50000.1", false, true},
		{"10.5", "50000.1", false, false},
		{"1", "", true, false}, // market minimum=2
		{"0", "", true, false},
		{"101", "", true, false},
		{"10", "50000.11", false, false},
		{"10", "50000.1", false, true},
		{"10", "0", false, false},
	} {
		err := ValidateCoinMOrderFilters(spec, tt.quantity, tt.price, tt.market)
		if (err == nil) != tt.wantValid {
			t.Errorf("quantity=%s price=%s market=%t accepted=%t want=%t: %v",
				tt.quantity, tt.price, tt.market, err == nil, tt.wantValid, err)
		}
	}
	for _, bad := range []delivery.Symbol{
		coinMTestSymbol("BTCUSD_261225", "CURRENT_QUARTER", "TRADING", 100),
		coinMTestSymbol("BTCUSD_PERP", "PERPETUAL", "SETTLING", 100),
		coinMTestSymbol("BTCUSD_PERP", "PERPETUAL", "TRADING", 0),
	} {
		if _, err := ParseCoinMPerpetualSpec(bad); err == nil {
			t.Fatalf("accepted ineligible symbol %+v", bad)
		}
	}
	invalid := coinMTestSymbol("BTCUSD_PERP", "PERPETUAL", "TRADING", 100)
	invalid.Filters = []map[string]any{{"filterType": "LOT_SIZE", "minQty": "1", "maxQty": "10", "stepSize": "1"}}
	if _, err := ParseCoinMPerpetualSpec(invalid); err == nil {
		t.Fatal("accepted missing PRICE/MARKET filters")
	}
	invalid = coinMTestSymbol("BTCUSD_PERP", "PERPETUAL", "TRADING", 100)
	invalid.Filters[1]["stepSize"] = "garbage"
	if _, err := ParseCoinMPerpetualSpec(invalid); err == nil {
		t.Fatal("accepted bad step size")
	}
	invalid.Filters[1]["stepSize"] = 1 // malformed JSON type must not panic
	if _, err := ParseCoinMPerpetualSpec(invalid); err == nil {
		t.Fatal("accepted non-string step size")
	}
}

var coinMFixtureOnce sync.Once
var coinMFixtureErr error

func coinMTestDB(t *testing.T) orm.Ormer {
	t.Helper()
	_ = orm.RegisterDriver("sqlite3", orm.DRSqlite)
	coinMFixtureOnce.Do(func() {
		orm.RegisterModel(new(models.DeliverySymbols))
		coinMFixtureErr = orm.RegisterDataBase(
			"feature_coinm_cm2", "sqlite3", "file:feature_coinm_cm2?mode=memory&cache=shared")
		if coinMFixtureErr == nil {
			coinMFixtureErr = orm.RunSyncdb("feature_coinm_cm2", false, false)
		}
	})
	if coinMFixtureErr != nil {
		t.Fatal(coinMFixtureErr)
	}
	o := orm.NewOrmUsingDB("feature_coinm_cm2")
	if _, err := o.Raw("DELETE FROM delivery_symbols").Exec(); err != nil {
		t.Fatal(err)
	}
	return o
}

func TestCM2ExchangeInfoRefreshPreservesLocalTradingConfiguration(t *testing.T) {
	o := coinMTestDB(t)
	// One historical, enabled row. Neither strategy nor old USDT budget may change.
	historical := &models.DeliverySymbols{
		Symbol: "BTCUSD_PERP", Enable: 1, Usdt: "10", Strategy: "{\"custom\":1}",
		Technology: "{\"MA\":[]}", Leverage: 4, Type: "PERP",
	}
	if _, err := o.Insert(historical); err != nil {
		t.Fatal(err)
	}
	info := &delivery.ExchangeInfo{Symbols: []delivery.Symbol{
		coinMTestSymbol("BTCUSD_PERP", "PERPETUAL", "TRADING", 100),
		coinMTestSymbol("ETHUSD_PERP", "PERPETUAL", "TRADING", 10),
		coinMTestSymbol("BTCUSD_261225", "CURRENT_QUARTER", "TRADING", 100),
		coinMTestSymbol("LTCUSD_PERP", "PERPETUAL", "SETTLING", 50),
	}}
	for i := 0; i < 2; i++ {
		count, err := RefreshCoinMPerpetualSymbols(context.Background(), o, info)
		if err != nil || count != 2 {
			t.Fatalf("refresh #%d count=%d: %v", i, count, err)
		}
	}
	var all []models.DeliverySymbols
	if _, err := o.QueryTable(new(models.DeliverySymbols)).All(&all); err != nil || len(all) != 2 {
		t.Fatalf("expected only two currently trading perpetuals; got %+v err %v", all, err)
	}
	var old models.DeliverySymbols
	if err := o.QueryTable(new(models.DeliverySymbols)).Filter("Symbol", "BTCUSD_PERP").One(&old); err != nil {
		t.Fatal(err)
	}
	if old.Enable != 1 || old.Usdt != "10" || old.Strategy != historical.Strategy ||
		old.Technology != historical.Technology || old.Leverage != 4 ||
		old.ContractSize != 100 || old.MarginAsset != "BTC" || old.ContractType != "PERPETUAL" || old.StepSize != "1" {
		t.Fatalf("refresh overwrote old trading config or metadata: %+v", old)
	}
	var fresh models.DeliverySymbols
	if err := o.QueryTable(new(models.DeliverySymbols)).Filter("Symbol", "ETHUSD_PERP").One(&fresh); err != nil {
		t.Fatal(err)
	}
	if fresh.Enable != 0 || fresh.Usdt != "0" || fresh.ContractSize != 10 || fresh.MarginAsset != "BTC" {
		t.Fatalf("new perpetual not safely initialized: %+v", fresh)
	}
	// On malformed exchangeInfo all writes must be rejected before transaction.
	bad := coinMTestSymbol("BTCUSD_PERP", "PERPETUAL", "TRADING", 0)
	_, err := RefreshCoinMPerpetualSymbols(context.Background(), o, &delivery.ExchangeInfo{Symbols: []delivery.Symbol{
		coinMTestSymbol("NEWUSD_PERP", "PERPETUAL", "TRADING", 100), bad,
	}})
	if err == nil {
		t.Fatal("partially valid exchangeInfo was accepted")
	}
	var count int
	if err := o.Raw("SELECT COUNT(*) FROM delivery_symbols").QueryRow(&count); err != nil || count != 2 {
		t.Fatalf("failed refresh inserted unwanted symbol: count=%d err=%v", count, err)
	}
	// Underlying API must not silently fall back to an unvalidated contract.
	if !strings.Contains(err.Error(), "missing contract size") {
		t.Fatalf("unexpected failure: %v", err)
	}
}
