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

// Never use sdk.WsUserDataServeSignature here: the SDK chooses its WS host
// using GLOBAL sdk.UseTestnet/sdK.UseDemo, not the injected Client.BaseURL.
// Switching sdk.UseTestnet globally would also redirect unrelated Spot code.
const spotQuantTestnetWSURL = "wss://ws-api.testnet.binance.vision/ws-api/v3"
const spotQuantTestnetRESTURL = "https://testnet.binance.vision"

func spotQuantStreamEndpoint(a *QuantAdapter) (string, error) {
	if a == nil || a.c == nil || !a.allowOrderMutation ||
		a.c.BaseURL != spotQuantTestnetRESTURL {
		return "", spotexecution.ErrDisabled
	}
	return spotQuantTestnetWSURL, nil
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
		if reconcileErr != nil {
			logs.Warn("spot quant user stream requires REST reconciliation: symbol=%s client=%s: %v", v.Symbol, clientID, reconcileErr)
		}
		// No disconnect for expected FILLED-before-myTrades races. The
		// ledger's reconciliation gate still blocks future new orders.
		return nil
	}
	return err
}

// RunSpotQuantUserStream connects ONLY to the fixed Spot Testnet WS API.
// It neither switches SDK global endpoints nor starts a background task.
// On connection-level failure the caller must reconcile REST before use.
func (a *QuantAdapter) RunSpotQuantUserStream(ctx context.Context, p *spotexecution.Processor) error {
	endpoint, err := spotQuantStreamEndpoint(a)
	if err != nil {
		return err
	}
	if p == nil {
		return fmt.Errorf("nil Spot Quant processor")
	}
	if a.c.APIKey == "" || a.c.SecretKey == "" {
		return fmt.Errorf("missing Spot Testnet credentials")
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
			return fmt.Errorf("Spot Testnet user stream disconnected; reconcile REST before new signals: %w", err)
		}
		ack, update, err := decodeSpotQuantMessage(message)
		if err != nil {
			return fmt.Errorf("Spot Testnet signed WS response: %w", err)
		}
		if ack {
			subscribed = true
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
	}
}
