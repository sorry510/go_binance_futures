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
	N, Long, Short int
	R1, R4, R12    float64
}

func (a *Agg) Add(side string, r1, r4, r12 float64) {
	a.N++
	if side == "LONG" {
		a.Long++
	} else {
		a.Short++
	}
	a.R1 += r1
	a.R4 += r4
	a.R12 += r12
}
func mean(x float64, n int) float64 {
	if n == 0 {
		return 0
	}
	return x / float64(n)
}

func eom(b []historicalmarket.ReplayKline, i int) (float64, bool) {
	if i < 1 || b[i].QuoteVolume <= 0 {
		return 0, false
	}
	mid := 0.5 * (b[i].High + b[i].Low)
	prev := 0.5 * (b[i-1].High + b[i-1].Low)
	return (mid - prev) * (b[i].High - b[i].Low) / b[i].QuoteVolume, true
}
func eomMean(b []historicalmarket.ReplayKline, i int) (float64, bool) {
	if i < 14 {
		return 0, false
	}
	var s float64
	for j := i - 13; j <= i; j++ {
		x, ok := eom(b, j)
		if !ok {
			return 0, false
		}
		s += x
	}
	return s / 14, true
}

func main() {
	const root = "strategy_templates/research/quote-ease-of-movement-14/2026-10-01-early-gate"
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
	start := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC).UnixMilli()
	end := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC).UnixMilli()
	loadStart := start - int64(36*time.Hour/time.Millisecond)
	symbols := []string{"SOLUSDT", "DOGEUSDT", "LTCUSDT", "AVAXUSDT", "UNIUSDT", "ZECUSDT"}
	repo := historicalmarket.NewRepository(nil)
	ctx := context.Background()
	out, err := os.Create(root + "/results/events.csv")
	if err != nil {
		panic(err)
	}
	defer out.Close()
	cw := csv.NewWriter(out)
	defer cw.Flush()
	_ = cw.Write([]string{"symbol", "signal_time", "side", "prev_eom14", "eom14", "entry_time", "r1", "r4", "r12"})
	total := Agg{}
	bySym := map[string]Agg{}
	byYear := map[int]Agg{}
	bySide := map[string]Agg{}
	for _, sym := range symbols {
		fmt.Println("LOAD", sym)
		bars, err := repo.LoadReplayKlines(ctx, historicalmarket.MarketFuturesUSDT, sym, "1h", loadStart, end-1)
		if err != nil {
			panic(err)
		}
		a := Agg{}
		for i := 15; i+12 < len(bars); i++ {
			if bars[i].CloseTime < start || bars[i].CloseTime >= end || bars[i+12].CloseTime >= end {
				continue
			}
			prev, ok1 := eomMean(bars, i-1)
			now, ok2 := eomMean(bars, i)
			if !ok1 || !ok2 {
				continue
			}
			side := ""
			dir := 0.0
			if prev <= 0 && now > 0 {
				side = "LONG"
				dir = 1
			}
			if prev >= 0 && now < 0 {
				side = "SHORT"
				dir = -1
			}
			if side == "" {
				continue
			}
			entry := bars[i+1].Open
			if entry <= 0 {
				continue
			}
			r1 := dir * math.Log(bars[i+1].Close/entry)
			r4 := dir * math.Log(bars[i+4].Close/entry)
			r12 := dir * math.Log(bars[i+12].Close/entry)
			a.Add(side, r1, r4, r12)
			total.Add(side, r1, r4, r12)
			y := time.UnixMilli(bars[i].CloseTime).UTC().Year()
			yy := byYear[y]
			yy.Add(side, r1, r4, r12)
			byYear[y] = yy
			ss := bySide[side]
			ss.Add(side, r1, r4, r12)
			bySide[side] = ss
			_ = cw.Write([]string{sym, strconv.FormatInt(bars[i].CloseTime, 10), side,
				strconv.FormatFloat(prev, 'g', -1, 64), strconv.FormatFloat(now, 'g', -1, 64),
				strconv.FormatInt(bars[i+1].OpenTime, 10), strconv.FormatFloat(r1, 'g', -1, 64),
				strconv.FormatFloat(r4, 'g', -1, 64), strconv.FormatFloat(r12, 'g', -1, 64)})
		}
		bySym[sym] = a
		fmt.Printf("SYM %s n=%d long=%d short=%d r1=%.6f r4=%.6f r12=%.6f\n", sym, a.N, a.Long, a.Short, mean(a.R1, a.N), mean(a.R4, a.N), mean(a.R12, a.N))
	}
	cw.Flush()
	if err := cw.Error(); err != nil {
		panic(err)
	}
	weeks := float64(end-start) / float64((7*24*time.Hour)/time.Millisecond) * float64(len(symbols))
	freq := float64(total.N) / weeks
	positive := 0
	for _, a := range bySym {
		if mean(a.R12, a.N) > 0 {
			positive++
		}
	}
	fmt.Printf("ALL n=%d long=%d short=%d r1=%.6f r4=%.6f r12=%.6f positive=%d/%d freq=%.6f\n",
		total.N, total.Long, total.Short, mean(total.R1, total.N), mean(total.R4, total.N), mean(total.R12, total.N), positive, len(symbols), freq)
	years := make([]int, 0, len(byYear))
	for y := range byYear {
		years = append(years, y)
	}
	sort.Ints(years)
	for _, y := range years {
		a := byYear[y]
		fmt.Printf("YEAR %d n=%d r1=%.6f r4=%.6f r12=%.6f\n", y, a.N, mean(a.R1, a.N), mean(a.R4, a.N), mean(a.R12, a.N))
	}
	for _, side := range []string{"LONG", "SHORT"} {
		a := bySide[side]
		fmt.Printf("SIDE %s n=%d r1=%.6f r4=%.6f r12=%.6f\n", side, a.N, mean(a.R1, a.N), mean(a.R4, a.N), mean(a.R12, a.N))
	}
	gate := mean(total.R12, total.N) >= 0.002 && positive >= 4 && freq >= 0.30
	for _, y := range []int{2023, 2024} {
		a, ok := byYear[y]
		if !ok || mean(a.R12, a.N) <= 0 {
			gate = false
		}
	}
	j := map[string]interface{}{
		"status": func() string {
			if gate {
				return "promoted_early_gate"
			}
			return "frozen_failed_early_gate"
		}(),
		"events": total.N, "long": total.Long, "short": total.Short,
		"mean_r1": mean(total.R1, total.N), "mean_r4": mean(total.R4, total.N), "mean_r12": mean(total.R12, total.N),
		"positive_symbols": fmt.Sprintf("%d/%d", positive, len(symbols)), "frequency_per_symbol_week": freq,
		"gate_pass": gate, "oos_evaluated": false, "strict_engine_run": false,
		"by_symbol": map[string]interface{}{}, "by_year": map[string]interface{}{}, "by_side": map[string]interface{}{},
	}
	for _, sym := range symbols {
		a := bySym[sym]
		j["by_symbol"].(map[string]interface{})[sym] = map[string]interface{}{"events": a.N, "long": a.Long, "short": a.Short, "mean_r1": mean(a.R1, a.N), "mean_r4": mean(a.R4, a.N), "mean_r12": mean(a.R12, a.N)}
	}
	for _, y := range years {
		a := byYear[y]
		j["by_year"].(map[string]interface{})[strconv.Itoa(y)] = map[string]interface{}{"events": a.N, "mean_r1": mean(a.R1, a.N), "mean_r4": mean(a.R4, a.N), "mean_r12": mean(a.R12, a.N)}
	}
	for _, side := range []string{"LONG", "SHORT"} {
		a := bySide[side]
		j["by_side"].(map[string]interface{})[side] = map[string]interface{}{"events": a.N, "mean_r1": mean(a.R1, a.N), "mean_r4": mean(a.R4, a.N), "mean_r12": mean(a.R12, a.N)}
	}
	b, _ := json.MarshalIndent(j, "", "  ")
	if err := os.WriteFile(root+"/results/summary.json", b, 0644); err != nil {
		panic(err)
	}
}
