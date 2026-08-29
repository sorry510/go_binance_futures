package market

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"go_binance_futures/models"
	markettypes "go_binance_futures/types"

	"github.com/beego/beego/v2/client/orm"
)

const regimeListLimit = 8

type RegimeSnapshot struct {
	AsOf                      string         `json:"as_of"`
	SymbolCount               int            `json:"symbol_count"`
	AdvancingCount            int            `json:"advancing_count"`
	DecliningCount            int            `json:"declining_count"`
	UnchangedCount            int            `json:"unchanged_count"`
	AdvancingRatio            float64        `json:"advancing_ratio"`
	DecliningRatio            float64        `json:"declining_ratio"`
	AverageChange             float64        `json:"average_change_pct"`
	MedianChange              float64        `json:"median_change_pct"`
	ChangeStdDev              float64        `json:"change_std_dev"`
	AverageRange              float64        `json:"average_range_pct"`
	MajorWeightedChange       float64        `json:"major_weighted_change_pct"`
	QuoteVolumeWeightedChange float64        `json:"quote_volume_weighted_change_pct"`
	MajorSymbols              []RegimeSymbol `json:"major_symbols"`
	TopGainers                []RegimeSymbol `json:"top_gainers"`
	TopLosers                 []RegimeSymbol `json:"top_losers"`
}
type RegimeSymbol struct {
	Symbol        string  `json:"symbol"`
	PercentChange float64 `json:"percent_change_pct"`
	Range         float64 `json:"range_pct,omitempty"`
	QuoteVolume   float64 `json:"quote_volume,omitempty"`
}

func LoadRegimeSymbols(ctx context.Context) ([]models.Symbols, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	var symbols []models.Symbols
	_, err := orm.NewOrm().QueryTable("symbols").Filter("type", "USDT").All(&symbols)
	if err != nil {
		return nil, fmt.Errorf("load market condition symbols: %w", err)
	}
	if len(symbols) <= 4 {
		return nil, fmt.Errorf("market condition requires more than four USDT symbols")
	}
	return symbols, nil
}

func CalculateAlgorithmCondition(symbols []models.Symbols) (int, error) {
	otherLengths := len(symbols) - 4
	if otherLengths <= 0 {
		return 0, fmt.Errorf("market condition requires more than four symbols")
	}
	otherWeight := 0.35 / float64(otherLengths)
	var weightedSum float64
	for _, symbol := range symbols {
		switch symbol.Symbol {
		case "BTCUSDT":
			weightedSum += symbol.PercentChange * 0.35
		case "ETHUSDT":
			weightedSum += symbol.PercentChange * 0.2
		case "SOLUSDT", "BNBUSDT":
			weightedSum += symbol.PercentChange * 0.05
		default:
			weightedSum += symbol.PercentChange * otherWeight
		}
	}
	m := math.Tanh(weightedSum)
	switch {
	case m >= 0.45:
		return markettypes.MarketConditionStrongBull, nil
	case m >= 0.2:
		return markettypes.MarketConditionBull, nil
	case m >= -0.2:
		return markettypes.MarketConditionSideways, nil
	case m >= -0.45:
		return markettypes.MarketConditionBear, nil
	default:
		return markettypes.MarketConditionStrongBear, nil
	}
}

func BuildRegimeSnapshot(symbols []models.Symbols) RegimeSnapshot {
	items := make([]RegimeSymbol, 0, len(symbols))
	changes := make([]float64, 0, len(symbols))
	majorWeights := map[string]float64{"BTCUSDT": 0.5, "ETHUSDT": 0.3, "SOLUSDT": 0.1, "BNBUSDT": 0.1}
	var advancingCount, decliningCount, unchangedCount int
	var rangeSum float64
	var rangeCount int
	var majorWeightedChange, majorWeightSum float64
	var volumeWeightedChange, quoteVolumeSum float64

	for _, symbol := range symbols {
		rangePercent := regimeRangePercent(symbol)
		item := RegimeSymbol{Symbol: symbol.Symbol, PercentChange: symbol.PercentChange, Range: rangePercent, QuoteVolume: symbol.QuoteVolume}
		items = append(items, item)
		changes = append(changes, symbol.PercentChange)
		switch {
		case symbol.PercentChange > 0:
			advancingCount++
		case symbol.PercentChange < 0:
			decliningCount++
		default:
			unchangedCount++
		}
		if rangePercent > 0 {
			rangeSum += rangePercent
			rangeCount++
		}
		if weight, exists := majorWeights[symbol.Symbol]; exists {
			majorWeightedChange += symbol.PercentChange * weight
			majorWeightSum += weight
		}
		if symbol.QuoteVolume > 0 {
			volumeWeightedChange += symbol.PercentChange * symbol.QuoteVolume
			quoteVolumeSum += symbol.QuoteVolume
		}
	}

	sort.Slice(items, func(i, j int) bool { return items[i].PercentChange > items[j].PercentChange })
	sortedChanges := append([]float64(nil), changes...)
	sort.Float64s(sortedChanges)
	averageChange := average(changes)
	snapshot := RegimeSnapshot{
		AsOf: time.Now().UTC().Format(time.RFC3339), SymbolCount: len(symbols),
		AdvancingCount: advancingCount, DecliningCount: decliningCount, UnchangedCount: unchangedCount,
		AdvancingRatio: float64(advancingCount) / float64(len(symbols)),
		DecliningRatio: float64(decliningCount) / float64(len(symbols)),
		AverageChange:  averageChange, MedianChange: median(sortedChanges), ChangeStdDev: stddev(changes, averageChange),
		MajorSymbols: filterRegimeMajors(items, majorWeights),
		TopGainers:   copyRegimeSymbols(items, 0, regimeListLimit),
		TopLosers:    copyRegimeSymbols(items, len(items)-regimeListLimit, len(items)),
	}
	if rangeCount > 0 {
		snapshot.AverageRange = rangeSum / float64(rangeCount)
	}
	if majorWeightSum > 0 {
		snapshot.MajorWeightedChange = majorWeightedChange / majorWeightSum
	}
	if quoteVolumeSum > 0 {
		snapshot.QuoteVolumeWeightedChange = volumeWeightedChange / quoteVolumeSum
	}
	return snapshot
}

func SaveMarketCondition(ctx context.Context, configID int64, condition int) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if !markettypes.IsValidMarketCondition(condition) {
		return fmt.Errorf("invalid market condition %d", condition)
	}
	_, err := orm.NewOrm().QueryTable("config").Filter("id", configID).Update(orm.Params{"market_condition": condition})
	if err != nil {
		return fmt.Errorf("update market condition: %w", err)
	}
	return nil
}
func SanitizeReason(reason string) string {
	reason = strings.TrimSpace(strings.ReplaceAll(strings.ReplaceAll(reason, "\r", " "), "\n", " "))
	runes := []rune(reason)
	if len(runes) > 200 {
		return string(runes[:200])
	}
	return reason
}

func regimeRangePercent(symbol models.Symbols) float64 {
	openPrice, openErr := strconv.ParseFloat(symbol.Open, 64)
	highPrice, highErr := strconv.ParseFloat(symbol.High, 64)
	lowPrice, lowErr := strconv.ParseFloat(symbol.Low, 64)
	if openErr != nil || highErr != nil || lowErr != nil || openPrice <= 0 || highPrice < lowPrice {
		return 0
	}
	return (highPrice - lowPrice) / openPrice * 100
}

func filterRegimeMajors(items []RegimeSymbol, weights map[string]float64) []RegimeSymbol {
	result := make([]RegimeSymbol, 0, len(weights))
	for _, item := range items {
		if _, exists := weights[item.Symbol]; exists {
			result = append(result, item)
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Symbol < result[j].Symbol })
	return result
}

func copyRegimeSymbols(items []RegimeSymbol, start, end int) []RegimeSymbol {
	if start < 0 {
		start = 0
	}
	if end > len(items) {
		end = len(items)
	}
	if start >= end {
		return []RegimeSymbol{}
	}
	return append([]RegimeSymbol(nil), items[start:end]...)
}

func average(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	var sum float64
	for _, value := range values {
		sum += value
	}
	return sum / float64(len(values))
}
func median(sortedValues []float64) float64 {
	if len(sortedValues) == 0 {
		return 0
	}
	middle := len(sortedValues) / 2
	if len(sortedValues)%2 == 0 {
		return (sortedValues[middle-1] + sortedValues[middle]) / 2
	}
	return sortedValues[middle]
}

func stddev(values []float64, avg float64) float64 {
	if len(values) == 0 {
		return 0
	}
	var sum float64
	for _, value := range values {
		difference := value - avg
		sum += difference * difference
	}
	return math.Sqrt(sum / float64(len(values)))
}
