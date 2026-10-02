package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"sort"
	"strconv"
	"time"

	"github.com/beego/beego/v2/client/orm"
	"github.com/beego/beego/v2/core/config"
	_ "github.com/go-sql-driver/mysql"
	_ "go_binance_futures/bootstrap"
	"go_binance_futures/service/historicalmarket"
)

const featureCount = 9
const minLeaf = 2000

var featureNames = [featureCount]string{
	"ret1_side", "ret4_side", "ret12_side", "body_range_side", "taker_imb_side",
	"quote_ratio_8h", "range_ratio_8h", "taker_delta_8h_side", "close_loc_12h_side",
}

type Sample struct {
	X        [featureCount]float64
	Reward   float64
	Year     int
	Sym      int
	Side     int
	SignalTs int64
}

type Split struct {
	Valid     bool    `json:"valid"`
	Feature   int     `json:"feature"`
	Threshold float64 `json:"threshold"`
	Score     float64 `json:"score"`
}

type Node struct {
	Leaf  bool
	Split Split
	Left  *Node
	Right *Node
	N     int
	Mean  float64
}

type Cond struct {
	F    int     `json:"feature"`
	Th   float64 `json:"threshold"`
	Left bool    `json:"left"`
}

type SelectedRule struct {
	LeafID     int     `json:"leaf_id"`
	TrainN     int     `json:"train_n"`
	TrainMean  float64 `json:"train_mean_reward"`
	Conditions []Cond  `json:"conditions"`
}

type Agg struct {
	N      int
	Sum    float64
	LongN  int
	ShortN int
}

func (a *Agg) add(s Sample) {
	a.N++
	a.Sum += s.Reward
	if s.Side > 0 {
		a.LongN++
	} else {
		a.ShortN++
	}
}

func avg(sum float64, n int) float64 {
	if n == 0 {
		return 0
	}
	return sum / float64(n)
}
func quantile(vals []float64, q float64) float64 {
	x := append([]float64(nil), vals...)
	sort.Float64s(x)
	if len(x) == 0 {
		return 0
	}
	pos := q * float64(len(x)-1)
	lo := int(pos)
	hi := lo
	if float64(lo) < pos {
		hi = lo + 1
	}
	if hi >= len(x) {
		hi = len(x) - 1
	}
	if hi == lo {
		return x[lo]
	}
	w := pos - float64(lo)
	return x[lo]*(1-w) + x[hi]*w
}

func stats(samples []Sample, ids []int) (int, float64) {
	if len(ids) == 0 {
		return 0, 0
	}
	sum := 0.0
	for _, id := range ids {
		sum += samples[id].Reward
	}
	return len(ids), sum / float64(len(ids))
}

func bestSplit(samples []Sample, ids []int, thresholds [featureCount][]float64) Split {
	best := Split{}
	bestScore := -1e300
	for f := 0; f < featureCount; f++ {
		for _, th := range thresholds[f] {
			nl, nr := 0, 0
			sl, sr := 0.0, 0.0
			for _, id := range ids {
				if samples[id].X[f] <= th {
					nl++
					sl += samples[id].Reward
				} else {
					nr++
					sr += samples[id].Reward
				}
			}
			if nl < minLeaf || nr < minLeaf {
				continue
			}
			score := sl*sl/float64(nl) + sr*sr/float64(nr)
			if score > bestScore {
				bestScore = score
				best = Split{Valid: true, Feature: f, Threshold: th, Score: score}
			}
		}
	}
	return best
}

func buildTree(samples []Sample, ids []int, thresholds [featureCount][]float64, depth int) *Node {
	n, mean := stats(samples, ids)
	node := &Node{Leaf: true, N: n, Mean: mean}
	if depth >= 2 {
		return node
	}
	sp := bestSplit(samples, ids, thresholds)
	if !sp.Valid {
		return node
	}
	left := make([]int, 0, n)
	right := make([]int, 0, n)
	for _, id := range ids {
		if samples[id].X[sp.Feature] <= sp.Threshold {
			left = append(left, id)
		} else {
			right = append(right, id)
		}
	}
	node.Leaf = false
	node.Split = sp
	node.Left = buildTree(samples, left, thresholds, depth+1)
	node.Right = buildTree(samples, right, thresholds, depth+1)
	return node
}

func assign(node *Node, x [featureCount]float64) *Node {
	cur := node
	for !cur.Leaf {
		if x[cur.Split.Feature] <= cur.Split.Threshold {
			cur = cur.Left
		} else {
			cur = cur.Right
		}
	}
	return cur
}

func collectSelected(node *Node, path []Cond, selected map[*Node]SelectedRule, nextID *int) {
	if node.Leaf {
		*nextID = *nextID + 1
		if node.Mean >= 1.0 {
			r := SelectedRule{LeafID: *nextID, TrainN: node.N, TrainMean: node.Mean, Conditions: append([]Cond(nil), path...)}
			selected[node] = r
		}
		fmt.Printf("LEAF %d n=%d meanReward=%.6f selected=%v path=", *nextID, node.N, node.Mean, node.Mean >= 1.0)
		for _, c := range path {
			op := ">"
			if c.Left {
				op = "<="
			}
			fmt.Printf(" %s%s%.8f", featureNames[c.F], op, c.Th)
		}
		fmt.Println()
		return
	}
	collectSelected(node.Left, append(path, Cond{F: node.Split.Feature, Th: node.Split.Threshold, Left: true}), selected, nextID)
	collectSelected(node.Right, append(path, Cond{F: node.Split.Feature, Th: node.Split.Threshold, Left: false}), selected, nextID)
}
func barTakerRatio(b historicalmarket.ReplayKline) float64 {
	if b.QuoteVolume <= 0 {
		return 0.5
	}
	return b.TakerBuyQuoteVolume / b.QuoteVolume
}

func features(rows []historicalmarket.ReplayKline, i int, side float64) ([featureCount]float64, bool) {
	var x [featureCount]float64
	if i < 12 {
		return x, false
	}
	b := rows[i]
	if b.Open <= 0 || b.High <= 0 || b.Low <= 0 || b.Close <= 0 || b.QuoteVolume <= 0 {
		return x, false
	}

	x[0] = side * math.Log(b.Close/b.Open)
	if rows[i-4].Close <= 0 || rows[i-12].Close <= 0 {
		return x, false
	}
	x[1] = side * math.Log(b.Close/rows[i-4].Close)
	x[2] = side * math.Log(b.Close/rows[i-12].Close)

	rng := b.High - b.Low
	if rng != 0 {
		x[3] = side * (b.Close - b.Open) / rng
	}
	x[4] = side * (2*barTakerRatio(b) - 1)

	qsum := 0.0
	rsum := 0.0
	tsum := 0.0
	for j := i - 8; j <= i-1; j++ {
		p := rows[j]
		if p.QuoteVolume <= 0 || p.High <= 0 || p.Low <= 0 {
			return x, false
		}
		qsum += p.QuoteVolume
		rsum += math.Log(p.High / p.Low)
		tsum += barTakerRatio(p)
	}
	qmean := qsum / 8
	rmean := rsum / 8
	tmean := tsum / 8
	if qmean <= 0 || rmean <= 0 {
		return x, false
	}
	x[5] = b.QuoteVolume / qmean
	x[6] = math.Log(b.High/b.Low) / rmean
	x[7] = side * (barTakerRatio(b) - tmean)

	lo := rows[i-11].Low
	hi := rows[i-11].High
	for j := i - 10; j <= i; j++ {
		if rows[j].Low < lo {
			lo = rows[j].Low
		}
		if rows[j].High > hi {
			hi = rows[j].High
		}
	}
	if hi > lo {
		x[8] = side * (2*(b.Close-lo)/(hi-lo) - 1)
	}
	for _, v := range x {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return [featureCount]float64{}, false
		}
	}
	return x, true
}

func label(mins []historicalmarket.ReplayKline, startIdx int, side int) (float64, bool) {
	if startIdx < 0 || startIdx >= len(mins) {
		return 0, false
	}
	entry := mins[startIdx].Open
	if entry <= 0 {
		return 0, false
	}
	if side > 0 {
		entry *= 1.0005
	} else {
		entry *= 0.9995
	}
	end := startIdx + 1440
	if end > len(mins) {
		end = len(mins)
	}
	if end-startIdx < 1440 {
		return 0, false
	}
	for j := startIdx; j < end; j++ {
		c := mins[j].Close
		if c <= 0 {
			return 0, false
		}
		roi := float64(side) * (c - entry) / entry * 4 * 100
		if roi >= 8 {
			return 8, true
		}
		if roi <= -6 {
			return -6, true
		}
	}
	return 0, false
}

func treeMap(n *Node) map[string]interface{} {
	if n == nil {
		return nil
	}
	out := map[string]interface{}{
		"leaf":        n.Leaf,
		"n":           n.N,
		"mean_reward": n.Mean,
	}
	if !n.Leaf {
		out["feature"] = featureNames[n.Split.Feature]
		out["feature_index"] = n.Split.Feature
		out["threshold"] = n.Split.Threshold
		out["score"] = n.Split.Score
		out["left"] = treeMap(n.Left)
		out["right"] = treeMap(n.Right)
	}
	return out
}
func main() {
	const root = "strategy_templates/research/interpretable-two-split-fast-move/2026-10-01-discovery"
	symbols := []string{"SOLUSDT", "DOGEUSDT", "LTCUSDT", "AVAXUSDT", "UNIUSDT", "ZECUSDT"}

	dbn, _ := config.String("database::dbname")
	if dbn != "go_bn_test" {
		panic("expected go_bn_test, got " + dbn)
	}
	user, _ := config.String("database::username")
	pass, _ := config.String("database::password")
	host, _ := config.String("database::host")
	port, _ := config.String("database::port")
	_ = orm.RegisterDriver("mysql", orm.DRMySQL)
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&collation=utf8mb4_unicode_ci", user, pass, host, port, dbn)
	if err := orm.RegisterDataBase("default", "mysql", dsn); err != nil {
		panic(err)
	}

	ctx := context.Background()
	repo := historicalmarket.NewRepository(nil)
	hourStart := time.Date(2022, 12, 30, 0, 0, 0, 0, time.UTC).UnixMilli()
	dataStart := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC).UnixMilli()
	dataEnd := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC).UnixMilli()

	train := make([]Sample, 0, 120000)
	valid := make([]Sample, 0, 120000)
	candidateByYear := map[int]int{}
	unresolvedByYear := map[int]int{}

	for si, sym := range symbols {
		fmt.Println("LOAD", sym, "1h")
		hours, err := repo.LoadReplayKlines(ctx, historicalmarket.MarketFuturesUSDT, sym, "1h", hourStart, dataEnd-1)
		if err != nil {
			panic(err)
		}
		fmt.Println("LOAD", sym, "1m")
		mins, err := repo.LoadReplayKlines(ctx, historicalmarket.MarketFuturesUSDT, sym, "1m", dataStart, dataEnd-1)
		if err != nil {
			panic(err)
		}
		minuteIndex := make(map[int64]int, len(mins))
		for i := range mins {
			minuteIndex[mins[i].OpenTime] = i
		}

		symTrain, symValid := 0, 0
		for i := 12; i < len(hours); i++ {
			entryT := hours[i].OpenTime + int64(time.Hour/time.Millisecond)
			y := time.UnixMilli(entryT).UTC().Year()
			if y != 2023 && y != 2024 {
				continue
			}
			yearEnd := time.Date(y+1, 1, 1, 0, 0, 0, 0, time.UTC).UnixMilli()
			if entryT+int64(24*time.Hour/time.Millisecond) > yearEnd {
				continue
			}
			mi, ok := minuteIndex[entryT]
			if !ok {
				continue
			}
			for _, side := range []int{1, -1} {
				candidateByYear[y]++
				x, ok := features(hours, i, float64(side))
				if !ok {
					continue
				}
				reward, resolved := label(mins, mi, side)
				if !resolved {
					unresolvedByYear[y]++
					continue
				}
				s := Sample{X: x, Reward: reward, Year: y, Sym: si, Side: side, SignalTs: hours[i].CloseTime}
				if y == 2023 {
					train = append(train, s)
					symTrain++
				} else {
					valid = append(valid, s)
					symValid++
				}
			}
		}
		fmt.Printf("SAMPLES %s train_resolved=%d validation_resolved=%d\n", sym, symTrain, symValid)
		mins = nil
		hours = nil
		minuteIndex = nil
	}
	fmt.Printf("TOTAL train_resolved=%d validation_resolved=%d candidates2023=%d candidates2024=%d unresolved2023=%d unresolved2024=%d\n",
		len(train), len(valid), candidateByYear[2023], candidateByYear[2024], unresolvedByYear[2023], unresolvedByYear[2024])
	if len(train) == 0 || len(valid) == 0 {
		panic("no train/validation samples")
	}

	var thresholds [featureCount][]float64
	qs := []float64{0.10, 0.25, 0.50, 0.75, 0.90}
	thresholdOut := map[string][]float64{}
	for f := 0; f < featureCount; f++ {
		vals := make([]float64, len(train))
		for i := range train {
			vals[i] = train[i].X[f]
		}
		seen := map[float64]bool{}
		for _, q := range qs {
			th := quantile(vals, q)
			if !seen[th] {
				thresholds[f] = append(thresholds[f], th)
				seen[th] = true
			}
		}
		thresholdOut[featureNames[f]] = append([]float64(nil), thresholds[f]...)
	}

	ids := make([]int, len(train))
	for i := range ids {
		ids[i] = i
	}
	tree := buildTree(train, ids, thresholds, 0)
	selected := map[*Node]SelectedRule{}
	leafID := 0
	collectSelected(tree, nil, selected, &leafID)
	fmt.Printf("TREE leaves=%d selected_leaves=%d\n", leafID, len(selected))

	trainSelected := Agg{}
	trainBySym := make([]Agg, len(symbols))
	trainBySide := map[int]*Agg{1: &Agg{}, -1: &Agg{}}
	for _, s := range train {
		leaf := assign(tree, s.X)
		if _, ok := selected[leaf]; !ok {
			continue
		}
		trainSelected.add(s)
		a := trainBySym[s.Sym]
		a.add(s)
		trainBySym[s.Sym] = a
		trainBySide[s.Side].add(s)
	}

	valSelected := Agg{}
	valBySym := make([]Agg, len(symbols))
	valBySide := map[int]*Agg{1: &Agg{}, -1: &Agg{}}
	valByLeaf := map[int]Agg{}
	for _, s := range valid {
		leaf := assign(tree, s.X)
		rule, ok := selected[leaf]
		if !ok {
			continue
		}
		valSelected.add(s)
		a := valBySym[s.Sym]
		a.add(s)
		valBySym[s.Sym] = a
		valBySide[s.Side].add(s)
		a = valByLeaf[rule.LeafID]
		a.add(s)
		valByLeaf[rule.LeafID] = a
	}

	positiveSymbols := 0
	for i, sym := range symbols {
		a := valBySym[i]
		if a.N > 0 && avg(a.Sum, a.N) > 0 {
			positiveSymbols++
		}
		fmt.Printf("VALID_SYM %s n=%d meanReward=%.6f long=%d short=%d\n", sym, a.N, avg(a.Sum, a.N), a.LongN, a.ShortN)
	}
	fmt.Printf("TRAIN_SELECTED n=%d meanReward=%.6f\n", trainSelected.N, avg(trainSelected.Sum, trainSelected.N))
	fmt.Printf("VALID_SELECTED n=%d meanReward=%.6f positive=%d/%d long=%d short=%d\n",
		valSelected.N, avg(valSelected.Sum, valSelected.N), positiveSymbols, len(symbols), valSelected.LongN, valSelected.ShortN)
	fmt.Printf("VALID_SIDE LONG n=%d meanReward=%.6f SHORT n=%d meanReward=%.6f\n",
		valBySide[1].N, avg(valBySide[1].Sum, valBySide[1].N),
		valBySide[-1].N, avg(valBySide[-1].Sum, valBySide[-1].N))

	gatePass := valSelected.N >= 1000 &&
		avg(valSelected.Sum, valSelected.N) >= 0.5 &&
		positiveSymbols >= 4
	rules := make([]SelectedRule, 0, len(selected))
	for _, r := range selected {
		rules = append(rules, r)
	}
	sort.Slice(rules, func(i, j int) bool { return rules[i].LeafID < rules[j].LeafID })

	bySymJSON := map[string]interface{}{}
	for i, sym := range symbols {
		a := valBySym[i]
		bySymJSON[sym] = map[string]interface{}{
			"n": a.N, "mean_reward": avg(a.Sum, a.N), "long": a.LongN, "short": a.ShortN,
		}
	}
	byLeafJSON := map[string]interface{}{}
	for _, r := range rules {
		a := valByLeaf[r.LeafID]
		byLeafJSON[strconv.Itoa(r.LeafID)] = map[string]interface{}{
			"n": a.N, "mean_reward": avg(a.Sum, a.N), "long": a.LongN, "short": a.ShortN,
		}
	}

	status := "frozen_failed_validation"
	if gatePass {
		status = "validation_pass_candidate"
	}
	summary := map[string]interface{}{
		"status":                          status,
		"version":                         "v142",
		"train_year":                      2023,
		"validation_year":                 2024,
		"train_resolved_samples":          len(train),
		"validation_resolved_samples":     len(valid),
		"train_candidates":                candidateByYear[2023],
		"validation_candidates":           candidateByYear[2024],
		"train_unresolved":                unresolvedByYear[2023],
		"validation_unresolved":           unresolvedByYear[2024],
		"tree_leaf_count":                 leafID,
		"selected_leaf_count":             len(selected),
		"train_selected_n":                trainSelected.N,
		"train_selected_mean_reward":      avg(trainSelected.Sum, trainSelected.N),
		"validation_selected_n":           valSelected.N,
		"validation_selected_mean_reward": avg(valSelected.Sum, valSelected.N),
		"validation_positive_symbols":     fmt.Sprintf("%d/%d", positiveSymbols, len(symbols)),
		"validation_by_symbol":            bySymJSON,
		"validation_by_leaf":              byLeafJSON,
		"validation_long": map[string]interface{}{
			"n": valBySide[1].N, "mean_reward": avg(valBySide[1].Sum, valBySide[1].N),
		},
		"validation_short": map[string]interface{}{
			"n": valBySide[-1].N, "mean_reward": avg(valBySide[-1].Sum, valBySide[-1].N),
		},
		"gate": map[string]interface{}{
			"min_validation_samples":     1000,
			"min_validation_mean_reward": 0.5,
			"min_positive_symbols":       4,
			"pass":                       gatePass,
		},
		"oos_2025_2026_read": false,
		"strict_engine_run":  false,
	}

	writeJSON := func(name string, value interface{}) {
		b, err := json.MarshalIndent(value, "", "  ")
		if err != nil {
			panic(err)
		}
		if err := os.WriteFile(root+"/results/"+name, b, 0644); err != nil {
			panic(err)
		}
	}
	writeJSON("summary.json", summary)
	writeJSON("tree.json", treeMap(tree))
	writeJSON("selected_rules.json", rules)
	writeJSON("thresholds.json", thresholdOut)
	fmt.Printf("GATE pass=%v\n", gatePass)
}
