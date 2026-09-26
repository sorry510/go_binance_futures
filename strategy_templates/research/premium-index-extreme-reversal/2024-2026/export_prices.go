package main

import (
    "database/sql"
    "encoding/csv"
    "fmt"
    "os"
    "strconv"

    _ "go_binance_futures/bootstrap"
    "github.com/beego/beego/v2/core/config"
    _ "github.com/go-sql-driver/mysql"
)

func main() {
    syms:=[]string{"BTCUSDT","ETHUSDT","BNBUSDT","XRPUSDT","SOLUSDT","DOGEUSDT","LTCUSDT","AVAXUSDT","UNIUSDT","ZECUSDT"}
    u,_:=config.String("database::username"); pw,_:=config.String("database::password")
    h,_:=config.String("database::host"); pt,_:=config.String("database::port"); dbn,_:=config.String("database::dbname")
    if dbn!="go_bn_test" { panic("refuse database "+dbn) }
    db,err:=sql.Open("mysql",fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4",u,pw,h,pt,dbn))
    if err!=nil{panic(err)}
    defer db.Close()
    ph:="'"+syms[0]+"'"
    for _,s:=range syms[1:]{ph+=",'"+s+"'"}
    start:=int64(1704067200000) // 2024-01-01
    end:=int64(1798761600000)   // 2027-01-01
    q:="SELECT symbol,open_time,open_price,close_price FROM market_klines_1h WHERE market='futures_usdt' AND symbol IN ("+ph+") AND open_time>=? AND open_time<? ORDER BY symbol,open_time"
    rows,err:=db.Query(q,start,end)
    if err!=nil{panic(err)}
    defer rows.Close()
    f,err:=os.Create("strategy_templates/research/premium-index-extreme-reversal/2024-2026/inputs/prices.csv")
    if err!=nil{panic(err)}
    defer f.Close()
    w:=csv.NewWriter(f); defer w.Flush()
    w.Write([]string{"symbol","open_time","open","close"})
    n:=0
    for rows.Next(){
        var s string; var t int64; var o,c float64
        if err:=rows.Scan(&s,&t,&o,&c);err!=nil{panic(err)}
        w.Write([]string{s,strconv.FormatInt(t,10),fmt.Sprint(o),fmt.Sprint(c)})
        n++
    }
    fmt.Println("rows",n)
}
