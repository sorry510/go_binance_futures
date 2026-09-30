import csv,io,zipfile,urllib.request,urllib.error,time,datetime,math,json,re,urllib.parse,xml.etree.ElementTree as ET
from pathlib import Path
from concurrent.futures import ThreadPoolExecutor,as_completed
UTC=datetime.timezone.utc
BASES=['BTC','ETH','BNB','XRP','ADA','LINK','BCH','DOT','LTC']
CACHE=Path('/tmp/term_structure_cache');CACHE.mkdir(exist_ok=True)
UA='Mozilla/5.0'
START=datetime.date(2022,11,1);END=datetime.date(2026,9,1)
def months(a=START,b=END):
    y,m=a.year,a.month;out=[]
    while (y,m)<=(b.year,b.month):
        out.append(f'{y:04d}-{m:02d}');m+=1
        if m==13:y+=1;m=1
    return out
def add_months(d,n):
    y=d.year+(d.month-1+n)//12;m=(d.month-1+n)%12+1
    return datetime.date(y,m,1)
def fetch(url,key):
    p=CACHE/key
    if p.exists():return p
    last=None
    for k in range(4):
        try:
            b=urllib.request.urlopen(urllib.request.Request(url,headers={'User-Agent':UA}),timeout=25).read();p.write_bytes(b);return p
        except urllib.error.HTTPError as e:
            if e.code==404:p.write_bytes(b'');return p
            last=e
        except Exception as e:last=e
        time.sleep(.4*(k+1))
    raise last
def readzip(p):
    if p.stat().st_size==0:return {}
    z=zipfile.ZipFile(p);fn=z.namelist()[0];out={}
    for r in csv.reader(io.TextIOWrapper(z.open(fn))):
        if not r or not r[0].isdigit():continue
        t=int(r[0]);t=t//1000 if t>10**15 else t
        out[t]=(float(r[1]),float(r[4]))
    return out
u='https://s3-ap-northeast-1.amazonaws.com/data.binance.vision?'+urllib.parse.urlencode({'prefix':'data/futures/cm/monthly/klines/','delimiter':'/'})
root=ET.fromstring(urllib.request.urlopen(u,timeout=30).read())
ns={'s':'http://s3.amazonaws.com/doc/2006-03-01/'}
syms=[x.text.rstrip('/').split('/')[-1] for x in root.findall('.//s:CommonPrefixes/s:Prefix',ns)]
quarters={}
for b in BASES:
    q=[]
    for s in syms:
        m=re.fullmatch(re.escape(b+'USD_')+r'(\d{6})',s)
        if m:q.append((s,m.group(1)))
    quarters[b]=sorted(q,key=lambda x:x[1])
jobs=[]
for b in BASES:
    for mo in months():
        jobs.append(('cm',b+'USD_PERP',mo))
        jobs.append(('um',b+'USDT',mo))
    for s,suf in quarters[b]:
        yy=2000+int(suf[:2]);mm=int(suf[2:4]);dd=int(suf[4:6])
        exp=datetime.date(yy,mm,dd)
        if exp<datetime.date(2023,1,1) or exp>datetime.date(2026,12,31):continue
        for k in range(-4,1):jobs.append(('cm',s,add_months(exp.replace(day=1),k).strftime('%Y-%m')))
def job_url(kind,s,mo):
    if kind=='cm':return f'https://data.binance.vision/data/futures/cm/monthly/klines/{s}/1h/{s}-1h-{mo}.zip'
    return f'https://data.binance.vision/data/futures/um/monthly/klines/{s}/1h/{s}-1h-{mo}.zip'
jobs=list(dict.fromkeys(jobs))
with ThreadPoolExecutor(max_workers=16) as ex:
    fs={ex.submit(fetch,job_url(*j),f'{j[0]}-{j[1]}-{j[2]}.zip'):j for j in jobs}
    for i,f in enumerate(as_completed(fs),1):
        f.result()
        if i%150==0:print('DOWNLOAD',i,'/',len(jobs),flush=True)
def load_series(kind,s):
    out={}
    for mo in months():out.update(readzip(CACHE/f'{kind}-{s}-{mo}.zip'))
    return out
def expiry_ms(suf):
    d=datetime.datetime(2000+int(suf[:2]),int(suf[2:4]),int(suf[4:6]),8,tzinfo=UTC)
    return int(d.timestamp()*1000)
events=[]
for b in BASES:
    cm=load_series('cm',b+'USD_PERP');um=load_series('um',b+'USDT')
    qmaps=[]
    for s,suf in quarters[b]:
        e=expiry_ms(suf)
        if e<int(datetime.datetime(2023,1,1,tzinfo=UTC).timestamp()*1000):continue
        q={}
        for k in range(-4,1):
            d=datetime.datetime.fromtimestamp(e/1000,UTC).date().replace(day=1)
            mo=add_months(d,k).strftime('%Y-%m');p=CACHE/f'cm-{s}-{mo}.zip'
            if p.exists():q.update(readzip(p))
        if q:qmaps.append((e,s,q))
    seq=[]
    first_um=min(um) if um else 0
    for t in sorted(set(cm)&set(um)):
        if t-first_um<730*86400000:continue
        cand=[]
        for e,s,q in qmaps:
            dte=(e-t)/86400000
            if dte>7 and t in q:cand.append((e,s,q[t][1],dte))
        if not cand:continue
        e,s,qc,dte=min(cand,key=lambda x:x[0])
        pc=cm[t][1]
        if pc<=0 or qc<=0:continue
        basis=math.log(qc/pc)*365.0/dte
        seq.append((t,basis))
    armed=True;n=0
    for i in range(720,len(seq)):
        t,zbase=seq[i]
        hist=[x[1] for x in seq[i-720:i]]
        m=sum(hist)/len(hist);sd=(sum((x-m)**2 for x in hist)/len(hist))**0.5
        if sd<=0:continue
        z=(zbase-m)/sd
        if abs(z)<1:armed=True
        if not armed or abs(z)<3:continue
        d=-1.0 if z>0 else 1.0
        t1=t+3600000;t4=t+4*3600000;t12=t+12*3600000
        if t1 not in um or t4 not in um or t12 not in um:continue
        entry=um[t1][0]
        if entry<=0:continue
        r1=d*math.log(um[t1][1]/entry)
        r4=d*math.log(um[t4][1]/entry)
        r12=d*math.log(um[t12][1]/entry)
        y=datetime.datetime.fromtimestamp(t/1000,UTC).year
        events.append((b,y,z,r1,r4,r12));armed=False;n+=1
    print('SYMBOL',b,'basis_hours',len(seq),'events',n,flush=True)
def summary(xs):
    if not xs:return {}
    sy=set(x[0] for x in xs)
    return {'n':len(xs),'mean1':sum(x[3] for x in xs)/len(xs),'mean4':sum(x[4] for x in xs)/len(xs),'mean12':sum(x[5] for x in xs)/len(xs),
            'win12':sum(x[5]>0 for x in xs)/len(xs),'positive_symbols12':sum((sum(x[5] for x in xs if x[0]==s)/len([x for x in xs if x[0]==s]))>0 for s in sy),'symbols':len(sy)}
res={'discovery_2023_2024':summary([x for x in events if x[1] in (2023,2024)]),
     'oos1_2025':summary([x for x in events if x[1]==2025]),
     'oos2_2026':summary([x for x in events if x[1]==2026]),
     'all':summary(events)}
print(json.dumps(res,indent=2),flush=True)
Path('/tmp/quarterly_term_structure_summary.json').write_text(json.dumps(res,indent=2))
