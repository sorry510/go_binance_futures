package controllers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"go_binance_futures/service/spotstrategy"

	beegocontext "github.com/beego/beego/v2/server/web/context"
	"github.com/golang-jwt/jwt/v5"
	_ "github.com/mattn/go-sqlite3"
)

// A private SQLite connection is supplied to the controller under test;
// never use the host's default ORM alias or any configured application DB.
func s4ControllerFixture(t *testing.T) (*sql.DB, int64, int64) {
	t.Helper()
	db, err := sql.Open("sqlite3", filepath.Join(t.TempDir(), "s4-rest.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	db.SetMaxOpenConns(4)
	for _, ddl := range []string{
		"CREATE TABLE config (id INTEGER PRIMARY KEY,spot_auto_trade_mode TEXT NOT NULL)",
		"CREATE TABLE spot_symbols (symbol TEXT PRIMARY KEY,type TEXT NOT NULL,strategy_trade_enabled INTEGER NOT NULL,strategy_budget_quote TEXT NOT NULL,strategy TEXT NOT NULL,technology TEXT NOT NULL,close TEXT NOT NULL,lastUpdateTime BIGINT NOT NULL)",
	} {
		if _, err = db.Exec(ddl); err != nil {
			t.Fatal(err)
		}
	}
	for _, ddl := range spotstrategy.SchemaFor(false) {
		if _, err = db.Exec(ddl); err != nil {
			t.Fatal(err)
		}
	}
	now := time.Now().UnixMilli()
	for _, q := range []string{
		"INSERT INTO config(id,spot_auto_trade_mode) VALUES(1,'legacy')",
		"INSERT INTO spot_symbols(symbol,type,strategy_trade_enabled,strategy_budget_quote,strategy,technology,close,lastUpdateTime) VALUES('BTCUSDT','USDT',0,'0','[]','{}','120',?)",
	} {
		if _, err = db.Exec(q, now); err != nil {
			// first insert takes no timestamp argument
			if q == "INSERT INTO config(id,spot_auto_trade_mode) VALUES(1,'legacy')" {
				_, err = db.Exec(q)
			}
			if err != nil {
				t.Fatalf("fixture insert %s: %v", q, err)
			}
		}
	}
	result, err := db.Exec(`INSERT INTO spot_strategy_holdings
 (account_id,owner,symbol,base_asset,quote_asset,owned_qty,avg_cost_quote,status,active_slot,created_at,updated_at)
 VALUES('main','auto_strategy','BTCUSDT','BTC','USDT','1','100','active','main:auto_strategy:BTCUSDT',?,?)`, now, now)
	if err != nil {
		t.Fatal(err)
	}
	holdingID, err := result.LastInsertId()
	if err != nil {
		t.Fatal(err)
	}
	result, err = db.Exec(`INSERT INTO spot_strategy_orders
 (holding_id,account_id,owner,symbol,side,order_type,intent,client_order_id,exchange_order_id,signal_key,
 requested_qty,filled_qty,executed_quote_qty,reserved_qty,status,last_error,created_at,updated_at)
 VALUES (?,'main','auto_strategy','BTCUSDT','BUY','MARKET','open','s4-http-cid','999','s4:http',
 '1','1','100','0','filled','signature=DO_NOT_LEAK',?,?)`, holdingID, now, now)
	if err != nil {
		t.Fatal(err)
	}
	orderID, err := result.LastInsertId()
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`INSERT INTO spot_strategy_fills
 (account_id,symbol,order_id,trade_id,holding_id,client_order_id,side,qty,price,quote_qty,commission,commission_asset,trade_time,rest_verified)
 VALUES('main','BTCUSDT','999','71',?,'s4-http-cid','BUY','1','100','100','0','USDT',?,1)`, holdingID, now)
	if err != nil {
		t.Fatal(err)
	}
	return db, holdingID, orderID
}

func s4CallSuccessfulHandler(t *testing.T, db *sql.DB, url string, call func(*SpotStrategyController)) (int, map[string]json.RawMessage) {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, url, nil)
	req.Header.Set("Accept", "application/yaml")
	rec := httptest.NewRecorder()
	ctx := beegocontext.NewContext()
	ctx.Reset(rec, req)
	ctx.Input.SetData("user", jwt.MapClaims{"Username": "s4-operator"})
	ctrl := &SpotStrategyController{readDB: db}
	ctrl.Init(ctx, "SpotStrategyController", "", nil)
	call(ctrl)
	var result struct {
		Code int                        `json:"code"`
		Msg  string                     `json:"msg"`
		Data map[string]json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatalf("non-JSON response status=%d body=%s err=%v", rec.Code, rec.Body.String(), err)
	}
	if rec.Header().Get("Content-Type") != "application/json" && rec.Header().Get("Content-Type") != "application/json; charset=utf-8" {
		t.Fatalf("Accept header changed JSON Content-Type %s", rec.Header().Get("Content-Type"))
	}
	if rec.Code != result.Code {
		t.Fatalf("status code mismatch HTTP=%d JSON=%d", rec.Code, result.Code)
	}
	return rec.Code, result.Data
}
func TestS4ControllerPositive200JSONAndScopedDTO(t *testing.T) {
	db, hid, oid := s4ControllerFixture(t)
	cases := []struct {
		name, url string
		call      func(*SpotStrategyController)
		want      string
	}{
		{"config", "/spots/symbols?account_id=main&owner=auto_strategy", (*SpotStrategyController).ListSymbols, "symbol"},
		{"holding", "/spots/strategy/holdings?account_id=main&owner=auto_strategy", (*SpotStrategyController).ListHoldings, "holding_id"},
		{"order", "/spots/strategy/orders?account_id=main&owner=auto_strategy&holding_id=" + strconv.FormatInt(hid, 10), (*SpotStrategyController).ListOrders, "client_order_id"},
		{"fill", "/spots/strategy/fills?account_id=main&owner=auto_strategy&order_id=" + strconv.FormatInt(oid, 10), (*SpotStrategyController).ListFills, "trade_id"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			status, data := s4CallSuccessfulHandler(t, db, tc.url, tc.call)
			if status != 200 {
				t.Fatalf("expected 200, got %d", status)
			}
			var items []map[string]any
			if err := json.Unmarshal(data["items"], &items); err != nil || len(items) != 1 {
				t.Fatalf("unexpected DTO items=%v err=%v", items, err)
			}
			if _, ok := items[0][tc.want]; !ok {
				t.Fatalf("missing contract field %q: %+v", tc.want, items[0])
			}
			if _, unsafe := items[0]["last_error"]; unsafe {
				t.Fatal("raw signed API error leaked")
			}
			if _, unsafe := items[0]["signature"]; unsafe {
				t.Fatal("signature leaked")
			}
			if tc.name == "holding" {
				if items[0]["reference_price_quote"] != "120" ||
					items[0]["unrealized_estimate_quote"] != "20" ||
					items[0]["reference_price_fresh"] != true {
					t.Fatalf("fresh mark with verified Fill not calculated: %+v", items[0])
				}
			}
		})
	}
	status, _ := s4CallSuccessfulHandler(t, db, "/spots/strategy/fills?account_id=main&owner=auto_strategy&order_id="+strconv.FormatInt(oid, 10)+"&symbol=ETHUSDT", (*SpotStrategyController).ListFills)
	if status != 200 {
		t.Fatalf("wrong-symbol filter returned %d", status)
	}
}
func TestS4ControllerRejectIncompleteAsHoldingStatus(t *testing.T) {
	db, _, _ := s4ControllerFixture(t)
	status, _ := s4CallSuccessfulHandler(t, db, "/spots/strategy/holdings?account_id=main&owner=auto_strategy&status=incomplete", (*SpotStrategyController).ListHoldings)
	if status != 400 {
		t.Fatalf("incomplete is data quality, not holding status; got %d", status)
	}
}

// The UI's version handshake and live state must be generated by the backend
// gate, not a controller literal that can drift after code changes.
func TestS4ControllerModeContractAndGateCoupled(t *testing.T) {
	for _, mode := range []string{spotstrategy.ModeLegacy, spotstrategy.ModeStrategy, spotstrategy.ModeOff} {
		data := spotModeStatus(mode)
		execution, approved, reason := spotstrategy.LiveStatus(mode)
		if data["contract_version"] != 4 || data["mode"] != mode ||
			data["execution"] != execution || data["live_approved"] != approved || data["reason"] != reason {
			t.Fatalf("S4 status drift: %v / mode=%s", data, mode)
		}
	}
}

// ListConfigs requires only authentication and a spot_symbols table. An
// absent trade-mode row and arbitrary legacy account query values must not
// influence the symbol result. Private Holdings keep their scope validation.
func TestS4ControllerConfigsWithoutAccountMode(t *testing.T) {
	db, _, _ := s4ControllerFixture(t)
	if _, err := db.Exec("DELETE FROM config"); err != nil {
		t.Fatal(err)
	}
	for _, url := range []string{
		"/spots/symbols",
		"/spots/symbols?account_id=lead&owner=manual",
	} {
		status, data := s4CallSuccessfulHandler(t, db, url, (*SpotStrategyController).ListSymbols)
		if status != 200 {
			t.Fatalf("symbol API wrongly required account or mode: status=%d data=%+v", status, data)
		}
		var items []map[string]any
		if err := json.Unmarshal(data["items"], &items); err != nil || len(items) != 1 {
			t.Fatalf("invalid list: %+v err=%v", items, err)
		}
		for _, name := range []string{"account_id", "owner", "execution", "mode"} {
			if _, ok := items[0][name]; ok {
				t.Fatalf("symbol DTO unexpectedly exposed %q", name)
			}
		}
	}
}

// Owners are an internal inventory guard, not a required UI parameter.
// The omission is safe only because ListHoldings/ListOrders/ListFills
// still query the server's fixed AccountMain/OwnerAuto with scoped SQL.
func TestS4AccountEndpointsDefaultServerOwnedScope(t *testing.T) {
	db, hid, oid := s4ControllerFixture(t)
	tests := []struct {
		path string
		call func(*SpotStrategyController)
	}{
		{"/spots/strategy/holdings", (*SpotStrategyController).ListHoldings},
		{"/spots/strategy/orders?holding_id=" + strconv.FormatInt(hid, 10), (*SpotStrategyController).ListOrders},
		{"/spots/strategy/fills?order_id=" + strconv.FormatInt(oid, 10), (*SpotStrategyController).ListFills},
	}
	for _, tc := range tests {
		status, data := s4CallSuccessfulHandler(t, db, tc.path, tc.call)
		if status != http.StatusOK {
			t.Fatalf("missing account fields rejected on %s: %d", tc.path, status)
		}
		var items []map[string]any
		if err := json.Unmarshal(data["items"], &items); err != nil || len(items) != 1 {
			t.Fatalf("expected one owned item %s: %v %+v", tc.path, err, items)
		}
		// Explicit malicious scope selection must STILL be rejected. Not
		// sending Account fields does not permit a second account/owner.
		for _, suffix := range []string{"account_id=lead&owner=auto_strategy", "account_id=main&owner=manual", "account_id=main"} {
			separator := "?"
			if strings.Contains(tc.path, "?") {
				separator = "&"
			}
			status, _ = s4CallSuccessfulHandler(t, db, tc.path+separator+suffix, tc.call)
			if status != http.StatusBadRequest {
				t.Fatalf("scope override accepted on %s with %s: %d", tc.path, suffix, status)
			}
		}
	}
}
