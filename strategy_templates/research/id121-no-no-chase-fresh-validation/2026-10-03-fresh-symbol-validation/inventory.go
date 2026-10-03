package main

import (
  "fmt"
  "sort"
  "strings"

  "github.com/beego/beego/v2/client/orm"
  "github.com/beego/beego/v2/core/config"
  _ "github.com/go-sql-driver/mysql"
  _ "go_binance_futures/bootstrap"
)

type R struct {
  Market string
  Symbol string
  MinStart int64
  MaxEnd int64
  Months int
  Points int64
}

func main(){
  dbn,_:=config.String("database::dbname")
  if dbn!="go_bn_test"{panic("expected go_bn_test, got "+dbn)}
  u,_:=config.String("database::username"); pw,_:=config.String("database::password")
  h,_:=config.String("database::host"); pt,_:=config.String("database::port")
  _=orm.RegisterDriver("mysql",orm.DRMySQL)
  if e:=orm.RegisterDataBase("default","mysql",fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&collation=utf8mb4_unicode_ci",u,pw,h,pt,dbn));e!=nil{panic(e)}
  o:=orm.NewOrm()
  var rows []R
  _,err:=o.Raw(`SELECT market,symbol,MIN(start_time) AS min_start,MAX(end_time) AS max_end,COUNT(*) AS months,SUM(point_count) AS points
    FROM market_klines_1m_chunks GROUP BY market,symbol ORDER BY symbol`).QueryRows(&rows)
  if err!=nil{panic(err)}
  src:=map[string]bool{}
  for _,s:=range strings.Split("ADAUSDT AVAXUSDT BNBUSDT BTCUSDT DOGEUSDT ETHUSDT LTCUSDT NEARUSDT SOLUSDT UNIUSDT XRPUSDT ZECUSDT 1000PEPEUSDT SUIUSDT ONDOUSDT"," "){src[s]=true}
  sort.Slice(rows,func(i,j int)bool{return rows[i].Symbol<rows[j].Symbol})
  for _,r:=range rows{
    fresh:="FRESH"
    if src[r.Symbol]{fresh="SOURCE"}
    fmt.Printf("%s %s %-12s min=%d max=%d months=%d points=%d\n",fresh,r.Market,r.Symbol,r.MinStart,r.MaxEnd,r.Months,r.Points)
  }
}
