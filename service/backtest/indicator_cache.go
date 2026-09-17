package backtest

import (
	"encoding/json"
	"fmt"
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
	seconds := klineQPSDurationSeconds(overlay)
	if seconds > 0 {
		entry.value.Qps[0] = overlay.QuoteVolume / seconds
	} else {
		entry.value.Qps[0] = 0
	}
	builder.klinePriceCache[interval] = entry
	return entry.value
}

func (builder *historicalEnvironment) addIndicatorsOptimized(env map[string]interface{}, asOf int64) error {
	prices := make(map[string]line.KLinePrice)
	tasks := make([]indicatorTask, 0)
	cachedValues := make(map[string]interface{})
	for _, group := range builder.indicatorGroups() {
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
	results := runIndicatorTasks(tasks)
	resultIndex := 0
	for _, group := range builder.indicatorGroups() {
		for _, item := range group.items {
			if !item.Enable {
				continue
			}
			if value, ok := cachedValues[item.Name]; ok {
				env[item.Name] = value
				continue
			}
			result := results[resultIndex]
			resultIndex++
			if result.err != nil {
				return fmt.Errorf("%w: indicator %s warmup at %d: %v", ErrInsufficientHistoricalBars, item.Name, asOf, result.err)
			}
			env[item.Name] = result.value
			for _, task := range tasks {
				if task.item.Name == item.Name && task.cacheable {
					builder.indicatorCache[item.Name] = cachedIndicatorValue{windowStart: task.windowStart, value: result.value}
					break
				}
			}
		}
	}
	return nil
}

func runIndicatorTasks(tasks []indicatorTask) []indicatorTaskResult {
	results := make([]indicatorTaskResult, len(tasks))
	if len(tasks) == 0 {
		return results
	}

	// Non-cacheable indicators depend on the current forming bar and therefore
	// run every replay minute. Their windows are intentionally small; spawning a
	// worker pool every minute costs more than the indicator calculation itself.
	// Keep them on the caller goroutine. Parallelism is reserved for completed-
	// indicator cache misses, which only happen at higher-interval boundaries.
	parallel := make([]int, 0, len(tasks))
	for i := range tasks {
		if !tasks[i].cacheable {
			results[i].value, results[i].err = calculateIndicatorValue(tasks[i])
			continue
		}
		parallel = append(parallel, i)
	}
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
