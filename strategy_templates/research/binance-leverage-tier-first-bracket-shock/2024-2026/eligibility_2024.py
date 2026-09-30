import csv,datetime,io,json,time,urllib.request,urllib.error,zipfile
from functools import lru_cache
UTC=datetime.timezone.utc
UA='Mozilla/5.0'
BASE='https://data.binance.vision/data/futures/um/monthly/klines'
events=[
('2024-08-22T09:30:00+00:00','CRVUSDT','LONG',25,75,'572598a13c384de6aa3c4d4dc1be5f81'),
('2024-11-11T08:30:00+00:00','COMPUSDT','LONG',25,75,'c178899ac5614333bf50d35cf5ff8b81'),
('2024-11-11T08:30:00+00:00','CTSIUSDT','LONG',25,75,'c178899ac5614333bf50d35cf5ff8b81'),
('2024-11-11T08:30:00+00:00','BICOUSDT','LONG',20,75,'c178899ac5614333bf50d35cf5ff8b81'),
('2024-11-11T08:30:00+00:00','DUSKUSDT','LONG',25,75,'c178899ac5614333bf50d35cf5ff8b81'),
('2024-11-11T08:30:00+00:00','XVSUSDT','LONG',20,75,'c178899ac5614333bf50d35cf5ff8b81'),
('2024-11-11T08:30:00+00:00','MAVUSDT','LONG',20,75,'c178899ac5614333bf50d35cf5ff8b81'),
('2024-11-11T08:30:00+00:00','1000XECUSDT','LONG',20,75,'c178899ac5614333bf50d35cf5ff8b81'),
('2024-11-11T08:30:00+00:00','KEYUSDT','SHORT',20,10,'c178899ac5614333bf50d35cf5ff8b81'),
('2024-11-11T08:30:00+00:00','RENUSDT','SHORT',50,20,'c178899ac5614333bf50d35cf5ff8b81'),
('2024-12-06T08:50:00+00:00','FTMUSDT','SHORT',75,50,'4cdde0518d1644c898df0d40080497a2'),
('2024-12-11T09:00:00+00:00','FTMUSDT','SHORT',50,20,'76800beea6d6459d902e9f5dcc486c01'),
('2024-12-11T09:00:00+00:00','BLZUSDT','SHORT',20,10,'76800beea6d6459d902e9f5dcc486c01'),
('2024-12-11T09:00:00+00:00','ZRXUSDT','LONG',50,75,'76800beea6d6459d902e9f5dcc486c01'),
('2024-12-11T09:00:00+00:00','DASHUSDT','LONG',50,75,'76800beea6d6459d902e9f5dcc486c01'),
]
def dt(s):return datetime.datetime.fromisoformat(s).astimezone(UTC)
def mon(x):return x.strftime('%Y-%m')
def sub2(x):
    try:return x.replace(year=x.year-2)
    except:return x.replace(year=x.year-2,day=28)
def prevmon(x):
    y,m=x.year,x.month
    if m==1:return f'{y-1}-12'
    return f'{y:04d}-{m-1:02d}'
@lru_cache(None)
def head(sym,m):
    u=f'{BASE}/{sym}/1h/{sym}-1h-{m}.zip'
    try:
        req=urllib.request.Request(u,method='HEAD',headers={'User-Agent':UA});urllib.request.urlopen(req,timeout=15).close();return True
    except urllib.error.HTTPError as e:return False if e.code==404 else False
    except:return False
@lru_cache(None)
def rows(sym,m):
    u=f'{BASE}/{sym}/1h/{sym}-1h-{m}.zip'
    try:b=urllib.request.urlopen(urllib.request.Request(u,headers={'User-Agent':UA}),timeout=25).read()
    except:return []
    z=zipfile.ZipFile(io.BytesIO(b));fn=z.namelist()[0];out=[]
    for r in csv.reader(io.TextIOWrapper(z.open(fn))):
        if not r or not r[0].isdigit():continue
        t=int(r[0]);t=t//1000 if t>10**15 else t
        out.append((t,float(r[1]),float(r[4]),float(r[7])))
    return out
res=[]
for s,sym,side,before,after,src in events:
    e=dt(s);em=mon(e);rec={'event_utc':s,'event_ms':int(e.timestamp()*1000),'symbol':sym,'side':side,'before':before,'after':after,'source_id':src}
    if not head(sym,em):rec.update(eligible=False,reason='no_event_month');res.append(rec);continue
    cutoff=sub2(e);cm=mon(cutoff)
    if not head(sym,cm):rec.update(eligible=False,reason='history_lt_2y');res.append(rec);continue
    hist=rows(sym,cm);cut=int(cutoff.timestamp()*1000)
    if not hist or hist[0][0]>cut:rec.update(eligible=False,reason='history_lt_2y');res.append(rec);continue
    hour=(rec['event_ms']//3600000)*3600000
    data=rows(sym,prevmon(e))+rows(sym,em)
    prior=[x for x in data if hour-24*3600000<=x[0]<hour]
    qv=sum(x[3] for x in prior);rec['quote_volume_24h']=qv
    if len(prior)<24:rec.update(eligible=False,reason='missing_24h');res.append(rec);continue
    if qv<5_000_000:rec.update(eligible=False,reason='qv_lt_5m');res.append(rec);continue
    rec.update(eligible=True,reason='ok');res.append(rec)
print(json.dumps(res,indent=2))
print('ELIGIBLE',sum(x['eligible'] for x in res),'/',len(res))
open('/tmp/leverage_tier_eligible.json','w').write(json.dumps(res,indent=2))
