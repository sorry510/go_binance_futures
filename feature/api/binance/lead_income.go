package binance

import (
	"context"
	"fmt"

	"github.com/adshao/go-binance/v2/futures"
)

// GetIncomeHistoryContext reads ONLY the explicitly-bound Futures account.
// It never uses the legacy Main client or a shared private-account cache.
// The caller must paginate and verify complete same-millisecond boundaries.
func (a *AccountClient) GetIncomeHistoryContext(ctx context.Context, startMs, endMs int64, limit int64) ([]*futures.IncomeHistory, error) {
	if a == nil || a.ID() != LeadAccountID {
		return nil, fmt.Errorf("lead income read requires explicit Lead account")
	}
	if startMs <= 0 || endMs < startMs || limit < 1 || limit > 1000 {
		return nil, fmt.Errorf("invalid Lead income history request")
	}
	return doAccountSigned(a, ctx, futuresSignedReadRecvWindow, func(c *futures.Client, ctx context.Context, o ...futures.RequestOption) ([]*futures.IncomeHistory, error) {
		return c.NewGetIncomeHistoryService().StartTime(startMs).EndTime(endMs).Limit(limit).Do(ctx, o...)
	})
}
