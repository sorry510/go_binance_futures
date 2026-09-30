import csv,datetime,io,json,urllib.request,zipfile
from functools import lru_cache
UTC=datetime.timezone.utc;UA='Mozilla/5.0';BASE='https://data.binance.vision/data/futures/um/monthly/klines'
E=[]
def add(ts,syms,side,b,a,src):
 for s in syms:E.append((ts,s,side,b,a,src))
add('2026-03-07T06:30:00+00:00',['NOMUSDT','SOPHUSDT','GOATUSDT','AIXBTUSDT','MAGICUSDT','MIRAUSDT','EULUSDT'],'SHORT',75,50,'a46249effc4645c89db2c18a198119d4')
add('2026-03-07T06:30:00+00:00',['TRUUSDT','4USDT','NTRNUSDT','LUMIAUSDT','GRIFFAINUSDT','HAEDALUSDT','PHBUSDT','MBOXUSDT','SYSUSDT','SHELLUSDT'],'SHORT',50,10,'a46249effc4645c89db2c18a198119d4')
add('2026-03-07T06:30:00+00:00',['MITOUSDT','SCRUSDT','HEMIUSDT','EPICUSDT','TOWNSUSDT','1000CATUSDT'],'SHORT',50,10,'a46249effc4645c89db2c18a198119d4')
add('2026-03-07T06:30:00+00:00',['SENTUSDT','LITUSDT','JSTUSDT'],'LONG',50,75,'a46249effc4645c89db2c18a198119d4')
add('2026-03-13T06:30:00+00:00',['PEOPLEUSDT','HUMAUSDT','LAUSDT'],'SHORT',75,50,'361baed0b23744a8b322bc7023cba3a9')
add('2026-03-13T06:30:00+00:00',['EDENUSDT','SYSUSDT','CLOUSDT','AIUSDT','B3USDT','DODOXUSDT','ATAUSDT','FORTHUSDT'],'SHORT',50,25,'361baed0b23744a8b322bc7023cba3a9')
add('2026-04-24T06:30:00+00:00',['ARKMUSDT','BIGTIMEUSDT','BOMEUSDT','AUCTIONUSDT','NEIROUSDT','MEMEUSDT','IOUSDT','LAYERUSDT','GMTUSDT','MERLUSDT','PNUTUSDT','SOMIUSDT','HOLOUSDT','NOTUSDT'],'SHORT',75,50,'26d93ead76134f94901e75edd3e008d4')
add('2026-04-24T06:30:00+00:00',['NOMUSDT','YBUSDT','MLNUSDT','ACEUSDT','HIGHUSDT','COOKIEUSDT'],'SHORT',50,10,'26d93ead76134f94901e75edd3e008d4')
add('2026-05-22T06:30:00+00:00',['HUSDT'],'SHORT',20,10,'7a9deab7bcad40a69dd61734a577387e')
add('2026-05-29T06:30:00+00:00',['ESPORTSUSDT'],'SHORT',50,10,'6a7aff2596ed4593af4fb924ad687286')
add('2026-05-29T06:30:00+00:00',['SYRUPUSDT','VVVUSDT','GOATUSDT'],'SHORT',50,25,'6a7aff2596ed4593af4fb924ad687286')
add('2026-05-29T06:30:00+00:00',['FFUSDT'],'SHORT',75,50,'6a7aff2596ed4593af4fb924ad687286')
add('2026-05-29T06:30:00+00:00',['BEATUSDT','NFPUSDT'],'SHORT',25,10,'6a7aff2596ed4593af4fb924ad687286')
add('2026-05-29T06:30:00+00:00',['LIGHTUSDT','TAKEUSDT'],'SHORT',20,10,'6a7aff2596ed4593af4fb924ad687286')
add('2026-06-12T06:30:00+00:00',['1000000MOGUSDT'],'SHORT',75,50,'4d11138559ff4598b60965c479517934')
add('2026-06-19T06:30:00+00:00',['ORDIUSDT'],'SHORT',75,50,'d1954a75b8514a2d91b9135a13348df5')
add('2026-07-10T06:30:00+00:00',['MANTAUSDT'],'SHORT',50,25,'f7dbb4547b4f443d975eb7d619ad2969')
add('2026-07-24T06:30:00+00:00',['BANKUSDT','GUNUSDT','LAUSDT','OPNUSDT','KATUSDT'],'SHORT',50,25,'97b4f93f9c684da2a5701ab09530d3df')
add('2026-07-24T06:30:00+00:00',['STARUSDT'],'SHORT',10,5,'97b4f93f9c684da2a5701ab09530d3df')
add('2026-07-24T06:30:00+00:00',['ZILUSDT'],'SHORT',75,25,'97b4f93f9c684da2a5701ab09530d3df')
add('2026-08-07T06:30:00+00:00',['0GUSDT','BERAUSDT','WUSDT','SUPERUSDT','TURBOUSDT','1MBABYDOGEUSDT'],'SHORT',75,50,'4bf81e4829b64eacae3caa802ec6fc59')
add('2026-08-07T06:30:00+00:00',['SANTOSUSDT','DYMUSDT','SOPHUSDT','GUNUSDT','KERNELUSDT','DOLOUSDT'],'SHORT',25,10,'4bf81e4829b64eacae3caa802ec6fc59')
add('2026-08-11T09:00:00+00:00',['SNDKUSDT'],'LONG',50,75,'3f18062274ea49199475b4ad9ff35738')
add('2026-09-25T06:30:00+00:00',['COOKIEUSDT','TLMUSDT','EPICUSDT'],'SHORT',20,10,'de15b9501a8f43e1aece2cbe7f3809de')
add('2026-09-25T06:30:00+00:00',['RAVEUSDT','DODOXUSDT','MOVEUSDT','SOPHUSDT','AVAUSDT'],'SHORT',25,10,'de15b9501a8f43e1aece2cbe7f3809de')
add('2026-09-25T06:30:00+00:00',['AWEUSDT','ARKUSDT'],'SHORT',40,10,'de15b9501a8f43e1aece2cbe7f3809de')
add('2026-09-25T06:30:00+00:00',['B2USDT','UAIUSDT','VELODROMEUSDT','BLURUSDT','MOVRUSDT','LSKUSDT'],'SHORT',50,25,'de15b9501a8f43e1aece2cbe7f3809de')
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
  req=urllib.request.Request(u,method='HEAD',headers={'User-Agent':UA});urllib.request.urlopen(req,timeout=12).close();return True
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
from collections import Counter
print('TOTAL',len(res),'ELIGIBLE',sum(x['eligible'] for x in res),'REASONS',Counter(x['reason'] for x in res))
for x in res:
 if x['eligible']:print('OK',x['event_utc'],x['symbol'],x['side'],x['before'],'->',x['after'],round(x.get('quote_volume_24h',0)))
open('/tmp/leverage_tier_2026_eligible.json','w').write(json.dumps(res,indent=2))
