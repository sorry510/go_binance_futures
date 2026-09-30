import csv,io,zipfile,urllib.request,urllib.parse,urllib.error,datetime,json,math,time,re,xml.etree.ElementTree as ET,subprocess
from pathlib import Path
from functools import lru_cache
from concurrent.futures import ThreadPoolExecutor,as_completed
UTC=datetime.timezone.utc;UA='Mozilla/5.0'
MAP={'BTCUSDT':'Bitcoin','ETHUSDT':'Ethereum','BNBUSDT':'BSC','SOLUSDT':'Solana','AVAXUSDT':'Avalanche','ADAUSDT':'Cardano','NEARUSDT':'Near','TRXUSDT':'Tron','MATICUSDT':'Polygon','FTMUSDT':'Fantom','OPUSDT':'Optimism','APTUSDT':'Aptos','ARBUSDT':'Arbitrum','SUIUSDT':'Sui','XRPUSDT':'Ripple'}
DV='https://data.binance.vision/data/futures/um/monthly/klines';S3='https://s3-ap-northeast-1.amazonaws.com/data.binance.vision'
CACHE=Path('/tmp/chain_dex_residual_cache');CACHE.mkdir(exist_ok=True)
def add2(d):
 try:return d.replace(year=d.year+2)
 except:return d.replace(year=d.year+2,day=28)
def first_kline(sym):
 pref=f'data/futures/um/monthly/klines/{sym}/1d/';u=S3+'?'+urllib.parse.urlencode({'prefix':pref,'max-keys':'5'})
 for k in range(5):
  try:
   root=ET.fromstring(urllib.request.urlopen(u,timeout=20).read());ns={'s':'http://s3.amazonaws.com/doc/2006-03-01/'}
   keys=[x.text for x in root.findall('.//s:Contents/s:Key',ns) if x.text and x.text.endswith('.zip')]
   if not keys:return None
   key=min(keys);b=urllib.request.urlopen(urllib.request.Request('https://data.binance.vision/'+key,headers={'User-Agent':UA}),timeout=20).read()
   z=zipfile.ZipFile(io.BytesIO(b));fn=z.namelist()[0]
   for r in csv.reader(io.TextIOWrapper(z.open(fn))):
    if r and r[0].isdigit():
     t=int(r[0]);t=t//1000 if t>10**15 else t;return datetime.datetime.fromtimestamp(t/1000,UTC)
  except Exception:time.sleep(.4*(k+1))
 return None
def months():
 out=[];d=datetime.date(2022,9,1);end=datetime.date(2026,9,1)
 while d<=end:
  out.append(d.strftime('%Y-%m'));d=(d.replace(day=28)+datetime.timedelta(days=4)).replace(day=1)
 return out
def fetch_k(sym,mo):
 p=CACHE/f'{sym}-{mo}.zip'
 if p.exists():return p
 u=f'{DV}/{sym}/1d/{sym}-1d-{mo}.zip'
 for k in range(4):
  try:b=urllib.request.urlopen(urllib.request.Request(u,headers={'User-Agent':UA}),timeout=15).read();p.write_bytes(b);return p
  except urllib.error.HTTPError as e:
   if e.code==404:p.write_bytes(b'');return p
   time.sleep(.3*(k+1))
  except Exception:time.sleep(.3*(k+1))
 p.write_bytes(b'');return p
jobs=[(s,m) for s in MAP for m in months()]
print('PRICE_FILES',len(jobs),flush=True)
with ThreadPoolExecutor(max_workers=20) as ex:
 fs=[ex.submit(fetch_k,*j) for j in jobs]
 for i,f in enumerate(as_completed(fs),1):
  f.result()
  if i%150==0 or i==len(fs):print('PRICE_DOWNLOAD',i,'/',len(fs),flush=True)
@lru_cache(None)
def price_daily(sym):
 out={}
 for mo in months():
  p=fetch_k(sym,mo)
  if not p.exists() or p.stat().st_size==0:continue
  try:z=zipfile.ZipFile(p)
  except:continue
  fn=z.namelist()[0]
  for r in csv.reader(io.TextIOWrapper(z.open(fn))):
   if not r or not r[0].isdigit():continue
   t=int(r[0]);t=t//1000 if t>10**15 else t;d=datetime.datetime.fromtimestamp(t/1000,UTC).date()
   out[d]=(float(r[1]),float(r[4]))
 return out
def dex_daily(chain):
 p=Path('/tmp/llama_dex_'+chain.replace(' ','_')+'.json')
 if not p.exists() or p.stat().st_size<100:
  u='https://api.llama.fi/overview/dexs/'+urllib.parse.quote(chain)+'?excludeTotalDataChartBreakdown=true&dataType=dailyVolume'
  q=subprocess.run(['curl','--http1.1','-L','--retry','2','--retry-all-errors','--connect-timeout','8','--max-time','30','-sS','-o',str(p),u])
 try:o=json.load(open(p))
 except:return {}
 out={}
 for x in o.get('totalDataChart') or []:
  try:
   d=datetime.datetime.fromtimestamp(int(x[0]),UTC).date();v=float(x[1])
   if v>0:out[d]=v
  except:pass
 return out
def beta(xs,ys):
 mx=sum(xs)/len(xs);my=sum(ys)/len(ys);den=sum((x-mx)**2 for x in xs)
 return 0.0 if den<=0 else sum((x-mx)*(y-my) for x,y in zip(xs,ys))/den
events=[];meta={}
for sym,chain in MAP.items():
 first=first_kline(sym)
 if not first:print('NO_FIRST',sym);continue
 eligible=add2(first).date();px=price_daily(sym);dv=dex_daily(chain);dates=sorted(set(px)&set(dv));rows=[]
 for d in dates:
  o,c=px[d];v=dv[d]
  if o>0 and c>0 and v>0:rows.append((d,o,c,v))
 residual=[None]*len(rows);flow=[None]*len(rows)
 for i in range(91,len(rows)):
  if (rows[i][0]-rows[i-1][0]).days!=1:continue
  xp=[];yv=[];ok=True
  for j in range(i-90,i):
   if j<=0 or (rows[j][0]-rows[j-1][0]).days!=1:ok=False;break
   xp.append(math.log(rows[j][2]/rows[j-1][2]));yv.append(math.log(rows[j][3]/rows[j-1][3]))
  if not ok:continue
  b=beta(xp,yv);rp=math.log(rows[i][2]/rows[i-1][2]);rv=math.log(rows[i][3]/rows[i-1][3]);residual[i]=rv-b*rp
  if i>=6 and all(residual[j] is not None for j in range(i-6,i+1)):flow[i]=sum(residual[j] for j in range(i-6,i+1))
 n=0
 for i in range(92,len(rows)-8):
  if rows[i][0]<eligible or flow[i] is None or flow[i-1] is None:continue
  direction=0.0
  if flow[i-1]<=0 and flow[i]>0:direction=1
  elif flow[i-1]>=0 and flow[i]<0:direction=-1
  else:continue
  if (rows[i+1][0]-rows[i][0]).days!=1:continue
  entry=rows[i+1][1]
  if entry<=0:continue
  events.append({'symbol':sym,'chain':chain,'signal_date':rows[i][0].isoformat(),'year':rows[i][0].year,'direction':'LONG' if direction>0 else 'SHORT','flow7':flow[i],'r1':direction*math.log(rows[i+1][2]/entry),'r3':direction*math.log(rows[i+3][2]/entry),'r7':direction*math.log(rows[i+7][2]/entry)})
  n+=1
 meta[sym]={'chain':chain,'first_futures':first.isoformat(),'eligible_from':eligible.isoformat(),'aligned_days':len(rows),'events':n}
 print('SYMBOL',sym,chain,'eligible',eligible,'days',len(rows),'events',n,flush=True)
def summary(es):
 if not es:return {'n':0}
 sy=sorted(set(x['symbol'] for x in es));by={};pos=0
 for s in sy:
  q=[x for x in es if x['symbol']==s];m=sum(x['r7'] for x in q)/len(q);by[s]={'n':len(q),'mean7':m,'win7':sum(x['r7']>0 for x in q)/len(q)}
  if m>0:pos+=1
 return {'n':len(es),'mean1':sum(x['r1'] for x in es)/len(es),'mean3':sum(x['r3'] for x in es)/len(es),'mean7':sum(x['r7'] for x in es)/len(es),'win7':sum(x['r7']>0 for x in es)/len(es),'positive_symbols7':pos,'symbols':len(sy),'by_symbol':by}
res={'meta':meta,'discovery_2023_2024':summary([x for x in events if x['year'] in (2023,2024)]),'oos1_2025':summary([x for x in events if x['year']==2025]),'oos2_2026':summary([x for x in events if x['year']==2026]),'all':summary(events)}
print('SUMMARY',json.dumps(res,indent=2),flush=True)
Path('/tmp/chain_dex_residual_events.json').write_text(json.dumps(events,indent=2));Path('/tmp/chain_dex_residual_summary.json').write_text(json.dumps(res,indent=2))
