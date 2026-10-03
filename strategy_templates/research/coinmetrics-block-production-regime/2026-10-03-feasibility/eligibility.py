import csv,datetime,io,json,time,urllib.request,urllib.error,zipfile
from functools import lru_cache
from pathlib import Path
from collections import Counter

ROOT=Path("strategy_templates/research/coinmetrics-block-production-regime/2026-10-03-feasibility")
EVENTS=json.loads((ROOT/"inputs/raw_signals.json").read_text())
CACHE=Path("/tmp/v195-binance-1h"); CACHE.mkdir(exist_ok=True)
BASE="https://data.binance.vision/data/futures/um/monthly/klines"
UTC=datetime.timezone.utc

def month(dt): return dt.strftime("%Y-%m")

@lru_cache(None)
def bars(sym,m):
    p=CACHE/f"{sym}-1h-{m}.zip"
    miss=CACHE/f"{sym}-1h-{m}.missing"
    if miss.exists(): return []
    if not p.exists():
        u=f"{BASE}/{sym}/1h/{sym}-1h-{m}.zip"
        last=None
        for i in range(5):
            try:
                req=urllib.request.Request(u,headers={"User-Agent":"Mozilla/5.0"})
                b=urllib.request.urlopen(req,timeout=25).read()
                zipfile.ZipFile(io.BytesIO(b))
                p.write_bytes(b); last=None; break
            except urllib.error.HTTPError as e:
                if e.code==404:
                    miss.write_text("404"); return []
                last=e
            except Exception as e:
                last=e
            time.sleep(.4*(i+1))
        if last is not None: raise last
    z=zipfile.ZipFile(p); out=[]
    with z.open(z.namelist()[0]) as f:
        for r in csv.reader(io.TextIOWrapper(f)):
            if not r or not r[0].isdigit() or len(r)<8: continue
            t=int(r[0]); t=t//1000 if t>10**15 else t
            out.append((t,float(r[1]),float(r[4]),float(r[7])))
    return out

res=[]
for i,e in enumerate(EVENTS,1):
    day=datetime.date.fromisoformat(e["signal_date"])
    signal_end=datetime.datetime.combine(day+datetime.timedelta(days=1),datetime.time(0),tzinfo=UTC)
    entry_ms=int(signal_end.timestamp()*1000)
    cutoff=signal_end-datetime.timedelta(days=730)
    cutoff_ms=int(cutoff.timestamp()*1000)

    sym=e["symbol"]
    crows=bars(sym,month(cutoff))
    if not crows or crows[0][0]>cutoff_ms:
        res.append({**e,"eligible":False,"reason":"history_lt_730d"})
        continue

    drows=bars(sym,month(signal_end-datetime.timedelta(seconds=1)))
    day_start=entry_ms-24*3600000
    prior=[r for r in drows if day_start<=r[0]<entry_ms]
    if len(prior)!=24:
        res.append({**e,"eligible":False,"reason":"missing_prior_24h","prior_bars":len(prior)})
        continue
    qv=sum(r[3] for r in prior)
    if qv<5_000_000:
        res.append({**e,"eligible":False,"reason":"qv_lt_5m","qv24":qv})
        continue

    erows=bars(sym,month(signal_end))
    if not any(r[0]==entry_ms for r in erows):
        res.append({**e,"eligible":False,"reason":"missing_next_day_open","qv24":qv})
        continue

    res.append({**e,"eligible":True,"reason":"ok","qv24":qv,"entry_time":entry_ms})
    if i%250==0: print("PROGRESS",i,"/",len(EVENTS),flush=True)

eligible=[x for x in res if x["eligible"]]
summary={
  "raw_signals":len(EVENTS),
  "eligible_signals":len(eligible),
  "eligible_symbols":len({x["symbol"] for x in eligible}),
  "signals_by_symbol":dict(Counter(x["symbol"] for x in eligible)),
  "long":sum(x["side"]=="LONG" for x in eligible),
  "short":sum(x["side"]=="SHORT" for x in eligible),
  "excluded_by_reason":dict(Counter(x["reason"] for x in res if not x["eligible"])),
  "post_signal_returns_read":False
}
(ROOT/"inputs/eligibility.json").write_text(json.dumps(res,indent=2))
(ROOT/"inputs/eligible_signals.json").write_text(json.dumps(eligible,indent=2))
(ROOT/"results/eligibility_summary.json").write_text(json.dumps(summary,indent=2))
print(json.dumps(summary,indent=2))
