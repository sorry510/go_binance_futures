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

func popStd(v []float64) float64 {
	if len(v) == 0 {
		return 0
	}
	var s float64
	for _, x := range v {
		s += x
	}
	m := s / float64(len(v))
	var q float64
	for _, x := range v {
		d := x - m
		q += d * d
	}
	return math.Sqrt(q / float64(len(v)))
}

func regularGap(a, b int64) bool {
	d := b - a
	return d >= int64(7*time.Hour+30*time.Minute)/int64(time.Millisecond) &&
		d <= int64(8*time.Hour+30*time.Minute)/int64(time.Millisecond)
}
func ratioAt(f []historicalmarket.FundingRate, i int) (float64, float64, bool) {
	// Window ending at i: prior 21 settlements, then recent 3 settlements.
	if i < 23 {
		return 0, 0, false
	}
	for j := i - 22; j <= i; j++ {
		if !regularGap(f[j-1].FundingTime, f[j].FundingTime) {
			return 0, 0, false
		}
	}
	base := make([]float64, 0, 21)
	recent := make([]float64, 0, 3)
	for j := i - 23; j <= i-3; j++ {
		base = append(base, f[j].FundingRate)
	}
	var recentSum float64
	for j := i - 2; j <= i; j++ {
		recent = append(recent, f[j].FundingRate)
		recentSum += f[j].FundingRate
	}
	bs := popStd(base)
	rs := popStd(recent)
	if bs <= 0 {
		return 0, 0, false
	}
	return rs / bs, recentSum, true
}

func eventAt(f []historicalmarket.FundingRate, i int) (float64, float64, bool) {
	if i < 24 {
		return 0, 0, false
	}
	prev, _, ok1 := ratioAt(f, i-1)
	now, recentSum, ok2 := ratioAt(f, i)
	if !ok1 || !ok2 || !(prev <= 1 && now > 1) || recentSum == 0 {
		return 0, 0, false
	}
	return now, recentSum, true
}

func main() {
	const root = "strategy_templates/research/funding-volatility-expansion-fade/2026-10-01-early-gate"
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
	warm := start - int64(10*24*time.Hour/time.Millisecond)
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
	_ = cw.Write([]string{"symbol", "funding_time", "side", "ratio", "recent_sum", "entry_time", "r1", "r4", "r12"})
	total := Agg{}
	bySym := map[string]Agg{}
	byYear := map[int]Agg{}
	bySide := map[string]Agg{}

	for _, sym := range symbols {
		fmt.Println("LOAD", sym)
		funding, err := repo.LoadFunding(ctx, historicalmarket.MarketFuturesUSDT, sym, warm, end-1)
		if err != nil {
			panic(err)
		}
		bars, err := repo.LoadReplayKlines(ctx, historicalmarket.MarketFuturesUSDT, sym, "1h", start, end-1)
		if err != nil {
			panic(err)
		}
		byOpen := make(map[int64]historicalmarket.ReplayKline, len(bars))
		for _, b := range bars {
			byOpen[b.OpenTime] = b
		}

		a := Agg{}
		for i := 24; i < len(funding); i++ {
			ratio, recentSum, ok := eventAt(funding, i)
			if !ok {
				continue
			}
			ft := funding[i].FundingTime
			if ft < start || ft >= end {
				continue
			}
			entryTime := ((ft / 3600000) + 1) * 3600000
			if entryTime+12*3600000 >= end {
				continue
			}
			entry, ok0 := byOpen[entryTime]
			b1, ok1 := byOpen[entryTime]
			b4, ok4 := byOpen[entryTime+3*3600000]
			b12, ok12 := byOpen[entryTime+11*3600000]
			if !ok0 || !ok1 || !ok4 || !ok12 || entry.Open <= 0 {
				continue
			}

			dir := -1.0
			side := "SHORT"
			if recentSum < 0 {
				dir = 1
				side = "LONG"
			}
			r1 := dir * math.Log(b1.Close/entry.Open)
			r4 := dir * math.Log(b4.Close/entry.Open)
			r12 := dir * math.Log(b12.Close/entry.Open)

			a.Add(side, r1, r4, r12)
			total.Add(side, r1, r4, r12)
			y := time.UnixMilli(ft).UTC().Year()
			yy := byYear[y]
			yy.Add(side, r1, r4, r12)
			byYear[y] = yy
			ss := bySide[side]
			ss.Add(side, r1, r4, r12)
			bySide[side] = ss
			_ = cw.Write([]string{
				sym, strconv.FormatInt(ft, 10), side,
				strconv.FormatFloat(ratio, 'f', -1, 64),
				strconv.FormatFloat(recentSum, 'f', -1, 64),
				strconv.FormatInt(entryTime, 10),
				strconv.FormatFloat(r1, 'f', -1, 64),
				strconv.FormatFloat(r4, 'f', -1, 64),
				strconv.FormatFloat(r12, 'f', -1, 64),
			})
		}
		bySym[sym] = a
		fmt.Printf("SYM %s n=%d long=%d short=%d r1=%.6f r4=%.6f r12=%.6f\n",
			sym, a.N, a.Long, a.Short, mean(a.R1, a.N), mean(a.R4, a.N), mean(a.R12, a.N))
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
		"positive_symbols":          fmt.Sprintf("%d/%d", positive, len(symbols)),
		"frequency_per_symbol_week": freq, "gate_pass": gate,
		"oos_evaluated": false, "strict_engine_run": false,
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
