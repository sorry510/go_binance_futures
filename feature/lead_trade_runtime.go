package feature

import (
	"context"
	"time"

	"github.com/beego/beego/v2/core/logs"
	binance "go_binance_futures/feature/api/binance"
	"go_binance_futures/models"
	"go_binance_futures/service/leadaccount"
)

// Called by the existing shared tick; no second scheduler is introduced.
// New Lead entries require the default-OFF persisted switch, a dedicated
// Lead client/WS and the normal risk checks. Main's order hooks are untouched.
func runLeadOnSharedTradeTick(mainConfig *models.Config) {
	if mainConfig == nil || !leadaccount.LeadReadOnlyStatus().Running {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()
	account, symbols, adapter, err := leadaccount.CurrentLeadTradeSession(ctx)
	if err != nil {
		logs.Warning("Lead strategy not ready: ", err)
		return
	}
	configStore, err := leadaccount.LeadConfigStoreFromDB()
	if err != nil {
		logs.Warning("Lead strategy config unavailable")
		return
	}
	settings, err := configStore.Load()
	if err != nil {
		logs.Warning("Lead strategy config invalid")
		return
	}
	io, err := newLeadTradeIO(account)
	if err != nil {
		logs.Warning("Lead ownership scope unavailable")
		return
	}
	// Public trading parameters and strategy DSL are shared with Main.
	// The Lead account alone controls new entries; Main's main switch also
	// prevents new Lead entries, but disabling either never blocks exits.
	cfg := *mainConfig
	r := &accountTradeRunner{
		AccountID:   binance.LeadAccountID,
		Config:      &cfg,
		LeadSymbols: symbols,
		AllowNewOpens: settings.AllowNewOpens && cfg.FutureEnable == 1 &&
			cfg.FutureMaxCount > 0 && cfg.LossMaxCount > 0 &&
			!leadaccount.LeadEmergencyPaused(),
	}
	if err := bindLeadOrderHooks(r, io, adapter); err != nil {
		logs.Warning("Lead strategy hooks unavailable: ", err)
		return
	}
	// The operator's persisted switch is sufficient; no separate
	// VerifyReadOnly action is required. Current signed Lead REST/WS and
	// existing RiskController still check live funds/position/rules.
	if err := adapter.ConfigureLiveSession(settings, cfg.FutureMaxCount, cfg.LossMaxCount); err != nil {
		logs.Warning("Lead trade session not ready: ", err)
		return
	}
	if err := r.validate(); err != nil {
		logs.Warning("Lead live strategy remains disabled: ", err)
		return
	}
	runAccountTradeCycle(r)
}
