import csv,io,zipfile,urllib.request,urllib.error,time,datetime,math,json
from concurrent.futures import ThreadPoolExecutor,as_completed
from pathlib import Path
UTC=datetime.timezone.utc;UA='Mozilla/5.0'
SYMS=['BTCUSDT','ETHUSDT','BNBUSDT','XRPUSDT']
START=datetime.date(2023,1,1);END=datetime.date(2024,12,31)
MC=Path('/tmp/oi_turnover_metrics');MC.mkdir(exist_ok=True)
KC=Path('/tmp/oi_turnover_klines');KC.mkdir(exist_ok=True)
def days():
 d=START;out=[]
 while d<=END:out.append(d);d+=datetime.timedelta(days=1)
 return out
def months():
 d=START.replace(day=1);out=[]
 while d<=END.replace(day=1):
  out.append(d.strftime('%Y-%m'));d=(d.replace(day=28)+datetime.timedelta(days=4)).replace(day=1)
 return out
def fetch_metric(sym,d):
 p=MC/f'{sym}-{d.isoformat()}.zip'
 if p.exists():return p
 u=f'https://data.binance.vision/data/futures/um/daily/metrics/{sym}/{sym}-metrics-{d.isoformat()}.zip'
 for k in range(4):
  try:b=urllib.request.urlopen(urllib.request.Request(u,headers={'User-Agent':UA}),timeout=20).read();p.write_bytes(b);return p
  except urllib.error.HTTPError as e:
   if e.code==404:p.write_bytes(b'');return p
   time.sleep(.3*(k+1))
  except Exception:time.sleep(.3*(k+1))
 p.write_bytes(b'');return p
def fetch_k(sym,mo):
 p=KC/f'{sym}-{mo}.zip'
 if p.exists():return p
 u=f'https://data.binance.vision/data/futures/um/monthly/klines/{sym}/1h/{sym}-1h-{mo}.zip'
 try:b=urllib.request.urlopen(urllib.request.Request(u,headers={'User-Agent':UA}),timeout=25).read();p.write_bytes(b)
 except urllib.error.HTTPError as e:
  if e.code==404:p.write_bytes(b'')
  else:raise
 return p
jobs=[('m',s,d) for s in SYMS for d in days()]+[('k',s,m) for s in SYMS for m in months()]
print('JOBS',len(jobs),flush=True)
with ThreadPoolExecutor(max_workers=32) as ex:
 fs=[]
 for kind,s,x in jobs:fs.append(ex.submit(fetch_metric,s,x) if kind=='m' else ex.submit(fetch_k,s,x))
 for i,f in enumerate(as_completed(fs),1):
  f.result()
  if i%500==0:print('DOWNLOAD',i,'/',len(fs),flush=True)
def parse_ts(s):return int(datetime.datetime.strptime(s,'%Y-%m-%d %H:%M:%S').replace(tzinfo=UTC).timestamp()*1000)
def oi_hourly(sym):
 out={}
 for d in days():
  p=MC/f'{sym}-{d.isoformat()}.zip'
  if not p.exists() or p.stat().st_size==0:continue
  z=zipfile.ZipFile(p);fn=z.namelist()[0]
  for r in csv.DictReader(io.TextIOWrapper(z.open(fn))):
   try:
    t=parse_ts(r['create_time']);val=float(r['sum_open_interest_value'])
    h=(t//3600000)*3600000
    if val>0:out[h]=(t,val)
   except:pass
 return {h:v for h,(t,v) in out.items()}
def k_hourly(sym):
 out={}
 for mo in months():
  p=KC/f'{sym}-{mo}.zip'
  if not p.exists() or p.stat().st_size==0:continue
  z=zipfile.ZipFile(p);fn=z.namelist()[0]
  for r in csv.reader(io.TextIOWrapper(z.open(fn))):
   if not r or not r[0].isdigit():continue
   t=int(r[0]);t=t//1000 if t>10**15 else t
   out[t]=(float(r[1]),float(r[4]),float(r[7]))
 return out
def msd(x):
 m=sum(x)/len(x);return m,(sum((v-m)**2 for v in x)/len(x))**.5
events=[]
for sym in SYMS:
 oi=oi_hourly(sym);k=k_hourly(sym);ts=sorted(set(oi)&set(k))
 series=[]
 for t in ts:
  qv=k[t][2]
  if qv>0 and oi[t]>0:series.append((t,k[t][0],k[t][1],math.log(oi[t]/qv)))
 armed=True;n=0
 for i in range(720,len(series)-13):
  # require 720h approximately continuous; gaps > 2h invalidate
  if series[i][0]-series[i-720][0]>722*3600000:continue
  hist=[x[3] for x in series[i-720:i]];m,sd=msd(hist)
  if sd<=0:continue
  z=(series[i][3]-m)/sd
  if z<1:armed=True
  if not armed or z<3:continue
  # reverse already completed trailing 12h return
  j=i-12
  if j<0:continue
  mom=math.log(series[i][2]/series[j][2])
  if mom==0:continue
  d=-1.0 if mom>0 else 1.0
  entry_i=i+1;entry=series[entry_i][1]
  r1=d*math.log(series[entry_i][2]/entry)
  r4=d*math.log(series[i+4][2]/entry)
  r12=d*math.log(series[i+12][2]/entry)
  events.append((sym,series[i][0],z,mom,r1,r4,r12));armed=False;n+=1
 print('SYMBOL',sym,'hours',len(series),'events',n,flush=True)
def summ(xs):
 sy=sorted(set(x[0] for x in xs));by={};pos=0
 for s in sy:
  q=[x for x in xs if x[0]==s];m=sum(x[6] for x in q)/len(q);by[s]={'n':len(q),'mean12':m};pos+=m>0
 return {'n':len(xs),'mean1':sum(x[4] for x in xs)/len(xs) if xs else None,'mean4':sum(x[5] for x in xs)/len(xs) if xs else None,'mean12':sum(x[6] for x in xs)/len(xs) if xs else None,'win12':sum(x[6]>0 for x in xs)/len(xs) if xs else None,'positive_symbols12':pos,'symbols':len(sy),'by_symbol':by}
res=summ(events);res['passes_gate']=bool(res.get('mean12') and res['mean12']>=0.001 and res['positive_symbols12']>=3)
print(json.dumps(res,indent=2),flush=True)
Path('/tmp/oi_turnover_gate_summary.json').write_text(json.dumps(res,indent=2))
