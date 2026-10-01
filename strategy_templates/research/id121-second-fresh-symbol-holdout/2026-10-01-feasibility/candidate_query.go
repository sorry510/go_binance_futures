package main
import (
  "fmt"
  "strings"
  "github.com/beego/beego/v2/client/orm"
  "github.com/beego/beego/v2/core/config"
  _ "github.com/go-sql-driver/mysql"
  _ "go_binance_futures/bootstrap"
)
type Row struct {
  Market string
  Symbol string
  Earliest int64
  Latest int64
  Months int
  QuoteVolume float64
  Enable int
  Type string
}
func main(){
  dbn,_:=config.String("database::dbname"); if dbn!="go_bn_test"{panic("expected go_bn_test, got "+dbn)}
  u,_:=config.String("database::username"); pw,_:=config.String("database::password"); h,_:=config.String("database::host"); pt,_:=config.String("database::port")
  _=orm.RegisterDriver("mysql",orm.DRMySQL)
  if e:=orm.RegisterDataBase("default","mysql",fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&collation=utf8mb4_unicode_ci",u,pw,h,pt,dbn));e!=nil{panic(e)}
  var rows []Row
  q:=`
SELECT c.market,c.symbol,MIN(c.start_time) earliest,MAX(c.end_time) latest,COUNT(*) months,
       COALESCE(s.quoteVolume,0) quote_volume,COALESCE(s.enable,0) enable,COALESCE(s.type,'') type
FROM market_klines_1m_chunks c
LEFT JOIN symbols s ON s.symbol=c.symbol
GROUP BY c.market,c.symbol,s.quoteVolume,s.enable,s.type
ORDER BY quote_volume DESC,c.symbol ASC`
  if _,e:=orm.NewOrm().Raw(q).QueryRows(&rows);e!=nil{panic(e)}
  excluded:=map[string]bool{}
  for _,s:=range strings.Fields("BTCUSDT ETHUSDT BNBUSDT XRPUSDT SOLUSDT DOGEUSDT LTCUSDT AVAXUSDT UNIUSDT ZECUSDT ADAUSDT NEARUSDT 1000PEPEUSDT SUIUSDT ONDOUSDT ALGOUSDT INJUSDT LDOUSDT PENDLEUSDT PYTHUSDT"){excluded[s]=true}
  const earliestNeed int64=1672531200000
  const latestNeed int64=1788220800000
  _ = latestNeed
  n:=0
  for _,r:=range rows{
    if excluded[r.Symbol] || !strings.EqualFold(r.Type,"USDT") || r.QuoteVolume<5000000 || r.Earliest>earliestNeed || r.Latest<1788220740000 {continue}
    fmt.Printf("%s market=%s qv=%.0f earliest=%d latest=%d months=%d\n",r.Symbol,r.Market,r.QuoteVolume,r.Earliest,r.Latest,r.Months)
    n++
  }
  fmt.Printf("ELIGIBLE_COUNT=%d\n",n)
}
