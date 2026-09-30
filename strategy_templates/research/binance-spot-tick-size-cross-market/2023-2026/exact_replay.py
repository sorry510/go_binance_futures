import csv,datetime,io,json,time,urllib.request,urllib.error,zipfile
from concurrent.futures import ThreadPoolExecutor,as_completed
from functools import lru_cache
from pathlib import Path
UTC=datetime.timezone.utc
BASE='https://data.binance.vision/data/futures/um/monthly'
UA='Mozilla/5.0'
ALL=json.load(open('/tmp/spot_tick_size_eligibility.json'))
events=[x for x in ALL if x.get('eligible')]
CACHE=Path('/tmp/spot_tick_exact_cache');CACHE.mkdir(exist_ok=True)
def month(dt):return dt.strftime('%Y-%m')
def next_month(dt):
 if dt.month==12:return dt.replace(year=dt.year+1,month=1,day=1)
 return dt.replace(month=dt.month+1,day=1)
def get_bytes(url,key):
 p=CACHE/key
 if p.exists():return p.read_bytes() or None
 for k in range(5):
  try:
   b=urllib.request.urlopen(urllib.request.Request(url,headers={'User-Agent':UA}),timeout=30).read();p.write_bytes(b);return b
  except urllib.error.HTTPError as e:
   if e.code==404:p.write_bytes(b'');return None
   time.sleep(.5*(k+1))
  except Exception:time.sleep(.5*(k+1))
 p.write_bytes(b'');return None
def fetch_pair(sym,m):
 get_bytes(f'{BASE}/klines/{sym}/1m/{sym}-1m-{m}.zip',f'k-{sym}-{m}.zip')
 get_bytes(f'{BASE}/fundingRate/{sym}/{sym}-fundingRate-{m}.zip',f'f-{sym}-{m}.zip')
jobs=set()
for e in events:
 d=datetime.datetime.fromtimestamp(e['effective_ms']/1000,UTC)
 end=d+datetime.timedelta(hours=73)
 cur=d.replace(day=1,hour=0,minute=0,second=0,microsecond=0)
 last=end.replace(day=1,hour=0,minute=0,second=0,microsecond=0)
 while cur<=last:
  jobs.add((e['symbol'],month(cur)));cur=next_month(cur)
print('FILES',len(jobs),flush=True)
with ThreadPoolExecutor(max_workers=8) as ex:
 fs=[ex.submit(fetch_pair,*j) for j in jobs]
 for i,f in enumerate(as_completed(fs),1):
  f.result()
  if i%5==0 or i==len(fs):print('DOWNLOAD',i,'/',len(fs),flush=True)
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
 names=[str(x).strip().lower() for x in (hdr or [])]
 ti=names.index('calc_time') if 'calc_time' in names else 0
 ri=names.index('last_funding_rate') if 'last_funding_rate' in names else 2
 mi=names.index('mark_price') if 'mark_price' in names else None
 out=[]
 for r in rows:
  try:
   t=int(r[ti]);t=t//1000 if t>10**15 else t
   rate=float(r[ri]);mark=float(r[mi]) if mi is not None and mi<len(r) and r[mi] else 0.0
   out.append((t,rate,mark))
  except:pass
 return out
def slip(p,side,enter):
 if (side=='LONG' and enter) or (side=='SHORT' and not enter):return p*1.0005
 return p*0.9995
def roi(entry,p,side):
 return ((p-entry)/p if side=='LONG' else (entry-p)/p)*4*100
def replay(e):
 sym=e['symbol'];side=e['direction'];ev=e['effective_ms'];horizon=ev+72*3600*1000
 sd=datetime.datetime.fromtimestamp(ev/1000,UTC);ed=datetime.datetime.fromtimestamp((horizon+120000)/1000,UTC)
 cur=sd.replace(day=1,hour=0,minute=0,second=0,microsecond=0);last=ed.replace(day=1,hour=0,minute=0,second=0,microsecond=0);months=[]
 while cur<=last:months.append(month(cur));cur=next_month(cur)
 bars=[]
 for m in months:bars.extend(klines(sym,m))
 bars.sort();target=(ev//60000+1)*60000
 idx=next((i for i,b in enumerate(bars) if b[0]>=target),None)
 if idx is None:return e|{'error':'no_entry'}
 ent_t,ent_o,_=bars[idx]
 if ent_t>ev+5*60000:return e|{'error':'entry_gap','entry_ts':ent_t}
 entry=slip(ent_o,side,True);qty=4.0/entry;open_fee=qty*entry*0.0005
 funds=[]
 for m in months:funds.extend(funding(sym,m))
 funds.sort();fi=0
 while fi<len(funds) and funds[fi][0]<ent_t:fi+=1
 fund=0.0;trigger=None;exit_idx=None;last_i=idx
 for i in range(idx,len(bars)):
  t,op,cl=bars[i]
  if t>horizon+60000:break
  last_i=i
  while fi<len(funds) and funds[fi][0]<=t+59999:
   ft,rate,mark=funds[fi]
   if ent_t<=ft<=horizon:
    notional=qty*(mark if mark>0 else cl)
    fund += (-notional*rate if side=='LONG' else notional*rate)
   fi+=1
  if t>=horizon:trigger='TIME';exit_idx=i;break
  rr=roi(entry,cl,side)
  if rr<=-6:trigger='SL';exit_idx=min(i+1,len(bars)-1);break
  if rr>=8:trigger='TP';exit_idx=min(i+1,len(bars)-1);break
 if exit_idx is None:trigger='EOF';exit_idx=last_i
 xt,xo,xc=bars[exit_idx];raw=xo if trigger in ('TP','SL','TIME') else xc;exitp=slip(raw,side,False)
 gross=((exitp-entry) if side=='LONG' else (entry-exitp))*qty
 close_fee=qty*exitp*0.0005;net=gross-open_fee-close_fee+fund
 return e|{'entry_ts':ent_t,'exit_ts':xt,'trigger':trigger,'entry':entry,'exit':exitp,'gross_pct':gross*100,'fees_pct':(open_fee+close_fee)*100,'funding_pct':fund*100,'net_pct':net*100,'hold_h':(xt-ent_t)/3600000}
def summary(xs):
 if not xs:return {}
 gp=sum(max(x['net_pct'],0) for x in xs);gl=-sum(min(x['net_pct'],0) for x in xs)
 return {'n':len(xs),'wins':sum(x['net_pct']>0 for x in xs),'losses':sum(x['net_pct']<=0 for x in xs),'tp':sum(x['trigger']=='TP' for x in xs),'sl':sum(x['trigger']=='SL' for x in xs),'time':sum(x['trigger']=='TIME' for x in xs),'pf':gp/gl if gl else None,'net_pct':sum(x['net_pct'] for x in xs),'avg_pct':sum(x['net_pct'] for x in xs)/len(xs)}
res=[]
for i,e in enumerate(events,1):
 x=replay(e);res.append(x);print('TRADE',i,'/',len(events),x['symbol'],x['direction'],x.get('trigger'),round(x.get('net_pct',0),4),flush=True)
good=[x for x in res if not x.get('error')]
out={'all':summary(good)}
for side in ['LONG','SHORT']:out[side]=summary([x for x in good if x['direction']==side])
for yr in [2023,2024]:out[str(yr)]=summary([x for x in good if datetime.datetime.fromtimestamp(x['effective_ms']/1000,UTC).year==yr])
batches={}
for x in good:batches.setdefault(x['effective_utc'],[]).append(x)
bm=[]
for b,q in sorted(batches.items()):
 s=summary(q);bm.append(sum(x['net_pct'] for x in q)/len(q));print('BATCH',b,json.dumps(s),flush=True)
out['batch_equal']={'n':len(bm),'mean_net_pct':sum(bm)/len(bm) if bm else None,'positive_batches':sum(x>0 for x in bm)}
print('SUMMARY',json.dumps(out,indent=2),flush=True)
Path('/tmp/spot_tick_exact_trades.json').write_text(json.dumps(res,ensure_ascii=False,indent=2))
Path('/tmp/spot_tick_exact_summary.json').write_text(json.dumps(out,indent=2))
