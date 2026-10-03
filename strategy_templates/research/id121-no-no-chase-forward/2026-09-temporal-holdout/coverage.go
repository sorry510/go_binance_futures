package main

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"time"

	"github.com/beego/beego/v2/client/orm"
	"github.com/beego/beego/v2/core/config"
	_ "github.com/go-sql-driver/mysql"
	_ "go_binance_futures/bootstrap"
)

type Row struct {
	Symbol          string `orm:"column(symbol)" json:"symbol"`
	RowCount        int    `orm:"column(row_count)" json:"row_count"`
	RowMin          int64  `orm:"column(row_min)" json:"row_min"`
	RowMax          int64  `orm:"column(row_max)" json:"row_max"`
	ChunkCount      int    `orm:"column(chunk_count)" json:"chunk_count"`
	ChunkPoints     int    `orm:"column(chunk_points)" json:"chunk_points"`
	ChunkStart      int64  `orm:"column(chunk_start)" json:"chunk_start"`
	ChunkEnd        int64  `orm:"column(chunk_end)" json:"chunk_end"`
	FundingCount    int    `orm:"column(funding_count)" json:"funding_count"`
	FundingMin      int64  `orm:"column(funding_min)" json:"funding_min"`
	FundingMax      int64  `orm:"column(funding_max)" json:"funding_max"`
	Complete1m      bool   `json:"complete_1m"`
	CompleteFunding bool   `json:"complete_funding"`
}

func main() {
	dbn, _ := config.String("database::dbname")
	if dbn != "go_bn_test" {
		panic("expected go_bn_test, got " + dbn)
	}
	u, _ := config.String("database::username")
	pw, _ := config.String("database::password")
	h, _ := config.String("database::host")
	pt, _ := config.String("database::port")
	_ = orm.RegisterDriver("mysql", orm.DRMySQL)
	if e := orm.RegisterDataBase("default", "mysql", fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&collation=utf8mb4_unicode_ci", u, pw, h, pt, dbn)); e != nil {
		panic(e)
	}
	o := orm.NewOrm()
	syms := []string{"BTCUSDT", "ETHUSDT", "BNBUSDT", "XRPUSDT", "SOLUSDT", "DOGEUSDT", "LTCUSDT", "AVAXUSDT", "UNIUSDT", "ZECUSDT", "ADAUSDT", "NEARUSDT", "1000PEPEUSDT", "SUIUSDT", "ONDOUSDT"}
	start := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC).UnixMilli()
	endOpen := time.Date(2026, 9, 30, 23, 59, 0, 0, time.UTC).UnixMilli()
	endExclusive := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC).UnixMilli()
	expected := 30 * 24 * 60
	monthStart := start
	out := make([]Row, 0, len(syms))
	all := true
	for _, s := range syms {
		r := Row{Symbol: s}
		_ = o.Raw("SELECT COUNT(*) row_count,COALESCE(MIN(open_time),0) row_min,COALESCE(MAX(open_time),0) row_max FROM market_klines_1m WHERE market=? AND symbol=? AND open_time>=? AND open_time<?", "futures_usdt", s, start, endExclusive).QueryRow(&r.RowCount, &r.RowMin, &r.RowMax)
		_ = o.Raw("SELECT COUNT(*) chunk_count,COALESCE(SUM(point_count),0) chunk_points,COALESCE(MIN(start_time),0) chunk_start,COALESCE(MAX(end_time),0) chunk_end FROM market_klines_1m_chunks WHERE market=? AND symbol=? AND month_start=?", "futures_usdt", s, monthStart).QueryRow(&r.ChunkCount, &r.ChunkPoints, &r.ChunkStart, &r.ChunkEnd)
		_ = o.Raw("SELECT COUNT(*) funding_count,COALESCE(MIN(funding_time),0) funding_min,COALESCE(MAX(funding_time),0) funding_max FROM market_funding_rates WHERE market=? AND symbol=? AND funding_time>=? AND funding_time<?", "futures_usdt", s, start, endExclusive).QueryRow(&r.FundingCount, &r.FundingMin, &r.FundingMax)
		r.Complete1m = (r.RowCount == expected && r.RowMin == start && r.RowMax == endOpen) || (r.ChunkCount == 1 && r.ChunkPoints == expected && r.ChunkStart == start && r.ChunkEnd >= endOpen)
		// Standard 8h funding gives roughly 90 rows; require coverage reaches the last Sep settlement.
		lastFundingNeed := time.Date(2026, 9, 30, 16, 0, 0, 0, time.UTC).UnixMilli()
		r.CompleteFunding = r.FundingCount >= 89 && r.FundingMin <= time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC).UnixMilli() && r.FundingMax >= lastFundingNeed
		if !r.Complete1m || !r.CompleteFunding {
			all = false
		}
		out = append(out, r)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Symbol < out[j].Symbol })
	summary := map[string]interface{}{"start": start, "end_exclusive": endExclusive, "expected_1m_points": expected, "all_15_complete": all, "symbols": out, "returns_read": false, "db_write": false}
	b, _ := json.MarshalIndent(summary, "", "  ")
	_ = os.WriteFile("strategy_templates/research/id121-no-no-chase-forward/2026-09-temporal-holdout/results/coverage.json", b, 0644)
	fmt.Println(string(b))
}
