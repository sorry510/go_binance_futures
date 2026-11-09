package tradefills

import (
	"errors"
	"fmt"
	"math/big"
	"regexp"
	"strconv"
	"strings"

	"go_binance_futures/models"

	"github.com/adshao/go-binance/v2/futures"
	"github.com/beego/beego/v2/client/orm"
	"github.com/beego/beego/v2/core/logs"
)

const MarketUSDM = "USDM"

var decimalPattern = regexp.MustCompile(`^-?[0-9]+(?:\.[0-9]+)?$`)

// Numeric strings retain the exact exchange representation. SQL DECIMAL(36,18)
// must not silently round a quantity, fee or PnL.
func decimal(value string, allowNegative bool) (*big.Rat, error) {
	s := strings.TrimSpace(value)
	if !decimalPattern.MatchString(s) || (!allowNegative && strings.HasPrefix(s, "-")) {
		return nil, fmt.Errorf("invalid decimal %q", value)
	}
	digits := strings.TrimPrefix(s, "-")
	parts := strings.SplitN(digits, ".", 2)
	fraction := ""
	if len(parts) == 2 {
		fraction = parts[1]
	}
	if len(fraction) > 18 || len(strings.TrimLeft(parts[0], "0")) > 18 {
		return nil, fmt.Errorf("decimal %q exceeds DECIMAL(36,18)", value)
	}
	result, ok := new(big.Rat).SetString(s)
	if !ok {
		return nil, fmt.Errorf("cannot parse decimal %q", value)
	}
	return result, nil
}

func validate(fill models.FuturesTradeFill) error {
	if fill.Market != MarketUSDM || (fill.AccountID != "main" && fill.AccountID != "lead") ||
		strings.TrimSpace(fill.Symbol) == "" || fill.OrderID == "" || fill.TradeID == "" || fill.TradeTime <= 0 {
		return fmt.Errorf("invalid trade fill identity")
	}
	if orderID, err := strconv.ParseInt(fill.OrderID, 10, 64); err != nil || orderID <= 0 {
		return fmt.Errorf("invalid exchange order id")
	}
	if id, err := strconv.ParseInt(fill.TradeID, 10, 64); err != nil || id <= 0 {
		return fmt.Errorf("invalid exchange trade id")
	}
	qty, err := decimal(fill.Qty, false)
	if err != nil || qty.Sign() <= 0 {
		return fmt.Errorf("invalid fill quantity: %v", err)
	}
	price, err := decimal(fill.Price, false)
	if err != nil || price.Sign() <= 0 {
		return fmt.Errorf("invalid fill price: %v", err)
	}
	if _, err := decimal(fill.Commission, true); err != nil {
		return fmt.Errorf("invalid fee: %w", err)
	}
	if _, err := decimal(fill.RealizedPnL, true); err != nil {
		return fmt.Errorf("invalid realized pnl: %w", err)
	}
	// The exchange can omit the fee denomination for a verified ZERO fee.
	// Empty asset on a nonzero fee remains unsafe and is rejected.
	if fill.RESTVerified && strings.TrimSpace(fill.CommissionAsset) == "" {
		fee, err := decimal(fill.Commission, true)
		if err != nil || fee.Sign() != 0 {
			return fmt.Errorf("verified REST nonzero/invalid fee has no asset")
		}
	}
	if len(fill.Symbol) > 32 || len(fill.ClientOrderID) > 64 || len(fill.CommissionAsset) > 16 ||
		len(fill.PnLAsset) > 16 || len(fill.Intent) > 24 || len(fill.PositionSide) > 8 ||
		len(fill.OrderID) > 32 || len(fill.TradeID) > 32 || len(fill.AccountID) > 16 {
		return fmt.Errorf("exchange trade value exceeds database column width")
	}
	if fill.PositionID != nil && *fill.PositionID <= 0 {
		return fmt.Errorf("invalid position_id")
	}
	if fill.PositionSide != "" && fill.PositionSide != "LONG" && fill.PositionSide != "SHORT" && fill.PositionSide != "BOTH" {
		return fmt.Errorf("invalid exchange position side")
	}
	return nil
}

// FromWS accepts only execution TRADE events with a real exchange trade ID.
// NEW, CANCELED and zero-quantity updates do not become synthetic fills.
func FromWS(account string, event futures.WsOrderTradeUpdate) (models.FuturesTradeFill, bool, error) {
	if string(event.ExecutionType) != "TRADE" {
		return models.FuturesTradeFill{}, false, nil
	}
	if event.TradeID <= 0 || event.ID <= 0 {
		return models.FuturesTradeFill{}, false, fmt.Errorf("TRADE event has no real trade/order ID")
	}
	qty, err := decimal(event.LastFilledQty, false)
	if err != nil {
		return models.FuturesTradeFill{}, false, err
	}
	if qty.Sign() <= 0 {
		return models.FuturesTradeFill{}, false, nil
	}
	// Binance may omit n/N entirely for a zero-fee trade. Preserve the
	// execution but never infer fee currency or treat WS fees as confirmed.
	commission := strings.TrimSpace(event.Commission)
	if commission == "" {
		commission = "0"
	}
	fill := models.FuturesTradeFill{
		Market: MarketUSDM, AccountID: account, Symbol: strings.ToUpper(strings.TrimSpace(event.Symbol)),
		OrderID: strconv.FormatInt(event.ID, 10), TradeID: strconv.FormatInt(event.TradeID, 10),
		ClientOrderID: event.ClientOrderID, PositionSide: string(event.PositionSide),
		Qty: event.LastFilledQty, Price: event.LastFilledPrice, Commission: commission,
		CommissionAsset: event.CommissionAsset, RealizedPnL: event.RealizedPnL,
		TradeTime: event.TradeTime,
	}
	return fill, true, validate(fill)
}

func FromREST(account string, trade *futures.AccountTrade) (models.FuturesTradeFill, error) {
	if trade == nil {
		return models.FuturesTradeFill{}, fmt.Errorf("nil REST trade")
	}
	fill := models.FuturesTradeFill{
		Market: MarketUSDM, AccountID: account, Symbol: strings.ToUpper(strings.TrimSpace(trade.Symbol)),
		OrderID: strconv.FormatInt(trade.OrderID, 10), TradeID: strconv.FormatInt(trade.ID, 10),
		PositionSide: string(trade.PositionSide), Qty: trade.Quantity, Price: trade.Price,
		Commission: trade.Commission, CommissionAsset: trade.CommissionAsset,
		RealizedPnL: trade.RealizedPnl, TradeTime: trade.Time, RESTVerified: true,
	}
	return fill, validate(fill)
}

// Identity is matched by exact account + symbol + REAL matching order ID,
// never by qty/time/symbol alone. AlgoId must not be confused with actualOrderId.
func matchingOrder(o orm.QueryExecutor, fill models.FuturesTradeFill) (*models.FuturesManagedOrder, error) {
	// Two indexed equality lookups; never scan every historical order for
	// every WS trade as this ledger grows.
	var byActual []models.FuturesManagedOrder
	_, err := o.QueryTable(new(models.FuturesManagedOrder)).
		Filter("account_id", fill.AccountID).Filter("symbol", fill.Symbol).
		Filter("actual_order_id", fill.OrderID).All(&byActual)
	if err != nil {
		return nil, err
	}
	var byDirect []models.FuturesManagedOrder
	_, err = o.QueryTable(new(models.FuturesManagedOrder)).
		Filter("account_id", fill.AccountID).Filter("symbol", fill.Symbol).
		Filter("exchange_order_id", fill.OrderID).All(&byDirect)
	if err != nil {
		return nil, err
	}
	var match *models.FuturesManagedOrder
	for i := range byActual {
		order := &byActual[i]
		if match != nil && match.ID != order.ID {
			return nil, fmt.Errorf("duplicate actual order ID mapping")
		}
		match = order
	}
	for i := range byDirect {
		order := &byDirect[i]
		// A matching algoId is NOT proof that the exchange matching order
		// carrying the fill is the algo's actualOrderId.
		if isAlgoOrder(order.OrderType) {
			continue
		}
		if match != nil && match.ID != order.ID {
			return nil, fmt.Errorf("ambiguous managed order ID mapping")
		}
		match = order
	}
	if match != nil && fill.PositionSide != "" && fill.PositionSide != "BOTH" && fill.PositionSide != match.PositionSide {
		return nil, fmt.Errorf("trade %s position side does not match managed order", fill.TradeID)
	}
	return match, nil
}

func isAlgoOrder(orderType string) bool {
	switch strings.ToUpper(strings.TrimSpace(orderType)) {
	case "STOP", "STOP_MARKET", "TAKE_PROFIT", "TAKE_PROFIT_MARKET", "TRAILING_STOP_MARKET":
		return true
	}
	return false
}

// Insert is idempotent and detects inconsistent WS/REST records. The caller
// chooses its own transaction and may commit the fill with another write.
func Insert(o orm.QueryExecutor, fill models.FuturesTradeFill) error {
	if err := validate(fill); err != nil {
		return err
	}
	order, err := matchingOrder(o, fill)
	if err != nil {
		return err
	}
	if order != nil {
		fill.PositionID = order.PositionID
		fill.ClientOrderID = order.ClientOrderID
		fill.Intent = order.Intent
		fill.PositionSide = order.PositionSide
	}
	var pos interface{}
	if fill.PositionID != nil {
		pos = *fill.PositionID
	}
	verified := 0
	if fill.RESTVerified {
		verified = 1
	}
	const statement = `INSERT INTO futures_trade_fills
       (market,account_id,symbol,order_id,trade_id,position_id,client_order_id,intent,position_side,
       qty,price,commission,commission_asset,realized_pnl,pnl_asset,trade_time,rest_verified)
       VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`
	_, err = o.Raw(statement, fill.Market, fill.AccountID, fill.Symbol, fill.OrderID, fill.TradeID,
		pos, fill.ClientOrderID, fill.Intent, fill.PositionSide, fill.Qty, fill.Price,
		fill.Commission, fill.CommissionAsset, fill.RealizedPnL, fill.PnLAsset, fill.TradeTime, verified).Exec()
	if err == nil {
		return nil
	}

	// Check whether this is a true duplicate; never swallow arbitrary DB
	// failures such as missing schema, permission errors or trigger failures.
	var old models.FuturesTradeFill
	var source int
	scanErr := o.Raw(`SELECT qty,price,commission,commission_asset,realized_pnl,
       trade_time,position_side,rest_verified FROM futures_trade_fills
       WHERE market=? AND account_id=? AND symbol=? AND order_id=? AND trade_id=?`,
		fill.Market, fill.AccountID, fill.Symbol, fill.OrderID, fill.TradeID).
		QueryRow(&old.Qty, &old.Price, &old.Commission, &old.CommissionAsset,
			&old.RealizedPnL, &old.TradeTime, &old.PositionSide, &source)
	if scanErr != nil {
		return err
	}
	checkNum := func(a, b string) bool {
		left, e1 := decimal(a, true)
		right, e2 := decimal(b, true)
		return e1 == nil && e2 == nil && left.Cmp(right) == 0
	}
	// Both sources must agree on core exchange execution facts. No
	// replacement of quantity or price under any circumstances.
	if !checkNum(old.Qty, fill.Qty) || !checkNum(old.Price, fill.Price) ||
		(old.PositionSide != "" && fill.PositionSide != "" && old.PositionSide != fill.PositionSide) {
		return fmt.Errorf("conflicting exchange trade identity %s/%s: %w", fill.OrderID, fill.TradeID, err)
	}
	if source == 0 && fill.RESTVerified {
		// WS n/N and T may be absent, cumulative or arrive with a distinct
		// millisecond timestamp. The official REST userTrades is the
		// authoritative per-trade fee, asset, realized PnL and timestamp.
		result, updateErr := o.Raw(`UPDATE futures_trade_fills SET commission=?,commission_asset=?,
            realized_pnl=?,trade_time=?,rest_verified=1
            WHERE market=? AND account_id=? AND symbol=? AND order_id=? AND trade_id=?
              AND rest_verified=0`,
			fill.Commission, fill.CommissionAsset, fill.RealizedPnL, fill.TradeTime,
			fill.Market, fill.AccountID, fill.Symbol, fill.OrderID, fill.TradeID).Exec()
		if updateErr != nil {
			return updateErr
		}
		changed, affectedErr := result.RowsAffected()
		if affectedErr != nil {
			return affectedErr
		}
		if changed != 1 {
			return fmt.Errorf("concurrent REST trade confirmation; retry %s/%s", fill.OrderID, fill.TradeID)
		}
		// Preserve observability for the real-money F1 Gate without letting
		// provisional WS n/N or T override the REST per-execution evidence.
		if !checkNum(old.Commission, fill.Commission) ||
			old.CommissionAsset != fill.CommissionAsset ||
			!checkNum(old.RealizedPnL, fill.RealizedPnL) ||
			old.TradeTime != fill.TradeTime {
			logs.Warning("报警: futures WS/REST trade money/time discrepancy; REST used as authoritative",
				" account=", fill.AccountID, " symbol=", fill.Symbol, " order_id=", fill.OrderID,
				" trade_id=", fill.TradeID, " ws_fee=", old.Commission,
				" ws_fee_asset=", old.CommissionAsset, " rest_fee=", fill.Commission,
				" rest_fee_asset=", fill.CommissionAsset, " ws_time=", old.TradeTime,
				" rest_time=", fill.TradeTime)
		}
	} else if source == 1 && !fill.RESTVerified {
		// Late WS events cannot overwrite or invalidate verified REST money.
	} else {
		// Same-source immutable duplicates must be consistent. For REST the
		// trade timestamp is authoritative; for WS it is still provisional.
		if !checkNum(old.Commission, fill.Commission) ||
			!checkNum(old.RealizedPnL, fill.RealizedPnL) ||
			old.CommissionAsset != fill.CommissionAsset ||
			old.TradeTime != fill.TradeTime {
			return fmt.Errorf("conflicting same-source trade payload %s/%s: %w", fill.OrderID, fill.TradeID, err)
		}
	}
	// The trade could precede the managed order/position registration.
	if order != nil && order.PositionID != nil {
		return Attach(o, *order)
	}
	return nil
}

// UnverifiedCount is the number of execution facts whose money fields have
// NOT been corroborated by REST. Quantity equality alone is not completion.
func UnverifiedCount(o orm.QueryExecutor, order models.FuturesManagedOrder) (int64, error) {
	actual := order.ExchangeOrderID
	if isAlgoOrder(order.OrderType) {
		if order.ActualOrderID == nil {
			return 0, nil
		}
		actual = *order.ActualOrderID
	}
	if order.PositionID == nil || actual == "" {
		return 0, nil
	}
	var count int64
	err := o.Raw(`SELECT COUNT(*) FROM futures_trade_fills WHERE market='USDM'
        AND account_id=? AND symbol=? AND order_id=? AND position_id=?
        AND rest_verified=0`, order.AccountID, order.Symbol, actual, *order.PositionID).QueryRow(&count)
	return count, err
}

// Attach binds previously observed fills only when a managed order has a
// verified exchange orderId (or algo's separately verified actualOrderId).
func Attach(o orm.QueryExecutor, order models.FuturesManagedOrder) error {
	if order.PositionID == nil || *order.PositionID <= 0 {
		return nil
	}
	actual := order.ExchangeOrderID
	if isAlgoOrder(order.OrderType) {
		if order.ActualOrderID == nil || *order.ActualOrderID == "" {
			return nil
		}
		actual = *order.ActualOrderID
	}
	if actual == "" {
		return nil
	}
	var mismatch int64
	if err := o.Raw(`SELECT COUNT(*) FROM futures_trade_fills
        WHERE market='USDM' AND account_id=? AND symbol=? AND order_id=?
          AND (position_id IS NOT NULL AND position_id<>? OR (position_side<>'' AND position_side<>'BOTH' AND position_side<>?))`,
		order.AccountID, order.Symbol, actual, *order.PositionID, order.PositionSide).QueryRow(&mismatch); err != nil {
		return err
	}
	if mismatch > 0 {
		return errors.New("fill already attached to another position or side")
	}
	_, err := o.Raw(`UPDATE futures_trade_fills SET position_id=?,client_order_id=?,intent=?,position_side=?
      WHERE market='USDM' AND account_id=? AND symbol=? AND order_id=? AND position_id IS NULL
        AND (position_side='' OR position_side='BOTH' OR position_side=?)`,
		*order.PositionID, order.ClientOrderID, order.Intent, order.PositionSide,
		order.AccountID, order.Symbol, actual, order.PositionSide).Exec()
	return err
}

// Quantity totals must be compared against Binance cumulative execution qty;
// a missing trade is never estimated from order average or the WS mirror.
func Quantity(o orm.QueryExecutor, order models.FuturesManagedOrder) (*big.Rat, error) {
	actual := order.ExchangeOrderID
	if isAlgoOrder(order.OrderType) {
		if order.ActualOrderID == nil {
			return new(big.Rat), nil
		}
		actual = *order.ActualOrderID
	}
	if actual == "" {
		return new(big.Rat), nil
	}
	if order.PositionID == nil {
		return new(big.Rat), nil
	}
	var qty string
	err := o.Raw(`SELECT COALESCE(SUM(qty),0) FROM futures_trade_fills
        WHERE market='USDM' AND account_id=? AND symbol=? AND order_id=? AND position_id=?`,
		order.AccountID, order.Symbol, actual, *order.PositionID).QueryRow(&qty)
	if err != nil {
		return nil, err
	}
	return decimal(qty, true)
}

// WeightedAverage uses the exact exchange strings, not a floating point
// reconstruction from consecutive cumulative order snapshots. An incomplete
// order is not considered covered by this function; callers compare qty.
func WeightedAverage(o orm.QueryExecutor, order models.FuturesManagedOrder) (*big.Rat, *big.Rat, error) {
	qtyTotal := new(big.Rat)
	notional := new(big.Rat)
	if order.PositionID == nil {
		return qtyTotal, nil, nil
	}
	actual := order.ExchangeOrderID
	if isAlgoOrder(order.OrderType) {
		if order.ActualOrderID == nil {
			return qtyTotal, nil, nil
		}
		actual = *order.ActualOrderID
	}
	if actual == "" {
		return qtyTotal, nil, nil
	}
	var fills []struct {
		Qty   string `orm:"column(qty)"`
		Price string `orm:"column(price)"`
	}
	_, err := o.Raw(`SELECT qty,price FROM futures_trade_fills WHERE market='USDM' AND account_id=?
        AND symbol=? AND order_id=? AND position_id=?`,
		order.AccountID, order.Symbol, actual, *order.PositionID).QueryRows(&fills)
	if err != nil {
		return nil, nil, err
	}
	for _, fill := range fills {
		qty, err := decimal(fill.Qty, false)
		if err != nil {
			return nil, nil, err
		}
		price, err := decimal(fill.Price, false)
		if err != nil {
			return nil, nil, err
		}
		qtyTotal.Add(qtyTotal, qty)
		notional.Add(notional, new(big.Rat).Mul(qty, price))
	}
	if qtyTotal.Sign() == 0 {
		return qtyTotal, nil, nil
	}
	return qtyTotal, new(big.Rat).Quo(notional, qtyTotal), nil
}
