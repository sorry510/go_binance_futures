import csv,io,zipfile,urllib.request,urllib.error,time,datetime,math,json,os
from concurrent.futures import ThreadPoolExecutor,as_completed
from pathlib import Path
SYMS=['BTCUSDT','ETHUSDT','BNBUSDT','XRPUSDT','SOLUSDT','DOGEUSDT','LTCUSDT','AVAXUSDT','UNIUSDT','ZECUSDT']
START=datetime.date(2023,1,1); END=datetime.date(2026,9,1)
UA='Mozilla/5.0'; CACHE=Path('/tmp/basis_cache'); CACHE.mkdir(exist_ok=True)
def months():
    y,m=START.year,START.month; out=[]
    while (y,m)<=(END.year,END.month):
        out.append(f'{y:04d}-{m:02d}')
        m+=1
        if m==13:y+=1;m=1
    return out
def url(kind,sym,mo):
    if kind=='spot':return f'https://data.binance.vision/data/spot/monthly/klines/{sym}/1h/{sym}-1h-{mo}.zip'
    return f'https://data.binance.vision/data/futures/um/monthly/klines/{sym}/1h/{sym}-1h-{mo}.zip'
def fetch(kind,sym,mo):
    p=CACHE/f'{kind}-{sym}-{mo}.zip'
    if p.exists(): return p
    u=url(kind,sym,mo); last=None
    for k in range(4):
        try:
            b=urllib.request.urlopen(urllib.request.Request(u,headers={'User-Agent':UA}),timeout=25).read();p.write_bytes(b);return p
        except urllib.error.HTTPError as e:
            if e.code==404:p.write_bytes(b'');return p
            last=e
        except Exception as e:last=e
        time.sleep(.4*(k+1))
    raise last
def readzip(p):
    if p.stat().st_size==0:return {}
    z=zipfile.ZipFile(p); fn=z.namelist()[0]; out={}
    for r in csv.reader(io.TextIOWrapper(z.open(fn))):
        if not r or not r[0].isdigit():continue
        t=int(r[0]); t=t//1000 if t>10**15 else t
        out[t]=(float(r[1]),float(r[4]))
    return out
jobs=[(k,s,m) for s in SYMS for m in months() for k in ('spot','fut')]
with ThreadPoolExecutor(max_workers=16) as ex:
    fs={ex.submit(fetch,*j):j for j in jobs}
    for i,f in enumerate(as_completed(fs),1):
        f.result()
        if i%100==0:print('DOWNLOAD',i,'/',len(jobs),flush=True)
def msd(xs):
    m=sum(xs)/len(xs); sd=(sum((x-m)**2 for x in xs)/len(xs))**.5
    return m,sd
events=[]
for si,s in enumerate(SYMS,1):
    seq=[]
    for mo in months():
        sp=readzip(CACHE/f'spot-{s}-{mo}.zip'); fu=readzip(CACHE/f'fut-{s}-{mo}.zip')
        for t in sorted(set(sp)&set(fu)):
            so,sc=sp[t]; fo,fc=fu[t]
            if sc>0 and fc>0:seq.append((t,fo,fc,math.log(fc/sc)))
    seq.sort(); armed=True; n=0
    for i in range(720,len(seq)-12):
        hist=[x[3] for x in seq[i-720:i]]; m,sd=msd(hist)
        if sd<=0:continue
        z=(seq[i][3]-m)/sd
        if abs(z)<1:armed=True
        if not armed or abs(z)<3:continue
        d=-1.0 if z>0 else 1.0
        entry=seq[i+1][1]
        r1=d*math.log(seq[i+1][2]/entry)
        r4=d*math.log(seq[i+4][2]/entry)
        r12=d*math.log(seq[i+12][2]/entry)
        y=datetime.datetime.fromtimestamp(seq[i][0]/1000,datetime.timezone.utc).year
        events.append((s,y,z,r1,r4,r12)); armed=False; n+=1
    print('SYMBOL',s,'aligned',len(seq),'events',n,flush=True)
def summary(xs):
    if not xs:return {}
    return {'n':len(xs),'mean1':sum(x[3] for x in xs)/len(xs),'mean4':sum(x[4] for x in xs)/len(xs),
    'mean12':sum(x[5] for x in xs)/len(xs),'win12':sum(x[5]>0 for x in xs)/len(xs),
    'positive_symbols12':sum(sum(x[5] for x in xs if x[0]==s)/len([x for x in xs if x[0]==s])>0 for s in set(x[0] for x in xs)),
    'symbols':len(set(x[0] for x in xs))}
res={'discovery_2023_2024':summary([x for x in events if x[1] in (2023,2024)]),
'oos1_2025':summary([x for x in events if x[1]==2025]),
'oos2_2026':summary([x for x in events if x[1]==2026]),
'all':summary(events)}
print(json.dumps(res,indent=2),flush=True)
Path('/tmp/spot_perp_basis_summary.json').write_text(json.dumps(res,indent=2))
