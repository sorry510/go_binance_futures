package binance

import (
	"context"
	"errors"
	"testing"

	"go_binance_futures/service/spotstrategy"
)

func TestS3RealSpotSDKWritesDeniedByDefault(t *testing.T) {
	t.Setenv("SPOT_STRATEGY_LIVE_TRADING_APPROVAL", "")
	gate := S3LiveClient{}
	request := spotstrategy.OrderRequest{AccountID: "main", Owner: "auto_strategy",
		Symbol: "BTCUSDT", Side: "BUY", Type: "LIMIT", Quantity: "1", Price: "100",
		ClientOrderID: "s3-live-should-block"}
	if _, err := gate.Place(context.Background(), request); !errors.Is(err, spotstrategy.ErrLiveWriteDisabled) {
		t.Fatalf("real Spot CREATE not blocked without independent approval: %v", err)
	}
	if err := gate.Cancel(context.Background(), "main", "BTCUSDT", request.ClientOrderID); !errors.Is(err, spotstrategy.ErrLiveWriteDisabled) {
		t.Fatalf("real Spot CANCEL not blocked without independent approval: %v", err)
	}
}
