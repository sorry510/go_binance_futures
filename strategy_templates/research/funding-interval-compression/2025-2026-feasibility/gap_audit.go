package main

import (
  "database/sql"
  "fmt"
  "math"
  "sort"
  _ "go_binance_futures/bootstrap"
  "github.com/beego/beego/v2/core/config"
  _ "github.com/go-sql-driver/mysql"
)

type key struct{ Gap float64 }
func main(){
  u,_:=config.String("database::username");pw,_:=config.String("database::password")
  h,_:=config.String("database::host");pt,_:=config.String("database::port");dbn,_:=config.String("database::dbname")
  if dbn!="go_bn_test"{panic(dbn)}
  db,err:=sql.Open("mysql",fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4",u,pw,h,pt,dbn));if err!=nil{panic(err)};defer db.Close()
  rows,err:=db.Query("SELECT symbol,funding_time FROM market_funding_rates WHERE market='futures_usdt' AND funding_time>=? ORDER BY symbol,funding_time",int64(1746144000000));if err!=nil{panic(err)};defer rows.Close()
  last:=map[string]int64{}; counts:=map[float64]int{}; syms:=map[float64]map[string]bool{}
  for rows.Next(){
    var s string;var t int64;rows.Scan(&s,&t)
    if p:=last[s];p>0{
      g:=math.Round((float64(t-p)/3600000.0)*100)/100
      counts[g]++
      if syms[g]==nil{syms[g]=map[string]bool{}}
      syms[g][s]=true
    }
    last[s]=t
  }
  var gaps []float64
  for g:=range counts{gaps=append(gaps,g)}
  sort.Slice(gaps,func(i,j int)bool{return counts[gaps[i]]>counts[gaps[j]]})
  for i,g:=range gaps{
    if i>=30{break}
    fmt.Println(g,counts[g],len(syms[g]))
  }
}
