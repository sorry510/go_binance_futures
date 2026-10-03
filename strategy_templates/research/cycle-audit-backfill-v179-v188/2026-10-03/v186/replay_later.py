import csv,datetime,io,json,math,zipfile
from pathlib import Path

ROOT=Path("strategy_templates/research/cycle-audit-backfill-v179-v188/2026-10-03/v186")
SYMS=["SOLUSDT","DOGEUSDT","LTCUSDT","AVAXUSDT","UNIUSDT","ZECUSDT"]
CACHE=Path("/tmp/v179_v188_1h")
MONTHS=["2024-12"]+[f"2025-{m:02d}" for m in range(1,13)]+[f"2026-{m:02d}" for m in range(1,10)]
UTC=datetime.timezone.utc
H=3600000
START=int(datetime.datetime(2025,1,1,tzinfo=UTC).timestamp()*1000)
END=int(datetime.datetime(2026,10,1,tzinfo=UTC).timestamp()*1000)

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
                out[t]={"open":float(r[1]),"high":float(r[2]),"low":float(r[3]),"close":float(r[4])}
    return out

def score4(bars,ts,j):
    win=[bars[ts[k]] for k in range(j-3,j+1)]
    hi=max(x["high"] for x in win); lo=min(x["low"] for x in win)
    hi_i=next(i for i,x in enumerate(win) if x["high"]==hi)
    lo_i=next(i for i,x in enumerate(win) if x["low"]==lo)
    return hi_i-lo_i

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
    for j in range(4,len(ts)-12):
        t=ts[j]
        if not (START<=t<END): continue
        if t-ts[j-4] != 4*H or ts[j+12]-t != 12*H: continue
        prev=score4(bars,ts,j-1); now=score4(bars,ts,j)
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
            if tt not in bars or bars[tt]["close"]<=0: good=False; break
            vals[h]=side*math.log(bars[tt]["close"]/entry)
        if not good: continue
        events.append({"symbol":sym,"signal_time":t,"year":y,
                       "side":"LONG" if side>0 else "SHORT",
                       "score_prev":prev,"score_now":now,
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
summary={"version":"v186","status":"promote_oos" if gate else "frozen_failed_early_gate",
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
