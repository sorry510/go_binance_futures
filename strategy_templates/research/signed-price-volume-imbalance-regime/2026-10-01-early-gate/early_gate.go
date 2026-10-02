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
func score(rows []historicalmarket.ReplayKline, i int) (float64, bool) {
	if i < 23 {
		return 0, false
	}
	var s float64
	for j := i - 23; j <= i; j++ {
		if rows[j].QuoteVolume <= 0 {
			return 0, false
		}
		s += 2*rows[j].TakerBuyQuoteVolume - rows[j].QuoteVolume
	}
	return s / 24, true
}

func main() {
	const root = "strategy_templates/research/signed-price-volume-imbalance-regime/2026-10-01-early-gate"
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

	syms := []string{"SOLUSDT", "DOGEUSDT", "LTCUSDT", "AVAXUSDT", "UNIUSDT", "ZECUSDT"}
	start := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC).UnixMilli()
	end := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC).UnixMilli()
	loadStart := start - int64(48*time.Hour/time.Millisecond)
	repo := historicalmarket.NewRepository(nil)
	ctx := context.Background()
	f, err := os.Create(root + "/results/events.csv")
	if err != nil {
		panic(err)
	}
	defer f.Close()
	cw := csv.NewWriter(f)
	defer cw.Flush()
	_ = cw.Write([]string{"symbol", "signal_time", "side", "score_prev", "score_now", "entry_time", "r1", "r4", "r12"})
	total := Agg{}
	bySym := map[string]Agg{}
	byYear := map[int]Agg{}
	bySide := map[string]Agg{}

	for _, sym := range syms {
		fmt.Println("LOAD", sym)
		rows, err := repo.LoadReplayKlines(ctx, historicalmarket.MarketFuturesUSDT, sym, "1h", loadStart, end-1)
		if err != nil {
			panic(err)
		}
		a := Agg{}
		for i := 24; i+12 < len(rows); i++ {
			if rows[i].CloseTime < start || rows[i].CloseTime >= end || rows[i+12].CloseTime >= end {
				continue
			}
			prev, ok1 := score(rows, i-1)
			now, ok2 := score(rows, i)
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
			y := time.UnixMilli(rows[i].CloseTime).UTC().Year()
			yy := byYear[y]
			yy.Add(r1, r4, r12)
			byYear[y] = yy
			ss := bySide[side]
			ss.Add(r1, r4, r12)
			bySide[side] = ss
			_ = cw.Write([]string{sym, strconv.FormatInt(rows[i].CloseTime, 10), side, strconv.FormatFloat(prev, 'g', -1, 64), strconv.FormatFloat(now, 'g', -1, 64), strconv.FormatInt(rows[i+1].OpenTime, 10), strconv.FormatFloat(r1, 'g', -1, 64), strconv.FormatFloat(r4, 'g', -1, 64), strconv.FormatFloat(r12, 'g', -1, 64)})
		}
		bySym[sym] = a
		fmt.Printf("SYM %s n=%d r1=%.6f r4=%.6f r12=%.6f\n", sym, a.N, avg(a.R1, a.N), avg(a.R4, a.N), avg(a.R12, a.N))
	}
	cw.Flush()
	if err := cw.Error(); err != nil {
		panic(err)
	}
	weeks := float64(end-start) / float64((7*24*time.Hour)/time.Millisecond) * float64(len(syms))
	freq := float64(total.N) / weeks
	pos := 0
	for _, a := range bySym {
		if avg(a.R12, a.N) > 0 {
			pos++
		}
	}
	fmt.Printf("ALL n=%d r1=%.6f r4=%.6f r12=%.6f positive=%d/%d freq=%.6f\n", total.N, avg(total.R1, total.N), avg(total.R4, total.N), avg(total.R12, total.N), pos, len(syms), freq)
	ys := make([]int, 0, len(byYear))
	for y := range byYear {
		ys = append(ys, y)
	}
	sort.Ints(ys)
	for _, y := range ys {
		a := byYear[y]
		fmt.Printf("YEAR %d n=%d r1=%.6f r4=%.6f r12=%.6f\n", y, a.N, avg(a.R1, a.N), avg(a.R4, a.N), avg(a.R12, a.N))
	}
	for _, s := range []string{"LONG", "SHORT"} {
		a := bySide[s]
		fmt.Printf("SIDE %s n=%d r1=%.6f r4=%.6f r12=%.6f\n", s, a.N, avg(a.R1, a.N), avg(a.R4, a.N), avg(a.R12, a.N))
	}
	summary := map[string]interface{}{"events": total.N, "mean_r1": avg(total.R1, total.N), "mean_r4": avg(total.R4, total.N), "mean_r12": avg(total.R12, total.N), "positive_symbols": fmt.Sprintf("%d/%d", pos, len(syms)), "frequency_per_symbol_week": freq, "oos_evaluated": false, "strict_engine_run": false}
	b, _ := json.MarshalIndent(summary, "", "  ")
	_ = os.WriteFile(root+"/results/summary.json", b, 0644)
}
