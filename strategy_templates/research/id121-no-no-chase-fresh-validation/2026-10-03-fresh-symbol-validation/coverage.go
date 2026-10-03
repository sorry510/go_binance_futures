package main

import (
  "context"
  "encoding/json"
  "fmt"
  "os"
  "time"

  "github.com/beego/beego/v2/client/orm"
  "github.com/beego/beego/v2/core/config"
  _ "github.com/go-sql-driver/mysql"
  _ "go_binance_futures/bootstrap"
  "go_binance_futures/service/backtest"
  "go_binance_futures/service/historicalmarket"
)

type Template struct {
  Name string `json:"name"`
  Technology json.RawMessage `json:"technology"`
  Strategy json.RawMessage `json:"strategy"`
}

type CoverageRow struct {
  Symbol string `json:"symbol"`
  DatasetStart int64 `json:"dataset_start"`
  DatasetEnd int64 `json:"dataset_end"`
  WarmupStart int64 `json:"warmup_start"`
  Intervals []string `json:"intervals"`
  ExecutionBars int `json:"execution_bars"`
  Complete bool `json:"complete"`
  Error string `json:"error,omitempty"`
}
type CoverageSummary struct {
  Symbols []CoverageRow `json:"symbols"`
  AllPass bool `json:"all_pass"`
  ReturnsRead bool `json:"returns_read"`
  DBWrite bool `json:"db_write"`
}

func main(){
  raw,err:=os.ReadFile("strategy_templates/research/id121-no-no-chase-fresh-validation/2026-10-03-fresh-symbol-validation/strategy.json")
  if err!=nil{panic(err)}
  var tpl Template
  if err=json.Unmarshal(raw,&tpl);err!=nil{panic(err)}

  dbn,_:=config.String("database::dbname")
  if dbn!="go_bn_test"{panic("expected go_bn_test, got "+dbn)}
  u,_:=config.String("database::username"); pw,_:=config.String("database::password")
  h,_:=config.String("database::host"); pt,_:=config.String("database::port")
  _=orm.RegisterDriver("mysql",orm.DRMySQL)
  if e:=orm.RegisterDataBase("default","mysql",fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&collation=utf8mb4_unicode_ci",u,pw,h,pt,dbn));e!=nil{panic(e)}

  start:=time.Date(2023,1,1,0,0,0,0,time.UTC).UnixMilli()
  end:=time.Date(2026,9,1,0,0,0,0,time.UTC).UnixMilli()
  syms:=[]string{"LINKUSDT","BCHUSDT","ETCUSDT","TRXUSDT","XLMUSDT","AAVEUSDT"}
  repo:=historicalmarket.NewRepository(nil)
  ctx:=context.Background()
  out:=CoverageSummary{ReturnsRead:false,DBWrite:false,AllPass:true}
  for i,sym:=range syms{
    fmt.Printf("COVERAGE %d/%d %s\n",i+1,len(syms),sym)
    ds,e:=(backtest.DatasetBuilder{Repository:repo}).Build(ctx,backtest.DatasetRequest{
      Symbol:sym,ExecutionInterval:"1m",StartTime:start,EndTime:end,
      TechnologyJSON:string(tpl.Technology),StrategyJSON:string(tpl.Strategy),
    })
    row:=CoverageRow{Symbol:sym}
    if e!=nil{
      row.Error=e.Error();row.Complete=false;out.AllPass=false
      out.Symbols=append(out.Symbols,row)
      fmt.Printf("BLOCK %s %v\n",sym,e)
      continue
    }
    row.DatasetStart=ds.StartTime;row.DatasetEnd=ds.EndTime;row.WarmupStart=ds.WarmupStartTime;row.Intervals=ds.Intervals
    row.ExecutionBars=len(ds.Bars[backtest.BarSeriesKey(sym,"1m")])
    row.Complete=ds.StartTime<=start && ds.EndTime>=end && row.ExecutionBars>0
    if !row.Complete{out.AllPass=false}
    out.Symbols=append(out.Symbols,row)
    fmt.Printf("OK %s start=%d end=%d warmup=%d intervals=%v bars1m=%d complete=%v\n",
      sym,row.DatasetStart,row.DatasetEnd,row.WarmupStart,row.Intervals,row.ExecutionBars,row.Complete)
  }
  b,_:=json.MarshalIndent(out,"","  ")
  if err=os.WriteFile("strategy_templates/research/id121-no-no-chase-fresh-validation/2026-10-03-fresh-symbol-validation/results/coverage.json",b,0644);err!=nil{panic(err)}
  fmt.Printf("ALL_PASS %v RETURNS_READ false\n",out.AllPass)
}
