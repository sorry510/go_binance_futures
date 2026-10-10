package binance

import (
	"context"
	"fmt"
	"strings"

	"github.com/adshao/go-binance/v2/futures"
)

// These GET calls are bound to the concrete Lead AccountClient. They do not
// change margin/leverage, place orders, or fall back to Main's SDK client.
func (a *AccountClient) GetLeadExchangeInfoContext(ctx context.Context) (*futures.ExchangeInfo, error) {
	if a == nil || a.ID() != LeadAccountID {
		return nil, fmt.Errorf("Lead exchange rules require Lead account")
	}
	a.signedMu.Lock()
	defer a.signedMu.Unlock()
	return a.client.NewExchangeInfoService().Do(ctx)
}
func (a *AccountClient) GetLeadLeverageBracketContext(ctx context.Context, symbol string) ([]*futures.LeverageBracket, error) {
	if a == nil || a.ID() != LeadAccountID || strings.TrimSpace(symbol) == "" {
		return nil, fmt.Errorf("Lead leverage brackets require Lead account and symbol")
	}
	return doAccountSigned(a, ctx, futuresSignedReadRecvWindow, func(c *futures.Client, ctx context.Context, opts ...futures.RequestOption) ([]*futures.LeverageBracket, error) {
		return c.NewGetLeverageBracketService().Symbol(symbol).Do(ctx, opts...)
	})
}
