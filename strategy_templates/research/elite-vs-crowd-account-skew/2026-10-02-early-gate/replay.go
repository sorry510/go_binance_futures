package main

import (
	"archive/zip"
	"bytes"
	"context"
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"sync"
	"time"

	_ "go_binance_futures/bootstrap"

	"github.com/beego/beego/v2/core/config"
	_ "github.com/go-sql-driver/mysql"
)

const (
	root  = "strategy_templates/research/elite-vs-crowd-account-skew/2026-10-02-early-gate"
	hour  = int64(time.Hour / time.Millisecond)
	cache = "/tmp/v147-binance-metrics"
)

type Point struct {
	Hour  int64
	Score float64
}

type Bar struct {
	Open  float64
	Close float64
}

type Event struct {
	Symbol string
	Time   int64
	Year   int
	Side   string
	Prev   float64
	Now    float64
	R1     float64
	R4     float64
	R12    float64
}

type Agg struct {
	N, Long, Short int
	S1, S4, S12    float64
	W12            int
}

func (a *Agg) Add(e Event) {
	a.N++
	if e.Side == "LONG" {
		a.Long++
	} else {
		a.Short++
	}
	a.S1 += e.R1
	a.S4 += e.R4
	a.S12 += e.R12
	if e.R12 > 0 {
		a.W12++
	}
}

func mean(v float64, n int) float64 {
	if n == 0 {
		return 0
	}
	return v / float64(n)
}

func summary(a Agg) map[string]interface{} {
	return map[string]interface{}{
		"n": a.N, "long": a.Long, "short": a.Short,
		"mean_r1": mean(a.S1, a.N), "mean_r4": mean(a.S4, a.N),
		"mean_r12": mean(a.S12, a.N), "win_r12": mean(float64(a.W12), a.N),
	}
}
func metricURL(symbol, day string) string {
	return fmt.Sprintf(
		"https://data.binance.vision/data/futures/um/daily/metrics/%s/%s-metrics-%s.zip",
		symbol, symbol, day,
	)
}

func fetchMetricZip(client *http.Client, symbol, day string) ([]byte, error) {
	if err := os.MkdirAll(cache, 0755); err != nil {
		return nil, err
	}
	path := filepath.Join(cache, symbol+"-"+day+".zip")
	if b, err := os.ReadFile(path); err == nil {
		return b, nil
	}
	var last error
	for i := 0; i < 5; i++ {
		req, _ := http.NewRequest("GET", metricURL(symbol, day), nil)
		req.Header.Set("User-Agent", "Mozilla/5.0")
		resp, err := client.Do(req)
		if err == nil && resp.StatusCode == 200 {
			b, e := io.ReadAll(resp.Body)
			resp.Body.Close()
			if e == nil {
				if e = os.WriteFile(path, b, 0644); e == nil {
					return b, nil
				}
			}
			last = e
		} else if err == nil {
			resp.Body.Close()
			if resp.StatusCode == 404 {
				return nil, nil
			}
			last = fmt.Errorf("http %d", resp.StatusCode)
		} else {
			last = err
		}
		time.Sleep(time.Duration(i+1) * 500 * time.Millisecond)
	}
	return nil, last
}
func parseMetricZip(b []byte) ([]Point, error) {
	if len(b) == 0 {
		return nil, nil
	}
	zr, err := zip.NewReader(bytes.NewReader(b), int64(len(b)))
	if err != nil {
		return nil, err
	}
	rc, err := zr.File[0].Open()
	if err != nil {
		return nil, err
	}
	defer rc.Close()

	cr := csv.NewReader(rc)
	head, err := cr.Read()
	if err != nil {
		return nil, err
	}
	idx := map[string]int{}
	for i, h := range head {
		idx[h] = i
	}
	want := []string{"create_time", "count_toptrader_long_short_ratio", "count_long_short_ratio"}
	for _, k := range want {
		if _, ok := idx[k]; !ok {
			return nil, fmt.Errorf("missing metrics column %s", k)
		}
	}

	last := map[int64]Point{}
	for {
		r, e := cr.Read()
		if e == io.EOF {
			break
		}
		if e != nil {
			return nil, e
		}
		t, e := time.ParseInLocation("2006-01-02 15:04:05", r[idx["create_time"]], time.UTC)
		if e != nil {
			continue
		}
		topAccount, e1 := strconv.ParseFloat(r[idx["count_toptrader_long_short_ratio"]], 64)
		globalAccount, e2 := strconv.ParseFloat(r[idx["count_long_short_ratio"]], 64)
		if e1 != nil || e2 != nil || topAccount <= 0 || globalAccount <= 0 {
			continue
		}
		h := t.Truncate(time.Hour).UnixMilli()
		last[h] = Point{Hour: h, Score: math.Log(topAccount / globalAccount)}
	}
	out := make([]Point, 0, len(last))
	for _, p := range last {
		out = append(out, p)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Hour < out[j].Hour })
	return out, nil
}

func days(start, end time.Time) []string {
	var out []string
	for d := start; !d.After(end); d = d.AddDate(0, 0, 1) {
		out = append(out, d.Format("2006-01-02"))
	}
	return out
}

type metricTask struct {
	Symbol string
	Day    string
}

type metricResult struct {
	Symbol  string
	Day     string
	Points  []Point
	Missing bool
	Err     error
}

func loadMetrics(symbols []string) (map[string][]Point, map[string]int, error) {
	ds := days(time.Date(2022, 12, 31, 0, 0, 0, 0, time.UTC),
		time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC))
	tasks := make(chan metricTask)
	results := make(chan metricResult)
	client := &http.Client{Timeout: 35 * time.Second}
	var wg sync.WaitGroup

	for w := 0; w < 16; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for t := range tasks {
				b, err := fetchMetricZip(client, t.Symbol, t.Day)
				if err != nil {
					results <- metricResult{Symbol: t.Symbol, Day: t.Day, Err: err}
					continue
				}
				if len(b) == 0 {
					results <- metricResult{Symbol: t.Symbol, Day: t.Day, Missing: true}
					continue
				}
				ps, err := parseMetricZip(b)
				results <- metricResult{Symbol: t.Symbol, Day: t.Day, Points: ps, Err: err}
			}
		}()
	}
	go func() {
		for _, s := range symbols {
			for _, day := range ds {
				tasks <- metricTask{Symbol: s, Day: day}
			}
		}
		close(tasks)
		wg.Wait()
		close(results)
	}()

	out := map[string][]Point{}
	missing := map[string]int{}
	for r := range results {
		if r.Err != nil {
			return nil, nil, fmt.Errorf("%s %s: %w", r.Symbol, r.Day, r.Err)
		}
		if r.Missing {
			missing[r.Symbol]++
			continue
		}
		out[r.Symbol] = append(out[r.Symbol], r.Points...)
	}
	for _, s := range symbols {
		sort.Slice(out[s], func(i, j int) bool { return out[s][i].Hour < out[s][j].Hour })
	}
	return out, missing, nil
}
func openDB() (*sql.DB, error) {
	u, _ := config.String("database::username")
	pw, _ := config.String("database::password")
	h, _ := config.String("database::host")
	pt, _ := config.String("database::port")
	dbn, _ := config.String("database::dbname")
	if dbn != "go_bn_test" {
		return nil, fmt.Errorf("refuse database %s", dbn)
	}
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4", u, pw, h, pt, dbn)
	return sql.Open("mysql", dsn)
}

func loadBars(db *sql.DB, symbol string, start, end int64) (map[int64]Bar, error) {
	q := "SELECT open_time,open_price,close_price FROM market_klines_1h " +
		"WHERE market='futures_usdt' AND symbol=? AND open_time>=? AND open_time<? ORDER BY open_time"
	rows, err := db.Query(q, symbol, start, end)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[int64]Bar{}
	for rows.Next() {
		var t int64
		var o, c float64
		if err := rows.Scan(&t, &o, &c); err != nil {
			return nil, err
		}
		out[t] = Bar{Open: o, Close: c}
	}
	return out, rows.Err()
}
func buildEvents(symbol string, points []Point, bars map[int64]Bar, start, end int64) []Event {
	var out []Event
	for i := 1; i < len(points); i++ {
		prev, now := points[i-1], points[i]
		if now.Hour-prev.Hour != hour {
			continue
		}
		signalTime := now.Hour + hour
		if signalTime < start || signalTime >= end {
			continue
		}
		side := 0.0
		sideName := ""
		if prev.Score <= 0 && now.Score > 0 {
			side, sideName = 1, "LONG"
		} else if prev.Score >= 0 && now.Score < 0 {
			side, sideName = -1, "SHORT"
		} else {
			continue
		}

		entryT := signalTime
		entry, ok := bars[entryT]
		if !ok || entry.Open <= 0 {
			continue
		}
		signalYear := time.UnixMilli(signalTime).UTC().Year()
		end12Close := entryT + 12*hour
		if time.UnixMilli(end12Close).UTC().Year() != signalYear {
			continue
		}
		get := func(h int) (float64, bool) {
			b, ok := bars[entryT+int64(h-1)*hour]
			if !ok || b.Close <= 0 {
				return 0, false
			}
			return side * math.Log(b.Close/entry.Open), true
		}
		r1, ok1 := get(1)
		r4, ok4 := get(4)
		r12, ok12 := get(12)
		if !ok1 || !ok4 || !ok12 {
			continue
		}
		out = append(out, Event{
			Symbol: symbol, Time: now.Hour, Year: signalYear, Side: sideName,
			Prev: prev.Score, Now: now.Score, R1: r1, R4: r4, R12: r12,
		})
	}
	return out
}

func main() {
	symbols := []string{"SOLUSDT", "DOGEUSDT", "LTCUSDT", "AVAXUSDT", "UNIUSDT", "ZECUSDT"}
	start := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC).UnixMilli()
	end := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC).UnixMilli()

	fmt.Println("LOAD METRICS")
	metrics, missing, err := loadMetrics(symbols)
	if err != nil {
		panic(err)
	}
	for _, s := range symbols {
		fmt.Printf("METRICS %s points=%d missing_days=%d\n", s, len(metrics[s]), missing[s])
	}
	db, err := openDB()
	if err != nil {
		panic(err)
	}
	defer db.Close()
	if err := db.PingContext(context.Background()); err != nil {
		panic(err)
	}

	out, err := os.Create(root + "/results/events.csv")
	if err != nil {
		panic(err)
	}
	defer out.Close()
	cw := csv.NewWriter(out)
	defer cw.Flush()
	_ = cw.Write([]string{
		"symbol", "signal_hour", "year", "side", "score_prev", "score_now",
		"entry_time", "r1", "r4", "r12",
	})

	all := []Event{}
	bySym := map[string]Agg{}
	byYear := map[int]Agg{}
	bySide := map[string]Agg{}

	for _, s := range symbols {
		bars, err := loadBars(db, s, start, end)
		if err != nil {
			panic(fmt.Errorf("%s bars: %w", s, err))
		}
		fmt.Printf("BARS %s %d\n", s, len(bars))
		evs := buildEvents(s, metrics[s], bars, start, end)
		a := Agg{}
		for _, e := range evs {
			all = append(all, e)
			a.Add(e)
			y := byYear[e.Year]
			y.Add(e)
			byYear[e.Year] = y
			sa := bySide[e.Side]
			sa.Add(e)
			bySide[e.Side] = sa
			_ = cw.Write([]string{
				e.Symbol, strconv.FormatInt(e.Time, 10), strconv.Itoa(e.Year), e.Side,
				strconv.FormatFloat(e.Prev, 'g', -1, 64), strconv.FormatFloat(e.Now, 'g', -1, 64),
				strconv.FormatInt(e.Time+hour, 10),
				strconv.FormatFloat(e.R1, 'g', -1, 64), strconv.FormatFloat(e.R4, 'g', -1, 64),
				strconv.FormatFloat(e.R12, 'g', -1, 64),
			})
		}
		bySym[s] = a
		fmt.Printf("SYM %s n=%d r1=%.6f r4=%.6f r12=%.6f\n",
			s, a.N, mean(a.S1, a.N), mean(a.S4, a.N), mean(a.S12, a.N))
	}
	cw.Flush()
	if err := cw.Error(); err != nil {
		panic(err)
	}

	total := Agg{}
	for _, e := range all {
		total.Add(e)
	}
	weeks := float64(end-start) / float64((7*24*time.Hour)/time.Millisecond) * float64(len(symbols))
	freq := float64(total.N) / weeks
	positive := 0
	for _, s := range symbols {
		a := bySym[s]
		if mean(a.S12, a.N) > 0 {
			positive++
		}
	}

	fmt.Printf("ALL n=%d r1=%.6f r4=%.6f r12=%.6f positive=%d/%d freq=%.6f\n",
		total.N, mean(total.S1, total.N), mean(total.S4, total.N), mean(total.S12, total.N),
		positive, len(symbols), freq)

	years := make([]int, 0, len(byYear))
	for y := range byYear {
		years = append(years, y)
	}
	sort.Ints(years)
	for _, y := range years {
		a := byYear[y]
		fmt.Printf("YEAR %d n=%d r1=%.6f r4=%.6f r12=%.6f\n",
			y, a.N, mean(a.S1, a.N), mean(a.S4, a.N), mean(a.S12, a.N))
	}
	for _, side := range []string{"LONG", "SHORT"} {
		a := bySide[side]
		fmt.Printf("SIDE %s n=%d r1=%.6f r4=%.6f r12=%.6f\n",
			side, a.N, mean(a.S1, a.N), mean(a.S4, a.N), mean(a.S12, a.N))
	}
	gate := mean(total.S12, total.N) >= 0.002 && positive >= 4 && freq >= 0.30
	if a, ok := byYear[2023]; ok {
		gate = gate && mean(a.S12, a.N) > 0
	} else {
		gate = false
	}
	if a, ok := byYear[2024]; ok {
		gate = gate && mean(a.S12, a.N) > 0
	} else {
		gate = false
	}

	bySymOut := map[string]interface{}{}
	for _, s := range symbols {
		bySymOut[s] = summary(bySym[s])
	}
	byYearOut := map[string]interface{}{}
	for _, y := range years {
		byYearOut[strconv.Itoa(y)] = summary(byYear[y])
	}
	bySideOut := map[string]interface{}{}
	for _, side := range []string{"LONG", "SHORT"} {
		bySideOut[side] = summary(bySide[side])
	}

	status := "frozen_failed_early_gate"
	if gate {
		status = "promote_oos"
	}
	sum := map[string]interface{}{
		"version":                   "v158",
		"status":                    status,
		"gate_pass":                 gate,
		"events":                    total.N,
		"mean_r1":                   mean(total.S1, total.N),
		"mean_r4":                   mean(total.S4, total.N),
		"mean_r12":                  mean(total.S12, total.N),
		"positive_symbols":          fmt.Sprintf("%d/%d", positive, len(symbols)),
		"frequency_per_symbol_week": freq,
		"by_symbol":                 bySymOut,
		"by_year":                   byYearOut,
		"by_side":                   bySideOut,
		"metrics_points":            map[string]int{},
		"missing_metric_days":       missing,
		"oos_evaluated":             false,
		"strict_engine_run":         false,
	}
	for _, s := range symbols {
		sum["metrics_points"].(map[string]int)[s] = len(metrics[s])
	}
	b, _ := json.MarshalIndent(sum, "", "  ")
	if err := os.WriteFile(root+"/results/summary.json", b, 0644); err != nil {
		panic(err)
	}
}
