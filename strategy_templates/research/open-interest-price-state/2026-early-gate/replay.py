#!/usr/bin/env python3
import csv, io, json, math, urllib.request, zipfile
from pathlib import Path
from datetime import datetime, timezone

ROOT=Path(__file__).resolve().parent
CFG=json.loads((ROOT/"config.json").read_text())
CACHE=ROOT/"inputs"/"metrics"; CACHE.mkdir(parents=True, exist_ok=True)
RESULTS=ROOT/"results"; RESULTS.mkdir(parents=True, exist_ok=True)

def fetch(sym, day):
    p=CACHE/f"{sym}-metrics-{day}.csv"
    if p.exists(): return p
    url=f"https://data.binance.vision/data/futures/um/daily/metrics/{sym}/{sym}-metrics-{day}.zip"
    req=urllib.request.Request(url, headers={"User-Agent":"Mozilla/5.0"})
    with urllib.request.urlopen(req,timeout=30) as r: b=r.read()
    z=zipfile.ZipFile(io.BytesIO(b)); name=z.namelist()[0]
    p.write_bytes(z.read(name)); return p

def ts(s):
    return datetime.strptime(s,"%Y-%m-%d %H:%M:%S").replace(tzinfo=timezone.utc).timestamp()

def load(sym):
    rows=[]
    for day in CFG["sample_days"]:
        p=fetch(sym,day)
        with p.open() as f:
            for r in csv.DictReader(f):
                oi=float(r["sum_open_interest"]); val=float(r["sum_open_interest_value"])
                if oi>0 and val>0: rows.append((ts(r["create_time"]),oi,val/oi))
    d={t:(oi,px) for t,oi,px in rows}
    return [(t,*d[t]) for t in sorted(d)]

def near_leq(rows,target,maxgap=1800):
    lo,hi=0,len(rows)
    while lo<hi:
        m=(lo+hi)//2
        if rows[m][0] <= target: lo=m+1
        else: hi=m
    i=lo-1
    if i<0 or target-rows[i][0]>maxgap:return None
    return rows[i]

def near_geq(rows,target,maxgap=1800):
    lo,hi=0,len(rows)
    while lo<hi:
        m=(lo+hi)//2
        if rows[m][0] < target: lo=m+1
        else: hi=m
    if lo>=len(rows) or rows[lo][0]-target>maxgap:return None
    return rows[lo]

def analyze(sym):
    rows=load(sym); out=[]; prev_oi4=None
    for t,oi,px in rows:
        b=near_leq(rows,t-4*3600)
        if not b: continue
        _,oi0,px0=b
        oi4=math.log(oi/oi0); pr4=math.log(px/px0)
        if prev_oi4 is None:
            prev_oi4=oi4; continue
        fam=None; direction=0
        if oi4>0 and prev_oi4<=0 and pr4!=0:
            fam="oi_expansion_continuation"; direction=1 if pr4>0 else -1
        elif oi4<0 and prev_oi4>=0 and pr4!=0:
            fam="oi_contraction_reversal"; direction=-1 if pr4>0 else 1
        prev_oi4=oi4
        if not fam: continue
        rec={"symbol":sym,"time":datetime.fromtimestamp(t,timezone.utc).isoformat(),"family":fam,"oi4":oi4,"price4":pr4}
        ok=True
        for h in CFG["future_horizons_hours"]:
            f=near_geq(rows,t+h*3600)
            if not f: ok=False; break
            rec[f"fwd{h}h"]=direction*math.log(f[2]/px)
        if ok: out.append(rec)
    return out

allr=[]
for s in CFG["symbols"]:
    try:
        x=analyze(s); allr+=x; print(s,len(x))
    except Exception as e:
        print("ERROR",s,repr(e))

if allr:
    with (RESULTS/"events.csv").open("w",newline="") as f:
        w=csv.DictWriter(f,fieldnames=list(allr[0])); w.writeheader(); w.writerows(allr)

summary={}
for fam in CFG["signals"]:
    rr=[r for r in allr if r["family"]==fam]
    by={}
    for s in CFG["symbols"]:
        q=[r for r in rr if r["symbol"]==s]
        by[s]={"n":len(q)}
        for h in CFG["future_horizons_hours"]:
            vals=[r[f"fwd{h}h"] for r in q]
            by[s][f"mean_{h}h"]=sum(vals)/len(vals) if vals else None
            by[s][f"win_{h}h"]=sum(v>0 for v in vals)/len(vals) if vals else None
    z={"n":len(rr),"by_symbol":by}
    for h in CFG["future_horizons_hours"]:
        vals=[r[f"fwd{h}h"] for r in rr]
        z[f"mean_{h}h"]=sum(vals)/len(vals) if vals else None
        z[f"win_{h}h"]=sum(v>0 for v in vals)/len(vals) if vals else None
    positives=sum(1 for s in CFG["symbols"] if by[s].get("mean_4h") is not None and by[s]["mean_4h"]>0)
    z["positive_symbols_4h"]=positives
    z["passes_early_gate"]=bool(z.get("mean_4h") is not None and z["mean_4h"]>=0.001 and positives>=3)
    summary[fam]=z
(RESULTS/"summary.json").write_text(json.dumps(summary,indent=2))
print(json.dumps(summary,indent=2))
