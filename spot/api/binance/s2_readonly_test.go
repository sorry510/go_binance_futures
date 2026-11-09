package binance

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	sdk "github.com/adshao/go-binance/v2"
	"go_binance_futures/service/spotstrategy"
)

// T2-10 drives the real SDK payload against localhost, NEVER Binance.
// Using quantity=3 and price=12 catches the earlier quantity-as-price bug.
func TestS2T10SellLimitUsesDistinctPrice(t *testing.T) {
	var form url.Values
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v3/order" || r.Method != http.MethodPost {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
		_ = r.ParseForm()
		form = r.Form
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"symbol":"BTCUSDT","orderId":101,"clientOrderId":"local","status":"NEW"}`)
	}))
	defer server.Close()
	c := sdk.NewClient("test-only-key", "test-only-secret")
	c.BaseURL = server.URL
	resp, err := s2BuildLegacySellLimit(c.NewCreateOrderService(), "BTCUSDT", 3, 12).Do(context.Background())
	if err != nil || resp == nil {
		t.Fatalf("SDK failed to build local request: %+v %v", resp, err)
	}
	if form.Get("symbol") != "BTCUSDT" || form.Get("side") != "SELL" || form.Get("type") != "LIMIT" ||
		form.Get("quantity") != "3" || form.Get("price") != "12" {
		t.Fatalf("wrong Spot limit order payload: quantity=%q price=%q side=%q", form.Get("quantity"), form.Get("price"), form.Get("side"))
	}
}

func TestS2ReadOnlyAdapterCannotWrite(t *testing.T) {
	c := S2ReadOnlyClient{}
	if _, err := c.Place(context.Background(), spotstrategy.OrderRequest{}); !errors.Is(err, spotstrategy.ErrLiveWriteDisabled) {
		t.Fatal(err)
	}
	if err := c.Cancel(context.Background(), "main", "BTCUSDT", "cid"); !errors.Is(err, spotstrategy.ErrLiveWriteDisabled) {
		t.Fatal(err)
	}
}
func TestS2ReadOnlyAdapterSDKLocalREST(t *testing.T) {
	calls := map[string]int{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls[r.URL.Path]++
		if r.Method != http.MethodGet {
			t.Errorf("unexpected private write: %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/v3/account":
			_, _ = io.WriteString(w, `{"canTrade":true,"balances":[{"asset":"BTC","free":"0","locked":"0"},{"asset":"USDT","free":"1000","locked":"100"}]}`)
		case "/api/v3/openOrders":
			_, _ = io.WriteString(w, `[]`)
		case "/api/v3/order":
			if r.URL.Query().Get("origClientOrderId") != "orig" || r.URL.Query().Get("orderId") != "" {
				t.Errorf("lookup must identify original client ID, not blindly trust an exchange order ID")
			}
			_, _ = io.WriteString(w, `{"symbol":"BTCUSDT","orderId":901,"clientOrderId":"orig","side":"BUY","status":"PARTIALLY_FILLED","origQty":"10","executedQty":"2","cummulativeQuoteQty":"200"}`)
		case "/api/v3/myTrades":
			q := r.URL.Query()
			if q.Get("orderId") != "901" || q.Get("fromId") != "11" || q.Get("limit") != "500" {
				t.Errorf("incorrect trade pagination query: %s", r.URL.RawQuery)
			}
			_, _ = io.WriteString(w, `[{"symbol":"BTCUSDT","orderId":901,"id":11,"price":"100","qty":"2","quoteQty":"200","commission":"0.01","commissionAsset":"BNB","time":1000,"isBuyer":true}]`)
		case "/api/v3/exchangeInfo":
			_, _ = io.WriteString(w, `{"symbols":[{"symbol":"BTCUSDT","status":"TRADING","baseAsset":"BTC","quoteAsset":"USDT","isSpotTradingAllowed":true,"filters":[{"filterType":"LOT_SIZE","minQty":"0.01","maxQty":"100","stepSize":"0.01"},{"filterType":"PRICE_FILTER","minPrice":"1","maxPrice":"1000000","tickSize":"0.01"},{"filterType":"NOTIONAL","minNotional":"5"}]}]}`)
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()
	prev := client
	client = sdk.NewClient("local-only", "local-only")
	client.BaseURL = server.URL
	defer func() { client = prev }()
	read := S2ReadOnlyClient{}
	balances, err := read.Balances(context.Background(), "main")
	if err != nil || balances["USDT"].Free != "1000" || balances["USDT"].Locked != "100" {
		t.Fatalf("balances %+v %v", balances, err)
	}
	spec, err := read.Instrument(context.Background(), "BTCUSDT")
	if err != nil || spec.BaseAsset != "BTC" || spec.QuoteAsset != "USDT" || spec.LotStep != "0.01" || spec.MinNotional != "5" {
		t.Fatalf("exchangeInfo %+v %v", spec, err)
	}
	open, err := read.OpenOrders(context.Background(), "main", "BTCUSDT")
	if err != nil || len(open) != 0 {
		t.Fatalf("open %+v %v", open, err)
	}
	snapshot, err := read.QueryOrder(context.Background(), "main", "BTCUSDT", "orig", "901")
	if err != nil || snapshot.ExecutedQty != "2" || snapshot.OrderID != "901" {
		t.Fatalf("order %+v %v", snapshot, err)
	}
	if _, err := read.QueryOrder(context.Background(), "main", "BTCUSDT", "orig", "902"); !errors.Is(err, spotstrategy.ErrUnmanagedOrder) {
		t.Fatalf("foreign/stale exchange order id must not be silently adopted: %v", err)
	}
	fills, err := read.Trades(context.Background(), "main", "BTCUSDT", "901", 11, 500)
	if err != nil || len(fills) != 1 || fills[0].CommissionAsset != "BNB" || fills[0].Qty != "2" {
		t.Fatalf("fills %+v %v", fills, err)
	}
	if calls["/api/v3/order"] != 2 || calls["/api/v3/myTrades"] != 1 {
		t.Fatal("not all read-only endpoints exercised")
	}
	if _, err := read.QueryOrder(context.Background(), "lead", "BTCUSDT", "orig", ""); err == nil {
		t.Fatal("wrong account accepted")
	}
	if _, err := read.Trades(context.Background(), "main", "BTCUSDT", "901", 0, 2000); err == nil {
		t.Fatal("oversized page accepted")
	}
	if s := strings.TrimSpace(spec.Status); s != "TRADING" {
		t.Fatalf("exchange status %q", s)
	}
}
