import csv, io, math, statistics, urllib.request, urllib.error, zipfile, time, json
from pathlib import Path
from concurrent.futures import ThreadPoolExecutor, as_completed

SYMS=["BTCUSDT","ETHUSDT","BNBUSDT","XRPUSDT","SOLUSDT","DOGEUSDT","LTCUSDT","AVAXUSDT","UNIUSDT","ZECUSDT"]
MONTHS=[]
for y in range(2023,2027):
    for m in range(1,13):
        if y==2023 and m<12: continue
        if y==2026 and m>9: continue
        MONTHS.append(f"{y:04d}-{m:02d}")
BASE="https://data.binance.vision/data/futures/um/monthly/premiumIndexKlines"
ROOT=Path(__file__).resolve().parent
CACHE=Path("/tmp/premium_index_cache"); CACHE.mkdir(exist_ok=True)
UA="Mozilla/5.0"

def fetch(task):
    sym,mon=task
    p=CACHE/f"{sym}-{mon}.zip"
    if p.exists() and p.stat().st_size>0: return sym,mon,True
    u=f"{BASE}/{sym}/5m/{sym}-5m-{mon}.zip"
    for k in range(4):
        try:
            req=urllib.request.Request(u,headers={"User-Agent":UA})
            with urllib.request.urlopen(req,timeout=25) as r:b=r.read()
            p.write_bytes(b);return sym,mon,True
        except urllib.error.HTTPError as e:
            if e.code==404:
                p.write_bytes(b"");return sym,mon,False
        except Exception:
            time.sleep(.3*(k+1))
    return sym,mon,False

tasks=[(s,m) for s in SYMS for m in MONTHS]
with ThreadPoolExecutor(max_workers=24) as ex:
    fut=[ex.submit(fetch,t) for t in tasks]
    done=0
    for f in as_completed(fut):
        done+=1
        if done%50==0: print("download",done,"/",len(tasks),flush=True)

prices={}
with open(ROOT/'inputs'/'prices.csv') as f:
    for r in csv.DictReader(f):
        prices[(r["symbol"],int(r["open_time"]))]=(float(r["open"]),float(r["close"]))

def ms(x):
    x=int(x)
    return x//1000 if x>10**15 else x

def load_premium(sym):
    out=[]
    for mon in MONTHS:
        p=CACHE/f"{sym}-{mon}.zip"
        if not p.exists() or p.stat().st_size==0: continue
        z=zipfile.ZipFile(p)
        fn=z.namelist()[0]
        rows=csv.reader(io.TextIOWrapper(z.open(fn)))
        hdr=next(rows,None)
        for r in rows:
            try:
                if not r or not str(r[0]).isdigit(): continue
                t=ms(r[0]); c=float(r[4])
                out.append((t,c))
            except: pass
    out.sort()
    return out

events=[]
for sym in SYMS:
    rows=load_premium(sym)
    vals=[]
    armed=True
    cnt=0
    for t,c in rows:
        if len(vals)>=288:
            hist=vals[-288:]
            mean=sum(hist)/288.0
            var=sum((x-mean)*(x-mean) for x in hist)/288.0
            sd=math.sqrt(var)
            z=(c-mean)/sd if sd>0 else 0.0
            if abs(z)<1: armed=True
            if armed and abs(z)>=3 and c!=0:
                event_close=t+5*60*1000
                hour=60*60*1000
                entry_t=((event_close+hour-1)//hour)*hour
                key=(sym,entry_t)
                if key in prices:
                    entry=prices[key][0]
                    direction=-1 if c>0 else 1
                    rec={"symbol":sym,"event_time":event_close,"year":__import__("datetime").datetime.fromtimestamp(event_close/1000,__import__("datetime").timezone.utc).year,
                         "premium":c,"z":z,"side":"SHORT" if direction<0 else "LONG"}
                    ok=True
                    for h in (1,4,12):
                        k=(sym,entry_t+(h-1)*hour)
                        if k not in prices: ok=False;break
                        exitp=prices[k][1]
                        rec[f"r{h}"]=direction*math.log(exitp/entry)
                    if ok:
                        events.append(rec);cnt+=1;armed=False
        vals.append(c)
    print("SYMBOL",sym,"premium_rows",len(rows),"events",cnt,flush=True)

def summary(ev):
    out={"n":len(ev),"by_symbol":{},"by_year":{}}
    for h in (1,4,12):
        v=[e[f"r{h}"] for e in ev]
        out[f"mean_{h}h"]=sum(v)/len(v) if v else None
        out[f"win_{h}h"]=sum(x>0 for x in v)/len(v) if v else None
    for s in SYMS:
        q=[e for e in ev if e["symbol"]==s]
        if q:
            out["by_symbol"][s]={"n":len(q),"mean4":sum(e["r4"] for e in q)/len(q),"mean12":sum(e["r12"] for e in q)/len(q)}
    for y in sorted(set(e["year"] for e in ev)):
        q=[e for e in ev if e["year"]==y]
        out["by_year"][str(y)]={"n":len(q),"mean4":sum(e["r4"] for e in q)/len(q),"mean12":sum(e["r12"] for e in q)/len(q)}
    out["positive_symbols_4h"]=sum(1 for s,z in out["by_symbol"].items() if z["mean4"]>0)
    out["symbols"]=len(out["by_symbol"])
    return out

res={
 "discovery_2024":summary([e for e in events if e["year"]==2024]),
 "oos1_2025":summary([e for e in events if e["year"]==2025]),
 "oos2_2026":summary([e for e in events if e["year"]==2026]),
 "all":summary(events),
}
print(json.dumps(res,indent=2),flush=True)
(ROOT/'results'/'summary.json').write_text(json.dumps(res,indent=2))
with open(ROOT/'results'/'events.csv','w',newline='') as f:
    if events:
        w=csv.DictWriter(f,fieldnames=list(events[0]));w.writeheader();w.writerows(events)
