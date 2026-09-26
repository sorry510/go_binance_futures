package main
import(
 "context";"database/sql";"encoding/csv";"encoding/json";"fmt";"os";"strconv";"time"
 _ "go_binance_futures/bootstrap"
 "go_binance_futures/service/backtest"
 "go_binance_futures/service/historicalmarket"
 "github.com/beego/beego/v2/client/orm"
 "github.com/beego/beego/v2/core/config"
 _ "github.com/go-sql-driver/mysql"
)
type P struct{Name string `json:"name"`;Technology any `json:"technology"`;Strategy any `json:"strategy"`}
func emit(w *csv.Writer,label,sym string,tr []backtest.Trade){for _,t:=range tr{notional:=t.EntryPrice*t.Quantity;nr:=0.0;if notional>0{nr=t.NetPnL/notional};tm:=time.UnixMilli(t.EntryTime).UTC();w.Write([]string{label,sym,strconv.FormatInt(t.EntryTime,10),tm.Format("2006-01"),t.Side,strconv.FormatFloat(nr,'f',12,64)})}}
func main(){u,_:=config.String("database::username");pw,_:=config.String("database::password");h,_:=config.String("database::host");pt,_:=config.String("database::port");dsn:=fmt.Sprintf("%s:%s@tcp(%s:%s)/go_bn_test?charset=utf8mb4&collation=utf8mb4_unicode_ci",u,pw,h,pt);_=orm.RegisterDriver("mysql",orm.DRMySQL);if e:=orm.RegisterDataBase("default","mysql",dsn);e!=nil{panic(e)}
 db,e:=sql.Open("mysql",dsn);if e!=nil{panic(e)};defer db.Close();var bn,bt,bs string;if e=db.QueryRow("SELECT name,technology,strategy FROM strategy_templates WHERE id=121").Scan(&bn,&bt,&bs);e!=nil{panic(e)}
 raw,e:=os.ReadFile("temp_strategy/v54/01-precompression-funding-bypass-long.json");if e!=nil{panic(e)};var v P;if e=json.Unmarshal(raw,&v);e!=nil{panic(e)};vt,_:=json.Marshal(v.Technology);vs,_:=json.Marshal(v.Strategy)
 f,e:=os.Create("/tmp/paired_121_v54.csv");if e!=nil{panic(e)};defer f.Close();w:=csv.NewWriter(f);defer w.Flush();w.Write([]string{"strategy","symbol","entry_time","month","side","norm_return"})
 cfg:=backtest.RunConfig{InitialEquity:1000,PositionSizePct:1,Leverage:4,FeeRate:.0005,SlippageBps:5,StopLossPct:6,TakeProfitPct:8};end:=time.Date(2026,9,1,0,0,0,0,time.UTC).UnixMilli();old:=time.Date(2024,8,15,0,0,0,0,time.UTC).UnixMilli()
 starts:=map[string]int64{"1000PEPEUSDT":time.Date(2025,5,5,0,0,0,0,time.UTC).UnixMilli(),"SUIUSDT":time.Date(2025,5,3,0,0,0,0,time.UTC).UnixMilli(),"ONDOUSDT":time.Date(2026,1,20,0,0,0,0,time.UTC).UnixMilli()}
 syms:=[]string{"ADAUSDT","AVAXUSDT","BNBUSDT","BTCUSDT","DOGEUSDT","ETHUSDT","LTCUSDT","NEARUSDT","SOLUSDT","UNIUSDT","XRPUSDT","ZECUSDT","1000PEPEUSDT","SUIUSDT","ONDOUSDT"}
 repo:=historicalmarket.NewRepository(nil);ctx:=context.Background()
 for _,sym:=range syms{st:=old;if x,ok:=starts[sym];ok{st=x};fmt.Println("START",sym);ds,e:=(backtest.DatasetBuilder{Repository:repo}).Build(ctx,backtest.DatasetRequest{Symbol:sym,ExecutionInterval:"1m",StartTime:st,EndTime:end,TechnologyJSON:string(vt),StrategyJSON:string(vs)});if e!=nil{panic(e)}
  b,e:=(backtest.Engine{}).Run(ctx,ds,backtest.StrategySnapshot{TemplateID:121,TemplateName:bn,TechnologyJSON:bt,StrategyJSON:bs,Version:"id121-paired"},cfg);if e!=nil{panic(e)}
  c,e:=(backtest.Engine{}).Run(ctx,ds,backtest.StrategySnapshot{TemplateName:v.Name,TechnologyJSON:string(vt),StrategyJSON:string(vs),Version:"v54-paired"},cfg);if e!=nil{panic(e)}
  emit(w,"ID121",sym,b.Trades);emit(w,"V54",sym,c.Trades);fmt.Printf("DONE %s ID121=%d V54=%d\n",sym,len(b.Trades),len(c.Trades))
 }
 w.Flush();fmt.Println("EXPORTED /tmp/paired_121_v54.csv")
}