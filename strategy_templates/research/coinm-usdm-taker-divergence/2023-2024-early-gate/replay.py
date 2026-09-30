import csv,io,zipfile,datetime,math,json
from pathlib import Path
BASES=['BTC','ETH','BNB','XRP']; MC=Path('/tmp/oi_metrics_cache'); KC=Path('/tmp/term_structure_cache')
START=datetime.date(2023,1,1);END=datetime.date(2024,12,31)
def dates():
 d=START;out=[]
 while d<=END:out.append(d);d+=datetime.timedelta(days=1)
 return out
def daily_imbalance(p):
 if not p.exists() or p.stat().st_size==0:return None
 z=zipfile.ZipFile(p);fn=z.namelist()[0];vals=[]
 for r in csv.DictReader(io.TextIOWrapper(z.open(fn))):
  try:
   ratio=float(r['sum_taker_long_short_vol_ratio'])
   if ratio>0:vals.append((ratio-1)/(ratio+1))
  except:pass
 return sum(vals)/len(vals) if vals else None
def read_month(kind,sym,mo):
 p=KC/f'{kind}-{sym}-{mo}.zip'
 if not p.exists() or p.stat().st_size==0:return {}
 z=zipfile.ZipFile(p);fn=z.namelist()[0];out={}
 for r in csv.reader(io.TextIOWrapper(z.open(fn))):
  if not r or not r[0].isdigit():continue
  t=int(r[0]);t=t//1000 if t>10**15 else t
  out[t]=float(r[4])
 return out
def msd(x):
 m=sum(x)/len(x);return m,(sum((v-m)**2 for v in x)/len(x))**.5
all_events=[]
for b in BASES:
 series=[];months={}
 for d in dates():
  u=daily_imbalance(MC/f'um-{b}USDT-{d.isoformat()}.zip');c=daily_imbalance(MC/f'cm-{b}USD_PERP-{d.isoformat()}.zip')
  if u is None or c is None:continue
  mo=d.strftime('%Y-%m')
  if mo not in months:months[mo]=read_month('um',b+'USDT',mo)
  t=int(datetime.datetime(d.year,d.month,d.day,23,tzinfo=datetime.timezone.utc).timestamp()*1000)
  if t not in months[mo]:continue
  series.append((d,months[mo][t],c-u))
 armed=True;ev=[]
 for i in range(30,len(series)-7):
  hist=[x[2] for x in series[i-30:i]];m,sd=msd(hist)
  if sd<=0:continue
  z=(series[i][2]-m)/sd
  if abs(z)<1:armed=True
  if not armed or abs(z)<2:continue
  d=1 if z>0 else -1;entry=series[i][1]
  r1=d*math.log(series[i+1][1]/entry);r3=d*math.log(series[i+3][1]/entry);r7=d*math.log(series[i+7][1]/entry)
  ev.append((b,series[i][0].isoformat(),z,r1,r3,r7));armed=False
 all_events.extend(ev);print('SYMBOL',b,'days',len(series),'events',len(ev))
def summ(xs):
 if not xs:return {}
 sy=sorted(set(x[0] for x in xs));by={};pos=0
 for s in sy:
  q=[x for x in xs if x[0]==s];m=sum(x[5] for x in q)/len(q);by[s]={'n':len(q),'mean7':m};pos+=m>0
 return {'n':len(xs),'mean1':sum(x[3] for x in xs)/len(xs),'mean3':sum(x[4] for x in xs)/len(xs),'mean7':sum(x[5] for x in xs)/len(xs),'win7':sum(x[5]>0 for x in xs)/len(xs),'positive_symbols7':pos,'symbols':len(sy),'by_symbol':by}
res=summ(all_events);print(json.dumps(res,indent=2));Path('/tmp/coinm_usdm_taker_divergence_summary.json').write_text(json.dumps(res,indent=2))
