import urllib.request,urllib.error,json,datetime,time,re,csv,io,zipfile
from functools import lru_cache
from pathlib import Path
UTC=datetime.timezone.utc;UA='Mozilla/5.0'
U='https://www.binance.com/bapi/composite/v1/public/cms/article/list/query'
arts=[];seen=False
for page in range(1,150):
    try:
        o=json.loads(urllib.request.urlopen(urllib.request.Request(f'{U}?type=1&pageNo={page}&pageSize=50&catalogId=49',headers={'User-Agent':UA,'Accept':'application/json'}),timeout=20).read())
    except Exception:time.sleep(.7);continue
    cats=((o.get('data') or {}).get('catalogs') or []);aa=(cats[0].get('articles') if cats else []) or []
    if not aa:break
    ds=[datetime.datetime.fromtimestamp(a['releaseDate']/1000,UTC) for a in aa]
    for a,d in zip(aa,ds):
        t=a.get('title','')
        if d.year in (2023,2024) and 'Simple Earn' in t and ('Locked Products' in t or 'Flexible Products' in t) and re.search(r'\b(Adds?|Launches)\b',t,re.I) and not re.search(r'\b\d+-Day\b',t,re.I):
            arts.append(a);seen=True
    if seen and min(ds).year<2023:break
    time.sleep(.03)
events=[]
for a in arts:
    t=a['title'];seg=''
    m=re.search(r'Binance Adds (.*?) on Simple Earn',t,re.I)
    if m:seg=m.group(1)
    m2=re.search(r'Binance Simple Earn Launches (.*?) Locked Products',t,re.I)
    if m2:seg=m2.group(1)
    toks=[]
    if seg and not re.search(r'^New Assets$',seg,re.I):
        toks=[x for x in re.findall(r'(?<![A-Z0-9])([A-Z][A-Z0-9]{0,19})(?![A-Z0-9])',seg) if x not in {'AND'}]
    elif 'Adds New Assets to Simple Earn Flexible Products' in t:
        toks=['ARB','ID','TUSD','USDC']
    toks=list(dict.fromkeys(toks))
    for tok in toks:events.append({'article_code':a['code'],'article_title':t,'release_time':a['releaseDate'],'ticker':tok,'symbol':tok+'USDT'})
print('ARTICLES',len(arts),'EVENTS',len(events),'UNIQUE',len(set(x['symbol'] for x in events)),flush=True)
VISION='https://data.binance.vision/data/futures/um/monthly/klines';CACHE=Path('/tmp/simple_earn_cache');CACHE.mkdir(exist_ok=True)
def mon(ms):return datetime.datetime.fromtimestamp(ms/1000,UTC).strftime('%Y-%m')
def prevm(ms):
    d=datetime.datetime.fromtimestamp(ms/1000,UTC);return f'{d.year-1}-12' if d.month==1 else f'{d.year:04d}-{d.month-1:02d}'
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
        try:b=urllib.request.urlopen(urllib.request.Request(u,headers={'User-Agent':UA}),timeout=20).read();p.write_bytes(b);return p
        except urllib.error.HTTPError as e:
            if e.code==404:p.write_bytes(b'');return p
            time.sleep(.4*(k+1))
        except Exception:time.sleep(.4*(k+1))
    p.write_bytes(b'');return p
jobs=set()
for e in events:
    jobs.add((e['symbol'],mon(sub2(e['release_time']))));jobs.add((e['symbol'],prevm(e['release_time'])));jobs.add((e['symbol'],mon(e['release_time'])))
from concurrent.futures import ThreadPoolExecutor,as_completed
with ThreadPoolExecutor(max_workers=16) as ex:
    fs=[ex.submit(fetch,*j) for j in jobs]
    for i,f in enumerate(as_completed(fs),1):
        f.result()
        if i%100==0:print('DOWNLOAD',i,'/',len(jobs),flush=True)
@lru_cache(None)
def rows(sym,mo):
    p=CACHE/f'{sym}-{mo}.zip'
    if not p.exists() or p.stat().st_size==0:return []
    z=zipfile.ZipFile(p);fn=z.namelist()[0];out=[]
    for r in csv.reader(io.TextIOWrapper(z.open(fn))):
        if not r or not r[0].isdigit():continue
        t=int(r[0]);t=t//1000 if t>10**15 else t;out.append((t,float(r[7]) if len(r)>7 else 0.0))
    return out
out=[]
for e in events:
    x=dict(e);evt=e['release_time'];cut=sub2(evt);hist=rows(e['symbol'],mon(cut))
    if not hist or hist[0][0]>cut:x.update(eligible=False,reason='history_lt_2y_or_missing');out.append(x);continue
    hour=(evt//3600000)*3600000;prior=[r for r in rows(e['symbol'],prevm(evt))+rows(e['symbol'],mon(evt)) if hour-86400000<=r[0]<=hour-1];q=sum(r[1] for r in prior)
    x['bars_24h']=len(prior);x['quote_volume_24h']=q
    if len(prior)<24:x.update(eligible=False,reason='missing_24h_bars');out.append(x);continue
    if q<5_000_000:x.update(eligible=False,reason='quote_volume_lt_5m');out.append(x);continue
    x.update(eligible=True,reason='ok');out.append(x)
good=[x for x in out if x['eligible']]
from collections import Counter
print('ELIGIBLE',len(good),'UNIQUE',len(set(x['symbol'] for x in good)),'YEARS',Counter(datetime.datetime.fromtimestamp(x['release_time']/1000,UTC).year for x in good),'REASONS',Counter(x['reason'] for x in out),flush=True)
for x in good:print('OK',datetime.datetime.fromtimestamp(x['release_time']/1000,UTC).isoformat(),x['symbol'],round(x['quote_volume_24h']),x['article_title'])
Path('/tmp/simple_earn_events.json').write_text(json.dumps(events,ensure_ascii=False,indent=2));Path('/tmp/simple_earn_eligibility.json').write_text(json.dumps(out,ensure_ascii=False,indent=2))
