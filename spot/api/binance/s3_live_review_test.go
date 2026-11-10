package binance

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"

	sdk "github.com/adshao/go-binance/v2"
	"go_binance_futures/service/spotstrategy"
)

type s3InterceptTransport func(*http.Request) (*http.Response, error)

func (f s3InterceptTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func s3LocalSignedSDK(t *testing.T, intercept s3InterceptTransport) *sdk.Client {
	t.Helper()
	c := sdk.NewClient("offline-s3-key", "offline-s3-secret")
	c.BaseURL = sdk.BaseAPIMainURL
	// There is deliberately NO network fallback. Requests to api.binance.com
	// are captured in memory and the test returns synthetic response bodies.
	c.HTTPClient = &http.Client{Transport: intercept}
	return c
}

func TestS3F2ProductionCreateSDKWirePayload(t *testing.T) {
	for _, tc := range []struct {
		name, side, kind, price, wantPrice, wantTIF string
	}{
		{"limit_buy", "BUY", "LIMIT", "12", "12", "GTC"},
		{"market_buy", "BUY", "MARKET", "", "", ""},
		{"limit_sell", "SELL", "LIMIT", "12", "12", "GTC"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			called := 0
			sdkClient := s3LocalSignedSDK(t, func(r *http.Request) (*http.Response, error) {
				called++
				if r.Method != http.MethodPost || r.URL.Path != "/api/v3/order" {
					t.Errorf("unexpected signed write: %s %s", r.Method, r.URL.Path)
				}
				if r.URL.Host != "api.binance.com" {
					t.Errorf("unexpected HTTP host %s", r.URL.Host)
				}
				_ = r.ParseForm()
				v := r.Form
				for key, want := range map[string]string{
					"symbol": "BTCUSDT", "side": tc.side, "type": tc.kind,
					"quantity": "3", "price": tc.wantPrice,
					"timeInForce": tc.wantTIF, "newClientOrderId": "s3-review-cid",
					"newOrderRespType": "RESULT",
				} {
					if got := v.Get(key); got != want {
						t.Errorf("%s=%q want %q; POST payload=%v", key, got, want, v)
					}
				}
				if v.Get("signature") == "" || v.Get("timestamp") == "" {
					t.Error("signed SDK request must include timestamp and signature")
				}
				body := fmt.Sprintf(`{"symbol":"BTCUSDT","orderId":93001,"clientOrderId":"s3-review-cid",
      "side":%q,"status":"NEW","origQty":"3","executedQty":"0","cummulativeQuoteQty":"0"}`, tc.side)
				return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": []string{"application/json"}},
					Body: io.NopCloser(strings.NewReader(body)), Request: r}, nil
			})
			req := spotstrategy.OrderRequest{AccountID: "main", Owner: "auto_strategy",
				Symbol: "BTCUSDT", Side: tc.side, Type: tc.kind, Quantity: "3",
				Price: tc.price, ClientOrderID: "s3-review-cid"}
			got, err := s3SendCreate(context.Background(), sdkClient.NewCreateOrderService(), req)
			if err != nil {
				t.Fatalf("real SDK POST parser failed: %v", err)
			}
			if called != 1 || got.OrderID != "93001" || got.ClientOrderID != req.ClientOrderID ||
				got.Side != tc.side || got.RequestedQty != "3" || got.ExecutedQty != "0" {
				t.Fatalf("wrong signed POST response: called=%d %+v", called, got)
			}
		})
	}
}
func TestS3F2ProductionCancelSDKWirePayload(t *testing.T) {
	called := 0
	c := s3LocalSignedSDK(t, func(r *http.Request) (*http.Response, error) {
		called++
		if r.Method != http.MethodDelete || r.URL.Path != "/api/v3/order" {
			t.Errorf("unexpected method/path: %s %s", r.Method, r.URL.Path)
		}
		// Go's Request.ParseForm does not automatically parse DELETE bodies.
		// The SDK puts DELETE symbol/origClientOrderId in form body while
		// timestamp/signature are in the URL query.
		body, readErr := io.ReadAll(r.Body)
		if readErr != nil {
			t.Fatalf("read signed DELETE body: %v", readErr)
		}
		v, parseErr := url.ParseQuery(string(body))
		if parseErr != nil {
			t.Fatalf("parse signed DELETE body: %v", parseErr)
		}
		for key, values := range r.URL.Query() {
			for _, value := range values {
				v.Add(key, value)
			}
		}
		if v.Get("symbol") != "BTCUSDT" || v.Get("origClientOrderId") != "s3-cancel-cid" ||
			v.Get("orderId") != "" || v.Get("signature") == "" || v.Get("timestamp") == "" {
			t.Errorf("wrong DELETE parameters: %v", v)
		}
		return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": []string{"application/json"}},
			Body: io.NopCloser(strings.NewReader(`{"symbol":"BTCUSDT","orderId":93001,
    "origClientOrderId":"s3-cancel-cid","clientOrderId":"s3-replaced-by-cancel","status":"CANCELED"}`)), Request: r}, nil
	})
	err := s3SendCancel(context.Background(), c.NewCancelOrderService(), "BTCUSDT", "s3-cancel-cid")
	if err != nil || called != 1 {
		t.Fatalf("real SDK DELETE flow failed: %v calls=%d", err, called)
	}
}
func TestS3F2ResponseIdentityMismatchFailClosed(t *testing.T) {
	c := s3LocalSignedSDK(t, func(r *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(
			`{"symbol":"BTCUSDT","orderId":93001,"clientOrderId":"another-order","side":"BUY","status":"NEW","origQty":"3","executedQty":"0"}`)), Request: r}, nil
	})
	_, err := s3SendCreate(context.Background(), c.NewCreateOrderService(), spotstrategy.OrderRequest{
		Symbol: "BTCUSDT", Side: "BUY", Type: "MARKET", Quantity: "3", ClientOrderID: "s3-own-cid"})
	if !errors.Is(err, spotstrategy.ErrReconciliation) {
		t.Fatalf("wrong client ID accepted: %v", err)
	}
}
func TestS3F1LiveGateRequiresMoreThanEnvironmentAndMode(t *testing.T) {
	// The review reproduced an env+config-row bypass. No unaudited grant can
	// be issued in this S3 binary, even if the legacy environment value leaks.
	t.Setenv("SPOT_STRATEGY_LIVE_TRADING_APPROVAL", "SPOT-S3-REAL-ORDERS-EXPLICITLY-APPROVED")
	if spotstrategy.LiveOrdersApproved() {
		t.Fatal("legacy env alone enabled real trades")
	}
	if err := spotstrategy.AuthorizeLiveMode(); !errors.Is(err, spotstrategy.ErrLiveWriteDisabled) {
		t.Fatalf("unaudited mode+env bypass: %v", err)
	}
}
func TestS3F5RealWriteDefaultDenialCountsZeroHTTPRequests(t *testing.T) {
	old := client
	count := 0
	c := s3LocalSignedSDK(t, func(*http.Request) (*http.Response, error) {
		count++
		return nil, errors.New("test MUST NOT reach actual transport")
	})
	client = c
	defer func() { client = old }()
	t.Setenv("SPOT_STRATEGY_LIVE_TRADING_APPROVAL", "SPOT-S3-REAL-ORDERS-EXPLICITLY-APPROVED")
	req := spotstrategy.OrderRequest{
		AccountID: "main", Owner: "auto_strategy", Symbol: "BTCUSDT",
		Side: "BUY", Type: "LIMIT", Quantity: "3", Price: "12", ClientOrderID: "s3-live-deny"}
	if _, err := (S3LiveClient{}).Place(context.Background(), req); !errors.Is(err, spotstrategy.ErrLiveWriteDisabled) {
		t.Fatalf("unaudited write passed: %v", err)
	}
	if err := (S3LiveClient{}).Cancel(context.Background(), "main", "BTCUSDT", "s3-live-deny"); !errors.Is(err, spotstrategy.ErrLiveWriteDisabled) {
		t.Fatalf("unaudited cancel passed: %v", err)
	}
	if count != 0 {
		t.Fatalf("default denied Spot POST/DELETE reached HTTP transport %d times", count)
	}
	// The endpoint check alone is no authorization, and non-main endpoints
	// should not be accepted even with a valid-looking approval string.
	client.BaseURL = "http://127.0.0.1:12345"
	if _, err := (S3LiveClient{}).Place(context.Background(), req); !errors.Is(err, spotstrategy.ErrLiveWriteDisabled) {
		t.Fatalf("non-main write allowed: %v", err)
	}
	if count != 0 {
		t.Fatalf("non-main endpoint reached HTTP transport %d times", count)
	}
}
func TestS3F2SignedSDKNotRawPublicURL(t *testing.T) {
	// Explicitly document why tests do not use httptest.Server: the signed
	// client's BaseURL remains mainnet but its transport never sends packets.
	u, e := url.Parse(sdk.BaseAPIMainURL)
	if e != nil || u.Scheme != "https" {
		t.Fatal("SDK mainnet URL must be HTTPS")
	}
}
