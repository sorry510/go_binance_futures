package binance

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/adshao/go-binance/v2/delivery"
)

// CoinMTradeOrder is a normal (non-conditional) DAPI order in CONTRACTS.
// The service always specifies LONG/SHORT, never switches account mode or
// silently adds reduceOnly in hedge mode.
type CoinMTradeOrder struct {
	Symbol        string
	PositionSide  string
	Side          string
	Type          string
	Contracts     string
	Price         string
	ClientOrderID string
}
type CoinMTrade struct {
	Symbol          string `json:"symbol"`
	TradeID         int64  `json:"id"`
	OrderID         int64  `json:"orderId"`
	Side            string `json:"side"`
	PositionSide    string `json:"positionSide"`
	Price           string `json:"price"`
	Quantity        string `json:"qty"`
	RealizedPnL     string `json:"realizedPnl"`
	MarginAsset     string `json:"marginAsset"`
	Commission      string `json:"commission"`
	CommissionAsset string `json:"commissionAsset"`
	Time            int64  `json:"time"`
}
type CoinMAlgoOrderRequest struct {
	Symbol       string
	PositionSide string
	Side         string
	Type         string // STOP_MARKET or TAKE_PROFIT_MARKET, only
	TriggerPrice string
	ClientAlgoID string
}
type CoinMAlgoOrderResponse struct {
	AlgoID       int64  `json:"algoId"`
	ClientAlgoID string `json:"clientAlgoId"`
	Symbol       string `json:"symbol"`
	Side         string `json:"side"`
	PositionSide string `json:"positionSide"`
	OrderType    string `json:"orderType"`
	AlgoStatus   string `json:"algoStatus"`
}

var coinMClientIDPattern = regexp.MustCompile(`^[a-zA-Z0-9_-]{1,36}$`)
var coinMSymbolPattern = regexp.MustCompile(`^[A-Z0-9]{2,24}USD_PERP$`)

func coinMCheckClientID(id string) error {
	if !coinMClientIDPattern.MatchString(id) {
		return errors.New("COIN-M client order id must use 1-36 safe ASCII characters")
	}
	return nil
}
func coinMCheckSymbol(symbol string) error {
	if !coinMSymbolPattern.MatchString(symbol) {
		return errors.New("COIN-M requires USD perpetual symbol")
	}
	return nil
}
func coinMCheckTradingReadiness() error {
	if err := coinMPrivateReadiness(); err != nil {
		return err
	}
	if deliveryClient.HTTPClient == nil {
		return errors.New("COIN-M signed transport unavailable")
	}
	return nil
}
func GetCoinMPositionModeContext(ctx context.Context) (*delivery.PositionMode, error) {
	return GetDeliveryPositionModeContext(ctx)
}
func CreateCoinMOrderContext(ctx context.Context, order CoinMTradeOrder) (*delivery.CreateOrderResponse, error) {
	if err := coinMCheckTradingReadiness(); err != nil {
		return nil, err
	}
	if err := coinMCheckSymbol(order.Symbol); err != nil {
		return nil, err
	}
	if err := coinMCheckClientID(order.ClientOrderID); err != nil {
		return nil, err
	}
	if order.PositionSide != "LONG" && order.PositionSide != "SHORT" {
		return nil, errors.New("COIN-M requires LONG/SHORT hedge positionSide")
	}
	if order.Side != "BUY" && order.Side != "SELL" {
		return nil, errors.New("invalid COIN-M order side")
	}
	if order.Type != "MARKET" && order.Type != "LIMIT" {
		return nil, errors.New("conditional orders require /dapi/v1/algoOrder")
	}
	if order.Contracts == "" {
		return nil, errors.New("COIN-M contract count required")
	}
	request := deliveryClient.NewCreateOrderService().Symbol(order.Symbol).
		Side(delivery.SideType(order.Side)).PositionSide(delivery.PositionSideType(order.PositionSide)).
		Type(delivery.OrderType(order.Type)).Quantity(order.Contracts).
		NewClientOrderID(order.ClientOrderID).NewOrderResponseType(delivery.NewOrderRespTypeRESULT)
	if order.Type == "LIMIT" {
		if order.Price == "" {
			return nil, errors.New("COIN-M LIMIT price required")
		}
		request.Price(order.Price).TimeInForce(delivery.TimeInForceTypeGTC)
	} else if order.Price != "" {
		return nil, errors.New("COIN-M MARKET cannot specify limit price")
	}
	return request.Do(ctx)
}
func QueryCoinMOrderContext(ctx context.Context, symbol, clientID string) (*delivery.Order, error) {
	if err := coinMCheckTradingReadiness(); err != nil {
		return nil, err
	}
	if err := coinMCheckSymbol(symbol); err != nil {
		return nil, err
	}
	if err := coinMCheckClientID(clientID); err != nil {
		return nil, err
	}
	return deliveryClient.NewGetOrderService().Symbol(symbol).OrigClientOrderID(clientID).Do(ctx)
}
func CancelCoinMOrderContext(ctx context.Context, symbol, clientID string) (*delivery.CancelOrderResponse, error) {
	if err := coinMCheckTradingReadiness(); err != nil {
		return nil, err
	}
	if err := coinMCheckSymbol(symbol); err != nil {
		return nil, err
	}
	if err := coinMCheckClientID(clientID); err != nil {
		return nil, err
	}
	return deliveryClient.NewCancelOrderService().Symbol(symbol).OrigClientOrderID(clientID).Do(ctx)
}

// signedCoinMJSON uses the existing DAPI host-guarded, weight-accounted HTTP
// client for endpoints not implemented by the installed SDK version.
// No client fallback, body logging or external override is permitted.
func signedCoinMJSON(ctx context.Context, method, path string, values url.Values, target any) error {
	if err := coinMCheckTradingReadiness(); err != nil {
		return err
	}
	if ctx == nil {
		return errors.New("nil COIN-M request context")
	}
	switch path {
	case "/dapi/v1/userTrades", "/dapi/v1/algoOrder", "/dapi/v1/openAlgoOrders":
	default:
		return errors.New("COIN-M signed endpoint not allowlisted")
	}
	if values == nil {
		values = make(url.Values)
	}
	values.Set("timestamp", strconv.FormatInt(time.Now().UnixMilli()-deliveryClient.TimeOffset, 10))
	values.Set("recvWindow", "5000")
	unsigned := values.Encode()
	mac := hmac.New(sha256.New, []byte(coinMRuntime.APISecret))
	_, _ = mac.Write([]byte(unsigned))
	values.Set("signature", hex.EncodeToString(mac.Sum(nil)))
	endpoint := strings.TrimRight(coinMRuntime.RESTURL, "/") + path
	var body io.Reader
	if method == http.MethodGet || method == http.MethodDelete {
		endpoint += "?" + values.Encode()
	} else if method == http.MethodPost {
		body = strings.NewReader(values.Encode())
	} else {
		return errors.New("unsupported COIN-M method")
	}
	req, err := http.NewRequestWithContext(ctx, method, endpoint, body)
	if err != nil {
		return err
	}
	req.Header.Set("X-MBX-APIKEY", coinMRuntime.APIKey)
	if method == http.MethodPost {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	res, err := deliveryClient.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("COIN-M %s %s failed: %w", method, path, err)
	}
	defer res.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(res.Body, 4<<20))
	if err != nil {
		return err
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		var failure struct {
			Code int    `json:"code"`
			Msg  string `json:"msg"`
		}
		_ = json.Unmarshal(raw, &failure)
		return fmt.Errorf("COIN-M %s %s rejected: HTTP %d code %d: %s", method, path, res.StatusCode, failure.Code, failure.Msg)
	}
	if target != nil {
		if err := json.Unmarshal(raw, target); err != nil {
			return fmt.Errorf("decode COIN-M %s response: %w", path, err)
		}
	}
	return nil
}
func CoinMUserTradesContext(ctx context.Context, symbol string, orderID int64) ([]CoinMTrade, error) {
	if err := coinMCheckSymbol(symbol); err != nil {
		return nil, err
	}
	if orderID <= 0 {
		return nil, errors.New("COIN-M userTrades requires exact orderID for managed Fill isolation")
	}
	q := url.Values{"symbol": {symbol}, "orderId": {strconv.FormatInt(orderID, 10)}, "limit": {"1000"}}
	var trades []CoinMTrade
	err := signedCoinMJSON(ctx, http.MethodGet, "/dapi/v1/userTrades", q, &trades)
	return trades, err
}
func CreateCoinMAlgoContext(ctx context.Context, r CoinMAlgoOrderRequest) (CoinMAlgoOrderResponse, error) {
	var result CoinMAlgoOrderResponse
	if err := coinMCheckSymbol(r.Symbol); err != nil {
		return result, err
	}
	if err := coinMCheckClientID(r.ClientAlgoID); err != nil {
		return result, err
	}
	if r.PositionSide != "LONG" && r.PositionSide != "SHORT" {
		return result, errors.New("COIN-M protective Algo needs LONG/SHORT hedge positionSide")
	}
	if r.Type != "STOP_MARKET" && r.Type != "TAKE_PROFIT_MARKET" {
		return result, errors.New("COIN-M protection allows only STOP_MARKET and TAKE_PROFIT_MARKET")
	}
	if r.Side != "BUY" && r.Side != "SELL" {
		return result, errors.New("invalid protective Algo side")
	}
	if (r.PositionSide == "LONG" && r.Side != "SELL") || (r.PositionSide == "SHORT" && r.Side != "BUY") {
		return result, errors.New("protective Algo must close, never add exposure")
	}
	if r.TriggerPrice == "" {
		return result, errors.New("protective Algo triggerPrice is required")
	}
	q := url.Values{
		"algoType": {"CONDITIONAL"}, "symbol": {r.Symbol},
		"side": {r.Side}, "positionSide": {r.PositionSide},
		"type": {r.Type}, "triggerPrice": {r.TriggerPrice},
		"clientAlgoId":  {r.ClientAlgoID},
		"closePosition": {"true"}, "workingType": {"MARK_PRICE"},
	}
	err := signedCoinMJSON(ctx, http.MethodPost, "/dapi/v1/algoOrder", q, &result)
	return result, err
}
func CancelCoinMAlgoContext(ctx context.Context, symbol, clientAlgoID string) error {
	if err := coinMCheckSymbol(symbol); err != nil {
		return err
	}
	if err := coinMCheckClientID(clientAlgoID); err != nil {
		return err
	}
	q := url.Values{"symbol": {symbol}, "clientAlgoId": {clientAlgoID}}
	return signedCoinMJSON(ctx, http.MethodDelete, "/dapi/v1/algoOrder", q, nil)
}
func ListOpenCoinMAlgosContext(ctx context.Context, symbol string) ([]CoinMAlgoOrderResponse, error) {
	if err := coinMCheckSymbol(symbol); err != nil {
		return nil, err
	}
	var list []CoinMAlgoOrderResponse
	err := signedCoinMJSON(ctx, http.MethodGet, "/dapi/v1/openAlgoOrders", url.Values{"symbol": {symbol}}, &list)
	return list, err
}
func StartCoinMUserStreamContext(ctx context.Context) (string, error) {
	if err := coinMCheckTradingReadiness(); err != nil {
		return "", err
	}
	return deliveryClient.NewStartUserStreamService().Do(ctx)
}
func KeepaliveCoinMUserStreamContext(ctx context.Context, listenKey string) error {
	if err := coinMCheckTradingReadiness(); err != nil {
		return err
	}
	if listenKey == "" {
		return errors.New("COIN-M listenKey missing")
	}
	return deliveryClient.NewKeepaliveUserStreamService().ListenKey(listenKey).Do(ctx)
}
func CloseCoinMUserStreamContext(ctx context.Context, listenKey string) error {
	if err := coinMCheckTradingReadiness(); err != nil {
		return err
	}
	if listenKey == "" {
		return nil
	}
	return deliveryClient.NewCloseUserStreamService().ListenKey(listenKey).Do(ctx)
}

func GetCoinMOpenOrdersContext(ctx context.Context, symbol string) ([]*delivery.Order, error) {
	if err := coinMCheckTradingReadiness(); err != nil {
		return nil, err
	}
	if err := coinMCheckSymbol(symbol); err != nil {
		return nil, err
	}
	return deliveryClient.NewListOpenOrdersService().Symbol(symbol).Do(ctx)
}

func CoinMUserDataServe(listenKey string, handler delivery.WsUserDataHandler, errHandler delivery.ErrHandler) (chan struct{}, chan struct{}, error) {
	if err := coinMCheckTradingReadiness(); err != nil {
		return nil, nil, err
	}
	if err := validateCoinMWSConfiguration(coinMRuntime); err != nil {
		return nil, nil, err
	}
	if listenKey == "" {
		return nil, nil, errors.New("COIN-M private WS listen key missing")
	}
	return proxyPool.WithDeliveryWS(func() (chan struct{}, chan struct{}, error) {
		return delivery.WsUserDataServe(listenKey, handler, errHandler)
	})
}

// CoinMAlgoDetail includes the exchange-generated normal child order ID only
// once an Algo has triggered; an empty/zero ID is NOT ownership evidence.
type CoinMAlgoDetail struct {
	AlgoID        int64           `json:"algoId"`
	ClientAlgoID  string          `json:"clientAlgoId"`
	Symbol        string          `json:"symbol"`
	Side          string          `json:"side"`
	PositionSide  string          `json:"positionSide"`
	OrderType     string          `json:"orderType"`
	AlgoStatus    string          `json:"algoStatus"`
	ActualOrderID json.RawMessage `json:"actualOrderId"`
}

func QueryCoinMAlgoContext(ctx context.Context, symbol, clientAlgoID string) (CoinMAlgoDetail, error) {
	var detail CoinMAlgoDetail
	if err := coinMCheckSymbol(symbol); err != nil {
		return detail, err
	}
	if err := coinMCheckClientID(clientAlgoID); err != nil {
		return detail, err
	}
	q := url.Values{"symbol": {symbol}, "clientAlgoId": {clientAlgoID}}
	err := signedCoinMJSON(ctx, http.MethodGet, "/dapi/v1/algoOrder", q, &detail)
	return detail, err
}
func QueryCoinMOrderByIDContext(ctx context.Context, symbol string, orderID int64) (*delivery.Order, error) {
	if err := coinMCheckTradingReadiness(); err != nil {
		return nil, err
	}
	if err := coinMCheckSymbol(symbol); err != nil {
		return nil, err
	}
	if orderID <= 0 {
		return nil, errors.New("COIN-M child order ID must be positive")
	}
	return deliveryClient.NewGetOrderService().Symbol(symbol).OrderID(orderID).Do(ctx)
}
