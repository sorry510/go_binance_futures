package agenttrade

import (
	"context"

	"go_binance_futures/models"
)

const (
	StatusRiskRejected       = "risk_rejected"
	StatusAwaitingApproval   = "awaiting_approval"
	StatusApproved           = "approved"
	StatusRejected           = "rejected"
	StatusExecuting          = "executing"
	StatusExecuted           = "executed"
	StatusExecutionFailed    = "execution_failed"
	StatusExecutionUncertain = "execution_uncertain"
	StatusExpired            = "expired"
)

const (
	RiskPass = "pass"
	RiskFail = "fail"
)

type RiskCheck struct {
	Name    string `json:"name"`
	Passed  bool   `json:"passed"`
	Message string `json:"message,omitempty"`
}

type RiskResult struct {
	Status              string      `json:"status"`
	Checks              []RiskCheck `json:"checks"`
	ReferencePrice      float64     `json:"reference_price"`
	EstimatedFillPrice  float64     `json:"estimated_fill_price"`
	SlippageBps         float64     `json:"slippage_bps"`
	Quantity            float64     `json:"quantity"`
	Leverage            int         `json:"leverage"`
	NotionalUSDT        float64     `json:"notional_usdt"`
	RiskUSDT            float64     `json:"risk_usdt"`
	CurrentExposureUSDT float64     `json:"current_exposure_usdt"`
	CheckedAt           int64       `json:"checked_at"`
}

type ListOptions struct {
	Status string
	Symbol string
	Page   int
	Limit  int
}

type ListResult struct {
	Page  int                         `json:"page"`
	Limit int                         `json:"limit"`
	Total int64                       `json:"total"`
	List  []models.AgentTradeProposal `json:"list"`
}

type ApprovalInput struct {
	Actor  string `json:"actor"`
	Reason string `json:"reason,omitempty"`
}

type BrokerOrderRequest struct {
	ProposalID     string
	ClientOrderID  string
	Symbol         string
	Side           string
	Quantity       float64
	Leverage       int
	ReferencePrice float64
}

type BrokerOrderResult struct {
	ExchangeOrderID string
	ClientOrderID   string
	AveragePrice    float64
}

type Broker interface {
	SubmitMarket(context.Context, BrokerOrderRequest) (BrokerOrderResult, error)
	LookupByClientOrderID(context.Context, string, string) (BrokerOrderResult, error)
}
