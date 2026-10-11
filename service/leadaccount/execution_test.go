package leadaccount

import (
	"context"
	"errors"
	"testing"

	"github.com/adshao/go-binance/v2/futures"
	"go_binance_futures/feature/api/binance"
	"go_binance_futures/service/futuresownership"
)

func TestStage43ReadOnlyLeadBrokerBoundAndLocked(t *testing.T) {
	if _, err := NewReadOnlyOrderBroker(nil); err == nil {
		t.Fatal("nil client accepted")
	}
	main, _ := binance.NewAccountClient(binance.MainAccountID, futures.NewClient("main", "secret"))
	if _, err := NewReadOnlyOrderBroker(main); err == nil {
		t.Fatal("main account accepted")
	}
	if _, err := NewReadOnlyExecutor(main); err == nil {
		t.Fatal("main account executor accepted")
	}
	lead, _ := binance.NewAccountClient(binance.LeadAccountID, futures.NewClient("lead", "secret"))
	broker, err := NewReadOnlyOrderBroker(lead)
	if err != nil {
		t.Fatal(err)
	}
	_, err = broker.Submit(context.Background(), futuresownership.OrderRequest{Intent: futuresownership.IntentOpen}, "x")
	if !errors.Is(err, ErrLiveExecutionLocked) {
		t.Fatalf("unexpected submit outcome: %v", err)
	}
	if err = broker.Cancel(context.Background(), "BTCUSDT", 10, "STOP_MARKET"); !errors.Is(err, ErrLiveExecutionLocked) {
		t.Fatalf("unexpected cancel outcome: %v", err)
	}
	_, err = NewReadOnlyExecutor(lead)
	if err != nil {
		t.Fatal(err)
	}
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err = broker.Submit(cancelled, futuresownership.OrderRequest{}, ""); !errors.Is(err, context.Canceled) {
		t.Fatalf("context: %v", err)
	}
	if _, err = broker.Lookup(cancelled, "BTCUSDT", "x", "MARKET"); !errors.Is(err, context.Canceled) {
		t.Fatalf("lookup context: %v", err)
	}
}
