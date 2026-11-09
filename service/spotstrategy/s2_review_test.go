package spotstrategy

import (
	"context"
	"errors"
	"fmt"
	"github.com/beego/beego/v2/client/orm"
	"strings"
	"testing"
	"time"
)

// S2-F1 reproduces the real timeline in CODE_REVIEW_02-S2: terminal -> a
// conflicting historical fill -> sticky freeze -> >3s -> RecoverPending with
// 429 -> no downgrade and no second BUY. No clock timestamp is manipulated.
func TestS2F1FrozenTerminalRecoveryCannotUnfreeze(t *testing.T) {
	e, f := prepareS2(t)
	id := mustPrepareSubmitS2(t, e, "f1-initial")
	f.Lock()
	f.trades = []Trade{tradeS2(901, "10", "100", "1000", "0.01", "BNB")}
	f.order.Status = "FILLED"
	f.order.ExecutedQty = "10"
	f.order.ExecutedQuoteQty = "1000"
	f.Unlock()
	finished, err := e.Reconcile(context.Background(), id.ID)
	if err != nil || finished.Status != StateFilled {
		t.Fatalf("terminal %+v %v", finished, err)
	}
	f.Lock()
	f.trades[0].Commission = "0.02"
	f.Unlock()
	if _, err = e.Reconcile(context.Background(), id.ID); !errors.Is(err, ErrReconciliation) {
		t.Fatalf("tampered trade not caught: %v", err)
	}
	frozen, err := e.Load(context.Background(), id.ID)
	if err != nil || frozen.Status != StateReconcile || frozen.ReservedQty != "0" {
		t.Fatalf("expected historical quarantine with released old slot: %+v %v", frozen, err)
	}
	originalReason := frozen.LastError
	// The review's bug only manifested after RecoverPending's three-second
	// selection threshold. Wait for the real window.
	time.Sleep(3100 * time.Millisecond)
	f.Lock()
	f.queryErr = errors.New("429 request weight exceeded")
	f.Unlock()
	scanned, err := e.RecoverPending(context.Background(), 10)
	if scanned != 1 || !errors.Is(err, ErrReconciliation) {
		t.Fatalf("recovery should probe quarantined row: %d %v", scanned, err)
	}
	after, err := e.Load(context.Background(), id.ID)
	if err != nil || after.Status != StateReconcile || after.LastError != originalReason {
		t.Fatalf("quarantine downgraded or evidence overwritten: before=%+v after=%+v err=%v", frozen, after, err)
	}
	if _, err = e.Prepare(context.Background(), requestS2("f1-followup")); !errors.Is(err, ErrReconciliation) {
		t.Fatalf("new BUY permitted after recovery: %v", err)
	}
	if countTableS2(t, "spot_strategy_orders") != 1 {
		t.Fatal("second order inserted after frozen history")
	}
	for _, queryErr := range []error{errors.New("418 IP banned"), context.DeadlineExceeded, errors.New("-2013 Order not found")} {
		f.Lock()
		f.queryErr = queryErr
		f.Unlock()
		_, _ = e.Reconcile(context.Background(), id.ID)
		e.markUnknown(context.Background(), id.ID, "spot_order_identity_mismatch")
		current, err := e.Load(context.Background(), id.ID)
		if err != nil || current.Status != StateReconcile || current.LastError != originalReason {
			t.Fatalf("frozen order changed on %v: %+v %v", queryErr, current, err)
		}
	}
}

func TestS2F2FakeReadOnlyEngineCannotForgeRESTVerified(t *testing.T) {
	e, f := prepareS2(t)
	order := mustPrepareSubmitS2(t, e, "f2-fake")
	f.Lock()
	f.trades = []Trade{tradeS2(905, "10", "100", "1000", "0.003", "BNB")}
	f.order.Status = "FILLED"
	f.order.ExecutedQty = "10"
	f.order.ExecutedQuoteQty = "1000"
	f.Unlock()
	db, err := orm.GetDB("default")
	if err != nil {
		t.Fatal(err)
	}
	// This is precisely the review probe: NewReadOnlyEngine with a Fake.
	readOnly := NewReadOnlyEngine(db, f)
	got, err := readOnly.Reconcile(context.Background(), order.ID)
	if err != nil || got.Status != StateFilled {
		t.Fatalf("fake read-only reconcile %+v %v", got, err)
	}
	fills, err := readOnly.ListFills(context.Background(), order.ID)
	if err != nil || len(fills) != 1 {
		t.Fatalf("fake read-only fills %+v %v", fills, err)
	}
	if fills[0].RESTVerified {
		t.Fatal("Fake Client marked simulated Fill as genuine REST evidence")
	}
	var n int
	if err = db.QueryRow(`SELECT COUNT(*) FROM spot_strategy_fills WHERE rest_verified<>0`).Scan(&n); err != nil || n != 0 {
		t.Fatalf("unverified source polluted REST flag: n=%d err=%v", n, err)
	}
}

func TestS2F4UnknownOrderAndForeignIdentityCannotBeManaged(t *testing.T) {
	e, f := prepareS2(t)
	for _, method := range []struct {
		name string
		call func() error
	}{
		{"Load", func() error { _, err := e.Load(context.Background(), 999999); return err }},
		{"Reconcile", func() error { _, err := e.Reconcile(context.Background(), 999999); return err }},
		{"CancelMock", func() error { _, err := e.CancelMock(context.Background(), 999999); return err }},
		{"ListFills", func() error { _, err := e.ListFills(context.Background(), 999999); return err }},
	} {
		if err := method.call(); !errors.Is(err, ErrUnmanagedOrder) {
			t.Fatalf("external %s returned %v, need ErrUnmanagedOrder", method.name, err)
		}
	}
	local := mustPrepareSubmitS2(t, e, "f4-owned")
	f.Lock()
	f.order.ClientOrderID = "foreign-client-order"
	f.order.Status = "CANCELED"
	f.Unlock()
	snap, err := e.Reconcile(context.Background(), local.ID)
	if !errors.Is(err, ErrUnmanagedOrder) || snap.Status != StateReconcile {
		t.Fatalf("foreign canceled client ID adopted: %+v err=%v", snap, err)
	}
	if f.canceled != 0 || countTableS2(t, "spot_strategy_fills") != 0 {
		t.Fatal("external order canceled or misattributed")
	}
	if _, err = e.CancelMock(context.Background(), local.ID); err == nil || f.canceled != 0 {
		t.Fatalf("quarantined foreign order canceled: %v", err)
	}
	if _, err = e.Prepare(context.Background(), requestS2("f4-followup")); !errors.Is(err, ErrReconciliation) {
		t.Fatalf("foreign order quarantine did not stop new risk: %v", err)
	}
}

func TestS2F5PrepareRedactsAllSignedRESTFailures(t *testing.T) {
	secret := "signature=do-not-leak-this"
	for _, phase := range []string{"instrument", "balances", "openOrders"} {
		t.Run(phase, func(t *testing.T) {
			e, f := prepareS2(t)
			sdkErr := fmt.Errorf("Get https://api.binance.com/api/v3/order?%s: 429", secret)
			switch phase {
			case "instrument":
				f.instrumentErr = sdkErr
			case "balances":
				f.balancesErr = sdkErr
			case "openOrders":
				f.openErr = sdkErr
			}
			_, err := e.Prepare(context.Background(), requestS2("redaction-"+phase))
			if err == nil || strings.Contains(err.Error(), secret) || strings.Contains(err.Error(), "api.binance.com") {
				t.Fatalf("raw signed REST error leaked at %s: %v", phase, err)
			}
			if !strings.Contains(err.Error(), "spot_rest_rate_limited_429") {
				t.Fatalf("lost safe error classification: %v", err)
			}
			if countTableS2(t, "spot_strategy_orders") != 0 {
				t.Fatal("read failure reserved order")
			}
		})
	}
}
