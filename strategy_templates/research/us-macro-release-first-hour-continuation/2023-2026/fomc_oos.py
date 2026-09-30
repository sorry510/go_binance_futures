import csv,io,zipfile,urllib.request,urllib.error,time,datetime,math,json
from pathlib import Path
from zoneinfo import ZoneInfo
from concurrent.futures import ThreadPoolExecutor,as_completed
UTC=datetime.timezone.utc;ET=ZoneInfo('America/New_York');UA='Mozilla/5.0'
SYMS=['BTCUSDT','ETHUSDT','BNBUSDT','XRPUSDT','SOLUSDT','DOGEUSDT','LTCUSDT','AVAXUSDT','UNIUSDT','ZECUSDT']
FOMC={2025:['01-29','03-19','05-07','06-18','07-30','09-17','10-29','12-10'],
      2026:['01-28','03-18','04-29','06-17','07-29','09-16']}
events=[]
for y,dates in FOMC.items():
    for md in dates:
        m,d=map(int,md.split('-'));dt=datetime.datetime(y,m,d,14,0,tzinfo=ET).astimezone(UTC)
        events.append({'t':int(dt.timestamp()*1000),'year':y,'date':f'{y}-{md}'})
events.sort(key=lambda x:x['t'])
print('EVENTS',len(events),events[0],events[-1],flush=True)
CACHE=Path('/tmp/macro_5m_cache');CACHE.mkdir(exist_ok=True)
months=[]
d=datetime.date(2025,1,1)
while d<=datetime.date(2026,9,1):
    months.append(d.strftime('%Y-%m'));d=(d.replace(day=28)+datetime.timedelta(days=4)).replace(day=1)
def fetch(sym,mo):
    p=CACHE/f'{sym}-{mo}.zip'
    if p.exists():return p
    u=f'https://data.binance.vision/data/futures/um/monthly/klines/{sym}/5m/{sym}-5m-{mo}.zip'
    for k in range(5):
        try:b=urllib.request.urlopen(urllib.request.Request(u,headers={'User-Agent':UA}),timeout=30).read();p.write_bytes(b);return p
        except urllib.error.HTTPError as e:
            if e.code==404:p.write_bytes(b'');return p
            time.sleep(.5*(k+1))
        except Exception:time.sleep(.5*(k+1))
    p.write_bytes(b'');return p
jobs=[(s,m) for s in SYMS for m in months]
with ThreadPoolExecutor(max_workers=16) as ex:
    fs=[ex.submit(fetch,*j) for j in jobs]
    for i,f in enumerate(as_completed(fs),1):
        f.result()
        if i%50==0:print('DOWNLOAD',i,'/',len(jobs),flush=True)
def read_month(sym,mo):
    p=CACHE/f'{sym}-{mo}.zip'
    if not p.exists() or p.stat().st_size==0:return []
    z=zipfile.ZipFile(p);fn=z.namelist()[0];out=[]
    for r in csv.reader(io.TextIOWrapper(z.open(fn))):
        if not r or not r[0].isdigit():continue
        t=int(r[0]);t=t//1000 if t>10**15 else t
        out.append((t,float(r[1]),float(r[4]),float(r[7]) if len(r)>7 else 0.0))
    return out
maps={}
for s in SYMS:
    q={}
    for mo in months:
        for r in read_month(s,mo):q[r[0]]=r
    maps[s]=q;print('MAP',s,len(q),flush=True)
res=[]
for s in SYMS:
    q=maps[s]
    for e in events:
        t=e['t'];required=[t+i*300000 for i in range(-288,157)]
        if any(x not in q for x in required):continue
        qv=sum(q[t+i*300000][3] for i in range(-288,0))
        if qv<5_000_000:continue
        ro=q[t][1];rc=q[t+11*300000][2]
        if ro<=0 or rc<=0:continue
        reaction=math.log(rc/ro)
        if reaction==0:continue
        sign=1.0 if reaction>0 else -1.0
        ent=t+12*300000;entry=q[ent][1]
        res.append({'symbol':s,'year':e['year'],'date':e['date'],'reaction':reaction,'qv24':qv,
          'r1':sign*math.log(q[ent+11*300000][2]/entry),
          'r4':sign*math.log(q[ent+47*300000][2]/entry),
          'r12':sign*math.log(q[ent+143*300000][2]/entry)})
def stats(xs,k):
    v=[x[k] for x in xs];return {'n':len(v),'mean':sum(v)/len(v) if v else None,'win':sum(z>0 for z in v)/len(v) if v else None}
byyear={};bysym={}
for y in [2025,2026]:byyear[str(y)]=stats([x for x in res if x['year']==y],'r12')
for s in SYMS:bysym[s]=stats([x for x in res if x['symbol']==s],'r12')
summary={'events':len(events),'instances':len(res),'r1':stats(res,'r1'),'r4':stats(res,'r4'),'r12':stats(res,'r12'),'by_year':byyear,'by_symbol':bysym,'positive_symbols12':sum(v['mean'] is not None and v['mean']>0 for v in bysym.values())}
print(json.dumps(summary,indent=2),flush=True)
Path('/tmp/fomc_oos_results.json').write_text(json.dumps(res,indent=2));Path('/tmp/fomc_oos_summary.json').write_text(json.dumps(summary,indent=2))
