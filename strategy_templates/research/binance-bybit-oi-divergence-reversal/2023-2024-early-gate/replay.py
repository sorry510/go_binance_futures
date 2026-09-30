import csv,io,zipfile,urllib.request,urllib.parse,urllib.error,time,datetime,json,math
from concurrent.futures import ThreadPoolExecutor,as_completed
from pathlib import Path
UTC=datetime.timezone.utc
SYMS=['BTCUSDT','ETHUSDT','BNBUSDT','XRPUSDT']
START=datetime.date(2022,12,1);END=datetime.date(2024,12,31)
UA='Mozilla/5.0';CACHE=Path('/tmp/binance_bybit_oi_cache');CACHE.mkdir(exist_ok=True)
def dates():
    d=START
    while d<=END:
        yield d
        d+=datetime.timedelta(days=1)
def fetch_bin(sym,d):
    ds=d.isoformat();p=CACHE/f'bin-{sym}-{ds}.zip'
    if p.exists():return p
    u=f'https://data.binance.vision/data/futures/um/daily/metrics/{sym}/{sym}-metrics-{ds}.zip'
    for k in range(4):
        try:b=urllib.request.urlopen(urllib.request.Request(u,headers={'User-Agent':UA}),timeout=20).read();p.write_bytes(b);return p
        except urllib.error.HTTPError as e:
            if e.code==404:p.write_bytes(b'');return p
            time.sleep(.3*(k+1))
        except Exception:time.sleep(.3*(k+1))
    p.write_bytes(b'');return p
def parse_bin(sym):
    out={}
    for d in dates():
        p=CACHE/f'bin-{sym}-{d.isoformat()}.zip'
        if not p.exists() or p.stat().st_size==0:continue
        z=zipfile.ZipFile(p);fn=z.namelist()[0]
        rd=csv.DictReader(io.TextIOWrapper(z.open(fn)))
        for r in rd:
            try:
                dt=datetime.datetime.strptime(r['create_time'],'%Y-%m-%d %H:%M:%S').replace(tzinfo=UTC)
                if dt.minute==0 and dt.second==0:
                    out[int(dt.timestamp()*1000)]=float(r['sum_open_interest'])
            except:pass
    return out
def fetch_bybit(sym):
    p=CACHE/f'bybit-{sym}.json'
    if p.exists():return json.loads(p.read_text())
    start=int(datetime.datetime.combine(START,datetime.time(),tzinfo=UTC).timestamp()*1000)
    end=int(datetime.datetime.combine(END+datetime.timedelta(days=1),datetime.time(),tzinfo=UTC).timestamp()*1000)-1
    vals={}
    cursor=end
    calls=0
    while cursor>=start:
        q=urllib.parse.urlencode({'category':'linear','symbol':sym,'intervalTime':'1h','endTime':cursor,'limit':200})
        u='https://api.bybit.com/v5/market/open-interest?'+q
        try:o=json.loads(urllib.request.urlopen(urllib.request.Request(u,headers={'User-Agent':UA}),timeout=20).read())
        except Exception:
            time.sleep(1);continue
        arr=((o.get('result') or {}).get('list') or [])
        if not arr:break
        oldest=cursor
        for x in arr:
            t=int(x['timestamp'])
            if t>=start:vals[str(t)]=float(x['openInterest'])
            oldest=min(oldest,t)
        calls+=1
        if oldest>=cursor:break
        cursor=oldest-1
        if oldest<start:break
        if calls%20==0:print('BYBIT',sym,calls,len(vals),flush=True)
        time.sleep(.05)
    p.write_text(json.dumps(vals));return vals
jobs=[(s,d) for s in SYMS for d in dates()]
print('BIN_JOBS',len(jobs),flush=True)
with ThreadPoolExecutor(max_workers=20) as ex:
    fs=[ex.submit(fetch_bin,*j) for j in jobs]
    for i,f in enumerate(as_completed(fs),1):
        f.result()
        if i%500==0:print('BIN_DOWNLOAD',i,'/',len(jobs),flush=True)
with ThreadPoolExecutor(max_workers=4) as ex:
    by=dict(zip(SYMS,ex.map(fetch_bybit,SYMS)))
def msd(x):
    m=sum(x)/len(x);return m,(sum((v-m)**2 for v in x)/len(x))**.5
# read Binance 1h target-price data exported from DB
prices={}
for r in csv.DictReader(open('/tmp/usdt_core4_1h.csv')):
    prices.setdefault(r['symbol'],{})[int(r['t'])]=(float(r['open']),float(r['close']))
events=[]
for s in SYMS:
    B=parse_bin(s);Y={int(k):v for k,v in by[s].items()}
    ts=sorted(set(B)&set(Y)&set(prices.get(s,{})))
    seq=[]
    for t in ts:
        if B[t]>0 and Y[t]>0:seq.append((t,math.log(B[t]/Y[t])))
    armed=True;n=0
    for i in range(720,len(seq)):
        t,zv=seq[i]
        if datetime.datetime.fromtimestamp(t/1000,UTC).year not in (2023,2024):continue
        # continuity prevents gaps from becoming synthetic windows
        if t-seq[i-720][0]>721*3600000:continue
        hist=[x[1] for x in seq[i-720:i]];m,sd=msd(hist)
        if sd<=0:continue
        z=(zv-m)/sd
        if abs(z)<1:armed=True
        if not armed or abs(z)<3:continue
        d=-1.0 if z>0 else 1.0
        nt=t+3600000
        if nt not in prices[s] or nt+11*3600000 not in prices[s]:continue
        entry=prices[s][nt][0]
        r1=d*math.log(prices[s][nt][1]/entry)
        r4=d*math.log(prices[s][nt+3*3600000][1]/entry) if nt+3*3600000 in prices[s] else None
        r12=d*math.log(prices[s][nt+11*3600000][1]/entry)
        if r4 is None:continue
        events.append((s,t,z,r1,r4,r12));armed=False;n+=1
    print('SYMBOL',s,'aligned',len(seq),'events',n,flush=True)
def summ(xs):
    sy=sorted(set(x[0] for x in xs));byx={};pos=0
    for s in sy:
        q=[x for x in xs if x[0]==s];m=sum(x[5] for x in q)/len(q);byx[s]={'n':len(q),'mean12':m};pos+=m>0
    return {'n':len(xs),'mean1':sum(x[3] for x in xs)/len(xs) if xs else None,'mean4':sum(x[4] for x in xs)/len(xs) if xs else None,'mean12':sum(x[5] for x in xs)/len(xs) if xs else None,'win12':sum(x[5]>0 for x in xs)/len(xs) if xs else None,'positive_symbols12':pos,'symbols':len(sy),'by_symbol':byx}
res={'gate_2023_2024':summ(events)}
print(json.dumps(res,indent=2),flush=True)
Path('/tmp/binance_bybit_oi_divergence_summary.json').write_text(json.dumps(res,indent=2))
