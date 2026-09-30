import csv,json,urllib.request,urllib.parse,time,math,datetime
from pathlib import Path
UTC=datetime.timezone.utc
SYMS=['BTCUSDT','ETHUSDT','BNBUSDT','XRPUSDT','SOLUSDT','DOGEUSDT','LTCUSDT','AVAXUSDT','UNIUSDT','ZECUSDT']
B={s:{} for s in SYMS}
for r in csv.DictReader(open('/tmp/binance_bybit_volume_1h.csv')):
    s=r['symbol'];t=int(r['t'])
    B[s][t]=(float(r['open']),float(r['close']),float(r['quote_volume']))
CACHE=Path('/tmp/bybit_volume_cache');CACHE.mkdir(exist_ok=True)
def fetch_chunk(sym,start,end):
    key=CACHE/f'{sym}-{start}-{end}.json'
    if key.exists():return json.loads(key.read_text())
    q=urllib.parse.urlencode({'category':'linear','symbol':sym,'interval':'60','start':start,'end':end,'limit':1000})
    u='https://api.bybit.com/v5/market/kline?'+q
    last=None
    for k in range(5):
        try:
            req=urllib.request.Request(u,headers={'User-Agent':'Mozilla/5.0','Accept':'application/json'})
            o=json.loads(urllib.request.urlopen(req,timeout=25).read())
            if o.get('retCode')!=0:raise RuntimeError(str(o)[:500])
            li=((o.get('result') or {}).get('list') or [])
            key.write_text(json.dumps(li));return li
        except Exception as e:last=e;time.sleep(.5*(k+1))
    raise last
def bybit(sym):
    ts=sorted(B[sym])
    start=max(ts[0],int(datetime.datetime(2022,12,1,tzinfo=UTC).timestamp()*1000))
    end=ts[-1]
    out={}
    cur=start; step=999*3600000
    n=0
    while cur<=end:
        e=min(end,cur+step)
        li=fetch_chunk(sym,cur,e)
        for r in li:
            try:
                t=int(r[0]);out[t]=(float(r[1]),float(r[4]),float(r[6]))
            except:pass
        cur=e+3600000;n+=1
        time.sleep(.06)
    print('BYBIT',sym,'chunks',n,'rows',len(out),'range',min(out) if out else None,max(out) if out else None,flush=True)
    return out
def msd(v):
    m=sum(v)/len(v);return m,(sum((x-m)**2 for x in v)/len(v))**0.5
events=[]
for sym in SYMS:
    Y=bybit(sym); seq=[]
    for t in sorted(set(B[sym])&set(Y)):
        bo,bc,bq=B[sym][t];yo,yc,yq=Y[t]
        if bq<=0 or yq<=0:continue
        seq.append((t,bo,bc,bq,yo,yc,yq,math.log(yq/bq)))
    armed=True;n=0
    for i in range(720,len(seq)-12):
        if seq[i][0]-seq[i-720][0]>721*3600000:continue
        if seq[i+12][0]-seq[i][0]!=12*3600000:continue
        hist=[x[7] for x in seq[i-720:i]];m,sd=msd(hist)
        if sd<=0:continue
        z=(seq[i][7]-m)/sd
        if abs(z)<1:armed=True
        if not armed or abs(z)<3:continue
        if z>0:r0=math.log(seq[i][5]/seq[i][4])
        else:r0=math.log(seq[i][2]/seq[i][1])
        if r0==0:continue
        d=1.0 if r0>0 else -1.0
        entry=seq[i+1][1]
        if entry<=0:continue
        r1=d*math.log(seq[i+1][2]/entry)
        r4=d*math.log(seq[i+4][2]/entry)
        r12=d*math.log(seq[i+12][2]/entry)
        y=datetime.datetime.fromtimestamp(seq[i][0]/1000,UTC).year
        if y>=2023:events.append((sym,y,seq[i][0],z,r0,r1,r4,r12))
        armed=False;n+=1
    print('SIGNALS',sym,'aligned',len(seq),'events',n,flush=True)
def summ(xs):
    if not xs:return {}
    sy=sorted(set(x[0] for x in xs));by={};pos=0
    for s in sy:
        q=[x for x in xs if x[0]==s];m=sum(x[7] for x in q)/len(q);by[s]={'n':len(q),'mean12':m};pos+=m>0
    return {'n':len(xs),'mean1':sum(x[5] for x in xs)/len(xs),'mean4':sum(x[6] for x in xs)/len(xs),'mean12':sum(x[7] for x in xs)/len(xs),
            'win12':sum(x[7]>0 for x in xs)/len(xs),'positive_symbols12':pos,'symbols':len(sy),'by_symbol':by}
res={'discovery_2023_2024':summ([x for x in events if x[1] in (2023,2024)]),
     'oos1_2025':summ([x for x in events if x[1]==2025]),
     'oos2_2026':summ([x for x in events if x[1]==2026]),
     'all':summ(events)}
print(json.dumps(res,indent=2),flush=True)
Path('/tmp/binance_bybit_volume_lead_summary.json').write_text(json.dumps(res,indent=2))
