import io,time,urllib.request,zipfile,datetime
from pathlib import Path
from concurrent.futures import ThreadPoolExecutor,as_completed
SYMS=["SOLUSDT","DOGEUSDT","LTCUSDT","AVAXUSDT","UNIUSDT","ZECUSDT"]
CACHE=Path("/tmp/v147-binance-metrics");CACHE.mkdir(exist_ok=True)
BASE="https://data.binance.vision/data/futures/um/daily/metrics"
days=[];d=datetime.date(2025,1,1);e=datetime.date(2026,9,30)
while d<=e:
    days.append(d.isoformat());d+=datetime.timedelta(days=1)
def fetch(sym,ds):
    p=CACHE/f"{sym}-{ds}.zip"
    if p.exists() and p.stat().st_size>100:return p
    u=f"{BASE}/{sym}/{sym}-metrics-{ds}.zip";last=None
    for i in range(5):
        try:
            b=urllib.request.urlopen(urllib.request.Request(u,headers={"User-Agent":"Mozilla/5.0"}),timeout=30).read()
            zipfile.ZipFile(io.BytesIO(b));p.write_bytes(b);return p
        except Exception as e:
            last=e;time.sleep(.3*(i+1))
    raise RuntimeError(f"{sym} {ds}: {last}")
jobs=[(s,d) for s in SYMS for d in days]
with ThreadPoolExecutor(max_workers=24) as ex:
    fs=[ex.submit(fetch,*j) for j in jobs]
    for i,f in enumerate(as_completed(fs),1):
        f.result()
        if i%500==0 or i==len(fs):print("DOWNLOAD",i,"/",len(fs),flush=True)
