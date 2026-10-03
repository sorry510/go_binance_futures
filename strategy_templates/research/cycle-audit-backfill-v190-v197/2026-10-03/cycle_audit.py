import csv, io, json, math, time, urllib.request, urllib.error, zipfile
from pathlib import Path
from datetime import date, datetime, timedelta, timezone
from collections import defaultdict
from concurrent.futures import ThreadPoolExecutor, as_completed

ROOT=Path("strategy_templates/research/cycle-audit-backfill-v190-v197/2026-10-03")
CACHE=Path("/tmp/cycle_audit_2023_2026_09"); CACHE.mkdir(exist_ok=True)
BASEURL="https://data.binance.vision/data/futures/um/monthly/klines"
SYMS=["SOLUSDT","DOGEUSDT","LTCUSDT","AVAXUSDT","UNIUSDT","ZECUSDT"]
UTC=timezone.utc; H=3_600_000; M=60_000
START=int(datetime(2023,1,1,tzinfo=UTC).timestamp()*1000)
END=int(datetime(2026,10,1,tzinfo=UTC).timestamp()*1000)

def months(a,b):
    out=[]; d=date(a.year,a.month,1); e=date(b.year,b.month,1)
    while d<=e:
        out.append(d.strftime("%Y-%m"))
        d=(d.replace(day=28)+timedelta(days=4)).replace(day=1)
    return out
MONTHS=months(date(2022,12,1),date(2026,9,1))

def zpath(sym,interval,mo): return CACHE/f"{sym}-{interval}-{mo}.zip"
def fetch(sym,interval,mo):
    p=zpath(sym,interval,mo)
    if p.exists() and p.stat().st_size>100: return p
    u=f"{BASEURL}/{sym}/{interval}/{sym}-{interval}-{mo}.zip"
    last=None
    for i in range(5):
        try:
            b=urllib.request.urlopen(urllib.request.Request(u,headers={"User-Agent":"Mozilla/5.0"}),timeout=30).read()
            zipfile.ZipFile(io.BytesIO(b))
            p.write_bytes(b); return p
        except Exception as e:
            last=e; time.sleep(.4*(i+1))
    raise RuntimeError(f"fetch failed {u}: {last}")

jobs=[(s,it,m) for s in SYMS for it in ("1h","1m") for m in MONTHS]
with ThreadPoolExecutor(max_workers=16) as ex:
    futs=[ex.submit(fetch,*j) for j in jobs]
    for i,f in enumerate(as_completed(futs),1):
        f.result()
        if i%50==0 or i==len(futs): print("DOWNLOAD",i,"/",len(futs),flush=True)

def rows(sym,interval):
    for mo in MONTHS:
        p=zpath(sym,interval,mo); z=zipfile.ZipFile(p)
        with z.open(z.namelist()[0]) as fh:
            for r in csv.reader(io.TextIOWrapper(fh)):
                if not r or not r[0].isdigit(): continue
                t=int(r[0]); t=t//1000 if t>10**15 else t
                yield t,r

def read_1h(sym):
    out={}
    for t,r in rows(sym,"1h"):
        if len(r)<11: continue
        o,h,l,c=float(r[1]),float(r[2]),float(r[3]),float(r[4])
        qv,cnt,tbq=float(r[7]),float(r[8]),float(r[10])
        if min(o,h,l,c)>0:
            out[t]={"open":o,"high":h,"low":l,"close":c,"qv":qv,"count":cnt,"tbq":tbq}
    return out

def minute_states(sym):
    sig={}; rec={}
    cur_hour=None; ref=None; prev_t=None; prev_close=None; closes=[]; times=[]
    def finalize(hour,ref,closes,times):
        if hour is None or ref is None or len(closes)!=60: return
        if times[0]!=hour or times[-1]!=hour+59*M: return
        if any(times[i]!=hour+i*M for i in range(60)): return
        if ref<=0 or any(c<=0 for c in closes): return
        rv1=0.0; p=ref
        for c in closes:
            x=math.log(c/p); rv1+=x*x; p=c
        rv5=0.0; p=ref
        for k in range(4,60,5):
            c=closes[k]; x=math.log(c/p); rv5+=x*x; p=c
        if rv1>0 and rv5>0:
            sig[hour]={"score":math.log(rv1/rv5),"formation":math.log(closes[-1]/ref),
                       "rv1":rv1,"rv5":rv5}
        ps=0; n=0
        for c in closes:
            dev=c-ref; s=1 if dev>0 else (-1 if dev<0 else 0)
            if s:
                if ps and s!=ps: n+=1
                ps=s
        rec[hour]={"recross":n,"formation":math.log(closes[-1]/ref)}
    for t,r in rows(sym,"1m"):
        c=float(r[4])
        hour=(t//H)*H
        if cur_hour is None:
            cur_hour=hour
            ref=prev_close if prev_t==hour-M else None
        elif hour!=cur_hour:
            finalize(cur_hour,ref,closes,times)
            cur_hour=hour; closes=[]; times=[]
            ref=prev_close if prev_t==hour-M else None
        closes.append(c); times.append(t)
        prev_t=t; prev_close=c
    finalize(cur_hour,ref,closes,times)
    return sig,rec

def mean(xs): return sum(xs)/len(xs) if xs else 0.0
def sm(es):
    return {"n":len(es),"mean_r1":mean([e["r1"] for e in es]),"mean_r4":mean([e["r4"] for e in es]),
            "mean_r12":mean([e["r12"] for e in es]),"win_r12":mean([1.0 if e["r12"]>0 else 0.0 for e in es]),
            "long":sum(e["side"]=="LONG" for e in es),"short":sum(e["side"]=="SHORT" for e in es)}
def fwd(bars,t,side):
    if t+12*H>=END: return None
    y=datetime.fromtimestamp(t/1000,UTC).year
    if datetime.fromtimestamp((t+12*H)/1000,UTC).year!=y: return None
    if t+H not in bars or bars[t+H]["open"]<=0: return None
    entry=bars[t+H]["open"]; vals={}
    for h in (1,4,12):
        tt=t+h*H
        if tt not in bars or bars[tt]["close"]<=0: return None
        vals[h]=side*math.log(bars[tt]["close"]/entry)
    return vals

all_events={k:[] for k in ("v190","v192","v194","v196")}
for sym in SYMS:
    bars=read_1h(sym); ts=sorted(bars); idx={t:i for i,t in enumerate(ts)}
    sig,rec=minute_states(sym)
    print("SOURCE",sym,"1h",len(bars),"sig_hours",len(sig),"rec_hours",len(rec),flush=True)

    # v190: preserve the original source rule that drops any 1h bar with non-positive QuoteVolume/TradeCount.
    vbars={t:b for t,b in bars.items() if b["qv"]>0 and b["count"]>0}
    vts=sorted(vbars)
    n=len(vts); qv=[0.0]*n; cnt=[0.0]*n; r2=[0.0]*n
    for i,t in enumerate(vts):
        qv[i]=vbars[t]["qv"]; cnt[i]=vbars[t]["count"]
        if i>0 and t-vts[i-1]==H:
            rr=math.log(vbars[t]["close"]/vbars[vts[i-1]]["close"]); r2[i]=rr*rr
    pq=[0.0]*(n+1); pc=[0.0]*(n+1); pr=[0.0]*(n+1)
    for i in range(n):
        pq[i+1]=pq[i]+qv[i];pc[i+1]=pc[i]+cnt[i];pr[i+1]=pr[i]+r2[i]
    def invariant(a,b):
        V=pq[b+1]-pq[a]; N=pc[b+1]-pc[a]; rv=pr[b+1]-pr[a]
        return None if V<=0 or N<=0 or rv<=0 else V*math.sqrt(rv)/(N**1.5)
    for j in range(49,n-12):
        t=vts[j]
        if not(START<=t<END) or t-vts[j-49]!=49*H or vts[j+12]-t!=12*H: continue
        cur=invariant(j-23,j); base=invariant(j-47,j-24); pcur=invariant(j-24,j-1); pbase=invariant(j-48,j-25)
        if None in (cur,base,pcur,pbase): continue
        now=math.log(cur/base); prev=math.log(pcur/pbase)
        if not(prev<=0<now): continue
        t4=t-4*H
        if t4 not in vbars: continue
        trend=math.log(vbars[t]["close"]/vbars[t4]["close"])
        if trend==0: continue
        side=-1.0 if trend>0 else 1.0; vals=fwd(vbars,t,side)
        if not vals: continue
        all_events["v190"].append({"symbol":sym,"signal_time":t,"year":datetime.fromtimestamp(t/1000,UTC).year,
            "side":"LONG" if side>0 else "SHORT","r1":vals[1],"r4":vals[4],"r12":vals[12]})

    # v194
    for j in range(2,n-12):
        t=ts[j]
        if not(START<=t<END) or t-ts[j-2]!=2*H or ts[j+12]-t!=12*H: continue
        a=bars[ts[j-2]]; c=bars[t]; side=0.0
        if c["low"]>a["high"]: side=-1.0
        elif c["high"]<a["low"]: side=1.0
        else: continue
        vals=fwd(bars,t,side)
        if not vals: continue
        all_events["v194"].append({"symbol":sym,"signal_time":t,"year":datetime.fromtimestamp(t/1000,UTC).year,
            "side":"LONG" if side>0 else "SHORT","r1":vals[1],"r4":vals[4],"r12":vals[12]})

    # v192
    st=sorted(sig)
    for t in st:
        if not(START<=t<END) or t-H not in sig: continue
        prev,now=sig[t-H],sig[t]
        if not(prev["score"]<=0<now["score"]) or now["formation"]==0: continue
        side=-1.0 if now["formation"]>0 else 1.0; vals=fwd(bars,t,side)
        if not vals: continue
        all_events["v192"].append({"symbol":sym,"signal_time":t,"year":datetime.fromtimestamp(t/1000,UTC).year,
            "side":"LONG" if side>0 else "SHORT","r1":vals[1],"r4":vals[4],"r12":vals[12]})

    # v196
    for t in sorted(rec):
        if not(START<=t<END) or t-H not in rec: continue
        prev,now=rec[t-H],rec[t]
        if prev["recross"]!=0 or now["recross"]<=0 or now["formation"]==0: continue
        side=-1.0 if now["formation"]>0 else 1.0; vals=fwd(bars,t,side)
        if not vals: continue
        all_events["v196"].append({"symbol":sym,"signal_time":t,"year":datetime.fromtimestamp(t/1000,UTC).year,
            "side":"LONG" if side>0 else "SHORT","r1":vals[1],"r4":vals[4],"r12":vals[12]})

# exact 2023-2024 parity against archived source events
archive_paths={
 "v190":Path("strategy_templates/research/trading-invariant-stress-reversal/2026-10-03-early-gate/results/events.json"),
 "v192":Path("strategy_templates/research/intrahour-volatility-signature-noise-reversal/2026-10-03-early-gate/results/events.json"),
 "v194":Path("strategy_templates/research/three-bar-fair-value-gap-fill-reversal/2026-10-03-early-gate/results/events.json"),
 "v196":Path("strategy_templates/research/intrahour-open-reference-recross-reversal/2026-10-03-early-gate/results/events.json"),
}
parity={}
for k,p in archive_paths.items():
    old=json.loads(p.read_text())
    new=[e for e in all_events[k] if e["year"] in (2023,2024)]
    ok={(e["symbol"],int(e["signal_time"]),e["side"]) for e in old}
    nk={(e["symbol"],int(e["signal_time"]),e["side"]) for e in new}
    parity[k]={"old":len(ok),"new":len(nk),"missing_from_new":len(ok-nk),"extra_in_new":len(nk-ok),"exact_key_parity":ok==nk}

def summarize(k,events):
    by_year={str(y):sm([e for e in events if e["year"]==y]) for y in (2023,2024,2025,2026)}
    by_sym={s:sm([e for e in events if e["symbol"]==s]) for s in SYMS}
    by_side={q:sm([e for e in events if e["side"]==q]) for q in ("LONG","SHORT")}
    ov=sm(events)
    pos=sum(by_sym[s]["mean_r12"]>0 for s in SYMS)
    years_pos=sum(by_year[str(y)]["mean_r12"]>0 for y in (2023,2024,2025,2026))
    return {"version":k,"events":len(events),"overall":ov,"by_year":by_year,"by_symbol":by_sym,"by_side":by_side,
            "positive_symbols":f"{pos}/6","positive_years":f"{years_pos}/4",
            "period":"2023-01-01..2026-09-30","parity_2023_2024":parity.get(k, {"not_applicable": True})}

summaries={k:summarize(k,v) for k,v in all_events.items()}

# v197 fixed consensus from full-period source events
lookup=defaultdict(dict)
for fam,es in all_events.items():
    for e in es:
        lookup[(e["symbol"],e["signal_time"])][fam]=e
cons=[]; mixed=0
for (sym,t),fr in sorted(lookup.items()):
    longs=sorted([f for f,e in fr.items() if e["side"]=="LONG"])
    shorts=sorted([f for f,e in fr.items() if e["side"]=="SHORT"])
    if longs and shorts: mixed+=1; continue
    fams=longs if len(longs)>=2 else (shorts if len(shorts)>=2 else [])
    if not fams: continue
    side="LONG" if longs else "SHORT"
    ref=fr[fams[0]]
    # all source events at same symbol/time/side inherit same fwd return; verify
    if any(abs(fr[f]["r12"]-ref["r12"])>1e-12 for f in fams): raise RuntimeError("consensus return mismatch")
    cons.append({"symbol":sym,"signal_time":t,"year":ref["year"],"side":side,"families":fams,
                 "r1":ref["r1"],"r4":ref["r4"],"r12":ref["r12"]})
summaries["v197"]=summarize("v197",cons); summaries["v197"]["mixed_direction_hours_discarded"]=mixed

for k,es in all_events.items():
    (ROOT/f"results/{k}_events.json").write_text(json.dumps(es,indent=2))
(ROOT/"results/v197_events.json").write_text(json.dumps(cons,indent=2))
(ROOT/"results/summary.json").write_text(json.dumps({"parity":parity,"summaries":summaries},indent=2))
print(json.dumps({
 "parity":parity,
 "summary":{k:{
    "events":v["events"],"r12":v["overall"]["mean_r12"],"positive_symbols":v["positive_symbols"],
    "positive_years":v["positive_years"],"years":{y:v["by_year"][y]["mean_r12"] for y in ("2023","2024","2025","2026")}
 } for k,v in summaries.items()}
},indent=2),flush=True)
