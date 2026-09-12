package agenttrade

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"go_binance_futures/agent/skills/symbolanalysis"
	"go_binance_futures/agent/task"
	"go_binance_futures/models"

	"github.com/beego/beego/v2/client/orm"
)

type TaskReader interface {
	Get(context.Context, string) (*task.Task, error)
}

type Service struct {
	Store     Store
	Tasks     TaskReader
	Risk      RiskEngine
	Broker    Broker
	Lifecycle PositionLifecycle
	Notifier  TradeNotifier
	Now       func() time.Time
}

func DefaultService() Service {
	store := Store{}
	risk := RiskEngine{Store: store, Data: DefaultRiskDataSource{}}
	return Service{Store: store, Tasks: task.NewORMStore(), Risk: risk, Broker: BinanceBroker{}, Lifecycle: DefaultOwnershipLifecycle(), Notifier: WebTradeNotifier{}}
}

func (s Service) now() time.Time {
	if s.Now != nil {
		return s.Now().UTC()
	}
	return time.Now().UTC()
}

func (s Service) CreateFromTask(ctx context.Context, taskID string) (models.AgentTradeProposal, error) {
	taskID = strings.TrimSpace(taskID)
	if taskID == "" {
		return models.AgentTradeProposal{}, fmt.Errorf("task_id is required")
	}
	if existing, err := s.Store.FindBySourceTask(ctx, taskID); err == nil {
		return existing, nil
	} else if err != orm.ErrNoRows {
		return models.AgentTradeProposal{}, err
	}
	if s.Tasks == nil {
		return models.AgentTradeProposal{}, fmt.Errorf("task reader is required")
	}
	item, err := s.Tasks.Get(ctx, taskID)
	if err != nil {
		return models.AgentTradeProposal{}, err
	}
	if item.Status != task.StatusSucceeded {
		return models.AgentTradeProposal{}, fmt.Errorf("source task must be succeeded")
	}
	if item.Skill != symbolanalysis.Name {
		return models.AgentTradeProposal{}, fmt.Errorf("only symbol_analysis tasks can create V2-12 trade proposals")
	}
	var plan symbolanalysis.TradingPlanV1
	if err := json.Unmarshal(item.Result, &plan); err != nil {
		return models.AgentTradeProposal{}, fmt.Errorf("decode TradingPlanV1: %w", err)
	}
	if plan.Direction != "long" && plan.Direction != "short" {
		return models.AgentTradeProposal{}, fmt.Errorf("neutral analysis cannot create a trade proposal")
	}
	if plan.StopLoss == nil || len(plan.EntryZones) == 0 || len(plan.TakeProfits) == 0 || len(plan.Evidence) == 0 {
		return models.AgentTradeProposal{}, fmt.Errorf("trade proposal requires entry zones, stop loss, take profits and evidence")
	}
	cfg, err := s.Risk.Data.Config(ctx)
	if err != nil {
		return models.AgentTradeProposal{}, err
	}
	ttl := cfg.AgentTradeProposalTTLMin
	if ttl <= 0 {
		ttl = 15
	}
	now := s.now()
	zonesJSON, _ := json.Marshal(plan.EntryZones)
	tpJSON, _ := json.Marshal(plan.TakeProfits)
	invalidJSON, _ := json.Marshal(plan.InvalidationConditions)
	evidenceJSON, _ := json.Marshal(plan.Evidence)
	entryCondition := plan.LongTrigger
	if plan.Direction == "short" {
		entryCondition = plan.ShortTrigger
	}
	marketCondition := 0
	if plan.MarketCondition != nil {
		marketCondition = *plan.MarketCondition
	}
	hash := sha256.Sum256(item.Result)
	proposalID, err := newProposalID()
	if err != nil {
		return models.AgentTradeProposal{}, err
	}
	proposal := models.AgentTradeProposal{
		ProposalID: proposalID, SourceTaskID: item.ID, SourceSkill: item.Skill,
		ContentHash: hex.EncodeToString(hash[:]), Symbol: plan.Symbol,
		Side: strings.ToUpper(plan.Direction), EntryCondition: entryCondition,
		EntryZonesJSON: string(zonesJSON), EntryLow: plan.EntryZones[0].Low, EntryHigh: plan.EntryZones[0].High,
		StopLoss: *plan.StopLoss, TakeProfitsJSON: string(tpJSON), InvalidationsJSON: string(invalidJSON), EvidenceJSON: string(evidenceJSON),
		MarketCondition: marketCondition, Status: StatusRiskRejected, RiskStatus: RiskFail,
		CreatedAt: now.UnixMilli(), UpdatedAt: now.UnixMilli(), ExpiresAt: now.Add(time.Duration(ttl) * time.Minute).UnixMilli(),
	}
	if err := s.Store.SaveProposal(ctx, &proposal); err != nil {
		return models.AgentTradeProposal{}, err
	}
	_ = s.Store.Audit(ctx, proposal.ProposalID, "proposal_created", "success", "agent_task", map[string]any{"source_task_id": item.ID, "content_hash": proposal.ContentHash})
	return s.EvaluateRisk(ctx, proposal.ProposalID)
}

func (s Service) EvaluateRisk(ctx context.Context, proposalID string) (models.AgentTradeProposal, error) {
	proposal, err := s.Store.GetProposal(ctx, proposalID)
	if err != nil {
		return proposal, err
	}
	now := s.now().UnixMilli()
	if proposal.ExpiresAt <= now {
		proposal.Status, proposal.UpdatedAt = StatusExpired, now
		_ = s.Store.SaveProposal(ctx, &proposal)
		_ = s.Store.Audit(ctx, proposal.ProposalID, "risk_check", "expired", "risk_engine", nil)
		return proposal, fmt.Errorf("proposal is expired")
	}
	result, err := s.Risk.Check(ctx, proposal)
	if err != nil {
		return proposal, err
	}
	proposal.RiskStatus, proposal.RiskJSON, proposal.RiskCheckedAt = result.Status, encodeRisk(result), result.CheckedAt
	proposal.ReferencePrice, proposal.Quantity, proposal.Leverage = result.ReferencePrice, result.Quantity, result.Leverage
	proposal.NotionalUSDT, proposal.RiskUSDT, proposal.UpdatedAt = result.NotionalUSDT, result.RiskUSDT, now
	if result.Status == RiskPass {
		if proposal.Status != StatusApproved {
			proposal.Status = StatusAwaitingApproval
		}
	} else {
		proposal.Status = StatusRiskRejected
		proposal.ApprovedAt, proposal.ApprovedBy = 0, ""
	}
	if err := s.Store.SaveProposal(ctx, &proposal); err != nil {
		return proposal, err
	}
	_ = s.Store.Audit(ctx, proposal.ProposalID, "risk_check", result.Status, "risk_engine", result)
	return proposal, nil
}

func (s Service) Approve(ctx context.Context, proposalID, actor string) (models.AgentTradeProposal, error) {
	proposal, err := s.Store.GetProposal(ctx, proposalID)
	if err != nil {
		return proposal, err
	}
	if proposal.Status == StatusApproved {
		return proposal, nil
	}
	if proposal.Status == StatusExecuted || proposal.Status == StatusExecuting || proposal.Status == StatusExecutionUncertain || proposal.Status == StatusProtectionFailed || proposal.Status == StatusClosed {
		return proposal, fmt.Errorf("proposal status %q cannot be approved", proposal.Status)
	}
	proposal, err = s.EvaluateRisk(ctx, proposalID)
	if err != nil {
		return proposal, err
	}
	if proposal.RiskStatus != RiskPass || proposal.Status != StatusAwaitingApproval {
		return proposal, fmt.Errorf("proposal did not pass deterministic risk checks")
	}
	actor = strings.TrimSpace(actor)
	if actor == "" {
		actor = "web_admin"
	}
	now := s.now().UnixMilli()
	proposal.Status, proposal.ApprovedBy, proposal.ApprovedAt, proposal.UpdatedAt = StatusApproved, actor, now, now
	proposal.RejectedReason, proposal.RejectedAt = "", 0
	if err := s.Store.SaveProposal(ctx, &proposal); err != nil {
		return proposal, err
	}
	_ = s.Store.Audit(ctx, proposal.ProposalID, "approved", "success", actor, map[string]any{"risk_checked_at": proposal.RiskCheckedAt})
	return proposal, nil
}

func (s Service) Reject(ctx context.Context, proposalID, actor, reason string) (models.AgentTradeProposal, error) {
	proposal, err := s.Store.GetProposal(ctx, proposalID)
	if err != nil {
		return proposal, err
	}
	if proposal.Status == StatusRejected {
		return proposal, nil
	}
	if proposal.Status == StatusExecuting || proposal.Status == StatusExecuted || proposal.Status == StatusExecutionUncertain || proposal.Status == StatusProtectionFailed || proposal.Status == StatusClosed {
		return proposal, fmt.Errorf("proposal status %q cannot be rejected", proposal.Status)
	}
	actor = strings.TrimSpace(actor)
	if actor == "" {
		actor = "web_admin"
	}
	reason = strings.TrimSpace(reason)
	if reason == "" {
		reason = "rejected by operator"
	}
	now := s.now().UnixMilli()
	proposal.Status, proposal.RejectedReason, proposal.RejectedAt, proposal.UpdatedAt = StatusRejected, reason, now, now
	proposal.ApprovedBy, proposal.ApprovedAt = "", 0
	if err := s.Store.SaveProposal(ctx, &proposal); err != nil {
		return proposal, err
	}
	_ = s.Store.Audit(ctx, proposal.ProposalID, "rejected", "success", actor, map[string]any{"reason": reason})
	return proposal, nil
}

func (s Service) Execute(ctx context.Context, proposalID, actor string) (models.AgentTradeProposal, models.AgentTradeExecution, error) {
	proposal, err := s.Store.GetProposal(ctx, proposalID)
	if err != nil {
		return proposal, models.AgentTradeExecution{}, err
	}
	if proposal.Status == StatusExecuted || proposal.Status == StatusProtectionFailed {
		execution, execErr := s.Store.GetExecution(ctx, proposalID)
		if execErr != nil {
			return proposal, execution, execErr
		}
		if s.Lifecycle == nil {
			return proposal, execution, nil
		}
		return s.reconcileExecutedLifecycle(ctx, proposal, execution, actor)
	}
	if proposal.Status == StatusClosed {
		execution, execErr := s.Store.GetExecution(ctx, proposalID)
		if execErr != nil {
			return proposal, execution, execErr
		}
		return proposal, execution, fmt.Errorf("proposal position is already closed")
	}
	if proposal.Status == StatusExecutionUncertain || proposal.Status == StatusExecuting {
		return s.Reconcile(ctx, proposalID, actor)
	}
	if proposal.Status != StatusApproved {
		return proposal, models.AgentTradeExecution{}, fmt.Errorf("proposal must be approved before execution")
	}
	proposal, err = s.EvaluateRisk(ctx, proposalID)
	if err != nil {
		return proposal, models.AgentTradeExecution{}, err
	}
	if proposal.Status != StatusApproved || proposal.RiskStatus != RiskPass {
		return proposal, models.AgentTradeExecution{}, fmt.Errorf("proposal no longer passes deterministic risk checks")
	}
	now := s.now().UnixMilli()
	claimed, err := s.Store.CompareAndSetProposalStatus(ctx, proposalID, StatusApproved, StatusExecuting, now)
	if err != nil {
		return proposal, models.AgentTradeExecution{}, err
	}
	if !claimed {
		latest, loadErr := s.Store.GetProposal(ctx, proposalID)
		if loadErr != nil {
			return proposal, models.AgentTradeExecution{}, loadErr
		}
		if latest.Status == StatusExecuted || latest.Status == StatusExecuting || latest.Status == StatusExecutionUncertain {
			return s.Reconcile(ctx, proposalID, actor)
		}
		return latest, models.AgentTradeExecution{}, fmt.Errorf("proposal execution was not claimed; current status %q", latest.Status)
	}
	proposal.Status, proposal.UpdatedAt = StatusExecuting, now
	clientOrderID := executionClientOrderID(proposal.ProposalID)
	execution := models.AgentTradeExecution{
		ProposalID: proposal.ProposalID, IdempotencyKey: proposal.ProposalID,
		ClientOrderID: clientOrderID, Status: StatusExecuting, Symbol: proposal.Symbol,
		Side: proposal.Side, OrderType: "MARKET", Quantity: proposal.Quantity,
		ReferencePrice: proposal.ReferencePrice, Leverage: proposal.Leverage,
		CreatedAt: now, UpdatedAt: now, SubmittedAt: now,
	}
	if err := s.Store.SaveExecution(ctx, &execution); err != nil {
		return proposal, execution, err
	}
	actor = strings.TrimSpace(actor)
	if actor == "" {
		actor = "web_admin"
	}
	_ = s.Store.Audit(ctx, proposal.ProposalID, "execution_claimed", "success", actor, map[string]any{"client_order_id": clientOrderID})
	return s.submitClaimed(ctx, proposal, execution, actor)
}

func (s Service) submitClaimed(ctx context.Context, proposal models.AgentTradeProposal, execution models.AgentTradeExecution, actor string) (models.AgentTradeProposal, models.AgentTradeExecution, error) {
	if s.Broker == nil {
		return proposal, execution, fmt.Errorf("trade broker is required")
	}
	if s.Risk.Data == nil {
		return proposal, execution, fmt.Errorf("risk data source is required")
	}
	cfg, cfgErr := s.Risk.Data.Config(ctx)
	if cfgErr != nil || cfg.AgentTradeExecutionEnable != 1 {
		reason := "AI trade execution kill switch is disabled before broker submission"
		if cfgErr != nil {
			reason = "cannot verify AI trade execution kill switch: " + cfgErr.Error()
		}
		now := s.now().UnixMilli()
		execution.Status, execution.Error, execution.UpdatedAt, execution.CompletedAt = StatusExecutionFailed, reason, now, now
		_ = s.Store.SaveExecution(ctx, &execution)
		proposal.Status, proposal.RiskStatus, proposal.UpdatedAt = StatusRiskRejected, RiskFail, now
		_ = s.Store.SaveProposal(ctx, &proposal)
		_ = s.Store.Audit(ctx, proposal.ProposalID, "execution_blocked", "risk_rejected", actor, map[string]any{"reason": reason})
		return proposal, execution, fmt.Errorf("%s", reason)
	}
	request := BrokerOrderRequest{
		ProposalID: proposal.ProposalID, ClientOrderID: execution.ClientOrderID,
		Symbol: proposal.Symbol, Side: proposal.Side, Quantity: proposal.Quantity,
		Leverage: proposal.Leverage, ReferencePrice: proposal.ReferencePrice,
	}
	result, err := s.Broker.SubmitMarket(ctx, request)
	if err == nil {
		return s.markExecuted(ctx, proposal, execution, result, actor)
	}
	_ = s.Store.Audit(ctx, proposal.ProposalID, "broker_submit", "error", actor, map[string]any{"error": err.Error(), "client_order_id": execution.ClientOrderID})
	lookup, lookupErr := s.Broker.LookupByClientOrderID(ctx, proposal.Symbol, execution.ClientOrderID)
	if lookupErr == nil && strings.TrimSpace(lookup.ExchangeOrderID) != "" {
		_ = s.Store.Audit(ctx, proposal.ProposalID, "broker_reconcile_after_error", "found", actor, map[string]any{"exchange_order_id": lookup.ExchangeOrderID})
		return s.markExecuted(ctx, proposal, execution, lookup, actor)
	}
	now := s.now().UnixMilli()
	execution.Status, execution.Error, execution.UpdatedAt = StatusExecutionUncertain, err.Error(), now
	_ = s.Store.SaveExecution(ctx, &execution)
	proposal.Status, proposal.UpdatedAt = StatusExecutionUncertain, now
	_ = s.Store.SaveProposal(ctx, &proposal)
	_ = s.Store.Audit(ctx, proposal.ProposalID, "execution_uncertain", "warning", actor, map[string]any{"submit_error": err.Error(), "lookup_error": errorText(lookupErr), "client_order_id": execution.ClientOrderID})
	return proposal, execution, fmt.Errorf("execution result is uncertain; reconcile by client_order_id before any retry")
}

func (s Service) markExecuted(ctx context.Context, proposal models.AgentTradeProposal, execution models.AgentTradeExecution, result BrokerOrderResult, actor string) (models.AgentTradeProposal, models.AgentTradeExecution, error) {
	now := s.now().UnixMilli()
	execution.Status, execution.ExchangeOrderID, execution.AveragePrice = StatusExecuted, strings.TrimSpace(result.ExchangeOrderID), result.AveragePrice
	if result.ClientOrderID != "" {
		execution.ClientOrderID = result.ClientOrderID
	}
	execution.Error, execution.UpdatedAt, execution.CompletedAt = "", now, now
	if err := s.Store.SaveExecution(ctx, &execution); err != nil {
		return proposal, execution, err
	}
	proposal.Status, proposal.ExecutedAt, proposal.UpdatedAt = StatusExecuted, now, now
	if err := s.Store.SaveProposal(ctx, &proposal); err != nil {
		return proposal, execution, err
	}
	_ = s.Store.Audit(ctx, proposal.ProposalID, "executed", "success", actor, map[string]any{"exchange_order_id": execution.ExchangeOrderID, "client_order_id": execution.ClientOrderID, "average_price": execution.AveragePrice})
	if s.Lifecycle == nil {
		return proposal, execution, nil
	}
	return s.reconcileExecutedLifecycle(ctx, proposal, execution, actor)
}

func (s Service) Reconcile(ctx context.Context, proposalID, actor string) (models.AgentTradeProposal, models.AgentTradeExecution, error) {
	proposal, err := s.Store.GetProposal(ctx, proposalID)
	if err != nil {
		return proposal, models.AgentTradeExecution{}, err
	}
	execution, err := s.Store.GetExecution(ctx, proposalID)
	if err == orm.ErrNoRows && proposal.Status == StatusExecuting {
		now := s.now().UnixMilli()
		execution = models.AgentTradeExecution{
			ProposalID: proposal.ProposalID, IdempotencyKey: proposal.ProposalID,
			ClientOrderID: executionClientOrderID(proposal.ProposalID), Status: StatusExecutionUncertain,
			Symbol: proposal.Symbol, Side: proposal.Side, OrderType: "MARKET", Quantity: proposal.Quantity,
			ReferencePrice: proposal.ReferencePrice, Leverage: proposal.Leverage,
			CreatedAt: now, UpdatedAt: now,
		}
		_ = s.Store.SaveExecution(ctx, &execution)
	} else if err != nil {
		return proposal, execution, err
	}
	if execution.Status == StatusExecuted || proposal.Status == StatusExecuted || proposal.Status == StatusProtectionFailed {
		if s.Lifecycle == nil {
			return proposal, execution, nil
		}
		return s.reconcileExecutedLifecycle(ctx, proposal, execution, actor)
	}
	if proposal.Status == StatusClosed {
		return proposal, execution, nil
	}
	if s.Broker == nil {
		return proposal, execution, fmt.Errorf("trade broker is required")
	}
	actor = strings.TrimSpace(actor)
	if actor == "" {
		actor = "web_admin"
	}
	result, lookupErr := s.Broker.LookupByClientOrderID(ctx, proposal.Symbol, execution.ClientOrderID)
	if lookupErr != nil || strings.TrimSpace(result.ExchangeOrderID) == "" {
		now := s.now().UnixMilli()
		execution.Status, execution.Error, execution.UpdatedAt = StatusExecutionUncertain, errorText(lookupErr), now
		_ = s.Store.SaveExecution(ctx, &execution)
		proposal.Status, proposal.UpdatedAt = StatusExecutionUncertain, now
		_ = s.Store.SaveProposal(ctx, &proposal)
		_ = s.Store.Audit(ctx, proposal.ProposalID, "reconcile", "not_found", actor, map[string]any{"client_order_id": execution.ClientOrderID, "error": errorText(lookupErr)})
		return proposal, execution, fmt.Errorf("order is not yet confirmed by client_order_id; do not resubmit")
	}
	_ = s.Store.Audit(ctx, proposal.ProposalID, "reconcile", "found", actor, map[string]any{"exchange_order_id": result.ExchangeOrderID})
	return s.markExecuted(ctx, proposal, execution, result, actor)
}

func (s Service) reconcileExecutedLifecycle(ctx context.Context, proposal models.AgentTradeProposal, execution models.AgentTradeExecution, actor string) (models.AgentTradeProposal, models.AgentTradeExecution, error) {
	actor = strings.TrimSpace(actor)
	if actor == "" {
		actor = "web_admin"
	}
	protection, err := s.Lifecycle.EnsureProtection(ctx, proposal)
	now := s.now().UnixMilli()
	if errors.Is(err, ErrManagedPositionClosed) {
		if _, cleanupErr := s.Lifecycle.Close(ctx, proposal); cleanupErr != nil {
			_ = s.Store.Audit(ctx, proposal.ProposalID, "protection_cleanup", "error", actor, map[string]any{"error": cleanupErr.Error()})
			return proposal, execution, cleanupErr
		}
		proposal.Status, proposal.UpdatedAt = StatusClosed, now
		if saveErr := s.Store.SaveProposal(ctx, &proposal); saveErr != nil {
			return proposal, execution, saveErr
		}
		_ = s.Store.Audit(ctx, proposal.ProposalID, "protection_reconcile", "position_closed", actor, nil)
		return proposal, execution, nil
	}
	if err != nil {
		proposal.Status, proposal.UpdatedAt = StatusProtectionFailed, now
		if saveErr := s.Store.SaveProposal(ctx, &proposal); saveErr != nil {
			return proposal, execution, saveErr
		}
		_ = s.Store.Audit(ctx, proposal.ProposalID, "protection", StatusProtectionFailed, actor, map[string]any{"error": err.Error(), "result": protection})
		if s.Notifier != nil {
			_ = s.Notifier.ProtectionFailed(ctx, proposal, err)
		}
		return proposal, execution, fmt.Errorf("entry is executed but protective stop is not confirmed: %w", err)
	}
	proposal.Status, proposal.UpdatedAt = StatusExecuted, now
	if err := s.Store.SaveProposal(ctx, &proposal); err != nil {
		return proposal, execution, err
	}
	status := "success"
	if protection.TakeProfitError != "" {
		status = "warning"
	}
	_ = s.Store.Audit(ctx, proposal.ProposalID, "protection", status, actor, protection)
	return proposal, execution, nil
}

func (s Service) Close(ctx context.Context, proposalID, actor string) (models.AgentTradeProposal, CloseResult, error) {
	proposal, err := s.Store.GetProposal(ctx, proposalID)
	if err != nil {
		return proposal, CloseResult{}, err
	}
	if proposal.Status == StatusClosed {
		return proposal, CloseResult{AlreadyClosed: true}, nil
	}
	if proposal.Status != StatusExecuted && proposal.Status != StatusProtectionFailed {
		return proposal, CloseResult{}, fmt.Errorf("proposal status %q has no agent managed position to close", proposal.Status)
	}
	if s.Lifecycle == nil {
		return proposal, CloseResult{}, fmt.Errorf("position lifecycle is required")
	}
	actor = strings.TrimSpace(actor)
	if actor == "" {
		actor = "web_admin"
	}
	result, closeErr := s.Lifecycle.Close(ctx, proposal)
	if closeErr != nil {
		_ = s.Store.Audit(ctx, proposal.ProposalID, "managed_close", "error", actor, map[string]any{"error": closeErr.Error(), "result": result})
		return proposal, result, closeErr
	}
	now := s.now().UnixMilli()
	proposal.Status, proposal.UpdatedAt = StatusClosed, now
	if err := s.Store.SaveProposal(ctx, &proposal); err != nil {
		return proposal, result, err
	}
	_ = s.Store.Audit(ctx, proposal.ProposalID, "managed_close", "success", actor, result)
	return proposal, result, nil
}

func newProposalID() (string, error) {
	buffer := make([]byte, 12)
	if _, err := rand.Read(buffer); err != nil {
		return "", err
	}
	return "tp_" + hex.EncodeToString(buffer), nil
}

func executionClientOrderID(proposalID string) string {
	value := "agt_" + strings.TrimSpace(proposalID)
	if len(value) > 36 {
		value = value[:36]
	}
	return value
}

func errorText(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
