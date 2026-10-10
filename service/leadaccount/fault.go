package leadaccount

import (
	binance "go_binance_futures/feature/api/binance"
	"strings"
)

// ClassifyLeadOpenRejection keeps ordinary trade-specific rejections out of
// the account-wide breaker. No user-controlled/error text is returned.
func ClassifyLeadOpenRejection(reason string) (LeadFaultClass, LeadFaultCode) {
	switch reason {
	case "lead_daily_loss_limit":
		return FaultTrip, FaultDailyLoss
	case "lead_drawdown_limit":
		return FaultTrip, FaultDrawdown
	case "lead_exchange_rules_unavailable":
		return FaultTemporary, FaultRulesUnavailable
	case "lead_ws_not_healthy":
		return FaultTemporary, FaultWSUnavailable
	case "portfolio_binding_not_confirmed":
		return FaultHardUnavailable, FaultPortfolioUnconfirmed
	case "lead_identity_not_verified":
		return FaultHardUnavailable, FaultIdentityInvalid
	case "lead_snapshot_stale", "lead_daily_pnl_unavailable", "lead_balance_invalid",
		"lead_snapshot_incomplete_or_uncertain", "lead_position_snapshot_invalid", "lead_open_order_snapshot_invalid":
		return FaultTemporary, FaultSnapshotInvalid
	case "lead_pending_reconcile":
		return FaultReconcile, FaultPendingReconcile
	case "lead_order_notional_limit", "lead_total_notional_limit",
		"lead_available_margin_insufficient", "lead_losing_position_count_limit",
		"lead_position_count_limit", "lead_exchange_notional_limits",
		"lead_symbol_not_allowed", "lead_risk_limits_not_configured",
		"lead_shared_coin_budget_exceeded", "lead_exchange_rules_unverified",
		"invalid_lead_symbol_or_side", "lead_order_parameters_invalid",
		"lead_leverage_not_allowed", "lead_same_side_position_or_order_exists",
		"new_opens_paused", "lead_disabled", "stage7_live_authorization_required":
		return FaultSkip, ""
	default:
		return FaultSkip, ""
	}
}
func (g *LeadRuntimeGuard) recordRiskDecision(account binance.AccountID, decision RiskDecision) {
	if g == nil || account != binance.LeadAccountID {
		return
	}
	seen := map[LeadFaultCode]struct{}{}
	for _, code := range decision.BlockingReasons {
		class, fault := ClassifyLeadOpenRejection(strings.TrimSpace(code))
		if fault == "" || class == FaultSkip {
			continue
		}
		if _, exists := seen[fault]; exists {
			continue
		}
		seen[fault] = struct{}{}
		g.RecordFault(account, fault)
	}
}
