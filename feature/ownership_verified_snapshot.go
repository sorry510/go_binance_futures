package feature

import (
	"context"
	"fmt"
	"math"
	"strconv"

	"go_binance_futures/feature/api/binance"
	"go_binance_futures/models"
	"go_binance_futures/service/binanceapiusage"
	"go_binance_futures/types"

	"github.com/beego/beego/v2/core/logs"
)

// A verified snapshot carries the actual rows, not just quantities: exits
// must not silently miss an owned position absent from an older WS mirror.
type ownershipVerifiedSnapshot struct {
	quantities map[string]float64
	positions  []types.FuturesPosition
	fresh      bool
}

func verifiedOwnershipAccountSnapshot(local []types.FuturesPosition, managed []models.FuturesManagedPosition, force bool) (ownershipVerifiedSnapshot, error) {
	return resolveOwnershipAccountSnapshot(local, managed, force, freshOwnershipAccountPositions)
}

func resolveOwnershipAccountSnapshot(local []types.FuturesPosition, managed []models.FuturesManagedPosition, force bool, freshRead func() ([]types.FuturesPosition, error)) (ownershipVerifiedSnapshot, error) {
	quantities, localErr := ownershipAccountQuantities(local)
	if localErr == nil && !needsFreshOwnershipSnapshot(managed, quantities, force) {
		return ownershipVerifiedSnapshot{quantities: quantities, positions: local}, nil
	}
	// An invalid cached row is not proof of zero; recover from the exchange,
	// preserving all other positions' exit evaluation when REST succeeds.
	if localErr != nil {
		logs.Warning("ownership account mirror has malformed quantity; using fresh Binance positions: ", localErr)
	}
	fresh, err := freshRead()
	if err != nil {
		return ownershipVerifiedSnapshot{}, fmt.Errorf("cannot verify ownership with fresh account positions: %w", err)
	}
	quantities, err = ownershipAccountQuantities(fresh)
	if err != nil {
		return ownershipVerifiedSnapshot{}, fmt.Errorf("invalid fresh account positions: %w", err)
	}
	return ownershipVerifiedSnapshot{quantities: quantities, positions: fresh, fresh: true}, nil
}

func freshOwnershipAccountPositions() ([]types.FuturesPosition, error) {
	return freshOwnershipAccountPositionsContext(context.Background())
}

func freshOwnershipAccountPositionsContext(ctx context.Context) ([]types.FuturesPosition, error) {
	ctx = binanceapiusage.WithSource(ctx, "start_trade")
	rows, err := binance.GetPositionFreshContext(ctx, binance.PositionParams{})
	if err != nil {
		return nil, err
	}
	out := make([]types.FuturesPosition, 0, len(rows))
	for _, row := range rows {
		if row == nil {
			return nil, fmt.Errorf("nil position in fresh Binance account snapshot")
		}
		qty, err := strconv.ParseFloat(row.PositionAmt, 64)
		if err != nil || math.IsNaN(qty) || math.IsInf(qty, 0) {
			return nil, fmt.Errorf("invalid Binance quantity for %s %s: %q", row.Symbol, row.PositionSide, row.PositionAmt)
		}
		if math.Abs(qty) <= 1e-12 {
			continue
		}
		mark, err := strconv.ParseFloat(row.MarkPrice, 64)
		if err != nil || mark <= 0 || math.IsNaN(mark) || math.IsInf(mark, 0) {
			return nil, fmt.Errorf("invalid Binance mark price for %s %s: %q", row.Symbol, row.PositionSide, row.MarkPrice)
		}
		leverage, err := strconv.ParseInt(row.Leverage, 10, 64)
		if err != nil || leverage <= 0 {
			return nil, fmt.Errorf("invalid Binance leverage for %s %s: %q", row.Symbol, row.PositionSide, row.Leverage)
		}
		if row.Symbol == "" || (row.PositionSide != "LONG" && row.PositionSide != "SHORT") {
			return nil, fmt.Errorf("invalid Binance hedge position identity: symbol=%q side=%q", row.Symbol, row.PositionSide)
		}
		out = append(out, types.FuturesPosition{
			Symbol: row.Symbol, Side: row.PositionSide, Amount: row.PositionAmt,
			MarginType: row.MarginType, Leverage: leverage, IsolatedWallet: row.IsolatedWallet,
			EntryPrice: row.EntryPrice, MarkPrice: row.MarkPrice, UnrealizedProfit: row.UnRealizedProfit,
			SourceType: "api",
		})
	}
	return out, nil
}
