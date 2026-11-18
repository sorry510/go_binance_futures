package coinmexecution

import (
	"errors"
	"fmt"

	"github.com/beego/beego/v2/client/orm"
	"go_binance_futures/feature/api/binance"
)

// RuntimeSummary is a LOCAL/READ-ONLY observation. It never opens an
// exchange connection, submits a Binance order or interprets absence of
// local records as proof of an empty remote account.
type RuntimeSummary struct {
	Environment                 string `json:"environment"`
	GlobalEntryEnabled          bool   `json:"global_entry_enabled"`
	EnabledPerpetualSymbols     int64  `json:"enabled_perpetual_symbols"`
	ManagedCyclesNeedingWork    int64  `json:"managed_cycles_needing_work"`
	OrdersNeedingReconciliation int64  `json:"orders_needing_reconciliation"`
	PrivateWSVerified           bool   `json:"private_ws_verified"`
	WalletVerified              bool   `json:"wallet_verified"`
	TradeActionsAvailable       bool   `json:"trade_actions_available"`
	ReadOnly                    bool   `json:"read_only"`
}

func ReadRuntimeSummary(db orm.Ormer) (RuntimeSummary, error) {
	summary := RuntimeSummary{
		Environment: binance.CoinMEnvironment(),
		// This read endpoint does not inspect active connections. Never use
		// a local switch as evidence that Binance account/WS is connected.
		PrivateWSVerified: false, WalletVerified: false,
		TradeActionsAvailable: false, ReadOnly: true,
	}
	if db == nil {
		return summary, errors.New("COIN-M runtime summary database unavailable")
	}
	enabled, err := coinMEntryEnabled(db)
	if err != nil {
		return summary, fmt.Errorf("COIN-M global entry switch read failed: %w", err)
	}
	summary.GlobalEntryEnabled = enabled
	for _, item := range []struct {
		query string
		dest  *int64
	}{
		{"SELECT COUNT(*) FROM delivery_symbols WHERE enable=1 AND contract_type='PERPETUAL'", &summary.EnabledPerpetualSymbols},
		{"SELECT COUNT(*) FROM coin_m_managed_positions WHERE remaining_contracts>0 OR status NOT IN ('CLOSED','CANCELED')", &summary.ManagedCyclesNeedingWork},
		{"SELECT COUNT(*) FROM coin_m_managed_orders WHERE status IN ('PENDING_SUBMIT','UNKNOWN','NEW','PARTIALLY_FILLED','ACTIVE','AWAITING_FILLS','TRIGGERED','PENDING_CANCEL') OR (intent!='PROTECT' AND status='FILLED' AND filled_contracts > (SELECT COALESCE(SUM(contracts),0) FROM coin_m_trade_fills WHERE order_id=coin_m_managed_orders.id))", &summary.OrdersNeedingReconciliation},
	} {
		if err := db.Raw(item.query).QueryRow(item.dest); err != nil {
			return summary, fmt.Errorf("COIN-M runtime summary query failed: %w", err)
		}
	}
	return summary, nil
}
