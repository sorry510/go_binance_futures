import csv, io, json, math, time, urllib.request, urllib.error, zipfile
from datetime import date, datetime, timedelta, timezone
from pathlib import Path
from collections import defaultdict
from concurrent.futures import ThreadPoolExecutor, as_completed

ROOT=Path("strategy_templates/research/coinmetrics-net-supply-growth/2026-10-02-discovery")
ROWS=json.loads((ROOT/"inputs/splycur_2023_2024.json").read_text())
ASSETS=["btc","eth","xrp","ada","bch","ltc","doge","zec"]
SYM={a:a.upper()+"USDT" for a in ASSETS}
UA="Mozilla/5.0"
CACHE=Path("/tmp/network_activity_cache")
CACHE.mkdir(exist_ok=True)
VISION="https://data.binance.vision/data/futures/um/monthly/klines"
UTC=timezone.utc

def months_between(a,b):
    d=date(a.year,a.month,1); e=date(b.year,b.month,1); out=[]
    while d<=e:
        out.append(d.strftime("%Y-%m"))
        d=(d.replace(day=28)+timedelta(days=4)).replace(day=1)
    return out

DISC_MONTHS=months_between(date(2022,12,1),date(2024,12,1))
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
            d=datetime.fromtimestamp(t/1000,UTC).date()
            out.append((d,float(r[1]),float(r[4]),float(r[7])))
    return out
# Point-in-time first USD-M 1d bars audited directly from Binance Vision before returns.
ONBOARD={
    "BTCUSDT":date(2020,1,1), "ETHUSDT":date(2020,1,1),
    "XRPUSDT":date(2020,1,6), "ADAUSDT":date(2020,1,31),
    "BCHUSDT":date(2020,1,1), "LTCUSDT":date(2020,1,9),
    "DOGEUSDT":date(2020,7,10), "ZECUSDT":date(2020,2,5),
}

jobs=[(SYM[a],mo) for a in ASSETS for mo in DISC_MONTHS]
with ThreadPoolExecutor(max_workers=12) as ex:
    fs=[ex.submit(fetch_month,*j) for j in jobs]
    for i,f in enumerate(as_completed(fs),1):
        f.result()
        if i%40==0 or i==len(fs):
            print("VISION",i,"/",len(fs),flush=True)

price={}
for a in ASSETS:
    sym=SYM[a]; pm={}
    for mo in DISC_MONTHS:
        for d,o,c,qv in read_month(sym,mo):
            pm[d]=(o,c,qv)
    price[a]=pm
    print("PRICE",a,len(pm),min(pm) if pm else None,max(pm) if pm else None,
          "onboard",ONBOARD[sym],flush=True)

supply={a:{} for a in ASSETS}
for r in ROWS:
    a=r.get("asset")
    if a not in supply: continue
    try: v=float(r["SplyCur"])
    except: continue
    if v>0:
        supply[a][date.fromisoformat(r["time"][:10])]=v
def accel_at(mp,d):
    d7=d-timedelta(days=7); d14=d-timedelta(days=14)
    if d not in mp or d7 not in mp or d14 not in mp: return None
    s0,s7,s14=mp[d],mp[d7],mp[d14]
    if min(s0,s7,s14)<=0: return None
    return math.log(s0/s7)-math.log(s7/s14)

events=[]
for a in ASSETS:
    mp=supply[a]; pm=price[a]; sym=SYM[a]
    ds=sorted(d for d in mp if date(2023,1,1)<=d<=date(2024,12,31))
    count=0
    for d in ds:
        prev=d-timedelta(days=1)
        cur_a=accel_at(mp,d); prev_a=accel_at(mp,prev)
        if cur_a is None or prev_a is None: continue
        side=0
        if prev_a<=0<cur_a: side=-1
        elif prev_a>=0>cur_a: side=1
        if side==0: continue

        # frozen production eligibility
        if (d-ONBOARD[sym]).days < 730: continue
        if d not in pm or pm[d][2] < 5_000_000: continue

        entry_day=d+timedelta(days=1)
        end1=entry_day
        end3=entry_day+timedelta(days=2)
        end7=entry_day+timedelta(days=6)
        if end7.year!=d.year: continue
        if any(x not in pm for x in (entry_day,end1,end3,end7)): continue
        entry=pm[entry_day][0]
        if entry<=0: continue
        vals={}
        for h,ed in ((1,end1),(3,end3),(7,end7)):
            close=pm[ed][1]
            if close<=0: break
            vals[h]=side*math.log(close/entry)
        if len(vals)!=3: continue
        events.append({
            "asset":a,"symbol":sym,"signal_date":d.isoformat(),
            "side":"LONG" if side>0 else "SHORT",
            "accel_prev":prev_a,"accel_now":cur_a,
            "supply_growth_now_7d":math.log(mp[d]/mp[d-timedelta(days=7)]),
            "qv_signal_day":pm[d][2],
            "r1":vals[1],"r3":vals[3],"r7":vals[7],
        }); count+=1
    print("EVENTS",a,count,flush=True)
def mean(xs): return sum(xs)/len(xs) if xs else 0.0
def summ(rows):
    if not rows: return {"n":0}
    return {
      "n":len(rows),
      "mean_r1":mean([x["r1"] for x in rows]),
      "mean_r3":mean([x["r3"] for x in rows]),
      "mean_r7":mean([x["r7"] for x in rows]),
      "win_r7":mean([1.0 if x["r7"]>0 else 0.0 for x in rows]),
      "long":sum(1 for x in rows if x["side"]=="LONG"),
      "short":sum(1 for x in rows if x["side"]=="SHORT"),
    }

by_asset=defaultdict(list); by_year=defaultdict(list); by_side=defaultdict(list)
for e in events:
    by_asset[e["asset"]].append(e)
    by_year[e["signal_date"][:4]].append(e)
    by_side[e["side"]].append(e)

triggered=sorted(by_asset)
positive=sum(1 for a in triggered if summ(by_asset[a])["mean_r7"]>0)
weeks=((date(2025,1,1)-date(2023,1,1)).days/7.0)*len(triggered) if triggered else 0
freq=len(events)/weeks if weeks else 0
overall=summ(events)
gate=(overall.get("mean_r7",0)>=0.0025 and triggered and
      positive/len(triggered)>=0.60 and freq>=0.30 and
      summ(by_year.get("2023",[])).get("mean_r7",0)>0 and
      summ(by_year.get("2024",[])).get("mean_r7",0)>0)

summary={
 "status":"promote_oos" if gate else "frozen_failed_discovery",
 "gate_pass":gate,
 "events":len(events),
 "triggered_symbols":len(triggered),
 "positive_symbols":f"{positive}/{len(triggered)}",
 "positive_symbol_fraction":positive/len(triggered) if triggered else 0,
 "frequency_per_symbol_week":freq,
 "overall":overall,
 "by_year":{k:summ(v) for k,v in sorted(by_year.items())},
 "by_side":{k:summ(v) for k,v in sorted(by_side.items())},
 "by_asset":{k:summ(v) for k,v in sorted(by_asset.items())},
 "onboard_dates":{SYM[a]:ONBOARD[SYM[a]].isoformat() for a in ASSETS},
 "oos_2025_plus_evaluated":False,
 "strict_engine_run":False
}
(ROOT/"results/events.json").write_text(json.dumps(events,indent=2))
(ROOT/"results/summary.json").write_text(json.dumps(summary,indent=2))
print(json.dumps({
 "events":len(events),
 "triggered_symbols":len(triggered),
 "positive_symbols":summary["positive_symbols"],
 "frequency_per_symbol_week":freq,
 "mean_r1":overall.get("mean_r1",0),
 "mean_r3":overall.get("mean_r3",0),
 "mean_r7":overall.get("mean_r7",0),
 "2023_r7":summary["by_year"].get("2023",{}).get("mean_r7",0),
 "2024_r7":summary["by_year"].get("2024",{}).get("mean_r7",0),
 "gate_pass":gate
},indent=2))
