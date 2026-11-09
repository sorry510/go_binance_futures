package binance

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"go_binance_futures/service/binanceapiusage"
	"go_binance_futures/service/spotstrategy"

	sdk "github.com/adshao/go-binance/v2"
)

// S2ReadOnlyClient uses the existing signed Spot client, proxy and REST
// throttling, but deliberately implements NO actual exchange write request.
// Never call the old Buy/Sell constructors from this adapter.
type S2ReadOnlyClient struct{}

var _ spotstrategy.Client = S2ReadOnlyClient{}

func s2ReadContext(ctx context.Context) context.Context {
	return binanceapiusage.WithSource(ctx, "spot_strategy_s2_read")
}
func (S2ReadOnlyClient) Balances(ctx context.Context, account string) (map[string]spotstrategy.Balance, error) {
	if account != spotstrategy.AccountMain {
		return nil, fmt.Errorf("unsupported Spot account")
	}
	res, err := client.NewGetAccountService().Do(s2ReadContext(ctx))
	if err != nil {
		return nil, err
	}
	if res == nil || !res.CanTrade {
		return nil, fmt.Errorf("Spot account trading permission not confirmed")
	}
	out := make(map[string]spotstrategy.Balance, len(res.Balances))
	for _, b := range res.Balances {
		out[b.Asset] = spotstrategy.Balance{Free: b.Free, Locked: b.Locked}
	}
	return out, nil
}
func (S2ReadOnlyClient) Instrument(ctx context.Context, symbol string) (spotstrategy.Instrument, error) {
	if symbol == "" {
		return spotstrategy.Instrument{}, fmt.Errorf("symbol required")
	}
	info, err := loadSpotExchangeInfo(s2ReadContext(ctx), symbol)
	if err != nil {
		return spotstrategy.Instrument{}, err
	}
	for _, s := range info.Symbols {
		if s.Symbol != symbol {
			continue
		}
		out := spotstrategy.Instrument{Symbol: s.Symbol, BaseAsset: s.BaseAsset, QuoteAsset: s.QuoteAsset,
			Status: s.Status, SpotAllowed: s.IsSpotTradingAllowed}
		for _, filter := range s.Filters {
			typ, _ := filter["filterType"].(string)
			str := func(field string) string {
				if v, ok := filter[field]; ok {
					return fmt.Sprint(v)
				}
				return ""
			}
			switch typ {
			case "LOT_SIZE":
				out.LotMin = str("minQty")
				out.LotMax = str("maxQty")
				out.LotStep = str("stepSize")
			case "MARKET_LOT_SIZE":
				out.MarketLotMin = str("minQty")
				out.MarketLotMax = str("maxQty")
				out.MarketLotStep = str("stepSize")
			case "PRICE_FILTER":
				out.PriceMin = str("minPrice")
				out.PriceMax = str("maxPrice")
				out.TickSize = str("tickSize")
			case "MIN_NOTIONAL", "NOTIONAL":
				if out.MinNotional == "" {
					out.MinNotional = str("minNotional")
				}
				if out.MaxNotional == "" {
					out.MaxNotional = str("maxNotional")
				}
			}
		}
		return out, nil
	}
	return spotstrategy.Instrument{}, fmt.Errorf("symbol not returned by Spot exchangeInfo")
}
func sdkToSnapshot(o *sdk.Order) spotstrategy.OrderSnapshot {
	return spotstrategy.OrderSnapshot{Symbol: o.Symbol, OrderID: strconv.FormatInt(o.OrderID, 10),
		ClientOrderID: o.ClientOrderID, Side: string(o.Side), Status: string(o.Status),
		RequestedQty: o.OrigQuantity, ExecutedQty: o.ExecutedQuantity,
		ExecutedQuoteQty: o.CummulativeQuoteQuantity}
}
func (S2ReadOnlyClient) OpenOrders(ctx context.Context, account, symbol string) ([]spotstrategy.OrderSnapshot, error) {
	if account != spotstrategy.AccountMain {
		return nil, fmt.Errorf("unsupported Spot account")
	}
	req := client.NewListOpenOrdersService()
	if symbol != "" {
		req = req.Symbol(symbol)
	}
	rows, err := req.Do(s2ReadContext(ctx))
	if err != nil {
		return nil, err
	}
	out := make([]spotstrategy.OrderSnapshot, 0, len(rows))
	for _, r := range rows {
		if r == nil {
			continue
		}
		out = append(out, sdkToSnapshot(r))
	}
	return out, nil
}
func (S2ReadOnlyClient) QueryOrder(ctx context.Context, account, symbol, clientID, orderID string) (spotstrategy.OrderSnapshot, error) {
	if account != spotstrategy.AccountMain {
		return spotstrategy.OrderSnapshot{}, fmt.Errorf("unsupported Spot account")
	}
	if symbol == "" || clientID == "" {
		return spotstrategy.OrderSnapshot{}, fmt.Errorf("order identity required")
	}
	// Always resolve by persisted origClientOrderID, even when an exchange
	// order ID is known. Looking up solely by orderId could adopt an external
	// order if the locally stored ID is stale/corrupted.
	if orderID != "" {
		id, parseErr := strconv.ParseInt(orderID, 10, 64)
		if parseErr != nil || id <= 0 {
			return spotstrategy.OrderSnapshot{}, spotstrategy.ErrUnmanagedOrder
		}
	}
	req := client.NewGetOrderService().Symbol(symbol).OrigClientOrderID(clientID)
	o, err := req.Do(s2ReadContext(ctx))
	if err != nil {
		return spotstrategy.OrderSnapshot{}, err
	}
	if o == nil || o.Symbol != symbol || o.ClientOrderID != clientID ||
		(orderID != "" && strconv.FormatInt(o.OrderID, 10) != orderID) {
		return spotstrategy.OrderSnapshot{}, spotstrategy.ErrUnmanagedOrder
	}
	return sdkToSnapshot(o), nil
}
func (S2ReadOnlyClient) Trades(ctx context.Context, account, symbol, orderID string, fromID int64, limit int) ([]spotstrategy.Trade, error) {
	if account != spotstrategy.AccountMain {
		return nil, fmt.Errorf("unsupported Spot account")
	}
	id, err := strconv.ParseInt(orderID, 10, 64)
	if err != nil || id <= 0 || symbol == "" || limit <= 0 || limit > 1000 {
		return nil, fmt.Errorf("invalid Spot trades scope")
	}
	req := client.NewListTradesService().Symbol(symbol).OrderId(id).Limit(limit)
	if fromID > 0 {
		req = req.FromID(fromID)
	}
	rows, err := req.Do(s2ReadContext(ctx))
	if err != nil {
		return nil, err
	}
	out := make([]spotstrategy.Trade, 0, len(rows))
	for _, v := range rows {
		if v == nil {
			continue
		}
		side := "SELL"
		if v.IsBuyer {
			side = "BUY"
		}
		out = append(out, spotstrategy.Trade{Symbol: v.Symbol, OrderID: strconv.FormatInt(v.OrderID, 10),
			TradeID: strconv.FormatInt(v.ID, 10), Side: side, Qty: v.Quantity,
			Price: v.Price, QuoteQty: v.QuoteQuantity, Commission: v.Commission,
			CommissionAsset: strings.ToUpper(v.CommissionAsset), Time: v.Time})
	}
	return out, nil
}
func (S2ReadOnlyClient) Place(context.Context, spotstrategy.OrderRequest) (spotstrategy.OrderSnapshot, error) {
	return spotstrategy.OrderSnapshot{}, spotstrategy.ErrLiveWriteDisabled
}
func (S2ReadOnlyClient) Cancel(context.Context, string, string, string) error {
	return spotstrategy.ErrLiveWriteDisabled
}
