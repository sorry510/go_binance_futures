package line

import (
	"testing"

	binance "github.com/adshao/go-binance/v2"
	"github.com/adshao/go-binance/v2/futures"
)

func TestConvertSpotKlinesPreservesMarketFields(t *testing.T) {
	input := []*binance.Kline{{
		OpenTime: 1000, CloseTime: 59999,
		Open: "1.1", High: "2.2", Low: "0.9", Close: "1.8",
		QuoteAssetVolume: "123.4", TakerBuyQuoteAssetVolume: "56.7",
	}}
	result := ConvertSpotKlines(input)
	if len(result) != 1 || result[0].Close != "1.8" ||
		result[0].QuoteAssetVolume != "123.4" || result[0].CloseTime != 59999 {
		t.Fatalf("incorrect spot kline conversion: %+v", result)
	}
}

func TestSpotIndicatorUsesInjectedKlines(t *testing.T) {
	calls := 0
	loadSpot := func(symbol, interval string, limit int) ([]*futures.Kline, error) {
		calls++
		if symbol != "BTCUSDT" || interval != "5m" || limit < 150 {
			t.Fatalf("unexpected spot kline request: %s %s %d", symbol, interval, limit)
		}
		result := make([]*futures.Kline, limit)
		for i := range result {
			result[i] = &futures.Kline{
				Open: "5", High: "6", Low: "4", Close: "5",
				QuoteAssetVolume: "100", OpenTime: int64(i) * 300000,
				CloseTime: int64(i+1)*300000 - 1,
			}
		}
		return result, nil
	}
	config, klines := ParseTechnologyConfigWithKlineLoader("BTCUSDT",
		`{"ma":[{"name":"spot_ma","kline_interval":"5m","period":5,"enable":true}]}`, loadSpot)
	if calls != 1 || len(klines["5m"].Close) < 150 {
		t.Fatalf("spot loader not used: calls=%d kline-count=%d", calls, len(klines["5m"].Close))
	}
	ma, ok := config["spot_ma"].(ConfigData)
	if !ok || len(ma.Data) == 0 {
		t.Fatalf("missing calculated spot MA: %#v", config["spot_ma"])
	}
}
