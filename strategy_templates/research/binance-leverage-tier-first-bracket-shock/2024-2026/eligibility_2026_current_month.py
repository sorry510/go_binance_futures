import csv,datetime,io,json,urllib.request,zipfile
from functools import lru_cache
UA='Mozilla/5.0';UTC=datetime.timezone.utc
MBASE='https://data.binance.vision/data/futures/um/monthly/klines'
DBASE='https://data.binance.vision/data/futures/um/daily/klines'
rows=json.load(open('/tmp/leverage_tier_2026_eligible.json'))
cand=[x for x in rows if x['event_utc'].startswith('2026-09-25') and x['reason']=='no_event_month']
@lru_cache(None)
def monthly_exists(sym,m):
 u=f'{MBASE}/{sym}/1h/{sym}-1h-{m}.zip'
 try:urllib.request.urlopen(urllib.request.Request(u,method='HEAD',headers={'User-Agent':UA}),timeout=12).close();return True
 except:return False
@lru_cache(None)
def monthly_rows(sym,m):
 u=f'{MBASE}/{sym}/1h/{sym}-1h-{m}.zip'
 try:b=urllib.request.urlopen(urllib.request.Request(u,headers={'User-Agent':UA}),timeout=20).read()
 except:return []
 z=zipfile.ZipFile(io.BytesIO(b));fn=z.namelist()[0];out=[]
 for r in csv.reader(io.TextIOWrapper(z.open(fn))):
  if not r or not r[0].isdigit():continue
  t=int(r[0]);t=t//1000 if t>10**15 else t
  out.append((t,float(r[7])))
 return out
@lru_cache(None)
def daily_rows(sym,d):
 u=f'{DBASE}/{sym}/1h/{sym}-1h-{d}.zip'
 try:b=urllib.request.urlopen(urllib.request.Request(u,headers={'User-Agent':UA}),timeout=20).read()
 except:return []
 z=zipfile.ZipFile(io.BytesIO(b));fn=z.namelist()[0];out=[]
 for r in csv.reader(io.TextIOWrapper(z.open(fn))):
  if not r or not r[0].isdigit():continue
  t=int(r[0]);t=t//1000 if t>10**15 else t
  out.append((t,float(r[7])))
 return out
out=[]
for x in cand:
 sym=x['symbol'];cut=datetime.datetime(2024,9,25,6,30,tzinfo=UTC);cutms=int(cut.timestamp()*1000)
 y=dict(x)
 if not monthly_exists(sym,'2024-09'):
  y.update(eligible=False,reason='history_lt_2y');out.append(y);continue
 hist=monthly_rows(sym,'2024-09')
 if not hist or hist[0][0]>cutms:
  y.update(eligible=False,reason='history_lt_2y');out.append(y);continue
 data=daily_rows(sym,'2026-09-24')+daily_rows(sym,'2026-09-25')
 event=x['event_ms'];hour=(event//3600000)*3600000
 prior=[r for r in data if hour-24*3600000<=r[0]<hour]
 qv=sum(r[1] for r in prior);y['quote_volume_24h']=qv
 if len(prior)<24:y.update(eligible=False,reason='missing_24h');out.append(y);continue
 if qv<5_000_000:y.update(eligible=False,reason='qv_lt_5m');out.append(y);continue
 y.update(eligible=True,reason='ok');out.append(y)
print('SEP TOTAL',len(out),'ELIGIBLE',sum(x['eligible'] for x in out))
for x in out:print(('OK' if x['eligible'] else 'NO'),x['symbol'],x['reason'],round(x.get('quote_volume_24h',0)))
open('/tmp/leverage_tier_2026_sep_eligible.json','w').write(json.dumps(out,indent=2))
# merge corrected Sep records
idx={x['symbol']:x for x in out}
merged=[]
for x in rows:
 if x['event_utc'].startswith('2026-09-25') and x['symbol'] in idx:merged.append(idx[x['symbol']])
 else:merged.append(x)
open('/tmp/leverage_tier_2026_eligible_merged.json','w').write(json.dumps(merged,indent=2))
