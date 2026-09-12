package agenttrade

import (
	"context"
	"fmt"

	"go_binance_futures/models"
	"go_binance_futures/webnotification"
)

type TradeNotifier interface {
	ProtectionFailed(context.Context, models.AgentTradeProposal, error) error
}

type WebTradeNotifier struct{}

func (WebTradeNotifier) ProtectionFailed(ctx context.Context, proposal models.AgentTradeProposal, cause error) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	_, err := webnotification.PublishWithOptions(
		"agent_trade",
		fmt.Sprintf("AI 受控交易保护单失败\nProposal: %s\nSymbol: %s %s\nError: %s", proposal.ProposalID, proposal.Symbol, proposal.Side, cause.Error()),
		webnotification.PublishOptions{Level: "error", EventType: "agent_trade_protection_failed", EventID: proposal.ProposalID, Symbol: proposal.Symbol},
	)
	return err
}
