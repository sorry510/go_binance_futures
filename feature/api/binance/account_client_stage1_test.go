package binance

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/adshao/go-binance/v2/futures"
	"go_binance_futures/service/binanceapiusage"
)

func stage1TestAccount(t *testing.T, id AccountID, server *httptest.Server) *AccountClient {
	t.Helper()
	client := futures.NewClient(string(id)+"-key", string(id)+"-secret")
	client.SetApiEndpoint(server.URL)
	client.HTTPClient = server.Client()
	a, err := NewAccountClient(id, client)
	if err != nil {
		t.Fatal(err)
	}
	return a
}
func TestStage1AccountIsolationOfSignedRequestsAndCaches(t *testing.T) {
	var mainCalls, leadCalls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := r.Header.Get("X-MBX-APIKEY")
		if r.URL.Path != "/fapi/v2/positionRisk" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		switch key {
		case "main-key":
			mainCalls.Add(1)
			_, _ = w.Write([]byte(`[{"symbol":"BTCUSDT","positionAmt":"1"}]`))
		case "lead-key":
			leadCalls.Add(1)
			_, _ = w.Write([]byte(`[{"symbol":"BTCUSDT","positionAmt":"2"}]`))
		default:
			t.Errorf("wrong key %s", key)
			w.WriteHeader(http.StatusForbidden)
		}
	}))
	defer server.Close()
	main := stage1TestAccount(t, MainAccountID, server)
	lead := stage1TestAccount(t, LeadAccountID, server)
	for i := 0; i < 2; i++ {
		m, e := main.GetPositionContext(context.Background(), PositionParams{})
		if e != nil || len(m) != 1 || m[0].PositionAmt != "1" {
			t.Fatalf("main %v %v", m, e)
		}
		l, e := lead.GetPositionContext(context.Background(), PositionParams{})
		if e != nil || len(l) != 1 || l[0].PositionAmt != "2" {
			t.Fatalf("lead %v %v", l, e)
		}
	}
	if mainCalls.Load() != 1 || leadCalls.Load() != 1 {
		t.Fatalf("main=%d lead=%d", mainCalls.Load(), leadCalls.Load())
	}
	lead.InvalidateAccountReadCache()
	_, _ = main.GetPositionContext(context.Background(), PositionParams{})
	_, _ = lead.GetPositionContext(context.Background(), PositionParams{})
	if mainCalls.Load() != 1 || leadCalls.Load() != 2 {
		t.Fatalf("account cache invalidation leaked: main=%d lead=%d", mainCalls.Load(), leadCalls.Load())
	}
}
func TestStage1TradeConfigCacheNeverSharesAccounts(t *testing.T) {
	var mainCalls, leadCalls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Header.Get("X-MBX-APIKEY") {
		case "main-key":
			mainCalls.Add(1)
		case "lead-key":
			leadCalls.Add(1)
		default:
			t.Fatalf("unexpected API key")
		}
		if r.URL.Path == "/fapi/v1/leverage" {
			_, _ = w.Write([]byte(`{"leverage":4,"maxNotionalValue":"100000","symbol":"BTCUSDT"}`))
			return
		}
		if r.URL.Path == "/fapi/v1/marginType" {
			_, _ = w.Write([]byte(`{"code":200,"msg":"success"}`))
			return
		}
		t.Errorf("wrong path %s", r.URL.Path)
	}))
	defer server.Close()
	main := stage1TestAccount(t, MainAccountID, server)
	lead := stage1TestAccount(t, LeadAccountID, server)
	for _, a := range []*AccountClient{main, lead, main, lead} {
		if err := a.EnsureTradeConfigContext(context.Background(), "BTCUSDT", futures.MarginTypeCrossed, 4); err != nil {
			t.Fatal(err)
		}
	}
	if mainCalls.Load() != 2 || leadCalls.Load() != 2 {
		t.Fatalf("margin/leverage calls: main=%d lead=%d", mainCalls.Load(), leadCalls.Load())
	}
	main.InvalidateTradeConfig("BTCUSDT")
	if err := main.EnsureTradeConfigContext(context.Background(), "BTCUSDT", futures.MarginTypeCrossed, 4); err != nil {
		t.Fatal(err)
	}
	if mainCalls.Load() != 4 || leadCalls.Load() != 2 {
		t.Fatal("trade configuration caches are not isolated")
	}
}
func TestStage1LeadOrderLimiterPerClientPreventsHTTPBeforeNineteenth(t *testing.T) {
	var seen atomic.Int32
	base := stage1Transport(func(r *http.Request) (*http.Response, error) {
		seen.Add(1)
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: http.NoBody, Request: r}, nil
	})
	limiter := &leadOrderLimiter{base: base}
	for i := 0; i < leadOrderLimitPerTenSeconds; i++ {
		r, _ := http.NewRequest(http.MethodPost, "https://fapi.binance.com/fapi/v1/order", nil)
		if _, err := limiter.RoundTrip(r); err != nil {
			t.Fatalf("early limiter error at %d: %v", i, err)
		}
	}
	r, _ := http.NewRequest(http.MethodPost, "https://fapi.binance.com/fapi/v1/order", nil)
	if _, err := limiter.RoundTrip(r); err != binanceapiusage.ErrBudgetDeferred {
		t.Fatalf("budget error=%v", err)
	}
	if seen.Load() != leadOrderLimitPerTenSeconds {
		t.Fatalf("HTTP reached exchange %d times", seen.Load())
	}
	// Another credential has its own order bucket.
	other := &leadOrderLimiter{base: base}
	if _, err := other.RoundTrip(r); err != nil {
		t.Fatal(err)
	}
}

type stage1Transport func(*http.Request) (*http.Response, error)

func (f stage1Transport) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }
func TestStage1LeadSAPIReadOnlySigningAndResponse(t *testing.T) {
	calls := map[string]int{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" || r.Header.Get("X-MBX-APIKEY") != "lead-key" {
			t.Error("incorrect signed request")
		}
		values := r.URL.Query()
		gotSig := values.Get("signature")
		values.Del("signature")
		h := hmac.New(sha256.New, []byte("lead-secret"))
		_, _ = h.Write([]byte(values.Encode()))
		if gotSig != hex.EncodeToString(h.Sum(nil)) {
			t.Errorf("signature mismatch")
		}
		if values.Get("recvWindow") != "10000" {
			t.Errorf("recvWindow=%s", values.Get("recvWindow"))
		}
		if _, err := time.ParseDuration(fmt.Sprintf("%sms", values.Get("timestamp"))); err != nil {
			t.Error(err)
		}
		calls[r.URL.Path]++
		switch r.URL.Path {
		case "/sapi/v1/copyTrading/futures/userStatus":
			_, _ = w.Write([]byte(`{"code":"000000","message":"success","success":true,"data":{"isLeadTrader":true,"time":1}}`))
		case "/sapi/v1/copyTrading/futures/leadSymbol":
			_, _ = w.Write([]byte(`{"code":"000000","message":"success","data":[{"symbol":"BTCUSDT","baseAsset":"BTC","quoteAsset":"USDT"}]}`))
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()
	client := futures.NewClient("lead-key", "lead-secret")
	hc := server.Client()
	orig := hc.Transport
	if orig == nil {
		orig = http.DefaultTransport
	}
	hc.Transport = stage1Transport(func(r *http.Request) (*http.Response, error) {
		clone := r.Clone(r.Context())
		u, _ := url.Parse(server.URL)
		clone.URL.Scheme = u.Scheme
		clone.URL.Host = u.Host
		clone.Host = u.Host
		return orig.RoundTrip(clone)
	})
	client.HTTPClient = hc
	a, err := NewAccountClient(LeadAccountID, client)
	if err != nil {
		t.Fatal(err)
	}
	status, err := a.LeadTraderStatus(context.Background())
	if err != nil || !status.Data.IsLeadTrader {
		t.Fatalf("status=%+v err=%v", status, err)
	}
	symbols, err := a.LeadTradingSymbols(context.Background())
	if err != nil || len(symbols) != 1 || symbols[0].Symbol != "BTCUSDT" {
		t.Fatalf("symbols=%+v err=%v", symbols, err)
	}
	for _, p := range []string{"/sapi/v1/copyTrading/futures/userStatus", "/sapi/v1/copyTrading/futures/leadSymbol"} {
		if calls[p] != 1 {
			t.Errorf("calls[%s]=%d", p, calls[p])
		}
	}
	main := stage1TestAccount(t, MainAccountID, server)
	_, err = main.LeadTraderStatus(context.Background())
	if err == nil || !strings.Contains(err.Error(), "lead") {
		t.Fatalf("main must be rejected before network: %v", err)
	}
}

func TestStage1AccountTimestampResyncDoesNotChangeOtherAccount(t *testing.T) {
	var leadCalls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/fapi/v1/time" {
			_, _ = fmt.Fprintf(w, `{"serverTime":%d}`, time.Now().Add(-2*time.Second).UnixMilli())
			return
		}
		if r.URL.Path != "/fapi/v1/openOrders" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		if r.Header.Get("X-MBX-APIKEY") == "lead-key" && leadCalls.Add(1) == 1 {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"code":-1021,"msg":"Timestamp outside recvWindow"}`))
			return
		}
		_, _ = w.Write([]byte(`[]`))
	}))
	defer server.Close()
	main := stage1TestAccount(t, MainAccountID, server)
	lead := stage1TestAccount(t, LeadAccountID, server)
	if _, err := main.GetOpenOrderContext(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, err := lead.GetOpenOrderContext(context.Background()); err != nil {
		t.Fatal(err)
	}
	if lead.client.TimeOffset < 1000 || lead.client.TimeOffset > 3000 {
		t.Fatalf("lead time offset=%d", lead.client.TimeOffset)
	}
	if main.client.TimeOffset != 0 {
		t.Fatalf("main offset changed=%d", main.client.TimeOffset)
	}
}
