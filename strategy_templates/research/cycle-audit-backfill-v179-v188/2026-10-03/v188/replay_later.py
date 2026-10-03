import csv,datetime,io,json,math,zipfile
from pathlib import Path

ROOT=Path("strategy_templates/research/cycle-audit-backfill-v179-v188/2026-10-03/v188")
SYMS=["SOLUSDT","DOGEUSDT","LTCUSDT","AVAXUSDT","UNIUSDT","ZECUSDT"]
CACHE=Path("/tmp/v179_v188_1h")
MONTHS=["2024-12"]+[f"2025-{m:02d}" for m in range(1,13)]+[f"2026-{m:02d}" for m in range(1,10)]
UTC=datetime.timezone.utc
H=3600000
START=int(datetime.datetime(2025,1,1,tzinfo=UTC).timestamp()*1000)
END=int(datetime.datetime(2026,10,1,tzinfo=UTC).timestamp()*1000)
DEN=3.0-2.0*math.sqrt(2.0)

def read(sym):
    out={}
    for m in MONTHS:
        p=CACHE/f"fut-{sym}-{m}.zip"
        if not p.exists(): continue
        z=zipfile.ZipFile(p)
        with z.open(z.namelist()[0]) as f:
            for r in csv.reader(io.TextIOWrapper(f)):
                if not r or not r[0].isdigit() or len(r)<5: continue
                t=int(r[0]); t=t//1000 if t>10**15 else t
                o,h,l,c=map(float,r[1:5])
                if min(o,h,l,c)<=0: continue
                out[t]={"open":o,"high":h,"low":l,"close":c}
    return out

def cs(a,b):
    beta=math.log(a["high"]/a["low"])**2 + math.log(b["high"]/b["low"])**2
    hh=max(a["high"],b["high"]); ll=min(a["low"],b["low"])
    gamma=math.log(hh/ll)**2
    if beta<0 or gamma<0: return None
    alpha=(math.sqrt(2.0*beta)-math.sqrt(beta))/DEN - math.sqrt(gamma/DEN)
    if alpha<=0: return 0.0
    ea=math.exp(min(alpha,50.0))
    return 2.0*(ea-1.0)/(1.0+ea)

def block_mean(bars,ts,a,b):
    # inclusive bar-index block a..b; pair estimates wholly within the block.
    vals=[]
    for k in range(a+1,b+1):
        x=cs(bars[ts[k-1]],bars[ts[k]])
        if x is None: return None
        vals.append(x)
    return sum(vals)/len(vals) if vals else None

def mean(xs): return sum(xs)/len(xs) if xs else 0.0
def sm(rows):
    return {"n":len(rows),
            "mean_r1":mean([x["r1"] for x in rows]),
            "mean_r4":mean([x["r4"] for x in rows]),
            "mean_r12":mean([x["r12"] for x in rows]),
            "win_r12":mean([1.0 if x["r12"]>0 else 0.0 for x in rows]),
            "long":sum(x["side"]=="LONG" for x in rows),
            "short":sum(x["side"]=="SHORT" for x in rows)}

events=[]; source={}
for sym in SYMS:
    bars=read(sym); ts=sorted(bars); source[sym]={"bars":len(ts)}
    n=0
    for j in range(48,len(ts)-12):
        t=ts[j]
        if not (START<=t<END): continue
        if t-ts[j-48] != 48*H or ts[j+12]-t != 12*H: continue

        cur=block_mean(bars,ts,j-23,j)
        base=block_mean(bars,ts,j-47,j-24)
        pcur=block_mean(bars,ts,j-24,j-1)
        pbase=block_mean(bars,ts,j-48,j-25)
        if None in (cur,base,pcur,pbase) or min(cur,base,pcur,pbase)<=0: continue
        now=math.log(cur/base); prev=math.log(pcur/pbase)
        if not (prev<=0<now): continue

        t4=t-4*H
        if t4 not in bars: continue
        trend=math.log(bars[t]["close"]/bars[t4]["close"])
        if trend==0: continue
        side=-1.0 if trend>0 else 1.0

        y=datetime.datetime.fromtimestamp(t/1000,UTC).year
        if datetime.datetime.fromtimestamp((t+12*H)/1000,UTC).year!=y: continue
        if t+H not in bars: continue
        entry=bars[t+H]["open"]
        vals={}; good=True
        for h in (1,4,12):
            tt=t+h*H
            if tt not in bars or bars[tt]["close"]<=0: good=False; break
            vals[h]=side*math.log(bars[tt]["close"]/entry)
        if not good: continue

        events.append({"symbol":sym,"signal_time":t,"year":y,
                       "side":"LONG" if side>0 else "SHORT",
                       "score_prev":prev,"score_now":now,
                       "spread_current":cur,"spread_previous":base,
                       "ret4_trailing":trend,
                       "r1":vals[1],"r4":vals[4],"r12":vals[12]})
        n+=1
    print("SYMBOL",sym,"events",n,flush=True)

by_sym={s:sm([e for e in events if e["symbol"]==s]) for s in SYMS}
by_year={str(y):sm([e for e in events if e["year"]==y]) for y in (2023,2024,2025,2026)}
by_side={q:sm([e for e in events if e["side"]==q]) for q in ("LONG","SHORT")}
overall=sm(events)
positive=sum(by_sym[s]["mean_r12"]>0 for s in SYMS)
weeks=((datetime.date(2026,10,1)-datetime.date(2025,1,1)).days/7)*len(SYMS)
freq=len(events)/weeks
gate=(overall["mean_r12"]>=.002 and positive>=4 and freq>=.30 and
      by_year["2023"]["mean_r12"]>0 and by_year["2024"]["mean_r12"]>0)
summary={"version":"v188","status":"promote_oos" if gate else "frozen_failed_early_gate",
         "gate_pass":gate,"events":len(events),"positive_symbols":f"{positive}/6",
         "frequency_per_symbol_week":freq,"overall":overall,"by_symbol":by_sym,
         "by_year":by_year,"by_side":by_side,"source_meta":source,
         "oos_evaluated":False,"strict_engine_run":False}
(ROOT/"results/events.json").write_text(json.dumps(events,indent=2))
(ROOT/"results/summary.json").write_text(json.dumps(summary,indent=2))
print(json.dumps({"events":len(events),"mean_r1":overall["mean_r1"],"mean_r4":overall["mean_r4"],
                  "mean_r12":overall["mean_r12"],"positive_symbols":summary["positive_symbols"],
                  "freq":freq,"2023":by_year["2023"]["mean_r12"],"2024":by_year["2024"]["mean_r12"],
                  "long_r12":by_side["LONG"]["mean_r12"],"short_r12":by_side["SHORT"]["mean_r12"],
                  "gate_pass":gate},indent=2),flush=True)
