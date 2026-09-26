package main
import(
 "database/sql";"fmt";"strings";"time"
 _ "go_binance_futures/bootstrap"
 "github.com/beego/beego/v2/core/config"
 _ "github.com/go-sql-driver/mysql"
)
type F struct{t int64;r float64}
func main(){
 u,_:=config.String("database::username");pw,_:=config.String("database::password");h,_:=config.String("database::host");pt,_:=config.String("database::port");dbn,_:=config.String("database::dbname")
 if dbn!="go_bn_test"{panic(dbn)}
 db,e:=sql.Open("mysql",fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4",u,pw,h,pt,dbn));if e!=nil{panic(e)};defer db.Close()
 rows,e:=db.Query("SELECT symbol,MIN(open_time) FROM market_klines_1h WHERE market='futures_usdt' GROUP BY symbol");if e!=nil{panic(e)}
 first:=map[string]int64{};for rows.Next(){var s string;var t int64;rows.Scan(&s,&t);first[s]=t};rows.Close()
 rows,e=db.Query("SELECT symbol,funding_time,funding_rate FROM market_funding_rates WHERE market='futures_usdt' ORDER BY symbol,funding_time");if e!=nil{panic(e)}
 cur:="";var fs []F;events:=0;eligible:=0
 flush:=func(){
   if len(fs)<4||!strings.HasSuffix(cur,"USDT"){return}
   for i:=3;i<len(fs);i++{
     g0:=float64(fs[i-2].t-fs[i-3].t)/3600000
     g1:=float64(fs[i-1].t-fs[i-2].t)/3600000
     g2:=float64(fs[i].t-fs[i-1].t)/3600000
     if g0>=7.5&&g1>=7.5&&g2>=3.5&&g2<=4.5{
       events++;age:=float64(fs[i].t-first[cur])/86400000
       ok:=first[cur]>0&&age>=730
       if ok{eligible++}
       fmt.Printf("%s %s prevRate=%g age=%.1f eligible=%v\n",cur,time.UnixMilli(fs[i].t).UTC().Format(time.RFC3339),fs[i-1].r,age,ok)
     }
   }
 }
 for rows.Next(){var s string;var t int64;var rate float64;rows.Scan(&s,&t,&rate);if cur!=""&&s!=cur{flush();fs=nil};cur=s;fs=append(fs,F{t,rate})};flush();rows.Close()
 fmt.Println("events",events,"eligible_ge2y",eligible)
}
