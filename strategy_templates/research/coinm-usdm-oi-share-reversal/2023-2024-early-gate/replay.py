import csv,io,zipfile,urllib.request,urllib.error,time,datetime,math,json
from concurrent.futures import ThreadPoolExecutor,as_completed
from pathlib import Path
UTC=datetime.timezone.utc
BASES=['BTC','ETH','BNB','XRP']
START=datetime.date(2023,1,1); END=datetime.date(2024,12,31)
UA='Mozilla/5.0'; MC=Path('/tmp/oi_metrics_cache'); MC.mkdir(exist_ok=True)
KC=Path('/tmp/term_structure_cache')
def dates():
 d=START; out=[]
 while d<=END:out.append(d);d+=datetime.timedelta(days=1)
 return out
def fetch(kind,sym,d):
 p=MC/f'{kind}-{sym}-{d.isoformat()}.zip'
 if p.exists():return p
 u=f'https://data.binance.vision/data/futures/{kind}/daily/metrics/{sym}/{sym}-metrics-{d.isoformat()}.zip'
 for k in range(4):
  try:
   b=urllib.request.urlopen(urllib.request.Request(u,headers={'User-Agent':UA}),timeout=20).read();p.write_bytes(b);return p
  except urllib.error.HTTPError as e:
   if e.code==404:p.write_bytes(b'');return p
   time.sleep(.25*(k+1))
  except Exception:time.sleep(.25*(k+1))
 p.write_bytes(b'');return p
def last_metric(p):
 if not p.exists() or p.stat().st_size==0:return None
 z=zipfile.ZipFile(p);fn=z.namelist()[0];rows=list(csv.DictReader(io.TextIOWrapper(z.open(fn))))
 if not rows:return None
 r=rows[-1]
 try:return float(r['sum_open_interest_value'])
 except:return None
jobs=[]
for b in BASES:
 for d in dates():
  jobs.append(('um',b+'USDT',d));jobs.append(('cm',b+'USD_PERP',d))
print('JOBS',len(jobs),flush=True)
with ThreadPoolExecutor(max_workers=32) as ex:
 fs=[ex.submit(fetch,*j) for j in jobs]
 for i,f in enumerate(as_completed(fs),1):
  f.result()
  if i%1000==0:print('DOWNLOAD',i,'/',len(jobs),flush=True)
def read_month(kind,sym,mo):
 p=KC/f'{kind}-{sym}-{mo}.zip'
 if not p.exists() or p.stat().st_size==0:return {}
 z=zipfile.ZipFile(p);fn=z.namelist()[0];out={}
 for r in csv.reader(io.TextIOWrapper(z.open(fn))):
  if not r or not r[0].isdigit():continue
  t=int(r[0]);t=t//1000 if t>10**15 else t
  out[t]=(float(r[1]),float(r[4]))
 return out
def msd(x):
 m=sum(x)/len(x);return m,(sum((v-m)**2 for v in x)/len(x))**.5
all_events=[]
for b in BASES:
 series=[]; month_cache={}
 for d in dates():
  uv=last_metric(MC/f'um-{b}USDT-{d.isoformat()}.zip');cv=last_metric(MC/f'cm-{b}USD_PERP-{d.isoformat()}.zip')
  if uv is None or cv is None or uv<=0 or cv<=0:continue
  mo=d.strftime('%Y-%m')
  if mo not in month_cache:
   month_cache[mo]=(read_month('um',b+'USDT',mo),read_month('cm',b+'USD_PERP',mo))
  um,cm=month_cache[mo]
  end=datetime.datetime(d.year,d.month,d.day,23,tzinfo=UTC);t=int(end.timestamp()*1000)
  if t not in um or t not in cm:continue
  cm_usd=cv*cm[t][1]
  share=cm_usd/(cm_usd+uv)
  series.append((d,um[t][1],share))
 armed=True;ev=[]
 for i in range(30,len(series)-7):
  hist=[x[2] for x in series[i-30:i]];m,sd=msd(hist)
  if sd<=0:continue
  z=(series[i][2]-m)/sd
  if z<1:armed=True
  if not armed or z<2:continue
  mom=math.log(series[i][1]/series[i-7][1])
  if mom==0:continue
  direction=-1 if mom>0 else 1
  entry=series[i][1]
  r1=direction*math.log(series[i+1][1]/entry)
  r3=direction*math.log(series[i+3][1]/entry)
  r7=direction*math.log(series[i+7][1]/entry)
  ev.append((b,series[i][0].isoformat(),z,r1,r3,r7));armed=False
 all_events.extend(ev);print('SYMBOL',b,'days',len(series),'events',len(ev),flush=True)
def summ(xs):
 if not xs:return {}
 sy=sorted(set(x[0] for x in xs));by={};pos=0
 for s in sy:
  q=[x for x in xs if x[0]==s];m=sum(x[5] for x in q)/len(q);by[s]={'n':len(q),'mean7':m};pos+=m>0
 return {'n':len(xs),'mean1':sum(x[3] for x in xs)/len(xs),'mean3':sum(x[4] for x in xs)/len(xs),'mean7':sum(x[5] for x in xs)/len(xs),'win7':sum(x[5]>0 for x in xs)/len(xs),'positive_symbols7':pos,'symbols':len(sy),'by_symbol':by}
res=summ(all_events);print(json.dumps(res,indent=2),flush=True)
Path('/tmp/coinm_oi_share_gate_summary.json').write_text(json.dumps(res,indent=2))
