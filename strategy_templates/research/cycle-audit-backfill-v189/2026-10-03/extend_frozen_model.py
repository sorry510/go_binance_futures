import csv, io, json, math, time, urllib.request, zipfile, hashlib
from pathlib import Path
from datetime import date, datetime, timedelta, timezone
from concurrent.futures import ThreadPoolExecutor, as_completed

ROOT=Path("strategy_templates/research/cycle-audit-backfill-v189/2026-10-03")
SRC=Path("strategy_templates/research/shared-native-feature-tree/2026-10-03-train2023-validate2024")
CACHE=Path("/tmp/v189_fullcycle_1h"); CACHE.mkdir(exist_ok=True)
BASE="https://data.binance.vision/data/futures/um/monthly/klines"
SYMS=["BTCUSDT","ETHUSDT","BNBUSDT","XRPUSDT","SOLUSDT","DOGEUSDT","LTCUSDT","AVAXUSDT","UNIUSDT","ZECUSDT"]
FEATURES=["ret4","ret12","ret24","rv24","path_eff12","qv4_vs20","taker4","trade4_vs20","ticket4_vs20"]
UTC=timezone.utc; H=3_600_000
START=int(datetime(2025,1,1,tzinfo=UTC).timestamp()*1000)
END=int(datetime(2026,10,1,tzinfo=UTC).timestamp()*1000)

model_bytes=(SRC/"model.json").read_bytes()
model_sha=hashlib.sha256(model_bytes).hexdigest()
EXPECTED="88289990d5a3936dfa0d11e4b5f11a8111f6b330a40a8319c7e1184b17526eaf"
if model_sha!=EXPECTED:
    raise RuntimeError(f"model hash changed: {model_sha}")
MODEL=json.loads(model_bytes)

def months(a,b):
    out=[]; d=date(a.year,a.month,1); e=date(b.year,b.month,1)
    while d<=e:
        out.append(d.strftime("%Y-%m"))
        d=(d.replace(day=28)+timedelta(days=4)).replace(day=1)
    return out
MONTHS=months(date(2024,12,1),date(2026,9,1))

def zpath(sym,mo): return CACHE/f"{sym}-1h-{mo}.zip"
def fetch(sym,mo):
    p=zpath(sym,mo)
    if p.exists() and p.stat().st_size>100: return p
    u=f"{BASE}/{sym}/1h/{sym}-1h-{mo}.zip"; last=None
    for i in range(5):
        try:
            b=urllib.request.urlopen(urllib.request.Request(u,headers={"User-Agent":"Mozilla/5.0"}),timeout=30).read()
            zipfile.ZipFile(io.BytesIO(b))
            p.write_bytes(b); return p
        except Exception as e:
            last=e; time.sleep(.4*(i+1))
    raise RuntimeError(f"fetch failed {u}: {last}")

jobs=[(s,m) for s in SYMS for m in MONTHS]
with ThreadPoolExecutor(max_workers=16) as ex:
    fs=[ex.submit(fetch,*j) for j in jobs]
    for i,f in enumerate(as_completed(fs),1):
        f.result()
        if i%50==0 or i==len(fs): print("DOWNLOAD",i,"/",len(fs),flush=True)

def read(sym):
    out={}
    for mo in MONTHS:
        z=zipfile.ZipFile(zpath(sym,mo))
        with z.open(z.namelist()[0]) as fh:
            for r in csv.reader(io.TextIOWrapper(fh)):
                if not r or not r[0].isdigit() or len(r)<11: continue
                t=int(r[0]); t=t//1000 if t>10**15 else t
                vals=[float(r[i]) for i in range(1,11)]
                o,h,l,c=vals[:4]; qv=vals[6]; cnt=vals[7]; tbq=vals[9]
                if min(o,h,l,c)<=0 or qv<=0 or cnt<=0: continue
                out[t]={"open":o,"high":h,"low":l,"close":c,"qv":qv,"count":cnt,"tbq":tbq}
    return out

def features(b,ts,i):
    if i<24 or ts[i]-ts[i-24]!=24*H: return None
    def lr(a,z): return math.log(b[ts[z]]["close"]/b[ts[a]]["close"])
    x=[0.0]*9
    x[0]=lr(i-4,i); x[1]=lr(i-12,i); x[2]=lr(i-24,i)
    rv=0.0; path=0.0
    for k in range(i-23,i+1):
        r=math.log(b[ts[k]]["close"]/b[ts[k-1]]["close"])
        rv+=r*r
        if k>=i-11: path+=abs(r)
    x[3]=math.sqrt(rv); x[4]=abs(x[1])/path if path>0 else 0.0
    q4=q20=c4=c20=taker=0.0
    for k in range(i-23,i+1):
        row=b[ts[k]]
        if row["qv"]<=0 or row["count"]<=0: return None
        if k>=i-3:
            q4+=row["qv"]; c4+=row["count"]; taker+=2*row["tbq"]/row["qv"]-1
        else:
            q20+=row["qv"]; c20+=row["count"]
    if min(q4,q20,c4,c20)<=0: return None
    x[5]=math.log((q4/4)/(q20/20))
    x[6]=taker/4
    x[7]=math.log((c4/4)/(c20/20))
    x[8]=math.log((q4/c4)/(q20/c20))
    if any(not math.isfinite(v) for v in x): return None
    return x

def predict(node,x):
    while not node["leaf"]:
        f=node.get("feature")
        if f is None:
            f=FEATURES.index(node["feature_name"])
        node=node["left"] if x[f] <= node["threshold"] else node["right"]
    return node["value"]

events=[]; source={}
for sym in SYMS:
    b=read(sym); ts=sorted(b)
    source[sym]={"bars":len(ts),"first":ts[0] if ts else None,"last":ts[-1] if ts else None}
    prev_pred=None; prev_t=None; n=0
    for i in range(24,len(ts)-12):
        t=ts[i]
        if not (START<=t<END): continue
        if ts[i+1]-t!=H or ts[i+12]-t!=12*H: continue
        if t+12*H>=END: continue
        x=features(b,ts,i)
        if x is None: continue
        p=predict(MODEL["root"],x)
        if prev_pred is None or prev_t is None or t-prev_t!=H:
            prev_pred=p; prev_t=t; continue
        side=0.0
        if prev_pred<=0<p: side=1.0
        elif prev_pred>=0>p: side=-1.0
        if side:
            y=datetime.fromtimestamp(t/1000,UTC).year
            if datetime.fromtimestamp((t+12*H)/1000,UTC).year==y:
                entry=b[ts[i+1]]["open"]; vals={}
                good=entry>0
                for hh in (1,4,12):
                    if ts[i+hh]-t!=hh*H or b[ts[i+hh]]["close"]<=0:
                        good=False; break
                    vals[hh]=side*math.log(b[ts[i+hh]]["close"]/entry)
                if good:
                    events.append({"symbol":sym,"signal_time":t,"year":y,
                                   "side":"LONG" if side>0 else "SHORT",
                                   "pred_prev":prev_pred,"pred_now":p,
                                   "r1":vals[1],"r4":vals[4],"r12":vals[12]})
                    n+=1
        prev_pred=p; prev_t=t
    print("SYMBOL",sym,"later_events",n,flush=True)

def mean(xs): return sum(xs)/len(xs) if xs else 0.0
def sm(es):
    return {"n":len(es),"mean_r1":mean([e["r1"] for e in es]),"mean_r4":mean([e["r4"] for e in es]),
            "mean_r12":mean([e["r12"] for e in es]),"win_r12":mean([1.0 if e["r12"]>0 else 0.0 for e in es]),
            "long":sum(e["side"]=="LONG" for e in es),"short":sum(e["side"]=="SHORT" for e in es)}

old=json.loads((SRC/"results/validation_events.json").read_text())
old_norm=[{**e,"year":2024} for e in old]
combined=old_norm+events
by_year={str(y):sm([e for e in combined if e["year"]==y]) for y in (2024,2025,2026)}
by_sym={s:sm([e for e in combined if e["symbol"]==s]) for s in SYMS}
by_side={q:sm([e for e in combined if e["side"]==q]) for q in ("LONG","SHORT")}
overall=sm(combined)
pos=sum(by_sym[s]["mean_r12"]>0 for s in SYMS)
yearpos=sum(by_year[str(y)]["mean_r12"]>0 for y in (2024,2025,2026))
weeks=((date(2027,1,1)-date(2024,1,1)).days/7 - (date(2026,12,31)-date(2026,9,30)).days/7)*len(SYMS)
summary={
 "version":"v189","model_sha256":model_sha,"train_period":"2023",
 "validation_period":"2024-01-01..2026-09-30","events":len(combined),
 "overall":overall,"by_year":by_year,"by_symbol":by_sym,"by_side":by_side,
 "positive_symbols":f"{pos}/10","positive_validation_years":f"{yearpos}/3",
 "frequency_per_symbol_week":len(combined)/weeks,
 "raw_economic_gate_0_20pct":overall["mean_r12"]>=0.002,
 "strict_engine_run":False,"db_write":False,"source_meta":source
}
(ROOT/"results/later_events.json").write_text(json.dumps(events,indent=2))
(ROOT/"results/full_validation_events.json").write_text(json.dumps(combined,indent=2))
(ROOT/"results/summary.json").write_text(json.dumps(summary,indent=2))
print(json.dumps({
 "events":summary["events"],"r12":overall["mean_r12"],"positive_symbols":summary["positive_symbols"],
 "positive_validation_years":summary["positive_validation_years"],
 "years":{y:by_year[y]["mean_r12"] for y in ("2024","2025","2026")},
 "long_r12":by_side["LONG"]["mean_r12"],"short_r12":by_side["SHORT"]["mean_r12"],
 "raw_gate":summary["raw_economic_gate_0_20pct"]
},indent=2),flush=True)
