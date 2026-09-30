import csv,io,zipfile,urllib.request,urllib.error,time,datetime,math,json,re
from concurrent.futures import ThreadPoolExecutor,as_completed
from pathlib import Path
UTC=datetime.timezone.utc
BASES=['BTC','ETH','BNB','XRP','ADA','LINK','BCH','DOT','LTC']
START=datetime.date(2022,12,1); END=datetime.date(2026,9,1)
UA='Mozilla/5.0'; CACHE=Path('/tmp/term_structure_cache'); CACHE.mkdir(exist_ok=True)
S3='https://s3-ap-northeast-1.amazonaws.com/data.binance.vision'
def months():
    y,m=START.year,START.month; out=[]
    while (y,m)<=(END.year,END.month):
        out.append(f'{y:04d}-{m:02d}');m+=1
        if m==13:y+=1;m=1
    return out
def list_cm_symbols():
    import xml.etree.ElementTree as ET, urllib.parse
    u=S3+'?'+urllib.parse.urlencode({'prefix':'data/futures/cm/monthly/klines/','delimiter':'/'})
    root=ET.fromstring(urllib.request.urlopen(u,timeout=30).read())
    ns={'s':'http://s3.amazonaws.com/doc/2006-03-01/'}
    return [x.text.rstrip('/').split('/')[-1] for x in root.findall('.//s:CommonPrefixes/s:Prefix',ns)]
ALLCM=list_cm_symbols()
QS={}
for b in BASES:
    qs=[]
    for s in ALLCM:
        m=re.fullmatch(re.escape(b)+r'USD_(\d{6})',s)
        if m: qs.append(s)
    QS[b]=sorted(qs)
    print('BASE',b,'quarterlies',len(qs),qs[-4:] if qs else [],flush=True)
def url(kind,sym,mo):
    if kind=='um':return f'https://data.binance.vision/data/futures/um/monthly/klines/{sym}/1h/{sym}-1h-{mo}.zip'
    return f'https://data.binance.vision/data/futures/cm/monthly/klines/{sym}/1h/{sym}-1h-{mo}.zip'
def fetch(kind,sym,mo):
    p=CACHE/f'{kind}-{sym}-{mo}.zip'
    if p.exists():return p
    u=url(kind,sym,mo)
    for k in range(4):
        try:
            b=urllib.request.urlopen(urllib.request.Request(u,headers={'User-Agent':UA}),timeout=25).read();p.write_bytes(b);return p
        except urllib.error.HTTPError as e:
            if e.code==404:p.write_bytes(b'');return p
            time.sleep(.4*(k+1))
        except Exception:time.sleep(.4*(k+1))
    p.write_bytes(b'');return p
def readzip(p):
    if not p.exists() or p.stat().st_size==0:return {}
    z=zipfile.ZipFile(p);fn=z.namelist()[0];out={}
    for r in csv.reader(io.TextIOWrapper(z.open(fn))):
        if not r or not r[0].isdigit():continue
        t=int(r[0]);t=t//1000 if t>10**15 else t
        out[t]=(float(r[1]),float(r[4]))
    return out
def expiry_ms(sym):
    d=sym.rsplit('_',1)[1]
    dt=datetime.datetime.strptime(d,'%y%m%d').replace(tzinfo=UTC,hour=8)
    return int(dt.timestamp()*1000)
jobs=[]
for b in BASES:
    for mo in months():
        jobs.append(('um',b+'USDT',mo));jobs.append(('cm',b+'USD_PERP',mo))
        for q in QS[b]:
            exp=expiry_ms(q); y,m=map(int,mo.split('-'))
            ms=int(datetime.datetime(y,m,15,tzinfo=UTC).timestamp()*1000)
            if abs(exp-ms)<=120*86400000: jobs.append(('cm',q,mo))
jobs=list(dict.fromkeys(jobs))
print('JOBS',len(jobs),flush=True)
with ThreadPoolExecutor(max_workers=20) as ex:
    fs=[ex.submit(fetch,*j) for j in jobs]
    for i,f in enumerate(as_completed(fs),1):
        f.result()
        if i%200==0:print('DOWNLOAD',i,'/',len(jobs),flush=True)
def msd(x):
    m=sum(x)/len(x);return m,(sum((v-m)**2 for v in x)/len(x))**.5
events=[]
for b in BASES:
    seq=[]
    for mo in months():
        um=readzip(CACHE/f'um-{b}USDT-{mo}.zip')
        cp=readzip(CACHE/f'cm-{b}USD_PERP-{mo}.zip')
        qmaps={q:readzip(CACHE/f'cm-{q}-{mo}.zip') for q in QS[b] if (CACHE/f'cm-{q}-{mo}.zip').exists()}
        for t in sorted(set(um)&set(cp)):
            candidates=[]
            for q,qm in qmaps.items():
                if t not in qm:continue
                dte=(expiry_ms(q)-t)/86400000
                if dte>7:candidates.append((dte,q,qm[t][1]))
            if not candidates:continue
            dte,q,qclose=min(candidates)
            cmclose=cp[t][1]
            if qclose<=0 or cmclose<=0:continue
            basis=math.log(qclose/cmclose)*365.0/dte
            seq.append((t,um[t][0],um[t][1],basis))
    seq.sort();armed=True;n=0
    for i in range(720,len(seq)-12):
        # require hourly continuity in history and forward window
        if seq[i][0]-seq[i-720][0]>721*3600000:continue
        hist=[x[3] for x in seq[i-720:i]];m,sd=msd(hist)
        if sd<=0:continue
        z=(seq[i][3]-m)/sd
        if abs(z)<1:armed=True
        if not armed or abs(z)<3:continue
        d=-1.0 if z>0 else 1.0
        entry=seq[i+1][1]
        if entry<=0:continue
        r1=d*math.log(seq[i+1][2]/entry);r4=d*math.log(seq[i+4][2]/entry);r12=d*math.log(seq[i+12][2]/entry)
        y=datetime.datetime.fromtimestamp(seq[i][0]/1000,UTC).year
        if y>=2023:events.append((b,y,z,r1,r4,r12))
        armed=False;n+=1
    print('SYMBOL',b,'aligned',len(seq),'events',n,flush=True)
def summary(xs):
    if not xs:return {}
    syms=sorted(set(x[0] for x in xs))
    pos=0
    by={}
    for s in syms:
        q=[x for x in xs if x[0]==s];m=sum(x[5] for x in q)/len(q);by[s]={'n':len(q),'mean12':m};pos+=m>0
    return {'n':len(xs),'mean1':sum(x[3] for x in xs)/len(xs),'mean4':sum(x[4] for x in xs)/len(xs),'mean12':sum(x[5] for x in xs)/len(xs),'win12':sum(x[5]>0 for x in xs)/len(xs),'positive_symbols12':pos,'symbols':len(syms),'by_symbol':by}
res={'discovery_2023_2024':summary([x for x in events if x[1] in (2023,2024)]),'oos1_2025':summary([x for x in events if x[1]==2025]),'oos2_2026':summary([x for x in events if x[1]==2026]),'all':summary(events)}
print(json.dumps(res,indent=2),flush=True)
Path('/tmp/quarterly_term_structure_summary.json').write_text(json.dumps(res,indent=2))
