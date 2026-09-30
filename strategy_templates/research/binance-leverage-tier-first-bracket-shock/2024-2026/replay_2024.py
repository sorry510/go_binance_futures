import csv,datetime,io,json,time,urllib.request,urllib.error,zipfile
from functools import lru_cache
from pathlib import Path
UTC=datetime.timezone.utc;UA='Mozilla/5.0';BASE='https://data.binance.vision/data/futures/um/monthly'
events=[x for x in json.load(open('/tmp/leverage_tier_eligible.json')) if x.get('eligible')]
CACHE=Path('/tmp/leverage_tier_cache');CACHE.mkdir(exist_ok=True)
def get_bytes(url,key):
 p=CACHE/key
 if p.exists():return p.read_bytes() or None
 last=None
 for k in range(5):
  try:
   b=urllib.request.urlopen(urllib.request.Request(url,headers={'User-Agent':UA}),timeout=30).read();p.write_bytes(b);return b
  except urllib.error.HTTPError as e:
   if e.code==404:p.write_bytes(b'');return None
   last=e
  except Exception as e:last=e
  time.sleep(.5*(k+1))
 raise last
def mon(dt):return dt.strftime('%Y-%m')
def next_month(dt):return (dt.replace(day=28)+datetime.timedelta(days=4)).replace(day=1,hour=0,minute=0,second=0,microsecond=0)
@lru_cache(None)
def klines(sym,m):
 b=get_bytes(f'{BASE}/klines/{sym}/1m/{sym}-1m-{m}.zip',f'k-{sym}-{m}.zip')
 if not b:return []
 z=zipfile.ZipFile(io.BytesIO(b));fn=z.namelist()[0];out=[]
 for r in csv.reader(io.TextIOWrapper(z.open(fn))):
  if not r or not r[0].isdigit():continue
  t=int(r[0]);t=t//1000 if t>10**15 else t
  out.append((t,float(r[1]),float(r[4])))
 return out
@lru_cache(None)
def funding(sym,m):
 b=get_bytes(f'{BASE}/fundingRate/{sym}/{sym}-fundingRate-{m}.zip',f'f-{sym}-{m}.zip')
 if not b:return []
 z=zipfile.ZipFile(io.BytesIO(b));fn=z.namelist()[0];rows=csv.reader(io.TextIOWrapper(z.open(fn)));hdr=next(rows,None)
 names=[str(x).strip().lower() for x in (hdr or [])];ti=names.index('calc_time') if 'calc_time' in names else 0;ri=names.index('last_funding_rate') if 'last_funding_rate' in names else 2;mi=names.index('mark_price') if 'mark_price' in names else None
 out=[]
 for r in rows:
  try:
   t=int(r[ti]);t=t//1000 if t>10**15 else t;rate=float(r[ri]);mark=float(r[mi]) if mi is not None and mi<len(r) and r[mi] else 0.0;out.append((t,rate,mark))
  except:pass
 return out
def replay(e):
 sym=e['symbol'];ev=e['event_ms'];side=e['side'];horizon=ev+24*3600*1000
 d0=datetime.datetime.fromtimestamp(ev/1000,UTC);d1=datetime.datetime.fromtimestamp((horizon+120000)/1000,UTC)
 months=[];d=d0.replace(day=1,hour=0,minute=0,second=0,microsecond=0);last=d1.replace(day=1,hour=0,minute=0,second=0,microsecond=0)
 while d<=last:months.append(mon(d));d=next_month(d)
 bars=[]
 for m in months:bars.extend(klines(sym,m))
 bars.sort();target=(ev//60000+1)*60000
 idx=next((i for i,b in enumerate(bars) if b[0]>=target),None)
 if idx is None or bars[idx][0]>target+5*60000:return {**e,'error':'entry_gap'}
 ent_t,ent_o,_=bars[idx];entry=ent_o*(1.0005 if side=='LONG' else 0.9995);qty=4.0/entry;open_fee=qty*entry*0.0005
 fs=[]
 for m in months:fs.extend(funding(sym,m))
 fs.sort();fi=0
 while fi<len(fs) and fs[fi][0]<ent_t:fi+=1
 fpnl=0.0;exit_i=None;reason='TIME';end=ent_t+24*3600*1000
 for i in range(idx,len(bars)):
  t,o,c=bars[i]
  if t>end:break
  while fi<len(fs) and fs[fi][0]<=t+59999:
   ft,rate,mark=fs[fi]
   if ent_t<=ft<=end:
    px=mark if mark>0 else c;fpnl+=(-1 if side=='LONG' else 1)*qty*px*rate
   fi+=1
  roi=((c-entry)/c if side=='LONG' else (entry-c)/c)*4*100
  if roi<=-6:reason='SL';exit_i=min(i+1,len(bars)-1);break
  if roi>=8:reason='TP';exit_i=min(i+1,len(bars)-1);break
  if t>=end:reason='TIME';exit_i=min(i+1,len(bars)-1);break
 if exit_i is None:exit_i=min(idx+24*60,len(bars)-1)
 xt,xo,xc=bars[exit_i];exitp=xo*(0.9995 if side=='LONG' else 1.0005)
 gross=((exitp-entry) if side=='LONG' else (entry-exitp))*qty;close_fee=qty*exitp*0.0005;net=gross-open_fee-close_fee+fpnl
 return {**e,'entry_time':ent_t,'exit_time':xt,'entry':entry,'exit':exitp,'trigger':reason,'gross_pct':gross*100,'fees_pct':(open_fee+close_fee)*100,'funding_pct':fpnl*100,'net_pct':net*100}
res=[]
for i,e in enumerate(events,1):
 x=replay(e);res.append(x);print(i,len(events),x['symbol'],x['side'],x.get('trigger'),round(x.get('net_pct',0),4),flush=True)
good=[x for x in res if 'error' not in x];gp=sum(max(x['net_pct'],0) for x in good);gl=-sum(min(x['net_pct'],0) for x in good)
summary={'n':len(good),'wins':sum(x['net_pct']>0 for x in good),'losses':sum(x['net_pct']<=0 for x in good),'tp':sum(x['trigger']=='TP' for x in good),'sl':sum(x['trigger']=='SL' for x in good),'time':sum(x['trigger']=='TIME' for x in good),'pf':gp/gl if gl else None,'net_pct':sum(x['net_pct'] for x in good),'avg_pct':sum(x['net_pct'] for x in good)/len(good),'gp':gp,'gl':gl}
print(json.dumps(summary,indent=2),flush=True)
Path('/tmp/leverage_tier_exact_trades.json').write_text(json.dumps(res,indent=2));Path('/tmp/leverage_tier_exact_summary.json').write_text(json.dumps(summary,indent=2))
