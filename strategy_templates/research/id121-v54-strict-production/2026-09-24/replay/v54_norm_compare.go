package main
import("context";"database/sql";"encoding/json";"fmt";"math";"os";"sort";"time"
 _ "go_binance_futures/bootstrap"
 "go_binance_futures/service/backtest"
 "go_binance_futures/service/historicalmarket"
 "github.com/beego/beego/v2/client/orm"
 "github.com/beego/beego/v2/core/config"
 _ "github.com/go-sql-driver/mysql")
type P struct{Name string `json:"name"`;Technology any `json:"technology"`;Strategy any `json:"strategy"`}
type S struct{N int;GP,GL,Net float64}
func(s *S)add(x float64){s.N++;s.Net+=x;if x>=0{s.GP+=x}else{s.GL-=x}}
func(s S)pf()float64{if s.GL==0{if s.GP>0{return 999};return 0};return s.GP/s.GL}
func add(a *S,b S){a.N+=b.N;a.GP+=b.GP;a.GL+=b.GL;a.Net+=b.Net}
func tstat(tr []backtest.Trade)(raw,norm S){for _,t:=range tr{raw.add(t.NetPnL);den:=math.Abs(t.EntryPrice*t.Quantity);if den>0{norm.add(t.NetPnL/den)}};return}
func main(){u,_:=config.String("database::username");pw,_:=config.String("database::password");h,_:=config.String("database::host");pt,_:=config.String("database::port");dsn:=fmt.Sprintf("%s:%s@tcp(%s:%s)/go_bn_test?charset=utf8mb4&collation=utf8mb4_unicode_ci",u,pw,h,pt);_=orm.RegisterDriver("mysql",orm.DRMySQL);if e:=orm.RegisterDataBase("default","mysql",dsn);e!=nil{panic(e)};db,e:=sql.Open("mysql",dsn);if e!=nil{panic(e)};defer db.Close()
 var bn,bt,bs string;if e=db.QueryRow("SELECT name,technology,strategy FROM strategy_templates WHERE id=121").Scan(&bn,&bt,&bs);e!=nil{panic(e)}
 raw,e:=os.ReadFile("temp_strategy/v54/01-precompression-funding-bypass-long.json");if e!=nil{panic(e)};var v54 P;if e=json.Unmarshal(raw,&v54);e!=nil{panic(e)};ct,_:=json.Marshal(v54.Technology);cs,_:=json.Marshal(v54.Strategy)
 cfg:=backtest.RunConfig{InitialEquity:1000,PositionSizePct:1,Leverage:4,FeeRate:.0005,SlippageBps:5,StopLossPct:6,TakeProfitPct:8};end:=time.Date(2026,9,1,0,0,0,0,time.UTC).UnixMilli();oldStart:=time.Date(2024,8,15,0,0,0,0,time.UTC).UnixMilli()
 starts:=map[string]int64{"1000PEPEUSDT":time.Date(2025,5,5,0,0,0,0,time.UTC).UnixMilli(),"SUIUSDT":time.Date(2025,5,3,0,0,0,0,time.UTC).UnixMilli(),"ONDOUSDT":time.Date(2026,1,20,0,0,0,0,time.UTC).UnixMilli()}
 syms:=[]string{"ADAUSDT","AVAXUSDT","BNBUSDT","BTCUSDT","DOGEUSDT","ETHUSDT","LTCUSDT","NEARUSDT","SOLUSDT","UNIUSDT","XRPUSDT","ZECUSDT","1000PEPEUSDT","SUIUSDT","ONDOUSDT"}
 type Agg struct{Raw,Norm S;Year map[int]*S;SymPF []float64;Pos int};ag:=map[string]*Agg{"ID121":{Year:map[int]*S{}},"V54":{Year:map[int]*S{}}}
 for _,sym:=range syms{st:=oldStart;if x,ok:=starts[sym];ok{st=x};fmt.Println("START",sym);ds,e:=(backtest.DatasetBuilder{Repository:historicalmarket.NewRepository(nil)}).Build(context.Background(),backtest.DatasetRequest{Symbol:sym,ExecutionInterval:"1m",StartTime:st,EndTime:end,TechnologyJSON:string(ct),StrategyJSON:string(cs)});if e!=nil{panic(e)}
  runs:=[]struct{Name string;Snap backtest.StrategySnapshot}{{"ID121",backtest.StrategySnapshot{TemplateID:121,TemplateName:bn,TechnologyJSON:bt,StrategyJSON:bs,Version:"norm"}},{"V54",backtest.StrategySnapshot{TemplateName:v54.Name,TechnologyJSON:string(ct),StrategyJSON:string(cs),Version:"norm"}}}
  for _,rr:=range runs{res,e:=(backtest.Engine{}).Run(context.Background(),ds,rr.Snap,cfg);if e!=nil{panic(e)};rawS,normS:=tstat(res.Trades);a:=ag[rr.Name];add(&a.Raw,rawS);add(&a.Norm,normS);a.SymPF=append(a.SymPF,normS.pf());if normS.Net>0{a.Pos++};for _,t:=range res.Trades{den:=math.Abs(t.EntryPrice*t.Quantity);if den<=0{continue};y:=time.UnixMilli(t.EntryTime).UTC().Year();if a.Year[y]==nil{a.Year[y]=&S{}};a.Year[y].add(t.NetPnL/den)};fmt.Printf("%s %s n=%d rawPF=%.3f normPF=%.3f normNet=%.4f\n",sym,rr.Name,rawS.N,rawS.pf(),normS.pf(),normS.Net)}
 }
 for _,name:=range []string{"ID121","V54"}{a:=ag[name];sort.Float64s(a.SymPF);med:=a.SymPF[len(a.SymPF)/2];fmt.Printf("TOTAL %s n=%d rawPF=%.6f rawNet=%.2f normPF=%.6f normNet=%.6f pos=%d/%d medianNormSymPF=%.6f\n",name,a.Raw.N,a.Raw.pf(),a.Raw.Net,a.Norm.pf(),a.Norm.Net,a.Pos,len(a.SymPF),med);for y:=2024;y<=2026;y++{s:=a.Year[y];fmt.Printf("YEAR %s %d n=%d normPF=%.6f normNet=%.6f\n",name,y,s.N,s.pf(),s.Net)}}
}