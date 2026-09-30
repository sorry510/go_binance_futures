import csv,io,json,re,time,urllib.request,urllib.error,zipfile,datetime
from concurrent.futures import ThreadPoolExecutor,as_completed
from functools import lru_cache
from pathlib import Path

UA='Mozilla/5.0'
UTC=datetime.timezone.utc
UP='https://api-manager.upbit.com/api/v1/announcements'
BV='https://data.binance.vision/data/futures/um/monthly'
OUT=Path('/tmp/upbit_risk_2023_2024')
OUT.mkdir(exist_ok=True)

def get_json(url):
    last=None
    for k in range(6):
        try:
            req=urllib.request.Request(url,headers={'User-Agent':UA,'Accept':'application/json'})
            return json.loads(urllib.request.urlopen(req,timeout=20).read())
        except urllib.error.HTTPError as e:
            last=e
            if e.code==429:
                time.sleep(3*(k+1)); continue
            raise
    raise last

events=[]
for page in range(9,17):
    o=get_json(f'{UP}?os=web&page={page}&per_page=30&category=trade')
    notices=((o.get('data') or {}).get('notices') or [])
    for n in notices:
        first=n.get('first_listed_at') or n.get('listed_at')
        title=n.get('title','')
        if not first or first[:4] not in ('2023','2024'): continue
        risk=(('유의 종목 지정 안내' in title) or ('투자 유의 촉구' in title) or ('유의 촉구 안내' in title))
        if not risk or '기간 연장' in title or '해제' in title or '거래지원 종료' in title: continue
        toks=[t for t in re.findall(r'(?<![A-Z0-9])([A-Z][A-Z0-9]{0,19})(?![A-Z0-9])',title) if t not in {'KRW','BTC','USDT'}]
        seen=[];ss=set()
        for t in toks:
            if t not in ss: ss.add(t);seen.append(t)
        events.append({'id':n.get('id'),'uuid':n.get('uuid'),'first_listed_at':first,'listed_at':n.get('listed_at'),'title':title,'tickers':seen})
    time.sleep(1.5)
(OUT/'event_universe.json').write_text(json.dumps(events,ensure_ascii=False,indent=2))
print('EVENTS',len(events),'TOKEN_EVENTS',sum(len(e['tickers']) for e in events),'UNIQUE',len(set(t for e in events for t in e['tickers'])),flush=True)

def dt(s): return datetime.datetime.fromisoformat(s).astimezone(UTC)
def mon(x): return x.strftime('%Y-%m')
def sub2(x):
    try:return x.replace(year=x.year-2)
    except ValueError:return x.replace(year=x.year-2,day=28)
def prevmon(x):
    y,m=x.year,x.month
    if m==1:y-=1;m=12
    else:m-=1
    return f'{y:04d}-{m:02d}'

@lru_cache(None)
def head(sym,interval,month):
    u=f'{BV}/klines/{sym}/{interval}/{sym}-{interval}-{month}.zip'
    for k in range(4):
        try:
            req=urllib.request.Request(u,method='HEAD',headers={'User-Agent':UA})
            with urllib.request.urlopen(req,timeout=15): return True
        except urllib.error.HTTPError as e:
            if e.code==404:return False
            if e.code in (429,500,502,503,504):time.sleep(.4*(k+1));continue
            return False
        except Exception: time.sleep(.4*(k+1))
    return False

@lru_cache(None)
def rows1h(sym,month):
    u=f'{BV}/klines/{sym}/1h/{sym}-1h-{month}.zip'
    for k in range(4):
        try:
            b=urllib.request.urlopen(urllib.request.Request(u,headers={'User-Agent':UA}),timeout=20).read()
            z=zipfile.ZipFile(io.BytesIO(b));fn=z.namelist()[0];out=[]
            for r in csv.reader(io.TextIOWrapper(z.open(fn))):
                if not r or not r[0].isdigit():continue
                t=int(r[0]); t=t//1000 if t>10**15 else t
                out.append((t,float(r[1]),float(r[4]),float(r[7]) if len(r)>7 else 0.0))
            return out
        except urllib.error.HTTPError as e:
            if e.code==404:return []
            time.sleep(.4*(k+1))
        except Exception: time.sleep(.4*(k+1))
    return []

pairs=[]
for n in events:
    e=dt(n['first_listed_at'])
    for ticker in n['tickers']: pairs.append((n,e,ticker))
keys=set()
for _,e,t in pairs:
    m=mon(e);keys.add((t+'USDT','1h',m));keys.add(('1000'+t+'USDT','1h',m))
with ThreadPoolExecutor(max_workers=12) as ex:
    fs=[ex.submit(head,*k) for k in keys]
    for f in as_completed(fs): f.result()

elig=[]
for n,e,ticker in pairs:
    em=mon(e);sym=ticker+'USDT'
    if not head(sym,'1h',em) and head('1000'+ticker+'USDT','1h',em): sym='1000'+ticker+'USDT'
    rec={'notice_id':n['id'],'uuid':n['uuid'],'title':n['title'],'ticker':ticker,'symbol':sym,'event_ms':int(e.timestamp()*1000),'event_utc':e.isoformat()}
    if not head(sym,'1h',em): rec.update(eligible=False,reason='no_event_month');elig.append(rec);continue
    cutoff=sub2(e);cm=mon(cutoff)
    if not head(sym,'1h',cm): rec.update(eligible=False,reason='no_24m_month');elig.append(rec);continue
    hist=rows1h(sym,cm);cut=int(cutoff.timestamp()*1000)
    if not hist or hist[0][0]>cut: rec.update(eligible=False,reason='history_lt_2y');elig.append(rec);continue
    hour=(rec['event_ms']//3600000)*3600000
    prior=[r for r in rows1h(sym,prevmon(e))+rows1h(sym,em) if hour-24*3600000<=r[0]<hour]
    qv=sum(r[3] for r in prior);rec['quote_volume_24h']=qv;rec['bars_24h']=len(prior)
    if len(prior)<24: rec.update(eligible=False,reason='missing_24h_bars');elig.append(rec);continue
    if qv<5_000_000: rec.update(eligible=False,reason='quote_volume_lt_5m');elig.append(rec);continue
    rec.update(eligible=True,reason='ok');elig.append(rec)
(OUT/'eligibility.json').write_text(json.dumps(elig,ensure_ascii=False,indent=2))
good=[x for x in elig if x['eligible']]
from collections import Counter
print('ELIGIBLE',len(good),'UNIQUE_SYMBOLS',len(set(x['symbol'] for x in good)),'REASONS',Counter(x['reason'] for x in elig),flush=True)
for x in good: print('OK',x['event_utc'],x['symbol'],round(x['quote_volume_24h']),x['title'],flush=True)
