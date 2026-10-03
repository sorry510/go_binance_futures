import io,time,urllib.request,zipfile
from pathlib import Path
from concurrent.futures import ThreadPoolExecutor,as_completed
SYMS=["SOLUSDT","DOGEUSDT","LTCUSDT","AVAXUSDT","UNIUSDT","ZECUSDT"]
MONTHS=["2024-12"]+[f"2025-{m:02d}" for m in range(1,13)]+[f"2026-{m:02d}" for m in range(1,10)]
CACHE=Path("/tmp/v179_v188_1h");CACHE.mkdir(exist_ok=True)
BASE="https://data.binance.vision/data/futures/um/monthly/klines"
def fetch(sym,mo):
    p=CACHE/f"fut-{sym}-{mo}.zip"
    if p.exists() and p.stat().st_size>100:return p
    u=f"{BASE}/{sym}/1h/{sym}-1h-{mo}.zip";last=None
    for i in range(5):
        try:
            b=urllib.request.urlopen(urllib.request.Request(u,headers={"User-Agent":"Mozilla/5.0"}),timeout=30).read()
            zipfile.ZipFile(io.BytesIO(b));p.write_bytes(b);return p
        except Exception as e:
            last=e;time.sleep(.5*(i+1))
    raise RuntimeError(f"{sym} {mo}: {last}")
jobs=[(s,m) for s in SYMS for m in MONTHS]
with ThreadPoolExecutor(max_workers=16) as ex:
    fs=[ex.submit(fetch,*j) for j in jobs]
    for i,f in enumerate(as_completed(fs),1):
        f.result()
        if i%30==0 or i==len(fs):print("DOWNLOAD",i,"/",len(fs),flush=True)
