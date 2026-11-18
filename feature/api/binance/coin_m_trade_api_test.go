package binance

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/adshao/go-binance/v2/delivery"
)

func TestCM5SignedDAPIOrderAlgoAndUserTradesHostIsolation(t *testing.T) {
	oldClient, oldMode, oldErr := deliveryClient, coinMRuntime, coinMSetupErr
	defer func() { deliveryClient, coinMRuntime, coinMSetupErr = oldClient, oldMode, oldErr }()
	var mu sync.Mutex
	hits := map[string]int{}
	secret := "CM5-MOCK-SIGNING-SECRET"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		hits[r.Method+" "+r.URL.Path]++
		mu.Unlock()
		if r.Header.Get("X-MBX-APIKEY") != "CM5-MOCK-TESTNET" {
			t.Errorf("DAPI credentials leaked/switched: %q", r.Header.Get("X-MBX-APIKEY"))
		}
		if err := r.ParseForm(); err != nil {
			t.Error(err)
			return
		}
		// Installed SDK signs queryString + bodyString (without a
		// delimiter). The CM-5 raw helper signs body-only for its POSTs.
		query := r.URL.Query()
		body := r.PostForm
		signature := query.Get("signature")
		if signature == "" {
			signature = body.Get("signature")
		}
		query.Del("signature")
		body.Del("signature")
		mac := hmac.New(sha256.New, []byte(secret))
		_, _ = io.WriteString(mac, query.Encode()+body.Encode())
		expected := hex.EncodeToString(mac.Sum(nil))
		if !hmac.Equal([]byte(expected), []byte(signature)) {
			t.Errorf("COIN-M invalid HMAC signature %s %s", r.Method, r.URL.Path)
		}
		if r.Form.Get("timestamp") == "" {
			t.Error("signed DAPI timestamp missing")
		}
		w.Header().Set("Content-Type", "application/json")
		switch r.Method + " " + r.URL.Path {
		case "POST /dapi/v1/order":
			if r.Form.Get("type") != "MARKET" || r.Form.Get("quantity") != "10" ||
				r.Form.Get("positionSide") != "LONG" || r.Form.Get("side") != "BUY" {
				t.Errorf("normal COIN-M order wrong params: %v", r.Form)
			}
			_, _ = io.WriteString(w, `{"symbol":"BTCUSD_PERP","clientOrderId":"cm5api01","orderId":5001,"side":"BUY","positionSide":"LONG","status":"NEW","executedQty":"0"}`)
		case "GET /dapi/v1/order":
			_, _ = io.WriteString(w, `{"symbol":"BTCUSD_PERP","clientOrderId":"cm5api01","orderId":5001,"side":"BUY","positionSide":"LONG","status":"NEW","executedQty":"0"}`)
		case "GET /dapi/v1/algoOrder":
			if r.Form.Get("clientAlgoId") != "cm5protectapi" || r.Form.Get("symbol") != "BTCUSD_PERP" {
				t.Errorf("Algo query must use exact owned client ID: %v", r.Form)
			}
			_, _ = io.WriteString(w, `{"algoId":6001,"clientAlgoId":"cm5protectapi","symbol":"BTCUSD_PERP","side":"SELL","positionSide":"LONG","orderType":"STOP_MARKET","algoStatus":"TRIGGERED","actualOrderId":"7001"}`)
		case "GET /dapi/v1/userTrades":
			if r.Form.Get("orderId") != "5001" || r.Form.Get("symbol") != "BTCUSD_PERP" {
				t.Errorf("userTrades not scoped to owned order: %v", r.Form)
			}
			_, _ = io.WriteString(w, `[{"symbol":"BTCUSD_PERP","id":601,"orderId":5001,"side":"BUY","positionSide":"LONG","price":"50000","qty":"10","realizedPnl":"0","marginAsset":"BTC","commission":"0.00001","commissionAsset":"BTC","time":1700000000000}]`)
		case "POST /dapi/v1/algoOrder":
			if r.Form.Get("algoType") != "CONDITIONAL" || r.Form.Get("type") != "STOP_MARKET" ||
				r.Form.Get("closePosition") != "true" || r.Form.Get("positionSide") != "LONG" ||
				r.Form.Get("side") != "SELL" || r.Form.Get("workingType") != "MARK_PRICE" {
				t.Errorf("protective COIN-M Algo wrong params: %v", r.Form)
			}
			_, _ = io.WriteString(w, `{"algoId":6001,"clientAlgoId":"cm5protectapi","symbol":"BTCUSD_PERP","side":"SELL","positionSide":"LONG","algoStatus":"NEW"}`)
		case "DELETE /dapi/v1/algoOrder":
			_, _ = io.WriteString(w, `{"code":200,"msg":"success"}`)
		case "GET /dapi/v1/openAlgoOrders":
			_, _ = io.WriteString(w, `[]`)
		case "GET /dapi/v1/openOrders":
			_, _ = io.WriteString(w, `[]`)
		default:
			t.Errorf("unexpected signed DAPI endpoint: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()
	guard, err := newCoinMRESTGuard(srv.Client().Transport, srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	coinMRuntime = coinMRuntimeConfig{Environment: "testnet", RESTURL: srv.URL, WSURL: delivery.BaseWsTestnetUrl,
		APIKey: "CM5-MOCK-TESTNET", APISecret: secret}
	coinMSetupErr = nil
	deliveryClient = delivery.NewClient(coinMRuntime.APIKey, coinMRuntime.APISecret)
	deliveryClient.BaseURL = srv.URL
	deliveryClient.HTTPClient = &http.Client{Transport: guard}
	ctx := context.Background()
	o, err := CreateCoinMOrderContext(ctx, CoinMTradeOrder{
		Symbol: "BTCUSD_PERP", Side: "BUY", PositionSide: "LONG", Type: "MARKET", Contracts: "10", ClientOrderID: "cm5api01",
	})
	if err != nil || o.OrderID != 5001 {
		t.Fatalf("DAPI normal MARKET failed: %+v %v", o, err)
	}
	if _, err = QueryCoinMOrderContext(ctx, "BTCUSD_PERP", "cm5api01"); err != nil {
		t.Fatal(err)
	}
	fills, err := CoinMUserTradesContext(ctx, "BTCUSD_PERP", 5001)
	if err != nil || len(fills) != 1 || fills[0].TradeID != 601 || fills[0].CommissionAsset != "BTC" {
		t.Fatalf("DAPI owned userTrades mismatch: %+v %v", fills, err)
	}
	algo, err := CreateCoinMAlgoContext(ctx, CoinMAlgoOrderRequest{
		Symbol: "BTCUSD_PERP", Side: "SELL", PositionSide: "LONG", Type: "STOP_MARKET",
		TriggerPrice: "45000", ClientAlgoID: "cm5protectapi"})
	if err != nil || algo.AlgoID != 6001 {
		t.Fatalf("new conditional protection failed: %+v %v", algo, err)
	}
	verifiedAlgo, err := QueryCoinMAlgoContext(ctx, "BTCUSD_PERP", "cm5protectapi")
	if err != nil || verifiedAlgo.ClientAlgoID != "cm5protectapi" || string(verifiedAlgo.ActualOrderID) != `"7001"` {
		t.Fatalf("signed Algo child identity query failed: %+v %v", verifiedAlgo, err)
	}
	if _, err = ListOpenCoinMAlgosContext(ctx, "BTCUSD_PERP"); err != nil {
		t.Fatal(err)
	}
	if err = CancelCoinMAlgoContext(ctx, "BTCUSD_PERP", "cm5protectapi"); err != nil {
		t.Fatal(err)
	}
	if _, err = GetCoinMOpenOrdersContext(ctx, "BTCUSD_PERP"); err != nil {
		t.Fatal(err)
	}
	if _, err = CreateCoinMOrderContext(ctx, CoinMTradeOrder{Symbol: "BTCUSD_PERP",
		Side: "SELL", PositionSide: "LONG", Type: "STOP_MARKET", Contracts: "10", ClientOrderID: "cm5wrong"}); err == nil {
		t.Fatal("legacy STOP_MARKET incorrectly sent to /dapi/v1/order")
	}
	if _, err = CreateCoinMAlgoContext(ctx, CoinMAlgoOrderRequest{Symbol: "BTCUSD_PERP",
		Side: "BUY", PositionSide: "LONG", Type: "STOP_MARKET", TriggerPrice: "45000", ClientAlgoID: "cm5wrongalgo"}); err == nil {
		t.Fatal("protective BUY would increase LONG instead of closing it")
	}
	deliveryClient.BaseURL = delivery.BaseApiMainUrl
	if _, err = CoinMUserTradesContext(ctx, "BTCUSD_PERP", 5001); err == nil || !strings.Contains(err.Error(), "endpoint mismatch") {
		t.Fatalf("testnet signed DAPI allowed mainnet host fallback: %v", err)
	}
	mu.Lock()
	defer mu.Unlock()
	if hits["POST /dapi/v1/order"] != 1 || hits["POST /dapi/v1/algoOrder"] != 1 ||
		hits["GET /dapi/v1/userTrades"] != 1 {
		t.Fatalf("wrong signed DAPI call counts: %+v", hits)
	}
}
