import csv,io,zipfile,urllib.request,urllib.error,time,datetime,json,math
from concurrent.futures import ThreadPoolExecutor,as_completed
from functools import lru_cache
from pathlib import Path
UTC=datetime.timezone.utc;UA='Mozilla/5.0'
EV=json.load(open('/tmp/collateral_events_strict.json'))
VISION='https://data.binance.vision/data/futures/um/monthly/klines'
CACHE=Path('/tmp/collateral_market_cache');CACHE.mkdir(exist_ok=True)
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
for e in EV:
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
for e in EV:
    sym=e['symbol'];evt=e['release_time'];cut=sub2(evt);hist=rows(sym,mon(cut))
    if not hist or hist[0][0]>cut:excluded['history_lt_2y_or_missing']=excluded.get('history_lt_2y_or_missing',0)+1;continue
    hour=(evt//3600000)*3600000
    prior=[r for r in rows(sym,prevm(evt))+rows(sym,mon(evt)) if hour-86400000<=r[0]<=hour-1];qv=sum(r[3] for r in prior)
    if len(prior)<24:excluded['missing_24h_bars']=excluded.get('missing_24h_bars',0)+1;continue
    if qv<5_000_000:excluded['quote_volume_lt_5m']=excluded.get('quote_volume_lt_5m',0)+1;continue
    ent=((evt//3600000)+1)*3600000;fwd=[r for r in rows(sym,mon(evt))+rows(sym,nextm(evt)) if ent<=r[0]<=ent+13*3600000];fwd.sort()
    if len(fwd)<13 or fwd[0][0]!=ent:excluded['missing_forward_bars']=excluded.get('missing_forward_bars',0)+1;continue
    entry=fwd[0][1];x=dict(e);x['quote_volume_24h']=qv;x['entry_time']=ent
    x['r1']=math.log(fwd[0][2]/entry);x['r4']=math.log(fwd[3][2]/entry);x['r12']=math.log(fwd[11][2]/entry);good.append(x)
def stats(a,k):
    v=[x[k] for x in a];return {'n':len(v),'mean':sum(v)/len(v) if v else None,'win':sum(z>0 for z in v)/len(v) if v else None}
def report(name,a):
    by={}
    for x in a:by.setdefault(x['symbol'],[]).append(x['r12'])
    batches={}
    for x in a:batches.setdefault(x['article_code'],[]).append(x['r12'])
    batch_means=[sum(v)/len(v) for v in batches.values()]
    print(name,'r1',stats(a,'r1'),'r4',stats(a,'r4'),'r12',stats(a,'r12'),
          'symbols',len(by),'positive_symbols',sum(sum(v)/len(v)>0 for v in by.values()),
          'batches',len(batches),'batch_equal_mean12',sum(batch_means)/len(batch_means) if batch_means else None,
          'positive_batches',sum(v>0 for v in batch_means),flush=True)
    return {'r1':stats(a,'r1'),'r4':stats(a,'r4'),'r12':stats(a,'r12'),'symbols':len(by),
            'positive_symbols12':sum(sum(v)/len(v)>0 for v in by.values()),'batches':len(batches),
            'batch_equal_mean12':sum(batch_means)/len(batch_means) if batch_means else None,
            'positive_batches12':sum(v>0 for v in batch_means)}
disc=[x for x in good if datetime.datetime.fromtimestamp(x['release_time']/1000,UTC).year==2023]
oos=[x for x in good if datetime.datetime.fromtimestamp(x['release_time']/1000,UTC).year==2024]
print('ELIGIBLE',len(good),'EXCLUDED',excluded,flush=True)
res={'discovery_2023':report('DISCOVERY_2023',disc),'oos1_2024':report('OOS1_2024',oos)}
Path('/tmp/collateral_diagnostic_events.json').write_text(json.dumps(good,ensure_ascii=False,indent=2))
Path('/tmp/collateral_diagnostic_summary.json').write_text(json.dumps(res|{'excluded':excluded,'eligible':len(good)},indent=2))
