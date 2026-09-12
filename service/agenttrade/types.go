package agenttrade

import (
	"context"
	"errors"

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
	StatusProtectionFailed   = "protection_failed"
	StatusClosed             = "closed"
	StatusExpired            = "expired"
)

var ErrManagedPositionClosed = errors.New("agent managed position is already closed")

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

type ProtectionResult struct {
	StopClientOrderID         string `json:"stop_client_order_id,omitempty"`
	StopExchangeOrderID       string `json:"stop_exchange_order_id,omitempty"`
	TakeProfitClientOrderID   string `json:"take_profit_client_order_id,omitempty"`
	TakeProfitExchangeOrderID string `json:"take_profit_exchange_order_id,omitempty"`
	TakeProfitError           string `json:"take_profit_error,omitempty"`
}

type CloseResult struct {
	ClientOrderID   string  `json:"client_order_id,omitempty"`
	ExchangeOrderID string  `json:"exchange_order_id,omitempty"`
	FilledQty       float64 `json:"filled_qty,omitempty"`
	AlreadyClosed   bool    `json:"already_closed,omitempty"`
}

type PositionLifecycle interface {
	EnsureProtection(context.Context, models.AgentTradeProposal) (ProtectionResult, error)
	Close(context.Context, models.AgentTradeProposal) (CloseResult, error)
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
