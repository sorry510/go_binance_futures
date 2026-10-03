import csv,datetime,io,json,math,time,urllib.request,zipfile
from pathlib import Path

ROOT=Path("strategy_templates/research/carry-adjusted-momentum-zero-cross/2026-10-03-early-gate")
SYMS=["SOLUSDT","DOGEUSDT","LTCUSDT","AVAXUSDT","UNIUSDT","ZECUSDT"]
KCACHE=Path("/tmp/v161_basis_momentum")
FCACHE=Path("/tmp/v178-funding"); FCACHE.mkdir(exist_ok=True)
MONTHS=["2022-12"]+[f"{y}-{m:02d}" for y in (2023,2024) for m in range(1,13)]
FBASE="https://data.binance.vision/data/futures/um/monthly/fundingRate"
UTC=datetime.timezone.utc
H=3600000
START=int(datetime.datetime(2023,1,1,tzinfo=UTC).timestamp()*1000)
END=int(datetime.datetime(2025,1,1,tzinfo=UTC).timestamp()*1000)

def kbars(sym):
    out={}
    for m in MONTHS:
        p=KCACHE/f"fut-{sym}-{m}.zip"
        if not p.exists(): continue
        z=zipfile.ZipFile(p)
        with z.open(z.namelist()[0]) as f:
            for r in csv.reader(io.TextIOWrapper(f)):
                if not r or not r[0].isdigit() or len(r)<5: continue
                t=int(r[0]); t=t//1000 if t>10**15 else t
                out[t]={"open":float(r[1]),"close":float(r[4])}
    return out

def fpath(sym,m):
    p=FCACHE/f"{sym}-{m}.zip"
    if p.exists(): return p
    u=f"{FBASE}/{sym}/{sym}-fundingRate-{m}.zip"
    last=None
    for i in range(5):
        try:
            b=urllib.request.urlopen(urllib.request.Request(u,headers={"User-Agent":"Mozilla/5.0"}),timeout=25).read()
            zipfile.ZipFile(io.BytesIO(b))
            p.write_bytes(b); return p
        except Exception as e:
            last=e; time.sleep(.4*(i+1))
    raise last

def funding(sym):
    out=[]
    for m in MONTHS:
        p=fpath(sym,m); z=zipfile.ZipFile(p)
        with z.open(z.namelist()[0]) as f:
            for r in csv.reader(io.TextIOWrapper(f)):
                if not r or not r[0].isdigit() or len(r)<3: continue
                t=int(r[0]); t=t//1000 if t>10**15 else t
                out.append((t,float(r[2])))
    return sorted(out)

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
    b=kbars(sym); ts=sorted(b); fs=funding(sym)
    source[sym]={"bars":len(ts),"funding_rows":len(fs)}
    # precompute funding sums by hour using exact causal interval (t-24h,t]
    n=0
    prev_score=None
    for j in range(24,len(ts)-12):
        t=ts[j]
        if not (START<=t<END): continue
        if t-ts[j-24] != 24*H or ts[j+12]-t != 12*H: continue
        c0=b[ts[j-24]]["close"]; c1=b[t]["close"]
        if c0<=0 or c1<=0: continue
        price_ret=math.log(c1/c0)
        fsum=sum(r for ft,r in fs if t-24*H < ft <= t)
        score=price_ret-fsum

        # Previous score must be independently reconstructible on exact continuous prior window.
        tp=ts[j-1]
        cprev0=b[ts[j-25]]["close"]; cprev1=b[tp]["close"]
        if cprev0<=0 or cprev1<=0: continue
        prev_fsum=sum(r for ft,r in fs if tp-24*H < ft <= tp)
        prev=math.log(cprev1/cprev0)-prev_fsum

        if prev<=0<score:
            side=1.0
        elif prev>=0>score:
            side=-1.0
        else:
            continue

        y=datetime.datetime.fromtimestamp(t/1000,UTC).year
        if datetime.datetime.fromtimestamp((t+12*H)/1000,UTC).year!=y: continue
        if t+H not in b: continue
        entry=b[t+H]["open"]
        vals={}; good=True
        for h in (1,4,12):
            tt=t+h*H
            if tt not in b or b[tt]["close"]<=0: good=False; break
            vals[h]=side*math.log(b[tt]["close"]/entry)
        if not good: continue
        events.append({"symbol":sym,"signal_time":t,"year":y,
                       "side":"LONG" if side>0 else "SHORT",
                       "score_prev":prev,"score_now":score,
                       "price_ret24":price_ret,"funding_sum24":fsum,
                       "r1":vals[1],"r4":vals[4],"r12":vals[12]})
        n+=1
    print("SYMBOL",sym,"events",n,"funding",len(fs),flush=True)

by_sym={s:sm([e for e in events if e["symbol"]==s]) for s in SYMS}
by_year={str(y):sm([e for e in events if e["year"]==y]) for y in (2023,2024)}
by_side={q:sm([e for e in events if e["side"]==q]) for q in ("LONG","SHORT")}
overall=sm(events)
positive=sum(by_sym[s]["mean_r12"]>0 for s in SYMS)
weeks=((datetime.date(2025,1,1)-datetime.date(2023,1,1)).days/7)*len(SYMS)
freq=len(events)/weeks
gate=(overall["mean_r12"]>=.002 and positive>=4 and freq>=.30 and
      by_year["2023"]["mean_r12"]>0 and by_year["2024"]["mean_r12"]>0)
summary={"version":"v178","status":"promote_oos" if gate else "frozen_failed_early_gate",
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
