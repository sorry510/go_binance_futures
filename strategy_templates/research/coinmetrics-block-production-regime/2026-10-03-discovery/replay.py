import csv, io, json, math, time, urllib.request, urllib.error, zipfile
from datetime import date, datetime, timedelta, timezone
from pathlib import Path
from collections import defaultdict
from concurrent.futures import ThreadPoolExecutor, as_completed

ROOT=Path("strategy_templates/research/coinmetrics-block-production-regime")
DISC=ROOT/"2026-10-03-discovery"
ELIG=json.loads((ROOT/"2026-10-03-feasibility/inputs/eligible_signals.json").read_text())
SYMS=sorted({e["symbol"] for e in ELIG if e.get("eligible")})
UA="Mozilla/5.0"
CACHE=Path("/tmp/network_activity_cache"); CACHE.mkdir(exist_ok=True)
VISION="https://data.binance.vision/data/futures/um/monthly/klines"
UTC=timezone.utc

MONTHS=[]
d=date(2022,12,1)
while d<=date(2024,12,1):
    MONTHS.append(d.strftime("%Y-%m"))
    d=(d.replace(day=28)+timedelta(days=4)).replace(day=1)

def fetch_month(sym,mo):
    p=CACHE/f"{sym}-{mo}.zip"
    if p.exists(): return p
    url=f"{VISION}/{sym}/1d/{sym}-1d-{mo}.zip"
    for i in range(5):
        try:
            req=urllib.request.Request(url,headers={"User-Agent":UA})
            b=urllib.request.urlopen(req,timeout=25).read()
            zipfile.ZipFile(io.BytesIO(b))
            p.write_bytes(b); return p
        except urllib.error.HTTPError as e:
            if e.code==404:
                p.write_bytes(b""); return p
            if i==4: raise
        except Exception:
            if i==4:
                p.write_bytes(b""); return p
        time.sleep(.5*(i+1))
    return p

def read_month(sym,mo):
    p=fetch_month(sym,mo)
    if not p.exists() or p.stat().st_size==0: return []
    z=zipfile.ZipFile(p)
    out=[]
    with z.open(z.namelist()[0]) as fh:
        for r in csv.reader(io.TextIOWrapper(fh)):
            if not r or not r[0].isdigit() or len(r)<8: continue
            t=int(r[0]); t=t//1000 if t>10**15 else t
            dd=datetime.fromtimestamp(t/1000,UTC).date()
            out.append((dd,float(r[1]),float(r[4]),float(r[7])))
    return out

jobs=[(s,m) for s in SYMS for m in MONTHS]
with ThreadPoolExecutor(max_workers=12) as ex:
    fs=[ex.submit(fetch_month,*j) for j in jobs]
    for i,f in enumerate(as_completed(fs),1):
        f.result()
        if i%50==0 or i==len(fs):
            print("VISION",i,"/",len(fs),flush=True)

price={}
for sym in SYMS:
    pm={}
    for mo in MONTHS:
        for dd,o,c,qv in read_month(sym,mo):
            pm[dd]=(o,c,qv)
    price[sym]=pm
    print("PRICE",sym,len(pm),min(pm) if pm else None,max(pm) if pm else None,flush=True)

events=[]; errors=[]
for e in ELIG:
    if not e.get("eligible"): continue
    sym=e["symbol"]; side=1.0 if e["side"]=="LONG" else -1.0
    sd=date.fromisoformat(e["signal_date"])
    entry_day=sd+timedelta(days=1)
    d1=entry_day
    d3=entry_day+timedelta(days=2)
    d7=entry_day+timedelta(days=6)
    if d7.year!=sd.year:
        errors.append({**e,"reason":"7d_endpoint_crosses_signal_year"}); continue
    pm=price[sym]
    if any(x not in pm for x in (entry_day,d1,d3,d7)):
        errors.append({**e,"reason":"missing_price_day"}); continue
    entry=pm[entry_day][0]
    if entry<=0:
        errors.append({**e,"reason":"invalid_entry"}); continue
    vals={}
    good=True
    for h,dd in ((1,d1),(3,d3),(7,d7)):
        close=pm[dd][1]
        if close<=0: good=False; break
        vals[h]=side*math.log(close/entry)
    if not good:
        errors.append({**e,"reason":"invalid_close"}); continue
    events.append({
      "asset":e["asset"],"symbol":sym,"signal_date":e["signal_date"],"side":e["side"],
      "blkcnt":e["blkcnt"],"baseline30":e["baseline30"],
      "score_prev":e["score_prev"],"score_now":e["score_now"],
      "qv24":e["qv24"],"r1":vals[1],"r3":vals[3],"r7":vals[7]
    })

def mean(xs): return sum(xs)/len(xs) if xs else 0.0
def summ(rows):
    if not rows: return {"n":0,"mean_r1":0.0,"mean_r3":0.0,"mean_r7":0.0,"win_r7":0.0,"long":0,"short":0}
    return {
      "n":len(rows),
      "mean_r1":mean([x["r1"] for x in rows]),
      "mean_r3":mean([x["r3"] for x in rows]),
      "mean_r7":mean([x["r7"] for x in rows]),
      "win_r7":mean([1.0 if x["r7"]>0 else 0.0 for x in rows]),
      "long":sum(1 for x in rows if x["side"]=="LONG"),
      "short":sum(1 for x in rows if x["side"]=="SHORT"),
    }

by_sym=defaultdict(list); by_year=defaultdict(list); by_side=defaultdict(list)
for e in events:
    by_sym[e["symbol"]].append(e)
    by_year[e["signal_date"][:4]].append(e)
    by_side[e["side"]].append(e)

symbols=sorted(by_sym)
positive=sum(1 for s in symbols if summ(by_sym[s])["mean_r7"]>0)
weeks=((date(2025,1,1)-date(2023,1,1)).days/7.0)*len(symbols) if symbols else 0
freq=len(events)/weeks if weeks else 0.0
overall=summ(events)
gate=(len(events)>=80 and len(symbols)>=8 and overall["mean_r7"]>=0.0025 and
      positive/len(symbols)>=0.60 and freq>=0.30 and
      summ(by_year["2023"])["mean_r7"]>0 and summ(by_year["2024"])["mean_r7"]>0)

summary={
 "version":"v195",
 "status":"promote_oos" if gate else "frozen_failed_discovery",
 "gate_pass":gate,
 "events":len(events),
 "symbols":len(symbols),
 "positive_symbols":f"{positive}/{len(symbols)}",
 "positive_symbol_fraction":positive/len(symbols) if symbols else 0,
 "frequency_per_symbol_week":freq,
 "overall":overall,
 "by_year":{k:summ(v) for k,v in sorted(by_year.items())},
 "by_side":{k:summ(v) for k,v in sorted(by_side.items())},
 "by_symbol":{k:summ(v) for k,v in sorted(by_sym.items())},
 "replay_errors":len(errors),
 "oos_2025_plus_evaluated":False,
 "strict_engine_run":False,
 "db_write":False
}
(DISC/"results/events.json").write_text(json.dumps(events,indent=2))
(DISC/"results/replay_errors.json").write_text(json.dumps(errors,indent=2))
(DISC/"results/summary.json").write_text(json.dumps(summary,indent=2))
print(json.dumps({
 "events":len(events),"symbols":len(symbols),"positive_symbols":summary["positive_symbols"],
 "freq":freq,"mean_r1":overall["mean_r1"],"mean_r3":overall["mean_r3"],"mean_r7":overall["mean_r7"],
 "2023_r7":summary["by_year"].get("2023",{}).get("mean_r7",0),
 "2024_r7":summary["by_year"].get("2024",{}).get("mean_r7",0),
 "long_r7":summary["by_side"].get("LONG",{}).get("mean_r7",0),
 "short_r7":summary["by_side"].get("SHORT",{}).get("mean_r7",0),
 "errors":len(errors),"gate_pass":gate
},indent=2),flush=True)
