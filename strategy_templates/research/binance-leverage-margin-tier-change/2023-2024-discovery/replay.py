import json,urllib.request,urllib.error,time,datetime,zipfile,io,csv
from pathlib import Path
UTC=datetime.timezone.utc;UA='Mozilla/5.0'
EV=[x for x in json.load(open('/tmp/binance_leverage_tier_eligibility.json')) if x.get('eligible')]
INFO=json.load(open('/tmp/fapi_exchangeInfo.json'));IM={x['symbol']:x for x in INFO['symbols']}
VISION='https://data.binance.vision/data/futures/um/monthly/klines'

def rest(sym,start,end):
    u=f'https://fapi.binance.com/fapi/v1/klines?symbol={sym}&interval=1h&startTime={start}&endTime={end}&limit=50'
    for k in range(4):
        try:
            return json.loads(urllib.request.urlopen(urllib.request.Request(u,headers={'User-Agent':UA}),timeout=15).read())
        except urllib.error.HTTPError as e:
            if e.code in (418,429,500,502,503,504):time.sleep(1*(k+1));continue
            return []
        except Exception:time.sleep(.6*(k+1))
    return []

def vision(sym,month):
    u=f'{VISION}/{sym}/1h/{sym}-1h-{month}.zip'
    try:b=urllib.request.urlopen(urllib.request.Request(u,headers={'User-Agent':UA}),timeout=20).read()
    except Exception:return []
    z=zipfile.ZipFile(io.BytesIO(b));fn=z.namelist()[0];out=[]
    for r in csv.reader(io.TextIOWrapper(z.open(fn))):
        if not r or not r[0].isdigit():continue
        t=int(r[0]);t=t//1000 if t>10**15 else t
        out.append((t,float(r[1]),float(r[4])))
    return out
def month(ms):return datetime.datetime.fromtimestamp(ms/1000,UTC).strftime('%Y-%m')
def nextmonth(ms):
    d=datetime.datetime.fromtimestamp(ms/1000,UTC)
    y,m=d.year,d.month
    if m==12:y+=1;m=1
    else:m+=1
    return f'{y:04d}-{m:02d}'
import math,collections
res=[]
for i,e in enumerate(EV,1):
    sym=e['symbol'];evt=e['effective_time']
    entry_hour=((evt//3600000)+1)*3600000
    end=entry_hour+13*3600000
    bars=[]
    if sym in IM:
        rr=rest(sym,entry_hour,end)
        bars=[(int(r[0]),float(r[1]),float(r[4])) for r in rr if entry_hour<=int(r[0])<=end]
    if len(bars)<13:
        vb=vision(sym,month(evt))+vision(sym,nextmonth(evt))
        bars=[r for r in vb if entry_hour<=r[0]<=end]
    bars.sort()
    if len(bars)<13 or bars[0][0]!=entry_hour:
        x=dict(e);x.update(error='missing_forward_bars',bars=len(bars));res.append(x);continue
    entry=bars[0][1]
    sign=1.0 if e['classification']=='loosening' else -1.0
    r1=sign*math.log(bars[0][2]/entry)
    r4=sign*math.log(bars[3][2]/entry)
    r12=sign*math.log(bars[11][2]/entry)
    x=dict(e);x.update(entry_time=entry_hour,entry=entry,r1=r1,r4=r4,r12=r12)
    res.append(x)
    if i%20==0:print('PROGRESS',i,'/',len(EV),flush=True)
    time.sleep(.03)

Path('/tmp/binance_leverage_tier_diagnostic.json').write_text(json.dumps(res,ensure_ascii=False,indent=2))
good=[x for x in res if not x.get('error')]
def stats(arr,key):
    vals=[x[key] for x in arr]
    return {'n':len(vals),'mean':sum(vals)/len(vals) if vals else None,'win':sum(v>0 for v in vals)/len(vals) if vals else None}
print('GOOD',len(good),'ERRORS',collections.Counter(x.get('error') for x in res if x.get('error')))
for key in ['r1','r4','r12']:print(key,stats(good,key))
print('BY_CLASS')
for cls in ['loosening','tightening']:
    q=[x for x in good if x['classification']==cls]
    print(cls,'n',len(q),'r1',stats(q,'r1'),'r4',stats(q,'r4'),'r12',stats(q,'r12'))
print('BY_YEAR')
for y in [2023,2024]:
    q=[x for x in good if datetime.datetime.fromtimestamp(x['effective_time']/1000,UTC).year==y]
    print(y,'n',len(q),'r1',stats(q,'r1'),'r4',stats(q,'r4'),'r12',stats(q,'r12'))
by={}
for x in good:
    by.setdefault(x['symbol'],[]).append(x['r12'])
pos=sum(sum(v)/len(v)>0 for v in by.values())
print('SYMBOLS',len(by),'POSITIVE_12H',pos)
for s,v in sorted(by.items()):
    print('SYM',s,'n',len(v),'mean12',sum(v)/len(v))
