package backtest

import (
	"encoding/json"
	"fmt"
	"math"
	"runtime"
	"sync"

	"go_binance_futures/feature/strategy/line"
	"go_binance_futures/technology"

	"github.com/expr-lang/expr/ast"
	"github.com/expr-lang/expr/parser"
)

type cachedKLinePrice struct {
	windowStart int64
	limit       int
	value       line.KLinePrice
}

type cachedIndicatorValue struct {
	windowStart int64
	value       interface{}
}

type cachedCurrentEMA struct {
	windowStart int64
	interval    string
	period      int
	value       line.ConfigData
}

type cachedCurrentRSI struct {
	windowStart   int64
	interval      string
	period        int
	previousClose float64
	avgGain       float64
	avgLoss       float64
	value         line.ConfigData
}

type cachedCurrentADX struct {
	windowStart     int64
	interval        string
	period          int
	previousHigh    float64
	previousLow     float64
	previousClose   float64
	previousADX     float64
	smoothedTR      float64
	smoothedPlusDM  float64
	smoothedMinusDM float64
	value           line.ADXConfigData
}

type indicatorTask struct {
	group       string
	item        technology.IndicatorConfig
	price       line.KLinePrice
	cacheable   bool
	windowStart int64
}

type indicatorTaskResult struct {
	value interface{}
	err   error
}

type indicatorGroup struct {
	name  string
	items []technology.IndicatorConfig
}

func (builder *historicalEnvironment) indicatorGroups() []indicatorGroup {
	return []indicatorGroup{
		{"ma", builder.technology.MA}, {"ema", builder.technology.EMA}, {"macd", builder.technology.MACD},
		{"rsi", builder.technology.RSI}, {"kc", builder.technology.KC}, {"boll", builder.technology.BOLL},
		{"atr", builder.technology.ATR}, {"adx", builder.technology.ADX}, {"mfi", builder.technology.MFI},
		{"obv", builder.technology.OBV}, {"cci", builder.technology.CCI}, {"roc", builder.technology.ROC},
		{"kdj", builder.technology.KDJ}, {"supertrend", builder.technology.Supertrend}, {"donchian", builder.technology.Donchian},
	}
}

func (builder *historicalEnvironment) enableStandardOptimizations(strategyJSON string) {
	builder.standardOptimizations = true
	builder.klinePriceCache = make(map[string]cachedKLinePrice)
	builder.indicatorCache = make(map[string]cachedIndicatorValue)
	builder.currentEMACache = make(map[string]cachedCurrentEMA)
	builder.currentRSICache = make(map[string]cachedCurrentRSI)
	builder.currentADXCache = make(map[string]cachedCurrentADX)
	builder.indicatorGroupsCache = builder.indicatorGroups()
	builder.indicatorPricesScratch = make(map[string]line.KLinePrice)
	builder.indicatorCachedScratch = make(map[string]interface{})
	builder.indicatorCacheable = analyzeCompletedIndicatorCacheability(strategyJSON, builder.technology)
}

func analyzeCompletedIndicatorCacheability(strategyJSON string, config technology.TechnologyConfig) map[string]bool {
	names := make(map[string]struct{})
	groups := [][]technology.IndicatorConfig{config.MA, config.EMA, config.MACD, config.RSI, config.KC, config.BOLL, config.ATR, config.ADX, config.MFI, config.OBV, config.CCI, config.ROC, config.KDJ, config.Supertrend, config.Donchian}
	for _, group := range groups {
		for _, item := range group {
			if item.Enable {
				names[item.Name] = struct{}{}
			}
		}
	}
	result := make(map[string]bool, len(names))
	for name := range names {
		result[name] = true
	}
	var rules []Rule
	if err := json.Unmarshal([]byte(strategyJSON), &rules); err != nil {
		for name := range result {
			result[name] = false
		}
		return result
	}
	for _, rule := range rules {
		if !rule.Enable {
			continue
		}
		tree, err := parser.Parse(rule.Code)
		if err != nil {
			for name := range result {
				result[name] = false
			}
			return result
		}
		inspectIndicatorNode(tree.Node, names, result)
	}
	return result
}

func inspectIndicatorNode(node ast.Node, names map[string]struct{}, result map[string]bool) {
	if node == nil {
		return
	}
	if name, index, ok := indicatorIndexedAccess(node, names); ok {
		if index < 1 {
			result[name] = false
		}
		return
	}
	if name, from, ok := indicatorSliceAccess(node, names); ok {
		if from < 1 {
			result[name] = false
		}
		return
	}
	switch n := node.(type) {
	case *ast.IdentifierNode:
		if _, ok := names[n.Value]; ok {
			result[n.Value] = false
		}
	case *ast.UnaryNode:
		inspectIndicatorNode(n.Node, names, result)
	case *ast.BinaryNode:
		inspectIndicatorNode(n.Left, names, result)
		inspectIndicatorNode(n.Right, names, result)
	case *ast.ChainNode:
		inspectIndicatorNode(n.Node, names, result)
	case *ast.MemberNode:
		inspectIndicatorNode(n.Node, names, result)
		inspectIndicatorNode(n.Property, names, result)
	case *ast.SliceNode:
		inspectIndicatorNode(n.Node, names, result)
		inspectIndicatorNode(n.From, names, result)
		inspectIndicatorNode(n.To, names, result)
	case *ast.CallNode:
		inspectIndicatorNode(n.Callee, names, result)
		for _, arg := range n.Arguments {
			inspectIndicatorNode(arg, names, result)
		}
	case *ast.BuiltinNode:
		for _, arg := range n.Arguments {
			inspectIndicatorNode(arg, names, result)
		}
	case *ast.PredicateNode:
		inspectIndicatorNode(n.Node, names, result)
	case *ast.VariableDeclaratorNode:
		inspectIndicatorNode(n.Value, names, result)
		inspectIndicatorNode(n.Expr, names, result)
	case *ast.SequenceNode:
		for _, child := range n.Nodes {
			inspectIndicatorNode(child, names, result)
		}
	case *ast.ConditionalNode:
		inspectIndicatorNode(n.Cond, names, result)
		inspectIndicatorNode(n.Exp1, names, result)
		inspectIndicatorNode(n.Exp2, names, result)
	case *ast.ArrayNode:
		for _, child := range n.Nodes {
			inspectIndicatorNode(child, names, result)
		}
	case *ast.MapNode:
		for _, pair := range n.Pairs {
			inspectIndicatorNode(pair, names, result)
		}
	case *ast.PairNode:
		inspectIndicatorNode(n.Key, names, result)
		inspectIndicatorNode(n.Value, names, result)
	}
}

func indicatorIndexedAccess(node ast.Node, names map[string]struct{}) (string, int, bool) {
	member, ok := node.(*ast.MemberNode)
	if !ok {
		return "", 0, false
	}
	index, ok := member.Property.(*ast.IntegerNode)
	if !ok {
		return "", 0, false
	}
	name, ok := indicatorFieldBase(member.Node, names)
	if !ok {
		return "", 0, false
	}
	return name, index.Value, true
}

func indicatorSliceAccess(node ast.Node, names map[string]struct{}) (string, int, bool) {
	slice, ok := node.(*ast.SliceNode)
	if !ok {
		return "", 0, false
	}
	name, ok := indicatorFieldBase(slice.Node, names)
	if !ok {
		return "", 0, false
	}
	from, ok := slice.From.(*ast.IntegerNode)
	if !ok {
		return name, 0, true
	}
	if slice.To != nil {
		if _, ok := slice.To.(*ast.IntegerNode); !ok {
			return name, 0, true
		}
	}
	return name, from.Value, true
}

func indicatorFieldBase(node ast.Node, names map[string]struct{}) (string, bool) {
	field, ok := node.(*ast.MemberNode)
	if !ok {
		return "", false
	}
	identifier, ok := field.Node.(*ast.IdentifierNode)
	if !ok {
		return "", false
	}
	if _, ok := names[identifier.Value]; !ok {
		return "", false
	}
	if _, ok := field.Property.(*ast.StringNode); !ok {
		return "", false
	}
	return identifier.Value, true
}

func (builder *historicalEnvironment) indicatorWindowStart(interval string) (int64, bool) {
	overlay, ok := builder.overlays[BarSeriesKey(builder.dataset.Symbol, interval)]
	if !ok || overlay.OpenTime <= 0 {
		return 0, false
	}
	return overlay.OpenTime, true
}

func (builder *historicalEnvironment) cachedKlinePriceSeries(interval string, asOf int64) line.KLinePrice {
	if !builder.standardOptimizations {
		return builder.klinePriceSeries(builder.dataset.Symbol, interval, asOf, builder.warmupBars)
	}
	windowStart, ok := builder.indicatorWindowStart(interval)
	if !ok {
		return builder.klinePriceSeries(builder.dataset.Symbol, interval, asOf, builder.warmupBars)
	}
	entry, exists := builder.klinePriceCache[interval]
	if !exists || entry.windowStart != windowStart || entry.limit != builder.warmupBars || len(entry.value.Close) == 0 {
		value := builder.klinePriceSeries(builder.dataset.Symbol, interval, asOf, builder.warmupBars)
		builder.klinePriceCache[interval] = cachedKLinePrice{windowStart: windowStart, limit: builder.warmupBars, value: value}
		return value
	}
	overlay, exists := builder.overlays[BarSeriesKey(builder.dataset.Symbol, interval)]
	if !exists || len(entry.value.Close) == 0 {
		return builder.klinePriceSeries(builder.dataset.Symbol, interval, asOf, builder.warmupBars)
	}
	entry.value.High[0] = overlay.High
	entry.value.Low[0] = overlay.Low
	entry.value.Close[0] = overlay.Close
	entry.value.Open[0] = overlay.Open
	entry.value.Amount[0] = overlay.QuoteVolume
	seconds := builder.cachedKlineQPSDurationSeconds(overlay)
	if seconds > 0 {
		entry.value.Qps[0] = overlay.QuoteVolume / seconds
	} else {
		entry.value.Qps[0] = 0
	}
	builder.klinePriceCache[interval] = entry
	return entry.value
}

func (builder *historicalEnvironment) addIndicatorsOptimized(env map[string]interface{}, asOf int64) error {
	prices := builder.indicatorPricesScratch
	for key := range prices {
		delete(prices, key)
	}
	tasks := builder.indicatorTasksScratch[:0]
	cachedValues := builder.indicatorCachedScratch
	for key := range cachedValues {
		delete(cachedValues, key)
	}
	groups := builder.indicatorGroupsCache
	for _, group := range groups {
		for _, item := range group.items {
			if !item.Enable {
				continue
			}
			p, ok := prices[item.KlineInterval]
			if !ok {
				p = builder.cachedKlinePriceSeries(item.KlineInterval, asOf)
				if len(p.Close) == 0 {
					return fmt.Errorf("%w: indicator %s has no visible bars", ErrInsufficientHistoricalBars, item.Name)
				}
				prices[item.KlineInterval] = p
				env["kline_"+item.KlineInterval] = p
			}
			windowStart, hasWindow := builder.indicatorWindowStart(item.KlineInterval)
			cacheable := hasWindow && builder.indicatorCacheable[item.Name]
			if cacheable {
				if cached, ok := builder.indicatorCache[item.Name]; ok && cached.windowStart == windowStart {
					cachedValues[item.Name] = cached.value
					continue
				}
			}
			tasks = append(tasks, indicatorTask{group: group.name, item: item, price: p, cacheable: cacheable, windowStart: windowStart})
		}
	}
	builder.indicatorTasksScratch = tasks
	results := builder.runIndicatorTasks(tasks)
	resultIndex := 0
	for _, group := range groups {
		for _, item := range group.items {
			if !item.Enable {
				continue
			}
			if value, ok := cachedValues[item.Name]; ok {
				env[item.Name] = value
				continue
			}
			task := tasks[resultIndex]
			result := results[resultIndex]
			resultIndex++
			if result.err != nil {
				return fmt.Errorf("%w: indicator %s warmup at %d: %v", ErrInsufficientHistoricalBars, item.Name, asOf, result.err)
			}
			env[item.Name] = result.value
			if task.cacheable {
				builder.indicatorCache[item.Name] = cachedIndicatorValue{windowStart: task.windowStart, value: result.value}
			}
		}
	}
	return nil
}

func (builder *historicalEnvironment) runIndicatorTasks(tasks []indicatorTask) []indicatorTaskResult {
	if cap(builder.indicatorResultsScratch) < len(tasks) {
		builder.indicatorResultsScratch = make([]indicatorTaskResult, len(tasks))
	}
	results := builder.indicatorResultsScratch[:len(tasks)]
	for i := range results {
		results[i] = indicatorTaskResult{}
	}
	if len(tasks) == 0 {
		return results
	}

	// Non-cacheable indicators depend on the current forming bar and therefore
	// run every replay minute. Their windows are intentionally small; spawning a
	// worker pool every minute costs more than the indicator calculation itself.
	// Keep them on the caller goroutine. Parallelism is reserved for completed-
	// indicator cache misses, which only happen at higher-interval boundaries.
	parallel := builder.indicatorParallelScratch[:0]
	for i := range tasks {
		if !tasks[i].cacheable {
			switch {
			case tasks[i].group == "ema" && tasks[i].windowStart > 0:
				results[i].value, results[i].err = builder.calculateCurrentEMAValue(tasks[i])
			case tasks[i].group == "rsi" && tasks[i].windowStart > 0:
				results[i].value, results[i].err = builder.calculateCurrentRSIValue(tasks[i])
			case tasks[i].group == "adx" && tasks[i].windowStart > 0:
				results[i].value, results[i].err = builder.calculateCurrentADXValue(tasks[i])
			default:
				results[i].value, results[i].err = calculateIndicatorValue(tasks[i])
			}
			continue
		}
		parallel = append(parallel, i)
	}
	builder.indicatorParallelScratch = parallel
	if len(parallel) == 0 {
		return results
	}
	workers := runtime.GOMAXPROCS(0)
	if workers > 4 {
		workers = 4
	}
	if workers > len(parallel) {
		workers = len(parallel)
	}
	if workers <= 1 || len(parallel) < 2 {
		for _, index := range parallel {
			results[index].value, results[index].err = calculateIndicatorValue(tasks[index])
		}
		return results
	}
	jobs := make(chan int)
	var wg sync.WaitGroup
	wg.Add(workers)
	for i := 0; i < workers; i++ {
		go func() {
			defer wg.Done()
			for index := range jobs {
				results[index].value, results[index].err = calculateIndicatorValue(tasks[index])
			}
		}()
	}
	for _, index := range parallel {
		jobs <- index
	}
	close(jobs)
	wg.Wait()
	return results
}

func (builder *historicalEnvironment) calculateCurrentEMAValue(task indicatorTask) (interface{}, error) {
	item, p := task.item, task.price
	if !builder.standardOptimizations || builder.currentEMACache == nil || len(p.Close) == 0 {
		return calculateIndicatorValue(task)
	}

	entry, ok := builder.currentEMACache[item.Name]
	if !ok || entry.windowStart != task.windowStart || entry.interval != item.KlineInterval || entry.period != item.Period || len(entry.value.Data) < 2 {
		value, err := calculateIndicatorValue(task)
		if err != nil {
			return value, err
		}
		config, ok := value.(line.ConfigData)
		if !ok || len(config.Data) < 2 {
			return value, nil
		}
		builder.currentEMACache[item.Name] = cachedCurrentEMA{
			windowStart: task.windowStart,
			interval:    item.KlineInterval,
			period:      item.Period,
			value:       config,
		}
		return config, nil
	}

	alpha := 2.0 / (float64(item.Period) + 1.0)
	entry.value.Data[0] = alpha*p.Close[0] + (1.0-alpha)*entry.value.Data[1]
	builder.currentEMACache[item.Name] = entry
	return entry.value, nil
}

func (builder *historicalEnvironment) calculateCurrentRSIValue(task indicatorTask) (interface{}, error) {
	item, p := task.item, task.price
	if !builder.standardOptimizations || builder.currentRSICache == nil || len(p.Close) < 2 {
		return calculateIndicatorValue(task)
	}

	entry, ok := builder.currentRSICache[item.Name]
	if !ok || entry.windowStart != task.windowStart || entry.interval != item.KlineInterval || entry.period != item.Period || len(entry.value.Data) < 2 {
		value, err := calculateIndicatorValue(task)
		if err != nil {
			return value, err
		}
		config, ok := value.(line.ConfigData)
		if !ok || len(config.Data) < 2 {
			return value, nil
		}
		avgGain, avgLoss, err := completedRSIState(p.Close[1:], item.Period)
		if err != nil {
			return value, nil // full calculation succeeded; keep the legacy result if state caching is unavailable
		}
		builder.currentRSICache[item.Name] = cachedCurrentRSI{
			windowStart:   task.windowStart,
			interval:      item.KlineInterval,
			period:        item.Period,
			previousClose: p.Close[1],
			avgGain:       avgGain,
			avgLoss:       avgLoss,
			value:         config,
		}
		return config, nil
	}

	gain, loss := 0.0, 0.0
	diff := p.Close[0] - entry.previousClose
	if diff > 0 {
		gain = diff
	} else {
		loss = math.Abs(diff)
	}
	period := float64(item.Period)
	avgGain := ((entry.avgGain * float64(item.Period-1)) + gain) / period
	avgLoss := ((entry.avgLoss * float64(item.Period-1)) + loss) / period
	entry.value.Data[0] = rsiValue(avgGain, avgLoss)
	builder.currentRSICache[item.Name] = entry
	return entry.value, nil
}

func completedRSIState(prices []float64, period int) (float64, float64, error) {
	if period <= 0 || len(prices) <= period {
		return 0, 0, fmt.Errorf("insufficient data for RSI state period %d: got %d values", period, len(prices))
	}
	reversed := make([]float64, len(prices))
	for i, value := range prices {
		reversed[len(prices)-1-i] = value
	}
	gains := make([]float64, len(reversed)-1)
	losses := make([]float64, len(reversed)-1)
	for i := 1; i < len(reversed); i++ {
		diff := reversed[i] - reversed[i-1]
		if diff > 0 {
			gains[i-1] = diff
		} else {
			losses[i-1] = math.Abs(diff)
		}
	}
	var sumGains, sumLosses float64
	for i := 0; i < period; i++ {
		sumGains += gains[i]
		sumLosses += losses[i]
	}
	avgGain := sumGains / float64(period)
	avgLoss := sumLosses / float64(period)
	for i := period; i < len(gains); i++ {
		avgGain = ((avgGain * float64(period-1)) + gains[i]) / float64(period)
		avgLoss = ((avgLoss * float64(period-1)) + losses[i]) / float64(period)
	}
	return avgGain, avgLoss, nil
}

func rsiValue(avgGain, avgLoss float64) float64 {
	if avgGain == 0 && avgLoss == 0 {
		return 50
	}
	if avgLoss == 0 {
		return 100
	}
	if avgGain == 0 {
		return 0
	}
	rs := avgGain / avgLoss
	return 100 - (100 / (1 + rs))
}

func (builder *historicalEnvironment) calculateCurrentADXValue(task indicatorTask) (interface{}, error) {
	item, p := task.item, task.price
	if !builder.standardOptimizations || builder.currentADXCache == nil || len(p.High) < 2 || len(p.Low) < 2 || len(p.Close) < 2 {
		return calculateIndicatorValue(task)
	}

	entry, ok := builder.currentADXCache[item.Name]
	if !ok || entry.windowStart != task.windowStart || entry.interval != item.KlineInterval || entry.period != item.Period || len(entry.value.ADX) < 2 || len(entry.value.PlusDI) < 2 || len(entry.value.MinusDI) < 2 {
		value, err := calculateIndicatorValue(task)
		if err != nil {
			return value, err
		}
		config, ok := value.(line.ADXConfigData)
		if !ok || len(config.ADX) < 2 || len(config.PlusDI) < 2 || len(config.MinusDI) < 2 {
			return value, nil
		}
		smoothedTR, smoothedPlusDM, smoothedMinusDM, err := completedADXDirectionalState(p.High[1:], p.Low[1:], p.Close[1:], item.Period)
		if err != nil {
			return value, nil // preserve the legacy value if state caching is unavailable
		}
		builder.currentADXCache[item.Name] = cachedCurrentADX{
			windowStart:     task.windowStart,
			interval:        item.KlineInterval,
			period:          item.Period,
			previousHigh:    p.High[1],
			previousLow:     p.Low[1],
			previousClose:   p.Close[1],
			previousADX:     config.ADX[1],
			smoothedTR:      smoothedTR,
			smoothedPlusDM:  smoothedPlusDM,
			smoothedMinusDM: smoothedMinusDM,
			value:           config,
		}
		return config, nil
	}

	trueRange := math.Max(p.High[0]-p.Low[0], math.Max(math.Abs(p.High[0]-entry.previousClose), math.Abs(p.Low[0]-entry.previousClose)))
	upMove := p.High[0] - entry.previousHigh
	downMove := entry.previousLow - p.Low[0]
	plusDM, minusDM := 0.0, 0.0
	if upMove > downMove && upMove > 0 {
		plusDM = upMove
	} else if downMove > upMove && downMove > 0 {
		minusDM = downMove
	}
	period := float64(item.Period)
	smoothedTR := (entry.smoothedTR*float64(item.Period-1) + trueRange) / period
	smoothedPlusDM := (entry.smoothedPlusDM*float64(item.Period-1) + plusDM) / period
	smoothedMinusDM := (entry.smoothedMinusDM*float64(item.Period-1) + minusDM) / period
	plusDI, minusDI, dx := 0.0, 0.0, 0.0
	if smoothedTR != 0 {
		plusDI = 100 * smoothedPlusDM / smoothedTR
		minusDI = 100 * smoothedMinusDM / smoothedTR
		directionalSum := plusDI + minusDI
		if directionalSum > 0 {
			dx = 100 * math.Abs(plusDI-minusDI) / directionalSum
		}
	}
	entry.value.PlusDI[0] = plusDI
	entry.value.MinusDI[0] = minusDI
	entry.value.ADX[0] = (entry.previousADX*float64(item.Period-1) + dx) / period
	builder.currentADXCache[item.Name] = entry
	return entry.value, nil
}

func completedADXDirectionalState(high, low, close []float64, period int) (float64, float64, float64, error) {
	if period <= 0 || len(high) != len(low) || len(high) != len(close) || len(high) < period+2 {
		return 0, 0, 0, fmt.Errorf("insufficient data for ADX state period %d: got %d values", period, len(high))
	}
	directionalCount := len(high) - 1
	tr := make([]float64, directionalCount)
	plusDM := make([]float64, directionalCount)
	minusDM := make([]float64, directionalCount)
	for i := 0; i < directionalCount; i++ {
		hl := high[i] - low[i]
		hpc := math.Abs(high[i] - close[i+1])
		lpc := math.Abs(low[i] - close[i+1])
		tr[i] = math.Max(hl, math.Max(hpc, lpc))
		upMove := high[i] - high[i+1]
		downMove := low[i+1] - low[i]
		if upMove > downMove && upMove > 0 {
			plusDM[i] = upMove
		} else if downMove > upMove && downMove > 0 {
			minusDM[i] = downMove
		}
	}
	smoothedTR, err := wilderLatest(tr, period)
	if err != nil {
		return 0, 0, 0, err
	}
	smoothedPlusDM, err := wilderLatest(plusDM, period)
	if err != nil {
		return 0, 0, 0, err
	}
	smoothedMinusDM, err := wilderLatest(minusDM, period)
	if err != nil {
		return 0, 0, 0, err
	}
	return smoothedTR, smoothedPlusDM, smoothedMinusDM, nil
}

func wilderLatest(values []float64, period int) (float64, error) {
	if period <= 0 || len(values) < period {
		return 0, fmt.Errorf("insufficient data for Wilder state period %d: got %d values", period, len(values))
	}
	reversed := make([]float64, len(values))
	for i, value := range values {
		reversed[len(values)-1-i] = value
	}
	var sum float64
	for i := 0; i < period; i++ {
		sum += reversed[i]
	}
	latest := sum / float64(period)
	for i := period; i < len(reversed); i++ {
		latest = (latest*float64(period-1) + reversed[i]) / float64(period)
	}
	return latest, nil
}

func calculateIndicatorValue(task indicatorTask) (interface{}, error) {
	item, p := task.item, task.price
	switch task.group {
	case "ma":
		d, err := line.CalculateSimpleMovingAverage(p.Close, item.Period)
		return line.ConfigData{KlineInterval: item.KlineInterval, Period: item.Period, Data: d}, err
	case "ema":
		d, err := line.CalculateExponentialMovingAverage(p.Close, item.Period)
		return line.ConfigData{KlineInterval: item.KlineInterval, Period: item.Period, Data: d}, err
	case "macd":
		a, b, c, err := line.CalculateMACD(p.Close, item.FastPeriod, item.SlowPeriod, item.SignalPeriod)
		return line.MACDConfigData{KlineInterval: item.KlineInterval, FastPeriod: item.FastPeriod, SlowPeriod: item.SlowPeriod, SignalPeriod: item.SignalPeriod, DIF: a, DEA: b, Histogram: c}, err
	case "rsi":
		d, err := line.CalculateRSI(p.Close, item.Period)
		return line.ConfigData{KlineInterval: item.KlineInterval, Period: item.Period, Data: d}, err
	case "kc":
		a, b, c, err := line.CalculateKeltnerChannels(p.High, p.Low, p.Close, item.Period, item.Multiplier)
		return line.ConfigData{KlineInterval: item.KlineInterval, Period: item.Period, Multiplier: item.Multiplier, High: a, Mid: b, Low: c}, err
	case "boll":
		a, b, c, err := line.CalculateBollingerBands(p.Close, item.Period, item.StdDevMultiplier)
		return line.ConfigData{KlineInterval: item.KlineInterval, Period: item.Period, StdDevMultiplier: item.StdDevMultiplier, High: a, Mid: b, Low: c}, err
	case "atr":
		d, err := line.CalculateAtr(p.High, p.Low, p.Close, item.Period)
		return line.ConfigData{KlineInterval: item.KlineInterval, Period: item.Period, Data: d}, err
	case "adx":
		a, b, c, err := line.CalculateADX(p.High, p.Low, p.Close, item.Period)
		return line.ADXConfigData{KlineInterval: item.KlineInterval, Period: item.Period, ADX: a, PlusDI: b, MinusDI: c}, err
	case "mfi":
		d, err := line.CalculateMFI(p.High, p.Low, p.Close, p.Amount, item.Period)
		return line.ConfigData{KlineInterval: item.KlineInterval, Period: item.Period, Data: d}, err
	case "obv":
		d, err := line.CalculateOBV(p.Close, p.Amount)
		return line.OBVConfigData{KlineInterval: item.KlineInterval, Data: d}, err
	case "cci":
		d, err := line.CalculateCCI(p.High, p.Low, p.Close, item.Period)
		return line.ConfigData{KlineInterval: item.KlineInterval, Period: item.Period, Data: d}, err
	case "roc":
		d, err := line.CalculateROC(p.Close, item.Period)
		return line.ConfigData{KlineInterval: item.KlineInterval, Period: item.Period, Data: d}, err
	case "kdj":
		a, b, c, err := line.Kdj(p.High, p.Low, p.Close, item.Period, item.KPeriod, item.DPeriod)
		return line.KDJConfigData{KlineInterval: item.KlineInterval, Period: item.Period, KPeriod: item.KPeriod, DPeriod: item.DPeriod, K: a, D: b, J: c}, err
	case "supertrend":
		a, b, err := line.CalculateSupertrend(p.High, p.Low, p.Close, item.Period, item.Multiplier)
		return line.SupertrendConfigData{KlineInterval: item.KlineInterval, Period: item.Period, Multiplier: item.Multiplier, Data: a, Trend: b}, err
	case "donchian":
		a, b, c, err := line.CalculateDonchianChannels(p.High, p.Low, item.Period)
		return line.ConfigData{KlineInterval: item.KlineInterval, Period: item.Period, High: a, Mid: b, Low: c}, err
	default:
		return nil, fmt.Errorf("unsupported indicator group %q", task.group)
	}
}
