package feature

import (
	"math"
	"strconv"
	"strings"

	"go_binance_futures/types"

	"github.com/adshao/go-binance/v2/futures"
)

type tradeCycleAccountSnapshot struct {
	positions      []types.FuturesPosition
	openOrders     []types.FuturesOrder
	positionByKey  map[string]bool
	openOrderByKey map[string]bool
	positionCount  int
	openOrderCount int
}

func newTradeCycleAccountSnapshot(positions []types.FuturesPosition, openOrders []types.FuturesOrder) *tradeCycleAccountSnapshot {
	snapshot := &tradeCycleAccountSnapshot{
		positions:      append([]types.FuturesPosition(nil), positions...),
		openOrders:     append([]types.FuturesOrder(nil), openOrders...),
		positionByKey:  make(map[string]bool),
		openOrderByKey: make(map[string]bool),
	}
	for _, position := range positions {
		qty, _ := strconv.ParseFloat(position.Amount, 64)
		if math.Abs(qty) <= 1e-12 {
			continue
		}
		snapshot.positionCount++
		snapshot.positionByKey[accountSlotKey(position.Symbol, position.Side)] = true
	}
	for _, order := range openOrders {
		if !isOpenOrderStatus(order.Status) {
			continue
		}
		snapshot.openOrderCount++
		if isOpeningSide(order.Side, order.PositionSide) {
			snapshot.openOrderByKey[accountSlotKey(order.Symbol, order.PositionSide)] = true
		}
	}
	return snapshot
}

func (s *tradeCycleAccountSnapshot) HasPosition(symbol string, side futures.PositionSideType) bool {
	return s != nil && s.positionByKey[accountSlotKey(symbol, string(side))]
}

func (s *tradeCycleAccountSnapshot) HasOpeningOrder(symbol string, side futures.PositionSideType) bool {
	return s != nil && s.openOrderByKey[accountSlotKey(symbol, string(side))]
}

func (s *tradeCycleAccountSnapshot) AccountSlotCount() int {
	if s == nil {
		return 0
	}
	return s.positionCount + s.openOrderCount
}

func (s *tradeCycleAccountSnapshot) RecordPendingOpen(symbol string, side futures.SideType, positionSide futures.PositionSideType, orderID int64) {
	if s == nil {
		return
	}
	key := accountSlotKey(symbol, string(positionSide))
	if !s.openOrderByKey[key] {
		s.openOrderByKey[key] = true
		s.openOrderCount++
	}
	s.openOrders = append(s.openOrders, types.FuturesOrder{
		Symbol:       strings.ToUpper(strings.TrimSpace(symbol)),
		OrderId:      orderID,
		Side:         string(side),
		PositionSide: string(positionSide),
		Status:       "NEW",
	})
}

func accountSlotKey(symbol, side string) string {
	return strings.ToUpper(strings.TrimSpace(symbol)) + "|" + strings.ToUpper(strings.TrimSpace(side))
}

func isOpenOrderStatus(status string) bool {
	status = strings.ToUpper(strings.TrimSpace(status))
	return status == "" || status == "NEW" || status == "PARTIALLY_FILLED"
}

func isOpeningSide(side, positionSide string) bool {
	side = strings.ToUpper(strings.TrimSpace(side))
	positionSide = strings.ToUpper(strings.TrimSpace(positionSide))
	return (positionSide == "LONG" && side == "BUY") || (positionSide == "SHORT" && side == "SELL")
}
