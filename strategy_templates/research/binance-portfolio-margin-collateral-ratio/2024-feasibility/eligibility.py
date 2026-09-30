import csv,datetime,io,json,urllib.request,zipfile
from functools import lru_cache
UTC=datetime.timezone.utc
UA='Mozilla/5.0'
BASE='https://data.binance.vision/data/futures/um/monthly/klines'
E=[]
def add(ts, pairs, side, src):
    for sym,b,a in pairs:
        E.append((ts,sym,side,b,a,src))

# 2024-06-10 decreases
add('2024-06-10T06:00:00+00:00',[
 ('LINKUSDT',90,80),('AAVEUSDT',90,80),('PAXGUSDT',90,80),('DOTUSDT',90,80),
 ('MATICUSDT',85,80),('FLUXUSDT',70,40),('PYRUSDT',70,40),('WOOUSDT',70,40)
],'SHORT','0806a835368b409e8d5ebd84d9fdc4ed')
# 2024-06-28 increases
add('2024-06-28T06:00:00+00:00',[
 ('PEPEUSDT',45,60),('WIFUSDT',45,60),('NOTUSDT',10,40),('LISTAUSDT',10,30),
 ('ZROUSDT',10,30),('ZKUSDT',10,30),('IOUSDT',10,30)
],'LONG','4e842671311c4ae9814e4da9ef47fcef')
# 2024-07-30 increases
add('2024-07-30T06:00:00+00:00',[
 ('PEPEUSDT',60,75),('NOTUSDT',40,75),('NEARUSDT',45,75),('BONKUSDT',50,75),
 ('ZROUSDT',30,50),('IOUSDT',30,50),('BBUSDT',10,35),('BANANAUSDT',10,25)
],'LONG','3f0978bcf69643aeb29424d45c9b810c')
# 2024-09-03 increases
add('2024-09-03T06:00:00+00:00',[
 ('FLOKIUSDT',35,40),('TONUSDT',20,30),('DOGSUSDT',10,30),('RENDERUSDT',10,30),
 ('SUNUSDT',10,30),('RAREUSDT',10,30),('ADXUSDT',10,20)
],'LONG','b9fad723c9c64240801d0e11dd77faa8')
# 2024-11-29 increases
add('2024-11-29T06:00:00+00:00',[
 ('NEIROUSDT',40,50),('APTUSDT',45,50),('BOMEUSDT',45,50),('WLDUSDT',45,50),
 ('PNUTUSDT',10,35),('ACTUSDT',10,25)
],'LONG','51bb9a7a59d2472cbd0c2b3ff38fe9bd')

def dt(s): return datetime.datetime.fromisoformat(s).astimezone(UTC)
def mon(x): return x.strftime('%Y-%m')
def sub2(x):
    try:return x.replace(year=x.year-2)
    except:return x.replace(year=x.year-2,day=28)
def prevmon(x):
    y,m=x.year,x.month
    return f'{y-1}-12' if m==1 else f'{y:04d}-{m-1:02d}'

@lru_cache(None)
def head(sym,m):
    u=f'{BASE}/{sym}/1h/{sym}-1h-{m}.zip'
    try:
        urllib.request.urlopen(urllib.request.Request(u,method='HEAD',headers={'User-Agent':UA}),timeout=12).close()
        return True
    except:return False

@lru_cache(None)
def rows(sym,m):
    u=f'{BASE}/{sym}/1h/{sym}-1h-{m}.zip'
    try:b=urllib.request.urlopen(urllib.request.Request(u,headers={'User-Agent':UA}),timeout=20).read()
    except:return []
    z=zipfile.ZipFile(io.BytesIO(b));fn=z.namelist()[0];out=[]
    for r in csv.reader(io.TextIOWrapper(z.open(fn))):
        if not r or not r[0].isdigit():continue
        t=int(r[0]);t=t//1000 if t>10**15 else t
        out.append((t,float(r[1]),float(r[4]),float(r[7])))
    return out

res=[]
for s,sym,side,before,after,src in E:
    e=dt(s);em=mon(e)
    rec={'event_utc':s,'event_ms':int(e.timestamp()*1000),'symbol':sym,'side':side,'before':before,'after':after,'source_id':src}
    if not head(sym,em):
        rec.update(eligible=False,reason='no_event_month');res.append(rec);continue
    cutoff=sub2(e);cm=mon(cutoff)
    if not head(sym,cm):
        rec.update(eligible=False,reason='history_lt_2y');res.append(rec);continue
    hist=rows(sym,cm);cut=int(cutoff.timestamp()*1000)
    if not hist or hist[0][0]>cut:
        rec.update(eligible=False,reason='history_lt_2y');res.append(rec);continue
    hour=(rec['event_ms']//3600000)*3600000
    data=rows(sym,prevmon(e))+rows(sym,em)
    prior=[x for x in data if hour-24*3600000<=x[0]<hour]
    qv=sum(x[3] for x in prior);rec['quote_volume_24h']=qv
    if len(prior)<24:
        rec.update(eligible=False,reason='missing_24h');res.append(rec);continue
    if qv<5_000_000:
        rec.update(eligible=False,reason='qv_lt_5m');res.append(rec);continue
    rec.update(eligible=True,reason='ok');res.append(rec)

from collections import Counter
print('TOTAL',len(res),'ELIGIBLE',sum(x['eligible'] for x in res),'REASONS',Counter(x['reason'] for x in res))
for x in res:
    if x['eligible']:
        print('OK',x['event_utc'],x['symbol'],x['side'],x['before'],'->',x['after'],round(x.get('quote_volume_24h',0)))
open('/tmp/collateral_ratio_2024_eligible.json','w').write(json.dumps(res,indent=2))
