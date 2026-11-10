package controllers

import (
	"encoding/json"
	"net/http"
	"testing"
)

// JWT-scoped handler checks that the query is applied by SQLite before the
// result page is constructed; an invalid quote gets a stable JSON 400.
func TestS4ControllerQuoteAssetFilterScopedAndValidated(t *testing.T) {
	db, _, _ := s4ControllerFixture(t)
	for _, sqlText := range []string{
		"INSERT INTO spot_symbols(symbol,type,strategy_trade_enabled,strategy_budget_quote,strategy,technology,close,lastUpdateTime) VALUES('ETHUSDC','USDC',0,'0','[]','{}','1',0)",
		"INSERT INTO spot_symbols(symbol,type,strategy_trade_enabled,strategy_budget_quote,strategy,technology,close,lastUpdateTime) VALUES('SOLUSDT','USDT',0,'0','[]','{}','1',0)",
	} {
		if _, err := db.Exec(sqlText); err != nil {
			t.Fatal(err)
		}
	}
	url := "/spots/symbols?account_id=main&owner=auto_strategy&quote_asset=USDC&limit=1"
	status, data := s4CallSuccessfulHandler(t, db, url, (*SpotStrategyController).ListSymbols)
	if status != http.StatusOK {
		t.Fatalf("valid quote got HTTP %d", status)
	}
	var total int64
	if err := json.Unmarshal(data["total"], &total); err != nil || total != 1 {
		t.Fatalf("USDC total wrong %d err=%v", total, err)
	}
	var items []map[string]any
	if err := json.Unmarshal(data["items"], &items); err != nil || len(items) != 1 || items[0]["symbol"] != "ETHUSDC" {
		t.Fatalf("wrong quote items %+v %v", items, err)
	}
	status, _ = s4CallSuccessfulHandler(t, db, url+"&status=invalid", (*SpotStrategyController).ListSymbols)
	if status != http.StatusBadRequest {
		t.Fatalf("invalid status accepted %d", status)
	}
	status, _ = s4CallSuccessfulHandler(t, db, "/spots/symbols?account_id=main&owner=auto_strategy&quote_asset=BUSD", (*SpotStrategyController).ListSymbols)
	if status != http.StatusBadRequest {
		t.Fatalf("invalid quote accepted %d", status)
	}
}
func TestS4SymbolsDoNotUseModeCapabilities(t *testing.T) {
	db, _, _ := s4ControllerFixture(t)
	status, data := s4CallSuccessfulHandler(t, db,
		"/spots/symbols?quote_asset=USDT", (*SpotStrategyController).ListSymbols)
	if status != http.StatusOK || data["total"] == nil {
		t.Fatalf("symbol listing unavailable: status=%d data=%+v", status, data)
	}
	if _, present := spotModeStatus("legacy")["capabilities"]; present {
		t.Fatal("symbol filtering must not require a private trade-mode API")
	}
}

// The authenticated Spot symbol endpoint must expose substring matching,
// including lowercase and single-character fragments, before SQL pagination.
func TestS4ControllerFuzzySymbolSearch(t *testing.T) {
	db, _, _ := s4ControllerFixture(t)
	if _, err := db.Exec("INSERT INTO spot_symbols(symbol,type,strategy_trade_enabled,strategy_budget_quote,strategy,technology,close,lastUpdateTime) VALUES('WBTCUSDT','USDT',0,'0','[]','{}','1',0)"); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		query string
		total int64
	}{
		{"btc", 2}, {"B", 2}, {"wbtc", 1}, {"Eth", 0},
	} {
		status, data := s4CallSuccessfulHandler(t, db, "/spots/symbols?quote_asset=USDT&symbol="+tc.query+"&limit=1", (*SpotStrategyController).ListSymbols)
		if status != http.StatusOK {
			t.Fatalf("search %q HTTP %d", tc.query, status)
		}
		var total int64
		if err := json.Unmarshal(data["total"], &total); err != nil || total != tc.total {
			t.Fatalf("search=%q total=%d expected=%d err=%v", tc.query, total, tc.total, err)
		}
		if tc.total > 1 {
			status, page2 := s4CallSuccessfulHandler(t, db, "/spots/symbols?quote_asset=USDT&symbol="+tc.query+"&limit=1&page=2", (*SpotStrategyController).ListSymbols)
			var items []map[string]any
			if err := json.Unmarshal(page2["items"], &items); status != http.StatusOK || err != nil || len(items) != 1 || items[0]["symbol"] != "WBTCUSDT" {
				t.Fatalf("search=%q page2 invalid: status=%d items=%+v err=%v", tc.query, status, items, err)
			}
		}
	}
	for _, fragment := range []string{"B%25", "B_", "B%27%20OR%201%3D1"} {
		status, _ := s4CallSuccessfulHandler(t, db, "/spots/symbols?symbol="+fragment, (*SpotStrategyController).ListSymbols)
		if status != http.StatusBadRequest {
			t.Fatalf("unsafe symbol filter %q accepted: %d", fragment, status)
		}
	}
}
