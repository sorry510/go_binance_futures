package leadaccount

import (
	"context"
	"fmt"
	"strings"

	"github.com/adshao/go-binance/v2/futures"
	binance "go_binance_futures/feature/api/binance"
)

type LeadRuleReader interface {
	GetLeadExchangeInfoContext(context.Context) (*futures.ExchangeInfo, error)
	GetPositionFreshContext(context.Context, binance.PositionParams) ([]*futures.PositionRisk, error)
	GetLeadLeverageBracketContext(context.Context, string) ([]*futures.LeverageBracket, error)
}
type LeadRuleSource struct {
	AccountID binance.AccountID
	Reader    LeadRuleReader
}

func NewLeadRuleSource(a *binance.AccountClient) (*LeadRuleSource, error) {
	if a == nil || a.ID() != binance.LeadAccountID {
		return nil, fmt.Errorf("Lead order rules require a Lead client")
	}
	return &LeadRuleSource{AccountID: binance.LeadAccountID, Reader: a}, nil
}
func (s *LeadRuleSource) Verify(ctx context.Context, order RiskOpenOrder, margin futures.MarginType) (VerifiedOrderRules, error) {
	if s == nil || s.AccountID != binance.LeadAccountID || s.Reader == nil || order.AccountID != binance.LeadAccountID {
		return VerifiedOrderRules{}, fmt.Errorf("Lead rule source is unbound")
	}
	if err := ctx.Err(); err != nil {
		return VerifiedOrderRules{}, err
	}
	exchange, err := s.Reader.GetLeadExchangeInfoContext(ctx)
	if err != nil || exchange == nil {
		return VerifiedOrderRules{}, fmt.Errorf("Lead exchange rules unavailable")
	}
	var match *futures.Symbol
	symbol := strings.ToUpper(strings.TrimSpace(order.Symbol))
	for i := range exchange.Symbols {
		if exchange.Symbols[i].Symbol == symbol {
			match = &exchange.Symbols[i]
			break
		}
	}
	if match == nil {
		return VerifiedOrderRules{}, fmt.Errorf("Lead trading symbol not found")
	}
	positions, err := s.Reader.GetPositionFreshContext(ctx, binance.PositionParams{Symbol: symbol})
	if err != nil {
		return VerifiedOrderRules{}, fmt.Errorf("Lead symbol position mode unavailable")
	}
	brackets, err := s.Reader.GetLeadLeverageBracketContext(ctx, symbol)
	if err != nil {
		return VerifiedOrderRules{}, fmt.Errorf("Lead symbol leverage tier unavailable")
	}
	if err := ctx.Err(); err != nil {
		return VerifiedOrderRules{}, err
	}
	return ValidateLeadOrderRules(order, *match, positions, brackets, margin)
}
