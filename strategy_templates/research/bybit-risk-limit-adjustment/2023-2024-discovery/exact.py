import json,datetime,urllib.request,urllib.error,time,zipfile,io,csv
from functools import lru_cache
from pathlib import Path
UTC=datetime.timezone.utc;UA='Mozilla/5.0'
EV=[x for x in json.load(open('/tmp/bybit_risk_limit_eligibility.json')) if x.get('eligible')]
BASE='https://data.binance.vision/data/futures/um/monthly'
CACHE=Path('/tmp/bybit_risk_limit_exact_cache');CACHE.mkdir(exist_ok=True)
def month(ms):return datetime.datetime.fromtimestamp(ms/1000,UTC).strftime('%Y-%m')
def get(url,key):
    p=CACHE/key
    if p.exists():return p.read_bytes() or None
    for k in range(5):
        try:
            b=urllib.request.urlopen(urllib.request.Request(url,headers={'User-Agent':UA}),timeout=30).read()
            p.write_bytes(b);return b
        except urllib.error.HTTPError as e:
            if e.code==404:p.write_bytes(b'');return None
            time.sleep(.4*(k+1))
        except Exception:time.sleep(.4*(k+1))
    return None
@lru_cache(None)
def bars(sym,mo):
    b=get(f'{BASE}/klines/{sym}/1m/{sym}-1m-{mo}.zip',f'k-{sym}-{mo}.zip')
    if not b:return []
    z=zipfile.ZipFile(io.BytesIO(b));fn=z.namelist()[0];out=[]
    for r in csv.reader(io.TextIOWrapper(z.open(fn))):
        if not r or not r[0].isdigit():continue
        t=int(r[0]);t=t//1000 if t>10**15 else t
        out.append((t,float(r[1]),float(r[4])))
    return out
@lru_cache(None)
def funding(sym,mo):
    b=get(f'{BASE}/fundingRate/{sym}/{sym}-fundingRate-{mo}.zip',f'f-{sym}-{mo}.zip')
    if not b:return []
    z=zipfile.ZipFile(io.BytesIO(b));fn=z.namelist()[0]
    rows=csv.reader(io.TextIOWrapper(z.open(fn)));hdr=next(rows,None)
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
def slip(price,side,enter):
    rate=5/10000
    if (side=='LONG' and enter) or (side=='SHORT' and not enter):return price*(1+rate)
    return price*(1-rate)
def roi(entry,price,side):
    if side=='LONG':return (price-entry)/price*4*100
    return (entry-price)/price*4*100
def replay(e):
    evms=int(e['event_ts']*1000);sym=e['symbol'];side='LONG' if e['classification']=='expansion' else 'SHORT'
    mo=month(evms);bs=bars(sym,mo);target=((evms//60000)+1)*60000
    idx=next((i for i,b in enumerate(bs) if b[0]>=target),None)
    if idx is None or bs[idx][0]>evms+5*60000:return e|{'error':'entry_gap'}
    ent_ts,ent_open,_=bs[idx];entry=slip(ent_open,side,True);qty=4.0/entry;open_fee=qty*entry*0.0005
    fs=funding(sym,mo);fi=0
    while fi<len(fs) and fs[fi][0]<ent_ts:fi+=1
    fp=0.0;horizon=evms+12*3600*1000;trigger=None;exit_idx=None;last=idx
    for i in range(idx,len(bs)):
        ts,op,cl=bs[i]
        if ts>horizon+60000:break
        last=i
        while fi<len(fs) and fs[fi][0]<=ts+59999:
            ft,rate,mark=fs[fi]
            if ent_ts<=ft<=horizon:
                px=mark if mark>0 else cl
                fp += (-1 if side=='LONG' else 1)*qty*px*rate
            fi+=1
        if ts>=horizon:
            trigger='TIME';exit_idx=i;break
        rr=roi(entry,cl,side)
        if rr<=-6:trigger='SL';exit_idx=i+1 if i+1<len(bs) else i;break
        if rr>=8:trigger='TP';exit_idx=i+1 if i+1<len(bs) else i;break
    if exit_idx is None:trigger='EOF';exit_idx=last
    xt,xo,xc=bs[exit_idx];raw=xo if trigger in ('TP','SL','TIME') else xc;exitp=slip(raw,side,False)
    gross=((exitp-entry) if side=='LONG' else (entry-exitp))*qty
    close_fee=qty*exitp*0.0005;net=gross-open_fee-close_fee+fp
    return e|{'side':side,'entry_ts':ent_ts,'exit_ts':xt,'trigger':trigger,'entry':entry,'exit':exitp,
              'gross_pct':gross*100,'fees_pct':(open_fee+close_fee)*100,'funding_pct':fp*100,'net_pct':net*100}
R=[]
for i,e in enumerate(EV,1):
    x=replay(e);R.append(x);print(i,len(EV),x['symbol'],x.get('side'),x.get('trigger'),round(x.get('net_pct',0),4),flush=True)
G=[x for x in R if not x.get('error')]
def summ(q):
    gp=sum(max(x['net_pct'],0) for x in q);gl=-sum(min(x['net_pct'],0) for x in q)
    return {'n':len(q),'wins':sum(x['net_pct']>0 for x in q),'losses':sum(x['net_pct']<=0 for x in q),
            'tp':sum(x.get('trigger')=='TP' for x in q),'sl':sum(x.get('trigger')=='SL' for x in q),
            'time':sum(x.get('trigger')=='TIME' for x in q),'pf':gp/gl if gl>0 else None,
            'net_pct':sum(x['net_pct'] for x in q),'avg_pct':sum(x['net_pct'] for x in q)/len(q) if q else None}
from collections import defaultdict
B=defaultdict(list)
for x in G:B[x['article_objectID']].append(x)
out={'all':summ(G),'expansion':summ([x for x in G if x['classification']=='expansion']),
     'contraction':summ([x for x in G if x['classification']=='contraction']),
     'batches':{k:summ(v) for k,v in B.items()}}
print(json.dumps(out,indent=2),flush=True)
Path('/tmp/bybit_risk_limit_exact_trades.json').write_text(json.dumps(R,ensure_ascii=False,indent=2))
Path('/tmp/bybit_risk_limit_exact_summary.json').write_text(json.dumps(out,indent=2))
