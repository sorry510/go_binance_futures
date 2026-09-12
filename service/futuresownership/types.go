package futuresownership

const (
	OwnerAutoStrategy    = "auto_strategy"
	OwnerNewCoinRush     = "new_coin_rush"
	OwnerNoticeAutoOrder = "notice_auto_order"
	OwnerFundingRate     = "funding_rate"
	OwnerAgentTrade      = "agent_trade"
)

const (
	PositionActive            = "active"
	PositionSuspended         = "suspended"
	PositionClosed            = "closed"
	PositionReconcileRequired = "reconcile_required"
)

const (
	OrderPending         = "pending"
	OrderSubmitted       = "submitted"
	OrderPartiallyFilled = "partially_filled"
	OrderFilled          = "filled"
	OrderCanceled        = "canceled"
	OrderFailed          = "failed"
	OrderReconcile       = "reconcile_required"
)

const (
	IntentOpen       = "open"
	IntentClose      = "close"
	IntentTakeProfit = "take_profit"
	IntentStopLoss   = "stop_loss"
)

type ClaimOrderInput struct {
	Owner         string
	Symbol        string
	PositionSide  string
	Intent        string
	ClientOrderID string
	RequestedQty  float64
	OrderType     string
	SourceRef     string
}
