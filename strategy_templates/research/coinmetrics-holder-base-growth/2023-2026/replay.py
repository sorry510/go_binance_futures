import urllib.request,urllib.parse,json,csv,io,zipfile,time,datetime,math
from concurrent.futures import ThreadPoolExecutor,as_completed
from pathlib import Path
ASSETS=['btc','eth','xrp','ada','link','bch','ltc','doge','uni','zec']
SYM={a:a.upper()+'USDT' for a in ASSETS}
START='2022-12-01';END='2026-09-20'
UA='Mozilla/5.0';CACHE=Path('/tmp/holder_base_growth_cache');CACHE.mkdir(exist_ok=True)
def cm_data():
    base='https://community-api.coinmetrics.io/v4/timeseries/asset-metrics'
    params={'assets':','.join(ASSETS),'metrics':'AdrBalCnt','frequency':'1d','start_time':START,'end_time':END,'page_size':10000}
    u=base+'?'+urllib.parse.urlencode(params)
    rows=[]
    while u:
        req=urllib.request.Request(u,headers={'User-Agent':UA,'Accept':'application/json'})
        o=json.loads(urllib.request.urlopen(req,timeout=30).read())
        rows.extend(o.get('data',[]));u=o.get('next_page_url')
        if u and u.startswith('https://api.coinmetrics.io'):u=u.replace('https://api.coinmetrics.io','https://community-api.coinmetrics.io',1)
        if u:time.sleep(.7)
    return rows
rows=cm_data();print('CM_ROWS',len(rows),flush=True)
by={a:{} for a in ASSETS}
for r in rows:
    try:
        a=r['asset'];d=r['time'][:10];v=float(r['AdrBalCnt'])
        if a in by and v>0:by[a][d]=v
    except:pass
for a in ASSETS:print('CM',a,len(by[a]),min(by[a]) if by[a] else None,max(by[a]) if by[a] else None,flush=True)
def months():
    s=datetime.date.fromisoformat(START).replace(day=1);e=datetime.date.fromisoformat(END).replace(day=1);out=[];d=s
    while d<=e:
        out.append(d.strftime('%Y-%m'))
        d=(d.replace(day=28)+datetime.timedelta(days=4)).replace(day=1)
    return out
def fetch(sym,mo):
    p=CACHE/f'{sym}-{mo}.zip'
    if p.exists():return p
    u=f'https://data.binance.vision/data/futures/um/monthly/klines/{sym}/1d/{sym}-1d-{mo}.zip'
    for k in range(4):
        try:
            b=urllib.request.urlopen(urllib.request.Request(u,headers={'User-Agent':UA}),timeout=25).read();p.write_bytes(b);return p
        except urllib.error.HTTPError as e:
            if e.code==404:p.write_bytes(b'');return p
            time.sleep(.4*(k+1))
        except Exception:time.sleep(.4*(k+1))
    p.write_bytes(b'');return p
import urllib.error
jobs=[(SYM[a],m) for a in ASSETS for m in months()]
with ThreadPoolExecutor(max_workers=16) as ex:
    fs=[ex.submit(fetch,*j) for j in jobs]
    for i,f in enumerate(as_completed(fs),1):
        f.result()
        if i%100==0:print('VISION',i,'/',len(jobs),flush=True)
def price_map(sym):
    out={}
    for mo in months():
        p=CACHE/f'{sym}-{mo}.zip'
        if not p.exists() or p.stat().st_size==0:continue
        z=zipfile.ZipFile(p);fn=z.namelist()[0]
        for r in csv.reader(io.TextIOWrapper(z.open(fn))):
            if not r or not r[0].isdigit():continue
            t=int(r[0]);t=t//1000 if t>10**15 else t
            d=datetime.datetime.fromtimestamp(t/1000,datetime.timezone.utc).date().isoformat()
            out[d]=(float(r[1]),float(r[4]))
    return out
events=[]
for a in ASSETS:
    pm=price_map(SYM[a]); ds=sorted(set(by[a])&set(pm))
    vals=[by[a][d] for d in ds]
    growth=[None]*len(ds)
    for i in range(13,len(ds)):
        cur=sum(vals[i-6:i+1])/7;prev=sum(vals[i-13:i-6])/7
        if cur>0 and prev>0:growth[i]=math.log(cur/prev)
    ev=[];prev_g=None
    for i in range(13,len(ds)-8):
        g=growth[i]
        if g is None:continue
        sig=0
        if prev_g is not None and prev_g<=0 and g>0:sig=1
        elif prev_g is not None and prev_g>=0 and g<0:sig=-1
        prev_g=g
        if sig==0:continue
        # signal day's network metric is only used after completion; enter next UTC day open
        entry_i=i+1;entry=pm[ds[entry_i]][0]
        if entry<=0:continue
        r1=sig*math.log(pm[ds[entry_i]][1]/entry)
        r3=sig*math.log(pm[ds[entry_i+2]][1]/entry)
        r7=sig*math.log(pm[ds[entry_i+6]][1]/entry)
        y=int(ds[i][:4]);ev.append((a,y,ds[i],sig,g,r1,r3,r7))
    events.extend(ev);print('EVENTS',a,len(ev),flush=True)
def summ(xs):
    if not xs:return {}
    sy=sorted(set(x[0] for x in xs));byx={};pos=0
    for s in sy:
        q=[x for x in xs if x[0]==s];m=sum(x[7] for x in q)/len(q);byx[s]={'n':len(q),'mean7':m};pos+=m>0
    return {'n':len(xs),'mean1':sum(x[5] for x in xs)/len(xs),'mean3':sum(x[6] for x in xs)/len(xs),'mean7':sum(x[7] for x in xs)/len(xs),'win7':sum(x[7]>0 for x in xs)/len(xs),'positive_symbols7':pos,'symbols':len(sy),'by_symbol':byx}
res={'discovery_2023_2024':summ([x for x in events if x[1] in (2023,2024)]),'oos1_2025':summ([x for x in events if x[1]==2025]),'oos2_2026':summ([x for x in events if x[1]==2026]),'all':summ(events)}
print(json.dumps(res,indent=2),flush=True)
Path('/tmp/holder_base_growth_summary.json').write_text(json.dumps(res,indent=2))
