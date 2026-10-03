import csv, io, json, math, time, urllib.request, zipfile
from pathlib import Path
from datetime import date, datetime, timedelta, timezone
from concurrent.futures import ThreadPoolExecutor, as_completed

ROOT=Path("strategy_templates/research/cycle-audit-backfill-v191-v193/2026-10-03")
CACHE=Path("/tmp/v191_v193_fullcycle"); CACHE.mkdir(exist_ok=True)
BASE="https://data.binance.vision/data/futures/um/monthly/klines"
SYMS=["SOLUSDT","DOGEUSDT","LTCUSDT","AVAXUSDT","UNIUSDT","ZECUSDT"]
UTC=timezone.utc; H=3_600_000; M=60_000
START=int(datetime(2025,1,1,tzinfo=UTC).timestamp()*1000)
END=int(datetime(2026,10,1,tzinfo=UTC).timestamp()*1000)

def months(a,b):
    out=[]; d=date(a.year,a.month,1); e=date(b.year,b.month,1)
    while d<=e:
        out.append(d.strftime("%Y-%m"))
        d=(d.replace(day=28)+timedelta(days=4)).replace(day=1)
    return out
MONTHS=months(date(2024,12,1),date(2026,9,1))

def zpath(sym,it,mo): return CACHE/f"{sym}-{it}-{mo}.zip"
def fetch(sym,it,mo):
    p=zpath(sym,it,mo)
    if p.exists() and p.stat().st_size>100: return p
    u=f"{BASE}/{sym}/{it}/{sym}-{it}-{mo}.zip"; last=None
    for i in range(5):
        try:
            b=urllib.request.urlopen(urllib.request.Request(u,headers={"User-Agent":"Mozilla/5.0"}),timeout=45).read()
            zipfile.ZipFile(io.BytesIO(b))
            p.write_bytes(b); return p
        except Exception as e:
            last=e; time.sleep(.5*(i+1))
    raise RuntimeError(f"fetch failed {u}: {last}")

jobs=[(s,it,m) for s in SYMS for it in ("1h","1m") for m in MONTHS]
with ThreadPoolExecutor(max_workers=16) as ex:
    fs=[ex.submit(fetch,*j) for j in jobs]
    for i,f in enumerate(as_completed(fs),1):
        f.result()
        if i%40==0 or i==len(fs): print("DOWNLOAD",i,"/",len(fs),flush=True)

def iter_rows(sym,it):
    for mo in MONTHS:
        z=zipfile.ZipFile(zpath(sym,it,mo))
        with z.open(z.namelist()[0]) as fh:
            for r in csv.reader(io.TextIOWrapper(fh)):
                if not r or not r[0].isdigit() or len(r)<5: continue
                t=int(r[0]); t=t//1000 if t>10**15 else t
                yield t,r

def read_hours(sym):
    out={}
    for t,r in iter_rows(sym,"1h"):
        o=float(r[1]); c=float(r[4])
        if o>0 and c>0: out[t]={"open":o,"close":c}
    return out

def sample_skew(xs):
    n=len(xs)
    if n<3: return None
    m=sum(xs)/n
    ss=sum((x-m)**2 for x in xs)
    cube=sum((x-m)**3 for x in xs)
    if ss<=0: return None
    s=math.sqrt(ss/(n-1))
    if s<=0: return None
    return n/((n-1)*(n-2))*cube/(s*s*s)

def minute_states(sym):
    closes={}
    for t,r in iter_rows(sym,"1m"):
        c=float(r[4])
        if c>0: closes[t]=c
    skew={}; stale={}
    for t in range(START,END,H):
        start_close=closes.get(t-M)
        if start_close is None or start_close<=0: continue
        prev=start_close; prev_t=t-M; rs=[]; zc=0; last=None; ok=True
        for k in range(60):
            mt=t+k*M; c=closes.get(mt)
            if c is None or c<=0 or mt-prev_t!=M:
                ok=False; break
            r=math.log(c/prev)
            rs.append(r)
            if c==prev: zc+=1
            prev=c; prev_t=mt; last=c
        if not ok or len(rs)!=60 or last is None: continue
        sk=sample_skew(rs)
        if sk is not None: skew[t]=sk
        stale[t]={"zeros":zc,"formation":math.log(last/start_close)}
    return skew,stale,len(closes)

def fwd(hours,t,side):
    if t+12*H>=END: return None
    y=datetime.fromtimestamp(t/1000,UTC).year
    if datetime.fromtimestamp((t+12*H)/1000,UTC).year!=y: return None
    entry=hours.get(t+H)
    if not entry or entry["open"]<=0: return None
    vals={}
    for hh in (1,4,12):
        b=hours.get(t+hh*H)
        if not b or b["close"]<=0: return None
        vals[hh]=side*math.log(b["close"]/entry["open"])
    return vals

later={"v191":[],"v193":[]}; source={}
for sym in SYMS:
    hours=read_hours(sym)
    skew,stale,nmins=minute_states(sym)
    source[sym]={"minute_rows":nmins,"hour_rows":len(hours),"skew_hours":len(skew),"stale_hours":len(stale)}
    n191=n193=0
    for t in sorted(skew):
        if not(START+H<=t<END) or t-H not in skew: continue
        prev=skew[t-H]; now=skew[t]
        side=0.0
        if prev<=0<now: side=-1.0
        elif prev>=0>now: side=1.0
        if not side: continue
        vals=fwd(hours,t,side)
        if vals:
            later["v191"].append({"symbol":sym,"signal_time":t,"year":datetime.fromtimestamp(t/1000,UTC).year,
                                  "side":"LONG" if side>0 else "SHORT","skew_prev":prev,"skew_now":now,
                                  "r1":vals[1],"r4":vals[4],"r12":vals[12]})
            n191+=1
    for t in sorted(stale):
        if not(START+H<=t<END) or t-H not in stale: continue
        prev=stale[t-H]; now=stale[t]
        if prev["zeros"]!=0 or now["zeros"]<=0 or now["formation"]==0: continue
        side=-1.0 if now["formation"]>0 else 1.0
        vals=fwd(hours,t,side)
        if vals:
            later["v193"].append({"symbol":sym,"signal_time":t,"year":datetime.fromtimestamp(t/1000,UTC).year,
                                  "side":"LONG" if side>0 else "SHORT","zero_prev":prev["zeros"],"zero_now":now["zeros"],
                                  "formation_return":now["formation"],"r1":vals[1],"r4":vals[4],"r12":vals[12]})
            n193+=1
    print("SYMBOL",sym,"v191",n191,"v193",n193,"minute_rows",nmins,flush=True)

OLD={
 "v191":Path("strategy_templates/research/intrahour-1m-realized-skew-reversal/2026-10-03-early-gate/results/events.json"),
 "v193":Path("strategy_templates/research/intrahour-price-staleness-activation-reversal/2026-10-03-early-gate/results/events.json")
}
def mean(xs): return sum(xs)/len(xs) if xs else 0.0
def sm(es):
    return {"n":len(es),"mean_r1":mean([e["r1"] for e in es]),"mean_r4":mean([e["r4"] for e in es]),
            "mean_r12":mean([e["r12"] for e in es]),"win_r12":mean([1.0 if e["r12"]>0 else 0.0 for e in es]),
            "long":sum(e["side"]=="LONG" for e in es),"short":sum(e["side"]=="SHORT" for e in es)}
def summarize(v,es):
    byy={str(y):sm([e for e in es if int(e["year"])==y]) for y in (2023,2024,2025,2026)}
    bys={s:sm([e for e in es if e["symbol"]==s]) for s in SYMS}
    byside={q:sm([e for e in es if e["side"]==q]) for q in ("LONG","SHORT")}
    weeks=((date(2026,10,1)-date(2023,1,1)).days/7)*len(SYMS)
    pos=sum(bys[s]["mean_r12"]>0 for s in SYMS)
    ypos=sum(byy[str(y)]["mean_r12"]>0 for y in (2023,2024,2025,2026))
    return {"version":v,"events":len(es),"overall":sm(es),"by_year":byy,"by_symbol":bys,"by_side":byside,
            "positive_symbols":f"{pos}/6","positive_years":f"{ypos}/4",
            "frequency_per_symbol_week":len(es)/weeks,"source_meta":source}

summaries={}
for v in ("v191","v193"):
    old=json.loads(OLD[v].read_text())
    if any(int(e["year"]) not in (2023,2024) for e in old): raise RuntimeError(v+" old contamination")
    if any(int(e["year"]) not in (2025,2026) for e in later[v]): raise RuntimeError(v+" later contamination")
    combined=old+later[v]
    summaries[v]=summarize(v,combined)
    (ROOT/"results"/f"{v}_later_events.json").write_text(json.dumps(later[v],indent=2))
    (ROOT/"results"/f"{v}_events.json").write_text(json.dumps(combined,indent=2))

(ROOT/"results/summary.json").write_text(json.dumps(summaries,indent=2))
print(json.dumps({v:{
    "events":s["events"],"r12":s["overall"]["mean_r12"],"positive_symbols":s["positive_symbols"],
    "positive_years":s["positive_years"],"freq":s["frequency_per_symbol_week"],
    "years":{y:s["by_year"][y]["mean_r12"] for y in ("2023","2024","2025","2026")},
    "long_r12":s["by_side"]["LONG"]["mean_r12"],"short_r12":s["by_side"]["SHORT"]["mean_r12"]
} for v,s in summaries.items()},indent=2),flush=True)
