package feature

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"go_binance_futures/feature/api/binance"
	"go_binance_futures/models"
	futuresownership "go_binance_futures/service/futuresownership"
	"go_binance_futures/types"
	"go_binance_futures/utils"

	"github.com/adshao/go-binance/v2/futures"
	"github.com/beego/beego/v2/client/orm"
	"github.com/beego/beego/v2/core/logs"
)

var sharedOwnership = futuresownership.DefaultService()
var sharedOwnershipExecutor = futuresownership.DefaultExecutor()

type ownedTradePosition struct {
	Position  types.FuturesPosition
	Owner     string
	SourceRef string
}

func syncStrategyExitPositions(accountPositions []types.FuturesPosition) ([]ownedTradePosition, error) {
	autoPositions, err := syncAutoStrategyOwnership(accountPositions)
	if err != nil {
		return nil, err
	}
	result := make([]ownedTradePosition, 0, len(autoPositions))
	for _, position := range autoPositions {
		result = append(result, ownedTradePosition{Position: position, Owner: futuresownership.OwnerAutoStrategy, SourceRef: "auto_strategy:" + strings.ToUpper(position.Symbol)})
	}

	accountByKey := make(map[string]types.FuturesPosition, len(accountPositions))
	for _, position := range accountPositions {
		accountByKey[managedPositionKey(position.Symbol, position.Side)] = position
	}
	accountQty, err := ownershipAccountQuantities(accountPositions, false)
	if err != nil {
		return nil, err
	}
	for _, owner := range []string{futuresownership.OwnerNewCoinRush, futuresownership.OwnerFundingRate} {
		managed, err := sharedOwnership.ActivePositions(context.Background(), owner)
		if err != nil {
			return nil, err
		}
		for _, position := range managed {
			if owner == futuresownership.OwnerFundingRate && !strings.HasPrefix(position.SourceRef, "funding_rate:") {
				continue
			}
			key := managedPositionKey(position.Symbol, position.PositionSide)
			if _, err := sharedOwnership.ReconcilePosition(context.Background(), owner, position.Symbol, position.PositionSide, accountQty[key]); err != nil && err != orm.ErrNoRows {
				return nil, err
			}
		}
		managed, err = sharedOwnership.ActivePositions(context.Background(), owner)
		if err != nil {
			return nil, err
		}
		for _, position := range managed {
			if owner == futuresownership.OwnerFundingRate && !strings.HasPrefix(position.SourceRef, "funding_rate:") {
				continue
			}
			account, ok := accountByKey[managedPositionKey(position.Symbol, position.PositionSide)]
			if !ok || position.ManagedQty <= 1e-12 {
				continue
			}
			result = append(result, ownedTradePosition{Position: managedAccountPosition(account, position), Owner: owner, SourceRef: position.SourceRef})
		}
	}
	return result, nil
}

func syncAutoStrategyOwnership(accountPositions []types.FuturesPosition) ([]types.FuturesPosition, error) {
	ctx := context.Background()
	managedOrders, err := sharedOwnership.ActiveOrders(ctx, futuresownership.OwnerAutoStrategy)
	if err != nil {
		return nil, err
	}
	for _, order := range managedOrders {
		if _, err := sharedOwnershipExecutor.Reconcile(ctx, order.Symbol, order.ClientOrderID); err != nil {
			logs.Warning("reconcile auto_strategy managed order:", order.ClientOrderID, err)
		}
	}

	managedPositions, err := sharedOwnership.ActivePositions(ctx, futuresownership.OwnerAutoStrategy)
	if err != nil {
		return nil, err
	}
	// accountPositions is already the current StartTrade snapshot: it comes from
	// the User Data WS/local tables when enabled, or directly from Binance REST
	// otherwise. Only refresh from Binance after reconciling active managed orders,
	// because a just-confirmed fill can make the pre-reconcile snapshot stale.
	accountQtyByKey, err := ownershipAccountQuantities(accountPositions, shouldRefreshOwnershipAccountQuantities(len(managedOrders)))
	if err != nil {
		return nil, err
	}
	for _, managed := range managedPositions {
		accountQty := accountQtyByKey[managedPositionKey(managed.Symbol, managed.PositionSide)]
		if _, err := sharedOwnership.ReconcilePosition(ctx, managed.Owner, managed.Symbol, managed.PositionSide, accountQty); err != nil && err != orm.ErrNoRows {
			return nil, err
		}
	}

	managedPositions, err = sharedOwnership.ActivePositions(ctx, futuresownership.OwnerAutoStrategy)
	if err != nil {
		return nil, err
	}
	managedByKey := make(map[string]models.FuturesManagedPosition, len(managedPositions))
	for _, managed := range managedPositions {
		managedByKey[managedPositionKey(managed.Symbol, managed.PositionSide)] = managed
	}

	result := make([]types.FuturesPosition, 0, len(managedPositions))
	for _, account := range accountPositions {
		managed, ok := managedByKey[managedPositionKey(account.Symbol, account.Side)]
		if !ok || managed.ManagedQty <= 0 {
			continue
		}
		result = append(result, managedAccountPosition(account, managed))
	}
	return result, nil
}

func shouldRefreshOwnershipAccountQuantities(activeManagedOrders int) bool {
	return activeManagedOrders > 0
}

func managedAccountPosition(account types.FuturesPosition, managed models.FuturesManagedPosition) types.FuturesPosition {
	position := account
	qty := managed.ManagedQty
	if strings.EqualFold(managed.PositionSide, "SHORT") {
		qty = -qty
	}
	position.Amount = strconv.FormatFloat(qty, 'f', -1, 64)
	if managed.EntryPrice > 0 {
		position.EntryPrice = strconv.FormatFloat(managed.EntryPrice, 'f', -1, 64)
		mark, _ := strconv.ParseFloat(position.MarkPrice, 64)
		position.UnrealizedProfit = strconv.FormatFloat((mark-managed.EntryPrice)*qty, 'f', -1, 64)
	}
	return position
}

func managedPositionKey(symbol, side string) string {
	return strings.ToUpper(strings.TrimSpace(symbol)) + "|" + strings.ToUpper(strings.TrimSpace(side))
}

func accountTradeRiskCounts(positions []types.FuturesPosition) (positionCount, lossCount int) {
	for _, position := range positions {
		qty, _ := strconv.ParseFloat(position.Amount, 64)
		qty = math.Abs(qty)
		if qty < 1e-12 {
			continue
		}
		positionCount++
		unrealized, _ := strconv.ParseFloat(position.UnrealizedProfit, 64)
		mark, _ := strconv.ParseFloat(position.MarkPrice, 64)
		if utils.FuturesLeveragedROI(unrealized, qty, mark, position.Leverage) < -0.1 {
			lossCount++
		}
	}
	return positionCount, lossCount
}

func cancelTimeoutAutoStrategyOrders(timeoutSec int64) error {
	ctx := context.Background()
	orders, err := sharedOwnership.ActiveOrders(ctx, futuresownership.OwnerAutoStrategy)
	if err != nil {
		return err
	}
	now := time.Now().UnixMilli()
	for _, order := range orders {
		if order.Intent != futuresownership.IntentOpen || now < order.CreatedAt+timeoutSec*1000 {
			continue
		}
		if strings.TrimSpace(order.ExchangeOrderID) == "" {
			if _, err := sharedOwnershipExecutor.Reconcile(ctx, order.Symbol, order.ClientOrderID); err != nil {
				logs.Warning("skip cancel for unresolved managed order:", order.ClientOrderID, err)
				continue
			}
			refreshed, loadErr := findManagedOrder(order.ClientOrderID)
			if loadErr != nil {
				continue
			}
			order = refreshed
		}
		if err := sharedOwnershipExecutor.Cancel(ctx, futuresownership.OwnerAutoStrategy, order); err != nil {
			logs.Error("cancel auto_strategy managed order:", order.ClientOrderID, err)
			continue
		}
		if order.ExchangeOrderID != "" {
			_, _ = orm.NewOrm().Raw("DELETE FROM `order` WHERE order_id = ?", order.ExchangeOrderID).Exec()
		}
	}
	return nil
}

func findManagedOrder(clientOrderID string) (models.FuturesManagedOrder, error) {
	var row models.FuturesManagedOrder
	err := orm.NewOrm().QueryTable(new(models.FuturesManagedOrder)).Filter("client_order_id", clientOrderID).One(&row)
	return row, err
}

func ensureAccountOpenSlotAvailable(symbol string, positionSide futures.PositionSideType) error {
	positions, err := GetTransformPositions()
	if err != nil {
		return fmt.Errorf("verify account positions before managed open: %w", err)
	}
	for _, position := range positions {
		qty, _ := strconv.ParseFloat(position.Amount, 64)
		if strings.EqualFold(position.Symbol, symbol) && strings.EqualFold(position.Side, string(positionSide)) && math.Abs(qty) > 1e-12 {
			return fmt.Errorf("%s %s already exists in account; ownership is not safe to merge", strings.ToUpper(symbol), positionSide)
		}
	}
	orders, err := getTransformOpenOrders()
	if err != nil {
		return fmt.Errorf("verify account open orders before managed open: %w", err)
	}
	for _, order := range orders {
		isOpening := (positionSide == futures.PositionSideTypeLong && strings.EqualFold(order.Side, "BUY")) ||
			(positionSide == futures.PositionSideTypeShort && strings.EqualFold(order.Side, "SELL"))
		if isOpening && strings.EqualFold(order.Symbol, symbol) && strings.EqualFold(order.PositionSide, string(positionSide)) {
			return fmt.Errorf("%s %s already has an account open order; ownership is not safe to merge", strings.ToUpper(symbol), positionSide)
		}
	}
	return nil
}

func submitAutoStrategyOpen(symbol string, quantity, price float64, side futures.SideType, positionSide futures.PositionSideType, orderType futures.OrderType) (*futures.CreateOrderResponse, error) {
	return submitOwnedFeatureOpen(futuresownership.OwnerAutoStrategy, "auto_strategy:"+strings.ToUpper(strings.TrimSpace(symbol)), symbol, quantity, price, side, positionSide, orderType)
}

func submitNewCoinRushOpen(sourceRef, symbol string, quantity, price float64, side futures.SideType, positionSide futures.PositionSideType, orderType futures.OrderType) (*futures.CreateOrderResponse, error) {
	return submitOwnedFeatureOpen(futuresownership.OwnerNewCoinRush, sourceRef, symbol, quantity, price, side, positionSide, orderType)
}

func submitNoticeAutoOpen(sourceRef, symbol string, quantity float64, side futures.SideType, positionSide futures.PositionSideType) (*futures.CreateOrderResponse, error) {
	return submitOwnedFeatureOpen(futuresownership.OwnerNoticeAutoOrder, sourceRef, symbol, quantity, 0, side, positionSide, futures.OrderTypeMarket)
}

func RepairNoticeAutoOrderProtections() {
	ctx := context.Background()
	positions, err := sharedOwnership.ActivePositions(ctx, futuresownership.OwnerNoticeAutoOrder)
	if err != nil {
		logs.Error("repair notice protections load positions:", err)
		return
	}
	if len(positions) == 0 {
		return
	}
	orders, err := sharedOwnership.ActiveOrders(ctx, futuresownership.OwnerNoticeAutoOrder)
	if err != nil {
		logs.Error("repair notice protections load orders:", err)
		return
	}
	for _, position := range positions {
		const prefix = "notice_auto_order:"
		if !strings.HasPrefix(position.SourceRef, prefix) {
			continue
		}
		id, parseErr := strconv.ParseInt(strings.TrimPrefix(position.SourceRef, prefix), 10, 64)
		if parseErr != nil {
			logs.Error("repair notice protections invalid source ref:", position.SourceRef, parseErr)
			continue
		}
		var notice models.NoticeSymbols
		if err := orm.NewOrm().QueryTable(new(models.NoticeSymbols)).Filter("id", id).One(&notice); err != nil {
			logs.Error("repair notice protections load source:", position.SourceRef, err)
			continue
		}
		protectionSide := futures.SideTypeSell
		positionSide := futures.PositionSideTypeLong
		if strings.EqualFold(position.PositionSide, "SHORT") {
			protectionSide = futures.SideTypeBuy
			positionSide = futures.PositionSideTypeShort
		}
		for _, spec := range []struct {
			intent string
			price  string
		}{
			{intent: futuresownership.IntentTakeProfit, price: notice.ProfitPrice},
			{intent: futuresownership.IntentStopLoss, price: notice.LossPrice},
		} {
			price, parseErr := strconv.ParseFloat(strings.TrimSpace(spec.price), 64)
			if parseErr != nil || price <= 0 {
				continue
			}
			price = utils.GetTradePrecision(price, notice.TickSize)
			hasExact, unresolved := false, false
			for _, order := range orders {
				if order.SourceRef != position.SourceRef || order.Symbol != position.Symbol || !strings.EqualFold(order.PositionSide, position.PositionSide) || order.Intent != spec.intent {
					continue
				}
				if math.Abs(order.RequestedQty-position.ManagedQty) <= 1e-12 && !hasExact {
					hasExact = true
					continue
				}
				if strings.TrimSpace(order.ExchangeOrderID) == "" {
					unresolved = true
					continue
				}
				if cancelErr := sharedOwnershipExecutor.Cancel(ctx, futuresownership.OwnerNoticeAutoOrder, order); cancelErr != nil {
					logs.Error("repair notice protections cancel stale order:", order.ClientOrderID, cancelErr)
					unresolved = true
				}
			}
			if hasExact || unresolved {
				continue
			}
			if _, err := submitNoticeProtection(position.SourceRef, position.Symbol, position.ManagedQty, price, protectionSide, positionSide, spec.intent); err != nil {
				logs.Error("repair notice protection submit:", position.Symbol, spec.intent, err)
			}
		}
	}
}

func noticeManagedQuantity(symbol string, positionSide futures.PositionSideType) (float64, error) {
	position, err := sharedOwnership.GetPosition(context.Background(), futuresownership.OwnerNoticeAutoOrder, symbol, string(positionSide))
	if err != nil {
		return 0, err
	}
	if position.ManagedQty <= 1e-12 {
		return 0, fmt.Errorf("no managed notice_auto_order quantity for %s %s", strings.ToUpper(symbol), positionSide)
	}
	return position.ManagedQty, nil
}

func submitNoticeProtection(sourceRef, symbol string, quantity, stopPrice float64, side futures.SideType, positionSide futures.PositionSideType, intent string) (*futures.CreateOrderResponse, error) {
	orderType := futures.OrderType("STOP_MARKET")
	if intent == futuresownership.IntentTakeProfit {
		orderType = futures.OrderType("TAKE_PROFIT_MARKET")
	}
	return submitOwnedFeatureOrder(futuresownership.OwnerNoticeAutoOrder, sourceRef, symbol, quantity, 0, stopPrice, side, positionSide, orderType, intent)
}

func submitFundingRateOpen(sourceRef, symbol string, quantity float64, side futures.SideType, positionSide futures.PositionSideType) (*futures.CreateOrderResponse, error) {
	return submitOwnedFeatureOpen(futuresownership.OwnerFundingRate, sourceRef, symbol, quantity, 0, side, positionSide, futures.OrderTypeMarket)
}

func submitOwnedFeatureOpen(owner, sourceRef, symbol string, quantity, price float64, side futures.SideType, positionSide futures.PositionSideType, orderType futures.OrderType) (*futures.CreateOrderResponse, error) {
	if err := ensureAccountOpenSlotAvailable(symbol, positionSide); err != nil {
		return nil, err
	}
	return submitOwnedFeatureOrder(owner, sourceRef, symbol, quantity, price, 0, side, positionSide, orderType, futuresownership.IntentOpen)
}

func submitManagedStrategyClose(owner, sourceRef, symbol string, quantity float64, positionSide futures.PositionSideType) (*futures.CreateOrderResponse, error) {
	accountQty, err := currentAccountPositionQty(symbol, positionSide)
	if err != nil {
		return nil, err
	}
	closeQty, err := sharedOwnership.CloseQuantity(context.Background(), owner, symbol, string(positionSide), accountQty)
	if err != nil {
		return nil, err
	}
	closeQty = math.Min(closeQty, math.Abs(quantity))
	if closeQty <= 1e-12 {
		return nil, fmt.Errorf("no managed %s %s quantity is available to close for owner %s", strings.ToUpper(symbol), positionSide, owner)
	}
	side := futures.SideTypeSell
	if positionSide == futures.PositionSideTypeShort {
		side = futures.SideTypeBuy
	}
	return submitOwnedFeatureOrder(owner, sourceRef, symbol, closeQty, 0, 0, side, positionSide, futures.OrderTypeMarket, futuresownership.IntentClose)
}

func submitAutoStrategyClose(symbol string, quantity float64, positionSide futures.PositionSideType) (*futures.CreateOrderResponse, error) {
	return submitManagedStrategyClose(futuresownership.OwnerAutoStrategy, "auto_strategy:"+strings.ToUpper(strings.TrimSpace(symbol)), symbol, quantity, positionSide)
}

func currentAccountPositionQty(symbol string, positionSide futures.PositionSideType) (float64, error) {
	positions, err := binance.GetPosition(binance.PositionParams{Symbol: strings.ToUpper(strings.TrimSpace(symbol))})
	if err != nil {
		return 0, fmt.Errorf("verify Binance account position before managed close: %w", err)
	}
	for _, position := range positions {
		if strings.EqualFold(position.Symbol, symbol) && strings.EqualFold(string(position.PositionSide), string(positionSide)) {
			qty, parseErr := strconv.ParseFloat(position.PositionAmt, 64)
			if parseErr != nil {
				return 0, fmt.Errorf("parse account quantity for %s %s: %w", symbol, positionSide, parseErr)
			}
			return math.Abs(qty), nil
		}
	}
	return 0, nil
}

func ownershipAccountQuantities(accountPositions []types.FuturesPosition, refreshFromBinance bool) (map[string]float64, error) {
	out := make(map[string]float64)
	if !refreshFromBinance {
		for _, position := range accountPositions {
			qty, err := strconv.ParseFloat(position.Amount, 64)
			if err != nil {
				continue
			}
			out[managedPositionKey(position.Symbol, position.Side)] = math.Abs(qty)
		}
		return out, nil
	}
	rows, err := binance.GetPosition(binance.PositionParams{})
	if err != nil {
		return nil, fmt.Errorf("refresh Binance positions for ownership reconcile: %w", err)
	}
	for _, position := range rows {
		qty, err := strconv.ParseFloat(position.PositionAmt, 64)
		if err != nil {
			continue
		}
		out[managedPositionKey(position.Symbol, string(position.PositionSide))] = math.Abs(qty)
	}
	return out, nil
}

func submitAutoStrategyOrder(symbol string, quantity, price float64, side futures.SideType, positionSide futures.PositionSideType, orderType futures.OrderType, intent string) (*futures.CreateOrderResponse, error) {
	return submitOwnedFeatureOrder(futuresownership.OwnerAutoStrategy, "auto_strategy:"+strings.ToUpper(strings.TrimSpace(symbol)), symbol, quantity, price, 0, side, positionSide, orderType, intent)
}

func submitOwnedFeatureOrder(owner, sourceRef, symbol string, quantity, price, stopPrice float64, side futures.SideType, positionSide futures.PositionSideType, orderType futures.OrderType, intent string) (*futures.CreateOrderResponse, error) {
	result, err := sharedOwnershipExecutor.Execute(context.Background(), futuresownership.OrderRequest{
		Owner: owner, Symbol: symbol, PositionSide: string(positionSide),
		Intent: intent, Side: string(side), OrderType: string(orderType), Quantity: quantity, Price: price, StopPrice: stopPrice,
		SourceRef: sourceRef,
	})
	if err != nil {
		return nil, err
	}
	orderID, err := strconv.ParseInt(result.ExchangeOrderID, 10, 64)
	if err != nil || orderID <= 0 {
		return nil, fmt.Errorf("managed order has invalid exchange order id %q", result.ExchangeOrderID)
	}
	return &futures.CreateOrderResponse{
		OrderID: orderID, ClientOrderID: result.ClientOrderID, Status: futures.OrderStatusType(result.Status),
		ExecutedQuantity: strconv.FormatFloat(result.FilledQty, 'f', -1, 64), AvgPrice: strconv.FormatFloat(result.AveragePrice, 'f', -1, 64),
	}, nil
}
