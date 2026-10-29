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
	"go_binance_futures/service/binanceapiusage"
	futuresownership "go_binance_futures/service/futuresownership"
	"go_binance_futures/types"
	"go_binance_futures/utils"

	"github.com/adshao/go-binance/v2/futures"
	"github.com/beego/beego/v2/client/orm"
	"github.com/beego/beego/v2/core/logs"
)

var sharedOwnership = futuresownership.DefaultService()
var sharedOwnershipExecutor = futuresownership.DefaultExecutor()

const autoStrategyOrderRESTReconcileInterval = 5 * time.Second

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
	result = append(result, autoPositions...)

	for _, owner := range []string{futuresownership.OwnerNewCoinRush, futuresownership.OwnerFundingRate} {
		managed, err := sharedOwnership.ActivePositions(context.Background(), owner)
		if err != nil {
			return nil, err
		}
		verified, err := verifiedOwnershipAccountSnapshot(accountPositions, managed, false)
		if err != nil {
			return nil, err
		}
		accountByKey := make(map[string]types.FuturesPosition, len(verified.positions))
		for _, account := range verified.positions {
			accountByKey[managedPositionKey(account.Symbol, account.Side)] = account
		}
		for _, position := range managed {
			if owner == futuresownership.OwnerFundingRate && !strings.HasPrefix(position.SourceRef, "funding_rate:") {
				continue
			}
			key := managedPositionKey(position.Symbol, position.PositionSide)
			updated, err := sharedOwnership.ReconcileObservedPosition(context.Background(), owner, position.Symbol, position.PositionSide, verified.quantities[key])
			if err != nil && err != orm.ErrNoRows {
				return nil, err
			}
			if err == nil && updated.Status != position.Status {
				logs.Warning("managed ownership verification: owner=", owner, " symbol=", position.Symbol,
					" side=", position.PositionSide, " status=", updated.Status,
					" observed_qty=", verified.quantities[key], " previous_managed_qty=", position.ManagedQty,
					" fresh_snapshot=", verified.fresh)
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

func syncAutoStrategyOwnership(accountPositions []types.FuturesPosition) ([]ownedTradePosition, error) {
	ctx := binanceapiusage.WithSource(context.Background(), "start_trade")
	managedOrders, err := sharedOwnership.ActiveOrders(ctx, futuresownership.OwnerAutoStrategy)
	if err != nil {
		return nil, err
	}
	positionRefreshRequired := false
	for _, order := range managedOrders {
		_, fillAdvanced, err := reconcileAutoStrategyManagedOrder(ctx, order)
		if err != nil {
			logs.Warning("reconcile auto_strategy managed order:", order.ClientOrderID, err)
			continue
		}
		if fillAdvanced {
			positionRefreshRequired = true
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
	verified, err := verifiedOwnershipAccountSnapshot(accountPositions, managedPositions, shouldRefreshOwnershipAccountQuantities(positionRefreshRequired))
	if err != nil {
		return nil, err
	}
	for _, managed := range managedPositions {
		accountQty := verified.quantities[managedPositionKey(managed.Symbol, managed.PositionSide)]
		if accountQty <= 1e-12 {
			// Do not expire ownership while a partially filled/uncertain opening
			// order may still increase the account leg after this snapshot.
			if autoStrategyOpenUnresolved(ctx, managed, managedOrders, sharedOwnership.GetOrder) {
				continue
			}
		}
		updated, err := sharedOwnership.ReconcileObservedPosition(ctx, managed.Owner, managed.Symbol, managed.PositionSide, accountQty)
		if err != nil && err != orm.ErrNoRows {
			return nil, err
		}
		if updated.Status != managed.Status && (updated.Status == futuresownership.PositionReconcileRequired || updated.Status == futuresownership.PositionClosed) {
			logs.Warning("auto_strategy ownership verification: ", managed.Symbol, " ", managed.PositionSide,
				" status=", updated.Status, " observed_qty=", accountQty,
				" previous_managed_qty=", managed.ManagedQty, " fresh_snapshot=", verified.fresh)
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

	result := make([]ownedTradePosition, 0, len(managedPositions))
	for _, account := range verified.positions {
		managed, ok := managedByKey[managedPositionKey(account.Symbol, account.Side)]
		if !ok || managed.ManagedQty <= 0 {
			continue
		}
		result = append(result, ownedTradePosition{Position: managedAccountPosition(account, managed), Owner: futuresownership.OwnerAutoStrategy, SourceRef: managed.SourceRef})
	}
	return result, nil
}

// If the local DB cannot read an opening order, the result is unknown, not
// "confirmed closed". Preserve just this leg and allow exits on other legs.
func autoStrategyOpenUnresolved(ctx context.Context, managed models.FuturesManagedPosition, orders []models.FuturesManagedOrder,
	load func(context.Context, string) (models.FuturesManagedOrder, error)) bool {
	for _, order := range orders {
		if order.Intent != futuresownership.IntentOpen || order.Symbol != managed.Symbol || order.PositionSide != managed.PositionSide {
			continue
		}
		latest, err := load(ctx, order.ClientOrderID)
		if err != nil {
			logs.Warning("auto_strategy ownership pending open status unavailable: ", managed.Symbol,
				" ", managed.PositionSide, " client_order_id=", order.ClientOrderID, " error=", err)
			return true
		}
		switch latest.Status {
		case futuresownership.OrderPending, futuresownership.OrderSubmitted, futuresownership.OrderPartiallyFilled, futuresownership.OrderReconcile:
			return true
		}
	}
	return false
}

func shouldRefreshOwnershipAccountQuantities(fillAdvanced bool) bool {
	return fillAdvanced
}

func reconcileAutoStrategyManagedOrder(ctx context.Context, order models.FuturesManagedOrder) (futuresownership.ExchangeOrder, bool, error) {
	// A generation-matched User Data mirror is authoritative for regular order
	// updates and is cheaper/faster than polling GET /order every StartTrade tick.
	if futuresUserDataMirrorUsable() {
		var local models.FuturesOrder
		err := orm.NewOrm().QueryTable(new(models.FuturesOrder)).
			Filter("account_id", "main").
			Filter("client_order_id", order.ClientOrderID).
			One(&local)
		if err == nil && local.UpdateTime > order.LastReconciledAt {
			observed, parseErr := futuresOrderObservation(local)
			if parseErr != nil {
				return futuresownership.ExchangeOrder{}, false, parseErr
			}
			applied, applyErr := sharedOwnershipExecutor.ApplyObservedExchange(ctx, order.ClientOrderID, observed)
			return applied, applied.FilledQty > order.FilledQty+1e-12, applyErr
		}
		if err != nil && err != orm.ErrNoRows {
			return futuresownership.ExchangeOrder{}, false, err
		}
	}

	// Successful submit/reconcile writes last_reconciled_at. Stable live orders
	// therefore use at most one REST status lookup per five seconds. Uncertain
	// submissions and orders without an exchange id bypass this throttle.
	if order.Status != futuresownership.OrderReconcile &&
		strings.TrimSpace(order.ExchangeOrderID) != "" &&
		order.LastReconciledAt > 0 &&
		time.Since(time.UnixMilli(order.LastReconciledAt)) < autoStrategyOrderRESTReconcileInterval {
		binanceapiusage.RecordOptimization("start_trade", "cache_hit", 1)
		binanceapiusage.RecordOptimization("start_trade", "prevented_duplicate", 1)
		return futuresownership.ExchangeOrder{}, false, nil
	}

	reconciled, err := sharedOwnershipExecutor.Reconcile(ctx, order.Symbol, order.ClientOrderID)
	if err != nil {
		return reconciled, false, err
	}
	return reconciled, reconciled.FilledQty > order.FilledQty+1e-12, nil
}

func futuresOrderObservation(order models.FuturesOrder) (futuresownership.ExchangeOrder, error) {
	exchangeOrderID := strings.TrimSpace(order.OrderId)
	if exchangeOrderID == "" {
		return futuresownership.ExchangeOrder{}, fmt.Errorf("local futures order %s has empty exchange order id", order.ClientOrderId)
	}
	if _, err := strconv.ParseInt(exchangeOrderID, 10, 64); err != nil {
		return futuresownership.ExchangeOrder{}, fmt.Errorf("parse local futures order id %q: %w", exchangeOrderID, err)
	}
	filled := 0.0
	if strings.TrimSpace(order.ExecutedQty) != "" {
		value, err := strconv.ParseFloat(order.ExecutedQty, 64)
		if err != nil {
			return futuresownership.ExchangeOrder{}, fmt.Errorf("parse local executed quantity for %s: %w", order.ClientOrderId, err)
		}
		filled = value
	}
	averagePrice := 0.0
	if strings.TrimSpace(order.AveragePrice) != "" {
		value, err := strconv.ParseFloat(order.AveragePrice, 64)
		if err != nil {
			return futuresownership.ExchangeOrder{}, fmt.Errorf("parse local average price for %s: %w", order.ClientOrderId, err)
		}
		averagePrice = value
	}
	return futuresownership.ExchangeOrder{
		ExchangeOrderID: exchangeOrderID,
		ClientOrderID:   order.ClientOrderId,
		Status:          order.Status,
		FilledQty:       filled,
		AveragePrice:    averagePrice,
	}, nil
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
	err := orm.NewOrm().QueryTable(new(models.FuturesManagedOrder)).Filter("account_id", "main").Filter("client_order_id", clientOrderID).One(&row)
	return row, err
}

func ensureAccountOpenSlotAvailable(owner, symbol string, positionSide futures.PositionSideType) error {
	ctx := binanceapiusage.WithSource(context.Background(), featureOwnerAPISource(owner))
	symbol = strings.ToUpper(strings.TrimSpace(symbol))

	if futuresUserDataMirrorUsable() {
		positions, err := GetTransformPositionsContext(ctx)
		if err != nil {
			return fmt.Errorf("verify local account positions before managed open: %w", err)
		}
		for _, position := range positions {
			qty, _ := strconv.ParseFloat(position.Amount, 64)
			if strings.EqualFold(position.Symbol, symbol) && strings.EqualFold(position.Side, string(positionSide)) && math.Abs(qty) > 1e-12 {
				return fmt.Errorf("%s %s already exists in account; ownership is not safe to merge", symbol, positionSide)
			}
		}
		orders, err := getTransformOpenOrdersContext(ctx)
		if err != nil {
			return fmt.Errorf("verify local account open orders before managed open: %w", err)
		}
		for _, order := range orders {
			isOpening := (positionSide == futures.PositionSideTypeLong && strings.EqualFold(order.Side, "BUY")) ||
				(positionSide == futures.PositionSideTypeShort && strings.EqualFold(order.Side, "SELL"))
			if isOpening && strings.EqualFold(order.Symbol, symbol) && strings.EqualFold(order.PositionSide, string(positionSide)) {
				return fmt.Errorf("%s %s already has an account open order; ownership is not safe to merge", symbol, positionSide)
			}
		}
		return nil
	}

	// Without a healthy User Data mirror, revalidate only the target symbol.
	// This preserves the final pre-mutation safety check while avoiding the
	// weight-40 account-wide openOrders request for every candidate.
	positions, err := binance.GetPositionContext(ctx, binance.PositionParams{Symbol: symbol})
	if err != nil {
		return fmt.Errorf("verify Binance account position before managed open: %w", err)
	}
	for _, position := range positions {
		qty, _ := strconv.ParseFloat(position.PositionAmt, 64)
		if strings.EqualFold(position.Symbol, symbol) &&
			strings.EqualFold(string(position.PositionSide), string(positionSide)) &&
			math.Abs(qty) > 1e-12 {
			return fmt.Errorf("%s %s already exists in account; ownership is not safe to merge", symbol, positionSide)
		}
	}

	orders, err := binance.GetOpenOrderContext(ctx, symbol)
	if err != nil {
		return fmt.Errorf("verify Binance symbol open orders before managed open: %w", err)
	}
	for _, order := range orders {
		isOpening := (positionSide == futures.PositionSideTypeLong && order.Side == futures.SideTypeBuy) ||
			(positionSide == futures.PositionSideTypeShort && order.Side == futures.SideTypeSell)
		if isOpening && strings.EqualFold(order.Symbol, symbol) &&
			strings.EqualFold(string(order.PositionSide), string(positionSide)) {
			return fmt.Errorf("%s %s already has an account open order; ownership is not safe to merge", symbol, positionSide)
		}
	}
	binanceapiusage.RecordOptimization(featureOwnerAPISource(owner), "prevented_duplicate", 1)
	return nil
}

func autoStrategySourceRef(symbol, strategyHash string) string {
	base := "auto_strategy:" + strings.ToUpper(strings.TrimSpace(symbol))
	strategyHash = strings.TrimSpace(strategyHash)
	if strategyHash == "" {
		return base
	}
	return base + ":" + strategyHash
}

func autoStrategyOpenRuleHash(owner, sourceRef string) string {
	if owner != futuresownership.OwnerAutoStrategy {
		return ""
	}
	parts := strings.Split(strings.TrimSpace(sourceRef), ":")
	if len(parts) != 3 || parts[0] != "auto_strategy" || len(parts[2]) != 64 {
		return ""
	}
	return parts[2]
}

func submitAutoStrategyOpen(snapshot *tradeCycleAccountSnapshot, symbol string, quantity, price float64, side futures.SideType, positionSide futures.PositionSideType, orderType futures.OrderType, strategyHash string) (*futures.CreateOrderResponse, error) {
	return submitOwnedFeatureOpenWithSnapshot(snapshot, futuresownership.OwnerAutoStrategy, autoStrategySourceRef(symbol, strategyHash), symbol, quantity, price, side, positionSide, orderType)
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
	return submitOwnedFeatureOpenWithSnapshot(nil, owner, sourceRef, symbol, quantity, price, side, positionSide, orderType)
}

func submitOwnedFeatureOpenWithSnapshot(snapshot *tradeCycleAccountSnapshot, owner, sourceRef, symbol string, quantity, price float64, side futures.SideType, positionSide futures.PositionSideType, orderType futures.OrderType) (*futures.CreateOrderResponse, error) {
	if snapshot != nil {
		if snapshot.HasPosition(symbol, positionSide) {
			return nil, fmt.Errorf("%s %s already exists in current account snapshot", strings.ToUpper(symbol), positionSide)
		}
		if snapshot.HasOpeningOrder(symbol, positionSide) {
			return nil, fmt.Errorf("%s %s already has an opening order in current account snapshot", strings.ToUpper(symbol), positionSide)
		}
	}
	if err := ensureAccountOpenSlotAvailable(owner, symbol, positionSide); err != nil {
		return nil, err
	}
	order, err := submitOwnedFeatureOrder(owner, sourceRef, symbol, quantity, price, 0, side, positionSide, orderType, futuresownership.IntentOpen)
	if err == nil && snapshot != nil {
		snapshot.RecordPendingOpen(symbol, side, positionSide, order.OrderID)
	}
	return order, err
}

func submitManagedStrategyClose(owner, sourceRef, symbol string, quantity float64, positionSide futures.PositionSideType) (*futures.CreateOrderResponse, error) {
	accountQty, err := currentAccountPositionQty(owner, symbol, positionSide)
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

func currentAccountPositionQty(owner, symbol string, positionSide futures.PositionSideType) (float64, error) {
	ctx := binanceapiusage.WithSource(context.Background(), featureOwnerAPISource(owner))
	symbol = strings.ToUpper(strings.TrimSpace(symbol))

	if futuresUserDataMirrorUsable() {
		positions, err := GetTransformPositionsContext(ctx)
		if err != nil {
			return 0, fmt.Errorf("verify local account position before managed close: %w", err)
		}
		for _, position := range positions {
			if strings.EqualFold(position.Symbol, symbol) && strings.EqualFold(position.Side, string(positionSide)) {
				qty, parseErr := strconv.ParseFloat(position.Amount, 64)
				if parseErr != nil {
					return 0, fmt.Errorf("parse local account quantity for %s %s: %w", symbol, positionSide, parseErr)
				}
				return math.Abs(qty), nil
			}
		}
		return 0, nil
	}

	positions, err := binance.GetPositionContext(ctx, binance.PositionParams{Symbol: symbol})
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

// A cached snapshot may omit a freshly filled position. Do not shrink or close
// ownership without an authoritative fresh exchange read.
func needsFreshOwnershipSnapshot(managed []models.FuturesManagedPosition, quantities map[string]float64, force bool) bool {
	if force {
		return true
	}
	for _, row := range managed {
		if quantities[managedPositionKey(row.Symbol, row.PositionSide)]+1e-12 < row.ManagedQty {
			return true
		}
	}
	return false
}

// Parse ONLY the supplied snapshot. Callers that need authoritative data
// must explicitly fetch it and keep the full rows for exit evaluation.
func ownershipAccountQuantities(positions []types.FuturesPosition) (map[string]float64, error) {
	out := make(map[string]float64, len(positions))
	for _, position := range positions {
		qty, err := strconv.ParseFloat(position.Amount, 64)
		if err != nil || math.IsNaN(qty) || math.IsInf(qty, 0) {
			return nil, fmt.Errorf("invalid account quantity for %s %s: %q", position.Symbol, position.Side, position.Amount)
		}
		out[managedPositionKey(position.Symbol, position.Side)] = math.Abs(qty)
	}
	return out, nil
}

func featureOwnerAPISource(owner string) string {
	switch owner {
	case futuresownership.OwnerAutoStrategy:
		return "start_trade"
	case futuresownership.OwnerNewCoinRush:
		return "new_coin_rush"
	case futuresownership.OwnerNoticeAutoOrder:
		return "notice_auto_order"
	case futuresownership.OwnerFundingRate:
		return "funding_rate"
	case futuresownership.OwnerAgentTrade:
		return "agent_trade"
	default:
		return "managed_trade"
	}
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
