import csv,io,zipfile,urllib.request,urllib.error,time,datetime,math,json
from concurrent.futures import ThreadPoolExecutor,as_completed
from pathlib import Path
ROOT=Path("strategy_templates/research/spot-perp-basis-momentum/2026-10-02-early-gate")
SYMS=["SOLUSDT","DOGEUSDT","LTCUSDT","AVAXUSDT","UNIUSDT","ZECUSDT"]
UA="Mozilla/5.0"; CACHE=Path("/tmp/v161_basis_momentum"); CACHE.mkdir(exist_ok=True)
MONTHS=["2022-12"]+[f"{y}-{m:02d}" for y in (2023,2024) for m in range(1,13)]
START=int(datetime.datetime(2023,1,1,tzinfo=datetime.timezone.utc).timestamp()*1000)
END=int(datetime.datetime(2025,1,1,tzinfo=datetime.timezone.utc).timestamp()*1000)
H=3600000

def url(kind,s,m):
    if kind=="spot": return f"https://data.binance.vision/data/spot/monthly/klines/{s}/1h/{s}-1h-{m}.zip"
    return f"https://data.binance.vision/data/futures/um/monthly/klines/{s}/1h/{s}-1h-{m}.zip"
def fetch(kind,s,m):
    p=CACHE/f"{kind}-{s}-{m}.zip"; miss=CACHE/f"{kind}-{s}-{m}.missing"
    if p.exists() or miss.exists(): return
    last=None
    for k in range(5):
        try:
            b=urllib.request.urlopen(urllib.request.Request(url(kind,s,m),headers={"User-Agent":UA}),timeout=25).read()
            zipfile.ZipFile(io.BytesIO(b)); p.write_bytes(b); return
        except urllib.error.HTTPError as e:
            if e.code==404: miss.write_text("404"); return
            last=e
        except Exception as e: last=e
        time.sleep(.5*(k+1))
    raise last
def read(kind,s,m):
    p=CACHE/f"{kind}-{s}-{m}.zip"
    if not p.exists(): return {}
    z=zipfile.ZipFile(p); out={}
    with z.open(z.namelist()[0]) as f:
        for r in csv.reader(io.TextIOWrapper(f)):
            if not r or not r[0].isdigit(): continue
            t=int(r[0]); t=t//1000 if t>10**15 else t
            out[t]=(float(r[1]),float(r[4]))
    return out

jobs=[(k,s,m) for s in SYMS for m in MONTHS for k in ("spot","fut")]
with ThreadPoolExecutor(max_workers=12) as ex:
    fs={ex.submit(fetch,*j):j for j in jobs}
    for i,f in enumerate(as_completed(fs),1):
        f.result()
        if i%50==0: print("DOWNLOAD",i,"/",len(jobs),flush=True)

events=[]
source={}
for s in SYMS:
    sp={}; fu={}
    for m in MONTHS:
        sp.update(read("spot",s,m)); fu.update(read("fut",s,m))
    common=sorted(set(sp)&set(fu))
    basis={t:math.log(fu[t][1]/sp[t][1]) for t in common if sp[t][1]>0 and fu[t][1]>0}
    source[s]={"spot_bars":len(sp),"fut_bars":len(fu),"aligned":len(basis)}
    for t in common:
        if not (START<=t<END): continue
        if t-H not in basis or t-24*H not in basis or t-25*H not in basis: continue
        prev=basis[t-H]-basis[t-25*H]
        now=basis[t]-basis[t-24*H]
        side=1 if prev<=0<now else (-1 if prev>=0>now else 0)
        if not side: continue
        if t+12*H not in fu or t+H not in fu: continue
        y=datetime.datetime.fromtimestamp(t/1000,datetime.timezone.utc).year
        if datetime.datetime.fromtimestamp((t+12*H)/1000,datetime.timezone.utc).year!=y: continue
        entry=fu[t+H][0]
        if entry<=0: continue
        rs={}
        ok=True
        for h in (1,4,12):
            tt=t+h*H
            if tt not in fu or fu[tt][1]<=0: ok=False; break
            rs[h]=side*math.log(fu[tt][1]/entry)
        if not ok: continue
        events.append({"symbol":s,"signal_time":t,"year":y,"side":"LONG" if side>0 else "SHORT",
                       "momentum_prev":prev,"momentum_now":now,"r1":rs[1],"r4":rs[4],"r12":rs[12]})
    print("SYMBOL",s,"events",sum(e["symbol"]==s for e in events),flush=True)

def mean(v): return sum(v)/len(v) if v else 0.0
def sm(rows):
    return {"n":len(rows),"mean_r1":mean([x["r1"] for x in rows]),"mean_r4":mean([x["r4"] for x in rows]),
            "mean_r12":mean([x["r12"] for x in rows]),"win_r12":mean([1.0 if x["r12"]>0 else 0.0 for x in rows]),
            "long":sum(x["side"]=="LONG" for x in rows),"short":sum(x["side"]=="SHORT" for x in rows)}
by_sym={s:sm([e for e in events if e["symbol"]==s]) for s in SYMS}
by_year={str(y):sm([e for e in events if e["year"]==y]) for y in (2023,2024)}
by_side={q:sm([e for e in events if e["side"]==q]) for q in ("LONG","SHORT")}
overall=sm(events)
positive=sum(by_sym[s]["mean_r12"]>0 for s in SYMS)
weeks=((datetime.date(2025,1,1)-datetime.date(2023,1,1)).days/7)*len(SYMS)
freq=len(events)/weeks
gate=(overall["mean_r12"]>=.002 and positive>=4 and freq>=.30 and by_year["2023"]["mean_r12"]>0 and by_year["2024"]["mean_r12"]>0)
summary={"version":"v161","status":"promote_oos" if gate else "frozen_failed_early_gate","gate_pass":gate,
         "events":len(events),"positive_symbols":f"{positive}/6","frequency_per_symbol_week":freq,
         "overall":overall,"by_symbol":by_sym,"by_year":by_year,"by_side":by_side,"source_meta":source,
         "oos_evaluated":False,"strict_engine_run":False}
(ROOT/"results/events.json").write_text(json.dumps(events,indent=2))
(ROOT/"results/summary.json").write_text(json.dumps(summary,indent=2))
print(json.dumps({"events":len(events),"mean_r12":overall["mean_r12"],"positive_symbols":summary["positive_symbols"],
                  "freq":freq,"2023":by_year["2023"]["mean_r12"],"2024":by_year["2024"]["mean_r12"],
                  "gate_pass":gate},indent=2),flush=True)
