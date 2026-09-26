import json,urllib.request,urllib.error,zipfile,io,csv,datetime,time
from concurrent.futures import ThreadPoolExecutor,as_completed
from functools import lru_cache

ROOT=Path(__file__).resolve().parent
EV=json.load(open(ROOT/'inputs'/'event_universe.json'))
BASE='https://data.binance.vision/data/futures/um/monthly/klines'
UA='Mozilla/5.0'
UTC=datetime.timezone.utc

def dt(s): return datetime.datetime.fromisoformat(s).astimezone(UTC)
def sub2(x):
    try:return x.replace(year=x.year-2)
    except ValueError:return x.replace(year=x.year-2,day=28)
def mon(x):return x.strftime('%Y-%m')
def prevmon(x):
    y,m=x.year,x.month
    if m==1:y-=1;m=12
    else:m-=1
    return f'{y:04d}-{m:02d}'
def url(sym,interval,month):
    return f'{BASE}/{sym}/{interval}/{sym}-{interval}-{month}.zip'
@lru_cache(None)
def head(sym,interval,month):
    u=url(sym,interval,month)
    for k in range(4):
        try:
            req=urllib.request.Request(u,method='HEAD',headers={'User-Agent':UA})
            with urllib.request.urlopen(req,timeout=15) as r:return True
        except urllib.error.HTTPError as e:
            if e.code==404:return False
            if e.code in (429,500,502,503,504):time.sleep(.4*(k+1));continue
            return False
        except Exception:
            time.sleep(.4*(k+1))
    return False
@lru_cache(None)
def rows(sym,month):
    u=url(sym,'1h',month)
    for k in range(4):
        try:
            req=urllib.request.Request(u,headers={'User-Agent':UA})
            b=urllib.request.urlopen(req,timeout=20).read()
            z=zipfile.ZipFile(io.BytesIO(b));fn=z.namelist()[0]
            out=[]
            for r in csv.reader(io.TextIOWrapper(z.open(fn))):
                if not r or not r[0].isdigit():continue
                t=int(r[0]); 
                if t>10**15:t//=1000
                out.append((t,float(r[1]),float(r[4]),float(r[7]) if len(r)>7 else 0.0))
            return out
        except urllib.error.HTTPError as e:
            if e.code==404:return []
            time.sleep(.4*(k+1))
        except Exception:
            time.sleep(.4*(k+1))
    return []

# Build candidate symbol mapping with parallel HEADs.
pairs=[]
for n in EV:
    event=dt(n['first_listed_at'])
    for ticker in n['tickers']:
        pairs.append((n,event,ticker))
keys=set()
for n,event,ticker in pairs:
    m=mon(event)
    keys.add((ticker+'USDT','1h',m));keys.add(('1000'+ticker+'USDT','1h',m))
with ThreadPoolExecutor(max_workers=16) as ex:
    fut={ex.submit(head,*k):k for k in keys}
    for i,f in enumerate(as_completed(fut),1):
        f.result()
        if i%30==0:print('HEAD',i,'/',len(keys),flush=True)

results=[]
for n,event,ticker in pairs:
    em=mon(event)
    sym=ticker+'USDT'
    if not head(sym,'1h',em) and head('1000'+ticker+'USDT','1h',em):
        sym='1000'+ticker+'USDT'
    rec={'notice_id':n['id'],'uuid':n['uuid'],'title':n['title'],'ticker':ticker,'symbol':sym,
         'event_ms':int(event.timestamp()*1000),'event_utc':event.isoformat()}
    if not head(sym,'1h',em):
        rec.update(eligible=False,reason='no_event_month');results.append(rec);continue
    cutoff=sub2(event); cm=mon(cutoff)
    if not head(sym,'1h',cm):
        rec.update(eligible=False,reason='no_24m_month');results.append(rec);continue
    hist=rows(sym,cm)
    cutoff_ms=int(cutoff.timestamp()*1000)
    if not hist or hist[0][0]>cutoff_ms:
        rec.update(eligible=False,reason='history_lt_2y',first_hist=hist[0][0] if hist else None);results.append(rec);continue
    # Previous 24 complete hours before announcement.
    hour=(rec['event_ms']//3600000)*3600000
    data=rows(sym,prevmon(event))+rows(sym,em)
    prior=[r for r in data if hour-24*3600000<=r[0]<hour]
    qv=sum(r[3] for r in prior)
    rec['quote_volume_24h']=qv;rec['bars_24h']=len(prior)
    if len(prior)<24:
        rec.update(eligible=False,reason='missing_24h_bars');results.append(rec);continue
    if qv<5_000_000:
        rec.update(eligible=False,reason='quote_volume_lt_5m');results.append(rec);continue
    after=next((r for r in rows(sym,em) if r[0]>=rec['event_ms']),None)
    if not after:
        rec.update(eligible=False,reason='not_trading_after_event');results.append(rec);continue
    rec.update(eligible=True,reason='ok',age_days=(rec['event_ms']-hist[0][0])/86400000)
    results.append(rec)

json.dump(results,open(ROOT/'inputs'/'eligibility.json','w'),ensure_ascii=False,indent=2)
eligible=[r for r in results if r['eligible']]
from collections import Counter
print('TOTAL',len(results),'ELIGIBLE',len(eligible),'UNIQUE_SYMBOLS',len(set(r['symbol'] for r in eligible)))
print('REASONS',Counter(r['reason'] for r in results))
for r in eligible:print('ELIGIBLE',r['event_utc'],r['symbol'],'qv',round(r['quote_volume_24h']),'notice',r['notice_id'])
