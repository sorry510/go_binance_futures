import csv,datetime,io,json,math,zipfile
from pathlib import Path

ROOT=Path("strategy_templates/research/trading-invariant-stress-reversal/2026-10-03-early-gate")
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
                if not r or not r[0].isdigit() or len(r)<9: continue
                t=int(r[0]); t=t//1000 if t>10**15 else t
                o,c,qv,cnt=float(r[1]),float(r[4]),float(r[7]),float(r[8])
                if min(o,c,qv,cnt)<=0: continue
                out[t]={"open":o,"close":c,"qv":qv,"count":cnt}
    return out

def mean(xs): return sum(xs)/len(xs) if xs else 0.0
def sm(rows):
    return {
      "n":len(rows),
      "mean_r1":mean([x["r1"] for x in rows]),
      "mean_r4":mean([x["r4"] for x in rows]),
      "mean_r12":mean([x["r12"] for x in rows]),
      "win_r12":mean([1.0 if x["r12"]>0 else 0.0 for x in rows]),
      "long":sum(x["side"]=="LONG" for x in rows),
      "short":sum(x["side"]=="SHORT" for x in rows)
    }

def invariant(prefix_qv,prefix_cnt,prefix_r2,a,b):
    # traded-hour bars a..b inclusive; 24 bars. sigma uses returns indices a..b,
    # where return index k is close[k]/close[k-1].
    V=prefix_qv[b+1]-prefix_qv[a]
    N=prefix_cnt[b+1]-prefix_cnt[a]
    rv2=prefix_r2[b+1]-prefix_r2[a]
    if V<=0 or N<=0 or rv2<=0: return None
    return V*math.sqrt(rv2)/(N**1.5)

events=[];source={}
for sym in SYMS:
    bars=read(sym); ts=sorted(bars)
    nbar=len(ts)
    qv=[0.0]*nbar; cnt=[0.0]*nbar; r2=[0.0]*nbar
    for i,t in enumerate(ts):
        qv[i]=bars[t]["qv"]; cnt[i]=bars[t]["count"]
        if i>0 and t-ts[i-1]==H:
            r=math.log(bars[t]["close"]/bars[ts[i-1]]["close"])
            r2[i]=r*r
    pq=[0.0]*(nbar+1); pc=[0.0]*(nbar+1); pr=[0.0]*(nbar+1)
    for i in range(nbar):
        pq[i+1]=pq[i]+qv[i]; pc[i+1]=pc[i]+cnt[i]; pr[i+1]=pr[i]+r2[i]
    source[sym]={"bars":nbar}
    n=0
    for j in range(49,nbar-12):
        t=ts[j]
        if not(START<=t<END): continue
        # Need exact history for current+baseline and prior score, and exact future.
        if t-ts[j-49] != 49*H or ts[j+12]-t != 12*H: continue

        cur=invariant(pq,pc,pr,j-23,j)
        base=invariant(pq,pc,pr,j-47,j-24)
        pcur=invariant(pq,pc,pr,j-24,j-1)
        pbase=invariant(pq,pc,pr,j-48,j-25)
        if None in (cur,base,pcur,pbase): continue
        now=math.log(cur/base); prev=math.log(pcur/pbase)
        if not(prev<=0<now): continue

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

        events.append({
          "symbol":sym,"signal_time":t,"year":y,
          "side":"LONG" if side>0 else "SHORT",
          "score_prev":prev,"score_now":now,
          "invariant_current":cur,"invariant_previous":base,
          "ret4_trailing":trend,
          "r1":vals[1],"r4":vals[4],"r12":vals[12]
        }); n+=1
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
 "version":"v190","status":"promote_oos" if gate else "frozen_failed_early_gate",
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
