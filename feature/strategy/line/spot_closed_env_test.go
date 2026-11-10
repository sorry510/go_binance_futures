package line

import (
	"context"
	"testing"
	"time"

	sdk "github.com/adshao/go-binance/v2"
)

func TestS3ClosedSpotEnvironmentNoFuturesVariables(t *testing.T) {
	now := time.Now().UnixMilli()
	candle := &sdk.Kline{OpenTime: now - 60000, CloseTime: now - 1000, Open: "100", High: "120", Low: "90", Close: "110"}
	env, err := InitClosedSpotParseEnv(context.Background(), "BTCUSDT", "{}", candle)
	if err != nil {
		t.Fatal(err)
	}
	if env["NowPrice"] != float64(110) || env["NowTime"] != candle.CloseTime {
		t.Fatalf("wrong closed candle env: %v", env)
	}
	for _, name := range []string{"FundingRate", "FuturesPosition", "PositionROI"} {
		if _, ok := env[name]; ok {
			t.Fatalf("futures variable %s leaked into Spot strategy", name)
		}
	}
	candle.CloseTime = now + 60000
	if _, err = InitClosedSpotParseEnv(context.Background(), "BTCUSDT", "{}", candle); err == nil {
		t.Fatal("future/unclosed Spot Kline accepted")
	}
}
