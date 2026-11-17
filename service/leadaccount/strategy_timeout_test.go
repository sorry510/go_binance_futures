package leadaccount

import (
	"context"
	"errors"
	"testing"
	"time"

	"go_binance_futures/models"
	"go_binance_futures/service/futuresownership"
)

func TestLead03TimeoutOnlyCancelsIncompleteAutoOpen(t *testing.T) {
	now := time.Now().UnixMilli()
	base := models.FuturesManagedOrder{
		AccountID: "lead", Owner: futuresownership.OwnerAutoStrategy,
		Symbol: "BTCUSDT", Intent: futuresownership.IntentOpen,
		CreatedAt: now - 90_000, RequestedQty: 2, FilledQty: 0.5,
		Status: futuresownership.OrderPartiallyFilled,
	}
	if !leadExpiredOpeningOrder(base, now, 60) {
		t.Fatal("partial expired managed opening was not eligible")
	}
	checks := []struct {
		name   string
		mutate func(*models.FuturesManagedOrder)
	}{
		{"main", func(o *models.FuturesManagedOrder) { o.AccountID = "main" }},
		{"manual", func(o *models.FuturesManagedOrder) { o.Owner = "manual" }},
		{"close", func(o *models.FuturesManagedOrder) { o.Intent = futuresownership.IntentClose }},
		{"new", func(o *models.FuturesManagedOrder) { o.CreatedAt = now - 30_000 }},
		{"filled", func(o *models.FuturesManagedOrder) { o.FilledQty = 2 }},
		{"canceled", func(o *models.FuturesManagedOrder) { o.Status = futuresownership.OrderCanceled }},
		{"negative_remaining", func(o *models.FuturesManagedOrder) { o.FilledQty = 3 }},
	}
	for _, tc := range checks {
		t.Run(tc.name, func(t *testing.T) {
			row := base
			tc.mutate(&row)
			if leadExpiredOpeningOrder(row, now, 60) {
				t.Fatalf("unsafe order canceled: %+v", row)
			}
		})
	}
	if leadExpiredOpeningOrder(base, now, 0) {
		t.Fatal("timeout disabled but cancellation selected")
	}
}

func TestLead03TimeoutCannotMutateWithoutLiveAuthorization(t *testing.T) {
	a, _ := stage43Adapter(t, true)
	a.risk.mu.Lock()
	a.risk.state.Stage7Authorized = false
	a.risk.mu.Unlock()
	if err := a.CancelExpiredStrategyOpens(context.Background(), 60); !errors.Is(err, ErrLeadOpenBlocked) {
		t.Fatalf("unauthorized account cancellation accepted: %v", err)
	}
}
