package binance

import (
	"context"
	"fmt"
	"strings"

	"github.com/adshao/go-binance/v2/futures"
)

// Lead stream ownership belongs to the bound Lead client, not the legacy
// Main user data loop. These calls never happen during normal startup.
func (a *AccountClient) CloseLeadListenKey(ctx context.Context, key string) error {
	if a == nil || a.ID() != LeadAccountID || strings.TrimSpace(key) == "" {
		return fmt.Errorf("Lead listenKey cleanup requires the bound Lead account")
	}
	_, err := doAccountSigned(a, ctx, futuresSignedReadRecvWindow, func(c *futures.Client, ctx context.Context, opts ...futures.RequestOption) (struct{}, error) {
		return struct{}{}, c.NewCloseUserStreamService().ListenKey(key).Do(ctx, opts...)
	})
	return err
}

// DialLeadUserStream deliberately reuses the application's existing proxy-aware
// WS transport. It is an opt-in operation; no goroutine is created at import.
func (a *AccountClient) DialLeadUserStream(key string, handler futures.WsUserDataHandler, onError futures.ErrHandler) (chan struct{}, chan struct{}, error) {
	if a == nil || a.ID() != LeadAccountID || key == "" {
		return nil, nil, fmt.Errorf("Lead WS requires bound credentials and listenKey")
	}
	return wsFuturesUserDataServe(key, handler, onError)
}
