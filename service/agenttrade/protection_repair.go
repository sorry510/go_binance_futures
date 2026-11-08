package agenttrade

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"strings"
	"time"

	"go_binance_futures/models"
	"go_binance_futures/service/futuresownership"

	"github.com/beego/beego/v2/client/orm"
	"github.com/beego/beego/v2/core/logs"
)

// A separate, explicit operator opt-in is required for *any* new real
// automatic SL/TP submission. Default is disabled, including after upgrade.
const ProtectionAutoRepairEnv = "BINANCE_AGENT_PROTECTION_AUTO_REPAIR"
const ProtectionAutoRepairEnabledValue = "enabled"

func AutomaticProtectionRepairEnabled() bool {
	return os.Getenv(ProtectionAutoRepairEnv) == ProtectionAutoRepairEnabledValue
}

// A single unresolved owner can indicate stale shared margin/position
// state. Do not authorize new protective submissions from a partial pass.
func ProtectionRepairReconcileOK(summaries []futuresownership.ReconcileSummary) bool {
	if len(summaries) == 0 {
		return false
	}
	hasAgent := false
	for _, summary := range summaries {
		if summary.OrdersUnresolved != 0 {
			return false
		}
		if summary.Owner == futuresownership.OwnerAgentTrade {
			hasAgent = true
		}
	}
	return hasAgent
}

const maxAgentRepairCyclesPerSweep = 2

func eligibleRepairProposal(p models.AgentTradeProposal, pos models.FuturesManagedPosition, now int64) error {
	if p.ProposalID == "" || p.ProposalID != strings.TrimSpace(pos.SourceRef) ||
		p.Symbol != pos.Symbol || !strings.EqualFold(p.Side, pos.PositionSide) {
		return fmt.Errorf("proposal ownership identity mismatch")
	}
	if p.Status != StatusExecuted && p.Status != StatusProtectionFailed {
		return fmt.Errorf("proposal status %q cannot authorize automatic protection", p.Status)
	}
	if p.RiskStatus != RiskPass || p.ApprovedAt <= 0 || p.ExecutedAt <= 0 || p.RejectedAt != 0 {
		return fmt.Errorf("proposal was not approved, executed or risk-passed")
	}
	if p.ExpiresAt <= now || p.ExpiresAt <= 0 {
		return fmt.Errorf("proposal has expired; manual reassessment required")
	}
	if math.IsNaN(p.StopLoss) || math.IsInf(p.StopLoss, 0) || p.StopLoss <= 0 {
		return fmt.Errorf("proposal has no valid approved stop-loss trigger")
	}
	return nil
}

func activeExactProtection(orders []models.FuturesManagedOrder, pos models.FuturesManagedPosition, intent string) (bool, error) {
	exact := false
	stale := false
	for _, order := range orders {
		if order.PositionID == nil || *order.PositionID != pos.ID ||
			order.SourceRef != pos.SourceRef || order.Intent != intent {
			continue
		}
		if order.Status == futuresownership.OrderPending || order.Status == futuresownership.OrderReconcile {
			return false, fmt.Errorf("exchange state unresolved for %s (%s)", intent, order.ClientOrderID)
		}
		if order.Status != futuresownership.OrderSubmitted && order.Status != futuresownership.OrderPartiallyFilled {
			continue
		}
		qty := order.RequestedQty - order.FilledQty
		if order.ExchangeOrderID == "" {
			return false, fmt.Errorf("active %s has unknown exchange identity %s", intent, order.ClientOrderID)
		}
		if math.Abs(qty-pos.ManagedQty) > 1e-9 {
			stale = true
			continue
		}
		if exact {
			return false, fmt.Errorf("duplicate active %s protection", intent)
		}
		exact = true
	}
	if stale && exact {
		return false, fmt.Errorf("multiple simultaneous active %s protections: exact and stale exchange quantities", intent)
	}
	if stale {
		// nextProtectionClientOrderID cancels this stale order before
		// replacing it; never trust a local requested_qty rewrite.
		return false, nil
	}
	return exact, nil
}

func agentTakeProfitTrigger(proposal models.AgentTradeProposal) (float64, bool, error) {
	if strings.TrimSpace(proposal.TakeProfitsJSON) == "" {
		return 0, false, nil
	}
	var values []float64
	if err := json.Unmarshal([]byte(proposal.TakeProfitsJSON), &values); err != nil {
		return 0, false, err
	}
	if len(values) == 0 {
		return 0, false, nil
	}
	if math.IsNaN(values[0]) || math.IsInf(values[0], 0) || values[0] <= 0 {
		return 0, false, fmt.Errorf("invalid take-profit trigger")
	}
	return values[0], true, nil
}

// RepairActiveManagedProtections is an opt-in-capable single sweep. The
// production scheduler checks the independent opt-in and a fresh, fully
// resolved reconciliation before invoking it. The method also fails closed
// on all proposal/ownership checks for direct callers, including tests.
// Each attempted order uses a pre-submit durable audit and a deterministic
// client-order ID; a crash cannot justify blind resubmission.
func (l OwnershipLifecycle) RepairActiveManagedProtections(ctx context.Context) (int, error) {
	if !AutomaticProtectionRepairEnabled() {
		return 0, fmt.Errorf("automatic agent protection re-arm disabled (explicit opt-in required)")
	}
	positions, err := l.ownership().ActivePositions(ctx, futuresownership.OwnerAgentTrade)
	if err != nil {
		return 0, err
	}
	if len(positions) == 0 {
		return 0, nil
	}
	orders, err := l.ownership().ActiveOrders(ctx, futuresownership.OwnerAgentTrade)
	if err != nil {
		return 0, err
	}
	repaired := 0
	attemptedCycles := 0
	failures := make([]error, 0)
	store := Store{}
	for _, pos := range positions {
		if err := ctx.Err(); err != nil {
			return repaired, err
		}
		if pos.ManagedQty <= 1e-12 {
			continue
		}
		stopOK, stopErr := activeExactProtection(orders, pos, futuresownership.IntentStopLoss)
		tpOK, tpErr := activeExactProtection(orders, pos, futuresownership.IntentTakeProfit)
		if stopErr != nil || tpErr != nil {
			failures = append(failures, fmt.Errorf("position %d protection unresolved: stop=%v tp=%v", pos.ID, stopErr, tpErr))
			continue
		}
		// A missing TP is repairable only if the approved proposal included
		// it. We never invent a profit target for a stop-only proposal.
		if stopOK && tpOK {
			continue
		}
		var proposal models.AgentTradeProposal
		ref := strings.TrimSpace(pos.SourceRef)
		if ref == "" {
			failures = append(failures, fmt.Errorf("position %d has no proposal source", pos.ID))
			continue
		}
		if e := orm.NewOrm().QueryTable(new(models.AgentTradeProposal)).
			Filter("proposal_id", ref).One(&proposal); e != nil {
			failures = append(failures, fmt.Errorf("position %d proposal unavailable: %w", pos.ID, e))
			continue
		}
		if e := eligibleRepairProposal(proposal, pos, time.Now().UnixMilli()); e != nil {
			failures = append(failures, fmt.Errorf("position %d ineligible: %w", pos.ID, e))
			continue
		}
		tpPrice, tpConfigured, e := agentTakeProfitTrigger(proposal)
		if e != nil {
			failures = append(failures, fmt.Errorf("position %d TP malformed: %w", pos.ID, e))
			continue
		}
		if stopOK && (tpOK || !tpConfigured) {
			continue
		}
		if attemptedCycles >= maxAgentRepairCyclesPerSweep {
			failures = append(failures, fmt.Errorf("position %d deferred by automatic protection repair budget", pos.ID))
			continue
		}
		attemptedCycles++
		didRearm := false
		for _, spec := range []struct {
			intent, kind, orderType string
			price                   float64
			needed                  bool
		}{
			{futuresownership.IntentStopLoss, "sl", "STOP_MARKET", proposal.StopLoss, !stopOK},
			{futuresownership.IntentTakeProfit, "tp", "TAKE_PROFIT_MARKET", tpPrice, tpConfigured && !tpOK},
		} {
			if !spec.needed {
				continue
			}
			// This audit must persist BEFORE any exchange order is created.
			// A failure to audit prevents trading.
			detail := map[string]any{"position_id": pos.ID, "account_id": pos.AccountID,
				"symbol": pos.Symbol, "intent": spec.intent, "remaining_qty": pos.ManagedQty,
				"stop_price": spec.price, "reason": "missing_or_canceled_protection"}
			if e := store.Audit(ctx, proposal.ProposalID, "protection_rearm_attempt", "pending", "ownership_repair", detail); e != nil {
				failures = append(failures, fmt.Errorf("position %d audit failed before submit: %w", pos.ID, e))
				break
			}
			clientID, e := l.nextProtectionClientOrderID(ctx, proposal, spec.intent, spec.kind, pos.ManagedQty)
			if e == nil {
				_, e = l.submitProtection(ctx, proposal, pos.ManagedQty, spec.price, spec.intent, spec.orderType, closeSide(proposal.Side), clientID)
			}
			if e != nil {
				failures = append(failures, fmt.Errorf("position %d rearm %s failed: %w", pos.ID, spec.intent, e))
				_ = store.Audit(ctx, proposal.ProposalID, "protection_rearm", "error", "ownership_repair",
					map[string]any{"position_id": pos.ID, "intent": spec.intent, "error": e.Error()})
				break
			}
			row, e := l.ownership().GetOrder(ctx, clientID)
			if e != nil || row.PositionID == nil || *row.PositionID != pos.ID ||
				row.ExchangeOrderID == "" ||
				(row.Status != futuresownership.OrderSubmitted && row.Status != futuresownership.OrderPartiallyFilled) ||
				math.Abs(row.RequestedQty-row.FilledQty-pos.ManagedQty) > 1e-9 {
				failures = append(failures, fmt.Errorf("position %d %s replacement is not exchange-confirmed: %v", pos.ID, spec.intent, e))
				break
			}
			if e := store.Audit(ctx, proposal.ProposalID, "protection_rearm", "success", "ownership_repair",
				map[string]any{"position_id": pos.ID, "account_id": pos.AccountID, "intent": spec.intent,
					"remaining_qty": pos.ManagedQty, "exchange_order_id": row.ExchangeOrderID,
					"client_order_id": row.ClientOrderID}); e != nil {
				logs.Warning("报警: protection rearm completed but audit result failed position=", pos.ID, " error=", e)
			}
			didRearm = true
		}
		if didRearm {
			repaired++
		}
	}
	if len(failures) > 0 {
		for _, e := range failures {
			logs.Warning("报警: agent protection cycle repair unresolved: ", e)
		}
		return repaired, errors.Join(failures...)
	}
	return repaired, nil
}
