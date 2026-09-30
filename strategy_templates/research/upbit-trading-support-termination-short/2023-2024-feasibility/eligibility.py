import urllib.request,urllib.error,json,re,time,datetime,csv,io,zipfile
from functools import lru_cache
UTC=datetime.timezone.utc
UA='Mozilla/5.0'
UP='https://api-manager.upbit.com/api/v1/announcements'
BV='https://data.binance.vision/data/futures/um/monthly/klines'
def getj(u):
    for k in range(6):
        try:
            req=urllib.request.Request(u,headers={'User-Agent':UA,'Accept':'application/json'})
            return json.loads(urllib.request.urlopen(req,timeout=20).read())
        except urllib.error.HTTPError as e:
            if e.code==429:
                time.sleep(2*(k+1));continue
            raise
    return {}
events=[]
for page in range(8,22):
    o=getj(f'{UP}?os=web&page={page}&per_page=30&category=trade')
    ns=((o.get('data') or {}).get('notices') or [])
    if not ns:break
    years=[]
    for n in ns:
        first=n.get('first_listed_at') or n.get('listed_at')
        if not first:continue
        years.append(int(first[:4]))
        title=n.get('title','')
        if first[:4] not in ('2023','2024') or '거래지원 종료 안내' not in title:continue
        toks=[t for t in re.findall(r'(?<![A-Z0-9])([A-Z][A-Z0-9]{0,19})(?![A-Z0-9])',title) if t not in {'KRW','BTC','USDT'}]
        seen=[];ss=set()
        for t in toks:
            if t not in ss:
                ss.add(t);seen.append(t)
        events.append({'event_time':first,'title':title,'tickers':seen})
    print('PAGE',page,'EVENTS',len(events),flush=True)
    if years and min(years)<2023:break
    time.sleep(1.2)
print('ARTICLES',len(events),'TOKENS',sum(len(x['tickers']) for x in events),'UNIQUE',len(set(t for x in events for t in x['tickers'])),flush=True)
for x in events:
    print('EV',x['event_time'],x['title'],x['tickers'],flush=True)
def dt(s):return datetime.datetime.fromisoformat(s).astimezone(UTC)
def mon(d):return d.strftime('%Y-%m')
def prevm(d):return f'{d.year-1}-12' if d.month==1 else f'{d.year:04d}-{d.month-1:02d}'
def sub2(d):
    try:return d.replace(year=d.year-2)
    except:return d.replace(year=d.year-2,day=28)
@lru_cache(None)
def head(sym,mo):
    u=f'{BV}/{sym}/1h/{sym}-1h-{mo}.zip'
    try:
        urllib.request.urlopen(urllib.request.Request(u,method='HEAD',headers={'User-Agent':UA}),timeout=12).close()
        return True
    except:return False
@lru_cache(None)
def rows(sym,mo):
    u=f'{BV}/{sym}/1h/{sym}-1h-{mo}.zip'
    try:
        b=urllib.request.urlopen(urllib.request.Request(u,headers={'User-Agent':UA}),timeout=20).read()
    except:return []
    z=zipfile.ZipFile(io.BytesIO(b));fn=z.namelist()[0];out=[]
    for r in csv.reader(io.TextIOWrapper(z.open(fn))):
        if not r or not r[0].isdigit():continue
        t=int(r[0]);t=t//1000 if t>10**15 else t
        out.append((t,float(r[7]) if len(r)>7 else 0.0))
    return out
elig=[]
for a in events:
    e=dt(a['event_time']);ems=int(e.timestamp()*1000)
    for ticker in a['tickers']:
        sym=ticker+'USDT'
        if not head(sym,mon(e)) and head('1000'+ticker+'USDT',mon(e)):
            sym='1000'+ticker+'USDT'
        rec={'symbol':sym,'ticker':ticker,'event_ms':ems,'event_time':e.isoformat(),'title':a['title']}
        if not head(sym,mon(e)):
            rec.update(eligible=False,reason='no_event_month');elig.append(rec);continue
        cutoff=sub2(e);hist=rows(sym,mon(cutoff))
        if not hist or hist[0][0]>int(cutoff.timestamp()*1000):
            rec.update(eligible=False,reason='history_lt_2y');elig.append(rec);continue
        hour=(ems//3600000)*3600000
        prior=rows(sym,prevm(e))+rows(sym,mon(e))
        prior=[r for r in prior if hour-86400000<=r[0]<hour]
        qv=sum(r[1] for r in prior)
        rec['bars_24h']=len(prior);rec['quote_volume_24h']=qv
        if len(prior)<24:
            rec.update(eligible=False,reason='missing_24h');elig.append(rec);continue
        if qv<5_000_000:
            rec.update(eligible=False,reason='qv_lt_5m');elig.append(rec);continue
        rec.update(eligible=True,reason='ok');elig.append(rec)
good=[x for x in elig if x['eligible']]
from collections import Counter
print('ELIGIBLE',len(good),'UNIQUE_SYMBOLS',len(set(x['symbol'] for x in good)),'REASONS',Counter(x['reason'] for x in elig),flush=True)
for x in good:
    print('OK',x['event_time'],x['symbol'],round(x['quote_volume_24h']),x['title'],flush=True)
open('/tmp/upbit_termination_events.json','w').write(json.dumps(events,ensure_ascii=False,indent=2))
open('/tmp/upbit_termination_eligibility.json','w').write(json.dumps(elig,ensure_ascii=False,indent=2))
