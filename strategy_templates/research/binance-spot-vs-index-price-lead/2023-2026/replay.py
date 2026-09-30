import csv,io,zipfile,urllib.request,urllib.error,time,datetime,math,json
from concurrent.futures import ThreadPoolExecutor,as_completed
from pathlib import Path
SYMS=['BTCUSDT','ETHUSDT','BNBUSDT','XRPUSDT','SOLUSDT','DOGEUSDT','LTCUSDT','AVAXUSDT','UNIUSDT','ZECUSDT']
START=(2022,12); END=(2026,9); UA='Mozilla/5.0'; C=Path('/tmp/spot_index_cache');C.mkdir(exist_ok=True)
def months():
 y,m=START;out=[]
 while (y,m)<=END:
  out.append(f'{y:04d}-{m:02d}');m+=1
  if m==13:y+=1;m=1
 return out
def url(k,s,m):
 if k=='spot':return f'https://data.binance.vision/data/spot/monthly/klines/{s}/1h/{s}-1h-{m}.zip'
 if k=='idx':return f'https://data.binance.vision/data/futures/um/monthly/indexPriceKlines/{s}/1h/{s}-1h-{m}.zip'
 return f'https://data.binance.vision/data/futures/um/monthly/klines/{s}/1h/{s}-1h-{m}.zip'
def fetch(k,s,m):
 p=C/f'{k}-{s}-{m}.zip'
 if p.exists():return p
 for i in range(4):
  try:
   b=urllib.request.urlopen(urllib.request.Request(url(k,s,m),headers={'User-Agent':UA}),timeout=20).read();p.write_bytes(b);return p
  except urllib.error.HTTPError as e:
   if e.code==404:p.write_bytes(b'');return p
  except:pass
  time.sleep(.3*(i+1))
 p.write_bytes(b'');return p
def read(p):
 if not p.exists() or p.stat().st_size==0:return {}
 z=zipfile.ZipFile(p);f=z.namelist()[0];o={}
 for r in csv.reader(io.TextIOWrapper(z.open(f))):
  if not r or not r[0].isdigit():continue
  t=int(r[0]);t=t//1000 if t>10**15 else t;o[t]=(float(r[1]),float(r[4]))
 return o
jobs=[(k,s,m) for s in SYMS for m in months() for k in ('spot','idx','um')]
with ThreadPoolExecutor(max_workers=20) as ex:
 fs=[ex.submit(fetch,*j) for j in jobs]
 for i,f in enumerate(as_completed(fs),1):
  f.result()
  if i%200==0:print('DOWNLOAD',i,'/',len(fs),flush=True)
def msd(x):
 m=sum(x)/len(x);return m,(sum((v-m)**2 for v in x)/len(x))**.5
E=[]
for s in SYMS:
 q=[]
 for m in months():
  a=read(C/f'spot-{s}-{m}.zip');b=read(C/f'idx-{s}-{m}.zip');u=read(C/f'um-{s}-{m}.zip')
  for t in sorted(set(a)&set(b)&set(u)):
   if a[t][1]>0 and b[t][1]>0:q.append((t,u[t][0],u[t][1],math.log(a[t][1]/b[t][1])))
 q.sort();armed=True;n=0
 for i in range(720,len(q)-12):
  if q[i][0]-q[i-720][0]>721*3600000:continue
  h=[x[3] for x in q[i-720:i]];m,sd=msd(h)
  if sd<=0:continue
  z=(q[i][3]-m)/sd
  if abs(z)<1:armed=True
  if not armed or abs(z)<3:continue
  d=1.0 if z>0 else -1.0;entry=q[i+1][1]
  y=datetime.datetime.fromtimestamp(q[i][0]/1000,datetime.timezone.utc).year
  E.append((s,y,z,d*math.log(q[i+1][2]/entry),d*math.log(q[i+4][2]/entry),d*math.log(q[i+12][2]/entry)));armed=False;n+=1
 print('SYMBOL',s,'aligned',len(q),'events',n,flush=True)
def summary(a):
 if not a:return {}
 ss=sorted(set(x[0] for x in a));by={};pos=0
 for s in ss:
  q=[x for x in a if x[0]==s];v=sum(x[5] for x in q)/len(q);by[s]={'n':len(q),'mean12':v};pos+=v>0
 return {'n':len(a),'mean1':sum(x[3] for x in a)/len(a),'mean4':sum(x[4] for x in a)/len(a),'mean12':sum(x[5] for x in a)/len(a),'win12':sum(x[5]>0 for x in a)/len(a),'positive_symbols12':pos,'symbols':len(ss),'by_symbol':by}
R={'discovery_2023_2024':summary([x for x in E if x[1] in (2023,2024)]),'oos1_2025':summary([x for x in E if x[1]==2025]),'oos2_2026':summary([x for x in E if x[1]==2026]),'all':summary(E)}
print(json.dumps(R,indent=2),flush=True)
Path('/tmp/binance_spot_index_lead_summary.json').write_text(json.dumps(R,indent=2))
