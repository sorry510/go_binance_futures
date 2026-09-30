import urllib.error,zipfile,io,csv,time,datetime,subprocess,json
from concurrent.futures import ThreadPoolExecutor,as_completed
from pathlib import Path
SYMS=['BTCUSDT','ETHUSDT','BNBUSDT','XRPUSDT']
START=datetime.date(2022,12,1); END=datetime.date(2025,1,1)
CACHE=Path('/tmp/spot_tradecount_1h');CACHE.mkdir(exist_ok=True)
def months():
    d=START;out=[]
    while d<END:
        out.append(d.strftime('%Y-%m'))
        d=(d.replace(day=28)+datetime.timedelta(days=4)).replace(day=1)
    return out
def fetch(sym,m):
    p=CACHE/f'{sym}-{m}.zip'
    if p.exists():return p
    u=f'https://data.binance.vision/data/spot/monthly/klines/{sym}/1h/{sym}-1h-{m}.zip'
    tmp=str(p)+'.part'
    q=subprocess.run(['curl','--http1.1','-L','--retry','4','--retry-all-errors',
                      '--connect-timeout','8','--max-time','40','-sS','-w','%{http_code}',
                      '-o',tmp,u],capture_output=True,text=True)
    code=(q.stdout or '')[-3:]
    if q.returncode==0 and code=='200': Path(tmp).replace(p)
    elif code=='404': Path(tmp).unlink(missing_ok=True); p.write_bytes(b'')
    else: Path(tmp).unlink(missing_ok=True); raise RuntimeError((sym,m,q.returncode,code))
    return p
jobs=[(s,m) for s in SYMS for m in months()]
with ThreadPoolExecutor(max_workers=12) as ex:
    fs=[ex.submit(fetch,*j) for j in jobs]
    for i,f in enumerate(as_completed(fs),1):
        f.result()
        if i%25==0 or i==len(fs): print('DOWNLOAD',i,'/',len(fs),flush=True)
spot={}
for s in SYMS:
    d={}
    for m in months():
        p=CACHE/f'{s}-{m}.zip'
        if not p.exists() or p.stat().st_size==0:continue
        z=zipfile.ZipFile(p);fn=z.namelist()[0]
        for r in csv.reader(io.TextIOWrapper(z.open(fn))):
            if not r or not r[0].isdigit():continue
            try:
                t=int(r[0]);t=t//1000 if t>10**15 else t
                n=float(r[8])
                if n>0:d[t]=n
            except:pass
    spot[s]=d
print('SPOT_ROWS',{s:len(v) for s,v in spot.items()},flush=True)
Path('/tmp/spot_tradecount_compact.json').write_text(json.dumps(spot))
