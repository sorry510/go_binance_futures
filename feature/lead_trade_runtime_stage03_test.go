package feature

import (
	"testing"

	"go_binance_futures/models"
	"go_binance_futures/service/leadaccount"
)

func TestLead03SharedTickIsInertUntilSeparateLiveApproval(t *testing.T) {
	if leadaccount.LeadDurableOpenGate() == nil {
		t.Fatal("Lead unexpectedly obtained a live trading capability")
	}
	// No credentials, WS, SAPI or local balance needed while locked.
	runLeadOnSharedTradeTick(&models.Config{FutureEnable: 1, FutureAllowLong: 1})
	StartTrade(&models.Config{FutureEnable: 0})
	r, _, _, _ := stage032Hooks(t, "MARKET")
	r.Mock = false
	r.leadMockPermit = nil
	if err := r.validate(); err == nil {
		t.Fatal("unapproved Lead runner passed validator")
	}
}
