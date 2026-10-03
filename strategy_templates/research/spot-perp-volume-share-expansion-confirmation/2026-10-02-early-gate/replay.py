import csv,io,zipfile,datetime,math,json
from pathlib import Path

ROOT=Path("strategy_templates/research/spot-perp-volume-share-expansion-confirmation/2026-10-02-early-gate")
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
            if not r or not r[0].isdigit() or len(r)<8: continue
            t=int(r[0]); t=t//1000 if t>10**15 else t
            out[t]={"open":float(r[1]),"close":float(r[4]),"qv":float(r[7])}
    return out

events=[]; source={}
for s in SYMS:
    sp={}; fu={}
    for m in MONTHS:
        sp.update(read("spot",s,m)); fu.update(read("fut",s,m))
    common=sorted(set(sp)&set(fu))
    source[s]={"spot_bars":len(sp),"fut_bars":len(fu),"aligned":len(common)}
    fq=[fu[t]["qv"] for t in common]
    sq=[sp[t]["qv"] for t in common]
    pf=[0.0]; ps=[0.0]
    for x in fq: pf.append(pf[-1]+x)
    for x in sq: ps.append(ps[-1]+x)

    def ratio(a,b):
        # inclusive indices a..b
        sf=pf[b+1]-pf[a]; ss=ps[b+1]-ps[a]
        return sf/ss if sf>0 and ss>0 else 0.0

    n=0
    for j in range(192,len(common)-12):
        t=common[j]
        if not (START<=t<END): continue
        # Need an exactly continuous 193-hour signal history and 12-hour forward endpoint.
        if common[j]-common[j-192] != 192*H: continue
        if common[j+12]-t != 12*H: continue

        now_cur=ratio(j-23,j); now_base=ratio(j-191,j-24)
        prev_cur=ratio(j-24,j-1); prev_base=ratio(j-192,j-25)
        if min(now_cur,now_base,prev_cur,prev_base)<=0: continue
        now=math.log(now_cur/now_base)
        prev=math.log(prev_cur/prev_base)
        if not (prev<=0<now): continue

        # Price direction uses only completed USD-M closes t vs t-24h.
        t24=t-24*H
        if t24 not in fu or fu[t24]["close"]<=0 or fu[t]["close"]<=0: continue
        ret24=math.log(fu[t]["close"]/fu[t24]["close"])
        if ret24==0: continue
        side=1.0 if ret24>0 else -1.0

        y=datetime.datetime.fromtimestamp(t/1000,datetime.timezone.utc).year
        if datetime.datetime.fromtimestamp((t+12*H)/1000,datetime.timezone.utc).year!=y: continue
        entry=fu[t+H]["open"]
        if entry<=0: continue
        rs={}
        ok=True
        for h in (1,4,12):
            tt=t+h*H
            if tt not in fu or fu[tt]["close"]<=0: ok=False; break
            rs[h]=side*math.log(fu[tt]["close"]/entry)
        if not ok: continue
        events.append({
          "symbol":s,"signal_time":t,"year":y,
          "side":"LONG" if side>0 else "SHORT",
          "score_prev":prev,"score_now":now,"ret24":ret24,
          "current_ratio":now_cur,"baseline_ratio":now_base,
          "r1":rs[1],"r4":rs[4],"r12":rs[12]
        }); n+=1
    print("SYMBOL",s,"events",n,flush=True)

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
 "version":"v163","status":"promote_oos" if gate else "frozen_failed_early_gate",
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
