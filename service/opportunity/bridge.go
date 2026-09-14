package opportunity

import (
	"context"
	"fmt"
	"strings"

	"go_binance_futures/models"
)

type TradeProposalCreator interface {
	CreateFromTask(context.Context, string) (models.AgentTradeProposal, error)
}

func (s Service) CreateTradeProposal(ctx context.Context, opportunityID string, creator TradeProposalCreator) (models.AgentTradeProposal, error) {
	if creator == nil {
		return models.AgentTradeProposal{}, fmt.Errorf("trade proposal creator is required")
	}
	_, _ = s.Expire(ctx)
	row, err := s.Get(ctx, opportunityID)
	if err != nil {
		return models.AgentTradeProposal{}, err
	}
	if !CanCreateProposal(row, s.now()) {
		return models.AgentTradeProposal{}, fmt.Errorf("opportunity is not eligible to create a trade proposal")
	}
	if strings.TrimSpace(row.AnalysisTaskID) == "" {
		return models.AgentTradeProposal{}, fmt.Errorf("opportunity has no symbol_analysis task")
	}
	return creator.CreateFromTask(ctx, row.AnalysisTaskID)
}
