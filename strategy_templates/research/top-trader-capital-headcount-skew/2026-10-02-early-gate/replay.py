import csv, io, json, math, time, urllib.error, urllib.request, zipfile
from concurrent.futures import ThreadPoolExecutor, as_completed
from datetime import date, datetime, timedelta, timezone
from pathlib import Path

ROOT = Path("strategy_templates/research/top-trader-capital-headcount-skew/2026-10-02-early-gate")
SYMS = ["SOLUSDT","DOGEUSDT","LTCUSDT","AVAXUSDT","UNIUSDT","ZECUSDT"]
START = date(2023,1,1)
END = date(2024,12,31)
UA = "Mozilla/5.0"
METRICS = "https://data.binance.vision/data/futures/um/daily/metrics"
KLINES = "https://data.binance.vision/data/futures/um/monthly/klines"
CACHE = Path("/tmp/v147_toptrader_skew")
CACHE.mkdir(exist_ok=True)

def daterange(a,b):
    d=a
    while d<=b:
        yield d
        d += timedelta(days=1)

def fetch(url, path, retries=5):
    if path.exists():
        return path.read_bytes()
    last=None
    for i in range(retries):
        try:
            req=urllib.request.Request(url,headers={"User-Agent":UA})
            with urllib.request.urlopen(req,timeout=30) as r:
                data=r.read()
            path.write_bytes(data)
            return data
        except Exception as e:
            last=e
            time.sleep(0.4*(i+1))
    raise last
def metric_task(sym, d):
    ds=d.isoformat()
    name=f"{sym}-metrics-{ds}.zip"
    url=f"{METRICS}/{sym}/{name}"
    path=CACHE/name
    try:
        data=fetch(url,path)
        return sym,d,data,None
    except urllib.error.HTTPError as e:
        if e.code==404:
            return sym,d,None,"404"
        return sym,d,None,str(e)
    except Exception as e:
        return sym,d,None,str(e)

def parse_metric_zip(data):
    z=zipfile.ZipFile(io.BytesIO(data))
    rows=[]
    with z.open(z.namelist()[0]) as fh:
        r=csv.DictReader(io.TextIOWrapper(fh))
        for x in r:
            t=datetime.strptime(x["create_time"],"%Y-%m-%d %H:%M:%S").replace(tzinfo=timezone.utc)
            try:
                pa=float(x["count_toptrader_long_short_ratio"])
                pp=float(x["sum_toptrader_long_short_ratio"])
            except (TypeError, ValueError):
                continue
            if math.isfinite(pa) and math.isfinite(pp) and pa>0 and pp>0:
                rows.append((t, math.log(pp/pa)))
    return rows
def load_metrics():
    by={s:[] for s in SYMS}
    failures=[]
    jobs=[(s,d) for s in SYMS for d in daterange(START,END)]
    with ThreadPoolExecutor(max_workers=32) as ex:
        futs=[ex.submit(metric_task,s,d) for s,d in jobs]
        for i,f in enumerate(as_completed(futs),1):
            sym,d,data,err=f.result()
            if data is not None:
                by[sym].extend(parse_metric_zip(data))
            else:
                failures.append({"symbol":sym,"date":d.isoformat(),"error":err})
            if i%500==0:
                print("METRICS",i,"/",len(jobs),"fail",len(failures),flush=True)
    hourly={}
    for sym,rows in by.items():
        bucket={}
        for t,score in sorted(rows):
            h=t.replace(minute=0,second=0,microsecond=0)
            prev=bucket.get(h)
            if prev is None or t>prev[0]:
                bucket[h]=(t,score)
        hourly[sym]=[(h,v[1]) for h,v in sorted(bucket.items())]
    return hourly, failures
def load_klines(sym):
    bars={}
    for y in (2023,2024):
        for m in range(1,13):
            ym=f"{y:04d}-{m:02d}"
            name=f"{sym}-1h-{ym}.zip"
            url=f"{KLINES}/{sym}/1h/{name}"
            data=fetch(url,CACHE/name)
            z=zipfile.ZipFile(io.BytesIO(data))
            with z.open(z.namelist()[0]) as fh:
                for row in csv.reader(io.TextIOWrapper(fh)):
                    if not row or not row[0].isdigit():
                        continue
                    t=int(row[0]); t=t//1000 if t>10**15 else t
                    dt=datetime.fromtimestamp(t/1000,timezone.utc)
                    bars[dt]=(float(row[1]),float(row[4]))
    return bars

def mean(xs):
    return sum(xs)/len(xs) if xs else 0.0

hourly, failures = load_metrics()
prices={}
with ThreadPoolExecutor(max_workers=6) as ex:
    futs={ex.submit(load_klines,s):s for s in SYMS}
    for f in as_completed(futs):
        s=futs[f]
        prices[s]=f.result()
        print("KLINES",s,len(prices[s]),flush=True)
events=[]
for sym in SYMS:
    rows=hourly[sym]
    bars=prices[sym]
    for i in range(1,len(rows)):
        h0,s0=rows[i-1]
        h1,s1=rows[i]
        if h1-h0 != timedelta(hours=1):
            continue
        side=1 if s0<=0<s1 else (-1 if s0>=0>s1 else 0)
        if side==0:
            continue
        entry_t=h1+timedelta(hours=1)
        if entry_t.year not in (2023,2024) or entry_t not in bars:
            continue
        end12=entry_t+timedelta(hours=11)
        if end12.year != entry_t.year:
            continue
        needed=[entry_t,entry_t+timedelta(hours=3),end12]
        if any(t not in bars for t in needed):
            continue
        entry=bars[entry_t][0]
        if entry<=0:
            continue
        r1=side*math.log(bars[entry_t][1]/entry)
        r4=side*math.log(bars[entry_t+timedelta(hours=3)][1]/entry)
        r12=side*math.log(bars[end12][1]/entry)
        events.append({
            "symbol":sym,"signal_time":entry_t.isoformat(),"side":"LONG" if side>0 else "SHORT",
            "score_prev":s0,"score_now":s1,"r1":r1,"r4":r4,"r12":r12
        })
events.sort(key=lambda x:(x["signal_time"],x["symbol"]))
with (ROOT/"results/events.csv").open("w",newline="") as f:
    cols=["symbol","signal_time","side","score_prev","score_now","r1","r4","r12"]
    w=csv.DictWriter(f,fieldnames=cols)
    w.writeheader(); w.writerows(events)

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

by_sym={s:[x for x in events if x["symbol"]==s] for s in SYMS}
by_year={str(y):[x for x in events if x["signal_time"].startswith(str(y))] for y in (2023,2024)}
by_side={side:[x for x in events if x["side"]==side] for side in ("LONG","SHORT")}
overall=summarize(events)
positive=sum(1 for s in SYMS if summarize(by_sym[s])["mean_r12"]>0)
weeks=((date(2025,1,1)-START).days/7.0)*len(SYMS)
freq=len(events)/weeks
gate_pass=(
    overall["mean_r12"]>=0.002
    and positive>=4
    and freq>=0.30
    and summarize(by_year["2023"])["mean_r12"]>0
    and summarize(by_year["2024"])["mean_r12"]>0
)
summary={
    "version":"v147",
    "status":"promote_strict_engine" if gate_pass else "frozen_failed_early_gate",
    "gate_pass":gate_pass,
    "events":len(events),
    "positive_symbols":f"{positive}/6",
    "frequency_per_symbol_week":freq,
    "overall":overall,
    "by_symbol":{s:summarize(by_sym[s]) for s in SYMS},
    "by_year":{y:summarize(rows) for y,rows in by_year.items()},
    "by_side":{s:summarize(rows) for s,rows in by_side.items()},
    "metrics_download_failures":failures,
    "oos_2025_plus_evaluated":False,
    "strict_engine_run":False
}
(ROOT/"results/summary.json").write_text(json.dumps(summary,indent=2))
(ROOT/"results/download_failures.json").write_text(json.dumps(failures,indent=2))
print(json.dumps({
    "events":len(events),"positive_symbols":summary["positive_symbols"],
    "frequency_per_symbol_week":freq,
    "mean_r1":overall["mean_r1"],"mean_r4":overall["mean_r4"],"mean_r12":overall["mean_r12"],
    "2023_r12":summary["by_year"]["2023"]["mean_r12"],
    "2024_r12":summary["by_year"]["2024"]["mean_r12"],
    "gate_pass":gate_pass,"download_failures":len(failures)
},indent=2))
