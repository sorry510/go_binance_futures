package main

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/beego/beego/v2/client/orm"
	"github.com/beego/beego/v2/core/config"
	_ "github.com/go-sql-driver/mysql"
	_ "go_binance_futures/bootstrap"
	"go_binance_futures/service/backtest"
	"go_binance_futures/service/historicalmarket"
)

type Template struct {
	Name       string          `json:"name"`
	Technology json.RawMessage `json:"technology"`
	Strategy   json.RawMessage `json:"strategy"`
}

type Agg struct {
	N, Long, Short int
	GP, GL, Net    float64
}

func (a *Agg) Add(t backtest.Trade) {
	a.N++
	if strings.EqualFold(t.Side, "long") {
		a.Long++
	} else {
		a.Short++
	}
	notional := t.EntryPrice * t.Quantity
	if notional <= 0 {
		return
	}
	r := t.NetPnL / notional
	a.Net += r
	if r > 0 {
		a.GP += r
	} else {
		a.GL -= r
	}
}
func (a Agg) PF() float64 {
	if a.GL <= 0 {
		if a.GP > 0 {
			return 999
		}
		return 0
	}
	return a.GP / a.GL
}
func aggMap(a Agg) map[string]interface{} {
	return map[string]interface{}{"trades": a.N, "long": a.Long, "short": a.Short, "pf": a.PF(), "net": a.Net}
}

func main() {
	const root = "strategy_templates/research/id121-regime-liquidity-sweep-reclaim/2026-10-02-discovery"
	raw, err := os.ReadFile(root + "/strategy.json")
	if err != nil {
		panic(err)
	}
	var tpl Template
	if err = json.Unmarshal(raw, &tpl); err != nil {
		panic(err)
	}

	dbn, _ := config.String("database::dbname")
	if dbn != "go_bn_test" {
		panic("expected go_bn_test, got " + dbn)
	}
	u, _ := config.String("database::username")
	pw, _ := config.String("database::password")
	h, _ := config.String("database::host")
	pt, _ := config.String("database::port")
	_ = orm.RegisterDriver("mysql", orm.DRMySQL)
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&collation=utf8mb4_unicode_ci", u, pw, h, pt, dbn)
	if e := orm.RegisterDataBase("default", "mysql", dsn); e != nil {
		panic(e)
	}

	out, err := os.Create(root + "/results/trades.csv")
	if err != nil {
		panic(err)
	}
	defer out.Close()
	cw := csv.NewWriter(out)
	defer cw.Flush()
	_ = cw.Write([]string{"symbol", "side", "entry_time", "exit_time", "entry_price", "exit_price", "quantity", "net_pnl", "normalized_return", "fees", "funding_pnl", "exit_reason"})

	cfg := backtest.RunConfig{InitialEquity: 1000, PositionSizePct: 1, Leverage: 4, FeeRate: .0005, SlippageBps: 5, StopLossPct: 6, TakeProfitPct: 8}
	start := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC).UnixMilli()
	end := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC).UnixMilli()
	syms := []string{"SOLUSDT", "DOGEUSDT", "LTCUSDT", "AVAXUSDT", "UNIUSDT", "ZECUSDT"}
	repo := historicalmarket.NewRepository(nil)
	ctx := context.Background()

	total := Agg{}
	bySym := map[string]Agg{}
	byYear := map[int]Agg{}
	byReason := map[string]int{}
	for i, sym := range syms {
		fmt.Printf("BUILD %d/%d %s\n", i+1, len(syms), sym)
		ds, e := (backtest.DatasetBuilder{Repository: repo}).Build(ctx, backtest.DatasetRequest{
			Symbol: sym, ExecutionInterval: "1m", StartTime: start, EndTime: end,
			TechnologyJSON: string(tpl.Technology), StrategyJSON: string(tpl.Strategy),
		})
		if e != nil {
			panic(e)
		}
		rr, e := (backtest.Engine{}).Run(ctx, ds, backtest.StrategySnapshot{
			TemplateName: tpl.Name, TechnologyJSON: string(tpl.Technology), StrategyJSON: string(tpl.Strategy), Version: "v165-discovery",
		}, cfg)
		if e != nil {
			panic(e)
		}
		a := Agg{}
		for _, t := range rr.Trades {
			a.Add(t)
			total.Add(t)
			y := time.UnixMilli(t.EntryTime).UTC().Year()
			yy := byYear[y]
			yy.Add(t)
			byYear[y] = yy
			byReason[t.ExitReason]++
			notional := t.EntryPrice * t.Quantity
			norm := 0.0
			if notional > 0 {
				norm = t.NetPnL / notional
			}
			_ = cw.Write([]string{
				sym, t.Side, strconv.FormatInt(t.EntryTime, 10), strconv.FormatInt(t.ExitTime, 10),
				strconv.FormatFloat(t.EntryPrice, 'f', -1, 64), strconv.FormatFloat(t.ExitPrice, 'f', -1, 64),
				strconv.FormatFloat(t.Quantity, 'f', -1, 64), strconv.FormatFloat(t.NetPnL, 'f', -1, 64),
				strconv.FormatFloat(norm, 'f', -1, 64), strconv.FormatFloat(t.Fees, 'f', -1, 64),
				strconv.FormatFloat(t.FundingPnL, 'f', -1, 64), t.ExitReason,
			})
		}
		bySym[sym] = a
		fmt.Printf("SYM %s trades=%d long=%d short=%d normPF=%.6f normNet=%.6f\n", sym, a.N, a.Long, a.Short, a.PF(), a.Net)
	}
	cw.Flush()
	if err = cw.Error(); err != nil {
		panic(err)
	}
	weeks := float64(end-start) / float64((7*24*time.Hour)/time.Millisecond) * float64(len(syms))
	pos := 0
	for _, a := range bySym {
		if a.Net > 0 {
			pos++
		}
	}
	freq := float64(total.N) / weeks
	fmt.Printf("ALL trades=%d long=%d short=%d normPF=%.6f normNet=%.6f positive=%d/%d freq=%.6f\n",
		total.N, total.Long, total.Short, total.PF(), total.Net, pos, len(syms), freq)

	years := make([]int, 0, len(byYear))
	for y := range byYear {
		years = append(years, y)
	}
	sort.Ints(years)
	for _, y := range years {
		a := byYear[y]
		fmt.Printf("YEAR %d trades=%d normPF=%.6f normNet=%.6f\n", y, a.N, a.PF(), a.Net)
	}
	keys := make([]string, 0, len(byReason))
	for k := range byReason {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		fmt.Printf("EXIT %s %d\n", k, byReason[k])
	}

	bySymOut := map[string]interface{}{}
	for _, s := range syms {
		bySymOut[s] = aggMap(bySym[s])
	}
	byYearOut := map[string]interface{}{}
	for _, y := range years {
		byYearOut[strconv.Itoa(y)] = aggMap(byYear[y])
	}
	summary := map[string]interface{}{
		"version": "v165", "status": "completed_discovery", "canonical_exit_rule": "ROI >= 8 || ROI <= -6",
		"trades": total.N, "long": total.Long, "short": total.Short, "normalized_pf": total.PF(), "normalized_net": total.Net,
		"positive_symbols": fmt.Sprintf("%d/%d", pos, len(syms)), "frequency_per_symbol_week": freq,
		"by_symbol": bySymOut, "by_year": byYearOut, "exit_reasons": byReason,
		"oos1_evaluated": false, "oos2_evaluated": false,
	}
	gate := total.PF() >= 1.15 && pos >= 4 && freq >= 0.30
	if a, ok := byYear[2023]; ok {
		gate = gate && a.PF() > 1
	}
	if a, ok := byYear[2024]; ok {
		gate = gate && a.PF() > 1
	}
	summary["gate_pass"] = gate
	if gate {
		summary["status"] = "promote_oos1"
	} else {
		summary["status"] = "frozen_failed_discovery"
	}
	b, _ := json.MarshalIndent(summary, "", "  ")
	if err := os.WriteFile(root+"/results/summary.json", b, 0644); err != nil {
		panic(err)
	}
}
