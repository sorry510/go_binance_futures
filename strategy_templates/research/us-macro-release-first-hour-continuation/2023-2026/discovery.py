import csv,io,zipfile,urllib.request,urllib.error,time,datetime,math,json
from pathlib import Path
from zoneinfo import ZoneInfo
from concurrent.futures import ThreadPoolExecutor,as_completed
UTC=datetime.timezone.utc;ET=ZoneInfo('America/New_York');UA='Mozilla/5.0'
SYMS=['BTCUSDT','ETHUSDT','BNBUSDT','XRPUSDT','SOLUSDT','DOGEUSDT','LTCUSDT','AVAXUSDT','UNIUSDT','ZECUSDT']
# Official BLS 08:30 ET release dates, fixed before returns.
BLS={
2023:{
'NFP':['01-06','02-03','03-10','04-07','05-05','06-02','07-07','08-04','09-01','10-06','11-03','12-08'],
'CPI':['01-12','02-14','03-14','04-12','05-10','06-13','07-12','08-10','09-13','10-12','11-14','12-12'],
'PPI':['01-18','02-16','03-15','04-13','05-11','06-14','07-13','08-11','09-14','10-11','11-15','12-13']},
2024:{
'NFP':['01-05','02-02','03-08','04-05','05-03','06-07','07-05','08-02','09-06','10-04','11-01','12-06'],
'CPI':['01-11','02-13','03-12','04-10','05-15','06-12','07-11','08-14','09-11','10-10','11-13','12-11'],
'PPI':['01-12','02-16','03-14','04-11','05-14','06-13','07-12','08-13','09-12','10-11','11-14','12-12']}}
FOMC={2023:['02-01','03-22','05-03','06-14','07-26','09-20','11-01','12-13'],
      2024:['01-31','03-20','05-01','06-12','07-31','09-18','11-07','12-18']}
events=[]
for y,dct in BLS.items():
    for typ,dates in dct.items():
        for md in dates:
            m,d=map(int,md.split('-'));dt=datetime.datetime(y,m,d,8,30,tzinfo=ET).astimezone(UTC)
            events.append({'t':int(dt.timestamp()*1000),'type':typ,'date':f'{y}-{md}'})
for y,dates in FOMC.items():
    for md in dates:
        m,d=map(int,md.split('-'));dt=datetime.datetime(y,m,d,14,0,tzinfo=ET).astimezone(UTC)
        events.append({'t':int(dt.timestamp()*1000),'type':'FOMC','date':f'{y}-{md}'})
events.sort(key=lambda x:x['t'])
print('EVENTS',len(events),'2023',sum(x['date'].startswith('2023') for x in events),'2024',sum(x['date'].startswith('2024') for x in events),flush=True)
CACHE=Path('/tmp/macro_5m_cache');CACHE.mkdir(exist_ok=True)
months=[]
d=datetime.date(2022,12,1)
while d<=datetime.date(2025,1,1):
    months.append(d.strftime('%Y-%m'));d=(d.replace(day=28)+datetime.timedelta(days=4)).replace(day=1)
def fetch(sym,mo):
    p=CACHE/f'{sym}-{mo}.zip'
    if p.exists():return p
    u=f'https://data.binance.vision/data/futures/um/monthly/klines/{sym}/5m/{sym}-5m-{mo}.zip'
    for k in range(5):
        try:
            b=urllib.request.urlopen(urllib.request.Request(u,headers={'User-Agent':UA}),timeout=30).read();p.write_bytes(b);return p
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
# Build per-symbol timestamp maps once.
maps={}
for s in SYMS:
    q={}
    first=None
    for mo in months:
        rr=read_month(s,mo)
        if rr and first is None:first=rr[0][0]
        for r in rr:q[r[0]]=r
    maps[s]=q
    print('MAP',s,len(q),datetime.datetime.fromtimestamp(first/1000,UTC).date() if first else None,flush=True)
res=[]
for s in SYMS:
    q=maps[s]
    for e in events:
        t=e['t']
        # History gate: use Dec-2022 availability plus known 2y requirement.
        # Since archive starts Dec-2022, require first data <= event-730d only when observable;
        # all selected legacy symbols are expected to predate this window.
        required=[t+i*300000 for i in range(-288,157)] # prior24h + reaction + 12h forward
        if any(x not in q for x in required):
            continue
        prior=[q[t+i*300000] for i in range(-288,0)]
        qv=sum(x[3] for x in prior)
        if qv<5_000_000:continue
        reaction_open=q[t][1];reaction_close=q[t+11*300000][2]
        if reaction_open<=0 or reaction_close<=0:continue
        react=math.log(reaction_close/reaction_open)
        if react==0:continue
        sign=1.0 if react>0 else -1.0
        entry_t=t+12*300000;entry=q[entry_t][1]
        r1=sign*math.log(q[entry_t+11*300000][2]/entry)
        r4=sign*math.log(q[entry_t+47*300000][2]/entry)
        r12=sign*math.log(q[entry_t+143*300000][2]/entry)
        res.append({'symbol':s,'year':int(e['date'][:4]),'date':e['date'],'type':e['type'],'event_t':t,'reaction':react,'qv24':qv,'r1':r1,'r4':r4,'r12':r12})
def stats(xs,key):
    v=[x[key] for x in xs]
    return {'n':len(v),'mean':sum(v)/len(v) if v else None,'win':sum(z>0 for z in v)/len(v) if v else None}
print('RESULTS',len(res),flush=True)
for key in ('r1','r4','r12'):print(key,stats(res,key),flush=True)
bytype={}
for typ in ['NFP','CPI','PPI','FOMC']:
    z=[x for x in res if x['type']==typ];bytype[typ]=stats(z,'r12');print('TYPE',typ,bytype[typ])
bysym={}
for s in SYMS:
    z=[x for x in res if x['symbol']==s];bysym[s]=stats(z,'r12');print('SYM',s,bysym[s])
byyear={}
for y in [2023,2024]:
    z=[x for x in res if x['year']==y];byyear[str(y)]=stats(z,'r12');print('YEAR',y,byyear[str(y)])
summary={'events':len(events),'instances':len(res),'r1':stats(res,'r1'),'r4':stats(res,'r4'),'r12':stats(res,'r12'),'by_type':bytype,'by_symbol':bysym,'by_year':byyear,'positive_symbols12':sum(v['mean'] is not None and v['mean']>0 for v in bysym.values())}
Path('/tmp/macro_2023_2024_results.json').write_text(json.dumps(res,indent=2))
Path('/tmp/macro_2023_2024_summary.json').write_text(json.dumps(summary,indent=2))
print(json.dumps(summary,indent=2),flush=True)
