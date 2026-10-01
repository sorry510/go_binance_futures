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
type Agg struct{ N,Long,Short int; GP,GL,Net float64 }
func (a *Agg) Add(t backtest.Trade){
  a.N++; if strings.EqualFold(t.Side,"long"){a.Long++}else{a.Short++}
  not:=t.EntryPrice*t.Quantity; if not<=0{return}
  r:=t.NetPnL/not; a.Net+=r; if r>0{a.GP+=r}else if r<0{a.GL-=r}
}
func (a Agg) PF()float64{if a.GL<=0{if a.GP>0{return 999};return 0};return a.GP/a.GL}

func main(){
  root:="strategy_templates/research/id121-exact-exit-forward/2026-09-01_2026-09-12-1559z"
  b,e:=os.ReadFile(root+"/strategy/id121-exact-exit.json");if e!=nil{panic(e)}
  var tpl Template;if e=json.Unmarshal(b,&tpl);e!=nil{panic(e)}
  dbn,_:=config.String("database::dbname");if dbn!="go_bn_test"{panic("expected go_bn_test, got "+dbn)}
  u,_:=config.String("database::username");pw,_:=config.String("database::password");h,_:=config.String("database::host");pt,_:=config.String("database::port")
  _=orm.RegisterDriver("mysql",orm.DRMySQL)
  if e:=orm.RegisterDataBase("default","mysql",fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&collation=utf8mb4_unicode_ci",u,pw,h,pt,dbn));e!=nil{panic(e)}
  cfg:=backtest.RunConfig{InitialEquity:1000,PositionSizePct:1,Leverage:4,FeeRate:.0005,SlippageBps:5,StopLossPct:6,TakeProfitPct:8}
  fwdStart:=time.Date(2026,9,1,0,0,0,0,time.UTC).UnixMilli()
  end:=time.Date(2026,9,12,16,0,0,0,time.UTC).UnixMilli()
  mature:=time.Date(2024,8,15,0,0,0,0,time.UTC).UnixMilli()
  starts:=map[string]int64{
    "1000PEPEUSDT":time.Date(2025,5,5,0,0,0,0,time.UTC).UnixMilli(),
    "SUIUSDT":time.Date(2025,5,3,0,0,0,0,time.UTC).UnixMilli(),
    "ONDOUSDT":time.Date(2026,1,20,0,0,0,0,time.UTC).UnixMilli(),
  }
  syms:=[]string{"ADAUSDT","AVAXUSDT","BNBUSDT","BTCUSDT","DOGEUSDT","ETHUSDT","LTCUSDT","NEARUSDT","SOLUSDT","UNIUSDT","XRPUSDT","ZECUSDT","1000PEPEUSDT","SUIUSDT","ONDOUSDT"}
  out,e:=os.Create(root+"/results/trades.csv");if e!=nil{panic(e)};defer out.Close()
  w:=csv.NewWriter(out);defer w.Flush()
  _=w.Write([]string{"symbol","entry_time","entry_utc","side","exit_reason","raw_net_pnl","norm_return"})
  repo:=historicalmarket.NewRepository(nil);ctx:=context.Background()
  total:=Agg{};closed:=Agg{};bySym:=map[string]*Agg{};bySide:=map[string]*Agg{};byReason:=map[string]int{}
  for i,sym:=range syms{
    st:=mature;if x,ok:=starts[sym];ok{st=x}
    fmt.Printf("BUILD %d/%d %s from=%s\n",i+1,len(syms),sym,time.UnixMilli(st).UTC().Format("2006-01-02"))
    ds,e:=(backtest.DatasetBuilder{Repository:repo}).Build(ctx,backtest.DatasetRequest{Symbol:sym,ExecutionInterval:"1m",StartTime:st,EndTime:end,TechnologyJSON:string(tpl.Technology),StrategyJSON:string(tpl.Strategy)});if e!=nil{panic(e)}
    rr,e:=(backtest.Engine{}).Run(ctx,ds,backtest.StrategySnapshot{TemplateName:tpl.Name,TechnologyJSON:string(tpl.Technology),StrategyJSON:string(tpl.Strategy),Version:"id121-exact-forward-202609"},cfg);if e!=nil{panic(e)}
    if bySym[sym]==nil{bySym[sym]=&Agg{}}
    for _,t:=range rr.Trades{
      if t.EntryTime<fwdStart{continue}
      total.Add(t);bySym[sym].Add(t)
      side:=strings.ToUpper(t.Side);if bySide[side]==nil{bySide[side]=&Agg{}};bySide[side].Add(t)
      byReason[t.ExitReason]++
      if t.ExitReason!="end_of_data"{closed.Add(t)}
      not:=t.EntryPrice*t.Quantity;nr:=0.0;if not>0{nr=t.NetPnL/not}
      _=w.Write([]string{sym,strconv.FormatInt(t.EntryTime,10),time.UnixMilli(t.EntryTime).UTC().Format(time.RFC3339),side,t.ExitReason,strconv.FormatFloat(t.NetPnL,'f',8,64),strconv.FormatFloat(nr,'f',12,64)})
    }
    a:=bySym[sym];fmt.Printf("FWD %s n=%d PF=%.6f net=%.9f\n",sym,a.N,a.PF(),a.Net)
  }
  w.Flush()
  pos:=0;for _,a:=range bySym{if a.Net>0{pos++}}
  weeks:=float64(end-fwdStart)/float64(7*24*time.Hour/time.Millisecond)
  fmt.Printf("ALL n=%d PF=%.6f net=%.9f positive=%d/%d total_per_week=%.6f per_symbol_week=%.6f\n",total.N,total.PF(),total.Net,pos,len(syms),float64(total.N)/weeks,float64(total.N)/(weeks*float64(len(syms))))
  fmt.Printf("CLOSED_ONLY n=%d PF=%.6f net=%.9f\n",closed.N,closed.PF(),closed.Net)
  sides:=[]string{"LONG","SHORT"};for _,s:=range sides{a:=bySide[s];if a!=nil{fmt.Printf("SIDE %s n=%d PF=%.6f net=%.9f\n",s,a.N,a.PF(),a.Net)}}
  keys:=make([]string,0,len(byReason));for k:=range byReason{keys=append(keys,k)};sort.Strings(keys);for _,k:=range keys{fmt.Printf("EXIT %s %d\n",k,byReason[k])}
}
