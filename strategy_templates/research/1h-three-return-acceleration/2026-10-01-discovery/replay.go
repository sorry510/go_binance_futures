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

type Template struct { Name string `json:"name"`; Technology json.RawMessage `json:"technology"`; Strategy json.RawMessage `json:"strategy"` }
type Agg struct { N, Long, Short int; GP, GL, Net float64 }
func (a *Agg) Add(t backtest.Trade) {
  a.N++
  if strings.EqualFold(t.Side,"long") { a.Long++ } else { a.Short++ }
  not:=t.EntryPrice*t.Quantity; if not<=0{return}; r:=t.NetPnL/not; a.Net+=r; if r>0{a.GP+=r}else{a.GL-=r}
}
func (a Agg) PF() float64 { if a.GL<=0 { if a.GP>0{return 999}; return 0 }; return a.GP/a.GL }
func main(){
  raw,err:=os.ReadFile("strategy_templates/research/1h-three-return-acceleration/2026-10-01-discovery/strategy.json"); if err!=nil{panic(err)}
  var tpl Template; if err=json.Unmarshal(raw,&tpl);err!=nil{panic(err)}
  dbn,_:=config.String("database::dbname"); if dbn!="go_bn_test"{panic("expected go_bn_test, got "+dbn)}
  u,_:=config.String("database::username"); pw,_:=config.String("database::password"); h,_:=config.String("database::host"); pt,_:=config.String("database::port")
  _=orm.RegisterDriver("mysql",orm.DRMySQL)
  if e:=orm.RegisterDataBase("default","mysql",fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&collation=utf8mb4_unicode_ci",u,pw,h,pt,dbn));e!=nil{panic(e)}
  cfg:=backtest.RunConfig{InitialEquity:1000,PositionSizePct:1,Leverage:4,FeeRate:.0005,SlippageBps:5,StopLossPct:6,TakeProfitPct:8}
  start:=time.Date(2023,1,1,0,0,0,0,time.UTC).UnixMilli(); end:=time.Date(2025,1,1,0,0,0,0,time.UTC).UnixMilli()
  syms:=[]string{"SOLUSDT","DOGEUSDT","LTCUSDT","AVAXUSDT","UNIUSDT","ZECUSDT"}
  repo:=historicalmarket.NewRepository(nil); ctx:=context.Background()
  total:=Agg{}; bySym:=map[string]Agg{}; byYear:=map[int]Agg{}; byReason:=map[string]int{}
  for i,sym:=range syms{
    fmt.Printf("BUILD %d/%d %s\n",i+1,len(syms),sym)
    ds,e:=(backtest.DatasetBuilder{Repository:repo}).Build(ctx,backtest.DatasetRequest{Symbol:sym,ExecutionInterval:"1m",StartTime:start,EndTime:end,TechnologyJSON:string(tpl.Technology),StrategyJSON:string(tpl.Strategy)}); if e!=nil{panic(e)}
    rr,e:=(backtest.Engine{}).Run(ctx,ds,backtest.StrategySnapshot{TemplateName:tpl.Name,TechnologyJSON:string(tpl.Technology),StrategyJSON:string(tpl.Strategy),Version:"v129-discovery"},cfg); if e!=nil{panic(e)}
    a:=Agg{}
    for _,t:=range rr.Trades{a.Add(t);total.Add(t);y:=time.UnixMilli(t.EntryTime).UTC().Year(); yy:=byYear[y];yy.Add(t);byYear[y]=yy;byReason[t.ExitReason]++}
    bySym[sym]=a
    fmt.Printf("SYM %s trades=%d long=%d short=%d normPF=%.6f normNet=%.6f\n",sym,a.N,a.Long,a.Short,a.PF(),a.Net)
  }
  weeks:=float64(end-start)/float64(7*24*time.Hour/time.Millisecond)*float64(len(syms))
  pos:=0;for _,a:=range bySym{if a.Net>0{pos++}}
  fmt.Printf("ALL trades=%d long=%d short=%d normPF=%.6f normNet=%.6f positive=%d/%d freq=%.6f\n",total.N,total.Long,total.Short,total.PF(),total.Net,pos,len(syms),float64(total.N)/weeks)
  years:=make([]int,0,len(byYear));for y:=range byYear{years=append(years,y)};sort.Ints(years)
  for _,y:=range years{a:=byYear[y];fmt.Printf("YEAR %d trades=%d normPF=%.6f normNet=%.6f\n",y,a.N,a.PF(),a.Net)}
  keys:=make([]string,0,len(byReason));for k:=range byReason{keys=append(keys,k)};sort.Strings(keys);for _,k:=range keys{fmt.Printf("EXIT %s %d\n",k,byReason[k])}
}
