package futuresownership

import (
	"context"
	"fmt"
	"math"
	"math/big"
	"strings"

	"go_binance_futures/models"

	"github.com/beego/beego/v2/client/orm"
)

const (
	SettlementLegacy     = "legacy"
	SettlementOpen       = "open"
	SettlementIncomplete = "incomplete"
	SettlementSettled    = "settled"
)

type SettlementResult struct {
	PositionID int64  `json:"position_id"`
	Status     string `json:"status"`
	Reason     string `json:"reason,omitempty"`
	Asset      string `json:"asset,omitempty"`
	GrossPnL   string `json:"gross_pnl,omitempty"`
	Fee        string `json:"fee,omitempty"`
	NetPnL     string `json:"net_pnl,omitempty"`
}

type settlementFill struct {
	OrderID         string `orm:"column(order_id)"`
	Qty             string `orm:"column(qty)"`
	Price           string `orm:"column(price)"`
	Commission      string `orm:"column(commission)"`
	CommissionAsset string `orm:"column(commission_asset)"`
	RealizedPnL     string `orm:"column(realized_pnl)"`
	PnLAsset        string `orm:"column(pnl_asset)"`
	AccountID       string `orm:"column(account_id)"`
	Symbol          string `orm:"column(symbol)"`
	PositionSide    string `orm:"column(position_side)"`
	Intent          string `orm:"column(intent)"`
	ClientOrderID   string `orm:"column(client_order_id)"`
	RESTVerified    int    `orm:"column(rest_verified)"`
}

func parsedSettlementAmount(raw string) (*big.Rat, error) {
	x, ok := new(big.Rat).SetString(strings.TrimSpace(raw))
	if !ok {
		return nil, fmt.Errorf("invalid exact settlement amount")
	}
	return x, nil
}
func quantityMatches(actual *big.Rat, expected float64) bool {
	if math.IsNaN(expected) || math.IsInf(expected, 0) || expected < 0 {
		return false
	}
	want, ok := new(big.Rat).SetString(fmt.Sprintf("%.12f", expected))
	if !ok {
		return false
	}
	return new(big.Rat).Abs(new(big.Rat).Sub(actual, want)).Cmp(big.NewRat(1, 1000000000)) <= 0
}
func actualMatchID(order models.FuturesManagedOrder) string {
	if isAlgoManagedOrderType(order.OrderType) {
		if order.ActualOrderID == nil {
			return ""
		}
		return *order.ActualOrderID
	}
	return order.ExchangeOrderID
}

func verifiedSettlementFacts(o orm.QueryExecutor, pos models.FuturesManagedPosition, asset string) (gross, fee, net *big.Rat, reason string, err error) {
	if asset != "USDT" && asset != "USDC" {
		return nil, nil, nil, "unsupported or unverified USD-M settlement asset", nil
	}
	var orders []models.FuturesManagedOrder
	if _, err = o.QueryTable(new(models.FuturesManagedOrder)).
		Filter("account_id", pos.AccountID).Filter("position_id", pos.ID).
		OrderBy("id").All(&orders); err != nil {
		return nil, nil, nil, "", err
	}
	if len(orders) == 0 {
		return nil, nil, nil, "no linked managed orders", nil
	}
	byID := make(map[string]models.FuturesManagedOrder, len(orders))
	hasOpen, hasClose := false, false
	for _, order := range orders {
		if order.Owner != pos.Owner || order.Symbol != pos.Symbol || order.PositionSide != pos.PositionSide {
			return nil, nil, nil, "position order identity mismatch", nil
		}
		if liveOrderStatus(order.Status) {
			return nil, nil, nil, "outstanding managed order", nil
		}
		if order.FilledQty <= qtyEpsilon {
			continue
		}
		id := actualMatchID(order)
		if id == "" {
			return nil, nil, nil, "actual matching order ID missing", nil
		}
		if _, exists := byID[id]; exists {
			return nil, nil, nil, "duplicate exchange matching order in position", nil
		}
		byID[id] = order
		if order.Intent == IntentOpen {
			hasOpen = true
		} else if order.Intent == IntentClose || order.Intent == IntentStopLoss || order.Intent == IntentTakeProfit {
			hasClose = true
		} else {
			return nil, nil, nil, "unsupported managed order intent", nil
		}
	}
	if !hasOpen || !hasClose {
		return nil, nil, nil, "both opening and closing execution are required", nil
	}
	var fills []settlementFill
	if _, err = o.Raw(`SELECT order_id,qty,price,commission,commission_asset,
        realized_pnl,pnl_asset,account_id,symbol,position_side,intent,client_order_id,rest_verified
        FROM futures_trade_fills WHERE market='USDM' AND position_id=? AND account_id=?`, pos.ID, pos.AccountID).
		QueryRows(&fills); err != nil {
		return nil, nil, nil, "", err
	}
	if len(fills) == 0 {
		return nil, nil, nil, "no confirmed exchange trade fills", nil
	}
	gross = new(big.Rat)
	fee = new(big.Rat)
	actualQty := make(map[string]*big.Rat)
	opened, closed := new(big.Rat), new(big.Rat)
	for _, fill := range fills {
		if fill.RESTVerified != 1 {
			return nil, nil, nil, "trade fee and realized PnL not confirmed by Binance REST", nil
		}
		order, ok := byID[fill.OrderID]
		if !ok || fill.AccountID != pos.AccountID || fill.Symbol != pos.Symbol ||
			fill.PositionSide != pos.PositionSide || fill.Intent != order.Intent ||
			fill.ClientOrderID != order.ClientOrderID {
			return nil, nil, nil, "trade fill order/cycle identity mismatch", nil
		}
		if fill.PnLAsset != "" && fill.PnLAsset != asset {
			return nil, nil, nil, "realized PnL asset conflicts with exchange margin asset", nil
		}
		qty, parseErr := parsedSettlementAmount(fill.Qty)
		if parseErr != nil || qty.Sign() <= 0 {
			return nil, nil, nil, "invalid exact trade quantity", nil
		}
		cost, parseErr := parsedSettlementAmount(fill.Commission)
		if parseErr != nil {
			return nil, nil, nil, "invalid exact commission", nil
		}
		if fill.CommissionAsset != asset {
			// A REST-confirmed exact zero fee has no asset value to
			// convert. All nonzero fees still require a proven denomination.
			if fill.CommissionAsset != "" || cost.Sign() != 0 {
				return nil, nil, nil, "commission in a different or unknown asset requires FX valuation", nil
			}
		}
		pnl, parseErr := parsedSettlementAmount(fill.RealizedPnL)
		if parseErr != nil {
			return nil, nil, nil, "invalid exact realized profit", nil
		}
		// Reject invalid price even though it does not directly determine PnL.
		price, parseErr := parsedSettlementAmount(fill.Price)
		if parseErr != nil || price.Sign() <= 0 {
			return nil, nil, nil, "invalid exact trade price", nil
		}
		if actualQty[fill.OrderID] == nil {
			actualQty[fill.OrderID] = new(big.Rat)
		}
		actualQty[fill.OrderID].Add(actualQty[fill.OrderID], qty)
		if order.Intent == IntentOpen {
			opened.Add(opened, qty)
		} else {
			closed.Add(closed, qty)
		}
		gross.Add(gross, pnl)
		fee.Add(fee, cost)
	}
	for id, order := range byID {
		qty := actualQty[id]
		if qty == nil || !quantityMatches(qty, order.FilledQty) {
			return nil, nil, nil, "some order fills are missing or exceed cumulative execution", nil
		}
	}
	if opened.Cmp(closed) != 0 {
		return nil, nil, nil, "opening and closing fills differ: external reduction or missing fills", nil
	}
	if opened.Sign() <= 0 {
		return nil, nil, nil, "no positive opening volume", nil
	}
	return gross, fee, new(big.Rat).Sub(gross, fee), "", nil
}

// settleVerifiedFlat must only be called after an account-bound FRESH REST
// position snapshot confirms ZERO for the same symbol/side. A WS mirror, old
// order history, estimated exit price or an external reduction is insufficient.
//
// A single transaction CAS-marks the cycle and applies durable config streaks;
// concurrent workers and process restarts cannot apply risk twice.
func (s Service) settleVerifiedFlat(ctx context.Context, positionID int64, verifiedMarginAsset string) (SettlementResult, error) {
	if err := ctx.Err(); err != nil {
		return SettlementResult{}, err
	}
	account, err := s.accountID()
	if err != nil {
		return SettlementResult{}, err
	}
	if positionID <= 0 {
		return SettlementResult{}, fmt.Errorf("invalid position id")
	}
	mutationMu.Lock()
	defer mutationMu.Unlock()
	tx, err := orm.NewOrm().Begin()
	if err != nil {
		return SettlementResult{}, err
	}
	defer tx.Rollback()
	var pos models.FuturesManagedPosition
	if err = tx.QueryTable(new(models.FuturesManagedPosition)).Filter("id", positionID).
		Filter("account_id", account).One(&pos); err != nil {
		return SettlementResult{}, err
	}
	result := SettlementResult{PositionID: pos.ID, Status: pos.SettlementStatus}
	if pos.SettlementStatus == SettlementSettled {
		return result, nil
	}
	if pos.SettlementStatus != SettlementIncomplete || pos.Status != PositionClosed ||
		pos.ManagedQty > qtyEpsilon || pos.ClosedAt <= 0 {
		result.Reason = "not a verified post-v22 closed cycle"
		return result, nil
	}
	asset := strings.ToUpper(strings.TrimSpace(verifiedMarginAsset))
	gross, fee, net, reason, err := verifiedSettlementFacts(tx, pos, asset)
	if err != nil {
		return result, err
	}
	if reason != "" {
		result.Reason = reason
		return result, nil
	}
	// ExchangeInfo.marginAsset is an authoritative contract property, not
	// part of the trade payload. Only after verifying it can pnl_asset be
	// backfilled for recorded fills that lack a denomination.
	if _, err = tx.Raw(`UPDATE futures_trade_fills SET pnl_asset=?
        WHERE market='USDM' AND position_id=? AND account_id=? AND pnl_asset=''`,
		asset, pos.ID, account).Exec(); err != nil {
		return result, err
	}
	sign := net.Sign()
	now := s.nowMillis()
	// CAS is the durable once-only barrier shared by multiple app instances.
	affected, err := tx.QueryTable(new(models.FuturesManagedPosition)).
		Filter("id", pos.ID).Filter("account_id", account).
		Filter("settlement_status", SettlementIncomplete).
		Filter("settlement_processed_at", 0).Filter("status", PositionClosed).
		Update(orm.Params{"settlement_status": SettlementSettled,
			"settlement_processed_at": now, "settlement_pnl_sign": sign})
	if err != nil {
		return result, err
	}
	if affected != 1 {
		return result, fmt.Errorf("position settlement was concurrently changed")
	}
	if pos.Owner == OwnerAutoStrategy && pos.AccountID == "main" {
		if err = applySettledRisk(tx, sign); err != nil {
			return result, err
		}
	}
	if err = tx.Commit(); err != nil {
		return result, err
	}
	result.Status = SettlementSettled
	result.Asset = asset
	result.GrossPnL = gross.FloatString(18)
	result.Fee = fee.FloatString(18)
	result.NetPnL = net.FloatString(18)
	return result, nil
}

// applySettledRisk is the ONLY active automatic loss-scale/test-mode trigger.
// It runs inside the once-only verified settlement transaction. Missing
// zero-position exchange proof must remain INCOMPLETE with operator alarm,
// never use old speculative prices as a fallback.
func applySettledRisk(tx orm.TxOrmer, netSign int) error {
	db, err := orm.GetDB("default")
	if err != nil {
		return err
	}
	query := "SELECT * FROM config ORDER BY id LIMIT 1"
	if strings.Contains(strings.ToLower(fmt.Sprintf("%T", db.Driver())), "mysql") {
		query += " FOR UPDATE"
	}
	var cfg models.Config
	if err = tx.Raw(query).QueryRow(&cfg); err != nil {
		return err
	}
	if cfg.ID <= 0 {
		return fmt.Errorf("missing risk configuration")
	}
	updates := orm.Params{}
	if cfg.LossAutoScale != 0 {
		if netSign >= 0 {
			cfg.SettlementScaleBalance++
		} else {
			cfg.SettlementScaleBalance--
		}
		if cfg.SettlementScaleBalance >= 2 {
			cfg.SettlementScaleBalance = 0
			cfg.LossMaxCount++
			updates["loss_max_count"] = cfg.LossMaxCount
		} else if cfg.SettlementScaleBalance <= -2 {
			cfg.SettlementScaleBalance = 0
			cfg.LossMaxCount--
			if cfg.LossMaxCount < 1 {
				cfg.LossMaxCount = 1
			}
			updates["loss_max_count"] = cfg.LossMaxCount
		}
		updates["settlement_scale_balance"] = cfg.SettlementScaleBalance
	}
	if cfg.FutureTestAutoTradeCountLimit > 0 {
		if netSign >= 0 {
			cfg.SettlementConsecutiveLosses = 0
		} else {
			cfg.SettlementConsecutiveLosses++
		}
		if cfg.SettlementConsecutiveLosses >= cfg.FutureTestAutoTradeCountLimit {
			cfg.SettlementConsecutiveLosses = 0
			updates["future_enable"] = 1
			updates["future_allow_long"] = 0
			updates["future_allow_short"] = 0
			updates["future_test"] = 1
		}
		updates["settlement_consecutive_losses"] = cfg.SettlementConsecutiveLosses
	}
	if len(updates) == 0 {
		return nil
	}
	n, err := tx.QueryTable(new(models.Config)).Filter("id", cfg.ID).Update(updates)
	if err != nil {
		return err
	}
	if n != 1 {
		return fmt.Errorf("risk configuration changed during settlement")
	}
	return nil
}
