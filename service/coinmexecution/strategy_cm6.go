package coinmexecution

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/big"
	"reflect"
	"regexp"
	"strings"
	"time"

	"go_binance_futures/feature/api/binance"
	"go_binance_futures/feature/strategy/line"
	"go_binance_futures/models"
	"go_binance_futures/service/coinmquant"
	"go_binance_futures/service/spotquant"
	strategyservice "go_binance_futures/service/strategy"
	"go_binance_futures/technology"
	"go_binance_futures/utils"

	"github.com/adshao/go-binance/v2/futures"
	"github.com/expr-lang/expr"
)

type CoinMSignalStrategy interface {
	Entry(context.Context, models.DeliverySymbols) (string, error)
	Exit(context.Context, models.DeliverySymbols, coinmquant.PositionSummary) (bool, error)
}
type coinMStrategy struct {
	// Isolated market injection for Fake tests; production uses only the
	// selected COIN-M DAPI and its own public ticker WS.
	Market func(context.Context, models.DeliverySymbols) (map[string]interface{}, error)
	Klines func(context.Context, string, string, int) ([]*futures.Kline, error)
}

func newCoinMStrategy() coinMStrategy {
	return coinMStrategy{Market: coinMVerifiedRuleEnvironment, Klines: binance.CoinMKlinesContext}
}
func coinMRuleMatch(rule coinmquant.Rule, env map[string]interface{}) (bool, error) {
	program, err := strategyservice.CompileRule(strings.TrimSpace(rule.Code), env, true, false)
	if err != nil {
		return false, fmt.Errorf("COIN-M strategy %q: %w", rule.Name, err)
	}
	result, err := expr.Run(program, env)
	if err != nil {
		return false, fmt.Errorf("COIN-M strategy %q: %w", rule.Name, err)
	}
	match, ok := result.(bool)
	if !ok {
		return false, fmt.Errorf("COIN-M strategy did not return boolean")
	}
	return match, nil
}

type coinMPositionCode struct {
	Symbol           string
	Side             string
	Amount           float64 // number of inverse contracts, not base-asset units
	Leverage         int64
	EntryPrice       float64
	MarkPrice        float64
	UnrealizedProfit float64 // in margin asset, not USD
	CreateTime       int64
}

func (e coinMStrategy) Entry(ctx context.Context, coin models.DeliverySymbols) (string, error) {
	rules, err := coinmquant.Rules(coin.Strategy)
	if err != nil {
		return "", err
	}
	hasEntry := false
	for _, r := range rules {
		if r.Enable && (r.Type == "long" || r.Type == "short") {
			hasEntry = true
		}
	}
	if !hasEntry {
		return "", nil
	}
	if e.Market == nil {
		return "", errors.New("COIN-M verified market unavailable")
	}
	env, err := e.Market(ctx, coin)
	if err != nil {
		return "", err
	}
	env["Positions"] = []coinMPositionCode{}
	// Resolve both directions before posting. Ambiguous simultaneous LONG
	// and SHORT signals never cause two positions in this personal system.
	direction := ""
	for _, r := range rules {
		if !r.Enable || (r.Type != "long" && r.Type != "short") {
			continue
		}
		match, e := coinMRuleMatch(r, env)
		if e != nil {
			return "", e
		}
		if !match {
			continue
		}
		side := strings.ToUpper(r.Type)
		if direction != "" && direction != side {
			return "", errors.New("ambiguous COIN-M LONG/SHORT signals")
		}
		direction = side
	}
	return direction, nil
}
func (e coinMStrategy) Exit(ctx context.Context, coin models.DeliverySymbols, summary coinmquant.PositionSummary) (bool, error) {
	if summary.Symbol != coin.Symbol || summary.Status != "OPEN" ||
		summary.PnLStatus != "verified" || summary.UnrealizedPnL == nil || summary.MarkPrice == nil {
		return false, errors.New("COIN-M exit requires verified Managed Fill cycle and fresh mark")
	}
	if (summary.PositionSide != "LONG" && summary.PositionSide != "SHORT") || coin.Leverage < 1 {
		return false, errors.New("invalid COIN-M exit side or leverage")
	}
	remain, err := positive(summary.RemainingContracts)
	if err != nil {
		return false, err
	}
	face, err := positive(summary.ContractSize)
	if err != nil {
		return false, err
	}
	entry, err := positive(summary.EntryPrice)
	if err != nil {
		return false, err
	}
	profit, err := decimal(*summary.UnrealizedPnL)
	if err != nil {
		return false, err
	}
	// This is a configured-leverage STRATEGY TRIGGER, not historical actual
	// Binance ROI: v27 does not persist initial margin/leverage at entry.
	nominal := newRatMul(remain, face)
	margin := newRatDiv(nominal, newRatMul(entry, ratInt(coin.Leverage)))
	roi := newRatMul(newRatDiv(profit, margin), ratInt(100))
	// Match the existing Futures/Spot close contract: CLOSE_LONG and
	// CLOSE_SHORT rules are confirmations AFTER a configured ROI threshold,
	// never unconditional permission to liquidate a healthy position.
	gain, err := coinMCheckedExitThreshold(coin.Profit)
	if err != nil {
		return false, err
	}
	loss, err := coinMCheckedExitThreshold(coin.Loss)
	if err != nil {
		return false, err
	}
	if roi.Cmp(gain) < 0 && roi.Cmp(newRatNeg(loss)) > 0 {
		return false, nil
	}
	if e.Market == nil {
		return false, errors.New("COIN-M verified market unavailable")
	}
	env, err := e.Market(ctx, coin)
	if err != nil {
		return false, err
	}
	price, err := positive(*summary.MarkPrice)
	if err != nil {
		return false, err
	}
	qtyFloat, _ := remain.Float64()
	entryFloat, _ := entry.Float64()
	priceFloat, _ := price.Float64()
	profitFloat, _ := profit.Float64()
	roiFloat, _ := roi.Float64()
	for _, v := range []float64{qtyFloat, entryFloat, priceFloat, profitFloat, roiFloat} {
		if math.IsInf(v, 0) || math.IsNaN(v) {
			return false, errors.New("COIN-M exit numerical overflow")
		}
	}
	pos := coinMPositionCode{Symbol: coin.Symbol, Side: summary.PositionSide, Amount: qtyFloat,
		Leverage: coin.Leverage, EntryPrice: entryFloat, MarkPrice: priceFloat,
		UnrealizedProfit: profitFloat, CreateTime: summary.CreatedAt}
	env["Position"] = pos
	env["Positions"] = []coinMPositionCode{pos}
	env["ROI"] = roiFloat
	rules, err := coinmquant.Rules(coin.Strategy)
	if err != nil {
		return false, err
	}
	wanted := "close_long"
	if summary.PositionSide == "SHORT" {
		wanted = "close_short"
	}
	closeRuleEnabled := false
	for _, r := range rules {
		if !r.Enable || r.Type != wanted {
			continue
		}
		closeRuleEnabled = true
		hit, e := coinMRuleMatch(r, env)
		if e != nil {
			return false, e
		}
		if hit {
			return true, nil
		}
	}
	if closeRuleEnabled {
		return false, nil
	}
	// A missing explicit close rule follows the same threshold-first default
	// as existing Futures. TP/SL Algo orders are maintained independently;
	// unreadable Klines never authorize a close.
	// Match existing Futures simpleCloseStrategy: positive/negative ROE
	// thresholds, a +/-3% neutral band, then two consecutive 5m closes.
	// An invalid DAPI market source NEVER authorizes an exit.
	if roi.Cmp(ratInt(3)) < 0 && roi.Cmp(ratInt(-3)) > 0 {
		return false, nil
	}
	if e.Klines == nil {
		return false, errors.New("COIN-M 5m fallback candle source unavailable")
	}
	bars, err := e.Klines(ctx, coin.Symbol, "5m", 2)
	if err != nil {
		return false, fmt.Errorf("COIN-M 5m close confirmation: %w", err)
	}
	if len(bars) != 2 || bars[0] == nil || bars[1] == nil {
		return false, errors.New("COIN-M 5m close confirmation requires two candles")
	}
	previous, err := positive(bars[0].Close)
	if err != nil {
		return false, err
	}
	latest, err := positive(bars[1].Close)
	if err != nil {
		return false, err
	}
	if summary.PositionSide == "LONG" {
		return previous.Cmp(latest) < 0, nil
	}
	return previous.Cmp(latest) > 0, nil
}

// Zero/empty threshold disables that direction (same sentinel as Spot).
// Invalid negative values fail closed instead of becoming a zero-cost exit.
func coinMCheckedExitThreshold(raw string) (*big.Rat, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "0" {
		return ratInt(1_000_000), nil
	}
	value, err := positive(raw)
	if err != nil {
		return nil, fmt.Errorf("invalid COIN-M close ROI threshold: %w", err)
	}
	return value, nil
}
func ratInt(n int64) *big.Rat          { return new(big.Rat).SetInt64(n) }
func newRatMul(a, b *big.Rat) *big.Rat { return new(big.Rat).Mul(a, b) }
func newRatDiv(a, b *big.Rat) *big.Rat { return new(big.Rat).Quo(a, b) }
func newRatNeg(a *big.Rat) *big.Rat    { return new(big.Rat).Neg(a) }

var coinMKlineRefs = regexp.MustCompile(`\bkline_(1m|3m|5m|15m|30m|1h|2h|4h|6h|8h|12h|1d|3d|1w|1M)\b`)

// CoinMVerifiedRuleEnvironment MUST NOT borrow Spot/FAPI candles or U-M
// indicators; every Kline is fetched from the selected DAPI client.
func coinMVerifiedRuleEnvironment(ctx context.Context, coin models.DeliverySymbols) (map[string]interface{}, error) {
	if ctx == nil || ctx.Err() != nil {
		return nil, errors.New("COIN-M market request cancelled")
	}
	if coin.ContractType != "PERPETUAL" || !strings.HasSuffix(coin.Symbol, "USD_PERP") {
		return nil, errors.New("COIN-M strategy requires USD inverse perpetual")
	}
	ms := coin.UpdateTime
	if ms > 0 && ms < 100000000000 {
		ms *= 1000
	}
	age := time.Now().UnixMilli() - ms
	if ms <= 0 || age < 0 || age > 30000 {
		return nil, errors.New("COIN-M ticker missing or older than 30 seconds")
	}
	tickPrice, e1 := positive(coin.Close)
	open, e2 := positive(coin.Open)
	low, e3 := positive(coin.Low)
	high, e4 := positive(coin.High)
	if e1 != nil || e2 != nil || e3 != nil || e4 != nil || high.Cmp(low) < 0 ||
		tickPrice.Cmp(low) < 0 || tickPrice.Cmp(high) > 0 ||
		math.IsNaN(coin.PercentChange) || math.IsInf(coin.PercentChange, 0) {
		return nil, errors.New("invalid COIN-M ticker OHLC data")
	}
	if strings.TrimSpace(coin.Technology) == "" {
		coin.Technology = "{}"
	}
	if err := spotquant.ValidateTechnology(coin.Technology); err != nil {
		return nil, err
	}
	var cfg technology.TechnologyConfig
	if err := json.Unmarshal([]byte(coin.Technology), &cfg); err != nil {
		return nil, err
	}
	if err := line.ValidateTechnologyConfig(cfg); err != nil {
		return nil, err
	}
	candles := map[string][]*futures.Kline{}
	load := func(symbol, interval string, limit int) ([]*futures.Kline, error) {
		if rows, ok := candles[interval]; ok && len(rows) >= limit {
			return rows, nil
		}
		child, cancel := context.WithTimeout(ctx, 8*time.Second)
		defer cancel()
		rows, err := binance.CoinMKlinesContext(child, symbol, interval, limit)
		if err != nil {
			return nil, err
		}
		if len(rows) < limit {
			return nil, fmt.Errorf("COIN-M %s Klines incomplete: received %d/%d", interval, len(rows), limit)
		}
		candles[interval] = rows
		return rows, nil
	}
	base, err := load(coin.Symbol, "1m", 3)
	if err != nil {
		return nil, err
	}
	baseLast := base[len(base)-1]
	price, err := positive(baseLast.Close)
	if err != nil {
		return nil, err
	}
	ratio := newRatDiv(price, tickPrice)
	if ratio.Cmp(big.NewRat(8, 10)) < 0 || ratio.Cmp(big.NewRat(12, 10)) > 0 {
		return nil, errors.New("COIN-M DAPI candle and local WS ticker disagree")
	}
	indicators, klines := line.ParseTechnologyConfigWithKlineLoader(coin.Symbol, coin.Technology, load)
	// ParseTechnologyConfigWithKlineLoader tolerates missing indicator input
	// for UI previews; live trading must reject every missing active series.
	values := reflect.ValueOf(cfg)
	for i := 0; i < values.NumField(); i++ {
		group := values.Field(i)
		for j := 0; j < group.Len(); j++ {
			entry := group.Index(j).Interface().(technology.IndicatorConfig)
			if !entry.Enable {
				continue
			}
			if entry.Name == "" || strings.HasPrefix(entry.Name, "kline_") {
				return nil, fmt.Errorf("invalid COIN-M indicator name")
			}
			result, ok := indicators[entry.Name]
			if !ok || !coinMIndicatorSamples(result) {
				return nil, fmt.Errorf("COIN-M indicator %q missing valid Klines", entry.Name)
			}
		}
	}
	refs := coinMKlineRefs.FindAllStringSubmatch(coin.Strategy, -1)
	for _, match := range refs {
		interval := match[1]
		if _, ok := klines[interval]; ok {
			continue
		}
		rows, err := load(coin.Symbol, interval, 150)
		if err != nil {
			return nil, err
		}
		klines[interval] = coinMKlinePrice(rows)
	}
	nowPrice, _ := price.Float64()
	tickerOpen, _ := open.Float64()
	tickerLow, _ := low.Float64()
	tickerHigh, _ := high.Float64()
	env := map[string]interface{}{
		"NowPrice": nowPrice, "NowSymbolClose": nowPrice,
		"NowSymbolOpen": tickerOpen, "NowSymbolLow": tickerLow, "NowSymbolHigh": tickerHigh,
		"NowSymbolPercentChange": coin.PercentChange, "NowTime": time.Now().UnixMilli(),
		"KdjSimple": line.KdjSimple, "IsAsc": utils.IsAsc, "IsDesc": utils.IsDesc,
	}
	for k, v := range indicators {
		env[k] = v
	}
	for k, v := range klines {
		env["kline_"+k] = v
	}
	return env, nil
}
func coinMKlinePrice(rows []*futures.Kline) line.KLinePrice {
	high, low, close, open, amount, qps, takerAmount, takerRatio := line.GetLineFloatValues(rows)
	return line.KLinePrice{High: high, Low: low, Close: close, Open: open,
		Amount: amount, Qps: qps, TakerBuyAmount: takerAmount, TakerBuyRatio: takerRatio}
}
func coinMIndicatorSamples(value interface{}) bool {
	checked := func(series ...[]float64) bool {
		if len(series) == 0 || len(series[0]) == 0 {
			return false
		}
		size := len(series[0])
		for _, values := range series {
			if len(values) != size {
				return false
			}
			for _, f := range values {
				if math.IsNaN(f) || math.IsInf(f, 0) {
					return false
				}
			}
		}
		return true
	}
	switch v := value.(type) {
	case line.ConfigData:
		return checked(v.Data) || checked(v.High, v.Mid, v.Low)
	case line.MACDConfigData:
		return checked(v.DIF, v.DEA, v.Histogram)
	case line.ADXConfigData:
		return checked(v.ADX, v.PlusDI, v.MinusDI)
	case line.KDJConfigData:
		return checked(v.K, v.D, v.J)
	case line.SupertrendConfigData:
		return checked(v.Data, v.Trend)
	case line.OBVConfigData:
		return checked(v.Data)
	default:
		return false
	}
}
