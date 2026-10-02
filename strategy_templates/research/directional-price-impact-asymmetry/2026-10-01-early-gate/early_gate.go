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

func (a *Agg) Add(r1, r4, r12 float64) {
	a.N++
	a.R1 += r1
	a.R4 += r4
	a.R12 += r12
}
func score(rows []historicalmarket.ReplayKline, end int) (float64, bool) {
	if end < 23 {
		return 0, false
	}
	var upR, dnR, upV, dnV float64
	for j := end - 23; j <= end; j++ {
		b := rows[j]
		if b.Open <= 0 || b.Close <= 0 || b.QuoteVolume <= 0 {
			continue
		}
		r := math.Log(b.Close / b.Open)
		if r > 0 {
			upR += r
			upV += b.QuoteVolume
		}
		if r < 0 {
			dnR += -r
			dnV += b.QuoteVolume
		}
	}
	if upR <= 0 || dnR <= 0 || upV <= 0 || dnV <= 0 {
		return 0, false
	}
	up := upR / upV
	dn := dnR / dnV
	if up <= 0 || dn <= 0 {
		return 0, false
	}
	return math.Log(up / dn), true
}

func mean(sum float64, n int) float64 {
	if n == 0 {
		return 0
	}
	return sum / float64(n)
}
func main() {
	const root = "strategy_templates/research/directional-price-impact-asymmetry/2026-10-01-early-gate"
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
	loadStart := start - int64(36*time.Hour/time.Millisecond)
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

	for _, sym := range symbols {
		fmt.Println("LOAD", sym)
		rows, err := repo.LoadReplayKlines(ctx, historicalmarket.MarketFuturesUSDT, sym, "1h", loadStart, end-1)
		if err != nil {
			panic(err)
		}
		a := Agg{}
		for i := 24; i+12 < len(rows); i++ {
			signal := rows[i]
			if signal.CloseTime < start || signal.CloseTime >= end {
				continue
			}
			if rows[i+12].CloseTime >= end {
				continue
			}

			prev, ok1 := score(rows, i-1)
			now, ok2 := score(rows, i)
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

			entry := rows[i+1].Open
			if entry <= 0 {
				continue
			}
			r1 := side * math.Log(rows[i+1].Close/entry)
			r4 := side * math.Log(rows[i+4].Close/entry)
			r12 := side * math.Log(rows[i+12].Close/entry)
			a.Add(r1, r4, r12)
			total.Add(r1, r4, r12)
			y := time.UnixMilli(signal.CloseTime).UTC().Year()
			yy := byYear[y]
			yy.Add(r1, r4, r12)
			byYear[y] = yy

			_ = cw.Write([]string{
				sym, strconv.FormatInt(signal.CloseTime, 10), sideName,
				strconv.FormatFloat(prev, 'f', -1, 64), strconv.FormatFloat(now, 'f', -1, 64),
				strconv.FormatInt(rows[i+1].OpenTime, 10),
				strconv.FormatFloat(r1, 'f', -1, 64), strconv.FormatFloat(r4, 'f', -1, 64),
				strconv.FormatFloat(r12, 'f', -1, 64),
			})
		}
		bySym[sym] = a
		fmt.Printf("SYM %s n=%d r1=%.6f r4=%.6f r12=%.6f\n", sym, a.N, mean(a.R1, a.N), mean(a.R4, a.N), mean(a.R12, a.N))
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
	fmt.Printf("ALL n=%d r1=%.6f r4=%.6f r12=%.6f positive=%d/%d freq=%.6f\n",
		total.N, mean(total.R1, total.N), mean(total.R4, total.N), mean(total.R12, total.N), positive, len(symbols), freq)

	years := make([]int, 0, len(byYear))
	for y := range byYear {
		years = append(years, y)
	}
	sort.Ints(years)
	for _, y := range years {
		a := byYear[y]
		fmt.Printf("YEAR %d n=%d r1=%.6f r4=%.6f r12=%.6f\n", y, a.N, mean(a.R1, a.N), mean(a.R4, a.N), mean(a.R12, a.N))
	}

	summary := map[string]interface{}{
		"events": total.N, "mean_r1": mean(total.R1, total.N), "mean_r4": mean(total.R4, total.N),
		"mean_r12": mean(total.R12, total.N), "positive_symbols": fmt.Sprintf("%d/%d", positive, len(symbols)),
		"frequency_per_symbol_week": freq,
	}
	b, _ := json.MarshalIndent(summary, "", "  ")
	_ = os.WriteFile(root+"/results/summary.json", b, 0644)
}
