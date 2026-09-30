import csv,io,zipfile,urllib.request,urllib.parse,urllib.error,xml.etree.ElementTree as ET,datetime,math,json,time
from pathlib import Path
from concurrent.futures import ThreadPoolExecutor,as_completed
UTC=datetime.timezone.utc
BASES=['BTC','ETH','BNB','XRP','ADA','LINK','BCH','DOT','LTC']
S3='https://s3-ap-northeast-1.amazonaws.com/data.binance.vision'
DV='https://data.binance.vision/'
UA='Mozilla/5.0'
CACHE=Path('/tmp/liq_snapshot_cache');CACHE.mkdir(exist_ok=True)
ns={'s':'http://s3.amazonaws.com/doc/2006-03-01/'}
def list_zip_keys(sym):
    pref=f'data/futures/cm/daily/liquidationSnapshot/{sym}/'
    u=S3+'?'+urllib.parse.urlencode({'prefix':pref,'max-keys':'1000'})
    root=ET.fromstring(urllib.request.urlopen(u,timeout=30).read())
    return [x.text for x in root.findall('.//s:Contents/s:Key',ns) if x.text.endswith('.zip')]
def fetch_key(key):
    p=CACHE/key.replace('/','__')
    if p.exists():return p
    last=None
    for k in range(4):
        try:
            b=urllib.request.urlopen(urllib.request.Request(DV+key,headers={'User-Agent':UA}),timeout=20).read();p.write_bytes(b);return p
        except Exception as e:last=e;time.sleep(.3*(k+1))
    raise last
keys=[]
bybase={}
for b in BASES:
    ks=list_zip_keys(b+'USD_PERP')
    ks=[k for k in ks if '2023-' in k or '2024-' in k]
    bybase[b]=ks;keys.extend(ks)
print('FILES',sum(len(v) for v in bybase.values()),{b:len(v) for b,v in bybase.items()},flush=True)
with ThreadPoolExecutor(max_workers=24) as ex:
    fs=[ex.submit(fetch_key,k) for k in keys]
    for i,f in enumerate(as_completed(fs),1):
        f.result()
        if i%500==0:print('DOWNLOAD',i,'/',len(fs),flush=True)
def read_events(path):
    z=zipfile.ZipFile(path);fn=z.namelist()[0];rows=csv.DictReader(io.TextIOWrapper(z.open(fn)))
    seen=set();out=[]
    for r in rows:
        key=(r['time'],r['side'],r['original_quantity'],r['price'],r['average_price'],r['accumulated_fill_quantity'])
        if key in seen:continue
        seen.add(key)
        try:out.append((int(r['time']),r['side'],float(r['accumulated_fill_quantity'])))
        except:pass
    return out
def month_range():
    out=[];y,m=2023,6
    while (y,m)<=(2024,10):
        out.append(f'{y:04d}-{m:02d}');m+=1
        if m==13:y+=1;m=1
    return out
def um_url(sym,mo):return f'https://data.binance.vision/data/futures/um/monthly/klines/{sym}/1h/{sym}-1h-{mo}.zip'
def fetch_um(sym,mo):
    p=CACHE/f'um__{sym}__{mo}.zip'
    if p.exists():return p
    try:
        b=urllib.request.urlopen(urllib.request.Request(um_url(sym,mo),headers={'User-Agent':UA}),timeout=25).read();p.write_bytes(b)
    except urllib.error.HTTPError as e:
        if e.code==404:p.write_bytes(b'')
        else:raise
    return p
umjobs=[(b+'USDT',mo) for b in BASES for mo in month_range()]
with ThreadPoolExecutor(max_workers=16) as ex:
    fs=[ex.submit(fetch_um,*j) for j in umjobs]
    for f in as_completed(fs):f.result()
def read_um(sym):
    out={}
    for mo in month_range():
        p=CACHE/f'um__{sym}__{mo}.zip'
        if not p.exists() or p.stat().st_size==0:continue
        z=zipfile.ZipFile(p);fn=z.namelist()[0]
        for r in csv.reader(io.TextIOWrapper(z.open(fn))):
            if not r or not r[0].isdigit():continue
            t=int(r[0]);t=t//1000 if t>10**15 else t
            out[t]=(float(r[1]),float(r[4]))
    return out
events=[]
for b in BASES:
    hourly={}
    for key in bybase[b]:
        p=CACHE/key.replace('/','__')
        for t,side,q in read_events(p):
            h=(t//3600000)*3600000
            x=hourly.setdefault(h,[0.0,0.0])
            if side=='BUY':x[0]+=q
            elif side=='SELL':x[1]+=q
    um=read_um(b+'USDT')
    if not hourly or not um:print('SKIP',b,len(hourly),len(um));continue
    start=max(min(hourly),int(datetime.datetime(2023,6,25,tzinfo=UTC).timestamp()*1000))
    end=min(max(hourly),int(datetime.datetime(2024,10,14,23,tzinfo=UTC).timestamp()*1000))
    hours=list(range(start,end+1,3600000))
    vals=[math.log1p(sum(hourly.get(t,[0,0]))) for t in hours]
    armed=True;n=0
    for i in range(720,len(hours)-13):
        hist=vals[i-720:i];m=sum(hist)/720;sd=(sum((x-m)**2 for x in hist)/720)**0.5
        if sd<=0:continue
        z=(vals[i]-m)/sd
        if z<1:armed=True
        if not armed or z<3:continue
        buy,sell=hourly.get(hours[i],[0,0])
        if buy==sell:continue
        d=1.0 if buy>sell else -1.0
        t1=hours[i]+3600000;t4=hours[i]+4*3600000;t12=hours[i]+12*3600000
        if t1 not in um or t4 not in um or t12 not in um:continue
        entry=um[t1][0]
        r1=d*math.log(um[t1][1]/entry);r4=d*math.log(um[t4][1]/entry);r12=d*math.log(um[t12][1]/entry)
        events.append((b,hours[i],z,buy,sell,r1,r4,r12));armed=False;n+=1
    print('SYMBOL',b,'raw_hours',len(hourly),'events',n,flush=True)
def summary(xs):
    if not xs:return {}
    sy=set(x[0] for x in xs)
    return {'n':len(xs),'mean1':sum(x[5] for x in xs)/len(xs),'mean4':sum(x[6] for x in xs)/len(xs),'mean12':sum(x[7] for x in xs)/len(xs),
            'win12':sum(x[7]>0 for x in xs)/len(xs),'positive_symbols12':sum((sum(x[7] for x in xs if x[0]==s)/len([x for x in xs if x[0]==s]))>0 for s in sy),'symbols':len(sy)}
cut=int(datetime.datetime(2024,1,1,tzinfo=UTC).timestamp()*1000)
res={'discovery_2023H2':summary([x for x in events if x[1]<cut]),
     'oos1_2024':summary([x for x in events if x[1]>=cut]),
     'all':summary(events)}
print(json.dumps(res,indent=2),flush=True)
Path('/tmp/coinm_liquidation_cascade_summary.json').write_text(json.dumps(res,indent=2))
