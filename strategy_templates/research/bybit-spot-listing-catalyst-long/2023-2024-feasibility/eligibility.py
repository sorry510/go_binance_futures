import json,datetime,urllib.request,urllib.error,time,zipfile,io,csv,re
from concurrent.futures import ThreadPoolExecutor,as_completed
from functools import lru_cache
from pathlib import Path
UTC=datetime.timezone.utc;UA='Mozilla/5.0'
EV=json.load(open('/tmp/bybit_spot_listing_events.json'))
BASE='https://data.binance.vision/data/futures/um/monthly/klines'
CACHE=Path('/tmp/bybit_spot_listing_binance_cache');CACHE.mkdir(exist_ok=True)
def dtsec(x):return datetime.datetime.fromtimestamp(x,UTC)
def month(d):return d.strftime('%Y-%m')
def prevmonth(d):return f'{d.year-1}-12' if d.month==1 else f'{d.year:04d}-{d.month-1:02d}'
def sub2(d):
    try:return d.replace(year=d.year-2)
    except ValueError:return d.replace(year=d.year-2,day=28)
def variants(bybit_symbol):
    base=bybit_symbol[:-4]
    stripped=re.sub(r'^[0-9]+','',base)
    vals=[bybit_symbol,stripped+'USDT','1000'+stripped+'USDT','10000'+stripped+'USDT','1000000'+stripped+'USDT']
    out=[]
    for s in vals:
        if s not in out:out.append(s)
    return out
def fetch(sym,mo):
    p=CACHE/f'{sym}-{mo}.zip'
    if p.exists():return p
    u=f'{BASE}/{sym}/1h/{sym}-1h-{mo}.zip'
    for k in range(4):
        try:
            b=urllib.request.urlopen(urllib.request.Request(u,headers={'User-Agent':UA}),timeout=20).read();p.write_bytes(b);return p
        except urllib.error.HTTPError as e:
            if e.code==404:p.write_bytes(b'');return p
            time.sleep(.35*(k+1))
        except Exception:time.sleep(.35*(k+1))
    p.write_bytes(b'');return p
jobs=set()
for e in EV:
    d=dtsec(e['event_ts'])
    for s in variants(e['bybit_symbol']):jobs.add((s,month(d)))
print('RESOLVE_JOBS',len(jobs),flush=True)
with ThreadPoolExecutor(max_workers=20) as ex:
    fs=[ex.submit(fetch,*j) for j in jobs]
    for i,f in enumerate(as_completed(fs),1):
        f.result()
        if i%100==0:print('RESOLVE',i,'/',len(jobs),flush=True)
resolved=[]
for e in EV:
    d=dtsec(e['event_ts'])
    found=[]
    for s in variants(e['bybit_symbol']):
        p=CACHE/f'{s}-{month(d)}.zip'
        if p.exists() and p.stat().st_size>0:found.append(s)
    x=dict(e)
    if not found:x.update(eligible=False,reason='no_binance_usdm_event_month');resolved.append(x);continue
    x['symbol']=found[0];resolved.append(x)
print('EVENT_MONTH_MATCH',sum('symbol' in x for x in resolved),flush=True)
jobs=set()
for e in resolved:
    if 'symbol' not in e:continue
    d=dtsec(e['event_ts']);cut=sub2(d)
    jobs.add((e['symbol'],month(cut)));jobs.add((e['symbol'],prevmonth(d)))
with ThreadPoolExecutor(max_workers=20) as ex:
    fs=[ex.submit(fetch,*j) for j in jobs]
    for i,f in enumerate(as_completed(fs),1):
        f.result()
        if i%50==0:print('QUALIFY',i,'/',len(jobs),flush=True)
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
out=[]
for e in resolved:
    if 'symbol' not in e:out.append(e);continue
    x=dict(e);d=dtsec(e['event_ts']);evt=int(e['event_ts']*1000);cutd=sub2(d);cut=int(cutd.timestamp()*1000)
    hist=rows(e['symbol'],month(cutd))
    if not hist or hist[0][0]>cut:x.update(eligible=False,reason='history_lt_2y');out.append(x);continue
    hour=(evt//3600000)*3600000;start=hour-24*3600000
    prior=rows(e['symbol'],prevmonth(d))+rows(e['symbol'],month(d));prior=[r for r in prior if start<=r[0]<hour]
    qv=sum(r[3] for r in prior);x['bars_24h']=len(prior);x['quote_volume_24h']=qv
    if len(prior)<24:x.update(eligible=False,reason='missing_24h_bars');out.append(x);continue
    if qv<5_000_000:x.update(eligible=False,reason='quote_volume_lt_5m');out.append(x);continue
    x.update(eligible=True,reason='ok');out.append(x)
good=[x for x in out if x.get('eligible')]
from collections import Counter
print('ELIGIBLE',len(good),'UNIQUE',len(set(x['symbol'] for x in good)))
print('REASONS',Counter(x.get('reason') for x in out))
for x in good:print('OK',dtsec(x['event_ts']).isoformat(),x['bybit_symbol'],'=>',x['symbol'],'qv',round(x['quote_volume_24h']),'|',x['title'])
Path('/tmp/bybit_spot_listing_eligibility.json').write_text(json.dumps(out,ensure_ascii=False,indent=2))
