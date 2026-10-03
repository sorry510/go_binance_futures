package main

import (
    "context"
    "database/sql"
    "encoding/json"
    "fmt"
    "math"
    "os"
    "sort"
    "time"
)

type v191Event struct {
    Symbol string  `json:"symbol"`
    SignalTime int64 `json:"signal_time"`
    Year int `json:"year"`
    Side string `json:"side"`
    SkewPrev float64 `json:"skew_prev"`
    SkewNow float64 `json:"skew_now"`
    R1 float64 `json:"r1"`
    R4 float64 `json:"r4"`
    R12 float64 `json:"r12"`
}

type v191Stats struct {
    N int `json:"n"`
    MeanR1 float64 `json:"mean_r1"`
    MeanR4 float64 `json:"mean_r4"`
    MeanR12 float64 `json:"mean_r12"`
    WinR12 float64 `json:"win_r12"`
    Long int `json:"long"`
    Short int `json:"short"`
}

type v191Source struct {
    MinuteRows int `json:"minute_rows"`
    HourRows int `json:"hour_rows"`
    MinuteFirst int64 `json:"minute_first"`
    MinuteLast int64 `json:"minute_last"`
    HourFirst int64 `json:"hour_first"`
    HourLast int64 `json:"hour_last"`
    CandidateHours int `json:"candidate_hours"`
    CompleteFormationHours int `json:"complete_formation_hours"`
}

type v191Summary struct {
    Version string `json:"version"`
    Status string `json:"status"`
    GatePass bool `json:"gate_pass"`
    Events int `json:"events"`
    PositiveSymbols string `json:"positive_symbols"`
    FrequencyPerSymbolWeek float64 `json:"frequency_per_symbol_week"`
    Overall v191Stats `json:"overall"`
    BySymbol map[string]v191Stats `json:"by_symbol"`
    ByYear map[string]v191Stats `json:"by_year"`
    BySide map[string]v191Stats `json:"by_side"`
    SourceMeta map[string]v191Source `json:"source_meta"`
    OOS2025PlusEvaluated bool `json:"oos_2025_plus_evaluated"`
    StrictEngineRun bool `json:"strict_engine_run"`
    DBWrite bool `json:"db_write"`
}

func v191Skew(xs []float64) (float64, bool) {
    n := len(xs)
    if n < 3 { return 0, false }
    var mean float64
    for _, x := range xs { mean += x }
    mean /= float64(n)
    var ss, cube float64
    for _, x := range xs {
        d := x - mean
        ss += d*d
        cube += d*d*d
    }
    if ss <= 0 { return 0, false }
    s := math.Sqrt(ss / float64(n-1))
    if s <= 0 { return 0, false }
    g := float64(n) / (float64(n-1)*float64(n-2)) * cube / (s*s*s)
    return g, true
}

func v191Summ(rows []v191Event) v191Stats {
    s := v191Stats{N: len(rows)}
    if len(rows)==0 { return s }
    for _, e := range rows {
        s.MeanR1 += e.R1
        s.MeanR4 += e.R4
        s.MeanR12 += e.R12
        if e.R12 > 0 { s.WinR12++ }
        if e.Side=="LONG" { s.Long++ } else { s.Short++ }
    }
    n:=float64(len(rows))
    s.MeanR1/=n; s.MeanR4/=n; s.MeanR12/=n; s.WinR12/=n
    return s
}

func main() {
    if err:=runV191(); err!=nil {
        fmt.Fprintln(os.Stderr, "v191 error:", err)
        os.Exit(1)
    }
}

func runV191() error {
    const H int64 = 3_600_000
    const M int64 = 60_000
    symbols := []string{"SOLUSDT","DOGEUSDT","LTCUSDT","AVAXUSDT","UNIUSDT","ZECUSDT"}
    start := time.Date(2023,1,1,0,0,0,0,time.UTC).UnixMilli()
    endExclusive := time.Date(2025,1,1,0,0,0,0,time.UTC).UnixMilli()
    loadStart := start - H // prior hh:59 close needed for the first signal hour
    loadEnd := endExclusive - 1

    cfg, err := parseArmResearchConfig("conf/app.conf")
    if err != nil { return err }
    db, err := openArmResearchDB(cfg, "go_binance")
    if err != nil { return err }
    defer db.Close()
    ctx, cancel := context.WithTimeout(context.Background(), 3*time.Hour)
    defer cancel()
    if err:=db.PingContext(ctx); err!=nil { return err }

    all := make([]v191Event,0,20000)
    sources:=map[string]v191Source{}

    for _, sym := range symbols {
        tx, err := db.BeginTx(ctx, &sql.TxOptions{ReadOnly:true})
        if err != nil { return err }
        mins, err := readArmMinutes(ctx, tx, sym, loadStart, loadEnd)
        if err != nil { tx.Rollback(); return fmt.Errorf("%s minutes: %w",sym,err) }
        hours, err := readArmSeries(ctx, tx, sym, "1h", start, loadEnd)
        if err != nil { tx.Rollback(); return fmt.Errorf("%s hours: %w",sym,err) }
        if err:=tx.Rollback(); err!=nil && err!=sql.ErrTxDone { return err }

        mclose:=make(map[int64]float64,len(mins))
        for _, b:=range mins { if b.Close>0 { mclose[b.OpenTime]=b.Close } }
        hmap:=make(map[int64]struct{Open,Close float64},len(hours))
        for _, b:=range hours { if b.Open>0 && b.Close>0 { hmap[b.OpenTime]=struct{Open,Close float64}{b.Open,b.Close} } }

        src:=v191Source{MinuteRows:len(mins),HourRows:len(hours)}
        if len(mins)>0 { src.MinuteFirst=mins[0].OpenTime; src.MinuteLast=mins[len(mins)-1].OpenTime }
        if len(hours)>0 { src.HourFirst=hours[0].OpenTime; src.HourLast=hours[len(hours)-1].OpenTime }

        skewByHour:=make(map[int64]float64, 18000)
        for t:=start; t<endExclusive; t+=H {
            src.CandidateHours++
            xs:=make([]float64,0,60)
            ok:=true
            prevTime:=t-M // previous hour hh:59 minute open time
            prevClose, exists:=mclose[prevTime]
            if !exists || prevClose<=0 { continue }
            for k:=int64(0); k<60; k++ {
                mt:=t+k*M
                c, exists:=mclose[mt]
                if !exists || c<=0 || mt-prevTime!=M { ok=false; break }
                xs=append(xs, math.Log(c/prevClose))
                prevClose=c
                prevTime=mt
            }
            if !ok || len(xs)!=60 { continue }
            sk, valid:=v191Skew(xs)
            if !valid { continue }
            skewByHour[t]=sk
            src.CompleteFormationHours++
        }

        symEvents:=make([]v191Event,0,5000)
        for t:=start+H; t<endExclusive; t+=H {
            now, ok1:=skewByHour[t]
            prev, ok2:=skewByHour[t-H]
            if !ok1 || !ok2 { continue }
            side:=0.0
            sideName:=""
            if prev<=0 && now>0 { side=-1; sideName="SHORT" } else if prev>=0 && now<0 { side=1; sideName="LONG" } else { continue }

            entryBar, ok:=hmap[t+H]
            if !ok || entryBar.Open<=0 { continue }
            dt:=time.UnixMilli(t).UTC()
            if time.UnixMilli(t+12*H).UTC().Year()!=dt.Year() { continue }
            vals:=map[int]float64{}
            good:=true
            for _, hh:=range []int{1,4,12} {
                b, ok:=hmap[t+int64(hh)*H]
                if !ok || b.Close<=0 { good=false; break }
                vals[hh]=side*math.Log(b.Close/entryBar.Open)
            }
            if !good { continue }
            symEvents=append(symEvents,v191Event{
                Symbol:sym,SignalTime:t,Year:dt.Year(),Side:sideName,
                SkewPrev:prev,SkewNow:now,R1:vals[1],R4:vals[4],R12:vals[12],
            })
        }
        all=append(all,symEvents...)
        sources[sym]=src
        fmt.Printf("SYMBOL %s minute_rows=%d complete_hours=%d events=%d\n",sym,len(mins),src.CompleteFormationHours,len(symEvents))
        mins=nil; hours=nil; mclose=nil; hmap=nil; skewByHour=nil
    }

    bySym:=map[string]v191Stats{}
    pos:=0
    for _, sym:=range symbols {
        rows:=make([]v191Event,0)
        for _,e:=range all { if e.Symbol==sym { rows=append(rows,e) } }
        st:=v191Summ(rows); bySym[sym]=st
        if st.MeanR12>0 { pos++ }
    }
    byYear:=map[string]v191Stats{}
    for _, y:=range []int{2023,2024} {
        rows:=make([]v191Event,0)
        for _,e:=range all { if e.Year==y { rows=append(rows,e) } }
        byYear[fmt.Sprintf("%d",y)]=v191Summ(rows)
    }
    bySide:=map[string]v191Stats{}
    for _,side:=range []string{"LONG","SHORT"} {
        rows:=make([]v191Event,0)
        for _,e:=range all { if e.Side==side { rows=append(rows,e) } }
        bySide[side]=v191Summ(rows)
    }
    overall:=v191Summ(all)
    weeks:=float64((time.Date(2025,1,1,0,0,0,0,time.UTC).Sub(time.Date(2023,1,1,0,0,0,0,time.UTC)).Hours()/24/7)*float64(len(symbols)))
    freq:=float64(len(all))/weeks
    gate:=overall.MeanR12>=0.002 && pos>=4 && freq>=0.30 && byYear["2023"].MeanR12>0 && byYear["2024"].MeanR12>0
    status:="frozen_failed_early_gate"; if gate { status="promote_oos" }
    summary:=v191Summary{
        Version:"v191",Status:status,GatePass:gate,Events:len(all),
        PositiveSymbols:fmt.Sprintf("%d/6",pos),FrequencyPerSymbolWeek:freq,
        Overall:overall,BySymbol:bySym,ByYear:byYear,BySide:bySide,SourceMeta:sources,
        OOS2025PlusEvaluated:false,StrictEngineRun:false,DBWrite:false,
    }
    sort.Slice(all,func(i,j int)bool{ if all[i].Symbol==all[j].Symbol{return all[i].SignalTime<all[j].SignalTime};return all[i].Symbol<all[j].Symbol })
    if err:=writeJSONV191("strategy_templates/research/intrahour-1m-realized-skew-reversal/2026-10-03-early-gate/results/events.json",all);err!=nil{return err}
    if err:=writeJSONV191("strategy_templates/research/intrahour-1m-realized-skew-reversal/2026-10-03-early-gate/results/summary.json",summary);err!=nil{return err}
    b,_:=json.MarshalIndent(map[string]any{
        "events":len(all),"mean_r1":overall.MeanR1,"mean_r4":overall.MeanR4,"mean_r12":overall.MeanR12,
        "positive_symbols":summary.PositiveSymbols,"freq":freq,"2023":byYear["2023"].MeanR12,"2024":byYear["2024"].MeanR12,
        "long_r12":bySide["LONG"].MeanR12,"short_r12":bySide["SHORT"].MeanR12,"gate_pass":gate,
    },"","  ")
    fmt.Println(string(b))
    return nil
}

func writeJSONV191(path string,v any) error {
    b,err:=json.MarshalIndent(v,"","  "); if err!=nil{return err}
    return os.WriteFile(path,b,0o644)
}
