import csv,datetime,io,json,time,urllib.request,urllib.error,zipfile
from functools import lru_cache
from pathlib import Path
UTC=datetime.timezone.utc
BASE='https://data.binance.vision/data/futures/um/monthly'
UA='Mozilla/5.0'
ALL=json.load(open('/tmp/loanable_diagnostic_events.json'))
events=[x for x in ALL if datetime.datetime.fromtimestamp(x['release_time']/1000,UTC).year==2023]
CACHE=Path('/tmp/loanable_exact_cache');CACHE.mkdir(exist_ok=True)
def mon(dt):return dt.strftime('%Y-%m')
def next_month(dt):
    return (dt.replace(day=28)+datetime.timedelta(days=4)).replace(day=1,hour=0,minute=0,second=0,microsecond=0)
def get_bytes(url,key):
    p=CACHE/key
    if p.exists():return p.read_bytes() or None
    for k in range(5):
        try:
            b=urllib.request.urlopen(urllib.request.Request(url,headers={'User-Agent':UA}),timeout=30).read();p.write_bytes(b);return b
        except urllib.error.HTTPError as e:
            if e.code==404:p.write_bytes(b'');return None
            time.sleep(.5*(k+1))
        except Exception:time.sleep(.5*(k+1))
    return None
@lru_cache(None)
def klines(sym,m):
    b=get_bytes(f'{BASE}/klines/{sym}/1m/{sym}-1m-{m}.zip',f'k-{sym}-{m}.zip')
    if not b:return []
    z=zipfile.ZipFile(io.BytesIO(b));fn=z.namelist()[0];out=[]
    for r in csv.reader(io.TextIOWrapper(z.open(fn))):
        if not r or not r[0].isdigit():continue
        t=int(r[0]);t=t//1000 if t>10**15 else t
        out.append((t,float(r[1]),float(r[4])))
    return out
@lru_cache(None)
def funding(sym,m):
    b=get_bytes(f'{BASE}/fundingRate/{sym}/{sym}-fundingRate-{m}.zip',f'f-{sym}-{m}.zip')
    if not b:return []
    z=zipfile.ZipFile(io.BytesIO(b));fn=z.namelist()[0];rows=csv.reader(io.TextIOWrapper(z.open(fn)));hdr=next(rows,None)
    names=[str(x).strip().lower() for x in (hdr or [])]
    ti=names.index('calc_time') if 'calc_time' in names else 0
    ri=names.index('last_funding_rate') if 'last_funding_rate' in names else 2
    mi=names.index('mark_price') if 'mark_price' in names else None
    out=[]
    for r in rows:
        try:
            t=int(r[ti]);t=t//1000 if t>10**15 else t
            rate=float(r[ri]);mark=float(r[mi]) if mi is not None and mi<len(r) and r[mi] else 0.0
            out.append((t,rate,mark))
        except:pass
    return out
def replay(e):
    sym=e['symbol'];ev=e['release_time'];startdt=datetime.datetime.fromtimestamp(ev/1000,UTC)
    horizon_ms=ev+12*3600*1000+5*60000
    enddt=datetime.datetime.fromtimestamp(horizon_ms/1000,UTC)
    months=[];d=startdt.replace(day=1,hour=0,minute=0,second=0,microsecond=0);last=enddt.replace(day=1,hour=0,minute=0,second=0,microsecond=0)
    while d<=last:months.append(mon(d));d=next_month(d)
    bars=[]
    for m in months:bars.extend(klines(sym,m))
    bars.sort();target=(ev//60000+1)*60000
    idx=next((i for i,b in enumerate(bars) if b[0]>=target),None)
    if idx is None or bars[idx][0]>target+5*60000:return dict(e,error='entry_gap')
    ent_t,ent_o,_=bars[idx];entry=ent_o*(1-0.0005);qty=4.0/entry;open_fee=qty*entry*0.0005
    fs=[]
    for m in months:fs.extend(funding(sym,m))
    fs.sort();fi=0
    while fi<len(fs) and fs[fi][0]<ent_t:fi+=1
    funding_pnl=0.0;horizon=ent_t+12*3600*1000;exit_i=None;reason='TIME'
    for i in range(idx,len(bars)):
        t,o,c=bars[i]
        if t>horizon:break
        while fi<len(fs) and fs[fi][0]<=t+59999:
            ft,rate,mark=fs[fi]
            if ent_t<=ft<=horizon:funding_pnl += qty*(mark if mark>0 else c)*rate
            fi+=1
        roi=(entry-c)/c*4*100
        if roi<=-6:reason='SL';exit_i=min(i+1,len(bars)-1);break
        if roi>=8:reason='TP';exit_i=min(i+1,len(bars)-1);break
        if t>=horizon:reason='TIME';exit_i=min(i+1,len(bars)-1);break
    if exit_i is None:exit_i=min(idx+12*60,len(bars)-1)
    xt,xo,_=bars[exit_i];exitp=xo*(1+0.0005)
    gross=(entry-exitp)*qty;close_fee=qty*exitp*0.0005;net=gross-open_fee-close_fee+funding_pnl
    return dict(e,entry_time=ent_t,exit_time=xt,trigger=reason,entry=entry,exit=exitp,
                gross_pct=gross*100,fees_pct=(open_fee+close_fee)*100,funding_pct=funding_pnl*100,net_pct=net*100)
def summarize(xs):
    gp=sum(max(x['net_pct'],0) for x in xs);gl=-sum(min(x['net_pct'],0) for x in xs)
    batches={}
    for x in xs:batches.setdefault(x['article_code'],[]).append(x['net_pct'])
    bm=[sum(v)/len(v) for v in batches.values()]
    syms={}
    for x in xs:syms.setdefault(x['symbol'],[]).append(x['net_pct'])
    return {'n':len(xs),'wins':sum(x['net_pct']>0 for x in xs),'losses':sum(x['net_pct']<=0 for x in xs),
            'tp':sum(x['trigger']=='TP' for x in xs),'sl':sum(x['trigger']=='SL' for x in xs),'time':sum(x['trigger']=='TIME' for x in xs),
            'pf':gp/gl if gl else None,'net_pct':sum(x['net_pct'] for x in xs),'avg_pct':sum(x['net_pct'] for x in xs)/len(xs) if xs else None,
            'symbols':len(syms),'positive_symbols':sum(sum(v)/len(v)>0 for v in syms.values()),
            'batches':len(batches),'batch_equal_net_pct':sum(bm)/len(bm) if bm else None,'positive_batches':sum(v>0 for v in bm)}
res=[]
for i,e in enumerate(events,1):
    x=replay(e);res.append(x)
    if i%5==0 or i==len(events):print('PROGRESS',i,'/',len(events),x['symbol'],x.get('trigger'),round(x.get('net_pct',0),4),flush=True)
good=[x for x in res if 'error' not in x]
summary=summarize(good)
print(json.dumps(summary,indent=2),flush=True)
Path('/tmp/loanable_exact_2023_trades.json').write_text(json.dumps(res,ensure_ascii=False,indent=2))
Path('/tmp/loanable_exact_2023_summary.json').write_text(json.dumps(summary,indent=2))
