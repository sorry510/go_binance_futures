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
  if not<=0{return}
  r:=t.NetPnL/not
  a.Net+=r
  if r>0 { a.GP+=r } else { a.GL-=r }
}
func (a Agg) PF() float64 { if a.GL<=0 { if a.GP>0{return 999};return 0 }; return a.GP/a.GL }

func main(){
  root:="strategy_templates/research/id121-mechanism-ablation/2026-09-30-audit"
  variants:=[]string{"base","no_funding","no_daily_regime","no_4h_strength","no_freshness","no_impulse","no_no_chase"}

  dbn,_:=config.String("database::dbname"); if dbn!="go_bn_test"{panic("expected go_bn_test, got "+dbn)}
  u,_:=config.String("database::username"); pw,_:=config.String("database::password"); h,_:=config.String("database::host"); pt,_:=config.String("database::port")
  _=orm.RegisterDriver("mysql",orm.DRMySQL)
  if e:=orm.RegisterDataBase("default","mysql",fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&collation=utf8mb4_unicode_ci",u,pw,h,pt,dbn));e!=nil{panic(e)}

  cfg:=backtest.RunConfig{InitialEquity:1000,PositionSizePct:1,Leverage:4,FeeRate:.0005,SlippageBps:5,StopLossPct:6,TakeProfitPct:8}
  end:=time.Date(2026,9,1,0,0,0,0,time.UTC).UnixMilli()
  mature:=time.Date(2024,8,15,0,0,0,0,time.UTC).UnixMilli()
  starts:=map[string]int64{
    "1000PEPEUSDT":time.Date(2025,5,5,0,0,0,0,time.UTC).UnixMilli(),
    "SUIUSDT":time.Date(2025,5,3,0,0,0,0,time.UTC).UnixMilli(),
    "ONDOUSDT":time.Date(2026,1,20,0,0,0,0,time.UTC).UnixMilli(),
  }
  syms:=[]string{"ADAUSDT","AVAXUSDT","BNBUSDT","BTCUSDT","DOGEUSDT","ETHUSDT","LTCUSDT","NEARUSDT","SOLUSDT","UNIUSDT","XRPUSDT","ZECUSDT","1000PEPEUSDT","SUIUSDT","ONDOUSDT"}
  repo:=historicalmarket.NewRepository(nil);ctx:=context.Background()

  for _,v:=range variants {
    raw,e:=os.ReadFile(root+"/strategies/"+v+".json");if e!=nil{panic(e)}
    var tpl Template;if e=json.Unmarshal(raw,&tpl);e!=nil{panic(e)}
    total:=Agg{}; bySym:=map[string]Agg{}; byYear:=map[int]Agg{}; weeks:=0.0
    fmt.Printf("VARIANT %s START\n",v)
    for i,sym:=range syms {
      st:=mature;if x,ok:=starts[sym];ok{st=x}
      weeks += float64(end-st)/float64(7*24*time.Hour/time.Millisecond)
      ds,e:=(backtest.DatasetBuilder{Repository:repo}).Build(ctx,backtest.DatasetRequest{
        Symbol:sym,ExecutionInterval:"1m",StartTime:st,EndTime:end,
        TechnologyJSON:string(tpl.Technology),StrategyJSON:string(tpl.Strategy),
      });if e!=nil{panic(e)}
      rr,e:=(backtest.Engine{}).Run(ctx,ds,backtest.StrategySnapshot{
        TemplateName:tpl.Name,TechnologyJSON:string(tpl.Technology),StrategyJSON:string(tpl.Strategy),Version:"id121-ablation-"+v,
      },cfg);if e!=nil{panic(e)}
      a:=Agg{}
      for _,t:=range rr.Trades {
        a.Add(t); total.Add(t)
        y:=time.UnixMilli(t.EntryTime).UTC().Year(); yy:=byYear[y]; yy.Add(t); byYear[y]=yy
      }
      bySym[sym]=a
      fmt.Printf("SYM %s %d/%d n=%d pf=%.6f net=%.9f\n",sym,i+1,len(syms),a.N,a.PF(),a.Net)
    }
    pos:=0;for _,a:=range bySym{if a.Net>0{pos++}}
    fmt.Printf("ALL %s n=%d long=%d short=%d pf=%.6f net=%.9f positive=%d/%d freq=%.6f\n",v,total.N,total.Long,total.Short,total.PF(),total.Net,pos,len(syms),float64(total.N)/weeks)
    ys:=make([]int,0,len(byYear));for y:=range byYear{ys=append(ys,y)};sort.Ints(ys)
    for _,y:=range ys {a:=byYear[y];fmt.Printf("YEAR %s %d n=%d pf=%.6f net=%.9f\n",v,y,a.N,a.PF(),a.Net)}
    fmt.Printf("VARIANT %s END\n",v)
  }
}
