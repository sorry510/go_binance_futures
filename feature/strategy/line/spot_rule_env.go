package line

import (
	"encoding/json"
	"fmt"
	"math"
	"reflect"
	"strconv"
	"strings"
	"time"

	"go_binance_futures/models"
	"go_binance_futures/service/spotquant"
	spotbinance "go_binance_futures/spot/api/binance"
	"go_binance_futures/technology"
	"go_binance_futures/utils"

	"github.com/adshao/go-binance/v2/futures"
	"github.com/beego/beego/v2/core/config"
)

// LoadVerifiedSpotRuleEnv is the production Spot-only environment for 06-7.
// It deliberately DOES NOT call the lenient InitSpotParseEnv used by the
// legacy monitoring preview: any missing, stale or invalid input blocks a
// live strategy decision instead of becoming zero.
func LoadVerifiedSpotRuleEnv(coin models.SpotSymbols) (map[string]interface{}, error) {
	livePrice, _ := spotbinance.GetFreshSpotTickerPrice(coin.Symbol)
	return buildVerifiedSpotRuleEnv(coin, time.Now(), livePrice, verifiedSpotKlines)
}

// LoadVerifiedSpotRuleEnvWithSource is the same strict Spot environment
// builder used in production, with an explicit market-data source for
// isolated full-chain tests (and an eventual authorized Testnet feed).
// Validation applies to the injected Klines too: no test-only bypass of
// freshness, OHLC validity or indicator warm-up is permitted.
func LoadVerifiedSpotRuleEnvWithSource(
	coin models.SpotSymbols, at time.Time, livePrice string,
	loader func(string, string, int) ([]*futures.Kline, error),
) (map[string]interface{}, error) {
	if loader == nil {
		return nil, fmt.Errorf("Spot market Kline source is required")
	}
	validated := func(symbol, interval string, limit int) ([]*futures.Kline, error) {
		rows, err := loader(symbol, interval, limit)
		if err != nil {
			return nil, err
		}
		if err := verifySpotKlineRows(rows, symbol, interval, at); err != nil {
			return nil, err
		}
		return rows, nil
	}
	return buildVerifiedSpotRuleEnv(coin, at, livePrice, validated)
}

func validSpotNumber(raw string) (float64, error) {
	n, err := strconv.ParseFloat(strings.TrimSpace(raw), 64)
	if err != nil || n <= 0 || math.IsNaN(n) || math.IsInf(n, 0) {
		return 0, fmt.Errorf("invalid positive Spot price")
	}
	return n, nil
}

func validSpotTick(coin models.SpotSymbols, now time.Time, livePrice string) (float64, error) {
	// Spot 24h OHLC/percent fields come from this same local row. Even a
	// fresh WS last price cannot make an hours-old OHLC snapshot safe for
	// expression evaluation, so require a recent local ticker snapshot.
	ms := coin.UpdateTime
	if ms > 0 && ms < 100000000000 {
		ms *= 1000
	}
	age := now.UnixMilli() - ms
	if ms <= 0 || age < 0 || age > int64((5*time.Minute)/time.Millisecond) {
		return 0, fmt.Errorf("Spot ticker %s is missing or stale", coin.Symbol)
	}
	if livePrice != "" {
		price, err := validSpotNumber(livePrice)
		if err != nil {
			return 0, fmt.Errorf("invalid live Spot ticker: %w", err)
		}
		return price, nil
	}
	return validSpotNumber(coin.Close)
}

// A valid indicator must expose every series required by its concrete
// indicator type, not just one nonempty slice from a partially loaded result.
func validSpotSeries(groups ...[]float64) bool {
	if len(groups) == 0 || len(groups[0]) == 0 {
		return false
	}
	expected := len(groups[0])
	for _, values := range groups {
		if len(values) != expected {
			return false
		}
		for _, v := range values {
			if math.IsNaN(v) || math.IsInf(v, 0) {
				return false
			}
		}
	}
	return true
}

func hasSpotIndicatorSamples(value interface{}) bool {
	switch v := value.(type) {
	case ConfigData:
		// MA/EMA/RSI/... have Data; KC/BOLL/Donchian have High/Mid/Low.
		return validSpotSeries(v.Data) || validSpotSeries(v.High, v.Mid, v.Low)
	case MACDConfigData:
		return validSpotSeries(v.DIF, v.DEA, v.Histogram)
	case ADXConfigData:
		return validSpotSeries(v.ADX, v.PlusDI, v.MinusDI)
	case KDJConfigData:
		return validSpotSeries(v.K, v.D, v.J)
	case SupertrendConfigData:
		return validSpotSeries(v.Data, v.Trend)
	case OBVConfigData:
		return validSpotSeries(v.Data)
	default:
		return false
	}
}

func requiredSpotIndicators(config technology.TechnologyConfig) []technology.IndicatorConfig {
	types := reflect.ValueOf(config)
	required := make([]technology.IndicatorConfig, 0)
	for i := 0; i < types.NumField(); i++ {
		group := types.Field(i)
		for j := 0; j < group.Len(); j++ {
			entry := group.Index(j).Interface().(technology.IndicatorConfig)
			if entry.Enable {
				required = append(required, entry)
			}
		}
	}
	return required
}

// Check input candle count, not the output series count: MA(5) needs
// five source candles but may correctly yield only one computed point.
func spotIndicatorMinimumBars(entry technology.IndicatorConfig) int {
	needed := entry.Period
	if entry.FastPeriod > needed {
		needed = entry.FastPeriod
	}
	if entry.SlowPeriod > needed {
		needed = entry.SlowPeriod
	}
	if entry.SlowPeriod > 0 && entry.SignalPeriod > 0 {
		needed = entry.SlowPeriod + entry.SignalPeriod - 1
	}
	if needed < 1 {
		return 1
	}
	return needed
}

func buildVerifiedSpotRuleEnv(
	coin models.SpotSymbols, now time.Time, livePrice string,
	loader func(string, string, int) ([]*futures.Kline, error),
) (map[string]interface{}, error) {
	if coin.Symbol == "" || spotquant.SpotQuoteAsset(coin.Symbol) == "" {
		return nil, fmt.Errorf("unsupported Spot trading pair")
	}
	if loader == nil {
		return nil, fmt.Errorf("Spot Kline loader is missing")
	}
	price, err := validSpotTick(coin, now, livePrice)
	if err != nil {
		return nil, err
	}
	rawTechnology := strings.TrimSpace(coin.Technology)
	if rawTechnology == "" {
		rawTechnology = "{}"
	}
	if err := spotquant.ValidateTechnology(rawTechnology); err != nil {
		return nil, err
	}
	var techConfig technology.TechnologyConfig
	if err := json.Unmarshal([]byte(rawTechnology), &techConfig); err != nil {
		return nil, err
	}
	if err := ValidateTechnologyConfig(techConfig); err != nil {
		return nil, err
	}
	indicators, klines := ParseTechnologyConfigWithKlineLoader(coin.Symbol, rawTechnology, loader)
	for _, entry := range requiredSpotIndicators(techConfig) {
		if entry.Name == "" || spotReservedRuleKey(entry.Name) {
			return nil, fmt.Errorf("Spot indicator has invalid or reserved name %q", entry.Name)
		}
		source, ok := klines[entry.KlineInterval]
		if !ok || len(source.Close) < spotIndicatorMinimumBars(entry) {
			return nil, fmt.Errorf("Spot indicator %q requires more source Klines", entry.Name)
		}
		value, ok := indicators[entry.Name]
		if !ok || !hasSpotIndicatorSamples(value) {
			return nil, fmt.Errorf("Spot indicator %q has insufficient market history", entry.Name)
		}
	}

	openPrice, err := validSpotNumber(coin.Open)
	if err != nil {
		return nil, fmt.Errorf("invalid Spot 24h open: %w", err)
	}
	lowPrice, err := validSpotNumber(coin.Low)
	if err != nil {
		return nil, fmt.Errorf("invalid Spot 24h low: %w", err)
	}
	highPrice, err := validSpotNumber(coin.High)
	if err != nil || highPrice < lowPrice {
		return nil, fmt.Errorf("invalid Spot 24h high")
	}
	if math.IsNaN(coin.PercentChange) || math.IsInf(coin.PercentChange, 0) {
		return nil, fmt.Errorf("invalid Spot 24h percent change")
	}
	systemStartText, _ := config.String("system_start_time")
	systemStartTime, _ := strconv.ParseInt(systemStartText, 10, 64)
	env := map[string]interface{}{
		"SystemStartTime":        systemStartTime,
		"NowTime":                now.UnixMilli(),
		"NowPrice":               price,
		"NowSymbolClose":         price,
		"NowSymbolOpen":          openPrice,
		"NowSymbolLow":           lowPrice,
		"NowSymbolHigh":          highPrice,
		"NowSymbolPercentChange": coin.PercentChange,
		"KdjSimple":              KdjSimple,
		"IsAsc":                  utils.IsAsc,
		"IsDesc":                 utils.IsDesc,
	}
	for key, value := range indicators {
		env[key] = value
	}
	for interval, value := range klines {
		if len(value.Close) == 0 {
			return nil, fmt.Errorf("Spot Kline %s has no prices", interval)
		}
		env["kline_"+interval] = value
	}
	return env, nil
}

func spotReservedRuleKey(name string) bool {
	switch name {
	case "SystemStartTime", "NowTime", "NowPrice", "NowSymbolClose", "NowSymbolOpen",
		"NowSymbolLow", "NowSymbolHigh", "NowSymbolPercentChange",
		"KdjSimple", "IsAsc", "IsDesc", "Position", "Positions",
		"ROI", "FundingRate", "OpenStrategyHash":
		return true
	}
	return strings.HasPrefix(name, "kline_")
}

func spotKlineInterval(interval string) time.Duration {
	switch interval {
	case "1d":
		return 24 * time.Hour
	case "3d":
		return 3 * 24 * time.Hour
	case "1w":
		return 7 * 24 * time.Hour
	case "1M":
		return 31 * 24 * time.Hour
	}
	d, err := time.ParseDuration(interval)
	if err != nil || d <= 0 {
		return 0
	}
	return d
}

// verifiedSpotKlines queries the existing Spot market cache/REST pipeline.
// Futures.Kline is only a reusable OHLC data shape consumed by indicators;
// the source of these candles is strictly spot/api/binance.
func verifiedSpotKlines(symbol, interval string, limit int) ([]*futures.Kline, error) {
	rows, err := spotIndicatorKlines(symbol, interval, limit)
	if err != nil {
		return nil, err
	}
	if err := verifySpotKlineRows(rows, symbol, interval, time.Now()); err != nil {
		return nil, err
	}
	return rows, nil
}

// A recent candle's OPEN timestamp must be within its own interval, not
// twice the interval. Otherwise a month-old market may look fresh for 62 days.
func verifySpotKlineRows(rows []*futures.Kline, symbol, interval string, now time.Time) error {
	if len(rows) == 0 {
		return fmt.Errorf("Spot %s %s history is empty", symbol, interval)
	}
	span := spotKlineInterval(interval)
	if span == 0 {
		return fmt.Errorf("unknown Spot Kline interval %q", interval)
	}
	for i, row := range rows {
		if row == nil {
			return fmt.Errorf("nil Spot Kline at index %d", i)
		}
		for _, value := range []string{row.Open, row.High, row.Low, row.Close} {
			if _, err := validSpotNumber(value); err != nil {
				return fmt.Errorf("invalid Spot %s Kline OHLC: %w", interval, err)
			}
		}
	}
	last := rows[len(rows)-1]
	age := now.Sub(time.UnixMilli(last.OpenTime))
	if last.OpenTime <= 0 || last.CloseTime < last.OpenTime ||
		age < -5*time.Minute || age > span+5*time.Minute {
		return fmt.Errorf("Spot %s %s Kline is stale or inconsistent", symbol, interval)
	}
	return nil
}
