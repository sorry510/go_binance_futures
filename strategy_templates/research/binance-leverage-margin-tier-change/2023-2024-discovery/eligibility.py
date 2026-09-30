import json,urllib.request,urllib.error,time,datetime,zipfile,io,csv
from pathlib import Path
UTC=datetime.timezone.utc;UA='Mozilla/5.0'
EV=[x for x in json.load(open('/tmp/binance_leverage_events_parsed.json')) if x['classification'] in ('loosening','tightening')]
INFO=json.load(open('/tmp/fapi_exchangeInfo.json'))
IM={x['symbol']:x for x in INFO['symbols']}
VISION='https://data.binance.vision/data/futures/um/monthly/klines'

def rest_klines(sym,start,end):
    u=f'https://fapi.binance.com/fapi/v1/klines?symbol={sym}&interval=1h&startTime={start}&endTime={end}&limit=100'
    for k in range(4):
        try:
            req=urllib.request.Request(u,headers={'User-Agent':UA})
            return json.loads(urllib.request.urlopen(req,timeout=15).read())
        except urllib.error.HTTPError as e:
            if e.code in (418,429,500,502,503,504):time.sleep(1.2*(k+1));continue
            return []
        except Exception:time.sleep(.7*(k+1))
    return []

def vision_month(sym,month):
    u=f'{VISION}/{sym}/1h/{sym}-1h-{month}.zip'
    try:
        b=urllib.request.urlopen(urllib.request.Request(u,headers={'User-Agent':UA}),timeout=20).read()
    except urllib.error.HTTPError as e:
        if e.code==404:return []
        raise
    z=zipfile.ZipFile(io.BytesIO(b));fn=z.namelist()[0];out=[]
    for r in csv.reader(io.TextIOWrapper(z.open(fn))):
        if not r or not r[0].isdigit():continue
        t=int(r[0]);t=t//1000 if t>10**15 else t
        out.append((t,float(r[7]) if len(r)>7 else 0.0))
    return out
def month(ms):return datetime.datetime.fromtimestamp(ms/1000,UTC).strftime('%Y-%m')
def prevmonth(ms):
    d=datetime.datetime.fromtimestamp(ms/1000,UTC);y,m=d.year,d.month
    if m==1:y-=1;m=12
    else:m-=1
    return f'{y:04d}-{m:02d}'
def sub2(ms):
    d=datetime.datetime.fromtimestamp(ms/1000,UTC)
    try:d=d.replace(year=d.year-2)
    except ValueError:d=d.replace(year=d.year-2,day=28)
    return int(d.timestamp()*1000)
out=[]
for i,e in enumerate(EV,1):
    x=dict(e);sym=x['symbol'];evt=x['effective_time'];info=IM.get(sym)
    age_ok=False;age_days=None
    if info:
        onboard=int(info.get('onboardDate') or 0)
        if onboard>0:
            age_days=(evt-onboard)/86400000
            age_ok=age_days>=730
    else:
        cutoff=sub2(evt);cm=month(cutoff)
        hist=vision_month(sym,cm)
        if hist and hist[0][0]<=cutoff:
            age_ok=True;age_days=(evt-hist[0][0])/86400000
    if not age_ok:
        x.update(eligible=False,reason='history_lt_2y_or_missing',age_days=age_days);out.append(x);continue
    start=(evt//3600000)*3600000-24*3600000
    end=(evt//3600000)*3600000-1
    if info:
        rows=rest_klines(sym,start,end)
        prior=[r for r in rows if start<=int(r[0])<=end]
        qv=sum(float(r[7]) for r in prior)
    else:
        prior=[]
        for m in {prevmonth(evt),month(evt)}:
            prior.extend(vision_month(sym,m))
        prior=[r for r in prior if start<=r[0]<=end]
        qv=sum(r[1] for r in prior)
    x['age_days']=age_days;x['bars_24h']=len(prior);x['quote_volume_24h']=qv
    if len(prior)<24:
        x.update(eligible=False,reason='missing_24h_bars');out.append(x);continue
    if qv<5_000_000:
        x.update(eligible=False,reason='quote_volume_lt_5m');out.append(x);continue
    x.update(eligible=True,reason='ok');out.append(x)
    if i%20==0:print('PROGRESS',i,'/',len(EV),flush=True)
    time.sleep(.04)

Path('/tmp/binance_leverage_tier_eligibility.json').write_text(json.dumps(out,ensure_ascii=False,indent=2))
good=[x for x in out if x.get('eligible')]
from collections import Counter
print('DIRECTIONAL',len(EV),'ELIGIBLE',len(good),'UNIQUE_SYMBOLS',len(set(x['symbol'] for x in good)))
print('CLASS',Counter(x['classification'] for x in good))
print('REASONS',Counter(x['reason'] for x in out))
for x in good:
    dt=datetime.datetime.fromtimestamp(x['effective_time']/1000,UTC)
    print('OK',dt.isoformat(),x['symbol'],x['classification'],x['old_first_max_leverage'],'->',x['new_first_max_leverage'],'qv',round(x['quote_volume_24h']),flush=True)
