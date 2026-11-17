package feature

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/adshao/go-binance/v2/futures"
	binance "go_binance_futures/feature/api/binance"
	"go_binance_futures/models"
	"go_binance_futures/service/futuresownership"
	"go_binance_futures/types"
)

// Lead-03 production data boundary. This adapter reads ONLY from the Lead
// account. It is not a scheduler, broker, preview endpoint or order submitter.
type leadTradeIO struct {
	account   *binance.AccountClient
	ownership futuresownership.Service
}

var errLeadTradeIOUnverified = errors.New("lead trading account or ownership snapshot is not verified")

func newLeadTradeIO(account *binance.AccountClient) (*leadTradeIO, error) {
	if account == nil || account.ID() != binance.LeadAccountID {
		return nil, errLeadTradeIOUnverified
	}
	owned, err := futuresownership.BindAccount(binance.LeadAccountID)
	if err != nil || owned.AccountID != string(binance.LeadAccountID) {
		return nil, errLeadTradeIOUnverified
	}
	return &leadTradeIO{account: account, ownership: owned}, nil
}

func (s *leadTradeIO) valid() bool {
	return s != nil && s.account != nil && s.account.ID() == binance.LeadAccountID &&
		s.ownership.AccountID == string(binance.LeadAccountID)
}

// Wire verified Lead reads into the shared strategy runner, without granting
// any production execution permit or introducing a second scheduler.
func (s *leadTradeIO) bindReadHooks(r *accountTradeRunner) error {
	if !s.valid() || r == nil || r.AccountID != binance.LeadAccountID {
		return errLeadTradeIOUnverified
	}
	r.ReadPositions = s.readPositions
	r.ReadOpenOrders = s.readOrders
	r.SyncPositions = func(live []types.FuturesPosition) ([]ownedTradePosition, error) {
		return s.managedPositions(context.Background(), live)
	}
	return nil
}

func (s *leadTradeIO) readPositions(ctx context.Context) ([]types.FuturesPosition, error) {
	if !s.valid() {
		return nil, errLeadTradeIOUnverified
	}
	rows, err := s.account.GetPositionFreshContext(ctx, binance.PositionParams{})
	if err != nil {
		return nil, fmt.Errorf("lead position fetch failed: %w", err)
	}
	return convertLeadAccountPositions(rows)
}

func convertLeadAccountPositions(rows []*futures.PositionRisk) ([]types.FuturesPosition, error) {
	result := make([]types.FuturesPosition, 0, len(rows))
	seen := make(map[string]bool)
	for _, row := range rows {
		if row == nil || strings.TrimSpace(row.Symbol) == "" {
			return nil, errLeadTradeIOUnverified
		}
		qty, err := strconv.ParseFloat(row.PositionAmt, 64)
		if err != nil || math.IsNaN(qty) || math.IsInf(qty, 0) {
			return nil, errLeadTradeIOUnverified
		}
		if math.Abs(qty) < 1e-12 {
			continue
		}
		side := strings.ToUpper(strings.TrimSpace(row.PositionSide))
		if (side != "LONG" && side != "SHORT") ||
			(side == "LONG" && qty <= 0) || (side == "SHORT" && qty >= 0) {
			return nil, errLeadTradeIOUnverified
		}
		key := strings.ToUpper(row.Symbol) + "|" + side
		if seen[key] {
			return nil, errLeadTradeIOUnverified
		}
		seen[key] = true
		leverage, err := strconv.ParseInt(row.Leverage, 10, 64)
		if err != nil || leverage < 1 {
			return nil, errLeadTradeIOUnverified
		}
		for index, number := range []string{row.EntryPrice, row.MarkPrice, row.UnRealizedProfit} {
			value, parseErr := strconv.ParseFloat(number, 64)
			if parseErr != nil || math.IsNaN(value) || math.IsInf(value, 0) ||
				(index < 2 && value <= 0) {
				return nil, errLeadTradeIOUnverified
			}
		}
		result = append(result, types.FuturesPosition{
			Symbol: strings.ToUpper(row.Symbol), Side: side,
			Amount: row.PositionAmt, Leverage: leverage,
			MarginType: row.MarginType, IsolatedWallet: row.IsolatedWallet,
			EntryPrice: row.EntryPrice, MarkPrice: row.MarkPrice,
			UnrealizedProfit: row.UnRealizedProfit, SourceType: "api",
		})
	}
	return result, nil
}

func (s *leadTradeIO) readOrders(ctx context.Context) ([]types.FuturesOrder, error) {
	if !s.valid() {
		return nil, errLeadTradeIOUnverified
	}
	rows, err := s.account.GetOpenOrderFreshContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("lead open order fetch failed: %w", err)
	}
	return convertLeadAccountOrders(rows)
}

func convertLeadAccountOrders(rows []*futures.Order) ([]types.FuturesOrder, error) {
	result := make([]types.FuturesOrder, 0, len(rows))
	seen := make(map[string]bool)
	for _, row := range rows {
		if row == nil || row.Symbol == "" || strings.TrimSpace(row.ClientOrderID) == "" || row.OrderID <= 0 ||
			(row.PositionSide != futures.PositionSideTypeLong && row.PositionSide != futures.PositionSideTypeShort) ||
			(row.Side != futures.SideTypeBuy && row.Side != futures.SideTypeSell) ||
			(row.Status != futures.OrderStatusTypeNew && row.Status != futures.OrderStatusTypePartiallyFilled) {
			return nil, errLeadTradeIOUnverified
		}
		original, qtyErr := strconv.ParseFloat(row.OrigQuantity, 64)
		executed, filledErr := strconv.ParseFloat(row.ExecutedQuantity, 64)
		if qtyErr != nil || filledErr != nil || math.IsNaN(original) || math.IsInf(original, 0) ||
			math.IsNaN(executed) || math.IsInf(executed, 0) || original <= 0 ||
			executed < 0 || executed > original+1e-10 {
			return nil, errLeadTradeIOUnverified
		}
		key := fmt.Sprintf("%s|%d", row.Symbol, row.OrderID)
		if seen[key] {
			return nil, errLeadTradeIOUnverified
		}
		seen[key] = true
		result = append(result, types.FuturesOrder{
			Symbol: row.Symbol, ClientOrderId: row.ClientOrderID,
			OrderId: row.OrderID, Side: string(row.Side),
			PositionSide: string(row.PositionSide), Type: string(row.Type),
			Status: string(row.Status), Price: row.Price,
			OrigQty: row.OrigQuantity, ExecutedQty: row.ExecutedQuantity,
			UpdateTime: row.UpdateTime,
		})
	}
	return result, nil
}

func (s *leadTradeIO) managedPositions(ctx context.Context, positions []types.FuturesPosition) ([]ownedTradePosition, error) {
	if !s.valid() {
		return nil, errLeadTradeIOUnverified
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	pending, err := s.ownership.OutstandingOrders(ctx)
	if err != nil {
		return nil, err
	}
	managed, err := s.ownership.ActivePositions(ctx, futuresownership.OwnerAutoStrategy)
	if err != nil {
		return nil, err
	}
	return verifyLeadManagedSnapshot(positions, managed, pending)
}

// managedPositionsBySymbol leaves normal account-level decisions independent:
// an unresolved BTC order cannot hide a verified ETH managed position, while
// BTC remains unavailable for both a duplicate open and an unsafe close.
func (s *leadTradeIO) managedPositionsBySymbol(ctx context.Context, live []types.FuturesPosition) ([]ownedTradePosition, map[string]bool, error) {
	if !s.valid() {
		return nil, nil, errLeadTradeIOUnverified
	}
	if err := ctx.Err(); err != nil {
		return nil, nil, err
	}
	pending, err := s.ownership.OutstandingOrders(ctx)
	if err != nil {
		return nil, nil, err
	}
	managed, err := s.ownership.ActivePositions(ctx, futuresownership.OwnerAutoStrategy)
	if err != nil {
		return nil, nil, err
	}
	return verifyLeadManagedBySymbol(live, managed, pending)
}

func verifyLeadManagedBySymbol(live []types.FuturesPosition, managed []models.FuturesManagedPosition, pending []models.FuturesManagedOrder) ([]ownedTradePosition, map[string]bool, error) {
	blocked := make(map[string]bool, len(pending))
	for _, order := range pending {
		if order.AccountID != string(binance.LeadAccountID) || order.Symbol == "" ||
			order.Symbol != strings.ToUpper(order.Symbol) ||
			(order.PositionSide != "LONG" && order.PositionSide != "SHORT") ||
			strings.TrimSpace(order.ClientOrderID) == "" {
			return nil, nil, errLeadTradeIOUnverified
		}
		blocked[order.Symbol] = true
	}
	safe := make([]models.FuturesManagedPosition, 0, len(managed))
	identities := make(map[string]bool, len(managed))
	for _, row := range managed {
		key := row.Symbol + "|" + row.PositionSide
		if row.AccountID != string(binance.LeadAccountID) ||
			row.Owner != futuresownership.OwnerAutoStrategy ||
			(row.PositionSide != "LONG" && row.PositionSide != "SHORT") ||
			row.Symbol == "" || row.Symbol != strings.ToUpper(row.Symbol) ||
			identities[key] {
			return nil, nil, errLeadTradeIOUnverified
		}
		identities[key] = true
		if !blocked[row.Symbol] {
			safe = append(safe, row)
		}
	}
	verified, err := verifyLeadManagedSnapshot(live, safe, nil)
	if err != nil {
		return nil, nil, err
	}
	return verified, blocked, nil
}

// No inferred Fill ownership and no silent position reduction. Any unresolved
// order or partial/manual reduction blocks this cycle until reconciliation.
// This protects against an accidental second open on a "missing" managed leg.
func verifyLeadManagedSnapshot(live []types.FuturesPosition, managed []models.FuturesManagedPosition, pending []models.FuturesManagedOrder) ([]ownedTradePosition, error) {
	if len(pending) != 0 {
		return nil, errLeadTradeIOUnverified
	}
	index := make(map[string]types.FuturesPosition)
	for _, row := range live {
		qty, err := strconv.ParseFloat(row.Amount, 64)
		if err != nil || math.IsNaN(qty) || math.IsInf(qty, 0) ||
			(row.Side != "LONG" && row.Side != "SHORT") {
			return nil, errLeadTradeIOUnverified
		}
		key := strings.ToUpper(row.Symbol) + "|" + row.Side
		if _, exists := index[key]; exists {
			return nil, errLeadTradeIOUnverified
		}
		index[key] = row
	}
	result := make([]ownedTradePosition, 0, len(managed))
	managedKeys := make(map[string]bool, len(managed))
	for _, row := range managed {
		if row.AccountID != string(binance.LeadAccountID) ||
			row.Owner != futuresownership.OwnerAutoStrategy ||
			row.Status != futuresownership.PositionActive ||
			row.ManagedQty <= 0 || math.IsNaN(row.ManagedQty) || math.IsInf(row.ManagedQty, 0) ||
			row.EntryPrice <= 0 || math.IsNaN(row.EntryPrice) || math.IsInf(row.EntryPrice, 0) {
			return nil, errLeadTradeIOUnverified
		}
		key := strings.ToUpper(row.Symbol) + "|" + row.PositionSide
		if managedKeys[key] {
			return nil, errLeadTradeIOUnverified
		}
		managedKeys[key] = true
		actual, ok := index[key]
		if !ok {
			return nil, errLeadTradeIOUnverified
		}
		qty, _ := strconv.ParseFloat(actual.Amount, 64)
		if math.Abs(qty)+1e-10 < row.ManagedQty {
			return nil, errLeadTradeIOUnverified
		}
		// managedAccountPosition caps the strategy's view to the actually
		// Fill-owned amount, never the user's manually added inventory.
		result = append(result, ownedTradePosition{
			Position:  managedAccountPosition(actual, row),
			Owner:     futuresownership.OwnerAutoStrategy,
			SourceRef: row.SourceRef,
		})
	}
	return result, nil
}
