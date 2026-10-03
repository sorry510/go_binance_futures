package main

import (
	"archive/zip"
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

const hourMS int64 = 3600000

var featureNames = []string{
	"ret4", "ret12", "ret24", "rv24", "path_eff12",
	"qv4_vs20", "taker4", "trade4_vs20", "ticket4_vs20",
}

var symbols = []string{
	"BTCUSDT", "ETHUSDT", "BNBUSDT", "XRPUSDT", "SOLUSDT",
	"DOGEUSDT", "LTCUSDT", "AVAXUSDT", "UNIUSDT", "ZECUSDT",
}

var cacheRoots = []string{
	"/tmp/github_core_release_price_cache",
	"/tmp/v147_toptrader_skew",
	"/tmp/v161_basis_momentum",
}

type Bar struct {
	T                      int64
	Open, High, Low, Close float64
	QV, Count, TakerBuyQ   float64
}

type Sample struct {
	X      [9]float64
	Y      float64
	Symbol string
	T      int64
}

type Node struct {
	Leaf        bool    `json:"leaf"`
	Value       float64 `json:"value"`
	N           int     `json:"n"`
	SSE         float64 `json:"sse"`
	Feature     int     `json:"feature,omitempty"`
	FeatureName string  `json:"feature_name,omitempty"`
	Threshold   float64 `json:"threshold,omitempty"`
	Gain        float64 `json:"gain,omitempty"`
	Left        *Node   `json:"left,omitempty"`
	Right       *Node   `json:"right,omitempty"`
}

type Model struct {
	Version      string   `json:"version"`
	Features     []string `json:"features"`
	MaxDepth     int      `json:"max_depth"`
	MinLeaf      int      `json:"min_leaf"`
	TrainStart   string   `json:"train_start"`
	TrainEnd     string   `json:"train_end"`
	TrainSamples int      `json:"train_samples"`
	Root         *Node    `json:"root"`
}

type Event struct {
	Symbol     string  `json:"symbol"`
	SignalTime int64   `json:"signal_time"`
	Side       string  `json:"side"`
	PredPrev   float64 `json:"pred_prev"`
	PredNow    float64 `json:"pred_now"`
	R1         float64 `json:"r1"`
	R4         float64 `json:"r4"`
	R12        float64 `json:"r12"`
	Half       string  `json:"half"`
}

type Stats struct {
	N       int     `json:"n"`
	MeanR1  float64 `json:"mean_r1"`
	MeanR4  float64 `json:"mean_r4"`
	MeanR12 float64 `json:"mean_r12"`
	WinR12  float64 `json:"win_r12"`
	Long    int     `json:"long"`
	Short   int     `json:"short"`
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}

func monthList(year int, warmup bool) []string {
	out := []string{}
	if warmup {
		out = append(out, fmt.Sprintf("%d-12", year-1))
	}
	for m := 1; m <= 12; m++ {
		out = append(out, fmt.Sprintf("%d-%02d", year, m))
	}
	return out
}

func resolveZip(sym, mon string) (string, error) {
	names := []string{fmt.Sprintf("%s-1h-%s.zip", sym, mon), fmt.Sprintf("fut-%s-%s.zip", sym, mon)}
	for _, r := range cacheRoots {
		for _, n := range names {
			p := filepath.Join(r, n)
			if st, err := os.Stat(p); err == nil && st.Size() > 0 {
				return p, nil
			}
		}
	}
	return "", fmt.Errorf("missing 1h cache %s %s", sym, mon)
}

func readZip(path string) ([]Bar, error) {
	zr, err := zip.OpenReader(path)
	if err != nil {
		return nil, err
	}
	defer zr.Close()
	if len(zr.File) == 0 {
		return nil, fmt.Errorf("empty zip %s", path)
	}
	f, err := zr.File[0].Open()
	if err != nil {
		return nil, err
	}
	defer f.Close()
	cr := csv.NewReader(f)
	cr.FieldsPerRecord = -1
	out := []Bar{}
	for {
		r, err := cr.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		if len(r) < 11 {
			continue
		}
		t, err := strconv.ParseInt(r[0], 10, 64)
		if err != nil {
			continue
		}
		if t > 1e15 {
			t /= 1000
		}
		vals := make([]float64, 10)
		ok := true
		for i := 1; i <= 10; i++ {
			v, e := strconv.ParseFloat(r[i], 64)
			if e != nil {
				ok = false
				break
			}
			vals[i-1] = v
		}
		if !ok {
			continue
		}
		b := Bar{T: t, Open: vals[0], High: vals[1], Low: vals[2], Close: vals[3], QV: vals[6], Count: vals[7], TakerBuyQ: vals[9]}
		if b.Open <= 0 || b.High <= 0 || b.Low <= 0 || b.Close <= 0 || b.QV <= 0 || b.Count <= 0 {
			continue
		}
		out = append(out, b)
	}
	return out, nil
}

func loadBars(sym string, year int) ([]Bar, error) {
	months := monthList(year, true)
	all := []Bar{}
	for _, m := range months {
		p, err := resolveZip(sym, m)
		if err != nil {
			return nil, err
		}
		b, err := readZip(p)
		if err != nil {
			return nil, err
		}
		all = append(all, b...)
	}
	sort.Slice(all, func(i, j int) bool { return all[i].T < all[j].T })
	// de-duplicate by timestamp.
	out := make([]Bar, 0, len(all))
	for _, b := range all {
		if len(out) > 0 && out[len(out)-1].T == b.T {
			out[len(out)-1] = b
		} else {
			out = append(out, b)
		}
	}
	return out, nil
}

func logret(a, b float64) float64 { return math.Log(b / a) }

func features(b []Bar, i int) ([9]float64, bool) {
	var x [9]float64
	if i < 24 {
		return x, false
	}
	if b[i].T-b[i-24].T != 24*hourMS {
		return x, false
	}
	x[0] = logret(b[i-4].Close, b[i].Close)
	x[1] = logret(b[i-12].Close, b[i].Close)
	x[2] = logret(b[i-24].Close, b[i].Close)
	rv := 0.0
	path := 0.0
	for k := i - 23; k <= i; k++ {
		r := logret(b[k-1].Close, b[k].Close)
		rv += r * r
		if k >= i-11 {
			path += math.Abs(r)
		}
	}
	x[3] = math.Sqrt(rv)
	if path > 0 {
		x[4] = math.Abs(x[1]) / path
	} else {
		x[4] = 0
	}
	q4, q20, c4, c20, taker := 0.0, 0.0, 0.0, 0.0, 0.0
	for k := i - 23; k <= i; k++ {
		if b[k].QV <= 0 || b[k].Count <= 0 {
			return x, false
		}
		if k >= i-3 {
			q4 += b[k].QV
			c4 += b[k].Count
			taker += 2*b[k].TakerBuyQ/b[k].QV - 1
		} else {
			q20 += b[k].QV
			c20 += b[k].Count
		}
	}
	if q4 <= 0 || q20 <= 0 || c4 <= 0 || c20 <= 0 {
		return x, false
	}
	x[5] = math.Log((q4 / 4) / (q20 / 20))
	x[6] = taker / 4
	x[7] = math.Log((c4 / 4) / (c20 / 20))
	x[8] = math.Log((q4 / c4) / (q20 / c20))
	for _, v := range x {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return x, false
		}
	}
	return x, true
}

func yearBounds(year int) (int64, int64) {
	a := time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC).UnixMilli()
	z := time.Date(year+1, 1, 1, 0, 0, 0, 0, time.UTC).UnixMilli()
	return a, z
}

func buildSamples(year int) ([]Sample, map[string]int, error) {
	start, end := yearBounds(year)
	out := []Sample{}
	counts := map[string]int{}
	for _, sym := range symbols {
		b, err := loadBars(sym, year)
		if err != nil {
			return nil, nil, err
		}
		n := 0
		for i := 24; i+12 < len(b); i++ {
			t := b[i].T
			if t < start || t >= end {
				continue
			}
			if b[i+12].T-t != 12*hourMS || b[i+1].T-t != hourMS {
				continue
			}
			if b[i+12].T >= end {
				continue
			}
			x, ok := features(b, i)
			if !ok {
				continue
			}
			y := math.Log(b[i+12].Close / b[i+1].Open)
			if math.IsNaN(y) || math.IsInf(y, 0) {
				continue
			}
			out = append(out, Sample{X: x, Y: y, Symbol: sym, T: t})
			n++
		}
		counts[sym] = n
		fmt.Printf("TRAIN_SOURCE %s samples %d\n", sym, n)
	}
	return out, counts, nil
}

func meanY(s []Sample) float64 {
	q := 0.0
	for _, x := range s {
		q += x.Y
	}
	if len(s) == 0 {
		return 0
	}
	return q / float64(len(s))
}
func sseSamples(s []Sample) float64 {
	if len(s) == 0 {
		return 0
	}
	sum, sum2 := 0.0, 0.0
	for _, x := range s {
		sum += x.Y
		sum2 += x.Y * x.Y
	}
	v := sum2 - sum*sum/float64(len(s))
	if v < 0 && v > -1e-12 {
		return 0
	}
	return v
}

type pair struct{ X, Y float64 }

func bestSplit(s []Sample, minLeaf int) (feat int, thr, gain float64, ok bool) {
	parent := sseSamples(s)
	bestGain := 0.0
	bestF := -1
	bestT := 0.0
	for f := 0; f < len(featureNames); f++ {
		ps := make([]pair, len(s))
		totalSum, totalSq := 0.0, 0.0
		for i, z := range s {
			ps[i] = pair{z.X[f], z.Y}
			totalSum += z.Y
			totalSq += z.Y * z.Y
		}
		sort.Slice(ps, func(i, j int) bool {
			if ps[i].X == ps[j].X {
				return ps[i].Y < ps[j].Y
			}
			return ps[i].X < ps[j].X
		})
		leftSum, leftSq := 0.0, 0.0
		for i := 0; i < len(ps)-1; i++ {
			leftSum += ps[i].Y
			leftSq += ps[i].Y * ps[i].Y
			ln := i + 1
			rn := len(ps) - ln
			if ln < minLeaf || rn < minLeaf {
				continue
			}
			if ps[i].X == ps[i+1].X {
				continue
			}
			rightSum := totalSum - leftSum
			rightSq := totalSq - leftSq
			lsse := leftSq - leftSum*leftSum/float64(ln)
			rsse := rightSq - rightSum*rightSum/float64(rn)
			g := parent - lsse - rsse
			t := ps[i].X + (ps[i+1].X-ps[i].X)/2
			if g > bestGain+1e-15 || (math.Abs(g-bestGain) <= 1e-15 && (bestF < 0 || f < bestF || (f == bestF && t < bestT))) {
				bestGain = g
				bestF = f
				bestT = t
			}
		}
	}
	if bestF < 0 || bestGain <= 0 {
		return 0, 0, 0, false
	}
	return bestF, bestT, bestGain, true
}

func buildTree(s []Sample, depth, maxDepth, minLeaf int) *Node {
	n := &Node{Leaf: true, Value: meanY(s), N: len(s), SSE: sseSamples(s)}
	if depth >= maxDepth || len(s) < 2*minLeaf {
		return n
	}
	f, t, g, ok := bestSplit(s, minLeaf)
	if !ok {
		return n
	}
	l := make([]Sample, 0, len(s))
	r := make([]Sample, 0, len(s))
	for _, z := range s {
		if z.X[f] <= t {
			l = append(l, z)
		} else {
			r = append(r, z)
		}
	}
	if len(l) < minLeaf || len(r) < minLeaf {
		return n
	}
	n.Leaf = false
	n.Feature = f
	n.FeatureName = featureNames[f]
	n.Threshold = t
	n.Gain = g
	n.Left = buildTree(l, depth+1, maxDepth, minLeaf)
	n.Right = buildTree(r, depth+1, maxDepth, minLeaf)
	return n
}

func predict(n *Node, x [9]float64) float64 {
	for !n.Leaf {
		if x[n.Feature] <= n.Threshold {
			n = n.Left
		} else {
			n = n.Right
		}
	}
	return n.Value
}

func leaves(n *Node, out *[]map[string]interface{}) {
	if n.Leaf {
		*out = append(*out, map[string]interface{}{"n": n.N, "value": n.Value, "sse": n.SSE})
		return
	}
	leaves(n.Left, out)
	leaves(n.Right, out)
}

func train(rootDir string) error {
	s, counts, err := buildSamples(2023)
	if err != nil {
		return err
	}
	fmt.Printf("TRAIN_TOTAL %d\n", len(s))
	tree := buildTree(s, 0, 3, 2000)
	model := Model{Version: "v189", Features: featureNames, MaxDepth: 3, MinLeaf: 2000,
		TrainStart: "2023-01-01T00:00:00Z", TrainEnd: "2024-01-01T00:00:00Z",
		TrainSamples: len(s), Root: tree}
	b, _ := json.MarshalIndent(model, "", "  ")
	if err := os.WriteFile(filepath.Join(rootDir, "model.json"), b, 0644); err != nil {
		return err
	}
	ls := []map[string]interface{}{}
	leaves(tree, &ls)
	summary := map[string]interface{}{
		"version": "v189", "train_year": 2023, "train_samples": len(s),
		"samples_by_symbol": counts, "root_sse": tree.SSE, "leaves": ls,
		"validation_read": false,
	}
	sb, _ := json.MarshalIndent(summary, "", "  ")
	if err := os.WriteFile(filepath.Join(rootDir, "results", "train_summary.json"), sb, 0644); err != nil {
		return err
	}
	fmt.Println(string(b))
	return nil
}

func stat(ev []Event) Stats {
	s := Stats{N: len(ev)}
	if len(ev) == 0 {
		return s
	}
	for _, e := range ev {
		s.MeanR1 += e.R1
		s.MeanR4 += e.R4
		s.MeanR12 += e.R12
		if e.R12 > 0 {
			s.WinR12++
		}
		if e.Side == "LONG" {
			s.Long++
		} else {
			s.Short++
		}
	}
	n := float64(len(ev))
	s.MeanR1 /= n
	s.MeanR4 /= n
	s.MeanR12 /= n
	s.WinR12 /= n
	return s
}

func validate(rootDir string) error {
	mb, err := os.ReadFile(filepath.Join(rootDir, "model.json"))
	if err != nil {
		return err
	}
	var model Model
	if err = json.Unmarshal(mb, &model); err != nil {
		return err
	}
	start, end := yearBounds(2024)
	events := []Event{}
	bySymbol := map[string][]Event{}
	byHalf := map[string][]Event{"H1": {}, "H2": {}}
	source := map[string]int{}
	for _, sym := range symbols {
		b, err := loadBars(sym, 2024)
		if err != nil {
			return err
		}
		source[sym] = len(b)
		var prevPred float64
		havePrev := false
		prevT := int64(0)
		n := 0
		for i := 24; i+12 < len(b); i++ {
			t := b[i].T
			if t < start || t >= end {
				continue
			}
			if b[i+12].T-t != 12*hourMS || b[i+1].T-t != hourMS || b[i+12].T >= end {
				continue
			}
			x, ok := features(b, i)
			if !ok {
				continue
			}
			p := predict(model.Root, x)
			if !havePrev || t-prevT != hourMS {
				prevPred = p
				prevT = t
				havePrev = true
				continue
			}
			side := 0.0
			if prevPred <= 0 && p > 0 {
				side = 1
			} else if prevPred >= 0 && p < 0 {
				side = -1
			}
			if side != 0 {
				entry := b[i+1].Open
				vals := map[int]float64{}
				good := entry > 0
				for _, h := range []int{1, 4, 12} {
					if i+h >= len(b) || b[i+h].T-t != int64(h)*hourMS || b[i+h].Close <= 0 {
						good = false
						break
					}
					vals[h] = side * math.Log(b[i+h].Close/entry)
				}
				if good {
					dt := time.UnixMilli(t).UTC()
					half := "H1"
					if dt.Month() >= 7 {
						half = "H2"
					}
					e := Event{Symbol: sym, SignalTime: t, Side: map[bool]string{true: "LONG", false: "SHORT"}[side > 0],
						PredPrev: prevPred, PredNow: p, R1: vals[1], R4: vals[4], R12: vals[12], Half: half}
					events = append(events, e)
					bySymbol[sym] = append(bySymbol[sym], e)
					byHalf[half] = append(byHalf[half], e)
					n++
				}
			}
			prevPred = p
			prevT = t
		}
		fmt.Printf("VALIDATE %s events %d\n", sym, n)
	}
	overall := stat(events)
	bstats := map[string]Stats{}
	positive := 0
	for _, s := range symbols {
		z := stat(bySymbol[s])
		bstats[s] = z
		if z.MeanR12 > 0 {
			positive++
		}
	}
	hstats := map[string]Stats{"H1": stat(byHalf["H1"]), "H2": stat(byHalf["H2"])}
	weeks := 366.0 / 7.0 * float64(len(symbols))
	freq := float64(len(events)) / weeks
	gate := overall.MeanR12 >= 0.002 && positive >= 6 && freq >= 0.30 && hstats["H1"].MeanR12 > 0 && hstats["H2"].MeanR12 > 0
	summary := map[string]interface{}{
		"version": "v189", "status": map[bool]string{true: "raw_gate_pass", false: "frozen_failed_raw_validation"}[gate],
		"raw_gate_pass": gate, "events": len(events), "positive_symbols": fmt.Sprintf("%d/10", positive),
		"frequency_per_symbol_week": freq, "overall": overall, "by_symbol": bstats, "by_half": hstats,
		"source_bars": source, "strict_2024_run": false, "oos_2025_read": false, "oos_2026_read": false,
	}
	eb, _ := json.MarshalIndent(events, "", "  ")
	os.WriteFile(filepath.Join(rootDir, "results", "validation_events.json"), eb, 0644)
	sb, _ := json.MarshalIndent(summary, "", "  ")
	os.WriteFile(filepath.Join(rootDir, "results", "validation_summary.json"), sb, 0644)
	fmt.Println(string(sb))
	return nil
}

func main() {
	mode := flag.String("mode", "train", "train or validate")
	root := flag.String("root", "strategy_templates/research/shared-native-feature-tree/2026-10-03-train2023-validate2024", "research root")
	flag.Parse()
	var err error
	switch strings.ToLower(*mode) {
	case "train":
		err = train(*root)
	case "validate":
		err = validate(*root)
	default:
		err = fmt.Errorf("unknown mode %s", *mode)
	}
	must(err)
}
