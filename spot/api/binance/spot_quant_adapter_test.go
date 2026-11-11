package binance

import (
	"context"
	"errors"
	"testing"

	"go_binance_futures/service/spotexecution"

	sdk "github.com/adshao/go-binance/v2"
)

func TestStage064SpotAdapterCannotTradeByDefault(t *testing.T) {
	a := NewSpotQuantAdapter()
	if a == nil || a.c == nil || a.allowOrderMutation {
		t.Fatal("default adapter should use existing Spot client but disable mutations")
	}
	_, err := a.Place(context.Background(), spotexecution.Request{
		Symbol: "BTCUSDT", Side: "BUY", Type: "MARKET", ClientOrderID: "local_only", Quantity: "0.01",
	})
	if !errors.Is(err, spotexecution.ErrDisabled) {
		t.Fatalf("default adapter made Spot POST attempt: %v", err)
	}
	_, err = a.Cancel(context.Background(), "BTCUSDT", "local_only")
	if !errors.Is(err, spotexecution.ErrDisabled) {
		t.Fatalf("default adapter can cancel remotely: %v", err)
	}
	if _, err = NewSpotQuantAdapterWithClient(nil, true); err == nil {
		t.Fatal("nil SDK client")
	}
	injected, err := NewSpotQuantAdapterWithClient(sdk.NewClient("fake_test_key", "fake_secret"), false)
	if err != nil || injected.allowOrderMutation {
		t.Fatalf("explicit read-only adapter error: %v", err)
	}
	if _, err := NewSpotQuantAdapterWithClient(sdk.NewClient("fake_test_key", "fake_secret"), true); err == nil {
		t.Fatal("06-4 mistakenly authorized mainnet Spot order mutation")
	}
	testnet := sdk.NewClient("fake_test_key", "fake_secret")
	testnet.SetApiEndpoint("https://testnet.binance.vision")
	approved, err := NewSpotQuantAdapterWithClient(testnet, true)
	if err != nil || !approved.allowOrderMutation {
		t.Fatalf("testnet-only explicit opt-in rejected: %v", err)
	}
	// SDK BaseURL is exported/mutable; POST/DELETE must recheck at action
	// time even after a valid Testnet adapter was constructed.
	testnet.SetApiEndpoint("https://api.binance.com")
	if _, err := approved.Place(context.Background(), spotexecution.Request{Symbol: "BTCUSDT", Side: "BUY", Type: "MARKET", ClientOrderID: "unsafe", Quantity: "1"}); !errors.Is(err, spotexecution.ErrDisabled) {
		t.Fatalf("mutated adapter could send Mainnet Spot order: %v", err)
	}
	if _, err := approved.Cancel(context.Background(), "BTCUSDT", "unsafe"); !errors.Is(err, spotexecution.ErrDisabled) {
		t.Fatalf("mutated adapter could cancel Mainnet Spot order: %v", err)
	}
}

func TestReview064WSExplicitTestnetEndpointAndDecode(t *testing.T) {
	client := sdk.NewClient("fake_key", "fake_secret")
	client.SetApiEndpoint("https://testnet.binance.vision")
	a, err := NewSpotQuantAdapterWithClient(client, true)
	if err != nil {
		t.Fatal(err)
	}
	// sdk.UseTestnet is intentionally false; our WS must never resolve
	// its endpoint from this process-global SDK flag.
	if sdk.UseTestnet {
		t.Skip("SDK global testnet already set by unrelated tests")
	}
	endpoint, err := spotQuantStreamEndpoint(a)
	if err != nil || endpoint != "wss://ws-api.testnet.binance.vision/ws-api/v3" {
		t.Fatalf("unexpected signed Spot WS endpoint %q err=%v", endpoint, err)
	}
	a.c.SetApiEndpoint("https://api.binance.com")
	if _, err = spotQuantStreamEndpoint(a); !errors.Is(err, spotexecution.ErrDisabled) {
		t.Fatalf("mutated Testnet adapter could target Mainnet WS: %v", err)
	}
	for _, tc := range []struct {
		payload   string
		ack       bool
		wantEvent bool
		wantErr   bool
	}{
		{`{"id":"1","status":200,"result":{}}`, true, false, false},
		{`{"id":"1","status":401,"error":{"msg":"not authorized"}}`, false, false, true},
		{`{"event":{"e":"executionReport","s":"BTCUSDT","c":"cl","S":"BUY","i":71,"X":"FILLED","x":"TRADE"}}`, false, true, false},
		{`{"data":{"e":"executionReport","s":"BTCUSDT","c":"cl","S":"BUY","i":72,"X":"NEW"}}`, false, true, false},
		{`{"e":"outboundAccountPosition"}`, false, false, false},
		{`{"e":"executionReport","s":"BTCUSDT","i":0}`, false, false, true},
		{`invalid json`, false, false, true},
	} {
		ack, event, err := decodeSpotQuantMessage([]byte(tc.payload))
		if ack != tc.ack || (event != nil) != tc.wantEvent || (err != nil) != tc.wantErr {
			t.Errorf("payload=%s ack=%v event=%+v err=%v", tc.payload, ack, event, err)
		}
	}
}

type fakeReviewStreamProcessor struct {
	handled      bool
	handleErr    error
	reconcileErr error
	reconciles   int
}

func (f *fakeReviewStreamProcessor) HandleExecutionReport(_ context.Context, _ spotexecution.ExecutionReport) (bool, error) {
	return f.handled, f.handleErr
}
func (f *fakeReviewStreamProcessor) ReconcileByClientID(_ context.Context, _ string) error {
	f.reconciles++
	return f.reconcileErr
}
func TestReview064WSFilledBeforeMyTradesKeepsStream(t *testing.T) {
	ev := sdk.WsOrderUpdate{Symbol: "BTCUSDT", ClientOrderId: "race_064", Side: "BUY",
		Status: "FILLED", ExecutionType: "NEW", Id: 777}
	transient := &fakeReviewStreamProcessor{handled: true, handleErr: spotexecution.ErrNeedsReview, reconcileErr: spotexecution.ErrNeedsReview}
	if err := applySpotQuantOrderEvent(context.Background(), transient, ev); err != nil {
		t.Fatalf("FILLED status ahead of myTrades disconnected WS: %v", err)
	}
	if transient.reconciles != 1 {
		t.Fatal("missing immediate read-only reconciliation attempt")
	}
	// Non-review failures are not swallowed.
	fatal := &fakeReviewStreamProcessor{handled: true, handleErr: errors.New("database unavailable")}
	if err := applySpotQuantOrderEvent(context.Background(), fatal, ev); err == nil {
		t.Fatal("unexpected fatal event error hidden")
	}
	if fatal.reconciles != 0 {
		t.Fatal("unexpected retry for fatal event error")
	}
}
