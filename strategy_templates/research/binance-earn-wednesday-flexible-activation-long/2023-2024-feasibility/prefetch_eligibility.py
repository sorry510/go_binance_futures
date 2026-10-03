import datetime, json, time, urllib.request, urllib.error
from concurrent.futures import ThreadPoolExecutor, as_completed
from pathlib import Path

UTC=datetime.timezone.utc
ROOT=Path("strategy_templates/research/binance-earn-wednesday-flexible-activation-long/2023-2024-feasibility")
EVENTS=json.loads((ROOT/"inputs/raw_events.json").read_text())
CACHE=Path("/tmp/binance-earn-wednesday-eligibility"); CACHE.mkdir(exist_ok=True)
VISION="https://data.binance.vision/data/futures/um/monthly/klines"
UA="Mozilla/5.0"; DAY=86400000

def month_key(ms):
    return datetime.datetime.fromtimestamp(ms/1000,UTC).strftime("%Y-%m")
def previous_month(ym):
    y,m=map(int,ym.split("-"))
    return f"{y-1:04d}-12" if m==1 else f"{y:04d}-{m-1:02d}"

need=set()
for e in EVENTS:
    evt=int(e["publish_ms"])
    for ym in [month_key(evt-730*DAY), previous_month(month_key(evt-730*DAY)),
               month_key(evt), previous_month(month_key(evt))]:
        need.add((e["symbol"],ym))
print("NEEDED",len(need),flush=True)
def fetch(item):
    sym,ym=item
    path=CACHE/f"{sym}-1h-{ym}.zip"; miss=CACHE/f"{sym}-1h-{ym}.missing"
    if path.exists() or miss.exists(): return "cached",sym,ym
    url=f"{VISION}/{sym}/1h/{sym}-1h-{ym}.zip"
    last=None
    for k in range(5):
        try:
            req=urllib.request.Request(url,headers={"User-Agent":UA})
            with urllib.request.urlopen(req,timeout=25) as r: data=r.read()
            path.write_bytes(data); return "ok",sym,ym
        except urllib.error.HTTPError as e:
            if e.code==404:
                miss.write_text("404"); return "404",sym,ym
            last=e
        except Exception as e: last=e
        time.sleep(.4*(k+1))
    return "error",sym,ym,repr(last)

counts={}; errors=[]
with ThreadPoolExecutor(max_workers=8) as ex:
    futs=[ex.submit(fetch,x) for x in sorted(need)]
    for i,f in enumerate(as_completed(futs),1):
        r=f.result(); counts[r[0]]=counts.get(r[0],0)+1
        if r[0]=="error": errors.append(r)
        if i%50==0 or i==len(futs): print("PREFETCH",i,len(futs),counts,flush=True)
if errors: raise RuntimeError(errors[:10])
