package line

import (
	"encoding/json"
	"fmt"
	"go_binance_futures/feature/api/binance"
	"go_binance_futures/models"
	"go_binance_futures/technology"
	"go_binance_futures/utils"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/adshao/go-binance/v2/futures"
	"github.com/beego/beego/v2/client/orm"
	"github.com/beego/beego/v2/core/config"
	"github.com/beego/beego/v2/core/logs"
)

type ConfigData struct {
	KlineInterval    string    `json:"kline_interval"`               // K线周期
	Period           int       `json:"period"`                       // 周期
	Multiplier       float64   `json:"multiplier,omitempty"`         // 可选字段
	StdDevMultiplier float64   `json:"std_dev_multiplier,omitempty"` // 可选字段
	Data             []float64 `json:"data,omitempty"`               // 可选字段
	High             []float64 `json:"high,omitempty"`               // 可选字段
	Low              []float64 `json:"low,omitempty"`                // 可选字段
	Mid              []float64 `json:"mid,omitempty"`                // 可选字段
}

type MACDConfigData struct {
	KlineInterval string    `json:"kline_interval"` // K-line interval
	FastPeriod    int       `json:"fast_period"`    // Fast EMA period
	SlowPeriod    int       `json:"slow_period"`    // Slow EMA period
	SignalPeriod  int       `json:"signal_period"`  // Signal EMA period
	DIF           []float64 `json:"dif"`            // Fast EMA minus slow EMA
	DEA           []float64 `json:"dea"`            // EMA of DIF
	Histogram     []float64 `json:"histogram"`      // DIF minus DEA
}

type ADXConfigData struct {
	KlineInterval string    `json:"kline_interval"` // K-line interval
	Period        int       `json:"period"`         // Wilder smoothing period
	ADX           []float64 `json:"adx"`            // Trend strength
	PlusDI        []float64 `json:"plus_di"`        // Positive directional indicator
	MinusDI       []float64 `json:"minus_di"`       // Negative directional indicator
}

type KDJConfigData struct {
	KlineInterval string    `json:"kline_interval"` // K-line interval
	Period        int       `json:"period"`         // RSV lookback period
	KPeriod       int       `json:"k_period"`       // K smoothing period
	DPeriod       int       `json:"d_period"`       // D smoothing period
	K             []float64 `json:"k"`              // Smoothed RSV
	D             []float64 `json:"d"`              // Smoothed K
	J             []float64 `json:"j"`              // Three K minus two D
}

type SupertrendConfigData struct {
	KlineInterval string    `json:"kline_interval"` // K-line interval
	Period        int       `json:"period"`         // ATR period
	Multiplier    float64   `json:"multiplier"`     // ATR multiplier
	Data          []float64 `json:"data"`           // Active trend line
	Trend         []float64 `json:"trend"`          // One for bullish and minus one for bearish
}

type OBVConfigData struct {
	KlineInterval string    `json:"kline_interval"` // K-line interval
	Data          []float64 `json:"data"`           // On-balance volume
}

type KLinePrice struct {
	High   []float64 `json:"high"`   // 最高价
	Low    []float64 `json:"low"`    // 最低价
	Close  []float64 `json:"close"`  // 收盘价
	Open   []float64 `json:"open"`   // 开盘价
	Amount []float64 `json:"amount"` // 成交额(成交量 * 平均价格)
	Qps    []float64 `json:"qps"`    // 每秒成交额
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
	usedIndicatorNames := make(map[string]struct{})
	for _, item := range technologyConfig.MA {
		if item.Enable {
			if err := validateIndicatorConfig(symbol, "ma", item, limit, usedIndicatorNames); err != nil {
				logs.Error("invalid MA config:", err.Error())
				continue
			}
			klinePrice, ok := klineMap[item.KlineInterval]
			if !ok {
				kline, err := binance.GetKlineData(symbol, item.KlineInterval, limit)
				if err != nil {
					logs.Error("kline error, symbol:", symbol)
					logs.Error("kline error in ParseTechnologyConfig:", err.Error())
					continue
				}
				klinePrice = newKLinePrice(kline)
				klineMap[item.KlineInterval] = klinePrice
			}
			maArr, err := CalculateSimpleMovingAverage(klinePrice.Close, item.Period)
			if err != nil {
				logs.Error("CalculateSimpleMovingAverage error:", err.Error())
				continue
			}
			config[item.Name] = ConfigData{
				KlineInterval: item.KlineInterval,
				Period:        item.Period,
				Data:          maArr,
			}
		}
	}
	for _, item := range technologyConfig.EMA {
		if item.Enable {
			if err := validateIndicatorConfig(symbol, "ema", item, limit, usedIndicatorNames); err != nil {
				logs.Error("invalid EMA config:", err.Error())
				continue
			}
			klinePrice, ok := klineMap[item.KlineInterval]
			if !ok {
				kline, err := binance.GetKlineData(symbol, item.KlineInterval, limit)
				if err != nil {
					logs.Error("kline error, symbol:", symbol)
					logs.Error("kline error in ParseTechnologyConfig:", err.Error())
					continue
				}
				klinePrice = newKLinePrice(kline)
				klineMap[item.KlineInterval] = klinePrice
			}

			emaArr, err := CalculateExponentialMovingAverage(klinePrice.Close, item.Period)
			if err != nil {
				logs.Error("CalculateExponentialMovingAverage error:", err.Error())
				continue
			}
			config[item.Name] = ConfigData{
				KlineInterval: item.KlineInterval,
				Period:        item.Period,
				Data:          emaArr,
			}
		}
	}
	for _, item := range technologyConfig.MACD {
		if item.Enable {
			if err := validateIndicatorConfig(symbol, "macd", item, limit, usedIndicatorNames); err != nil {
				logs.Error("invalid MACD config:", err.Error())
				continue
			}
			klinePrice, ok := klineMap[item.KlineInterval]
			if !ok {
				kline, err := binance.GetKlineData(symbol, item.KlineInterval, limit)
				if err != nil {
					logs.Error("kline error, symbol:", symbol)
					logs.Error("kline error in ParseTechnologyConfig:", err.Error())
					continue
				}
				klinePrice = newKLinePrice(kline)
				klineMap[item.KlineInterval] = klinePrice
			}
			dif, dea, histogram, err := CalculateMACD(klinePrice.Close, item.FastPeriod, item.SlowPeriod, item.SignalPeriod)
			if err != nil {
				logs.Error("CalculateMACD error:", err.Error())
				continue
			}
			config[item.Name] = MACDConfigData{
				KlineInterval: item.KlineInterval,
				FastPeriod:    item.FastPeriod,
				SlowPeriod:    item.SlowPeriod,
				SignalPeriod:  item.SignalPeriod,
				DIF:           dif,
				DEA:           dea,
				Histogram:     histogram,
			}
		}
	}
	for _, item := range technologyConfig.RSI {
		if item.Enable {
			if err := validateIndicatorConfig(symbol, "rsi", item, limit, usedIndicatorNames); err != nil {
				logs.Error("invalid RSI config:", err.Error())
				continue
			}
			klinePrice, ok := klineMap[item.KlineInterval]
			if !ok {
				kline, err := binance.GetKlineData(symbol, item.KlineInterval, limit)
				if err != nil {
					logs.Error("kline error, symbol:", symbol)
					logs.Error("kline error in ParseTechnologyConfig:", err.Error())
					continue
				}
				klinePrice = newKLinePrice(kline)
				klineMap[item.KlineInterval] = klinePrice
			}
			rsiArr, err := CalculateRSI(klinePrice.Close, item.Period)
			if err != nil {
				logs.Error("CalculateRSI error:", err.Error())
				continue
			}
			config[item.Name] = ConfigData{
				KlineInterval: item.KlineInterval,
				Period:        item.Period,
				Data:          rsiArr,
			}
		}
	}
	for _, item := range technologyConfig.ROC {
		if item.Enable {
			if err := validateIndicatorConfig(symbol, "roc", item, limit, usedIndicatorNames); err != nil {
				logs.Error("invalid ROC config:", err.Error())
				continue
			}
			klinePrice, ok := klineMap[item.KlineInterval]
			if !ok {
				kline, err := binance.GetKlineData(symbol, item.KlineInterval, limit)
				if err != nil {
					logs.Error("kline error, symbol:", symbol)
					logs.Error("kline error in ParseTechnologyConfig:", err.Error())
					continue
				}
				klinePrice = newKLinePrice(kline)
				klineMap[item.KlineInterval] = klinePrice
			}
			roc, err := CalculateROC(klinePrice.Close, item.Period)
			if err != nil {
				logs.Error("CalculateROC error:", err.Error())
				continue
			}
			config[item.Name] = ConfigData{
				KlineInterval: item.KlineInterval,
				Period:        item.Period,
				Data:          roc,
			}
		}
	}
	for _, item := range technologyConfig.MFI {
		if item.Enable {
			if err := validateIndicatorConfig(symbol, "mfi", item, limit, usedIndicatorNames); err != nil {
				logs.Error("invalid MFI config:", err.Error())
				continue
			}
			klinePrice, ok := klineMap[item.KlineInterval]
			if !ok {
				kline, err := binance.GetKlineData(symbol, item.KlineInterval, limit)
				if err != nil {
					logs.Error("kline error, symbol:", symbol)
					logs.Error("kline error in ParseTechnologyConfig:", err.Error())
					continue
				}
				klinePrice = newKLinePrice(kline)
				klineMap[item.KlineInterval] = klinePrice
			}
			mfi, err := CalculateMFI(klinePrice.High, klinePrice.Low, klinePrice.Close, klinePrice.Amount, item.Period)
			if err != nil {
				logs.Error("CalculateMFI error:", err.Error())
				continue
			}
			config[item.Name] = ConfigData{
				KlineInterval: item.KlineInterval,
				Period:        item.Period,
				Data:          mfi,
			}
		}
	}
	for _, item := range technologyConfig.OBV {
		if item.Enable {
			if err := validateIndicatorConfig(symbol, "obv", item, limit, usedIndicatorNames); err != nil {
				logs.Error("invalid OBV config:", err.Error())
				continue
			}
			klinePrice, ok := klineMap[item.KlineInterval]
			if !ok {
				kline, err := binance.GetKlineData(symbol, item.KlineInterval, limit)
				if err != nil {
					logs.Error("kline error, symbol:", symbol)
					logs.Error("kline error in ParseTechnologyConfig:", err.Error())
					continue
				}
				klinePrice = newKLinePrice(kline)
				klineMap[item.KlineInterval] = klinePrice
			}
			obv, err := CalculateOBV(klinePrice.Close, klinePrice.Amount)
			if err != nil {
				logs.Error("CalculateOBV error:", err.Error())
				continue
			}
			config[item.Name] = OBVConfigData{
				KlineInterval: item.KlineInterval,
				Data:          obv,
			}
		}
	}
	for _, item := range technologyConfig.CCI {
		if item.Enable {
			if err := validateIndicatorConfig(symbol, "cci", item, limit, usedIndicatorNames); err != nil {
				logs.Error("invalid CCI config:", err.Error())
				continue
			}
			klinePrice, ok := klineMap[item.KlineInterval]
			if !ok {
				kline, err := binance.GetKlineData(symbol, item.KlineInterval, limit)
				if err != nil {
					logs.Error("kline error, symbol:", symbol)
					logs.Error("kline error in ParseTechnologyConfig:", err.Error())
					continue
				}
				klinePrice = newKLinePrice(kline)
				klineMap[item.KlineInterval] = klinePrice
			}
			cci, err := CalculateCCI(klinePrice.High, klinePrice.Low, klinePrice.Close, item.Period)
			if err != nil {
				logs.Error("CalculateCCI error:", err.Error())
				continue
			}
			config[item.Name] = ConfigData{
				KlineInterval: item.KlineInterval,
				Period:        item.Period,
				Data:          cci,
			}
		}
	}
	for _, item := range technologyConfig.KDJ {
		if item.Enable {
			if err := validateIndicatorConfig(symbol, "kdj", item, limit, usedIndicatorNames); err != nil {
				logs.Error("invalid KDJ config:", err.Error())
				continue
			}
			klinePrice, ok := klineMap[item.KlineInterval]
			if !ok {
				kline, err := binance.GetKlineData(symbol, item.KlineInterval, limit)
				if err != nil {
					logs.Error("kline error, symbol:", symbol)
					logs.Error("kline error in ParseTechnologyConfig:", err.Error())
					continue
				}
				klinePrice = newKLinePrice(kline)
				klineMap[item.KlineInterval] = klinePrice
			}
			k, d, j, err := Kdj(klinePrice.High, klinePrice.Low, klinePrice.Close, item.Period, item.KPeriod, item.DPeriod)
			if err != nil {
				logs.Error("Kdj error:", err.Error())
				continue
			}
			config[item.Name] = KDJConfigData{
				KlineInterval: item.KlineInterval,
				Period:        item.Period,
				KPeriod:       item.KPeriod,
				DPeriod:       item.DPeriod,
				K:             k,
				D:             d,
				J:             j,
			}
		}
	}
	for _, item := range technologyConfig.KC {
		if item.Enable {
			if err := validateIndicatorConfig(symbol, "kc", item, limit, usedIndicatorNames); err != nil {
				logs.Error("invalid KC config:", err.Error())
				continue
			}
			klinePrice, ok := klineMap[item.KlineInterval]
			if !ok {
				kline, err := binance.GetKlineData(symbol, item.KlineInterval, limit)
				if err != nil {
					logs.Error("kline error, symbol:", symbol)
					logs.Error("kline error in ParseTechnologyConfig:", err.Error())
					continue
				}
				klinePrice = newKLinePrice(kline)
				klineMap[item.KlineInterval] = klinePrice
			}

			high, mid, low, err := CalculateKeltnerChannels(klinePrice.High, klinePrice.Low, klinePrice.Close, item.Period, item.Multiplier)
			if err != nil {
				logs.Error("CalculateKeltnerChannels error:", err.Error())
				continue
			}

			config[item.Name] = ConfigData{
				KlineInterval: item.KlineInterval,
				Period:        item.Period,
				Multiplier:    item.Multiplier,
				High:          high,
				Low:           low,
				Mid:           mid,
			}
		}
	}
	for _, item := range technologyConfig.BOLL {
		if item.Enable {
			if err := validateIndicatorConfig(symbol, "boll", item, limit, usedIndicatorNames); err != nil {
				logs.Error("invalid BOLL config:", err.Error())
				continue
			}
			klinePrice, ok := klineMap[item.KlineInterval]
			if !ok {
				kline, err := binance.GetKlineData(symbol, item.KlineInterval, limit)
				if err != nil {
					logs.Error("kline error, symbol:", symbol)
					logs.Error("kline error in ParseTechnologyConfig:", err.Error())
					continue
				}
				klinePrice = newKLinePrice(kline)
				klineMap[item.KlineInterval] = klinePrice
			}

			up, mid, dn, err := CalculateBollingerBands(klinePrice.Close, item.Period, item.StdDevMultiplier)
			if err != nil {
				logs.Error("CalculateBollingerBands error:", err.Error())
				continue
			}
			config[item.Name] = ConfigData{
				KlineInterval:    item.KlineInterval,
				Period:           item.Period,
				Multiplier:       item.Multiplier,
				StdDevMultiplier: item.StdDevMultiplier,
				High:             up,
				Mid:              mid,
				Low:              dn,
			}
		}
	}
	for _, item := range technologyConfig.Donchian {
		if item.Enable {
			if err := validateIndicatorConfig(symbol, "donchian", item, limit, usedIndicatorNames); err != nil {
				logs.Error("invalid Donchian config:", err.Error())
				continue
			}
			klinePrice, ok := klineMap[item.KlineInterval]
			if !ok {
				kline, err := binance.GetKlineData(symbol, item.KlineInterval, limit)
				if err != nil {
					logs.Error("kline error, symbol:", symbol)
					logs.Error("kline error in ParseTechnologyConfig:", err.Error())
					continue
				}
				klinePrice = newKLinePrice(kline)
				klineMap[item.KlineInterval] = klinePrice
			}
			upper, middle, lower, err := CalculateDonchianChannels(klinePrice.High, klinePrice.Low, item.Period)
			if err != nil {
				logs.Error("CalculateDonchianChannels error:", err.Error())
				continue
			}
			config[item.Name] = ConfigData{
				KlineInterval: item.KlineInterval,
				Period:        item.Period,
				High:          upper,
				Mid:           middle,
				Low:           lower,
			}
		}
	}
	for _, item := range technologyConfig.ATR {
		if item.Enable {
			if err := validateIndicatorConfig(symbol, "atr", item, limit, usedIndicatorNames); err != nil {
				logs.Error("invalid ATR config:", err.Error())
				continue
			}
			klinePrice, ok := klineMap[item.KlineInterval]
			if !ok {
				kline, err := binance.GetKlineData(symbol, item.KlineInterval, limit)
				if err != nil {
					logs.Error("kline error, symbol:", symbol)
					logs.Error("kline error in ParseTechnologyConfig:", err.Error())
					continue
				}
				klinePrice = newKLinePrice(kline)
				klineMap[item.KlineInterval] = klinePrice
			}
			atrArr, err := CalculateAtr(klinePrice.High, klinePrice.Low, klinePrice.Close, item.Period)
			if err != nil {
				logs.Error("CalculateAtr error:", err.Error())
				continue
			}
			config[item.Name] = ConfigData{
				KlineInterval: item.KlineInterval,
				Period:        item.Period,
				Data:          atrArr,
			}
		}
	}
	for _, item := range technologyConfig.Supertrend {
		if item.Enable {
			if err := validateIndicatorConfig(symbol, "supertrend", item, limit, usedIndicatorNames); err != nil {
				logs.Error("invalid Supertrend config:", err.Error())
				continue
			}
			klinePrice, ok := klineMap[item.KlineInterval]
			if !ok {
				kline, err := binance.GetKlineData(symbol, item.KlineInterval, limit)
				if err != nil {
					logs.Error("kline error, symbol:", symbol)
					logs.Error("kline error in ParseTechnologyConfig:", err.Error())
					continue
				}
				klinePrice = newKLinePrice(kline)
				klineMap[item.KlineInterval] = klinePrice
			}
			data, trend, err := CalculateSupertrend(klinePrice.High, klinePrice.Low, klinePrice.Close, item.Period, item.Multiplier)
			if err != nil {
				logs.Error("CalculateSupertrend error:", err.Error())
				continue
			}
			config[item.Name] = SupertrendConfigData{
				KlineInterval: item.KlineInterval,
				Period:        item.Period,
				Multiplier:    item.Multiplier,
				Data:          data,
				Trend:         trend,
			}
		}
	}
	for _, item := range technologyConfig.ADX {
		if item.Enable {
			if err := validateIndicatorConfig(symbol, "adx", item, limit, usedIndicatorNames); err != nil {
				logs.Error("invalid ADX config:", err.Error())
				continue
			}
			klinePrice, ok := klineMap[item.KlineInterval]
			if !ok {
				kline, err := binance.GetKlineData(symbol, item.KlineInterval, limit)
				if err != nil {
					logs.Error("kline error, symbol:", symbol)
					logs.Error("kline error in ParseTechnologyConfig:", err.Error())
					continue
				}
				klinePrice = newKLinePrice(kline)
				klineMap[item.KlineInterval] = klinePrice
			}
			adx, plusDI, minusDI, err := CalculateADX(klinePrice.High, klinePrice.Low, klinePrice.Close, item.Period)
			if err != nil {
				logs.Error("CalculateADX error:", err.Error())
				continue
			}
			config[item.Name] = ADXConfigData{
				KlineInterval: item.KlineInterval,
				Period:        item.Period,
				ADX:           adx,
				PlusDI:        plusDI,
				MinusDI:       minusDI,
			}
		}
	}

	return config, klineMap
}

func newKLinePrice(kline []*futures.Kline) KLinePrice {
	high, low, close, open, amount, qps := GetLineFloatValues(kline)
	return KLinePrice{
		High:   high,
		Low:    low,
		Close:  close,
		Open:   open,
		Amount: amount,
		Qps:    qps,
	}
}

var indicatorNamePattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

var supportedKlineIntervals = map[string]struct{}{
	"1m": {}, "3m": {}, "5m": {}, "15m": {}, "30m": {},
	"1h": {}, "2h": {}, "4h": {}, "6h": {}, "8h": {}, "12h": {},
	"1d": {}, "3d": {}, "1w": {}, "1M": {},
}

var reservedIndicatorNames = map[string]struct{}{
	"SystemStartTime": {}, "MarketCondition": {}, "NowTime": {}, "NowPrice": {},
	"NowSymbolPercentChange": {}, "NowSymbolClose": {}, "NowSymbolOpen": {},
	"NowSymbolLow": {}, "NowSymbolHigh": {}, "BasicTrend": {},
	"KdjSimple": {}, "IsAsc": {}, "IsDesc": {}, "ROI": {}, "Position": {}, "Positions": {},
	"BTCUSDT": {}, "ETHUSDT": {}, "SOLUSDT": {}, "BNBUSDT": {},
}

func validateIndicatorConfig(symbol, indicatorType string, item technology.IndicatorConfig, maxPeriod int, usedNames map[string]struct{}) error {
	name := strings.TrimSpace(item.Name)
	if name == "" {
		return fmt.Errorf("%s indicator name must not be empty", indicatorType)
	}
	if name != item.Name || !indicatorNamePattern.MatchString(name) {
		return fmt.Errorf("%s indicator name %q must be a valid expression identifier without surrounding spaces", indicatorType, item.Name)
	}
	if _, reserved := reservedIndicatorNames[name]; reserved || name == symbol || strings.HasPrefix(name, "kline_") {
		return fmt.Errorf("%s indicator name %q is reserved", indicatorType, name)
	}
	if _, exists := usedNames[name]; exists {
		return fmt.Errorf("indicator name %q must be unique", name)
	}
	if _, supported := supportedKlineIntervals[item.KlineInterval]; !supported {
		return fmt.Errorf("%s indicator %q has unsupported K-line interval %q", indicatorType, name, item.KlineInterval)
	}
	if indicatorType == "macd" {
		if item.FastPeriod <= 0 || item.SlowPeriod <= 0 || item.SignalPeriod <= 0 {
			return fmt.Errorf("MACD indicator %q periods must be greater than zero", name)
		}
		if item.FastPeriod >= item.SlowPeriod {
			return fmt.Errorf("MACD indicator %q fast period must be less than slow period", name)
		}
		if item.SlowPeriod+item.SignalPeriod-1 > maxPeriod {
			return fmt.Errorf("MACD indicator %q requires more than %d K-lines", name, maxPeriod)
		}
		usedNames[name] = struct{}{}
		return nil
	}
	if indicatorType == "obv" {
		usedNames[name] = struct{}{}
		return nil
	}
	if item.Period <= 0 || item.Period > maxPeriod {
		return fmt.Errorf("%s indicator %q period must be between 1 and %d", indicatorType, name, maxPeriod)
	}
	if indicatorType == "rsi" && item.Period >= maxPeriod {
		return fmt.Errorf("RSI indicator %q period must be between 1 and %d", name, maxPeriod-1)
	}
	if indicatorType == "mfi" && item.Period >= maxPeriod {
		return fmt.Errorf("MFI indicator %q period must be between 1 and %d", name, maxPeriod-1)
	}
	if indicatorType == "roc" && item.Period >= maxPeriod {
		return fmt.Errorf("ROC indicator %q period must be between 1 and %d", name, maxPeriod-1)
	}
	if indicatorType == "kdj" && (item.KPeriod <= 0 || item.KPeriod > maxPeriod || item.DPeriod <= 0 || item.DPeriod > maxPeriod) {
		return fmt.Errorf("KDJ indicator %q smoothing periods must be between 1 and %d", name, maxPeriod)
	}
	if indicatorType == "adx" && item.Period > maxPeriod/2 {
		return fmt.Errorf("ADX indicator %q period must be between 1 and %d", name, maxPeriod/2)
	}
	if indicatorType == "kc" && item.Multiplier < 0 {
		return fmt.Errorf("KC indicator %q multiplier must not be negative", name)
	}
	if indicatorType == "supertrend" && item.Multiplier <= 0 {
		return fmt.Errorf("Supertrend indicator %q multiplier must be greater than zero", name)
	}
	if indicatorType == "boll" && item.StdDevMultiplier < 0 {
		return fmt.Errorf("BOLL indicator %q standard deviation multiplier must not be negative", name)
	}
	usedNames[name] = struct{}{}
	return nil
}

func InitParseEnv(symbol string, strTechnology string) map[string]interface{} {
	o := orm.NewOrm()
	var symbols []models.Symbols

	sql := "SELECT * FROM symbols WHERE symbol = ? OR symbol = ? OR symbol = ? OR symbol = ? OR symbol = ?"
	_, err := o.Raw(sql, "BTCUSDT", "ETHUSDT", "SOLUSDT", "BNBUSDT", symbol).QueryRows(&symbols)
	if err != nil {
		logs.Error("error", err.Error())
	}

	// resPrice, _ := binance.GetTickerPrice(symbol)
	// nowPrice, _ := strconv.ParseFloat(resPrice[0].Price, 64)
	marketConditionStr, _ := config.String("MarketCondition")
	system_start_time_str, _ := config.String("system_start_time")
	system_start_time_int, _ := strconv.ParseInt(system_start_time_str, 10, 64)
	tConfig, klineMap := ParseTechnologyConfig(symbol, strTechnology)
	env := map[string]interface{}{
		// build-in
		// "NowPrice": nowPrice, // 当前价格
		"SystemStartTime": system_start_time_int,    // 系统启动时间, 毫秒时间戳
		"MarketCondition": marketConditionStr,       // 当前市场行情趋势
		"NowTime":         time.Now().Unix() * 1000, // 毫秒时间戳

		// function
		"KdjSimple":  KdjSimple,    // 计算是否是金叉,
		"IsAsc":      utils.IsAsc,  // 是否是升序数组
		"IsDesc":     utils.IsDesc, // 是否是降序数组,
		"BasicTrend": 0.0,          // 基础趋势涨跌幅 (btc * 0.6 + eth * 0.3 + sol * 0.05 + bnb * 0.05)
	}
	basicTrend := 0.0

	for _, v := range symbols {
		if v.Symbol == "BTCUSDT" {
			basicTrend += v.PercentChange * 0.6
		} else if v.Symbol == "ETHUSDT" {
			basicTrend += v.PercentChange * 0.3
		} else if v.Symbol == "SOLUSDT" {
			basicTrend += v.PercentChange * 0.05
		} else if v.Symbol == "BNBUSDT" {
			basicTrend += v.PercentChange * 0.05
		}
		close, _ := strconv.ParseFloat(v.Close, 64)
		open, _ := strconv.ParseFloat(v.Open, 64)
		low, _ := strconv.ParseFloat(v.Low, 64)
		high, _ := strconv.ParseFloat(v.High, 64)
		item := map[string]interface{}{
			"PercentChange": v.PercentChange,
			"Close":         close,
			"Open":          open,
			"Low":           low,
			"High":          high,
		}
		env[v.Symbol] = item
		if v.Symbol == symbol {
			env["NowPrice"] = close                         // 当前价格
			env["NowSymbolPercentChange"] = v.PercentChange // 当前涨跌幅
			env["NowSymbolClose"] = close
			env["NowSymbolOpen"] = open
			env["NowSymbolLow"] = low
			env["NowSymbolHigh"] = high
		}
	}
	env["BasicTrend"] = basicTrend

	// technology
	for k, v := range tConfig {
		env[k] = v
	}

	// kline data
	for k, v := range klineMap {
		env["kline_"+k] = v
	}

	// logs.Info(utils.ToJson(klineMap))
	return env
}
