package main
import("database/sql";"fmt";"time"
 _ "go_binance_futures/bootstrap";"github.com/beego/beego/v2/core/config";_ "github.com/go-sql-driver/mysql")
type T struct{Sym string;E,X int64;P float64}
type S struct{N int;GP,GL,Net float64}
func(s *S)add(p float64){s.N++;s.Net+=p;if p>=0{s.GP+=p}else{s.GL-=p}}
func(s S)pf()float64{if s.GL==0{if s.GP>0{return 999};return 0};return s.GP/s.GL}
func main(){u,_:=config.String("database::username");p,_:=config.String("database::password");h,_:=config.String("database::host");pt,_:=config.String("database::port");db,e:=sql.Open("mysql",fmt.Sprintf("%s:%s@tcp(%s:%s)/go_bn_test?charset=utf8mb4",u,p,h,pt));if e!=nil{panic(e)};defer db.Close()
 start:=time.Date(2024,1,1,0,0,0,0,time.UTC).UnixMilli()
 rows,e:=db.Query(`SELECT t.net_pnl FROM agent_backtest_trades t JOIN agent_backtest_runs r ON r.run_id=t.run_id WHERE r.strategy_template_id=121 AND r.status='succeeded' AND t.entry_time>=?`,start);if e!=nil{panic(e)}
 var a S;for rows.Next(){var p float64;rows.Scan(&p);a.add(p)};rows.Close()
 fmt.Printf("OLD12_2024PLUS n=%d PF=%.9f GP=%.6f GL=%.6f net=%.6f\n",a.N,a.pf(),a.GP,a.GL,a.Net)
}