import csv,io,zipfile,urllib.request,urllib.error,time,datetime,json,math
from concurrent.futures import ThreadPoolExecutor,as_completed
from pathlib import Path
UTC=datetime.timezone.utc;UA='Mozilla/5.0'
SYMS=['BTCUSDT','ETHUSDT','BNBUSDT','XRPUSDT','SOLUSDT','DOGEUSDT','LTCUSDT','AVAXUSDT','UNIUSDT','ZECUSDT']
START=datetime.date(2022,10,1);END=datetime.date(2026,8,1)
CACHE=Path('/tmp/cross_funding_cache');CACHE.mkdir(exist_ok=True)
def months():
    y,m=START.year,START.month;out=[]
    while (y,m)<=(END.year,END.month):
        out.append(f'{y:04d}-{m:02d}');m+=1
        if m==13:y+=1;m=1
    return out
def get(url,tries=5):
    last=None
    for k in range(tries):
        try:return urllib.request.urlopen(urllib.request.Request(url,headers={'User-Agent':UA}),timeout=25).read()
        except urllib.error.HTTPError as e:
            if e.code==404:return None
            last=e
            if e.code in (403,418,429,500,502,503,504):time.sleep(.5*(k+1));continue
            raise
        except Exception as e:last=e;time.sleep(.4*(k+1))
    raise last
def fetch_bin(sym,mo,kind):
    p=CACHE/f'bin-{kind}-{sym}-{mo}.zip'
    if p.exists():return p
    base='https://data.binance.vision/data/futures/um/monthly'
    url=f'{base}/fundingRate/{sym}/{sym}-fundingRate-{mo}.zip' if kind=='fund' else f'{base}/klines/{sym}/1h/{sym}-1h-{mo}.zip'
    b=get(url);p.write_bytes(b or b'');return p
jobs=[(s,m,k) for s in SYMS for m in months() for k in ('fund','kline')]
with ThreadPoolExecutor(max_workers=16) as ex:
    fs=[ex.submit(fetch_bin,*j) for j in jobs]
    for i,f in enumerate(as_completed(fs),1):
        f.result()
        if i%200==0:print('BINANCE',i,'/',len(jobs),flush=True)
def parse_bin_fund(sym):
    out={}
    for mo in months():
        p=CACHE/f'bin-fund-{sym}-{mo}.zip'
        if not p.exists() or p.stat().st_size==0:continue
        z=zipfile.ZipFile(p);fn=z.namelist()[0];rr=csv.reader(io.TextIOWrapper(z.open(fn)));hdr=next(rr,None)
        names=[str(x).strip().lower() for x in (hdr or [])]
        ti=names.index('calc_time') if 'calc_time' in names else 0
        ri=names.index('last_funding_rate') if 'last_funding_rate' in names else 2
        ii=names.index('funding_interval_hours') if 'funding_interval_hours' in names else None
        for r in rr:
            try:
                t=int(r[ti]);t=t//1000 if t>10**15 else t;rate=float(r[ri]);interval=float(r[ii]) if ii is not None and r[ii] else 8.0
                if interval>0:out[t]=rate/interval
            except:pass
    return out
def parse_klines(sym):
    out={}
    for mo in months():
        p=CACHE/f'bin-kline-{sym}-{mo}.zip'
        if not p.exists() or p.stat().st_size==0:continue
        z=zipfile.ZipFile(p);fn=z.namelist()[0]
        for r in csv.reader(io.TextIOWrapper(z.open(fn))):
            if not r or not r[0].isdigit():continue
            t=int(r[0]);t=t//1000 if t>10**15 else t
            out[t]=(float(r[1]),float(r[4]))
    return out
def fetch_bybit(sym):
    p=CACHE/f'bybit-{sym}.json'
    if p.exists():return json.loads(p.read_text())
    start=int(datetime.datetime(2022,10,1,tzinfo=UTC).timestamp()*1000);end=int(datetime.datetime(2026,9,1,tzinfo=UTC).timestamp()*1000)-1
    vals={};calls=0
    while end>=start:
        u=f'https://api.bybit.com/v5/market/funding/history?category=linear&symbol={sym}&endTime={end}&limit=200'
        o=json.loads(get(u));lst=((o.get('result') or {}).get('list') or [])
        if not lst:break
        calls+=1
        ts=[]
        for x in lst:
            t=int(x['fundingRateTimestamp'])
            if t>=start:vals[t]=float(x['fundingRate'])
            ts.append(t)
        oldest=min(ts);end=oldest-1
        if oldest<start:break
        time.sleep(.06)
    arr=sorted(vals.items());p.write_text(json.dumps(arr))
    print('BYBIT',sym,'records',len(arr),'calls',calls,flush=True);return arr
with ThreadPoolExecutor(max_workers=4) as ex:
    by=dict(zip(SYMS,ex.map(fetch_bybit,SYMS)))
def normalize_bybit(arr):
    out={}
    arr=sorted((int(t),float(r)) for t,r in arr)
    for i in range(1,len(arr)):
        t,r=arr[i];prev=arr[i-1][0];h=(t-prev)/3600000
        if 0.5<=h<=24:out[t]=r/h
    return out
def stats(xs):
    if not xs:return {}
    syms=sorted(set(x['symbol'] for x in xs));by={}
    for s in syms:
        q=[x for x in xs if x['symbol']==s];by[s]={'n':len(q),'mean12':sum(x['r12'] for x in q)/len(q)}
    return {'n':len(xs),'mean1':sum(x['r1'] for x in xs)/len(xs),'mean4':sum(x['r4'] for x in xs)/len(xs),'mean12':sum(x['r12'] for x in xs)/len(xs),'win12':sum(x['r12']>0 for x in xs)/len(xs),'symbols':len(syms),'positive_symbols12':sum(v['mean12']>0 for v in by.values()),'by_symbol':by}
events=[]
for sym in SYMS:
    bf=parse_bin_fund(sym);yf=normalize_bybit(by[sym]);px=parse_klines(sym);ts=sorted(set(bf)&set(yf))
    if not ts:print('NO_MATCH',sym);continue
    diffs=[(t,bf[t]-yf[t]) for t in ts];first=diffs[0][0];armed=True;n=0
    for t,d in diffs:
        if t-first<90*86400000:continue
        prior=[v for tt,v in diffs if t-90*86400000<=tt<t]
        if len(prior)<30:continue
        m=sum(prior)/len(prior);sd=(sum((v-m)**2 for v in prior)/len(prior))**.5
        if sd<=0:continue
        z=(d-m)/sd
        if abs(z)<1:armed=True
        if not armed or abs(z)<3:continue
        ent=((t//3600000)+1)*3600000
        bars=[px.get(ent+i*3600000) for i in range(13)]
        if any(b is None for b in bars):continue
        direction=-1.0 if d>0 else 1.0;entry=bars[0][0]
        r1=direction*math.log(bars[0][1]/entry);r4=direction*math.log(bars[3][1]/entry);r12=direction*math.log(bars[11][1]/entry)
        y=datetime.datetime.fromtimestamp(t/1000,UTC).year
        if y>=2023:events.append({'symbol':sym,'time':t,'year':y,'z':z,'diff_per_hour':d,'r1':r1,'r4':r4,'r12':r12})
        armed=False;n+=1
    print('SYMBOL',sym,'matched',len(ts),'events',n,flush=True)
res={'discovery_2023_2024':stats([x for x in events if x['year'] in (2023,2024)]),'oos1_2025':stats([x for x in events if x['year']==2025]),'oos2_2026':stats([x for x in events if x['year']==2026]),'all':stats(events)}
print(json.dumps(res,indent=2),flush=True)
Path('/tmp/cross_exchange_funding_summary.json').write_text(json.dumps(res,indent=2));Path('/tmp/cross_exchange_funding_events.json').write_text(json.dumps(events,indent=2))
