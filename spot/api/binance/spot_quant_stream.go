package binance

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"go_binance_futures/service/spotexecution"

	sdk "github.com/adshao/go-binance/v2"
	wsrequest "github.com/adshao/go-binance/v2/common/websocket"
	"github.com/beego/beego/v2/core/logs"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// Select Spot WS by THIS adapter's REST environment, not SDK globals.
// Switching the global SDK flag would affect unrelated Spot/Futures code.
const (
	spotQuantTestnetWSURL   = "wss://ws-api.testnet.binance.vision/ws-api/v3"
	spotQuantMainnetWSURL   = "wss://ws-api.binance.com/ws-api/v3"
	spotQuantTestnetRESTURL = "https://testnet.binance.vision"
)

func spotQuantStreamEndpoint(a *QuantAdapter) (string, error) {
	if !a.canMutate() {
		return "", spotexecution.ErrDisabled
	}
	if a.endpoint == spotQuantTestnetRESTURL {
		return spotQuantTestnetWSURL, nil
	}
	return spotQuantMainnetWSURL, nil
}

// decodeSpotQuantMessage recognizes signed WS API subscription ACKs and
// executionReport pushes, including the SDK's supported event envelopes.
func decodeSpotQuantMessage(data []byte) (bool, *sdk.WsOrderUpdate, error) {
	var msg map[string]json.RawMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		return false, nil, err
	}
	if raw, ok := msg["id"]; ok && len(raw) > 0 {
		var response struct {
			Status int `json:"status"`
		}
		if err := json.Unmarshal(data, &response); err != nil {
			return false, nil, err
		}
		if response.Status != 200 {
			return false, nil, fmt.Errorf("Spot WS subscription failed with status %d", response.Status)
		}
		return true, nil, nil
	}
	for _, name := range []string{"event", "result", "data"} {
		if content, ok := msg[name]; ok && len(content) > 0 && content[0] == '{' {
			return decodeSpotQuantMessage(content)
		}
	}
	var kind string
	if err := json.Unmarshal(msg["e"], &kind); err != nil {
		return false, nil, fmt.Errorf("missing Spot WS event type: %w", err)
	}
	if kind != "executionReport" {
		return false, nil, nil
	}
	var report sdk.WsOrderUpdate
	if err := json.Unmarshal(data, &report); err != nil {
		return false, nil, err
	}
	if report.Id <= 0 || report.Symbol == "" {
		return false, nil, fmt.Errorf("Spot WS executionReport missing order identity")
	}
	return false, &report, nil
}

// spotQuantEventProcessor keeps the WS event logic independently testable
// without a real Binance socket, credentials, or main process.
type spotQuantEventProcessor interface {
	HandleExecutionReport(context.Context, spotexecution.ExecutionReport) (bool, error)
	ReconcileByClientID(context.Context, string) error
}

// ApplySpotQuantOrderEvent performs no new order placement. Known
// out-of-order FILLED status can be transient until REST myTrades catches
// up; retain the WebSocket and attempt a read-only reconciliation.
func applySpotQuantOrderEvent(ctx context.Context, p spotQuantEventProcessor, v sdk.WsOrderUpdate) error {
	clientID := v.ClientOrderId
	if v.OrigCustomOrderId != "" {
		clientID = v.OrigCustomOrderId
	}
	report := spotexecution.ExecutionReport{
		Symbol: v.Symbol, ClientOrderID: clientID, Side: v.Side,
		ExchangeOrderID: strconv.FormatInt(v.Id, 10), Status: v.Status,
		ExecutionType: v.ExecutionType, TradeID: strconv.FormatInt(v.TradeId, 10),
		LastQty: v.LatestVolume, LastPrice: v.LatestPrice, LastQuoteQty: v.LatestQuoteVolume,
		Fee: v.FeeCost, FeeAsset: v.FeeAsset, TradeTime: v.TransactionTime,
	}
	handled, err := p.HandleExecutionReport(ctx, report)
	if !handled || err == nil {
		return err
	}
	if errors.Is(err, spotexecution.ErrNeedsReview) {
		reconcileErr := p.ReconcileByClientID(ctx, clientID)
		if reconcileErr != nil && ctx.Err() == nil {
			// Signed SDK errors can contain the REST URL's signature.
			logs.Warn("spot quant user stream requires REST reconciliation: symbol=%s client=%s error_type=%T", v.Symbol, clientID, reconcileErr)
		}
		// No disconnect for expected FILLED-before-myTrades races. The
		// ledger's reconciliation gate still blocks future new orders.
		return nil
	}
	return err
}

// RunSpotQuantUserStream connects to the WS API matching this Spot account.
// It neither switches SDK global endpoints nor starts a background task.
// On connection-level failure the caller must reconcile REST before use.
func (a *QuantAdapter) RunSpotQuantUserStream(ctx context.Context, p *spotexecution.Processor) error {
	return a.runSpotQuantUserStream(ctx, p, nil, nil)
}

// RunSpotQuantUserStreamWithStatus reports the confirmed signed WS ACK,
// then always reports false when disconnected. No order is submitted here.
// The Runner must require ready=true at each new BUY/SELL boundary.
func (a *QuantAdapter) RunSpotQuantUserStreamWithStatus(
	ctx context.Context, p *spotexecution.Processor, onStatus func(bool),
) error {
	if onStatus == nil {
		return fmt.Errorf("Spot user WS readiness callback required")
	}
	defer onStatus(false)
	return a.runSpotQuantUserStream(ctx, p, func() { onStatus(true) }, nil)
}

// The callbacks are optional observer-only test hooks: they never authorize
// trading, and never touch the Testnet/Mainnet endpoint selection. Production
// calls the public method above with both hooks nil.
func (a *QuantAdapter) runSpotQuantUserStream(
	ctx context.Context, p *spotexecution.Processor,
	onSubscribed func(), onExecutionReport func(sdk.WsOrderUpdate),
) error {
	endpoint, err := spotQuantStreamEndpoint(a)
	if err != nil {
		return err
	}
	if p == nil {
		return fmt.Errorf("nil Spot Quant processor")
	}
	if a.c.APIKey == "" || a.c.SecretKey == "" {
		return fmt.Errorf("missing Spot account credentials")
	}
	dialer, err := proxyPool.WebSocketDialer()
	if err != nil {
		return err
	}
	conn, _, err := dialer.DialContext(ctx, endpoint, nil)
	if err != nil {
		return err
	}
	defer conn.Close()
	data := wsrequest.NewRequestData(uuid.NewString(), a.c.APIKey, a.c.SecretKey, a.c.TimeOffset, a.c.KeyType)
	subscription, err := wsrequest.CreateRequest(data, wsrequest.UserDataStreamSubscribeSignatureSpotWsApiMethod, map[string]any{})
	if err != nil {
		return err
	}
	if err = conn.SetWriteDeadline(time.Now().Add(15 * time.Second)); err != nil {
		return err
	}
	if err = conn.WriteMessage(websocket.TextMessage, subscription); err != nil {
		return err
	}
	_ = conn.SetWriteDeadline(time.Time{})
	done := make(chan struct{})
	defer close(done)
	go func() {
		select {
		case <-ctx.Done():
			_ = conn.Close()
		case <-done:
		}
	}()
	// Authentication must be acknowledged before processing any events.
	subscribed := false
	for {
		if !subscribed {
			_ = conn.SetReadDeadline(time.Now().Add(30 * time.Second))
		} else {
			_ = conn.SetReadDeadline(time.Time{})
		}
		_, message, err := conn.ReadMessage()
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			return fmt.Errorf("Spot user stream disconnected; reconcile REST before new signals: %w", err)
		}
		ack, update, err := decodeSpotQuantMessage(message)
		if err != nil {
			return fmt.Errorf("Spot signed WS response: %w", err)
		}
		if ack {
			if !subscribed {
				subscribed = true
				if onSubscribed != nil {
					onSubscribed()
				}
			}
			continue
		}
		if !subscribed {
			return fmt.Errorf("Spot WS received event before signed subscription ACK")
		}
		if update == nil {
			continue
		}
		if err := applySpotQuantOrderEvent(ctx, p, *update); err != nil {
			return fmt.Errorf("Spot WS executionReport failed: %w", err)
		}
		if onExecutionReport != nil {
			onExecutionReport(*update)
		}
	}
}
