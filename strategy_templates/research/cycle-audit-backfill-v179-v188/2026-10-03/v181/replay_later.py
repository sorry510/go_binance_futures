import csv,datetime,io,json,math,zipfile
from pathlib import Path

ROOT=Path("strategy_templates/research/cycle-audit-backfill-v179-v188/2026-10-03/v181")
SYMS=["SOLUSDT","DOGEUSDT","LTCUSDT","AVAXUSDT","UNIUSDT","ZECUSDT"]
CACHE=Path("/tmp/v179_v188_1h")
MONTHS=["2024-12"]+[f"2025-{m:02d}" for m in range(1,13)]+[f"2026-{m:02d}" for m in range(1,10)]
UTC=datetime.timezone.utc; H=3600000
START=int(datetime.datetime(2025,1,1,tzinfo=UTC).timestamp()*1000)
END=int(datetime.datetime(2026,10,1,tzinfo=UTC).timestamp()*1000)

def read(sym):
    out={}
    for m in MONTHS:
        p=CACHE/f"fut-{sym}-{m}.zip"
        if not p.exists():continue
        z=zipfile.ZipFile(p)
        with z.open(z.namelist()[0]) as f:
            for r in csv.reader(io.TextIOWrapper(f)):
                if not r or not r[0].isdigit() or len(r)<8:continue
                t=int(r[0]);t=t//1000 if t>10**15 else t
                q=float(r[7])
                if q<=0:continue
                out[t]={"open":float(r[1]),"close":float(r[4]),"qv":q}
    return out

def corr(a,b):
    ma=sum(a)/len(a);mb=sum(b)/len(b)
    va=sum((x-ma)**2 for x in a);vb=sum((x-mb)**2 for x in b)
    if va<=0 or vb<=0:return None
    return sum((x-ma)*(y-mb) for x,y in zip(a,b))/math.sqrt(va*vb)

def mean(xs):return sum(xs)/len(xs) if xs else 0.0
def sm(rows):
    return {"n":len(rows),"mean_r1":mean([x["r1"] for x in rows]),"mean_r4":mean([x["r4"] for x in rows]),
            "mean_r12":mean([x["r12"] for x in rows]),"win_r12":mean([1.0 if x["r12"]>0 else 0.0 for x in rows]),
            "long":sum(x["side"]=="LONG" for x in rows),"short":sum(x["side"]=="SHORT" for x in rows)}

events=[];source={}
for sym in SYMS:
    b=read(sym);ts=sorted(b);act={};vol={}
    for i in range(1,len(ts)):
        if ts[i]-ts[i-1]!=H:continue
        p0=b[ts[i-1]]["close"];p1=b[ts[i]]["close"]
        if p0<=0 or p1<=0:continue
        act[ts[i]]=math.log(b[ts[i]]["qv"]/b[ts[i-1]]["qv"])
        r=math.log(p1/p0);vol[ts[i]]=r*r
    source[sym]={"bars":len(ts),"activity_changes":len(act)}
    n=0
    for j in range(49,len(ts)-12):
        t=ts[j]
        if not(START<=t<END):continue
        if t-ts[j-49]!=49*H or ts[j+12]-t!=12*H:continue

        ca=[];cv=[];pa=[];pv=[];ok=True
        for k in range(j-47,j+1):
            tt=ts[k]
            if tt not in act or tt not in vol:ok=False;break
            ca.append(act[tt]);cv.append(vol[tt])
        if not ok:continue
        for k in range(j-48,j):
            tt=ts[k]
            if tt not in act or tt not in vol:ok=False;break
            pa.append(act[tt]);pv.append(vol[tt])
        if not ok:continue
        now=corr(ca,cv);prev=corr(pa,pv)
        if now is None or prev is None:continue

        t4=t-4*H
        if t4 not in b:continue
        trend=math.log(b[t]["close"]/b[t4]["close"])
        if trend==0:continue
        trend_side=1.0 if trend>0 else -1.0
        if prev<=0<now:side=trend_side
        elif prev>=0>now:side=-trend_side
        else:continue

        y=datetime.datetime.fromtimestamp(t/1000,UTC).year
        if datetime.datetime.fromtimestamp((t+12*H)/1000,UTC).year!=y:continue
        entry=b[t+H]["open"];vals={};good=True
        for h in (1,4,12):
            tt=t+h*H
            if tt not in b or b[tt]["close"]<=0:good=False;break
            vals[h]=side*math.log(b[tt]["close"]/entry)
        if not good:continue
        events.append({"symbol":sym,"signal_time":t,"year":y,"side":"LONG" if side>0 else "SHORT",
                       "score_prev":prev,"score_now":now,"ret4_trailing":trend,
                       "r1":vals[1],"r4":vals[4],"r12":vals[12]});n+=1
    print("SYMBOL",sym,"events",n,flush=True)

by_sym={s:sm([e for e in events if e["symbol"]==s]) for s in SYMS}
by_year={str(y):sm([e for e in events if e["year"]==y]) for y in (2023,2024,2025,2026)}
by_side={q:sm([e for e in events if e["side"]==q]) for q in ("LONG","SHORT")}
overall=sm(events);positive=sum(by_sym[s]["mean_r12"]>0 for s in SYMS)
weeks=((datetime.date(2026,10,1)-datetime.date(2025,1,1)).days/7)*len(SYMS);freq=len(events)/weeks
gate=(overall["mean_r12"]>=.002 and positive>=4 and freq>=.30 and by_year["2023"]["mean_r12"]>0 and by_year["2024"]["mean_r12"]>0)
summary={"version":"v181","status":"promote_oos" if gate else "frozen_failed_early_gate","gate_pass":gate,
         "events":len(events),"positive_symbols":f"{positive}/6","frequency_per_symbol_week":freq,
         "overall":overall,"by_symbol":by_sym,"by_year":by_year,"by_side":by_side,
         "source_meta":source,"oos_evaluated":False,"strict_engine_run":False}
(ROOT/"results/events.json").write_text(json.dumps(events,indent=2))
(ROOT/"results/summary.json").write_text(json.dumps(summary,indent=2))
print(json.dumps({"events":len(events),"mean_r1":overall["mean_r1"],"mean_r4":overall["mean_r4"],
                  "mean_r12":overall["mean_r12"],"positive_symbols":summary["positive_symbols"],
                  "freq":freq,"2023":by_year["2023"]["mean_r12"],"2024":by_year["2024"]["mean_r12"],
                  "long_r12":by_side["LONG"]["mean_r12"],"short_r12":by_side["SHORT"]["mean_r12"],
                  "gate_pass":gate},indent=2),flush=True)
