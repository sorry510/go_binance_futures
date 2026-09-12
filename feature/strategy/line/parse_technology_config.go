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

const (
	// Keep the historical 150-bar baseline so existing EMA/Wilder-based strategy values
	// do not change merely because their mathematical minimum is smaller. The value is
	// now a floor, not a hard cap: indicators with longer warmup automatically request more.
	defaultStrategyKlineLimit      = 150
	strategyIndicatorOutputReserve = 32
	// Binance Futures Kline REST supports at most 1500 rows per request.
	maxStrategyKlineLimit = 1500
)

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

	limit := technologyKlineLimit(technologyConfig)
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
	"NowSymbolLow": {}, "NowSymbolHigh": {},
	"KdjSimple": {}, "IsAsc": {}, "IsDesc": {}, "ROI": {}, "Position": {}, "Positions": {},
}

func indicatorRequiredKlines(indicatorType string, item technology.IndicatorConfig) int {
	reserve := strategyIndicatorOutputReserve
	switch indicatorType {
	case "macd":
		// MACD output length is N-slow-signal+2.
		return item.SlowPeriod + item.SignalPeriod + reserve - 2
	case "rsi", "roc", "mfi":
		// These outputs have length N-period.
		return item.Period + reserve
	case "adx":
		// ADX output length is N-2*period+1.
		return 2*item.Period + reserve - 1
	case "obv":
		return reserve
	default:
		// MA/EMA/CCI/KDJ/KC/BOLL/Donchian/ATR/Supertrend outputs are N-period+1.
		return item.Period + reserve - 1
	}
}

func technologyIndicatorGroups(config technology.TechnologyConfig) []struct {
	name  string
	items []technology.IndicatorConfig
} {
	return []struct {
		name  string
		items []technology.IndicatorConfig
	}{
		{name: "ma", items: config.MA},
		{name: "ema", items: config.EMA},
		{name: "macd", items: config.MACD},
		{name: "adx", items: config.ADX},
		{name: "mfi", items: config.MFI},
		{name: "obv", items: config.OBV},
		{name: "cci", items: config.CCI},
		{name: "roc", items: config.ROC},
		{name: "kdj", items: config.KDJ},
		{name: "rsi", items: config.RSI},
		{name: "kc", items: config.KC},
		{name: "boll", items: config.BOLL},
		{name: "donchian", items: config.Donchian},
		{name: "atr", items: config.ATR},
		{name: "supertrend", items: config.Supertrend},
	}
}

func technologyKlineLimit(config technology.TechnologyConfig) int {
	limit := defaultStrategyKlineLimit
	for _, group := range technologyIndicatorGroups(config) {
		for _, item := range group.items {
			if !item.Enable {
				continue
			}
			required := indicatorRequiredKlines(group.name, item)
			if required > limit {
				limit = required
			}
		}
	}
	if limit > maxStrategyKlineLimit {
		return maxStrategyKlineLimit
	}
	return limit
}

// ValidateTechnologyConfig validates enabled indicator settings without loading market data.
func ValidateTechnologyConfig(config technology.TechnologyConfig) error {
	usedNames := make(map[string]struct{})
	indicatorGroups := technologyIndicatorGroups(config)

	for _, group := range indicatorGroups {
		for _, item := range group.items {
			if !item.Enable {
				continue
			}
			if err := validateIndicatorConfig("", group.name, item, maxStrategyKlineLimit, usedNames); err != nil {
				return err
			}
		}
	}

	return nil
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
		required := indicatorRequiredKlines(indicatorType, item)
		if required > maxPeriod {
			return fmt.Errorf("MACD indicator %q requires %d K-lines (including %d recent outputs), maximum is %d", name, required, strategyIndicatorOutputReserve, maxPeriod)
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
	required := indicatorRequiredKlines(indicatorType, item)
	if required > maxPeriod {
		return fmt.Errorf("%s indicator %q requires %d K-lines (including %d recent outputs), maximum is %d", strings.ToUpper(indicatorType), name, required, strategyIndicatorOutputReserve, maxPeriod)
	}
	usedNames[name] = struct{}{}
	return nil
}

func InitParseEnv(symbol string, strTechnology string) map[string]interface{} {
	o := orm.NewOrm()
	var symbols []models.Symbols

	_, err := o.QueryTable("symbols").Filter("symbol", symbol).All(&symbols)
	if err != nil {
		logs.Error("error", err.Error())
	}

	marketConditionStr, _ := config.String("MarketCondition")
	systemStartTimeText, _ := config.String("system_start_time")
	systemStartTime, _ := strconv.ParseInt(systemStartTimeText, 10, 64)
	tConfig, klineMap := ParseTechnologyConfig(symbol, strTechnology)
	env := map[string]interface{}{
		"SystemStartTime": systemStartTime,
		"MarketCondition": marketConditionStr,
		"NowTime":         time.Now().Unix() * 1000,
		"KdjSimple":       KdjSimple,
		"IsAsc":           utils.IsAsc,
		"IsDesc":          utils.IsDesc,
	}

	for _, v := range symbols {
		closePrice, _ := strconv.ParseFloat(v.Close, 64)
		openPrice, _ := strconv.ParseFloat(v.Open, 64)
		lowPrice, _ := strconv.ParseFloat(v.Low, 64)
		highPrice, _ := strconv.ParseFloat(v.High, 64)
		env["NowPrice"] = closePrice
		env["NowSymbolPercentChange"] = v.PercentChange
		env["NowSymbolClose"] = closePrice
		env["NowSymbolOpen"] = openPrice
		env["NowSymbolLow"] = lowPrice
		env["NowSymbolHigh"] = highPrice
	}

	for k, v := range tConfig {
		env[k] = v
	}
	for k, v := range klineMap {
		env["kline_"+k] = v
	}
	return env
}
