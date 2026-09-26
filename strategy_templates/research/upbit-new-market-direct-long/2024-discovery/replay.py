import json,datetime,urllib.request,urllib.error,zipfile,io,csv,time
from functools import lru_cache
from pathlib import Path
UTC=datetime.timezone.utc
BASE='https://data.binance.vision/data/futures/um/monthly'
UA='Mozilla/5.0'
ROOT=Path(__file__).resolve().parent
events=[x for x in json.load(open(ROOT/'inputs'/'eligibility.json')) if x.get('eligible')]

def get_bytes(url,tries=5,timeout=30):
    last=None
    for k in range(tries):
        try:
            req=urllib.request.Request(url,headers={'User-Agent':UA})
            with urllib.request.urlopen(req,timeout=timeout) as r:return r.read()
        except urllib.error.HTTPError as e:
            if e.code==404:return None
            last=e
        except Exception as e:last=e
        time.sleep(.4*(k+1))
    raise last
def month(dt):return dt.strftime('%Y-%m')
def next_month(dt):
    if dt.month==12:return dt.replace(year=dt.year+1,month=1,day=1)
    return dt.replace(month=dt.month+1,day=1)
@lru_cache(None)
def load_month(sym,m):
    u=f'{BASE}/klines/{sym}/1m/{sym}-1m-{m}.zip'
    b=get_bytes(u)
    if not b:return []
    z=zipfile.ZipFile(io.BytesIO(b));fn=z.namelist()[0];out=[]
    for r in csv.reader(io.TextIOWrapper(z.open(fn))):
        if not r or not r[0].isdigit():continue
        t=int(r[0]); 
        if t>10**15:t//=1000
        out.append((t,float(r[1]),float(r[4])))
    return out
@lru_cache(None)
def load_funding(sym,m):
    u=f'{BASE}/fundingRate/{sym}/{sym}-fundingRate-{m}.zip'
    b=get_bytes(u)
    if not b:return []
    z=zipfile.ZipFile(io.BytesIO(b));fn=z.namelist()[0];out=[]
    rows=csv.reader(io.TextIOWrapper(z.open(fn)));hdr=next(rows,None)
    names=[str(x).strip().lower() for x in (hdr or [])]
    ti=names.index('calc_time') if 'calc_time' in names else 0
    ri=names.index('last_funding_rate') if 'last_funding_rate' in names else 2
    mi=names.index('mark_price') if 'mark_price' in names else None
    for r in rows:
        try:
            t=int(r[ti]); 
            if t>10**15:t//=1000
            rate=float(r[ri]);mark=float(r[mi]) if mi is not None and mi<len(r) and r[mi] else 0.0
            out.append((t,rate,mark))
        except:pass
    return out
def slip(p,entering):
    return p*(1.0005 if entering else 0.9995)
def roi_long(entry,p):
    return (p-entry)/p*4*100

def replay(ev):
    evms=ev['event_ms'];sym=ev['symbol'];horizon=evms+72*3600*1000
    startdt=datetime.datetime.fromtimestamp(evms/1000,UTC)
    enddt=datetime.datetime.fromtimestamp((horizon+120000)/1000,UTC)
    cur=startdt.replace(day=1,hour=0,minute=0,second=0,microsecond=0)
    endm=enddt.replace(day=1,hour=0,minute=0,second=0,microsecond=0)
    months=[]
    while cur<=endm:months.append(month(cur));cur=next_month(cur)
    bars=[]
    for m in months:bars.extend(load_month(sym,m))
    bars.sort()
    target=(evms//60000+1)*60000
    idx=next((i for i,b in enumerate(bars) if b[0]>=target),None)
    if idx is None:return ev|{'error':'no_entry'}
    ent_ts,ent_open,_=bars[idx]
    if ent_ts>evms+5*60000:return ev|{'error':'entry_gap','entry_ts':ent_ts}
    entry=slip(ent_open,True);qty=4.0/entry
    open_fee=qty*entry*0.0005
    funds=[]
    for m in months:funds.extend(load_funding(sym,m))
    funds.sort();fi=0
    while fi<len(funds) and funds[fi][0]<ent_ts:fi+=1
    funding=0.0;trigger=None;exit_idx=None;last=idx
    i=idx
    while i<len(bars):
        t,op,cl=bars[i]
        if t>horizon+60000:break
        last=i
        while fi<len(funds) and funds[fi][0]<=t+59999:
            ft,rate,mark=funds[fi]
            if ft>=ent_ts and ft<=horizon:
                funding-=qty*(mark if mark>0 else cl)*rate
            fi+=1
        if t>=horizon:
            trigger='TIME';exit_idx=i;break
        roi=roi_long(entry,cl)
        if roi<=-6:
            trigger='SL';exit_idx=min(i+1,len(bars)-1);break
        if roi>=8:
            trigger='TP';exit_idx=min(i+1,len(bars)-1);break
        i+=1
    if exit_idx is None:trigger='EOF';exit_idx=last
    xt,xo,xc=bars[exit_idx]
    raw=xo if trigger in ('TP','SL','TIME') else xc
    exitp=slip(raw,False)
    gross=(exitp-entry)*qty
    close_fee=qty*exitp*0.0005
    net=gross-open_fee-close_fee+funding
    return ev|{'entry_ts':ent_ts,'exit_ts':xt,'entry':entry,'exit':exitp,'trigger':trigger,
               'gross_pct':gross*100,'fees_pct':(open_fee+close_fee)*100,'funding_pct':funding*100,
               'net_pct':net*100,'hold_h':(xt-ent_ts)/3600000}

res=[]
for i,e in enumerate(events,1):
    x=replay(e);res.append(x)
    print(i,len(events),x['symbol'],x.get('trigger'),round(x.get('net_pct',0),4),flush=True)
good=[x for x in res if not x.get('error')]
gp=sum(max(x['net_pct'],0) for x in good);gl=-sum(min(x['net_pct'],0) for x in good)
pf=gp/gl if gl>0 else None
summary={'n':len(good),'wins':sum(x['net_pct']>0 for x in good),'losses':sum(x['net_pct']<=0 for x in good),
         'tp':sum(x.get('trigger')=='TP' for x in good),'sl':sum(x.get('trigger')=='SL' for x in good),
         'time':sum(x.get('trigger')=='TIME' for x in good),'pf':pf,'net_pct':sum(x['net_pct'] for x in good),
         'avg_pct':sum(x['net_pct'] for x in good)/len(good) if good else None,'gp':gp,'gl':gl}
print(json.dumps(summary,indent=2))
(ROOT/'results'/'trades.json').write_text(json.dumps(res,ensure_ascii=False,indent=2))
(ROOT/'results'/'summary.json').write_text(json.dumps(summary,indent=2))
