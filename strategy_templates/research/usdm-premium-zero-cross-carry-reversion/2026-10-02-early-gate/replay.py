import csv, io, json, math, time, urllib.error, urllib.request, zipfile
from concurrent.futures import ThreadPoolExecutor, as_completed
from datetime import datetime, timedelta, timezone
from pathlib import Path

ROOT=Path("strategy_templates/research/usdm-premium-zero-cross-carry-reversion/2026-10-02-early-gate")
SYMS=["SOLUSDT","DOGEUSDT","LTCUSDT","AVAXUSDT","UNIUSDT","ZECUSDT"]
BASE="https://data.binance.vision/data/futures/um/monthly"
UA="Mozilla/5.0"
CACHE=Path("/tmp/v148_premium_zero_cross")
CACHE.mkdir(exist_ok=True)
MONTHS=[f"{y:04d}-{m:02d}" for y in (2023,2024) for m in range(1,13)]

def fetch(url,path,retries=5):
    if path.exists(): return path.read_bytes()
    last=None
    for i in range(retries):
        try:
            req=urllib.request.Request(url,headers={"User-Agent":UA})
            with urllib.request.urlopen(req,timeout=30) as r: data=r.read()
            path.write_bytes(data); return data
        except Exception as e:
            last=e; time.sleep(.4*(i+1))
    raise last

def parse_zip(data):
    z=zipfile.ZipFile(io.BytesIO(data))
    out=[]
    with z.open(z.namelist()[0]) as f:
        for r in csv.reader(io.TextIOWrapper(f)):
            if not r or not r[0].isdigit(): continue
            t=int(r[0]); t=t//1000 if t>10**15 else t
            out.append((datetime.fromtimestamp(t/1000,timezone.utc),r))
    return out
def load_symbol(sym):
    premium={}
    price={}
    failures=[]
    for ym in MONTHS:
        for kind,interval,target in [
            ("premiumIndexKlines","1h",premium),
            ("klines","1h",price),
        ]:
            name=f"{sym}-1h-{ym}.zip"
            url=f"{BASE}/{kind}/{sym}/{interval}/{name}"
            path=CACHE/f"{kind}-{name}"
            try:
                data=fetch(url,path)
                for dt,r in parse_zip(data):
                    if kind=="premiumIndexKlines":
                        target[dt]=float(r[4])
                    else:
                        target[dt]=(float(r[1]),float(r[4]))
            except Exception as e:
                failures.append({"symbol":sym,"month":ym,"kind":kind,"error":str(e)})
    return sym,premium,price,failures

loaded={}
failures=[]
with ThreadPoolExecutor(max_workers=6) as ex:
    futs={ex.submit(load_symbol,s):s for s in SYMS}
    for f in as_completed(futs):
        sym,premium,price,errs=f.result()
        loaded[sym]=(premium,price)
        failures.extend(errs)
        print("LOADED",sym,"premium",len(premium),"price",len(price),"errors",len(errs),flush=True)
events=[]
for sym in SYMS:
    premium,price=loaded[sym]
    pts=sorted(premium.items())
    for i in range(1,len(pts)):
        t0,p0=pts[i-1]
        t1,p1=pts[i]
        if t1-t0!=timedelta(hours=1):
            continue
        side=-1 if p0<=0<p1 else (1 if p0>=0>p1 else 0)
        if side==0:
            continue
        entry_t=t1+timedelta(hours=1)
        if entry_t.year not in (2023,2024) or entry_t not in price:
            continue
        t4=entry_t+timedelta(hours=3)
        t12=entry_t+timedelta(hours=11)
        if t12.year!=entry_t.year:
            continue
        if t4 not in price or t12 not in price:
            continue
        entry=price[entry_t][0]
        if entry<=0:
            continue
        r1=side*math.log(price[entry_t][1]/entry)
        r4=side*math.log(price[t4][1]/entry)
        r12=side*math.log(price[t12][1]/entry)
        events.append({
            "symbol":sym,"signal_time":entry_t.isoformat(),
            "side":"LONG" if side>0 else "SHORT",
            "premium_prev":p0,"premium_now":p1,
            "r1":r1,"r4":r4,"r12":r12
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

with (ROOT/"results/events.csv").open("w",newline="") as f:
    cols=["symbol","signal_time","side","premium_prev","premium_now","r1","r4","r12"]
    w=csv.DictWriter(f,fieldnames=cols); w.writeheader(); w.writerows(events)

by_sym={s:[x for x in events if x["symbol"]==s] for s in SYMS}
by_year={str(y):[x for x in events if x["signal_time"].startswith(str(y))] for y in (2023,2024)}
by_side={s:[x for x in events if x["side"]==s] for s in ("LONG","SHORT")}
overall=summarize(events)
positive=sum(1 for s in SYMS if summarize(by_sym[s])["mean_r12"]>0)
weeks=(731/7)*len(SYMS)
freq=len(events)/weeks
gate_pass=(
    overall["mean_r12"]>=0.002 and positive>=4 and freq>=0.30
    and summarize(by_year["2023"])["mean_r12"]>0
    and summarize(by_year["2024"])["mean_r12"]>0
)
summary={
    "version":"v148",
    "status":"promote_strict_validation" if gate_pass else "frozen_failed_early_gate",
    "gate_pass":gate_pass,
    "events":len(events),
    "positive_symbols":f"{positive}/6",
    "frequency_per_symbol_week":freq,
    "overall":overall,
    "by_symbol":{s:summarize(by_sym[s]) for s in SYMS},
    "by_year":{y:summarize(rows) for y,rows in by_year.items()},
    "by_side":{s:summarize(rows) for s,rows in by_side.items()},
    "download_failures":failures,
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
