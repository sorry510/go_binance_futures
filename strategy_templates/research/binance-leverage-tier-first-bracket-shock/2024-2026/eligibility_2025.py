import csv,datetime,io,json,time,urllib.request,urllib.error,zipfile
from functools import lru_cache
UTC=datetime.timezone.utc
UA='Mozilla/5.0'
BASE='https://data.binance.vision/data/futures/um/monthly/klines'
events=[
('2025-01-25T15:30:00+00:00','BANUSDT','SHORT',25,10,'1a1248ee804542d691e9ce7f3e2fd948'),
('2025-01-26T11:00:00+00:00','OMUSDT','SHORT',75,10,'b9258701c4d749fe89fa31fca08f1aad'),
('2025-02-09T12:15:00+00:00','BNXUSDT','SHORT',20,10,'f271df557b024c8cb44e2e3395b7a6d8'),
('2025-03-11T09:30:00+00:00','RAREUSDT','SHORT',75,25,'e200477a2dd94a99aed490241b4be04a'),
('2025-04-27T06:30:00+00:00','ALPACAUSDT','SHORT',10,8,'93fe7d8426a34f16b1a929b3f1a11463'),
('2025-05-02T06:30:00+00:00','PEOPLEUSDT','LONG',20,75,'93fe7d8426a34f16b1a929b3f1a11463'),
('2025-05-02T06:30:00+00:00','1MBABYDOGEUSDT','LONG',20,75,'93fe7d8426a34f16b1a929b3f1a11463'),
('2025-05-02T07:30:00+00:00','THEUSDT','LONG',20,75,'93fe7d8426a34f16b1a929b3f1a11463'),
('2025-05-02T07:30:00+00:00','JELLYJELLYUSDT','LONG',20,75,'93fe7d8426a34f16b1a929b3f1a11463'),
('2025-05-02T07:30:00+00:00','BSWUSDT','LONG',10,25,'93fe7d8426a34f16b1a929b3f1a11463'),
('2025-05-02T07:30:00+00:00','JASMYUSDT','LONG',9,25,'93fe7d8426a34f16b1a929b3f1a11463'),
('2025-05-09T06:30:00+00:00','SXPUSDT','LONG',20,75,'be0b2e950851420d8b7f9f427097be08'),
('2025-05-09T06:30:00+00:00','EGLDUSDT','LONG',20,75,'be0b2e950851420d8b7f9f427097be08'),
('2025-05-09T06:30:00+00:00','1INCHUSDT','LONG',20,75,'be0b2e950851420d8b7f9f427097be08'),
('2025-05-09T06:30:00+00:00','ARUSDT','LONG',20,75,'be0b2e950851420d8b7f9f427097be08'),
('2025-05-09T06:30:00+00:00','CKBUSDT','LONG',20,75,'be0b2e950851420d8b7f9f427097be08'),
('2025-05-09T06:30:00+00:00','ZKUSDT','LONG',20,75,'be0b2e950851420d8b7f9f427097be08'),
('2025-05-09T06:30:00+00:00','HMSTRUSDT','LONG',20,75,'be0b2e950851420d8b7f9f427097be08'),
('2025-05-09T06:30:00+00:00','1000CATUSDT','LONG',20,75,'be0b2e950851420d8b7f9f427097be08'),
('2025-05-09T06:30:00+00:00','DRIFTUSDT','LONG',20,75,'be0b2e950851420d8b7f9f427097be08'),
('2025-05-09T06:30:00+00:00','ACXUSDT','LONG',20,75,'be0b2e950851420d8b7f9f427097be08'),
('2025-09-30T06:30:00+00:00','PUMPBTCUSDT','SHORT',50,25,'2571ff99a0f3480f92416699f624a584'),
('2025-09-30T06:30:00+00:00','QUICKUSDT','SHORT',50,25,'2571ff99a0f3480f92416699f624a584'),
('2025-09-30T06:30:00+00:00','TWTUSDT','LONG',50,75,'2571ff99a0f3480f92416699f624a584'),
('2025-09-30T06:30:00+00:00','HIFIUSDT','SHORT',25,20,'2571ff99a0f3480f92416699f624a584'),
('2025-10-01T06:30:00+00:00','HIFIUSDT','SHORT',20,10,'2571ff99a0f3480f92416699f624a584'),
('2025-12-12T06:30:00+00:00','NKNUSDT','SHORT',25,10,'47071723d8fe4302ab15dafbd9a2edc8'),
('2025-12-12T06:30:00+00:00','ALPINEUSDT','SHORT',50,10,'47071723d8fe4302ab15dafbd9a2edc8'),
('2025-12-12T06:30:00+00:00','ASRUSDT','SHORT',50,10,'47071723d8fe4302ab15dafbd9a2edc8'),
('2025-12-12T06:30:00+00:00','BELUSDT','SHORT',50,10,'47071723d8fe4302ab15dafbd9a2edc8'),
('2025-12-12T06:30:00+00:00','BRUSDT','SHORT',50,25,'47071723d8fe4302ab15dafbd9a2edc8'),
('2025-12-12T06:30:00+00:00','1000000BOBUSDT','SHORT',40,10,'47071723d8fe4302ab15dafbd9a2edc8'),
('2025-12-12T06:30:00+00:00','DEGOUSDT','SHORT',40,10,'47071723d8fe4302ab15dafbd9a2edc8'),
('2025-12-12T06:30:00+00:00','BLUAIUSDT','SHORT',50,10,'47071723d8fe4302ab15dafbd9a2edc8'),
('2025-12-12T06:30:00+00:00','COMMONUSDT','SHORT',50,10,'47071723d8fe4302ab15dafbd9a2edc8'),
('2025-12-12T06:30:00+00:00','DUSDT','SHORT',50,10,'47071723d8fe4302ab15dafbd9a2edc8'),
('2025-12-12T06:30:00+00:00','HOOKUSDT','SHORT',50,10,'47071723d8fe4302ab15dafbd9a2edc8'),
('2025-12-12T06:30:00+00:00','PIPPINUSDT','SHORT',50,10,'47071723d8fe4302ab15dafbd9a2edc8'),
('2025-12-12T06:30:00+00:00','SYNUSDT','SHORT',50,10,'47071723d8fe4302ab15dafbd9a2edc8'),
('2025-12-12T06:30:00+00:00','TACUSDT','SHORT',50,10,'47071723d8fe4302ab15dafbd9a2edc8'),
('2025-12-12T06:30:00+00:00','XNYUSDT','SHORT',50,10,'47071723d8fe4302ab15dafbd9a2edc8'),
('2025-12-12T06:30:00+00:00','AVAAIUSDT','SHORT',50,25,'47071723d8fe4302ab15dafbd9a2edc8'),
('2025-12-12T06:30:00+00:00','BMTUSDT','SHORT',50,25,'47071723d8fe4302ab15dafbd9a2edc8'),
('2025-12-12T06:30:00+00:00','BROCCOLI714USDT','SHORT',50,25,'47071723d8fe4302ab15dafbd9a2edc8'),
('2025-12-12T06:30:00+00:00','DFUSDT','SHORT',50,25,'47071723d8fe4302ab15dafbd9a2edc8'),
('2025-12-12T06:30:00+00:00','GHSTUSDT','SHORT',50,25,'47071723d8fe4302ab15dafbd9a2edc8'),
('2025-12-12T06:30:00+00:00','GTCUSDT','SHORT',50,25,'47071723d8fe4302ab15dafbd9a2edc8'),
('2025-12-12T06:30:00+00:00','HMSTRUSDT','SHORT',50,25,'47071723d8fe4302ab15dafbd9a2edc8'),
('2025-12-12T06:30:00+00:00','NFPUSDT','SHORT',50,25,'47071723d8fe4302ab15dafbd9a2edc8'),
('2025-12-12T06:30:00+00:00','PUFFERUSDT','SHORT',50,25,'47071723d8fe4302ab15dafbd9a2edc8'),
('2025-12-12T06:30:00+00:00','RDNTUSDT','SHORT',50,25,'47071723d8fe4302ab15dafbd9a2edc8'),
('2025-12-12T06:30:00+00:00','SWARMSUSDT','SHORT',50,25,'47071723d8fe4302ab15dafbd9a2edc8'),
('2025-12-12T06:30:00+00:00','TLMUSDT','SHORT',50,25,'47071723d8fe4302ab15dafbd9a2edc8'),
('2025-12-12T06:30:00+00:00','TSTUSDT','SHORT',50,25,'47071723d8fe4302ab15dafbd9a2edc8'),
('2025-12-12T06:30:00+00:00','TUTUSDT','SHORT',50,25,'47071723d8fe4302ab15dafbd9a2edc8'),
('2025-12-12T06:30:00+00:00','VICUSDT','SHORT',50,25,'47071723d8fe4302ab15dafbd9a2edc8'),
('2025-12-12T06:30:00+00:00','EDENUSDT','SHORT',75,50,'47071723d8fe4302ab15dafbd9a2edc8'),
('2025-12-12T06:30:00+00:00','ENSOUSDT','SHORT',75,50,'47071723d8fe4302ab15dafbd9a2edc8'),
]
def dt(s):return datetime.datetime.fromisoformat(s).astimezone(UTC)
def mon(x):return x.strftime('%Y-%m')
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
  req=urllib.request.Request(u,method='HEAD',headers={'User-Agent':UA});urllib.request.urlopen(req,timeout=15).close();return True
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
 hour=(rec['event_ms']//3600000)*3600000;data=rows(sym,prevmon(e))+rows(sym,em)
 prior=[x for x in data if hour-24*3600000<=x[0]<hour];qv=sum(x[3] for x in prior);rec['quote_volume_24h']=qv
 if len(prior)<24:rec.update(eligible=False,reason='missing_24h');res.append(rec);continue
 if qv<5_000_000:rec.update(eligible=False,reason='qv_lt_5m');res.append(rec);continue
 rec.update(eligible=True,reason='ok');res.append(rec)
print('TOTAL',len(res),'ELIGIBLE',sum(x['eligible'] for x in res))
from collections import Counter
print('REASONS',Counter(x['reason'] for x in res))
for x in res:
 print(('OK' if x['eligible'] else 'NO'),x['event_utc'],x['symbol'],x['side'],x['before'],'->',x['after'],x['reason'],round(x.get('quote_volume_24h',0)))
open('/tmp/leverage_tier_2025_eligible.json','w').write(json.dumps(res,indent=2))
