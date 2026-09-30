import csv,io,zipfile,urllib.request,urllib.error,time,datetime,json,math
from concurrent.futures import ThreadPoolExecutor,as_completed
from pathlib import Path
UTC=datetime.timezone.utc;UA='Mozilla/5.0'
MAP={'BTCUSDT':'BTCUSD_PERP','ETHUSDT':'ETHUSD_PERP','BNBUSDT':'BNBUSD_PERP','XRPUSDT':'XRPUSD_PERP'}
SYMS=list(MAP)
START=datetime.date(2022,10,1);END=datetime.date(2024,12,1)
CACHE=Path('/tmp/cm_um_funding_div_cache');CACHE.mkdir(exist_ok=True)
def months():
    y,m=START.year,START.month;out=[]
    while (y,m)<=(END.year,END.month):
        out.append(f'{y:04d}-{m:02d}');m+=1
        if m==13:y+=1;m=1
    return out
def get(url):
    for k in range(4):
        try:return urllib.request.urlopen(urllib.request.Request(url,headers={'User-Agent':UA}),timeout=20).read()
        except urllib.error.HTTPError as e:
            if e.code==404:return None
            time.sleep(.4*(k+1))
        except Exception:time.sleep(.4*(k+1))
    return None
def fetch(kind,sym,mo):
    p=CACHE/f'{kind}-{sym}-{mo}.zip'
    if p.exists():return p
    market='um' if kind in ('umfund','umk') else 'cm'
    cat='fundingRate' if kind in ('umfund','cmfund') else 'klines'
    interval='' if cat=='fundingRate' else '/1h'
    base=f'https://data.binance.vision/data/futures/{market}/monthly/{cat}/{sym}{interval}/{sym}-{"fundingRate" if cat=="fundingRate" else "1h"}-{mo}.zip'
    b=get(base);p.write_bytes(b or b'');return p
jobs=[]
for usym,csym in MAP.items():
    for mo in months():
        jobs += [('umfund',usym,mo),('cmfund',csym,mo),('umk',usym,mo)]
with ThreadPoolExecutor(max_workers=16) as ex:
    fs=[ex.submit(fetch,*j) for j in jobs]
    for i,f in enumerate(as_completed(fs),1):
        f.result()
        if i%100==0:print('DOWNLOAD',i,'/',len(jobs),flush=True)
def parse_fund(kind,sym):
    out={}
    for mo in months():
        p=CACHE/f'{kind}-{sym}-{mo}.zip'
        if not p.exists() or p.stat().st_size==0:continue
        z=zipfile.ZipFile(p);fn=z.namelist()[0];rd=csv.reader(io.TextIOWrapper(z.open(fn)));hdr=next(rd,None)
        names=[str(x).strip().lower() for x in hdr]
        ti=names.index('calc_time');ii=names.index('funding_interval_hours');ri=names.index('last_funding_rate')
        for r in rd:
            try:
                t=int(r[ti]);t=t//1000 if t>10**15 else t
                h=float(r[ii]);rate=float(r[ri])
                if h>0:out[t]=rate/h
            except:pass
    return out
def parse_k(sym):
    out={}
    for mo in months():
        p=CACHE/f'umk-{sym}-{mo}.zip'
        if not p.exists() or p.stat().st_size==0:continue
        z=zipfile.ZipFile(p);fn=z.namelist()[0]
        for r in csv.reader(io.TextIOWrapper(z.open(fn))):
            if not r or not r[0].isdigit():continue
            t=int(r[0]);t=t//1000 if t>10**15 else t
            out[t]=(float(r[1]),float(r[4]))
    return out
def stats(xs):
    if not xs:return {}
    sy=sorted(set(x['symbol'] for x in xs));by={}
    for s in sy:
        q=[x for x in xs if x['symbol']==s];by[s]={'n':len(q),'mean12':sum(x['r12'] for x in q)/len(q)}
    return {'n':len(xs),'mean1':sum(x['r1'] for x in xs)/len(xs),'mean4':sum(x['r4'] for x in xs)/len(xs),'mean12':sum(x['r12'] for x in xs)/len(xs),'win12':sum(x['r12']>0 for x in xs)/len(xs),'positive_symbols12':sum(v['mean12']>0 for v in by.values()),'symbols':len(sy),'by_symbol':by}
events=[]
for usym,csym in MAP.items():
    u=parse_fund('umfund',usym);c=parse_fund('cmfund',csym);px=parse_k(usym);ts=sorted(set(u)&set(c))
    diffs=[(t,c[t]-u[t]) for t in ts];armed=True;n=0
    for t,d in diffs:
        if datetime.datetime.fromtimestamp(t/1000,UTC).year not in (2023,2024):continue
        prior=[v for tt,v in diffs if t-90*86400000<=tt<t]
        if len(prior)<200:continue
        m=sum(prior)/len(prior);sd=(sum((v-m)**2 for v in prior)/len(prior))**.5
        if sd<=0:continue
        z=(d-m)/sd
        if abs(z)<1:armed=True
        if not armed or abs(z)<3:continue
        ent=((t//3600000)+1)*3600000
        bars=[px.get(ent+i*3600000) for i in range(13)]
        if any(x is None for x in bars):continue
        direction=-1.0 if d>0 else 1.0;entry=bars[0][0]
        events.append({'symbol':usym,'time':t,'z':z,'diff_per_hour':d,
            'r1':direction*math.log(bars[0][1]/entry),
            'r4':direction*math.log(bars[3][1]/entry),
            'r12':direction*math.log(bars[11][1]/entry)})
        armed=False;n+=1
    print('SYMBOL',usym,'paired',len(ts),'events',n,flush=True)
res={'gate_2023_2024':stats(events)}
print(json.dumps(res,indent=2),flush=True)
Path('/tmp/cm_um_funding_divergence_summary.json').write_text(json.dumps(res,indent=2))
