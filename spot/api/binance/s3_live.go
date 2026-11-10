package binance

import (
	"context"
	"fmt"
	"strconv"

	"go_binance_futures/service/binanceapiusage"
	"go_binance_futures/service/spotstrategy"

	sdk "github.com/adshao/go-binance/v2"
)

// S3LiveClient shares the signed Spot SDK and the existing REST limiter. It
// has real POST/DELETE implementations, but S1 mode reconciliation + the
// separate explicit runtime approval are enforced at the HTTP write boundary.
// Neither is authorized by changing the SpotSymbols.Enable flag.
type S3LiveClient struct{ S2ReadOnlyClient }

var _ spotstrategy.Client = S3LiveClient{}

func (S3LiveClient) SpotSignedRESTProvenance() bool {
	// Only checks configured target endpoint for the production adapter.
	// This is NOT response-level authentication. Every S3 Fill still retains
	// rest_verified=0; no Holding can be created from it without stronger
	// response-bound provenance and independent release approval.
	return client != nil && client.BaseURL == sdk.BaseAPIMainURL
}

func (g S3LiveClient) Place(ctx context.Context, req spotstrategy.OrderRequest) (spotstrategy.OrderSnapshot, error) {
	if !g.SpotSignedRESTProvenance() {
		return spotstrategy.OrderSnapshot{}, spotstrategy.ErrLiveWriteDisabled
	}
	if err := spotstrategy.AuthorizeLiveWrite(ctx, req, false); err != nil {
		return spotstrategy.OrderSnapshot{}, err
	}
	if req.Side != "BUY" && req.Side != "SELL" {
		return spotstrategy.OrderSnapshot{}, fmt.Errorf("invalid Spot side")
	}
	if req.Type != "MARKET" && req.Type != "LIMIT" {
		return spotstrategy.OrderSnapshot{}, fmt.Errorf("invalid Spot order type")
	}
	return s3SendCreate(ctx, client.NewCreateOrderService(), req)
}

// s3SendCreate contains the exact SDK signed POST used by Place. The
// production caller MUST pass AuthorizeLiveWrite before constructing the
// service; tests exercise this one shared implementation with a no-network
// RoundTripper, rather than duplicating Binance request builders.
func s3SendCreate(ctx context.Context, svc *sdk.CreateOrderService, req spotstrategy.OrderRequest) (spotstrategy.OrderSnapshot, error) {
	reqSvc := svc.Symbol(req.Symbol).
		Side(sdk.SideType(req.Side)).Type(sdk.OrderType(req.Type)).
		Quantity(req.Quantity).NewClientOrderID(req.ClientOrderID).
		NewOrderRespType(sdk.NewOrderRespTypeRESULT)
	if req.Type == "LIMIT" {
		if req.Price == "" {
			return spotstrategy.OrderSnapshot{}, fmt.Errorf("Spot LIMIT price required")
		}
		reqSvc = reqSvc.Price(req.Price).TimeInForce(sdk.TimeInForceTypeGTC)
	}
	resp, err := reqSvc.Do(binanceapiusage.WithSource(ctx, "spot_strategy_s3_real_create"))
	if err != nil {
		return spotstrategy.OrderSnapshot{}, err
	}
	if resp == nil || resp.OrderID <= 0 || resp.ClientOrderID != req.ClientOrderID ||
		resp.Symbol != req.Symbol || string(resp.Side) != req.Side {
		return spotstrategy.OrderSnapshot{}, spotstrategy.ErrReconciliation
	}
	return spotstrategy.OrderSnapshot{Symbol: resp.Symbol, OrderID: strconv.FormatInt(resp.OrderID, 10),
		ClientOrderID: resp.ClientOrderID, Side: string(resp.Side), Status: string(resp.Status),
		RequestedQty: resp.OrigQuantity, ExecutedQty: resp.ExecutedQuantity,
		ExecutedQuoteQty: resp.CummulativeQuoteQuantity}, nil
}

func (g S3LiveClient) Cancel(ctx context.Context, account, symbol, clientID string) error {
	if !g.SpotSignedRESTProvenance() {
		return spotstrategy.ErrLiveWriteDisabled
	}
	req := spotstrategy.OrderRequest{AccountID: account, Owner: spotstrategy.OwnerAuto,
		Symbol: symbol, ClientOrderID: clientID}
	if err := spotstrategy.AuthorizeLiveWrite(ctx, req, true); err != nil {
		return err
	}
	return s3SendCancel(ctx, client.NewCancelOrderService(), symbol, clientID)
}

// s3SendCancel is the signed DELETE path shared with the offline transport
// regression. The only production callsite is the ownership-gated Cancel.
func s3SendCancel(ctx context.Context, svc *sdk.CancelOrderService, symbol, clientID string) error {
	resp, err := svc.Symbol(symbol).
		OrigClientOrderID(clientID).Do(binanceapiusage.WithSource(ctx, "spot_strategy_s3_real_cancel"))
	if err != nil {
		return err
	}
	if resp == nil || resp.Symbol != symbol || resp.OrderID <= 0 || resp.OrigClientOrderID != clientID {
		return spotstrategy.ErrReconciliation
	}
	return nil
}
