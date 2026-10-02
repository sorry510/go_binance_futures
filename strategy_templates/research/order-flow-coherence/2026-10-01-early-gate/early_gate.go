package main

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"runtime"
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
func coherence(net, gross float64) float64 {
	if gross <= 0 {
		return 0
	}
	return math.Abs(net) / gross
}
func main() {
	const root = "strategy_templates/research/order-flow-coherence/2026-10-01-early-gate"
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
	warm := start - int64(61*time.Minute/time.Millisecond)
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
	_ = cw.Write([]string{"symbol", "signal_time", "side", "coherence_prev", "coherence_now", "net_flow", "gross_flow", "entry_time", "r1", "r4", "r12"})

	total := Agg{}
	bySym := map[string]Agg{}
	byYear := map[int]Agg{}
	for _, sym := range symbols {
		fmt.Println("LOAD", sym)
		rows, err := repo.LoadReplayKlines(ctx, historicalmarket.MarketFuturesUSDT, sym, "1m", warm, end-1)
		if err != nil {
			panic(err)
		}
		if len(rows) < 800 {
			panic("insufficient rows for " + sym)
		}

		pn := make([]float64, len(rows)+1)
		pg := make([]float64, len(rows)+1)
		for i, b := range rows {
			f := 2*b.TakerBuyQuoteVolume - b.QuoteVolume
			pn[i+1] = pn[i] + f
			pg[i+1] = pg[i] + math.Abs(f)
		}
		a := Agg{}
		for i := 60; i+720 < len(rows); i++ {
			signal := rows[i]
			if signal.CloseTime < start || signal.CloseTime >= end {
				continue
			}
			// Current trailing 60 complete minutes: i-59..i.
			netNow := pn[i+1] - pn[i-59]
			grossNow := pg[i+1] - pg[i-59]
			// Previous trailing 60 complete minutes: i-60..i-1.
			netPrev := pn[i] - pn[i-60]
			grossPrev := pg[i] - pg[i-60]
			cNow := coherence(netNow, grossNow)
			cPrev := coherence(netPrev, grossPrev)
			if !(cPrev <= 0.5 && cNow > 0.5) || netNow == 0 {
				continue
			}

			entry := rows[i+1]
			// 12h endpoint must stay entirely inside discovery.
			if rows[i+720].CloseTime >= end || entry.Open <= 0 {
				continue
			}
			dir := 1.0
			side := "LONG"
			if netNow < 0 {
				dir = -1
				side = "SHORT"
			}
			r1 := dir * math.Log(rows[i+60].Close/entry.Open)
			r4 := dir * math.Log(rows[i+240].Close/entry.Open)
			r12 := dir * math.Log(rows[i+720].Close/entry.Open)
			a.Add(side, r1, r4, r12)
			total.Add(side, r1, r4, r12)
			y := time.UnixMilli(signal.CloseTime).UTC().Year()
			yy := byYear[y]
			yy.Add(side, r1, r4, r12)
			byYear[y] = yy
			_ = cw.Write([]string{
				sym, strconv.FormatInt(signal.CloseTime, 10), side,
				strconv.FormatFloat(cPrev, 'f', -1, 64), strconv.FormatFloat(cNow, 'f', -1, 64),
				strconv.FormatFloat(netNow, 'f', -1, 64), strconv.FormatFloat(grossNow, 'f', -1, 64),
				strconv.FormatInt(entry.OpenTime, 10),
				strconv.FormatFloat(r1, 'f', -1, 64), strconv.FormatFloat(r4, 'f', -1, 64), strconv.FormatFloat(r12, 'f', -1, 64),
			})
		}
		bySym[sym] = a
		fmt.Printf("SYM %s n=%d long=%d short=%d r1=%.6f r4=%.6f r12=%.6f\n",
			sym, a.N, a.Long, a.Short, mean(a.R1, a.N), mean(a.R4, a.N), mean(a.R12, a.N))
		rows = nil
		pn = nil
		pg = nil
		runtime.GC()
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
		"positive_symbols": fmt.Sprintf("%d/%d", positive, len(symbols)), "frequency_per_symbol_week": freq,
		"by_symbol": symJSON, "by_year": yearJSON, "gate_pass": gate,
		"oos_evaluated": false, "strict_engine_run": false,
	}
	b, _ := json.MarshalIndent(summary, "", "  ")
	if err := os.WriteFile(root+"/results/summary.json", b, 0644); err != nil {
		panic(err)
	}
}
