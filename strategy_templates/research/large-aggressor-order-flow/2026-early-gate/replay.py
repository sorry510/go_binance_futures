import csv,io,zipfile,urllib.request,urllib.error,time,datetime,json,math,statistics
from concurrent.futures import ThreadPoolExecutor,as_completed
from pathlib import Path
UTC=datetime.timezone.utc;UA='Mozilla/5.0'
SYMS=['BTCUSDT','ETHUSDT','SOLUSDT','XRPUSDT'];SIGNAL_DATES=['2026-01-15','2026-04-15','2026-07-15']
CACHE=Path('/tmp/large_aggflow_cache');CACHE.mkdir(exist_ok=True)
def dayoff(s,n):return (datetime.date.fromisoformat(s)+datetime.timedelta(days=n)).isoformat()
def fetch(sym,date,kind):
    p=CACHE/f'{kind}-{sym}-{date}.zip'
    if p.exists():return p
    u=(f'https://data.binance.vision/data/futures/um/daily/aggTrades/{sym}/{sym}-aggTrades-{date}.zip' if kind=='agg'
       else f'https://data.binance.vision/data/futures/um/daily/klines/{sym}/1h/{sym}-1h-{date}.zip')
    for k in range(5):
        try:b=urllib.request.urlopen(urllib.request.Request(u,headers={'User-Agent':UA}),timeout=30).read();p.write_bytes(b);return p
        except urllib.error.HTTPError as e:
            if e.code==404:p.write_bytes(b'');return p
            time.sleep(.5*(k+1))
        except Exception:time.sleep(.5*(k+1))
    p.write_bytes(b'');return p
jobs=set()
for s in SYMS:
    for d in SIGNAL_DATES:
        jobs|={(s,dayoff(d,-1),'agg'),(s,d,'agg'),(s,d,'kline'),(s,dayoff(d,1),'kline')}
print('JOBS',len(jobs),flush=True)
with ThreadPoolExecutor(max_workers=8) as ex:
    fs=[ex.submit(fetch,*j) for j in jobs]
    for i,f in enumerate(as_completed(fs),1):
        f.result()
        if i%8==0:print('DOWNLOAD',i,'/',len(jobs),flush=True)
def agg_file(sym,date):
    p=CACHE/f'agg-{sym}-{date}.zip'
    if not p.exists() or p.stat().st_size==0:return None,None
    z=zipfile.ZipFile(p);fn=z.namelist()[0];return z,fn
def baseline_q99(sym,date):
    z,fn=agg_file(sym,date)
    if z is None:return None,0
    vals=[]
    for r in csv.reader(io.TextIOWrapper(z.open(fn))):
        if not r or not r[0].lstrip('-').isdigit():continue
        try:vals.append(float(r[1])*float(r[2]))
        except:pass
    z.close()
    if not vals:return None,0
    vals.sort();idx=min(len(vals)-1,max(0,int(math.ceil(.99*len(vals)))-1))
    return vals[idx],len(vals)
def hourly_tail_file(sym,date,threshold):
    z,fn=agg_file(sym,date)
    if z is None:return {}
    a={}
    for r in csv.reader(io.TextIOWrapper(z.open(fn))):
        if not r or not r[0].lstrip('-').isdigit():continue
        try:
            n=float(r[1])*float(r[2])
            if n<threshold:continue
            t=int(r[5]);t=t//1000 if t>10**15 else t;maker=str(r[6]).strip().lower() in ('true','1')
            h=(t//3600000)*3600000;x=a.setdefault(h,[0.0,0.0]);x[0]+=(-n if maker else n);x[1]+=n
        except:pass
    z.close();return {h:(v[0]/v[1] if v[1] else 0.0) for h,v in a.items()}
def parse_k(sym,dates):
    out={}
    for date in dates:
        p=CACHE/f'kline-{sym}-{date}.zip'
        if not p.exists() or p.stat().st_size==0:continue
        z=zipfile.ZipFile(p);fn=z.namelist()[0]
        for r in csv.reader(io.TextIOWrapper(z.open(fn))):
            if not r or not r[0].isdigit():continue
            t=int(r[0]);t=t//1000 if t>10**15 else t;out[t]=(float(r[1]),float(r[4]))
        z.close()
    return out
events=[]
for sym in SYMS:
  for d in SIGNAL_DATES:
    bd=dayoff(d,-1);q99,nbase=baseline_q99(sym,bd)
    if q99 is None:print('MISSING',sym,d);continue
    bh=hourly_tail_file(sym,bd,q99);ch=hourly_tail_file(sym,d,q99)
    bstart=datetime.datetime.fromisoformat(bd).replace(tzinfo=UTC)
    feats=[bh.get(int((bstart+datetime.timedelta(hours=h)).timestamp()*1000),0.0) for h in range(24)]
    mu=statistics.fmean(feats);sd=statistics.pstdev(feats)
    if sd<=0:continue
    px=parse_k(sym,[d,dayoff(d,1)]);start=datetime.datetime.fromisoformat(d).replace(tzinfo=UTC);armed=True;nev=0
    for h in range(24):
        ht=int((start+datetime.timedelta(hours=h)).timestamp()*1000);f=ch.get(ht,0.0);zv=(f-mu)/sd
        if abs(zv)<1:armed=True
        if not armed or abs(zv)<3:continue
        ent=ht+3600000;bars=[px.get(ent+i*3600000) for i in range(12)]
        if any(x is None for x in bars):continue
        direction=1.0 if zv>0 else -1.0;entry=bars[0][0]
        events.append({'symbol':sym,'date':d,'hour':h,'q99':q99,'feature':f,'z':zv,
          'r1':direction*math.log(bars[0][1]/entry),'r4':direction*math.log(bars[3][1]/entry),'r12':direction*math.log(bars[11][1]/entry)})
        armed=False;nev+=1
    print('GATE',sym,d,'baseline_orders',nbase,'q99',round(q99,2),'events',nev,flush=True)
def summary(xs):
    if not xs:return {}
    by={}
    for s in SYMS:
        q=[x for x in xs if x['symbol']==s]
        if q:by[s]={'n':len(q),'mean1':sum(x['r1'] for x in q)/len(q),'mean4':sum(x['r4'] for x in q)/len(q),'mean12':sum(x['r12'] for x in q)/len(q)}
    return {'n':len(xs),'mean1':sum(x['r1'] for x in xs)/len(xs),'win1':sum(x['r1']>0 for x in xs)/len(xs),
      'mean4':sum(x['r4'] for x in xs)/len(xs),'win4':sum(x['r4']>0 for x in xs)/len(xs),
      'mean12':sum(x['r12'] for x in xs)/len(xs),'win12':sum(x['r12']>0 for x in xs)/len(xs),
      'positive_symbols1':sum(v['mean1']>0 for v in by.values()),'positive_symbols12':sum(v['mean12']>0 for v in by.values()),'symbols':len(by),'by_symbol':by}
res=summary(events);print(json.dumps(res,indent=2),flush=True)
Path('/tmp/large_aggtrade_flow_events.json').write_text(json.dumps(events,indent=2));Path('/tmp/large_aggtrade_flow_summary.json').write_text(json.dumps(res,indent=2))
