import csv,io,zipfile,urllib.request,urllib.error,time,datetime,json,re
from concurrent.futures import ThreadPoolExecutor,as_completed
from functools import lru_cache
from pathlib import Path
UTC=datetime.timezone.utc; UA='Mozilla/5.0'
SRC=Path('/Users/zhz/work/binance/go_binance_futures/strategy_templates/research/binance-leverage-margin-tier-change/2023-2024-discovery/inputs/events_parsed.json')
VISION='https://data.binance.vision/data/futures/um/monthly/klines'
CACHE=Path('/tmp/first_tier_capacity_cache');CACHE.mkdir(exist_ok=True)
def cap(s):
    s=str(s).replace(',','')
    nums=[float(x) for x in re.findall(r'\d+(?:\.\d+)?',s)]
    return max(nums) if nums else None
events=[]
for x in json.load(open(SRC)):
    if x.get('classification')!='unchanged':continue
    row=x.get('first_row') or []
    if len(row)<6:continue
    old,new=cap(row[1]),cap(row[4])
    if old is None or new is None or old==new:continue
    y=dict(x);y['old_first_notional_cap']=old;y['new_first_notional_cap']=new
    y['capacity_class']='expansion' if new>old else 'contraction'
    events.append(y)
print('CAPACITY_EVENTS',len(events),'EXP',sum(x['capacity_class']=='expansion' for x in events),'CON',sum(x['capacity_class']=='contraction' for x in events),flush=True)
def month_ms(ms):return datetime.datetime.fromtimestamp(ms/1000,UTC).strftime('%Y-%m')
def prevmonth(ms):
    d=datetime.datetime.fromtimestamp(ms/1000,UTC);y,m=d.year,d.month
    return f'{y-1}-12' if m==1 else f'{y:04d}-{m-1:02d}'
def nextmonth(ms):
    d=datetime.datetime.fromtimestamp(ms/1000,UTC);y,m=d.year,d.month
    return f'{y+1}-01' if m==12 else f'{y:04d}-{m+1:02d}'
def sub2(ms):
    d=datetime.datetime.fromtimestamp(ms/1000,UTC)
    try:d=d.replace(year=d.year-2)
    except ValueError:d=d.replace(year=d.year-2,day=28)
    return int(d.timestamp()*1000)
def fetch(sym,mo):
    p=CACHE/f'{sym}-{mo}.zip'
    if p.exists():return p
    u=f'{VISION}/{sym}/1h/{sym}-1h-{mo}.zip'
    for k in range(4):
        try:
            b=urllib.request.urlopen(urllib.request.Request(u,headers={'User-Agent':UA}),timeout=20).read();p.write_bytes(b);return p
        except urllib.error.HTTPError as e:
            if e.code==404:p.write_bytes(b'');return p
            time.sleep(.4*(k+1))
        except Exception:time.sleep(.4*(k+1))
    p.write_bytes(b'');return p
jobs=set()
for e in events:
    jobs.add((e['symbol'],month_ms(sub2(e['effective_time']))))
    for mo in (prevmonth(e['effective_time']),month_ms(e['effective_time']),nextmonth(e['effective_time'])):jobs.add((e['symbol'],mo))
print('JOBS',len(jobs),flush=True)
with ThreadPoolExecutor(max_workers=16) as ex:
    fs=[ex.submit(fetch,*j) for j in jobs]
    for i,f in enumerate(as_completed(fs),1):
        f.result()
        if i%100==0:print('DOWNLOAD',i,'/',len(jobs),flush=True)
@lru_cache(None)
def rows(sym,mo):
    p=CACHE/f'{sym}-{mo}.zip'
    if not p.exists() or p.stat().st_size==0:return []
    z=zipfile.ZipFile(p);fn=z.namelist()[0];out=[]
    for r in csv.reader(io.TextIOWrapper(z.open(fn))):
        if not r or not r[0].isdigit():continue
        t=int(r[0]);t=t//1000 if t>10**15 else t
        out.append((t,float(r[1]),float(r[4]),float(r[7]) if len(r)>7 else 0.0))
    return out
good=[];excluded={}
for e in events:
    sym=e['symbol'];evt=e['effective_time'];cut=sub2(evt)
    hist=rows(sym,month_ms(cut))
    if not hist or hist[0][0]>cut:
        excluded['history_lt_2y_or_missing']=excluded.get('history_lt_2y_or_missing',0)+1;continue
    hour=(evt//3600000)*3600000;start=hour-24*3600000;end=hour-1
    prior=rows(sym,prevmonth(evt))+rows(sym,month_ms(evt))
    prior=[r for r in prior if start<=r[0]<=end];qv=sum(r[3] for r in prior)
    if len(prior)<24:
        excluded['missing_24h_bars']=excluded.get('missing_24h_bars',0)+1;continue
    if qv<5_000_000:
        excluded['quote_volume_lt_5m']=excluded.get('quote_volume_lt_5m',0)+1;continue
    entry_hour=((evt//3600000)+1)*3600000;wend=entry_hour+13*3600000
    fwd=rows(sym,month_ms(evt))+rows(sym,nextmonth(evt))
    fwd=[r for r in fwd if entry_hour<=r[0]<=wend];fwd.sort()
    if len(fwd)<13 or fwd[0][0]!=entry_hour:
        excluded['missing_forward_bars']=excluded.get('missing_forward_bars',0)+1;continue
    sign=1.0 if e['capacity_class']=='expansion' else -1.0;entry=fwd[0][1]
    x=dict(e);x['quote_volume_24h']=qv;x['entry_time']=entry_hour
    x['r1']=sign*__import__('math').log(fwd[0][2]/entry)
    x['r4']=sign*__import__('math').log(fwd[3][2]/entry)
    x['r12']=sign*__import__('math').log(fwd[11][2]/entry)
    good.append(x)
print('ELIGIBLE_GOOD',len(good),'EXCLUDED',excluded,flush=True)
def stats(a,key):
    v=[x[key] for x in a]
    return {'n':len(v),'mean':sum(v)/len(v) if v else None,'win':sum(x>0 for x in v)/len(v) if v else None}
for k in ('r1','r4','r12'):print(k,stats(good,k))
for cls in ('expansion','contraction'):
    q=[x for x in good if x['capacity_class']==cls];print('CLASS',cls,'r12',stats(q,'r12'))
for y in (2023,2024):
    q=[x for x in good if datetime.datetime.fromtimestamp(x['effective_time']/1000,UTC).year==y];print('YEAR',y,'r12',stats(q,'r12'))
by={}
for x in good:by.setdefault(x['symbol'],[]).append(x['r12'])
print('SYMBOLS',len(by),'POSITIVE',sum(sum(v)/len(v)>0 for v in by.values()))
Path('/tmp/first_tier_capacity_results.json').write_text(json.dumps(good,ensure_ascii=False,indent=2))
summary={'capacity_events':len(events),'eligible':len(good),'excluded':excluded,'r1':stats(good,'r1'),'r4':stats(good,'r4'),'r12':stats(good,'r12'),'expansion_r12':stats([x for x in good if x['capacity_class']=='expansion'],'r12'),'contraction_r12':stats([x for x in good if x['capacity_class']=='contraction'],'r12'),'symbols':len(by),'positive_symbols12':sum(sum(v)/len(v)>0 for v in by.values())}
Path('/tmp/first_tier_capacity_summary.json').write_text(json.dumps(summary,indent=2))
