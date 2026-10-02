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
	Long, Short int
	R1, R4, R12 float64
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

func mean(sum float64, n int) float64 {
	if n == 0 {
		return 0
	}
	return sum / float64(n)
}

func sign(x float64) int {
	if x > 0 {
		return 1
	}
	if x < 0 {
		return -1
	}
	return 0
}

func regularGap(a, b int64) bool {
	d := b - a
	return d >= int64(7*time.Hour+30*time.Minute)/int64(time.Millisecond) &&
		d <= int64(8*time.Hour+30*time.Minute)/int64(time.Millisecond)
}
func firstStreakEvent(f []historicalmarket.FundingRate, i int) (int, bool) {
	if i < 2 {
		return 0, false
	}
	s := sign(f[i].FundingRate)
	if s == 0 || sign(f[i-1].FundingRate) != s || sign(f[i-2].FundingRate) != s {
		return 0, false
	}
	if !regularGap(f[i-2].FundingTime, f[i-1].FundingTime) ||
		!regularGap(f[i-1].FundingTime, f[i].FundingTime) {
		return 0, false
	}
	// Only fire when the regular-cadence same-sign streak first reaches length 3.
	if i >= 3 && sign(f[i-3].FundingRate) == s &&
		regularGap(f[i-3].FundingTime, f[i-2].FundingTime) {
		return 0, false
	}
	return s, true
}

func main() {
	const root = "strategy_templates/research/funding-sign-persistence-reversal/2026-10-01-early-gate"
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
	warm := start - int64(48*time.Hour/time.Millisecond)
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
	_ = cw.Write([]string{"symbol", "funding_time", "side", "funding_1", "funding_2", "funding_3", "entry_time", "r1", "r4", "r12"})

	total := Agg{}
	bySym := map[string]Agg{}
	byYear := map[int]Agg{}

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
		for i := 2; i < len(funding); i++ {
			s, ok := firstStreakEvent(funding, i)
			if !ok {
				continue
			}
			ft := funding[i].FundingTime
			if ft < start || ft >= end {
				continue
			}
			entryTime := ((ft / 3600000) + 1) * 3600000
			// Require the entire 12h diagnostic endpoint to remain inside discovery.
			if entryTime+12*3600000 >= end {
				continue
			}
			entryBar, ok0 := byOpen[entryTime]
			b1, ok1 := byOpen[entryTime]
			b4, ok4 := byOpen[entryTime+3*3600000]
			b12, ok12 := byOpen[entryTime+11*3600000]
			if !ok0 || !ok1 || !ok4 || !ok12 || entryBar.Open <= 0 {
				continue
			}
			dir := float64(-s) // positive funding -> SHORT; negative -> LONG
			side := "SHORT"
			if dir > 0 {
				side = "LONG"
			}
			r1 := dir * math.Log(b1.Close/entryBar.Open)
			r4 := dir * math.Log(b4.Close/entryBar.Open)
			r12 := dir * math.Log(b12.Close/entryBar.Open)
			a.Add(side, r1, r4, r12)
			total.Add(side, r1, r4, r12)
			y := time.UnixMilli(ft).UTC().Year()
			yy := byYear[y]
			yy.Add(side, r1, r4, r12)
			byYear[y] = yy
			_ = cw.Write([]string{
				sym, strconv.FormatInt(ft, 10), side,
				strconv.FormatFloat(funding[i-2].FundingRate, 'f', -1, 64),
				strconv.FormatFloat(funding[i-1].FundingRate, 'f', -1, 64),
				strconv.FormatFloat(funding[i].FundingRate, 'f', -1, 64),
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
	yearJSON := map[string]interface{}{}
	for _, y := range years {
		a := byYear[y]
		fmt.Printf("YEAR %d n=%d long=%d short=%d r1=%.6f r4=%.6f r12=%.6f\n",
			y, a.N, a.Long, a.Short, mean(a.R1, a.N), mean(a.R4, a.N), mean(a.R12, a.N))
		yearJSON[strconv.Itoa(y)] = map[string]interface{}{
			"events": a.N, "long": a.Long, "short": a.Short,
			"mean_r1": mean(a.R1, a.N), "mean_r4": mean(a.R4, a.N), "mean_r12": mean(a.R12, a.N),
		}
	}
	symJSON := map[string]interface{}{}
	for _, sym := range symbols {
		a := bySym[sym]
		symJSON[sym] = map[string]interface{}{
			"events": a.N, "long": a.Long, "short": a.Short,
			"mean_r1": mean(a.R1, a.N), "mean_r4": mean(a.R4, a.N), "mean_r12": mean(a.R12, a.N),
		}
	}
	gate := mean(total.R12, total.N) >= 0.002 && positive >= 4 && freq >= 0.30
	for _, y := range []int{2023, 2024} {
		a, ok := byYear[y]
		if !ok || mean(a.R12, a.N) <= 0 {
			gate = false
		}
	}
	summary := map[string]interface{}{
		"status": func() string {
			if gate {
				return "promoted_early_gate"
			}
			return "frozen_failed_early_gate"
		}(),
		"events": total.N, "long": total.Long, "short": total.Short,
		"mean_r1": mean(total.R1, total.N), "mean_r4": mean(total.R4, total.N), "mean_r12": mean(total.R12, total.N),
		"positive_symbols":          fmt.Sprintf("%d/%d", positive, len(symbols)),
		"frequency_per_symbol_week": freq, "by_symbol": symJSON, "by_year": yearJSON,
		"gate_pass": gate, "oos_evaluated": false, "strict_engine_run": false,
	}
	b, _ := json.MarshalIndent(summary, "", "  ")
	if err := os.WriteFile(root+"/results/summary.json", b, 0644); err != nil {
		panic(err)
	}
}
