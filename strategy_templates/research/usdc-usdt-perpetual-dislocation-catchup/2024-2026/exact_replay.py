import csv,datetime,io,json,math,time,urllib.request,urllib.error,zipfile
from concurrent.futures import ThreadPoolExecutor,as_completed
from functools import lru_cache
from pathlib import Path
UTC=datetime.timezone.utc;UA='Mozilla/5.0'
SRC=[('/tmp/usdc_usdt_dislocation_events.json','discovery'),('/tmp/usdc_usdt_oos_2026_events.json','oos')]
events=[]
for fn,period in SRC:
 for x in json.load(open(fn)):
  y=dict(x);y['period']=period;y['side']='LONG' if y['z']>0 else 'SHORT';y['entry_target']=y['t']+3600000;events.append(y)
events.sort(key=lambda x:(x['symbol'],x['entry_target']))
CACHE=Path('/tmp/usdc_usdt_exact_cache');CACHE.mkdir(exist_ok=True)
KBASE='https://data.binance.vision/data/futures/um/daily/klines'
FBASE='https://data.binance.vision/data/futures/um/monthly/fundingRate'
def daystr(ms):return datetime.datetime.fromtimestamp(ms/1000,UTC).strftime('%Y-%m-%d')
def monthstr(ms):return datetime.datetime.fromtimestamp(ms/1000,UTC).strftime('%Y-%m')
def nextday(ds):return (datetime.date.fromisoformat(ds)+datetime.timedelta(days=1)).isoformat()
def fetch_url(url,p):
 if p.exists():return p
 for k in range(5):
  try:
   b=urllib.request.urlopen(urllib.request.Request(url,headers={'User-Agent':UA}),timeout=25).read();p.write_bytes(b);return p
  except urllib.error.HTTPError as e:
   if e.code==404:p.write_bytes(b'');return p
   time.sleep(.4*(k+1))
  except Exception:time.sleep(.4*(k+1))
 p.write_bytes(b'');return p
def fetch_job(j):
 kind,sym,key=j
 if kind=='k':
  p=CACHE/f'k-{sym}-{key}.zip';u=f'{KBASE}/{sym}/1m/{sym}-1m-{key}.zip'
 else:
  p=CACHE/f'f-{sym}-{key}.zip';u=f'{FBASE}/{sym}/{sym}-fundingRate-{key}.zip'
 fetch_url(u,p);return j
jobs=set()
for e in events:
 ds=daystr(e['entry_target']);nd=nextday(ds);jobs.add(('k',e['symbol'],ds));jobs.add(('k',e['symbol'],nd))
 jobs.add(('f',e['symbol'],monthstr(e['entry_target'])));jobs.add(('f',e['symbol'],monthstr(e['entry_target']+12*3600000)))
print('EVENTS',len(events),'FILES',len(jobs),flush=True)
with ThreadPoolExecutor(max_workers=24) as ex:
 fs=[ex.submit(fetch_job,j) for j in jobs]
 for i,f in enumerate(as_completed(fs),1):
  f.result()
  if i%250==0 or i==len(fs):print('DOWNLOAD',i,'/',len(fs),flush=True)
@lru_cache(None)
def klines(sym,ds):
 p=CACHE/f'k-{sym}-{ds}.zip'
 if not p.exists() or p.stat().st_size==0:return []
 z=zipfile.ZipFile(p);fn=z.namelist()[0];out=[]
 for r in csv.reader(io.TextIOWrapper(z.open(fn))):
  if not r or not r[0].isdigit():continue
  try:
   t=int(r[0]);t=t//1000 if t>10**15 else t
   out.append((t,float(r[1]),float(r[4])))
  except:pass
 z.close();return out
@lru_cache(None)
def funding(sym,mo):
 p=CACHE/f'f-{sym}-{mo}.zip'
 if not p.exists() or p.stat().st_size==0:return []
 z=zipfile.ZipFile(p);fn=z.namelist()[0];rows=csv.reader(io.TextIOWrapper(z.open(fn)));hdr=next(rows,None)
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
 z.close();return out
def slip(p,side,enter):
 return p*(1.0005 if ((side=='LONG' and enter) or (side=='SHORT' and not enter)) else 0.9995)
def roi(entry,p,side):
 return ((p-entry)/p if side=='LONG' else (entry-p)/p)*4*100
def replay(e):
 ent=e['entry_target'];side=e['side'];sym=e['symbol'];horizon=ent+12*3600000
 ds=daystr(ent);days=[ds,nextday(ds)]
 bars=[]
 for d in days:bars.extend(klines(sym,d))
 bars.sort()
 idx=next((i for i,b in enumerate(bars) if b[0]>=ent),None)
 if idx is None:return e|{'error':'no_entry'}
 if bars[idx][0]!=ent:return e|{'error':'entry_gap','actual_entry':bars[idx][0]}
 entry=slip(bars[idx][1],side,True);qty=4.0/entry;open_fee=qty*entry*0.0005
 months=sorted({monthstr(ent),monthstr(horizon)})
 funds=[]
 for m in months:funds.extend(funding(sym,m))
 funds.sort();fi=0
 while fi<len(funds) and funds[fi][0]<ent:fi+=1
 fp=0.0;trigger=None;exit_idx=None;last=idx
 for i in range(idx,len(bars)):
  t,op,cl=bars[i]
  if t>horizon+60000:break
  last=i
  while fi<len(funds) and funds[fi][0]<=t+59999:
   ft,rate,mark=funds[fi]
   if ent<=ft<=horizon:
    notional=qty*(mark if mark>0 else cl)
    fp += (-notional*rate if side=='LONG' else notional*rate)
   fi+=1
  if t>=horizon:trigger='TIME';exit_idx=i;break
  rr=roi(entry,cl,side)
  if rr<=-6:trigger='SL';exit_idx=min(i+1,len(bars)-1);break
  if rr>=8:trigger='TP';exit_idx=min(i+1,len(bars)-1);break
 if exit_idx is None:trigger='EOF';exit_idx=last
 xt,xo,xc=bars[exit_idx];raw=xo if trigger in ('TP','SL','TIME') else xc;exitp=slip(raw,side,False)
 gross=((exitp-entry) if side=='LONG' else (entry-exitp))*qty;close_fee=qty*exitp*0.0005;net=gross-open_fee-close_fee+fp
 return e|{'entry_time':ent,'exit_time':xt,'trigger':trigger,'entry':entry,'exit':exitp,'gross_pct':gross*100,'fees_pct':(open_fee+close_fee)*100,'funding_pct':fp*100,'net_pct':net*100,'hold_h':(xt-ent)/3600000}
by={}
for e in events:by.setdefault(e['symbol'],[]).append(e)
res=[];skipped=[];errors=[]
for si,(sym,es) in enumerate(sorted(by.items()),1):
 available=-1
 for e in es:
  if e['entry_target']<available:
   skipped.append(e|{'skip_reason':'single_position','blocked_until':available});continue
  x=replay(e)
  if x.get('error'):errors.append(x);continue
  res.append(x);available=x['exit_time']
 print('REPLAY_SYMBOL',si,'/',len(by),sym,'signals',len(es),'trades',sum(x['symbol']==sym for x in res),'skipped',sum(x['symbol']==sym for x in skipped),flush=True)
def summ(xs):
 if not xs:return {'n':0}
 gp=sum(max(x['net_pct'],0) for x in xs);gl=-sum(min(x['net_pct'],0) for x in xs)
 sy={}
 for x in xs:sy.setdefault(x['symbol'],[]).append(x['net_pct'])
 return {'n':len(xs),'wins':sum(x['net_pct']>0 for x in xs),'losses':sum(x['net_pct']<=0 for x in xs),'tp':sum(x['trigger']=='TP' for x in xs),'sl':sum(x['trigger']=='SL' for x in xs),'time':sum(x['trigger']=='TIME' for x in xs),'eof':sum(x['trigger']=='EOF' for x in xs),'pf':gp/gl if gl else None,'net_pct':sum(x['net_pct'] for x in xs),'avg_pct':sum(x['net_pct'] for x in xs)/len(xs),'symbols':len(sy),'positive_symbols':sum(sum(v)/len(v)>0 for v in sy.values())}
out={'all':summ(res),'discovery':summ([x for x in res if x['period']=='discovery']),'oos':summ([x for x in res if x['period']=='oos']),'LONG':summ([x for x in res if x['side']=='LONG']),'SHORT':summ([x for x in res if x['side']=='SHORT']),'skipped_single_position':len(skipped),'errors':len(errors)}
for p in ['discovery','oos']:
 q=[x for x in res if x['period']==p]
 out[p+'_LONG']=summ([x for x in q if x['side']=='LONG']);out[p+'_SHORT']=summ([x for x in q if x['side']=='SHORT'])
print('SUMMARY',json.dumps(out,indent=2),flush=True)
Path('/tmp/usdc_usdt_exact_trades.json').write_text(json.dumps(res,indent=2))
Path('/tmp/usdc_usdt_exact_skipped.json').write_text(json.dumps(skipped,indent=2))
Path('/tmp/usdc_usdt_exact_errors.json').write_text(json.dumps(errors,indent=2))
Path('/tmp/usdc_usdt_exact_summary.json').write_text(json.dumps(out,indent=2))
