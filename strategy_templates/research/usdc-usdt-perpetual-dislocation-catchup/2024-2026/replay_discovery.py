import csv,io,zipfile,urllib.request,urllib.error,time,datetime,json,math,statistics
from concurrent.futures import ThreadPoolExecutor,as_completed
from pathlib import Path
UTC=datetime.timezone.utc;UA='Mozilla/5.0'
CANDS=json.load(open('/tmp/usdc_usdt_candidates.json'))
BASE='https://data.binance.vision/data/futures/um/monthly/klines'
CACHE=Path('/tmp/usdc_usdt_1h_cache');CACHE.mkdir(exist_ok=True)
START=datetime.date(2023,12,1);END=datetime.date(2025,12,1)
def months(a,b):
 y,m=a.year,a.month;out=[]
 while (y,m)<=(b.year,b.month):
  out.append(f'{y:04d}-{m:02d}');m+=1
  if m==13:y+=1;m=1
 return out
MOS=months(START,END)
def fetch(sym,mo):
 p=CACHE/f'{sym}-{mo}.zip'
 if p.exists():return p
 u=f'{BASE}/{sym}/1h/{sym}-1h-{mo}.zip'
 for k in range(5):
  try:
   b=urllib.request.urlopen(urllib.request.Request(u,headers={'User-Agent':UA}),timeout=20).read();p.write_bytes(b);return p
  except urllib.error.HTTPError as e:
   if e.code==404:p.write_bytes(b'');return p
   time.sleep(.4*(k+1))
  except Exception:time.sleep(.4*(k+1))
 p.write_bytes(b'');return p
jobs={(s,mo) for x in CANDS for s in (x['usdc'],x['usdt']) for mo in MOS}
print('FILES',len(jobs),flush=True)
with ThreadPoolExecutor(max_workers=24) as ex:
 fs=[ex.submit(fetch,*j) for j in jobs]
 for i,f in enumerate(as_completed(fs),1):
  f.result()
  if i%150==0 or i==len(fs):print('DOWNLOAD',i,'/',len(fs),flush=True)
def read(sym):
 out={}
 for mo in MOS:
  p=CACHE/f'{sym}-{mo}.zip'
  if not p.exists() or p.stat().st_size==0:continue
  z=zipfile.ZipFile(p);fn=z.namelist()[0]
  for r in csv.reader(io.TextIOWrapper(z.open(fn))):
   if not r or not r[0].isdigit():continue
   t=int(r[0]);t=t//1000 if t>10**15 else t
   out[t]=(float(r[1]),float(r[4]))
  z.close()
 return out
def month_start_ms(s):
 y,m=map(int,s.split('-'))
 return int(datetime.datetime(y,m,1,tzinfo=UTC).timestamp()*1000)
events=[]
for x in CANDS:
 a=read(x['usdc']);b=read(x['usdt']);ts=sorted(set(a)&set(b))
 if len(ts)<744:continue
 # target USDT production age gate; also require 720h of actual aligned USDC/USDT history.
 age=month_start_ms(x['target_2y_month'])
 prem=[];valid_ts=[]
 for t in ts:
  if a[t][1]>0 and b[t][1]>0:
   prem.append(math.log(a[t][1]/b[t][1]));valid_ts.append(t)
 if len(valid_ts)<744:continue
 armed=True;nev=0
 for i in range(720,len(valid_ts)-12):
  t=valid_ts[i]
  if t<age or t<int(datetime.datetime(2024,1,1,tzinfo=UTC).timestamp()*1000):continue
  if t>=int(datetime.datetime(2026,1,1,tzinfo=UTC).timestamp()*1000):break
  # require continuous 720h lookback and 12h forward alignment.
  if valid_ts[i]-valid_ts[i-720]>721*3600000 or valid_ts[i+12]-valid_ts[i]>13*3600000:continue
  h=prem[i-720:i];mu=sum(h)/720;sd=(sum((v-mu)**2 for v in h)/720)**.5
  if sd<=0:continue
  z=(prem[i]-mu)/sd
  if abs(z)<1:armed=True
  if not armed or abs(z)<3:continue
  # catch-up target: USDC rich => USDT LONG; USDC cheap => USDT SHORT
  d=1.0 if z>0 else -1.0
  ent_t=valid_ts[i+1];entry=b[ent_t][0]
  if entry<=0:continue
  r1=d*math.log(b[valid_ts[i+1]][1]/entry)
  r4=d*math.log(b[valid_ts[i+4]][1]/entry)
  r12=d*math.log(b[valid_ts[i+12]][1]/entry)
  y=datetime.datetime.fromtimestamp(t/1000,UTC).year
  events.append({'symbol':x['usdt'],'usdc':x['usdc'],'year':y,'t':t,'z':z,'r1':r1,'r4':r4,'r12':r12})
  armed=False;nev+=1
 print('SYMBOL',x['usdt'],'aligned',len(valid_ts),'events',nev,flush=True)
def summary(xs):
 if not xs:return {}
 by={}
 for s in sorted(set(x['symbol'] for x in xs)):
  q=[x for x in xs if x['symbol']==s];by[s]={'n':len(q),'mean12':sum(x['r12'] for x in q)/len(q)}
 return {'n':len(xs),'mean1':sum(x['r1'] for x in xs)/len(xs),'mean4':sum(x['r4'] for x in xs)/len(xs),'mean12':sum(x['r12'] for x in xs)/len(xs),'win12':sum(x['r12']>0 for x in xs)/len(xs),'symbols':len(by),'positive_symbols12':sum(v['mean12']>0 for v in by.values()),'by_symbol':by}
res={'discovery_2024_2025':summary(events),'year_2024':summary([x for x in events if x['year']==2024]),'year_2025':summary([x for x in events if x['year']==2025])}
print(json.dumps(res,indent=2),flush=True)
Path('/tmp/usdc_usdt_dislocation_events.json').write_text(json.dumps(events,indent=2))
Path('/tmp/usdc_usdt_dislocation_summary.json').write_text(json.dumps(res,indent=2))
