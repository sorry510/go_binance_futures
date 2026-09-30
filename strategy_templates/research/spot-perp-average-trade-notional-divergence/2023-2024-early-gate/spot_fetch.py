import urllib.request,urllib.error,zipfile,io,csv,time,math,datetime,subprocess,json
from concurrent.futures import ThreadPoolExecutor,as_completed
from pathlib import Path
SYMS=['BTCUSDT','ETHUSDT','BNBUSDT','XRPUSDT']
START=datetime.date(2022,12,1); END=datetime.date(2025,1,1)
CACHE=Path('/tmp/spot_avgtrade_1h');CACHE.mkdir(exist_ok=True)
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
 for k in range(4):
  try:
   b=urllib.request.urlopen(urllib.request.Request(u,headers={'User-Agent':'Mozilla/5.0'}),timeout=20).read();p.write_bytes(b);return p
  except urllib.error.HTTPError as e:
   if e.code==404:p.write_bytes(b'');return p
   time.sleep(.3*(k+1))
  except Exception:time.sleep(.3*(k+1))
 p.write_bytes(b'');return p
jobs=[(s,m) for s in SYMS for m in months()]
with ThreadPoolExecutor(max_workers=12) as ex:
 fs=[ex.submit(fetch,*j) for j in jobs]
 for i,f in enumerate(as_completed(fs),1):
  f.result()
  if i%25==0 or i==len(fs):print('DOWNLOAD',i,'/',len(fs),flush=True)
# emit spot compact TSV
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
    q=float(r[7]);n=float(r[8])
    if q>0 and n>0:d[t]=q/n
   except:pass
 spot[s]=d
print('SPOT_ROWS',{s:len(v) for s,v in spot.items()},flush=True)
Path('/tmp/spot_avgtrade_compact.json').write_text(json.dumps(spot))
