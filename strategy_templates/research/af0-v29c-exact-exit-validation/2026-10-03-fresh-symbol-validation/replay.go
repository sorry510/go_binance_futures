package main

import (
  "context"
  "encoding/json"
  "fmt"
  "os"
  "sort"
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
  Name string `json:"name"`
  Technology json.RawMessage `json:"technology"`
  Strategy json.RawMessage `json:"strategy"`
}
type Agg struct { N, Long, Short int; GP, GL, Net float64 }
func (a *Agg) Add(t backtest.Trade) {
  a.N++
  if strings.EqualFold(t.Side,"long") { a.Long++ } else { a.Short++ }
  not:=t.EntryPrice*t.Quantity
  if not<=0 { return }
  r:=t.NetPnL/not
  a.Net+=r
  if r>0 { a.GP+=r } else { a.GL-=r }
}
func (a Agg) PF() float64 {
  if a.GL<=0 { if a.GP>0 { return 999 }; return 0 }
  return a.GP/a.GL
}
type Row struct {
  Trades int `json:"trades"`
  Long int `json:"long"`
  Short int `json:"short"`
  PF float64 `json:"pf"`
  Net float64 `json:"net"`
}
type Summary struct {
  Status string `json:"status"`
  GatePass bool `json:"gate_pass"`
  CanonicalExitRule string `json:"canonical_exit_rule"`
  Symbols []string `json:"symbols"`
  Overall Row `json:"overall"`
  PositiveSymbols string `json:"positive_symbols"`
  FrequencyPerSymbolWeek float64 `json:"frequency_per_symbol_week"`
  BySymbol map[string]Row `json:"by_symbol"`
  ByYear map[string]Row `json:"by_year"`
  BySide map[string]Row `json:"by_side"`
  Exits map[string]int `json:"exits"`
  DBWrite bool `json:"db_write"`
}
func row(a Agg) Row { return Row{Trades:a.N,Long:a.Long,Short:a.Short,PF:a.PF(),Net:a.Net} }

func main() {
  raw,err:=os.ReadFile("strategy_templates/research/af0-v29c-exact-exit-validation/2026-10-03-fresh-symbol-validation/strategy/af0-exact-exit.json")
  if err!=nil { panic(err) }
  var tpl Template
  if err=json.Unmarshal(raw,&tpl);err!=nil { panic(err) }

  dbn,_:=config.String("database::dbname")
  if dbn!="go_bn_test" { panic("expected go_bn_test, got "+dbn) }
  u,_:=config.String("database::username")
  pw,_:=config.String("database::password")
  h,_:=config.String("database::host")
  pt,_:=config.String("database::port")
  _=orm.RegisterDriver("mysql",orm.DRMySQL)
  if e:=orm.RegisterDataBase("default","mysql",fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&collation=utf8mb4_unicode_ci",u,pw,h,pt,dbn));e!=nil{panic(e)}

  cfg:=backtest.RunConfig{
    InitialEquity:1000, PositionSizePct:1, Leverage:4,
    FeeRate:.0005, SlippageBps:5, StopLossPct:6, TakeProfitPct:8,
  }
  start:=time.Date(2023,1,1,0,0,0,0,time.UTC).UnixMilli()
  end:=time.Date(2026,9,1,0,0,0,0,time.UTC).UnixMilli()
  syms:=[]string{"DOGEUSDT","LTCUSDT","AVAXUSDT","UNIUSDT","ZECUSDT","ADAUSDT"}

  // Source=nil is deliberate: read existing DB/cache only; never REST-import gaps.
  repo:=historicalmarket.NewRepository(nil)
  ctx:=context.Background()
  total:=Agg{}
  bySym:=map[string]Agg{}
  byYear:=map[int]Agg{}
  bySide:=map[string]Agg{}
  byReason:=map[string]int{}

  for i,sym:=range syms {
    fmt.Printf("BUILD %d/%d %s\n",i+1,len(syms),sym)
    ds,e:=(backtest.DatasetBuilder{Repository:repo}).Build(ctx,backtest.DatasetRequest{
      Symbol:sym,ExecutionInterval:"1m",StartTime:start,EndTime:end,
      TechnologyJSON:string(tpl.Technology),StrategyJSON:string(tpl.Strategy),
    })
    if e!=nil { panic(e) }
    rr,e:=(backtest.Engine{}).Run(ctx,ds,backtest.StrategySnapshot{
      TemplateName:tpl.Name,TechnologyJSON:string(tpl.Technology),
      StrategyJSON:string(tpl.Strategy),Version:"af0-v29c-exact-fresh-v1",
    },cfg)
    if e!=nil { panic(e) }

    a:=Agg{}
    for _,t:=range rr.Trades {
      a.Add(t); total.Add(t)
      y:=time.UnixMilli(t.EntryTime).UTC().Year()
      yy:=byYear[y]; yy.Add(t); byYear[y]=yy
      ss:=bySide[strings.ToUpper(t.Side)]; ss.Add(t); bySide[strings.ToUpper(t.Side)]=ss
      byReason[t.ExitReason]++
    }
    bySym[sym]=a
    fmt.Printf("SYM %s trades=%d long=%d short=%d normPF=%.6f normNet=%.6f\n",sym,a.N,a.Long,a.Short,a.PF(),a.Net)
  }

  weeks:=float64(end-start)/float64(7*24*time.Hour/time.Millisecond)*float64(len(syms))
  freq:=float64(total.N)/weeks
  pos:=0
  for _,a:=range bySym { if a.Net>0 { pos++ } }

  materialYearCollapse:=false
  for _,a:=range byYear {
    if a.N>=30 && a.PF()<0.90 { materialYearCollapse=true }
  }
  gate:=total.PF()>=1.15 && pos>=4 && freq>=0.30 && !materialYearCollapse
  status:="frozen_failed_fresh_validation"
  if gate { status="passed_fresh_validation_candidate" }

  bySymJSON:=map[string]Row{}
  for _,s:=range syms { bySymJSON[s]=row(bySym[s]) }
  years:=make([]int,0,len(byYear))
  for y:=range byYear { years=append(years,y) }
  sort.Ints(years)
  byYearJSON:=map[string]Row{}
  for _,y:=range years { byYearJSON[fmt.Sprintf("%d",y)]=row(byYear[y]) }
  bySideJSON:=map[string]Row{}
  for _,s:=range []string{"LONG","SHORT"} { bySideJSON[s]=row(bySide[s]) }

  out:=Summary{
    Status:status,GatePass:gate,CanonicalExitRule:"ROI >= 8 || ROI <= -6",
    Symbols:syms,Overall:row(total),PositiveSymbols:fmt.Sprintf("%d/%d",pos,len(syms)),
    FrequencyPerSymbolWeek:freq,BySymbol:bySymJSON,ByYear:byYearJSON,
    BySide:bySideJSON,Exits:byReason,DBWrite:false,
  }
  b,_:=json.MarshalIndent(out,"","  ")
  if err:=os.WriteFile("strategy_templates/research/af0-v29c-exact-exit-validation/2026-10-03-fresh-symbol-validation/results/summary.json",b,0644);err!=nil{panic(err)}

  fmt.Printf("ALL trades=%d long=%d short=%d normPF=%.6f normNet=%.6f positive=%d/%d freq=%.6f gate=%v\n",total.N,total.Long,total.Short,total.PF(),total.Net,pos,len(syms),freq,gate)
  for _,y:=range years { a:=byYear[y]; fmt.Printf("YEAR %d trades=%d normPF=%.6f normNet=%.6f\n",y,a.N,a.PF(),a.Net) }
  for _,s:=range []string{"LONG","SHORT"} { a:=bySide[s]; fmt.Printf("SIDE %s trades=%d normPF=%.6f normNet=%.6f\n",s,a.N,a.PF(),a.Net) }
  keys:=make([]string,0,len(byReason)); for k:=range byReason { keys=append(keys,k) }; sort.Strings(keys)
  for _,k:=range keys { fmt.Printf("EXIT %s %d\n",k,byReason[k]) }
}
