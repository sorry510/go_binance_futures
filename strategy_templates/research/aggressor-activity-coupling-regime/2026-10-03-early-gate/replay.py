import csv,datetime,io,json,math,zipfile
from pathlib import Path

ROOT=Path("strategy_templates/research/aggressor-activity-coupling-regime/2026-10-03-early-gate")
SYMS=["SOLUSDT","DOGEUSDT","LTCUSDT","AVAXUSDT","UNIUSDT","ZECUSDT"]
CACHE=Path("/tmp/v161_basis_momentum")
MONTHS=["2022-12"]+[f"{y}-{m:02d}" for y in (2023,2024) for m in range(1,13)]
UTC=datetime.timezone.utc
H=3600000
START=int(datetime.datetime(2023,1,1,tzinfo=UTC).timestamp()*1000)
END=int(datetime.datetime(2025,1,1,tzinfo=UTC).timestamp()*1000)

def read(sym):
    out={}
    for m in MONTHS:
        p=CACHE/f"fut-{sym}-{m}.zip"
        if not p.exists(): continue
        z=zipfile.ZipFile(p)
        with z.open(z.namelist()[0]) as f:
            for r in csv.reader(io.TextIOWrapper(f)):
                if not r or not r[0].isdigit() or len(r)<11: continue
                t=int(r[0]); t=t//1000 if t>10**15 else t
                qv=float(r[7]); tbq=float(r[10])
                if qv<=0: continue
                out[t]={
                    "open":float(r[1]),
                    "close":float(r[4]),
                    "qv":qv,
                    "flow":2.0*tbq/qv-1.0,
                }
    return out

def corr(a,b):
    ma=sum(a)/len(a); mb=sum(b)/len(b)
    va=sum((x-ma)*(x-ma) for x in a)
    vb=sum((x-mb)*(x-mb) for x in b)
    if va<=0 or vb<=0: return None
    cov=sum((x-ma)*(y-mb) for x,y in zip(a,b))
    return cov/math.sqrt(va*vb)

def mean(xs): return sum(xs)/len(xs) if xs else 0.0

def sm(rows):
    return {
        "n":len(rows),
        "mean_r1":mean([x["r1"] for x in rows]),
        "mean_r4":mean([x["r4"] for x in rows]),
        "mean_r12":mean([x["r12"] for x in rows]),
        "win_r12":mean([1.0 if x["r12"]>0 else 0.0 for x in rows]),
        "long":sum(x["side"]=="LONG" for x in rows),
        "short":sum(x["side"]=="SHORT" for x in rows),
    }

events=[]; source={}
for sym in SYMS:
    bars=read(sym); ts=sorted(bars)
    dlogq={}
    for i in range(1,len(ts)):
        if ts[i]-ts[i-1]==H:
            q0=bars[ts[i-1]]["qv"]; q1=bars[ts[i]]["qv"]
            if q0>0 and q1>0:
                dlogq[ts[i]]=math.log(q1/q0)
    source[sym]={"bars":len(ts),"activity_changes":len(dlogq)}
    n=0
    for j in range(49,len(ts)-12):
        t=ts[j]
        if not (START<=t<END): continue
        # Need previous/current 48h windows and 12h forward path all continuous.
        if t-ts[j-49] != 49*H or ts[j+12]-t != 12*H: continue

        curr_f=[]; curr_a=[]; prev_f=[]; prev_a=[]; ok=True
        # current window uses hours j-47..j, each activity change needs prior hour.
        for k in range(j-47,j+1):
            a=dlogq.get(ts[k])
            if a is None: ok=False; break
            curr_f.append(bars[ts[k]]["flow"]); curr_a.append(a)
        if not ok: continue
        for k in range(j-48,j):
            a=dlogq.get(ts[k])
            if a is None: ok=False; break
            prev_f.append(bars[ts[k]]["flow"]); prev_a.append(a)
        if not ok: continue

        now=corr(curr_f,curr_a); prev=corr(prev_f,prev_a)
        if now is None or prev is None: continue
        if prev<=0<now:
            side=1.0
        elif prev>=0>now:
            side=-1.0
        else:
            continue

        y=datetime.datetime.fromtimestamp(t/1000,UTC).year
        if datetime.datetime.fromtimestamp((t+12*H)/1000,UTC).year!=y: continue
        if t+H not in bars: continue
        entry=bars[t+H]["open"]
        vals={}; good=True
        for h in (1,4,12):
            tt=t+h*H
            if tt not in bars or bars[tt]["close"]<=0:
                good=False; break
            vals[h]=side*math.log(bars[tt]["close"]/entry)
        if not good: continue

        events.append({
            "symbol":sym,"signal_time":t,"year":y,
            "side":"LONG" if side>0 else "SHORT",
            "score_prev":prev,"score_now":now,
            "r1":vals[1],"r4":vals[4],"r12":vals[12],
        })
        n+=1
    print("SYMBOL",sym,"events",n,flush=True)

by_sym={s:sm([e for e in events if e["symbol"]==s]) for s in SYMS}
by_year={str(y):sm([e for e in events if e["year"]==y]) for y in (2023,2024)}
by_side={q:sm([e for e in events if e["side"]==q]) for q in ("LONG","SHORT")}
overall=sm(events)
positive=sum(by_sym[s]["mean_r12"]>0 for s in SYMS)
weeks=((datetime.date(2025,1,1)-datetime.date(2023,1,1)).days/7)*len(SYMS)
freq=len(events)/weeks
gate=(overall["mean_r12"]>=.002 and positive>=4 and freq>=.30 and
      by_year["2023"]["mean_r12"]>0 and by_year["2024"]["mean_r12"]>0)

summary={
    "version":"v176",
    "status":"promote_oos" if gate else "frozen_failed_early_gate",
    "gate_pass":gate,
    "events":len(events),
    "positive_symbols":f"{positive}/6",
    "frequency_per_symbol_week":freq,
    "overall":overall,
    "by_symbol":by_sym,
    "by_year":by_year,
    "by_side":by_side,
    "source_meta":source,
    "oos_evaluated":False,
    "strict_engine_run":False,
}
(ROOT/"results/events.json").write_text(json.dumps(events,indent=2))
(ROOT/"results/summary.json").write_text(json.dumps(summary,indent=2))
print(json.dumps({
    "events":len(events),
    "mean_r1":overall["mean_r1"],
    "mean_r4":overall["mean_r4"],
    "mean_r12":overall["mean_r12"],
    "positive_symbols":summary["positive_symbols"],
    "freq":freq,
    "2023":by_year["2023"]["mean_r12"],
    "2024":by_year["2024"]["mean_r12"],
    "long_r12":by_side["LONG"]["mean_r12"],
    "short_r12":by_side["SHORT"]["mean_r12"],
    "gate_pass":gate,
},indent=2),flush=True)
