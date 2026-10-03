import csv,datetime,io,json,math,zipfile
from pathlib import Path

ROOT=Path("strategy_templates/research/return-energy-concentration-regime/2026-10-03-early-gate")
SYMS=["SOLUSDT","DOGEUSDT","LTCUSDT","AVAXUSDT","UNIUSDT","ZECUSDT"]
CACHE=Path("/tmp/v161_basis_momentum")
MONTHS=["2022-12"]+[f"{y}-{m:02d}" for y in (2023,2024) for m in range(1,13)]
UTC=datetime.timezone.utc; H=3600000
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
                if not r or not r[0].isdigit() or len(r)<5: continue
                t=int(r[0]); t=t//1000 if t>10**15 else t
                out[t]={"open":float(r[1]),"close":float(r[4])}
    return out

def hhi(vals):
    e=[x*x for x in vals]; s=sum(e)
    if s<=0:return None
    return sum((x/s)*(x/s) for x in e)

def mean(xs): return sum(xs)/len(xs) if xs else 0.0
def sm(rows):
    return {"n":len(rows),"mean_r1":mean([x["r1"] for x in rows]),"mean_r4":mean([x["r4"] for x in rows]),
            "mean_r12":mean([x["r12"] for x in rows]),"win_r12":mean([1.0 if x["r12"]>0 else 0.0 for x in rows]),
            "long":sum(x["side"]=="LONG" for x in rows),"short":sum(x["side"]=="SHORT" for x in rows)}

events=[];source={}
for sym in SYMS:
    b=read(sym);ts=sorted(b);rets={}
    for i in range(1,len(ts)):
        if ts[i]-ts[i-1]==H and b[ts[i-1]]["close"]>0 and b[ts[i]]["close"]>0:
            rets[ts[i]]=math.log(b[ts[i]]["close"]/b[ts[i-1]]["close"])
    source[sym]={"bars":len(ts),"returns":len(rets)}
    n=0
    for j in range(49,len(ts)-12):
        t=ts[j]
        if not(START<=t<END):continue
        if t-ts[j-49]!=49*H or ts[j+12]-t!=12*H:continue
        prev_block=[rets.get(ts[k]) for k in range(j-47,j-23)]
        curr_block=[rets.get(ts[k]) for k in range(j-23,j+1)]
        # previous score for zero-cross uses blocks shifted one hour back
        prev2=[rets.get(ts[k]) for k in range(j-48,j-24)]
        curr2=[rets.get(ts[k]) for k in range(j-24,j)]
        if any(x is None for x in prev_block+curr_block+prev2+curr2):continue
        hp=hhi(prev_block);hc=hhi(curr_block); hp2=hhi(prev2);hc2=hhi(curr2)
        if None in (hp,hc,hp2,hc2):continue
        now=math.log(hc/hp); prev=math.log(hc2/hp2)
        t4=t-4*H
        if t4 not in b:continue
        trend=math.log(b[t]["close"]/b[t4]["close"])
        if trend==0:continue
        if prev<=0<now: side=-1.0 if trend>0 else 1.0
        elif prev>=0>now: side=1.0 if trend>0 else -1.0
        else:continue
        y=datetime.datetime.fromtimestamp(t/1000,UTC).year
        if datetime.datetime.fromtimestamp((t+12*H)/1000,UTC).year!=y:continue
        entry=b[t+H]["open"]; vals={}
        ok=True
        for h in (1,4,12):
            tt=t+h*H
            if tt not in b or b[tt]["close"]<=0:ok=False;break
            vals[h]=side*math.log(b[tt]["close"]/entry)
        if not ok:continue
        events.append({"symbol":sym,"signal_time":t,"year":y,"side":"LONG" if side>0 else "SHORT",
                       "score_prev":prev,"score_now":now,"ret4_trailing":trend,
                       "r1":vals[1],"r4":vals[4],"r12":vals[12]});n+=1
    print("SYMBOL",sym,"events",n,flush=True)

by_sym={s:sm([e for e in events if e["symbol"]==s]) for s in SYMS}
by_year={str(y):sm([e for e in events if e["year"]==y]) for y in (2023,2024)}
by_side={q:sm([e for e in events if e["side"]==q]) for q in ("LONG","SHORT")}
overall=sm(events);positive=sum(by_sym[s]["mean_r12"]>0 for s in SYMS)
weeks=((datetime.date(2025,1,1)-datetime.date(2023,1,1)).days/7)*len(SYMS);freq=len(events)/weeks
gate=(overall["mean_r12"]>=.002 and positive>=4 and freq>=.30 and by_year["2023"]["mean_r12"]>0 and by_year["2024"]["mean_r12"]>0)
summary={"version":"v179","status":"promote_oos" if gate else "frozen_failed_early_gate","gate_pass":gate,
         "events":len(events),"positive_symbols":f"{positive}/6","frequency_per_symbol_week":freq,
         "overall":overall,"by_symbol":by_sym,"by_year":by_year,"by_side":by_side,
         "source_meta":source,"oos_evaluated":False,"strict_engine_run":False}
(ROOT/"results/events.json").write_text(json.dumps(events,indent=2))
(ROOT/"results/summary.json").write_text(json.dumps(summary,indent=2))
print(json.dumps({"events":len(events),"mean_r1":overall["mean_r1"],"mean_r4":overall["mean_r4"],"mean_r12":overall["mean_r12"],
                  "positive_symbols":summary["positive_symbols"],"freq":freq,"2023":by_year["2023"]["mean_r12"],
                  "2024":by_year["2024"]["mean_r12"],"long_r12":by_side["LONG"]["mean_r12"],
                  "short_r12":by_side["SHORT"]["mean_r12"],"gate_pass":gate},indent=2),flush=True)
