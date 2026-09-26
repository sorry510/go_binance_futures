package main

import (
    "database/sql"
    "encoding/csv"
    "encoding/json"
    "fmt"
    "math"
    "os"
    "sort"
    "strconv"
    "time"

    _ "go_binance_futures/bootstrap"
    "github.com/beego/beego/v2/core/config"
    _ "github.com/go-sql-driver/mysql"
)

type Funding struct { T int64; R float64 }
type Bar struct { T int64; O, C float64 }
type Event struct {
    Symbol string
    FundingTime int64
    Year int
    Rate, Z float64
    Side string
    R1, R4, R12 float64
}
type Group struct {
    N int
    Mean1, Win1, Mean4, Win4, Mean12, Win12 float64
    BySymbol map[string]map[string]float64
    ByYear map[string]map[string]float64
}

func meanStd(x []float64) (float64, float64) {
    var s float64
    for _, v := range x { s += v }
    m := s / float64(len(x))
    var q float64
    for _, v := range x { d := v-m; q += d*d }
    return m, math.Sqrt(q/float64(len(x)))
}

func summarize(ev []Event, syms []string) Group {
    g := Group{N:len(ev), BySymbol:map[string]map[string]float64{}, ByYear:map[string]map[string]float64{}}
    acc := func(sel []Event, f func(Event)float64) (float64,float64) {
        if len(sel)==0 { return 0,0 }
        var s float64; var w int
        for _, e := range sel { v:=f(e); s+=v; if v>0 { w++ } }
        return s/float64(len(sel)), float64(w)/float64(len(sel))
    }
    g.Mean1,g.Win1=acc(ev,func(e Event)float64{return e.R1})
    g.Mean4,g.Win4=acc(ev,func(e Event)float64{return e.R4})
    g.Mean12,g.Win12=acc(ev,func(e Event)float64{return e.R12})

    for _, s := range syms {
        var q []Event
        for _, e := range ev { if e.Symbol==s { q=append(q,e) } }
        if len(q)>0 {
            m,w:=acc(q,func(e Event)float64{return e.R4})
            g.BySymbol[s]=map[string]float64{"n":float64(len(q)),"mean4":m,"win4":w}
        }
    }
    ys:=map[int]bool{}
    for _, e := range ev { ys[e.Year]=true }
    var yl []int
    for y := range ys { yl=append(yl,y) }
    sort.Ints(yl)
    for _, y := range yl {
        var q []Event
        for _, e := range ev { if e.Year==y { q=append(q,e) } }
        m,w:=acc(q,func(e Event)float64{return e.R4})
        g.ByYear[strconv.Itoa(y)]=map[string]float64{"n":float64(len(q)),"mean4":m,"win4":w}
    }
    return g
}

func main() {
    syms:=[]string{"BTCUSDT","ETHUSDT","BNBUSDT","XRPUSDT","SOLUSDT","DOGEUSDT","LTCUSDT","AVAXUSDT","UNIUSDT","ZECUSDT"}
    u,_:=config.String("database::username")
    pw,_:=config.String("database::password")
    h,_:=config.String("database::host")
    pt,_:=config.String("database::port")
    dbn,_:=config.String("database::dbname")
    if dbn!="go_bn_test" { panic("refuse database "+dbn) }
    dsn:=fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4",u,pw,h,pt,dbn)
    db,err:=sql.Open("mysql",dsn)
    if err!=nil { panic(err) }
    defer db.Close()

    start:=int64(1669852800000)
    end:=int64(1798761600000)
    fm:=map[string][]Funding{}
    bm:=map[string]map[int64]Bar{}
    ph:="'"+syms[0]+"'"
    for _,s:=range syms[1:] { ph+=",'"+s+"'" }

    qf:="SELECT symbol,funding_time,funding_rate FROM market_funding_rates WHERE market='futures_usdt' AND symbol IN ("+ph+") AND funding_time>=? AND funding_time<? ORDER BY symbol,funding_time"
    rows,err:=db.Query(qf,start,end)
    if err!=nil { panic(err) }
    for rows.Next() {
        var s string; var t int64; var r float64
        if err:=rows.Scan(&s,&t,&r); err!=nil { panic(err) }
        fm[s]=append(fm[s],Funding{t,r})
    }
    rows.Close()

    qb:="SELECT symbol,open_time,open_price,close_price FROM market_klines_1h WHERE market='futures_usdt' AND symbol IN ("+ph+") AND open_time>=? AND open_time<? ORDER BY symbol,open_time"
    rows,err=db.Query(qb,start,end)
    if err!=nil { panic(err) }
    for rows.Next() {
        var s string; var t int64; var o,c float64
        if err:=rows.Scan(&s,&t,&o,&c); err!=nil { panic(err) }
        if bm[s]==nil { bm[s]=map[int64]Bar{} }
        bm[s][t]=Bar{t,o,c}
    }
    rows.Close()

    var all []Event
    for _,s:=range syms {
        rates:=[]float64{}
        armed:=true
        for _,f:=range fm[s] {
            if len(rates)>=30 {
                hist:=rates[len(rates)-30:]
                m,sd:=meanStd(hist)
                z:=0.0
                if sd>0 { z=(f.R-m)/sd }
                if math.Abs(z)<1 { armed=true }
                if armed && math.Abs(z)>=2 && f.R!=0 {
                    entryT:=(f.T/3600000+1)*3600000
                    b,ok:=bm[s][entryT]
                    if ok {
                        dir:=1.0; side:="LONG"
                        if f.R>0 { dir=-1; side="SHORT" }
                        get:=func(hours int)(float64,bool) {
                            x,ok:=bm[s][entryT+int64(hours-1)*3600000]
                            if !ok { return 0,false }
                            return dir*math.Log(x.C/b.O),true
                        }
                        r1,o1:=get(1); r4,o4:=get(4); r12,o12:=get(12)
                        if o1&&o4&&o12 {
                            y:=time.UnixMilli(f.T).UTC().Year()
                            all=append(all,Event{s,f.T,y,f.R,z,side,r1,r4,r12})
                            armed=false
                        }
                    }
                }
            }
            rates=append(rates,f.R)
        }
        fmt.Println("SYMBOL",s,"funding",len(fm[s]))
    }

    var disc,oos []Event
    for _,e:=range all {
        if e.Year==2023||e.Year==2024 { disc=append(disc,e) }
        if e.Year==2025||e.Year==2026 { oos=append(oos,e) }
    }
    res:=map[string]Group{
        "discovery_2023_2024":summarize(disc,syms),
        "oos_2025_2026":summarize(oos,syms),
        "all":summarize(all,syms),
    }
    b,_:=json.MarshalIndent(res,"","  ")
    fmt.Println(string(b))
    os.WriteFile("/tmp/funding_shock_summary.json",b,0644)

    f,_:=os.Create("/tmp/funding_shock_events.csv")
    defer f.Close()
    w:=csv.NewWriter(f)
    defer w.Flush()
    w.Write([]string{"symbol","funding_time","year","rate","z","side","r1","r4","r12"})
    for _,e:=range all {
        w.Write([]string{e.Symbol,strconv.FormatInt(e.FundingTime,10),strconv.Itoa(e.Year),fmt.Sprint(e.Rate),fmt.Sprint(e.Z),e.Side,fmt.Sprint(e.R1),fmt.Sprint(e.R4),fmt.Sprint(e.R12)})
    }
}
