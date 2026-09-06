package agenttrade

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	binanceapi "go_binance_futures/feature/api/binance"
	"go_binance_futures/models"
	"go_binance_futures/utils"

	"github.com/adshao/go-binance/v2/futures"
	"github.com/beego/beego/v2/client/orm"
)

type DefaultRiskDataSource struct{}

func (DefaultRiskDataSource) Config(ctx context.Context) (models.Config, error) {
	if err := ctx.Err(); err != nil {
		return models.Config{}, err
	}
	return utils.GetSystemConfig()
}

func (DefaultRiskDataSource) Symbol(ctx context.Context, symbol string) (models.Symbols, error) {
	if err := ctx.Err(); err != nil {
		return models.Symbols{}, err
	}
	var row models.Symbols
	err := orm.NewOrm().QueryTable(new(models.Symbols)).Filter("symbol", strings.ToUpper(strings.TrimSpace(symbol))).One(&row)
	return row, err
}

func (DefaultRiskDataSource) Positions(ctx context.Context) ([]PositionSnapshot, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	rows, err := binanceapi.GetPosition(binanceapi.PositionParams{})
	if err != nil {
		return nil, err
	}
	out := make([]PositionSnapshot, 0, len(rows))
	for _, row := range rows {
		qty, _ := strconv.ParseFloat(row.PositionAmt, 64)
		price, _ := strconv.ParseFloat(row.MarkPrice, 64)
		if qty == 0 {
			continue
		}
		out = append(out, PositionSnapshot{Symbol: row.Symbol, Side: string(row.PositionSide), Quantity: qty, Price: price})
	}
	return out, nil
}

func (DefaultRiskDataSource) OpenOrders(ctx context.Context) ([]OpenOrderSnapshot, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	rows, err := binanceapi.GetOpenOrder()
	if err != nil {
		return nil, err
	}
	out := make([]OpenOrderSnapshot, 0, len(rows))
	for _, row := range rows {
		qty, _ := strconv.ParseFloat(row.OrigQuantity, 64)
		price, _ := strconv.ParseFloat(row.Price, 64)
		if price <= 0 {
			price, _ = strconv.ParseFloat(row.StopPrice, 64)
		}
		out = append(out, OpenOrderSnapshot{
			Symbol: row.Symbol, Side: string(row.Side), PositionSide: string(row.PositionSide),
			Quantity: qty, Price: price,
		})
	}
	return out, nil
}

func (DefaultRiskDataSource) EstimatedFillPrice(ctx context.Context, symbol, side string) (float64, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	bidAverage, askAverage, err := binanceapi.GetDepthAvgPrice(strings.ToUpper(strings.TrimSpace(symbol)), 5)
	if err != nil {
		return 0, err
	}
	if strings.EqualFold(side, "LONG") {
		return askAverage, nil
	}
	if strings.EqualFold(side, "SHORT") {
		return bidAverage, nil
	}
	return 0, fmt.Errorf("unsupported trade side %q", side)
}

type BinanceBroker struct{}

func (BinanceBroker) SubmitMarket(ctx context.Context, request BrokerOrderRequest) (BrokerOrderResult, error) {
	if request.Leverage <= 0 {
		return BrokerOrderResult{}, fmt.Errorf("invalid leverage")
	}
	if _, err := binanceapi.SetLeverage(request.Symbol, request.Leverage); err != nil {
		return BrokerOrderResult{}, fmt.Errorf("set leverage: %w", err)
	}
	var side futures.SideType
	var positionSide futures.PositionSideType
	if strings.EqualFold(request.Side, "LONG") {
		side, positionSide = futures.SideTypeBuy, futures.PositionSideTypeLong
	} else if strings.EqualFold(request.Side, "SHORT") {
		side, positionSide = futures.SideTypeSell, futures.PositionSideTypeShort
	} else {
		return BrokerOrderResult{}, fmt.Errorf("unsupported trade side %q", request.Side)
	}
	order, err := binanceapi.CreateAgentMarketOrder(ctx, request.Symbol, request.Quantity, side, positionSide, request.ClientOrderID)
	if err != nil {
		return BrokerOrderResult{}, err
	}
	avg, _ := strconv.ParseFloat(order.AvgPrice, 64)
	return BrokerOrderResult{ExchangeOrderID: strconv.FormatInt(order.OrderID, 10), ClientOrderID: order.ClientOrderID, AveragePrice: avg}, nil
}

func (BinanceBroker) LookupByClientOrderID(ctx context.Context, symbol, clientOrderID string) (BrokerOrderResult, error) {
	order, err := binanceapi.GetOrderByClientOrderID(ctx, strings.ToUpper(strings.TrimSpace(symbol)), strings.TrimSpace(clientOrderID))
	if err != nil {
		return BrokerOrderResult{}, err
	}
	avg, _ := strconv.ParseFloat(order.AvgPrice, 64)
	return BrokerOrderResult{
		ExchangeOrderID: strconv.FormatInt(order.OrderID, 10),
		ClientOrderID:   order.ClientOrderID,
		AveragePrice:    avg,
	}, nil
}
