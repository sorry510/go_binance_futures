import csv,io,zipfile,urllib.request,urllib.parse,urllib.error,datetime,json,math,time,re,xml.etree.ElementTree as ET
from pathlib import Path
from functools import lru_cache
UTC=datetime.timezone.utc;UA='Mozilla/5.0'
MAP={'BTCUSDT':'Bitcoin','ETHUSDT':'Ethereum','BNBUSDT':'BSC','XRPUSDT':'Ripple','SOLUSDT':'Solana','AVAXUSDT':'Avalanche','ADAUSDT':'Cardano','NEARUSDT':'Near','SUIUSDT':'Sui'}
DV='https://data.binance.vision/data/futures/um/monthly/klines'
S3='https://s3-ap-northeast-1.amazonaws.com/data.binance.vision'
CACHE=Path('/tmp/defillama_tvl_residual_cache');CACHE.mkdir(exist_ok=True)
def log(x):return math.log(x) if x>0 else None
def sub_or_add_2y(d,plus=True):
    y=d.year+(2 if plus else -2)
    try:return d.replace(year=y)
    except ValueError:return d.replace(year=y,day=28)
def first_kline(sym):
    pref=f'data/futures/um/monthly/klines/{sym}/1d/'
    u=S3+'?'+urllib.parse.urlencode({'prefix':pref,'max-keys':'1000'})
    root=ET.fromstring(urllib.request.urlopen(u,timeout=30).read())
    ns={'s':'http://s3.amazonaws.com/doc/2006-03-01/'}
    keys=[x.text for x in root.findall('.//s:Contents/s:Key',ns) if x.text.endswith('.zip')]
    if not keys:return None
    key=min(keys)
    b=urllib.request.urlopen(urllib.request.Request('https://data.binance.vision/'+key,headers={'User-Agent':UA}),timeout=20).read()
    z=zipfile.ZipFile(io.BytesIO(b));fn=z.namelist()[0]
    for r in csv.reader(io.TextIOWrapper(z.open(fn))):
        if r and r[0].isdigit():
            t=int(r[0]);t=t//1000 if t>10**15 else t
            return datetime.datetime.fromtimestamp(t/1000,UTC)
    return None
def months():
    out=[];d=datetime.date(2022,9,1);end=datetime.date(2026,9,1)
    while d<=end:
        out.append(d.strftime('%Y-%m'))
        d=(d.replace(day=28)+datetime.timedelta(days=4)).replace(day=1)
    return out
def fetch_k(sym,mo):
    p=CACHE/f'{sym}-{mo}.zip'
    if p.exists():return p
    u=f'{DV}/{sym}/1d/{sym}-1d-{mo}.zip'
    try:
        b=urllib.request.urlopen(urllib.request.Request(u,headers={'User-Agent':UA}),timeout=20).read();p.write_bytes(b)
    except urllib.error.HTTPError as e:
        if e.code==404:p.write_bytes(b'')
        else:raise
    return p
@lru_cache(None)
def price_daily(sym):
    out={}
    for mo in months():
        p=fetch_k(sym,mo)
        if not p.exists() or p.stat().st_size==0:continue
        z=zipfile.ZipFile(p);fn=z.namelist()[0]
        for r in csv.reader(io.TextIOWrapper(z.open(fn))):
            if not r or not r[0].isdigit():continue
            t=int(r[0]);t=t//1000 if t>10**15 else t
            d=datetime.datetime.fromtimestamp(t/1000,UTC).date()
            out[d]=(float(r[1]),float(r[4]))
    return out
@lru_cache(None)
def tvl_daily(chain):
    u='https://api.llama.fi/v2/historicalChainTvl/'+urllib.parse.quote(chain)
    a=json.loads(urllib.request.urlopen(urllib.request.Request(u,headers={'User-Agent':UA,'Accept':'application/json'}),timeout=30).read())
    out={}
    for x in a:
        try:
            d=datetime.datetime.fromtimestamp(int(x['date']),UTC).date();v=float(x['tvl'])
            if v>0:out[d]=v
        except:pass
    return out
def beta(xs,ys):
    mx=sum(xs)/len(xs);my=sum(ys)/len(ys)
    den=sum((x-mx)*(x-mx) for x in xs)
    if den<=0:return 0.0
    return sum((x-mx)*(y-my) for x,y in zip(xs,ys))/den
events=[];meta={}
for sym,chain in MAP.items():
    first=first_kline(sym)
    if not first:
        print('NO_FIRST',sym);continue
    eligible=sub_or_add_2y(first,plus=True).date()
    px=price_daily(sym);tv=tvl_daily(chain)
    dates=sorted(set(px)&set(tv))
    rows=[]
    for d in dates:
        o,c=px[d];v=tv[d]
        if o>0 and c>0 and v>0:rows.append((d,o,c,v))
    residuals=[None]*len(rows);flows=[None]*len(rows)
    for i in range(1,len(rows)):
        if (rows[i][0]-rows[i-1][0]).days!=1:continue
        rp=math.log(rows[i][2]/rows[i-1][2]);rt=math.log(rows[i][3]/rows[i-1][3])
        if i<91:continue
        xp=[];yt=[];ok=True
        for j in range(i-90,i):
            if j<=0 or (rows[j][0]-rows[j-1][0]).days!=1:ok=False;break
            xp.append(math.log(rows[j][2]/rows[j-1][2]))
            yt.append(math.log(rows[j][3]/rows[j-1][3]))
        if not ok:continue
        b=beta(xp,yt)
        residuals[i]=rt-b*rp
        if i>=6 and all(residuals[j] is not None for j in range(i-6,i+1)):
            flows[i]=sum(residuals[j] for j in range(i-6,i+1))
    n=0
    for i in range(92,len(rows)-8):
        if rows[i][0]<eligible or flows[i] is None or flows[i-1] is None:continue
        d=0.0
        if flows[i-1]<=0 and flows[i]>0:d=1.0
        elif flows[i-1]>=0 and flows[i]<0:d=-1.0
        else:continue
        if (rows[i+1][0]-rows[i][0]).days!=1:continue
        entry=rows[i+1][1]
        if entry<=0:continue
        r1=d*math.log(rows[i+1][2]/entry)
        r3=d*math.log(rows[i+3][2]/entry)
        r7=d*math.log(rows[i+7][2]/entry)
        events.append({'symbol':sym,'chain':chain,'signal_date':rows[i][0].isoformat(),'year':rows[i][0].year,'direction':'LONG' if d>0 else 'SHORT','flow7':flows[i],'r1':r1,'r3':r3,'r7':r7})
        n+=1
    meta[sym]={'chain':chain,'first_futures':first.isoformat(),'eligible_from':eligible.isoformat(),'aligned_days':len(rows),'events':n}
    print('SYMBOL',sym,'chain',chain,'eligible',eligible,'days',len(rows),'events',n,flush=True)
def summary(es):
    if not es:return {'n':0}
    sy=sorted(set(x['symbol'] for x in es));by={};pos=0
    for s in sy:
        q=[x for x in es if x['symbol']==s]
        m=sum(x['r7'] for x in q)/len(q);by[s]={'n':len(q),'mean7':m,'win7':sum(x['r7']>0 for x in q)/len(q)}
        if m>0:pos+=1
    return {'n':len(es),'mean1':sum(x['r1'] for x in es)/len(es),'mean3':sum(x['r3'] for x in es)/len(es),'mean7':sum(x['r7'] for x in es)/len(es),'win7':sum(x['r7']>0 for x in es)/len(es),'positive_symbols7':pos,'symbols':len(sy),'by_symbol':by}
res={'meta':meta,'discovery_2023_2024':summary([x for x in events if x['year'] in (2023,2024)]),'oos1_2025':summary([x for x in events if x['year']==2025]),'oos2_2026':summary([x for x in events if x['year']==2026]),'all':summary(events)}
print(json.dumps(res,indent=2),flush=True)
Path('/tmp/defillama_chain_tvl_residual_summary.json').write_text(json.dumps(res,indent=2))
Path('/tmp/defillama_chain_tvl_residual_events.json').write_text(json.dumps(events,indent=2))
