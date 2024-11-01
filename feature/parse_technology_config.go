package feature

import (
	"encoding/json"
	"go_binance_futures/feature/api/binance"
	"go_binance_futures/feature/strategy/line"
	"go_binance_futures/technology"

	"github.com/adshao/go-binance/v2/futures"
	"github.com/beego/beego/v2/core/logs"
)

type Channel struct {
	High []float64
	Low []float64
	Mid []float64
}

func ParseTechnologyConfig(symbol string, strTechnology string) (
	map[string][]float64,
	map[string][]float64,
	map[string][]float64,
	map[string]Channel,
	map[string]Channel) {
	var (
		technologyConfig technology.TechnologyConfig
		ma = make(map[string][]float64)
		ema = make(map[string][]float64)
		rsi = make(map[string][]float64)
		kc = make(map[string]Channel)
		boll = make(map[string]Channel)
	)
	err := json.Unmarshal([]byte(strTechnology), &technologyConfig)
	if err != nil {
		logs.Error("Error unmarshalling JSON:", err.Error())
		return ma, ema, rsi, kc, boll
	}
	
	limit := 150
	klineMap := make(map[string][]*futures.Kline)
	for _, item := range technologyConfig.MA {
		if item.Enable {
			kline, ok := klineMap[item.KlineInterval]
			if !ok {
				kline, err = binance.GetKlineData(symbol, item.KlineInterval, limit)
				if err != nil {
					logs.Error("kline error, symbol:", symbol)
					continue
				}
				klineMap[item.KlineInterval] = kline
			}
			
			close := line.GetLineClosePrices(kline)
			maArr, err := line.CalculateSimpleMovingAverage(close, item.Period)
			if err != nil {
				logs.Error("CalculateSimpleMovingAverage error:", err.Error())
				continue
			}
			ma[item.Name] = maArr
		}
	}
	for _, item := range technologyConfig.EMA {
		if item.Enable {
			kline, ok := klineMap[item.KlineInterval]
			if !ok {
				kline, err = binance.GetKlineData(symbol, item.KlineInterval, limit)
				if err != nil {
					logs.Error("kline error, symbol:", symbol)
					continue
				}
				klineMap[item.KlineInterval] = kline
			}
			
			close := line.GetLineClosePrices(kline)
			emaArr, err := line.CalculateExponentialMovingAverage(close, item.Period)
			if err != nil {
				logs.Error("CalculateExponentialMovingAverage error:", err.Error())
				continue
			}
			ema[item.Name] = emaArr
		}
	}
	for _, item := range technologyConfig.RSI {
		if item.Enable {
			kline, ok := klineMap[item.KlineInterval]
			if !ok {
				kline, err = binance.GetKlineData(symbol, item.KlineInterval, limit)
				if err != nil {
					logs.Error("kline error, symbol:", symbol)
					continue
				}
				klineMap[item.KlineInterval] = kline
			}
			
			close := line.GetLineClosePrices(kline)
			rsiArr, err := line.CalculateRSI(close, item.Period)
			if err != nil {
				logs.Error("CalculateRSI error:", err.Error())
				continue
			}
			rsi[item.Name] = rsiArr
		}
	}
	for _, item := range technologyConfig.KC {
		if item.Enable {
			kline, ok := klineMap[item.KlineInterval]
			if !ok {
				kline, err = binance.GetKlineData(symbol, item.KlineInterval, limit)
				if err != nil {
					logs.Error("kline error, symbol:", symbol)
					continue
				}
				klineMap[item.KlineInterval] = kline
			}
			
			high, low, close := line.GetLineFloatPrices(kline)
			high, low, close = line.CalculateKeltnerChannels(high, low, close, item.Period, item.Multiplier)
			kc[item.Name] = Channel{
				High: high,
				Low: low,
				Mid: close,
			}
		}
	}
	for _, item := range technologyConfig.BOLL {
		if item.Enable {
			kline, ok := klineMap[item.KlineInterval]
			if !ok {
				kline, err = binance.GetKlineData(symbol, item.KlineInterval, limit)
				if err != nil {
					logs.Error("kline error, symbol:", symbol)
					continue
				}
				klineMap[item.KlineInterval] = kline
			}
			
			close := line.GetLineClosePrices(kline)
			up, mid, dn, err := line.CalculateBollingerBands(close, item.Period, item.StdDevMultiplier)
			if err != nil {
				logs.Error("CalculateBollingerBands error:", err.Error())
				continue
			}
			boll[item.Name] = Channel{
				High: up,
				Mid: mid,
				Low: dn,
			}
		}
	}
	
	// logs.Info(utils.ToJson(rsi))
	
	return ma, ema, rsi, kc, boll
}