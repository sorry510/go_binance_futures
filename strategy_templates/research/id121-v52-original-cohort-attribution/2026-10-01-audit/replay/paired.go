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
type Agg struct {
  N, Long, Short int
  RawGP, RawGL, RawNet float64
  NormGP, NormGL, NormNet float64
}
func (a *Agg) Add(t backtest.Trade) {
  a.N++
  if strings.EqualFold(t.Side,"long") { a.Long++ } else { a.Short++ }
  r:=t.NetPnL
  a.RawNet += r
  if r>0 { a.RawGP += r } else { a.RawGL -= r }
  notional:=t.EntryPrice*t.Quantity
  if notional>0 {
    nr:=r/notional
    a.NormNet += nr
    if nr>0 { a.NormGP += nr } else { a.NormGL -= nr }
  }
}
func pf(g,l float64) float64 { if l<=0 { if g>0{return 999}; return 0}; return g/l }

func main(){
  root:="strategy_templates/research/id121-v52-original-cohort-attribution/2026-10-01-audit"
  readTpl:=func(path string) Template {
    b,e:=os.ReadFile(path); if e!=nil{panic(e)}
    var t Template; if e=json.Unmarshal(b,&t);e!=nil{panic(e)}; return t
  }
  id:=readTpl(root+"/strategy/id121.json")
  v52:=readTpl(root+"/strategy/v52.json")

  dbn,_:=config.String("database::dbname"); if dbn!="go_bn_test"{panic("expected go_bn_test, got "+dbn)}
  u,_:=config.String("database::username");pw,_:=config.String("database::password");h,_:=config.String("database::host");pt,_:=config.String("database::port")
  _=orm.RegisterDriver("mysql",orm.DRMySQL)
  if e:=orm.RegisterDataBase("default","mysql",fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&collation=utf8mb4_unicode_ci",u,pw,h,pt,dbn));e!=nil{panic(e)}

  f,e:=os.Create(root+"/results/trades.csv"); if e!=nil{panic(e)}; defer f.Close()
  w:=csv.NewWriter(f); defer w.Flush()
  w.Write([]string{"strategy","symbol","entry_time","year","side","exit_reason","raw_net_pnl","norm_return"})

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
  totals:=map[string]*Agg{"ID121":{}, "V52":{}}
  bySym:=map[string]map[string]*Agg{"ID121":{}, "V52":{}}
  byYear:=map[string]map[int]*Agg{"ID121":{}, "V52":{}}
  weeksSum:=0.0
  emit:=func(label,sym string,tr []backtest.Trade){
    if bySym[label][sym]==nil { bySym[label][sym]=&Agg{} }
    for _,t:=range tr {
      totals[label].Add(t); bySym[label][sym].Add(t)
      y:=time.UnixMilli(t.EntryTime).UTC().Year()
      if byYear[label][y]==nil { byYear[label][y]=&Agg{} }
      byYear[label][y].Add(t)
      notional:=t.EntryPrice*t.Quantity; nr:=0.0;if notional>0{nr=t.NetPnL/notional}
      w.Write([]string{label,sym,strconv.FormatInt(t.EntryTime,10),strconv.Itoa(y),t.Side,t.ExitReason,strconv.FormatFloat(t.NetPnL,'f',8,64),strconv.FormatFloat(nr,'f',12,64)})
    }
  }
  for i,sym:=range syms {
    st:=mature;if x,ok:=starts[sym];ok{st=x}
    weeksSum += float64(end-st)/float64(7*24*time.Hour/time.Millisecond)
    fmt.Printf("BUILD %d/%d %s start=%s\n",i+1,len(syms),sym,time.UnixMilli(st).UTC().Format("2006-01-02"))
    ds,e:=(backtest.DatasetBuilder{Repository:repo}).Build(ctx,backtest.DatasetRequest{Symbol:sym,ExecutionInterval:"1m",StartTime:st,EndTime:end,TechnologyJSON:string(id.Technology),StrategyJSON:string(id.Strategy)});if e!=nil{panic(e)}
    r1,e:=(backtest.Engine{}).Run(ctx,ds,backtest.StrategySnapshot{TemplateName:id.Name,TechnologyJSON:string(id.Technology),StrategyJSON:string(id.Strategy),Version:"id121-exact-exit"},cfg);if e!=nil{panic(e)}
    r2,e:=(backtest.Engine{}).Run(ctx,ds,backtest.StrategySnapshot{TemplateName:v52.Name,TechnologyJSON:string(v52.Technology),StrategyJSON:string(v52.Strategy),Version:"v52-strict-original-cohort"},cfg);if e!=nil{panic(e)}
    emit("ID121",sym,r1.Trades);emit("V52",sym,r2.Trades)
    a:=bySym["ID121"][sym];b:=bySym["V52"][sym]
    fmt.Printf("SYM %s ID121 n=%d rawPF=%.6f normPF=%.6f V52 n=%d rawPF=%.6f normPF=%.6f\n",sym,a.N,pf(a.RawGP,a.RawGL),pf(a.NormGP,a.NormGL),b.N,pf(b.RawGP,b.RawGL),pf(b.NormGP,b.NormGL))
  }
  w.Flush()
  for _,label:=range []string{"ID121","V52"} {
    a:=totals[label];pos:=0
    for _,x:=range bySym[label]{if x.NormNet>0{pos++}}
    fmt.Printf("ALL %s n=%d rawPF=%.6f rawNet=%.6f normPF=%.6f normNet=%.9f positive=%d/%d freq=%.6f\n",label,a.N,pf(a.RawGP,a.RawGL),a.RawNet,pf(a.NormGP,a.NormGL),a.NormNet,pos,len(syms),float64(a.N)/weeksSum)
    ys:=make([]int,0,len(byYear[label]));for y:=range byYear[label]{ys=append(ys,y)};sort.Ints(ys)
    for _,y:=range ys {x:=byYear[label][y];fmt.Printf("YEAR %s %d n=%d rawPF=%.6f normPF=%.6f normNet=%.9f\n",label,y,x.N,pf(x.RawGP,x.RawGL),pf(x.NormGP,x.NormGL),x.NormNet)}
  }
}
