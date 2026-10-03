package main

import (
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"sort"
	"strconv"
	"time"

	_ "go_binance_futures/bootstrap"

	"github.com/beego/beego/v2/core/config"
	_ "github.com/go-sql-driver/mysql"
)

type Bar struct {
	T int64
	O float64
	C float64
}
type Event struct {
	Symbol      string
	Time        int64
	Year        int
	Side        string
	PrevScore   int
	Score       int
	R1, R4, R12 float64
}
type Agg struct {
	N, Long, Short int
	S1, S4, S12    float64
	W12            int
}

func (a *Agg) Add(e Event) {
	a.N++
	a.S1 += e.R1
	a.S4 += e.R4
	a.S12 += e.R12
	if e.R12 > 0 {
		a.W12++
	}
	if e.Side == "LONG" {
		a.Long++
	} else {
		a.Short++
	}
}
func mean(x float64, n int) float64 {
	if n == 0 {
		return 0
	}
	return x / float64(n)
}
func score(b []Bar, end int) (int, bool) {
	if end < 23 {
		return 0, false
	}
	for j := end - 22; j <= end; j++ {
		if b[j].T-b[j-1].T != 3600000 {
			return 0, false
		}
	}
	hi := b[end-23].C
	lo := hi
	up, down := 0, 0
	for j := end - 22; j <= end; j++ {
		c := b[j].C
		if c > hi {
			up++
			hi = c
		}
		if c < lo {
			down++
			lo = c
		}
	}
	return up - down, true
}
func sm(a Agg) map[string]interface{} {
	return map[string]interface{}{
		"n": a.N, "long": a.Long, "short": a.Short,
		"mean_r1": mean(a.S1, a.N), "mean_r4": mean(a.S4, a.N), "mean_r12": mean(a.S12, a.N),
		"win_r12": mean(float64(a.W12), a.N),
	}
}
func main() {
	const root = "strategy_templates/research/record-arrival-imbalance/2026-10-02-early-gate"
	syms := []string{"SOLUSDT", "DOGEUSDT", "LTCUSDT", "AVAXUSDT", "UNIUSDT", "ZECUSDT"}
	start := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC).UnixMilli()
	end := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC).UnixMilli()

	u, _ := config.String("database::username")
	pw, _ := config.String("database::password")
	h, _ := config.String("database::host")
	pt, _ := config.String("database::port")
	dbn, _ := config.String("database::dbname")
	if dbn != "go_bn_test" {
		panic("refuse database " + dbn)
	}
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4", u, pw, h, pt, dbn)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	f, err := os.Create(root + "/results/events.csv")
	if err != nil {
		panic(err)
	}
	defer f.Close()
	cw := csv.NewWriter(f)
	defer cw.Flush()
	_ = cw.Write([]string{"symbol", "signal_time", "year", "side", "prev_score", "score", "entry_time", "r1", "r4", "r12"})

	all := Agg{}
	bySym := map[string]Agg{}
	byYear := map[int]Agg{}
	bySide := map[string]Agg{}
	events := 0
	for _, sym := range syms {
		q := "SELECT open_time,open_price,close_price FROM market_klines_1h WHERE market='futures_usdt' AND symbol=? AND open_time>=? AND open_time<? ORDER BY open_time"
		rows, e := db.Query(q, sym, start-48*3600000, end)
		if e != nil {
			panic(e)
		}
		var bars []Bar
		for rows.Next() {
			var b Bar
			if e := rows.Scan(&b.T, &b.O, &b.C); e != nil {
				panic(e)
			}
			bars = append(bars, b)
		}
		rows.Close()
		a := Agg{}
		for i := 24; i+12 < len(bars); i++ {
			if bars[i].T < start || bars[i].T >= end {
				continue
			}
			prev, ok1 := score(bars, i-1)
			now, ok2 := score(bars, i)
			if !ok1 || !ok2 {
				continue
			}
			side := 0.0
			sideName := ""
			if prev <= 0 && now > 0 {
				side = 1
				sideName = "LONG"
			}
			if prev >= 0 && now < 0 {
				side = -1
				sideName = "SHORT"
			}
			if side == 0 {
				continue
			}
			// require forward continuity through the 12h endpoint
			ok := true
			for j := i + 1; j <= i+12; j++ {
				if bars[j].T-bars[j-1].T != 3600000 {
					ok = false
					break
				}
			}
			if !ok {
				continue
			}
			y := time.UnixMilli(bars[i].T).UTC().Year()
			if time.UnixMilli(bars[i+12].T).UTC().Year() != y {
				continue
			}
			entry := bars[i+1].O
			if entry <= 0 {
				continue
			}
			r1 := side * math.Log(bars[i+1].C/entry)
			r4 := side * math.Log(bars[i+4].C/entry)
			r12 := side * math.Log(bars[i+12].C/entry)
			e := Event{sym, bars[i].T, y, sideName, prev, now, r1, r4, r12}
			a.Add(e)
			all.Add(e)
			yy := byYear[y]
			yy.Add(e)
			byYear[y] = yy
			ss := bySide[sideName]
			ss.Add(e)
			bySide[sideName] = ss
			events++
			_ = cw.Write([]string{sym, strconv.FormatInt(bars[i].T, 10), strconv.Itoa(y), sideName, strconv.Itoa(prev), strconv.Itoa(now), strconv.FormatInt(bars[i+1].T, 10), strconv.FormatFloat(r1, 'g', -1, 64), strconv.FormatFloat(r4, 'g', -1, 64), strconv.FormatFloat(r12, 'g', -1, 64)})
		}
		bySym[sym] = a
		fmt.Printf("SYM %s n=%d r1=%.6f r4=%.6f r12=%.6f\n", sym, a.N, mean(a.S1, a.N), mean(a.S4, a.N), mean(a.S12, a.N))
	}
	cw.Flush()
	if e := cw.Error(); e != nil {
		panic(e)
	}
	weeks := float64(end-start) / float64((7*24*time.Hour)/time.Millisecond) * float64(len(syms))
	freq := float64(all.N) / weeks
	pos := 0
	for _, s := range syms {
		if mean(bySym[s].S12, bySym[s].N) > 0 {
			pos++
		}
	}
	fmt.Printf("ALL n=%d r1=%.6f r4=%.6f r12=%.6f positive=%d/%d freq=%.6f\n", all.N, mean(all.S1, all.N), mean(all.S4, all.N), mean(all.S12, all.N), pos, len(syms), freq)
	years := []int{}
	for y := range byYear {
		years = append(years, y)
	}
	sort.Ints(years)
	for _, y := range years {
		a := byYear[y]
		fmt.Printf("YEAR %d n=%d r12=%.6f\n", y, a.N, mean(a.S12, a.N))
	}
	for _, side := range []string{"LONG", "SHORT"} {
		a := bySide[side]
		fmt.Printf("SIDE %s n=%d r12=%.6f\n", side, a.N, mean(a.S12, a.N))
	}
	gate := mean(all.S12, all.N) >= 0.002 && pos >= 4 && freq >= 0.30 && mean(byYear[2023].S12, byYear[2023].N) > 0 && mean(byYear[2024].S12, byYear[2024].N) > 0
	out := map[string]interface{}{"version": "v160", "status": "frozen_failed_early_gate", "gate_pass": gate, "events": events, "mean_r1": mean(all.S1, all.N), "mean_r4": mean(all.S4, all.N), "mean_r12": mean(all.S12, all.N), "positive_symbols": fmt.Sprintf("%d/%d", pos, len(syms)), "frequency_per_symbol_week": freq, "by_symbol": map[string]interface{}{}, "by_year": map[string]interface{}{}, "by_side": map[string]interface{}{}, "oos_evaluated": false, "strict_engine_run": false}
	if gate {
		out["status"] = "promote_translation"
	}
	for _, s := range syms {
		out["by_symbol"].(map[string]interface{})[s] = sm(bySym[s])
	}
	for _, y := range years {
		out["by_year"].(map[string]interface{})[strconv.Itoa(y)] = sm(byYear[y])
	}
	for _, s := range []string{"LONG", "SHORT"} {
		out["by_side"].(map[string]interface{})[s] = sm(bySide[s])
	}
	b, _ := json.MarshalIndent(out, "", "  ")
	if e := os.WriteFile(root+"/results/summary.json", b, 0644); e != nil {
		panic(e)
	}
}
