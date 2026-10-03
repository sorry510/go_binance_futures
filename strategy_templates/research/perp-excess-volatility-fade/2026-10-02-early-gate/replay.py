import csv,io,zipfile,datetime,math,json
from pathlib import Path

ROOT=Path("strategy_templates/research/perp-excess-volatility-fade/2026-10-02-early-gate")
SYMS=["SOLUSDT","DOGEUSDT","LTCUSDT","AVAXUSDT","UNIUSDT","ZECUSDT"]
CACHE=Path("/tmp/v161_basis_momentum")
MONTHS=["2022-12"]+[f"{y}-{m:02d}" for y in (2023,2024) for m in range(1,13)]
START=int(datetime.datetime(2023,1,1,tzinfo=datetime.timezone.utc).timestamp()*1000)
END=int(datetime.datetime(2025,1,1,tzinfo=datetime.timezone.utc).timestamp()*1000)
H=3600000

def read(kind,s,m):
    p=CACHE/f"{kind}-{s}-{m}.zip"
    if not p.exists(): return {}
    z=zipfile.ZipFile(p); out={}
    with z.open(z.namelist()[0]) as f:
        for r in csv.reader(io.TextIOWrapper(f)):
            if not r or not r[0].isdigit() or len(r)<5: continue
            t=int(r[0]); t=t//1000 if t>10**15 else t
            out[t]={"open":float(r[1]),"close":float(r[4])}
    return out

def rv(rows, times, a, b):
    # 24 one-hour close-to-close returns for indices a+1..b, where b-a=24.
    s=0.0
    for k in range(a+1,b+1):
        c0=rows[times[k-1]]["close"]; c1=rows[times[k]]["close"]
        if c0<=0 or c1<=0: return 0.0
        r=math.log(c1/c0); s+=r*r
    return math.sqrt(s)

events=[]; source={}
for sym in SYMS:
    sp={}; fu={}
    for m in MONTHS:
        sp.update(read("spot",sym,m)); fu.update(read("fut",sym,m))
    common=sorted(set(sp)&set(fu))
    source[sym]={"spot_bars":len(sp),"fut_bars":len(fu),"aligned":len(common)}
    n=0
    for j in range(25,len(common)-12):
        t=common[j]
        if not (START<=t<END): continue
        # prev score needs closes from j-25..j-1; now score j-24..j.
        if common[j]-common[j-25] != 25*H: continue
        if common[j+12]-t != 12*H: continue

        rvp=rv(fu,common,j-24,j); rvs=rv(sp,common,j-24,j)
        pvp=rv(fu,common,j-25,j-1); pvs=rv(sp,common,j-25,j-1)
        if min(rvp,rvs,pvp,pvs)<=0: continue
        now=math.log(rvp/rvs); prev=math.log(pvp/pvs)
        if not (prev<=0<now): continue

        t24=t-24*H
        if t24 not in fu or fu[t24]["close"]<=0 or fu[t]["close"]<=0: continue
        ret24=math.log(fu[t]["close"]/fu[t24]["close"])
        if ret24==0: continue
        side=-1.0 if ret24>0 else 1.0

        y=datetime.datetime.fromtimestamp(t/1000,datetime.timezone.utc).year
        if datetime.datetime.fromtimestamp((t+12*H)/1000,datetime.timezone.utc).year!=y: continue
        entry=fu[t+H]["open"]
        if entry<=0: continue
        rs={}; ok=True
        for h in (1,4,12):
            tt=t+h*H
            if tt not in fu or fu[tt]["close"]<=0: ok=False; break
            rs[h]=side*math.log(fu[tt]["close"]/entry)
        if not ok: continue
        events.append({
          "symbol":sym,"signal_time":t,"year":y,
          "side":"LONG" if side>0 else "SHORT",
          "score_prev":prev,"score_now":now,
          "rv_perp":rvp,"rv_spot":rvs,"ret24":ret24,
          "r1":rs[1],"r4":rs[4],"r12":rs[12]
        }); n+=1
    print("SYMBOL",sym,"events",n,flush=True)

def mean(v): return sum(v)/len(v) if v else 0.0
def sm(rows):
    return {"n":len(rows),
      "mean_r1":mean([x["r1"] for x in rows]),
      "mean_r4":mean([x["r4"] for x in rows]),
      "mean_r12":mean([x["r12"] for x in rows]),
      "win_r12":mean([1.0 if x["r12"]>0 else 0.0 for x in rows]),
      "long":sum(x["side"]=="LONG" for x in rows),
      "short":sum(x["side"]=="SHORT" for x in rows)}

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
 "version":"v164","status":"promote_oos" if gate else "frozen_failed_early_gate",
 "gate_pass":gate,"events":len(events),"positive_symbols":f"{positive}/6",
 "frequency_per_symbol_week":freq,"overall":overall,"by_symbol":by_sym,
 "by_year":by_year,"by_side":by_side,"source_meta":source,
 "oos_evaluated":False,"strict_engine_run":False
}
(ROOT/"results/events.json").write_text(json.dumps(events,indent=2))
(ROOT/"results/summary.json").write_text(json.dumps(summary,indent=2))
print(json.dumps({
 "events":len(events),"mean_r1":overall["mean_r1"],"mean_r4":overall["mean_r4"],
 "mean_r12":overall["mean_r12"],"positive_symbols":summary["positive_symbols"],
 "freq":freq,"2023":by_year["2023"]["mean_r12"],"2024":by_year["2024"]["mean_r12"],
 "long_r12":by_side["LONG"]["mean_r12"],"short_r12":by_side["SHORT"]["mean_r12"],
 "gate_pass":gate
},indent=2),flush=True)
