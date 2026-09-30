import urllib.request,urllib.error,json,datetime,time,re,csv,io,zipfile,math
from concurrent.futures import ThreadPoolExecutor,as_completed
from functools import lru_cache
from pathlib import Path
UTC=datetime.timezone.utc;UA='Mozilla/5.0'
LIST='https://www.binance.com/bapi/composite/v1/public/cms/article/list/query'
VISION='https://data.binance.vision/data/futures/um/monthly/klines'
CACHE=Path('/tmp/network_upgrade_cache');CACHE.mkdir(exist_ok=True)
def get_json(u):
    last=None
    for k in range(6):
        try:
            req=urllib.request.Request(u,headers={'User-Agent':UA,'Accept':'application/json'})
            return json.loads(urllib.request.urlopen(req,timeout=20).read())
        except Exception as e:
            last=e; time.sleep(1.2*(k+1))
    raise last
arts=[];seen=False
for page in range(1,80):
    u=f'{LIST}?type=1&pageNo={page}&pageSize=50&catalogId=157'
    o=get_json(u)
    aa=((((o.get('data') or {}).get('catalogs') or [{}])[0]).get('articles') or [])
    if not aa:break
    ds=[datetime.datetime.fromtimestamp(a['releaseDate']/1000,UTC) for a in aa]
    for a,d in zip(aa,ds):
        if d.year in (2023,2024):
            seen=True;t=a.get('title','')
            if re.search(r'Network Upgrade|Hard Fork',t,re.I):
                arts.append({'code':a.get('code'),'title':t,'release_time':a['releaseDate']})
    if seen and min(ds).year<2023:break
    time.sleep(.05)
print('ARTICLES',len(arts),flush=True)
events=[]
for a in arts:
    toks=[]
    for t in re.findall(r'\(([A-Z0-9]{2,20})\)',a['title']):
        if t not in {'BEP2','BEP20','ERC20','N3'} and t not in toks:toks.append(t)
    for t in toks:
        events.append({**a,'ticker':t,'symbol':t+'USDT'})
print('TOKEN_EVENTS_RAW',len(events),'UNIQUE',len(set(x['symbol'] for x in events)),flush=True)
def mon(ms):return datetime.datetime.fromtimestamp(ms/1000,UTC).strftime('%Y-%m')
def prevm(ms):
    d=datetime.datetime.fromtimestamp(ms/1000,UTC);return f'{d.year-1}-12' if d.month==1 else f'{d.year:04d}-{d.month-1:02d}'
def nextm(ms):
    d=datetime.datetime.fromtimestamp(ms/1000,UTC);return f'{d.year+1}-01' if d.month==12 else f'{d.year:04d}-{d.month+1:02d}'
def sub2(ms):
    d=datetime.datetime.fromtimestamp(ms/1000,UTC)
    try:d=d.replace(year=d.year-2)
    except ValueError:d=d.replace(year=d.year-2,day=28)
    return int(d.timestamp()*1000)
def fetch(sym,mo):
    p=CACHE/f'{sym}-{mo}.zip'
    if p.exists():return p
    u=f'{VISION}/{sym}/1h/{sym}-1h-{mo}.zip'
    for k in range(4):
        try:
            b=urllib.request.urlopen(urllib.request.Request(u,headers={'User-Agent':UA}),timeout=20).read();p.write_bytes(b);return p
        except urllib.error.HTTPError as e:
            if e.code==404:p.write_bytes(b'');return p
            time.sleep(.4*(k+1))
        except Exception:time.sleep(.4*(k+1))
    p.write_bytes(b'');return p
jobs=set()
for e in events:
    evt=e['release_time'];jobs.add((e['symbol'],mon(sub2(evt))))
    for m in (prevm(evt),mon(evt),nextm(evt)):jobs.add((e['symbol'],m))
print('JOBS',len(jobs),flush=True)
with ThreadPoolExecutor(max_workers=18) as ex:
    fs=[ex.submit(fetch,*j) for j in jobs]
    for i,f in enumerate(as_completed(fs),1):
        f.result()
        if i%200==0:print('DOWNLOAD',i,'/',len(jobs),flush=True)
@lru_cache(None)
def rows(sym,mo):
    p=CACHE/f'{sym}-{mo}.zip'
    if not p.exists() or p.stat().st_size==0:return []
    z=zipfile.ZipFile(p);fn=z.namelist()[0];out=[]
    for r in csv.reader(io.TextIOWrapper(z.open(fn))):
        if not r or not r[0].isdigit():continue
        t=int(r[0]);t=t//1000 if t>10**15 else t
        out.append((t,float(r[1]),float(r[4]),float(r[7]) if len(r)>7 else 0.0))
    return out
good=[];excluded={}
for e in events:
    sym=e['symbol'];evt=e['release_time'];cut=sub2(evt);hist=rows(sym,mon(cut))
    if not hist or hist[0][0]>cut:excluded['history_lt_2y_or_missing']=excluded.get('history_lt_2y_or_missing',0)+1;continue
    hour=(evt//3600000)*3600000
    prior=[r for r in rows(sym,prevm(evt))+rows(sym,mon(evt)) if hour-86400000<=r[0]<=hour-1]
    qv=sum(r[3] for r in prior)
    if len(prior)<24:excluded['missing_24h_bars']=excluded.get('missing_24h_bars',0)+1;continue
    if qv<5_000_000:excluded['quote_volume_lt_5m']=excluded.get('quote_volume_lt_5m',0)+1;continue
    ent=((evt//3600000)+1)*3600000
    fwd=[r for r in rows(sym,mon(evt))+rows(sym,nextm(evt)) if ent<=r[0]<=ent+13*3600000];fwd.sort()
    if len(fwd)<13 or fwd[0][0]!=ent:excluded['missing_forward_bars']=excluded.get('missing_forward_bars',0)+1;continue
    entry=fwd[0][1];x=dict(e);x['quote_volume_24h']=qv;x['entry_time']=ent
    x['r1']=math.log(fwd[0][2]/entry);x['r4']=math.log(fwd[3][2]/entry);x['r12']=math.log(fwd[11][2]/entry);good.append(x)
def stats(a,k):
    v=[x[k] for x in a];return {'n':len(v),'mean':sum(v)/len(v) if v else None,'win':sum(z>0 for z in v)/len(v) if v else None}
print('ELIGIBLE',len(good),'UNIQUE_SYMBOLS',len(set(x['symbol'] for x in good)),'EXCLUDED',excluded,flush=True)
for k in ('r1','r4','r12'):print(k,stats(good,k))
for y in (2023,2024):
    q=[x for x in good if datetime.datetime.fromtimestamp(x['release_time']/1000,UTC).year==y];print('YEAR',y,stats(q,'r12'))
by={}
for x in good:by.setdefault(x['symbol'],[]).append(x['r12'])
print('SYMBOLS',len(by),'POSITIVE',sum(sum(v)/len(v)>0 for v in by.values()))
for s,v in sorted(by.items()):print('SYM',s,'n',len(v),'mean12',round(sum(v)/len(v)*100,3))
summary={'articles':len(arts),'raw_events':len(events),'eligible':len(good),'unique_symbols':len(by),'excluded':excluded,'r1':stats(good,'r1'),'r4':stats(good,'r4'),'r12':stats(good,'r12'),'positive_symbols12':sum(sum(v)/len(v)>0 for v in by.values())}
Path('/tmp/network_upgrade_events.json').write_text(json.dumps(good,ensure_ascii=False,indent=2));Path('/tmp/network_upgrade_summary.json').write_text(json.dumps(summary,indent=2))
