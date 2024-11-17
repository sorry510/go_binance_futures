package line

import (
	"encoding/json"
	"go_binance_futures/feature/api/binance"
	"go_binance_futures/models"
	"go_binance_futures/technology"
	"go_binance_futures/utils"
	"strconv"
	"time"

	"github.com/beego/beego/v2/client/orm"
	"github.com/beego/beego/v2/core/logs"
)

type ConfigData struct {
	KlineInterval string  `json:"kline_interval"` // K线周期
	Period        int     `json:"period"` // 周期
	Multiplier    float64 `json:"multiplier,omitempty"` // 可选字段
	StdDevMultiplier float64 `json:"std_dev_multiplier,omitempty"` // 可选字段
	Data []float64 `json:"data,omitempty"` // 可选字段
	High []float64 `json:"high,omitempty"` // 可选字段
	Low []float64 `json:"low,omitempty"` // 可选字段
	Mid []float64 `json:"mid,omitempty"` // 可选字段
}

type KLinePrice struct {
	High []float64 `json:"high"` // 最高价
	Low []float64 `json:"low"` // 最低价
	Close []float64 `json:"close"` // 收盘价
	Open []float64 `json:"open"` // 开盘价
}

func ParseTechnologyConfig(symbol string, strTechnology string) (config map[string]interface{}, klineMap map[string]KLinePrice) {
	var (
		technologyConfig technology.TechnologyConfig
	)
	config = make(map[string]interface{})
	klineMap = make(map[string]KLinePrice)
	err := json.Unmarshal([]byte(strTechnology), &technologyConfig)
	if err != nil {
		logs.Error("Error unmarshalling JSON:", err.Error())
		return config, klineMap
	}
	
	limit := 150
	for _, item := range technologyConfig.MA {
		if item.Enable {
			klinePrice, ok := klineMap[item.KlineInterval]
			if !ok {
				kline, err := binance.GetKlineData(symbol, item.KlineInterval, limit)
				if err != nil {
					logs.Error("kline error, symbol:", symbol)
					continue
				}
				high, low, close, open := GetLineFloatPrices(kline)
				klinePrice = KLinePrice{
					High: high,
					Low: low,
					Close: close,
					Open: open,
				}
				klineMap[item.KlineInterval] = klinePrice
			}
			maArr, err := CalculateSimpleMovingAverage(klinePrice.Close, item.Period)
			if err != nil {
				logs.Error("CalculateSimpleMovingAverage error:", err.Error())
				continue
			}
			config[item.Name] = ConfigData{
				KlineInterval: item.KlineInterval,
				Period: item.Period,
				Data: maArr,
			}
		}
	}
	for _, item := range technologyConfig.EMA {
		if item.Enable {
			klinePrice, ok := klineMap[item.KlineInterval]
			if !ok {
				kline, err := binance.GetKlineData(symbol, item.KlineInterval, limit)
				if err != nil {
					logs.Error("kline error, symbol:", symbol)
					continue
				}
				high, low, close, open := GetLineFloatPrices(kline)
				klinePrice = KLinePrice{
					High: high,
					Low: low,
					Close: close,
					Open: open,
				}
				klineMap[item.KlineInterval] = klinePrice
			}
			
			emaArr, err := CalculateExponentialMovingAverage(klinePrice.Close, item.Period)
			if err != nil {
				logs.Error("CalculateExponentialMovingAverage error:", err.Error())
				continue
			}
			config[item.Name] = ConfigData{
				KlineInterval: item.KlineInterval,
				Period: item.Period,
				Data: emaArr,
			}
		}
	}
	for _, item := range technologyConfig.RSI {
		if item.Enable {
			klinePrice, ok := klineMap[item.KlineInterval]
			if !ok {
				kline, err := binance.GetKlineData(symbol, item.KlineInterval, limit)
				if err != nil {
					logs.Error("kline error, symbol:", symbol)
					continue
				}
				high, low, close, open := GetLineFloatPrices(kline)
				klinePrice = KLinePrice{
					High: high,
					Low: low,
					Close: close,
					Open: open,
				}
				klineMap[item.KlineInterval] = klinePrice
			}
			rsiArr, err := CalculateRSI(klinePrice.Close, item.Period)
			if err != nil {
				logs.Error("CalculateRSI error:", err.Error())
				continue
			}
			config[item.Name] = ConfigData{
				KlineInterval: item.KlineInterval,
				Period: item.Period,
				Data: rsiArr,
			}
		}
	}
	for _, item := range technologyConfig.KC {
		if item.Enable {
			klinePrice, ok := klineMap[item.KlineInterval]
			if !ok {
				kline, err := binance.GetKlineData(symbol, item.KlineInterval, limit)
				if err != nil {
					logs.Error("kline error, symbol:", symbol)
					continue
				}
				high, low, close, open := GetLineFloatPrices(kline)
				klinePrice = KLinePrice{
					High: high,
					Low: low,
					Close: close,
					Open: open,
				}
				klineMap[item.KlineInterval] = klinePrice
			}
			
			high, mid, low := CalculateKeltnerChannels(klinePrice.High, klinePrice.Low, klinePrice.Close, item.Period, item.Multiplier)
			
			config[item.Name] = ConfigData{
				KlineInterval: item.KlineInterval,
				Period: item.Period,
				Multiplier: item.Multiplier,
				High: high,
				Low: low,
				Mid: mid,
			}
		}
	}
	for _, item := range technologyConfig.BOLL {
		if item.Enable {
			klinePrice, ok := klineMap[item.KlineInterval]
			if !ok {
				kline, err := binance.GetKlineData(symbol, item.KlineInterval, limit)
				if err != nil {
					logs.Error("kline error, symbol:", symbol)
					continue
				}
				high, low, close, open := GetLineFloatPrices(kline)
				klinePrice = KLinePrice{
					High: high,
					Low: low,
					Close: close,
					Open: open,
				}
				klineMap[item.KlineInterval] = klinePrice
			}
			
			up, mid, dn, err := CalculateBollingerBands(klinePrice.Close, item.Period, item.StdDevMultiplier)
			if err != nil {
				logs.Error("CalculateBollingerBands error:", err.Error())
				continue
			}
			config[item.Name] = ConfigData{
				KlineInterval: item.KlineInterval,
				Period: item.Period,
				Multiplier: item.Multiplier,
				StdDevMultiplier: item.StdDevMultiplier,
				High: up,
				Mid: mid,
				Low: dn,
			}
		}
	}
	for _, item := range technologyConfig.ATR {
		if item.Enable {
			klinePrice, ok := klineMap[item.KlineInterval]
			if !ok {
				kline, err := binance.GetKlineData(symbol, item.KlineInterval, limit)
				if err != nil {
					logs.Error("kline error, symbol:", symbol)
					continue
				}
				high, low, close, open := GetLineFloatPrices(kline)
				klinePrice = KLinePrice{
					High: high,
					Low: low,
					Close: close,
					Open: open,
				}
				klineMap[item.KlineInterval] = klinePrice
			}
			atrArr, err := CalculateAtr(klinePrice.High, klinePrice.Low, klinePrice.Close, item.Period)
			if err != nil {
				logs.Error("CalculateAtr error:", err.Error())
				continue
			}
			config[item.Name] = ConfigData{
				KlineInterval: item.KlineInterval,
				Period: item.Period,
				Data: atrArr,
			}
		}
	}
	
	return config, klineMap
}

func InitParseEnv(symbol string, strTechnology string) (map[string]interface{}) {
	o := orm.NewOrm()
	var symbols []models.Symbols
	
	sql := "SELECT * FROM symbols WHERE symbol = ? OR symbol = ? OR symbol = ?"
	_, err := o.Raw(sql, "BTCUSDT", "ETHUSDT", symbol).QueryRows(&symbols)
	if err != nil {
		logs.Error("error", err.Error())
	}
	
	// resPrice, _ := binance.GetTickerPrice(symbol)
	// nowPrice, _ := strconv.ParseFloat(resPrice[0].Price, 64)
	config, klineMap := ParseTechnologyConfig(symbol, strTechnology)
	env := map[string]interface{} {
		// build-in
		// "NowPrice": nowPrice, // 当前价格
		"NowTime": time.Now().Unix() * 1000, // 毫秒时间戳
		
		// function
		"Kdj": Kdj, // 计算是否是金叉,
		"IsAsc": utils.IsAsc, // 是否是升序数组
		"IsDesc": utils.IsDesc, // 是否是降序数组,
	}
	
	for _, v := range symbols {
		close, _ := strconv.ParseFloat(v.Close, 64)
		open, _ := strconv.ParseFloat(v.Open, 64)
		low, _ := strconv.ParseFloat(v.Low, 64)
		high, _ := strconv.ParseFloat(v.High, 64)
		item := map[string]interface{}{
			"PercentChange": v.PercentChange,
			"Close": close,
			"Open": open,
			"Low": low,
			"High": high,
		}
		env[v.Symbol] = item
		if (v.Symbol == symbol) {
			env["NowPrice"] = close // 当前价格
			env["NowSymbolPercentChange"] = v.PercentChange // 当前涨跌幅
			env["NowSymbolClose"] = close
			env["NowSymbolOpen"] = open
			env["NowSymbolLow"] = low
			env["NowSymbolHigh"] = high
		}
	}
	
	// technology
	for k, v := range config {
		env[k] = v
	}
	
	// kline data
	for k, v := range klineMap {
		env["kline_" + k] = v
	}
	
	// logs.Info(utils.ToJson(klineMap))
	return env
}