package binance

import (
	"sync"
	"testing"
	"time"

	"github.com/adshao/go-binance/v2/delivery"
	"github.com/beego/beego/v2/client/orm"
	_ "github.com/mattn/go-sqlite3"
)

var cm2TickerDBOnce sync.Once
var cm2TickerDBErr error

func TestCM2TickerWSUpdatesOnlyKnownPerpetualPrices(t *testing.T) {
	_ = orm.RegisterDriver("sqlite3", orm.DRSqlite)
	const alias = "coinm_cm2_ticker_db"
	cm2TickerDBOnce.Do(func() {
		cm2TickerDBErr = orm.RegisterDataBase(alias, "sqlite3", "file:cm2_ticker_ws?mode=memory&cache=shared")
	})
	if cm2TickerDBErr != nil {
		t.Fatal(cm2TickerDBErr)
	}
	db := orm.NewOrmUsingDB(alias)
	if _, err := db.Raw("DROP TABLE IF EXISTS delivery_symbols").Exec(); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Raw(`CREATE TABLE delivery_symbols (
		symbol TEXT PRIMARY KEY, contract_type TEXT NOT NULL, enable INTEGER NOT NULL,
		budget_usd DECIMAL(36,18) NOT NULL DEFAULT 0, strategy TEXT NOT NULL DEFAULT '',
		percentChange TEXT, close TEXT, open TEXT, low TEXT, high TEXT,
		updateTime BIGINT, baseVolume TEXT, quoteVolume TEXT, closeQty TEXT,
		tradeCount BIGINT, lastClose TEXT, lastUpdateTime BIGINT
	)`).Exec(); err != nil {
		t.Fatal(err)
	}
	for _, sym := range []struct{ symbol, contractType string }{
		{"BTCUSD_PERP", "PERPETUAL"},
		{"ETHUSD_PERP", "CURRENT_QUARTER"},
	} {
		if _, err := db.Raw("INSERT INTO delivery_symbols(symbol,contract_type,enable,budget_usd,strategy,close) VALUES (?,?,1,0,'{}','100')", sym.symbol, sym.contractType).Exec(); err != nil {
			t.Fatal(err)
		}
	}
	now := time.Now().UnixMilli()
	btc := &delivery.WsMarketTickerEvent{
		Symbol: "BTCUSD_PERP", Time: now, ClosePrice: "55000", OpenPrice: "50000",
		LowPrice: "49000", HighPrice: "56000", BaseVolume: "20", QuoteVolume: "1000000",
		CloseQty: "1", PriceChangePercent: "10", TradeCount: 200,
	}
	changed, err := persistCoinMTickers(db, delivery.WsAllMarketTickerEvent{
		nil, btc,
		&delivery.WsMarketTickerEvent{Symbol: "BTCUSD_261225", Time: now, ClosePrice: "65000"},
		&delivery.WsMarketTickerEvent{Symbol: "ETHUSD_PERP", Time: now, ClosePrice: "9000"},
		&delivery.WsMarketTickerEvent{Symbol: "SOLUSD_PERP", Time: now, ClosePrice: "190"},
		&delivery.WsMarketTickerEvent{Symbol: "BTCUSD_PERP", Time: now, ClosePrice: "not_a_number"},
	})
	if err != nil || changed != 1 {
		t.Fatalf("ticker updates=%d error=%v", changed, err)
	}
	var closeBTC, closeETH, btcStrategy, btcBudget string
	var btcEnabled int
	if err := db.Raw("SELECT close, enable, strategy, budget_usd FROM delivery_symbols WHERE symbol='BTCUSD_PERP'").QueryRow(&closeBTC, &btcEnabled, &btcStrategy, &btcBudget); err != nil {
		t.Fatal(err)
	}
	if err := db.Raw("SELECT close FROM delivery_symbols WHERE symbol='ETHUSD_PERP'").QueryRow(&closeETH); err != nil {
		t.Fatal(err)
	}
	if closeBTC != "55000" || closeETH != "100" || btcEnabled != 1 || btcStrategy != "{}" || btcBudget != "0" {
		t.Fatalf("market WS touched trading configuration or non-perp contract: BTC=%s ETH=%s enable=%d strategy=%s budget=%s", closeBTC, closeETH, btcEnabled, btcStrategy, btcBudget)
	}
	changed, err = persistCoinMTickers(db, delivery.WsAllMarketTickerEvent{
		&delivery.WsMarketTickerEvent{Symbol: "NONEXISTUSD_PERP", Time: now, ClosePrice: "123"},
	})
	if err != nil || changed != 0 {
		t.Fatalf("unknown symbol counted as valid market WS data: changed=%d err=%v", changed, err)
	}
}
