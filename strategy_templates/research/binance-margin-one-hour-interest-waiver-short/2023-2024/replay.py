import csv,io,zipfile,urllib.request,urllib.error,time,datetime,math,json
from functools import lru_cache
from pathlib import Path
UTC=datetime.timezone.utc
BASE='https://data.binance.vision/data/futures/um/monthly/klines'
UA='Mozilla/5.0'
CACHE=Path('/tmp/margin_waiver_cache');CACHE.mkdir(exist_ok=True)
BATCHES=[
('2023-08-14T06:00:00+00:00',['BTC','ETH','DOGE','SHIB','XRP','LTC','PEPE','OP']),
('2023-09-13T06:00:00+00:00',['BTC','ETH','XRP','SOL','DOGE','OP','ARB','SHIB','LINK','LTC']),
('2023-12-04T09:00:00+00:00',['BTC','DOGE','ETH','GALA','GMT','LINK','MATIC','ORDI','SEI','SOL','TIA','XRP']),
('2024-02-27T09:00:00+00:00',['BTC','ETH','SOL','MATIC','XRP','FIL','WLD','LPT','DOGE','HBAR','VET','GRT','ARB','OP','LINK','FET','AGIX']),
('2024-03-12T11:00:00+00:00',['1000SATS','BNX','BONK','DOGE','FLOKI','GALA','MEME','PEPE','PIXEL','PORTAL','SHIB','WIF','YGG']),
('2024-05-08T11:00:00+00:00',['USDC','BTC','BOME','DOGE','ETH','ENA','FET','FLOKI','NEAR','REZ','RNDR','SOL','SHIB','PEPE','WLD','WIF']),
('2024-11-11T11:00:00+00:00',['BTC','DOGE','ETH','NEIRO','SHIB','SOL','SUI','PEPE','XRP']),
]
ALIASES={'SHIB':['1000SHIBUSDT','SHIBUSDT'],'PEPE':['1000PEPEUSDT','PEPEUSDT'],'BONK':['1000BONKUSDT','BONKUSDT'],'FLOKI':['1000FLOKIUSDT','FLOKIUSDT']}
def mon(d):return d.strftime('%Y-%m')
def prevmon(d):
    return f'{d.year-1}-12' if d.month==1 else f'{d.year:04d}-{d.month-1:02d}'
def sub2(d):
    try:return d.replace(year=d.year-2)
    except ValueError:return d.replace(year=d.year-2,day=28)
def candidates(tok):
    if tok in ALIASES:return ALIASES[tok]
    if tok.endswith('USDT'):return [tok]
    return [tok+'USDT','1000'+tok+'USDT']
def get(sym,mo):
    p=CACHE/f'{sym}-{mo}.zip'
    if p.exists():return p
    u=f'{BASE}/{sym}/1h/{sym}-1h-{mo}.zip'
    for k in range(4):
        try:
            b=urllib.request.urlopen(urllib.request.Request(u,headers={'User-Agent':UA}),timeout=20).read();p.write_bytes(b);return p
        except urllib.error.HTTPError as e:
            if e.code==404:p.write_bytes(b'');return p
            if e.code in (429,500,502,503,504):time.sleep(.4*(k+1));continue
            raise
        except Exception:
            time.sleep(.4*(k+1))
    return p
@lru_cache(None)
def rows(sym,mo):
    p=get(sym,mo)
    if not p.exists() or p.stat().st_size==0:return []
    z=zipfile.ZipFile(p);fn=z.namelist()[0];out=[]
    for r in csv.reader(io.TextIOWrapper(z.open(fn))):
        if not r or not r[0].isdigit():continue
        t=int(r[0]);t=t//1000 if t>10**15 else t
        out.append((t,float(r[1]),float(r[4]),float(r[7]) if len(r)>7 else 0.0))
    return out
def resolve(tok,e):
    for sym in candidates(tok):
        if rows(sym,mon(e)):return sym
    return candidates(tok)[0]
good=[]; excluded=[]
for ts,toks in BATCHES:
    e=datetime.datetime.fromisoformat(ts); ems=int(e.timestamp()*1000)
    for tok in toks:
        sym=resolve(tok,e)
        hist=rows(sym,mon(sub2(e))); cutoff=int(sub2(e).timestamp()*1000)
        if not hist or hist[0][0]>cutoff:
            excluded.append((ts,tok,sym,'history_lt_2y'));continue
        hour=(ems//3600000)*3600000
        data=rows(sym,prevmon(e))+rows(sym,mon(e))
        prior=[r for r in data if hour-24*3600000<=r[0]<hour]
        q=sum(r[3] for r in prior)
        if len(prior)<24 or q<5_000_000:
            excluded.append((ts,tok,sym,'liq_lt_5m'));continue
        fut=[r for r in rows(sym,mon(e)) if r[0]>=ems]
        if len(fut)<12:
            excluded.append((ts,tok,sym,'missing_forward'));continue
        entry=fut[0][1]
        r1=-math.log(fut[0][2]/entry);r4=-math.log(fut[3][2]/entry);r12=-math.log(fut[11][2]/entry)
        year=e.year
        good.append({'event':ts,'token':tok,'symbol':sym,'year':year,'q24':q,'r1':r1,'r4':r4,'r12':r12})
        print('OK',ts,tok,sym,'r12%',round(r12*100,3),flush=True)
def summary(xs):
    if not xs:return {}
    sy=set(x['symbol'] for x in xs)
    return {'n':len(xs),'mean1':sum(x['r1'] for x in xs)/len(xs),'mean4':sum(x['r4'] for x in xs)/len(xs),'mean12':sum(x['r12'] for x in xs)/len(xs),
            'win12':sum(x['r12']>0 for x in xs)/len(xs),
            'positive_symbols12':sum((sum(x['r12'] for x in xs if x['symbol']==s)/len([x for x in xs if x['symbol']==s]))>0 for s in sy),
            'symbols':len(sy)}
res={'discovery_2023':summary([x for x in good if x['year']==2023]),
     'oos1_2024':summary([x for x in good if x['year']==2024]),
     'all':summary(good),
     'eligible_by_batch':{},
     'excluded_count':len(excluded)}
for ts,_ in BATCHES:
    res['eligible_by_batch'][ts]=len([x for x in good if x['event']==ts])
print('SUMMARY',json.dumps(res,indent=2),flush=True)
print('EXCLUDED_REASONS')
from collections import Counter
print(Counter(x[3] for x in excluded))
Path('/tmp/margin_waiver_short_summary.json').write_text(json.dumps(res,indent=2))
Path('/tmp/margin_waiver_short_events.json').write_text(json.dumps(good,indent=2))
