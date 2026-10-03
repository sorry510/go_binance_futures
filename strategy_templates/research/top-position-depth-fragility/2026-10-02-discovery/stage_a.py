import csv, io, json, math, time, urllib.error, urllib.request, zipfile
from concurrent.futures import ThreadPoolExecutor, as_completed
from datetime import date, datetime, timedelta, timezone
from pathlib import Path

ROOT=Path("strategy_templates/research/top-position-depth-fragility/2026-10-02-discovery")
SYMS=["SOLUSDT","DOGEUSDT","LTCUSDT","AVAXUSDT","UNIUSDT","ZECUSDT"]
YEAR=2023
UA="Mozilla/5.0"
VISION="https://data.binance.vision/data/futures/um/daily"
METRIC_CACHE=Path("/tmp/v147_toptrader_skew")
PRICE_CACHE=METRIC_CACHE
START=date(YEAR,1,1); END=date(YEAR,12,31)

def dates(a,b):
    d=a
    while d<=b:
        yield d
        d+=timedelta(days=1)

def fetch_bytes(url,retries=5):
    last=None
    for i in range(retries):
        try:
            req=urllib.request.Request(url,headers={"User-Agent":UA})
            with urllib.request.urlopen(req,timeout=35) as r:
                return r.read()
        except Exception as e:
            last=e; time.sleep(.5*(i+1))
    raise last
def parse_metrics(sym):
    hourly={}
    failures=[]
    for day in dates(START,END):
        ds=day.isoformat()
        name=f"{sym}-metrics-{ds}.zip"
        path=METRIC_CACHE/name
        try:
            if path.exists():
                data=path.read_bytes()
            else:
                url=f"{VISION}/metrics/{sym}/{name}"
                data=fetch_bytes(url)
                path.write_bytes(data)
            z=zipfile.ZipFile(io.BytesIO(data))
            with z.open(z.namelist()[0]) as fh:
                r=csv.DictReader(io.TextIOWrapper(fh))
                for x in r:
                    try:
                        t=datetime.strptime(x["create_time"],"%Y-%m-%d %H:%M:%S").replace(tzinfo=timezone.utc)
                        pos=float(x["sum_toptrader_long_short_ratio"])
                    except (TypeError,ValueError):
                        continue
                    if not math.isfinite(pos) or pos<=0:
                        continue
                    h=t.replace(minute=0,second=0,microsecond=0)
                    prev=hourly.get(h)
                    if prev is None or t>prev[0]:
                        hourly[h]=(t,pos)
        except Exception as e:
            failures.append({"symbol":sym,"date":ds,"kind":"metrics","error":str(e)})
    return {h:v[1] for h,v in hourly.items()},failures
def parse_depth_zip(data):
    z=zipfile.ZipFile(io.BytesIO(data))
    best={}
    cur_t=None; bid=None; ask=None

    def finalize(t,bid,ask):
        if t is None or bid is None or ask is None or bid<=0 or ask<=0:
            return
        h=t.replace(minute=0,second=0,microsecond=0)
        prev=best.get(h)
        if prev is None or t>prev[0]:
            best[h]=(t,bid,ask)

    with z.open(z.namelist()[0]) as fh:
        r=csv.DictReader(io.TextIOWrapper(fh))
        for x in r:
            t=datetime.strptime(x["timestamp"],"%Y-%m-%d %H:%M:%S").replace(tzinfo=timezone.utc)
            if cur_t is not None and t!=cur_t:
                finalize(cur_t,bid,ask)
                bid=None; ask=None
            cur_t=t
            pct=int(float(x["percentage"]))
            n=float(x["notional"])
            if pct==-1: bid=n
            elif pct==1: ask=n
        finalize(cur_t,bid,ask)
    return {h:(v[1],v[2]) for h,v in best.items()}

def depth_task(sym,day):
    ds=day.isoformat()
    name=f"{sym}-bookDepth-{ds}.zip"
    url=f"{VISION}/bookDepth/{sym}/{name}"
    try:
        return sym,day,parse_depth_zip(fetch_bytes(url)),None
    except Exception as e:
        return sym,day,{},str(e)
metrics={}
failures=[]
for sym in SYMS:
    m,e=parse_metrics(sym)
    metrics[sym]=m; failures.extend(e)
    print("METRICS",sym,len(m),"errors",len(e),flush=True)

depth={s:{} for s in SYMS}
jobs=[(s,d) for s in SYMS for d in dates(START,END)]
with ThreadPoolExecutor(max_workers=24) as ex:
    futs=[ex.submit(depth_task,s,d) for s,d in jobs]
    for i,f in enumerate(as_completed(futs),1):
        sym,day,vals,err=f.result()
        depth[sym].update(vals)
        if err:
            failures.append({"symbol":sym,"date":day.isoformat(),"kind":"bookDepth","error":err})
        if i%250==0:
            print("DEPTH",i,"/",len(jobs),"errors",sum(1 for x in failures if x["kind"]=="bookDepth"),flush=True)

def load_price(sym):
    bars={}
    for m in range(1,13):
        ym=f"{YEAR:04d}-{m:02d}"
        name=f"{sym}-1h-{ym}.zip"
        path=PRICE_CACHE/name
        if not path.exists():
            url=f"https://data.binance.vision/data/futures/um/monthly/klines/{sym}/1h/{name}"
            path.write_bytes(fetch_bytes(url))
        z=zipfile.ZipFile(path)
        with z.open(z.namelist()[0]) as fh:
            for r in csv.reader(io.TextIOWrapper(fh)):
                if not r or not r[0].isdigit(): continue
                t=int(r[0]); t=t//1000 if t>10**15 else t
                dt=datetime.fromtimestamp(t/1000,timezone.utc)
                bars[dt]=(float(r[1]),float(r[4]))
    return bars
features={}
coverage={}
hours_total=365*24
for sym in SYMS:
    rows={}
    for h,pos in metrics[sym].items():
        if h in depth[sym]:
            bid,ask=depth[sym][h]
            if bid>0 and ask>0 and pos>0:
                rows[h]={
                    "position_ratio":pos,
                    "bid_1pct":bid,
                    "ask_1pct":ask,
                    "score":math.log(pos*ask/bid),
                }
    features[sym]=rows
    coverage[sym]=len(rows)/hours_total
    print("FEATURE",sym,len(rows),f"coverage={coverage[sym]:.4f}",flush=True)

data_gate=all(coverage[s]>=0.95 for s in SYMS)
with (ROOT/"inputs/hourly_features_2023.csv").open("w",newline="") as f:
    cols=["symbol","hour","position_ratio","bid_1pct","ask_1pct","score"]
    w=csv.DictWriter(f,fieldnames=cols); w.writeheader()
    for sym in SYMS:
        for h,r in sorted(features[sym].items()):
            w.writerow({"symbol":sym,"hour":h.isoformat(),**r})

(ROOT/"results/data_coverage.json").write_text(json.dumps({
    "hours_total_per_symbol":hours_total,
    "coverage":coverage,
    "data_gate_pass":data_gate,
    "source_failures":failures,
},indent=2))
if not data_gate:
    print(json.dumps({"data_gate_pass":False,"coverage":coverage,"failures":len(failures)},indent=2))
    raise SystemExit(0)
prices={}
with ThreadPoolExecutor(max_workers=6) as ex:
    futs={ex.submit(load_price,s):s for s in SYMS}
    for f in as_completed(futs):
        s=futs[f]; prices[s]=f.result()
        print("PRICE",s,len(prices[s]),flush=True)

events=[]
for sym in SYMS:
    pts=sorted((h,r["score"]) for h,r in features[sym].items())
    bars=prices[sym]
    for i in range(1,len(pts)):
        h0,s0=pts[i-1]; h1,s1=pts[i]
        if h1-h0!=timedelta(hours=1):
            continue
        side=-1 if s0<=0<s1 else (1 if s0>=0>s1 else 0)
        if side==0:
            continue
        entry_t=h1+timedelta(hours=1)
        t4=entry_t+timedelta(hours=3)
        t12=entry_t+timedelta(hours=11)
        if entry_t.year!=YEAR or t12.year!=YEAR:
            continue
        if entry_t not in bars or t4 not in bars or t12 not in bars:
            continue
        entry=bars[entry_t][0]
        if entry<=0: continue
        events.append({
            "symbol":sym,"signal_time":entry_t.isoformat(),
            "side":"LONG" if side>0 else "SHORT",
            "score_prev":s0,"score_now":s1,
            "r1":side*math.log(bars[entry_t][1]/entry),
            "r4":side*math.log(bars[t4][1]/entry),
            "r12":side*math.log(bars[t12][1]/entry),
        })
events.sort(key=lambda x:(x["signal_time"],x["symbol"]))
def mean(xs): return sum(xs)/len(xs) if xs else 0.0
def summarize(rows):
    return {
        "n":len(rows),
        "mean_r1":mean([x["r1"] for x in rows]),
        "mean_r4":mean([x["r4"] for x in rows]),
        "mean_r12":mean([x["r12"] for x in rows]),
        "win_r12":mean([1.0 if x["r12"]>0 else 0.0 for x in rows]),
        "long":sum(1 for x in rows if x["side"]=="LONG"),
        "short":sum(1 for x in rows if x["side"]=="SHORT"),
    }

with (ROOT/"results/stage_a_events.csv").open("w",newline="") as f:
    cols=["symbol","signal_time","side","score_prev","score_now","r1","r4","r12"]
    w=csv.DictWriter(f,fieldnames=cols); w.writeheader(); w.writerows(events)

by_sym={s:[x for x in events if x["symbol"]==s] for s in SYMS}
by_side={s:[x for x in events if x["side"]==s] for s in ("LONG","SHORT")}
overall=summarize(events)
positive=sum(1 for s in SYMS if summarize(by_sym[s])["mean_r12"]>0)
weeks=(365/7)*len(SYMS)
freq=len(events)/weeks
gate=overall["mean_r12"]>=0.002 and positive>=4 and freq>=0.30
summary={
    "version":"v149","stage":"A_2023","data_gate_pass":True,
    "stage_gate_pass":gate,"status":"promote_stage_b_2024" if gate else "frozen_failed_stage_a",
    "events":len(events),"positive_symbols":f"{positive}/6","frequency_per_symbol_week":freq,
    "overall":overall,"by_symbol":{s:summarize(by_sym[s]) for s in SYMS},
    "by_side":{s:summarize(by_side[s]) for s in ("LONG","SHORT")},
    "coverage":coverage,"2024_evaluated":False,"2025_plus_evaluated":False
}
(ROOT/"results/stage_a_summary.json").write_text(json.dumps(summary,indent=2))
print(json.dumps({
    "data_gate_pass":True,"events":len(events),"positive_symbols":summary["positive_symbols"],
    "frequency_per_symbol_week":freq,"mean_r1":overall["mean_r1"],
    "mean_r4":overall["mean_r4"],"mean_r12":overall["mean_r12"],
    "stage_gate_pass":gate
},indent=2))
