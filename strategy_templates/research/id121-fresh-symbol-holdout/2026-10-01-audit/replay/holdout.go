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
  Name string `json:"name"`
  Technology json.RawMessage `json:"technology"`
  Strategy json.RawMessage `json:"strategy"`
}
type Agg struct { N, Long, Short int; GP, GL, Net float64 }
func (a *Agg) Add(t backtest.Trade) {
  a.N++
  if strings.EqualFold(t.Side,"long"){a.Long++}else{a.Short++}
  notional:=t.EntryPrice*t.Quantity
  if notional<=0{return}
  r:=t.NetPnL/notional
  a.Net += r
  if r>0{a.GP+=r}else{a.GL-=r}
}
func (a Agg) PF() float64 {
  if a.GL<=0 { if a.GP>0{return 999}; return 0 }
  return a.GP/a.GL
}

func main(){
  root:="strategy_templates/research/id121-fresh-symbol-holdout/2026-10-01-audit"
  b,e:=os.ReadFile(root+"/strategy/id121-exact-exit.json"); if e!=nil{panic(e)}
  var tpl Template; if e=json.Unmarshal(b,&tpl);e!=nil{panic(e)}

  dbn,_:=config.String("database::dbname"); if dbn!="go_bn_test"{panic("expected go_bn_test, got "+dbn)}
  u,_:=config.String("database::username"); pw,_:=config.String("database::password")
  h,_:=config.String("database::host"); pt,_:=config.String("database::port")
  _=orm.RegisterDriver("mysql",orm.DRMySQL)
  if e:=orm.RegisterDataBase("default","mysql",fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&collation=utf8mb4_unicode_ci",u,pw,h,pt,dbn));e!=nil{panic(e)}

  starts:=map[string]int64{
    "ALGOUSDT":time.Date(2024,12,1,0,0,0,0,time.UTC).UnixMilli(),
    "INJUSDT":time.Date(2024,12,1,0,0,0,0,time.UTC).UnixMilli(),
    "LDOUSDT":time.Date(2024,12,1,0,0,0,0,time.UTC).UnixMilli(),
    "PENDLEUSDT":time.Date(2025,7,1,0,0,0,0,time.UTC).UnixMilli(),
    "PYTHUSDT":time.Date(2025,11,1,0,0,0,0,time.UTC).UnixMilli(),
  }
  syms:=[]string{"ALGOUSDT","INJUSDT","LDOUSDT","PENDLEUSDT","PYTHUSDT"}
  end:=time.Date(2026,9,1,0,0,0,0,time.UTC).UnixMilli()
  cfg:=backtest.RunConfig{InitialEquity:1000,PositionSizePct:1,Leverage:4,FeeRate:.0005,SlippageBps:5,StopLossPct:6,TakeProfitPct:8}
  repo:=historicalmarket.NewRepository(nil); ctx:=context.Background()

  f,e:=os.Create(root+"/results/trades.csv"); if e!=nil{panic(e)}; defer f.Close()
  w:=csv.NewWriter(f); defer w.Flush()
  _=w.Write([]string{"symbol","entry_time","year","side","exit_reason","norm_return"})

  total:=Agg{}; bySym:=map[string]Agg{}; byYear:=map[int]Agg{}; byReason:=map[string]int{}
  weeks:=0.0
  for i,sym:=range syms{
    st:=starts[sym]
    weeks += float64(end-st)/float64(7*24*time.Hour/time.Millisecond)
    fmt.Printf("BUILD %d/%d %s start=%s\n",i+1,len(syms),sym,time.UnixMilli(st).UTC().Format("2006-01-02"))
    ds,e:=(backtest.DatasetBuilder{Repository:repo}).Build(ctx,backtest.DatasetRequest{
      Symbol:sym,ExecutionInterval:"1m",StartTime:st,EndTime:end,
      TechnologyJSON:string(tpl.Technology),StrategyJSON:string(tpl.Strategy),
    }); if e!=nil{panic(e)}
    rr,e:=(backtest.Engine{}).Run(ctx,ds,backtest.StrategySnapshot{
      TemplateName:tpl.Name,TechnologyJSON:string(tpl.Technology),StrategyJSON:string(tpl.Strategy),Version:"id121-fresh-symbol-holdout",
    },cfg); if e!=nil{panic(e)}

    a:=Agg{}
    for _,t:=range rr.Trades{
      a.Add(t); total.Add(t)
      y:=time.UnixMilli(t.EntryTime).UTC().Year()
      yy:=byYear[y]; yy.Add(t); byYear[y]=yy
      byReason[t.ExitReason]++
      notional:=t.EntryPrice*t.Quantity; nr:=0.0; if notional>0{nr=t.NetPnL/notional}
      _=w.Write([]string{sym,strconv.FormatInt(t.EntryTime,10),strconv.Itoa(y),t.Side,t.ExitReason,strconv.FormatFloat(nr,'f',12,64)})
    }
    bySym[sym]=a
    fmt.Printf("SYM %s trades=%d long=%d short=%d normPF=%.6f normNet=%.6f\n",sym,a.N,a.Long,a.Short,a.PF(),a.Net)
  }
  w.Flush()

  pos:=0; for _,a:=range bySym{if a.Net>0{pos++}}
  fmt.Printf("ALL trades=%d long=%d short=%d normPF=%.6f normNet=%.6f positive=%d/%d freq=%.6f\n",
    total.N,total.Long,total.Short,total.PF(),total.Net,pos,len(syms),float64(total.N)/weeks)
  ys:=[]int{}; for y:=range byYear{ys=append(ys,y)}; sort.Ints(ys)
  for _,y:=range ys{a:=byYear[y]; fmt.Printf("YEAR %d trades=%d normPF=%.6f normNet=%.6f\n",y,a.N,a.PF(),a.Net)}
  ks:=[]string{}; for k:=range byReason{ks=append(ks,k)}; sort.Strings(ks)
  for _,k:=range ks{fmt.Printf("EXIT %s %d\n",k,byReason[k])}
}
