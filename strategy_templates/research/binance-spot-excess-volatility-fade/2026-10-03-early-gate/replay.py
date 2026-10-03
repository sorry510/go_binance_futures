import csv,datetime,io,json,math,time,urllib.error,urllib.request,zipfile
from pathlib import Path

ROOT=Path("strategy_templates/research/binance-spot-excess-volatility-fade/2026-10-03-early-gate")
SYMS=["SOLUSDT","DOGEUSDT","LTCUSDT","AVAXUSDT","UNIUSDT","ZECUSDT"]
SPOT_CACHE=Path("/tmp/v161_basis_momentum")
INDEX_CACHE=Path("/tmp/v169-index"); INDEX_CACHE.mkdir(exist_ok=True)
MONTHS=["2022-12"]+[f"{y}-{m:02d}" for y in (2023,2024) for m in range(1,13)]
BASE="https://data.binance.vision/data/futures/um/monthly/indexPriceKlines"
START=int(datetime.datetime(2023,1,1,tzinfo=datetime.timezone.utc).timestamp()*1000)
END=int(datetime.datetime(2025,1,1,tzinfo=datetime.timezone.utc).timestamp()*1000)
H=3600000

def ensure_index(sym,m):
    p=INDEX_CACHE/f"{sym}-1h-{m}.zip"
    if p.exists(): return p
    u=f"{BASE}/{sym}/1h/{sym}-1h-{m}.zip"; last=None
    for i in range(5):
        try:
            req=urllib.request.Request(u,headers={"User-Agent":"Mozilla/5.0"})
            data=urllib.request.urlopen(req,timeout=25).read()
            zipfile.ZipFile(io.BytesIO(data))
            p.write_bytes(data); return p
        except Exception as e:
            last=e; time.sleep(.4*(i+1))
    raise last

def read_zip(p):
    z=zipfile.ZipFile(p);out={}
    with z.open(z.namelist()[0]) as f:
        for r in csv.reader(io.TextIOWrapper(f)):
            if not r or not r[0].isdigit() or len(r)<5: continue
            t=int(r[0]);t=t//1000 if t>10**15 else t
            out[t]={"open":float(r[1]),"close":float(r[4])}
    return out

def rv(rows,times,a,b):
    s=0.0
    for k in range(a+1,b+1):
        x0=rows[times[k-1]]["close"];x1=rows[times[k]]["close"]
        if x0<=0 or x1<=0:return 0.0
        q=math.log(x1/x0);s+=q*q
    return math.sqrt(s)

def mean(xs):return sum(xs)/len(xs) if xs else 0.0
def sm(rows):
    return {"n":len(rows),"mean_r1":mean([x["r1"] for x in rows]),"mean_r4":mean([x["r4"] for x in rows]),
            "mean_r12":mean([x["r12"] for x in rows]),"win_r12":mean([1.0 if x["r12"]>0 else 0.0 for x in rows]),
            "long":sum(x["side"]=="LONG" for x in rows),"short":sum(x["side"]=="SHORT" for x in rows)}

events=[];source={}
for sym in SYMS:
    sp={};idx={};fu={}
    for m in MONTHS:
        spath=SPOT_CACHE/f"spot-{sym}-{m}.zip";fpath=SPOT_CACHE/f"fut-{sym}-{m}.zip"
        if spath.exists():sp.update(read_zip(spath))
        if fpath.exists():fu.update(read_zip(fpath))
        idx.update(read_zip(ensure_index(sym,m)))
    common=sorted(set(sp)&set(idx))
    source[sym]={"spot_bars":len(sp),"index_bars":len(idx),"fut_bars":len(fu),"aligned":len(common)}
    n=0
    for j in range(25,len(common)-12):
        t=common[j]
        if not(START<=t<END):continue
        if common[j]-common[j-25]!=25*H:continue
        rvs=rv(sp,common,j-24,j);rvi=rv(idx,common,j-24,j)
        pvs=rv(sp,common,j-25,j-1);pvi=rv(idx,common,j-25,j-1)
        if min(rvs,rvi,pvs,pvi)<=0:continue
        now=math.log(rvs/rvi);prev=math.log(pvs/pvi)
        if not(prev<=0<now):continue
        t24=t-24*H
        if t24 not in sp or t not in sp or t+12*H not in fu:continue
        trend=math.log(sp[t]["close"]/sp[t24]["close"])
        if trend==0:continue
        side=-1.0 if trend>0 else 1.0
        y=datetime.datetime.fromtimestamp(t/1000,datetime.timezone.utc).year
        if datetime.datetime.fromtimestamp((t+12*H)/1000,datetime.timezone.utc).year!=y:continue
        if t+H not in fu:continue
        entry=fu[t+H]["open"];vals={};ok=True
        for h in (1,4,12):
            tt=t+h*H
            if tt not in fu or fu[tt]["close"]<=0:ok=False;break
            vals[h]=side*math.log(fu[tt]["close"]/entry)
        if not ok:continue
        events.append({"symbol":sym,"signal_time":t,"year":y,"side":"LONG" if side>0 else "SHORT",
                       "score_prev":prev,"score_now":now,"rv_spot":rvs,"rv_index":rvi,"ret24_spot":trend,
                       "r1":vals[1],"r4":vals[4],"r12":vals[12]});n+=1
    print("SYMBOL",sym,"events",n,flush=True)

by_sym={s:sm([e for e in events if e["symbol"]==s]) for s in SYMS}
by_year={str(y):sm([e for e in events if e["year"]==y]) for y in (2023,2024)}
by_side={q:sm([e for e in events if e["side"]==q]) for q in ("LONG","SHORT")}
overall=sm(events);positive=sum(by_sym[s]["mean_r12"]>0 for s in SYMS)
weeks=((datetime.date(2025,1,1)-datetime.date(2023,1,1)).days/7)*len(SYMS);freq=len(events)/weeks
gate=(overall["mean_r12"]>=.002 and positive>=4 and freq>=.30 and by_year["2023"]["mean_r12"]>0 and by_year["2024"]["mean_r12"]>0)
summary={"version":"v169","status":"promote_oos" if gate else "frozen_failed_early_gate","gate_pass":gate,
         "events":len(events),"positive_symbols":f"{positive}/6","frequency_per_symbol_week":freq,
         "overall":overall,"by_symbol":by_sym,"by_year":by_year,"by_side":by_side,"source_meta":source,
         "oos_evaluated":False,"strict_engine_run":False}
(ROOT/"results/events.json").write_text(json.dumps(events,indent=2))
(ROOT/"results/summary.json").write_text(json.dumps(summary,indent=2))
print(json.dumps({"events":len(events),"mean_r1":overall["mean_r1"],"mean_r4":overall["mean_r4"],"mean_r12":overall["mean_r12"],
                  "positive_symbols":summary["positive_symbols"],"freq":freq,"2023":by_year["2023"]["mean_r12"],
                  "2024":by_year["2024"]["mean_r12"],"long_r12":by_side["LONG"]["mean_r12"],
                  "short_r12":by_side["SHORT"]["mean_r12"],"gate_pass":gate},indent=2),flush=True)
