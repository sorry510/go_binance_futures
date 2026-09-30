import urllib.request,urllib.parse,urllib.error,xml.etree.ElementTree as ET,zipfile,io,csv,datetime,json,time
from concurrent.futures import ThreadPoolExecutor,as_completed
UTC=datetime.timezone.utc;UA='Mozilla/5.0'
BASES=['BTC','ETH','BNB','SOL','XRP','DOGE','SUI','LINK','ORDI','AVAX','1000SHIB','BCH','WIF','LTC','NEAR','ARB','NEO','FIL','BOME','TIA','MATIC','ENA','ETHFI','1000BONK','CRV']
S3='https://s3-ap-northeast-1.amazonaws.com/data.binance.vision'
VISION='https://data.binance.vision/data/futures/um/monthly/klines'
def first_usdc_launch(base):
    sym=base+'USDC';prefix=f'data/futures/um/monthly/klines/{sym}/1h/'
    u=S3+'?'+urllib.parse.urlencode({'prefix':prefix})
    try:
        root=ET.fromstring(urllib.request.urlopen(u,timeout=20).read())
    except Exception:return None
    ns={'s':'http://s3.amazonaws.com/doc/2006-03-01/'}
    keys=sorted(x.text for x in root.findall('.//s:Contents/s:Key',ns) if x.text.endswith('.zip'))
    if not keys:return None
    k=keys[0];url='https://data.binance.vision/'+k
    try:b=urllib.request.urlopen(urllib.request.Request(url,headers={'User-Agent':UA}),timeout=25).read()
    except Exception:return None
    z=zipfile.ZipFile(io.BytesIO(b));fn=z.namelist()[0]
    for r in csv.reader(io.TextIOWrapper(z.open(fn))):
        if r and r[0].isdigit():
            t=int(r[0]);t=t//1000 if t>10**15 else t
            return t
    return None
def month(ms):return datetime.datetime.fromtimestamp(ms/1000,UTC).strftime('%Y-%m')
def prevmonth(ms):
    d=datetime.datetime.fromtimestamp(ms/1000,UTC)
    return f'{d.year-1}-12' if d.month==1 else f'{d.year:04d}-{d.month-1:02d}'
def sub2(ms):
    d=datetime.datetime.fromtimestamp(ms/1000,UTC)
    try:d=d.replace(year=d.year-2)
    except ValueError:d=d.replace(year=d.year-2,day=28)
    return int(d.timestamp()*1000)
def rows1h(sym,mo):
    u=f'{VISION}/{sym}/1h/{sym}-1h-{mo}.zip'
    try:b=urllib.request.urlopen(urllib.request.Request(u,headers={'User-Agent':UA}),timeout=20).read()
    except urllib.error.HTTPError as e:
        if e.code==404:return []
        return []
    except Exception:return []
    z=zipfile.ZipFile(io.BytesIO(b));fn=z.namelist()[0];out=[]
    for r in csv.reader(io.TextIOWrapper(z.open(fn))):
        if not r or not r[0].isdigit():continue
        t=int(r[0]);t=t//1000 if t>10**15 else t
        out.append((t,float(r[7]) if len(r)>7 else 0.0))
    return out
print('TOKENS',len(BASES),flush=True)
launch={}
with ThreadPoolExecutor(max_workers=12) as ex:
    fs={ex.submit(first_usdc_launch,b):b for b in BASES}
    for f in as_completed(fs):
        b=fs[f];launch[b]=f.result()
        print('LAUNCH',b,datetime.datetime.fromtimestamp(launch[b]/1000,UTC).isoformat() if launch[b] else None,flush=True)
good=[];drop=[]
for b in BASES:
    evt=launch.get(b);usdt=b+'USDT'
    if not evt:
        drop.append((b,'no_usdc_history'));continue
    cutoff=sub2(evt);hist=rows1h(usdt,month(cutoff))
    if not hist or hist[0][0]>cutoff:
        drop.append((b,'history_lt_2y_or_missing'));continue
    hour=(evt//3600000)*3600000;start=hour-24*3600000;end=hour-1
    prior=rows1h(usdt,prevmonth(evt))+rows1h(usdt,month(evt))
    prior=[r for r in prior if start<=r[0]<=end];qv=sum(r[1] for r in prior)
    if len(prior)<24:
        drop.append((b,'missing_24h'));continue
    if qv<5_000_000:
        drop.append((b,'qv_lt_5m'));continue
    good.append({'base':b,'usdt_symbol':usdt,'usdc_symbol':b+'USDC','launch_ms':evt,'launch_utc':datetime.datetime.fromtimestamp(evt/1000,UTC).isoformat(),'quote_volume_24h':qv})
print('ELIGIBLE',len(good),'DROP',len(drop),flush=True)
for x in good:print('OK',x['launch_utc'],x['base'],'qv',round(x['quote_volume_24h']),flush=True)
from collections import Counter
print('DROP_REASONS',Counter(x[1] for x in drop),flush=True)
open('/tmp/usdc_perp_launch_eligibility.json','w').write(json.dumps({'eligible':good,'drops':drop},indent=2))
