#!/usr/bin/env python3
import csv,io,json,math,time,urllib.request,zipfile
from pathlib import Path
from datetime import datetime,timezone,date,timedelta
from concurrent.futures import ThreadPoolExecutor,as_completed
ROOT=Path(__file__).resolve().parent
CACHE=ROOT/"inputs"/"metrics_2025"; CACHE.mkdir(parents=True,exist_ok=True)
OUT=ROOT/"results"; OUT.mkdir(parents=True,exist_ok=True)
SYMS=["BTCUSDT","ETHUSDT","BNBUSDT","XRPUSDT"]
D0=date(2025,1,1); D1=date(2025,12,31)
DAYS=[]; d=D0
while d<=D1: DAYS.append(d.isoformat()); d+=timedelta(days=1)
def fetch(task):
    sym,day=task; p=CACHE/f"{sym}-metrics-{day}.csv"
    if p.exists(): return str(p)
    u=f"https://data.binance.vision/data/futures/um/daily/metrics/{sym}/{sym}-metrics-{day}.zip"
    for k in range(4):
        try:
            req=urllib.request.Request(u,headers={"User-Agent":"Mozilla/5.0"})
            with urllib.request.urlopen(req,timeout=25) as r:b=r.read()
            z=zipfile.ZipFile(io.BytesIO(b)); p.write_bytes(z.read(z.namelist()[0])); return str(p)
        except Exception:
            if k==3:return None
            time.sleep(0.5*(k+1))
def parse_ts(s): return datetime.strptime(s,"%Y-%m-%d %H:%M:%S").replace(tzinfo=timezone.utc).timestamp()
tasks=[(s,d) for s in SYMS for d in DAYS]
with ThreadPoolExecutor(max_workers=24) as ex:
    fut=[ex.submit(fetch,t) for t in tasks]
    done=0
    for f in as_completed(fut):
        done+=1
        if done%200==0: print("download",done,"/",len(tasks),flush=True)
def load(sym):
    rows=[]
    for day in DAYS:
        p=CACHE/f"{sym}-metrics-{day}.csv"
        if not p.exists(): continue
        with p.open() as f:
            for r in csv.DictReader(f):
                try:
                    oi=float(r["sum_open_interest"]); val=float(r["sum_open_interest_value"])
                    if oi>0 and val>0: rows.append((parse_ts(r["create_time"]),oi,val/oi))
                except: pass
    dd={t:(oi,px) for t,oi,px in rows}; return [(t,*dd[t]) for t in sorted(dd)]
def leq(rows,target,maxgap=1800):
    lo,hi=0,len(rows)
    while lo<hi:
        m=(lo+hi)//2
        if rows[m][0]<=target:lo=m+1
        else:hi=m
    i=lo-1
    return rows[i] if i>=0 and target-rows[i][0]<=maxgap else None
def geq(rows,target,maxgap=1800):
    lo,hi=0,len(rows)
    while lo<hi:
        m=(lo+hi)//2
        if rows[m][0]<target:lo=m+1
        else:hi=m
    return rows[lo] if lo<len(rows) and rows[lo][0]-target<=maxgap else None
events=[]
for sym in SYMS:
    rows=load(sym); prev=None; n=0
    for t,oi,px in rows:
        b=leq(rows,t-4*3600)
        if not b: continue
        oi4=math.log(oi/b[1]); pr4=math.log(px/b[2])
        if prev is not None and oi4<0 and prev>=0 and pr4!=0:
            direction=-1 if pr4>0 else 1
            rec={"symbol":sym,"time":datetime.fromtimestamp(t,timezone.utc).isoformat(),"oi4":oi4,"price4":pr4}
            ok=True
            for h in (1,4,12):
                q=geq(rows,t+h*3600)
                if not q:ok=False;break
                rec[f"fwd{h}h"]=direction*math.log(q[2]/px)
            if ok:events.append(rec);n+=1
        prev=oi4
    print(sym,"events",n,flush=True)
summ={"n":len(events),"by_symbol":{}}
for s in SYMS:
    q=[e for e in events if e["symbol"]==s]; z={"n":len(q)}
    for h in (1,4,12):
        v=[e[f"fwd{h}h"] for e in q]
        z[f"mean_{h}h"]=sum(v)/len(v) if v else None
        z[f"win_{h}h"]=sum(x>0 for x in v)/len(v) if v else None
    summ["by_symbol"][s]=z
for h in (1,4,12):
    v=[e[f"fwd{h}h"] for e in events]
    summ[f"mean_{h}h"]=sum(v)/len(v) if v else None
    summ[f"win_{h}h"]=sum(x>0 for x in v)/len(v) if v else None
summ["positive_symbols_4h"]=sum(1 for s in SYMS if summ["by_symbol"][s]["mean_4h"] and summ["by_symbol"][s]["mean_4h"]>0)
summ["passes_stage2"]=bool(summ["mean_4h"] is not None and summ["mean_4h"]>=0.001 and summ["positive_symbols_4h"]>=3)
(OUT/"summary_2025.json").write_text(json.dumps(summ,indent=2))
if events:
    with (OUT/"events_2025.csv").open("w",newline="") as f:
        w=csv.DictWriter(f,fieldnames=list(events[0]));w.writeheader();w.writerows(events)
print(json.dumps(summ,indent=2))
