package main

import (
	"context"
	"encoding/csv"
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

type Agg struct {
	N           int
	R1, R4, R12 float64
}

func (a *Agg) Add(r1, r4, r12 float64) { a.N++; a.R1 += r1; a.R4 += r4; a.R12 += r12 }
func avg(x float64, n int) float64 {
	if n == 0 {
		return 0
	}
	return x / float64(n)
}
func predictor(rows []historicalmarket.ReplayKline, i int) (float64, bool) {
	if i < 25 || rows[i].QuoteVolume <= 0 || rows[i-1].QuoteVolume <= 0 {
		return 0, false
	}
	xs := make([]float64, 0, 24)
	ys := make([]float64, 0, 24)
	for k := i - 24; k <= i-1; k++ {
		if k-1 < 0 || k+1 >= len(rows) {
			return 0, false
		}
		if rows[k].QuoteVolume <= 0 || rows[k-1].QuoteVolume <= 0 || rows[k+1].Open <= 0 || rows[k+1].Close <= 0 {
			return 0, false
		}
		xs = append(xs, math.Log(rows[k].QuoteVolume/rows[k-1].QuoteVolume))
		ys = append(ys, math.Log(rows[k+1].Close/rows[k+1].Open))
	}
	var mx, my float64
	for j := range xs {
		mx += xs[j]
		my += ys[j]
	}
	mx /= float64(len(xs))
	my /= float64(len(ys))
	var cov float64
	for j := range xs {
		cov += (xs[j] - mx) * (ys[j] - my)
	}
	cov /= float64(len(xs))
	xnow := math.Log(rows[i].QuoteVolume / rows[i-1].QuoteVolume)
	return cov * xnow, true
}
func main() {
	const root = "strategy_templates/research/lagged-quote-volume-price-lead/2026-10-01-early-gate"
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

	symbols := []string{"SOLUSDT", "DOGEUSDT", "LTCUSDT", "AVAXUSDT", "UNIUSDT", "ZECUSDT"}
	start := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC).UnixMilli()
	end := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC).UnixMilli()
	loadStart := start - int64(72*time.Hour/time.Millisecond)
	repo := historicalmarket.NewRepository(nil)
	ctx := context.Background()

	f, err := os.Create(root + "/results/events.csv")
	if err != nil {
		panic(err)
	}
	defer f.Close()
	cw := csv.NewWriter(f)
	defer cw.Flush()
	_ = cw.Write([]string{"symbol", "signal_time", "side", "predictor_prev", "predictor_now", "entry_time", "r1", "r4", "r12"})
	total := Agg{}
	bySym := map[string]Agg{}
	byYear := map[int]Agg{}
	bySide := map[string]Agg{}
	for _, sym := range symbols {
		fmt.Println("LOAD", sym)
		rows, err := repo.LoadReplayKlines(ctx, historicalmarket.MarketFuturesUSDT, sym, "1h", loadStart, end-1)
		if err != nil {
			panic(err)
		}
		a := Agg{}
		for i := 26; i+12 < len(rows); i++ {
			signal := rows[i]
			if signal.CloseTime < start || signal.CloseTime >= end || rows[i+12].CloseTime >= end {
				continue
			}
			prev, ok1 := predictor(rows, i-1)
			now, ok2 := predictor(rows, i)
			if !ok1 || !ok2 {
				continue
			}
			dir := 0.0
			side := ""
			if prev <= 0 && now > 0 {
				dir = 1
				side = "LONG"
			}
			if prev >= 0 && now < 0 {
				dir = -1
				side = "SHORT"
			}
			if dir == 0 {
				continue
			}
			entry := rows[i+1].Open
			if entry <= 0 {
				continue
			}
			r1 := dir * math.Log(rows[i+1].Close/entry)
			r4 := dir * math.Log(rows[i+4].Close/entry)
			r12 := dir * math.Log(rows[i+12].Close/entry)
			a.Add(r1, r4, r12)
			total.Add(r1, r4, r12)
			yy := byYear[time.UnixMilli(signal.CloseTime).UTC().Year()]
			yy.Add(r1, r4, r12)
			byYear[time.UnixMilli(signal.CloseTime).UTC().Year()] = yy
			ss := bySide[side]
			ss.Add(r1, r4, r12)
			bySide[side] = ss
			_ = cw.Write([]string{sym, strconv.FormatInt(signal.CloseTime, 10), side, strconv.FormatFloat(prev, 'g', -1, 64), strconv.FormatFloat(now, 'g', -1, 64), strconv.FormatInt(rows[i+1].OpenTime, 10), strconv.FormatFloat(r1, 'g', -1, 64), strconv.FormatFloat(r4, 'g', -1, 64), strconv.FormatFloat(r12, 'g', -1, 64)})
		}
		bySym[sym] = a
		fmt.Printf("SYM %s n=%d r1=%.6f r4=%.6f r12=%.6f\n", sym, a.N, avg(a.R1, a.N), avg(a.R4, a.N), avg(a.R12, a.N))
	}
	cw.Flush()
	if err := cw.Error(); err != nil {
		panic(err)
	}
	weeks := float64(end-start) / float64((7*24*time.Hour)/time.Millisecond) * float64(len(symbols))
	positive := 0
	for _, a := range bySym {
		if avg(a.R12, a.N) > 0 {
			positive++
		}
	}
	freq := float64(total.N) / weeks
	fmt.Printf("ALL n=%d r1=%.6f r4=%.6f r12=%.6f positive=%d/%d freq=%.6f\n", total.N, avg(total.R1, total.N), avg(total.R4, total.N), avg(total.R12, total.N), positive, len(symbols), freq)
	years := make([]int, 0, len(byYear))
	for y := range byYear {
		years = append(years, y)
	}
	sort.Ints(years)
	for _, y := range years {
		a := byYear[y]
		fmt.Printf("YEAR %d n=%d r1=%.6f r4=%.6f r12=%.6f\n", y, a.N, avg(a.R1, a.N), avg(a.R4, a.N), avg(a.R12, a.N))
	}
	sides := []string{"LONG", "SHORT"}
	for _, s := range sides {
		a := bySide[s]
		fmt.Printf("SIDE %s n=%d r1=%.6f r4=%.6f r12=%.6f\n", s, a.N, avg(a.R1, a.N), avg(a.R4, a.N), avg(a.R12, a.N))
	}
	summary := map[string]interface{}{
		"events": total.N, "mean_r1": avg(total.R1, total.N), "mean_r4": avg(total.R4, total.N), "mean_r12": avg(total.R12, total.N),
		"positive_symbols": fmt.Sprintf("%d/%d", positive, len(symbols)), "frequency_per_symbol_week": freq,
		"oos_evaluated": false, "strict_engine_run": false,
	}
	b, _ := json.MarshalIndent(summary, "", "  ")
	_ = os.WriteFile(root+"/results/summary.json", b, 0644)
}
