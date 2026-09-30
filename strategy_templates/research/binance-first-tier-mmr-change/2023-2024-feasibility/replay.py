import csv,io,zipfile,urllib.request,urllib.error,time,datetime,json,re
from concurrent.futures import ThreadPoolExecutor,as_completed
from functools import lru_cache
from pathlib import Path
UTC=datetime.timezone.utc;UA='Mozilla/5.0'
SRC=Path('/Users/zhz/work/binance/go_binance_futures/strategy_templates/research/binance-leverage-margin-tier-change/2023-2024-discovery/inputs/events_parsed.json')
VISION='https://data.binance.vision/data/futures/um/monthly/klines';CACHE=Path('/tmp/first_tier_mmr_cache');CACHE.mkdir(exist_ok=True)
def nums(s):
    return [float(x) for x in re.findall(r'\d+(?:\.\d+)?',str(s).replace(',',''))]
def cap(s):
    x=nums(s);return max(x) if x else None
def pct(s):
    x=nums(s);return x[-1] if x else None
events=[]
for x in json.load(open(SRC)):
    r=x.get('first_row') or []
    if len(r)<6:continue
    oldcap,newcap=cap(r[1]),cap(r[4]);oldm,newm=pct(r[2]),pct(r[5])
    if x.get('old_first_max_leverage')!=x.get('new_first_max_leverage') or oldcap!=newcap:continue
    if oldm is None or newm is None or oldm==newm:continue
    y=dict(x);y['old_first_mmr_pct']=oldm;y['new_first_mmr_pct']=newm;y['mmr_class']='decrease' if newm<oldm else 'increase';events.append(y)
print('MMR_EVENTS',len(events),'DEC',sum(x['mmr_class']=='decrease' for x in events),'INC',sum(x['mmr_class']=='increase' for x in events),flush=True)
def month(ms):return datetime.datetime.fromtimestamp(ms/1000,UTC).strftime('%Y-%m')
def prevm(ms):
    d=datetime.datetime.fromtimestamp(ms/1000,UTC);return f'{d.year-1}-12' if d.month==1 else f'{d.year:04d}-{d.month-1:02d}'
def nextm(ms):
    d=datetime.datetime.fromtimestamp(ms/1000,UTC);return f'{d.year+1}-01' if d.month==12 else f'{d.year:04d}-{d.month+1:02d}'
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
        try:b=urllib.request.urlopen(urllib.request.Request(u,headers={'User-Agent':UA}),timeout=20).read();p.write_bytes(b);return p
        except urllib.error.HTTPError as e:
            if e.code==404:p.write_bytes(b'');return p
            time.sleep(.4*(k+1))
        except Exception:time.sleep(.4*(k+1))
    p.write_bytes(b'');return p
jobs=set()
for e in events:
    jobs.add((e['symbol'],month(sub2(e['effective_time']))))
    for m in (prevm(e['effective_time']),month(e['effective_time']),nextm(e['effective_time'])):jobs.add((e['symbol'],m))
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
        t=int(r[0]);t=t//1000 if t>10**15 else t;out.append((t,float(r[1]),float(r[4]),float(r[7]) if len(r)>7 else 0.0))
    return out
good=[];excluded={}
for e in events:
    sym=e['symbol'];evt=e['effective_time'];cut=sub2(evt);hist=rows(sym,month(cut))
    if not hist or hist[0][0]>cut:excluded['history_lt_2y_or_missing']=excluded.get('history_lt_2y_or_missing',0)+1;continue
    hour=(evt//3600000)*3600000;prior=[r for r in rows(sym,prevm(evt))+rows(sym,month(evt)) if hour-86400000<=r[0]<=hour-1];qv=sum(r[3] for r in prior)
    if len(prior)<24:excluded['missing_24h_bars']=excluded.get('missing_24h_bars',0)+1;continue
    if qv<5_000_000:excluded['quote_volume_lt_5m']=excluded.get('quote_volume_lt_5m',0)+1;continue
    ent=((evt//3600000)+1)*3600000;fwd=[r for r in rows(sym,month(evt))+rows(sym,nextm(evt)) if ent<=r[0]<=ent+13*3600000];fwd.sort()
    if len(fwd)<13 or fwd[0][0]!=ent:excluded['missing_forward_bars']=excluded.get('missing_forward_bars',0)+1;continue
    sign=1.0 if e['mmr_class']=='decrease' else -1.0;entry=fwd[0][1];import math
    x=dict(e);x['quote_volume_24h']=qv;x['r1']=sign*math.log(fwd[0][2]/entry);x['r4']=sign*math.log(fwd[3][2]/entry);x['r12']=sign*math.log(fwd[11][2]/entry);good.append(x)
def stats(a,k):
    v=[x[k] for x in a];return {'n':len(v),'mean':sum(v)/len(v) if v else None,'win':sum(z>0 for z in v)/len(v) if v else None}
print('ELIGIBLE',len(good),'EXCLUDED',excluded)
for k in ('r1','r4','r12'):print(k,stats(good,k))
for c in ('decrease','increase'):print('CLASS',c,stats([x for x in good if x['mmr_class']==c],'r12'))
by={}
for x in good:by.setdefault(x['symbol'],[]).append(x['r12'])
print('SYMBOLS',len(by),'POSITIVE',sum(sum(v)/len(v)>0 for v in by.values()))
summary={'events':len(events),'eligible':len(good),'excluded':excluded,'r1':stats(good,'r1'),'r4':stats(good,'r4'),'r12':stats(good,'r12'),'decrease_r12':stats([x for x in good if x['mmr_class']=='decrease'],'r12'),'increase_r12':stats([x for x in good if x['mmr_class']=='increase'],'r12'),'symbols':len(by),'positive_symbols12':sum(sum(v)/len(v)>0 for v in by.values())}
Path('/tmp/first_tier_mmr_results.json').write_text(json.dumps(good,ensure_ascii=False,indent=2));Path('/tmp/first_tier_mmr_summary.json').write_text(json.dumps(summary,indent=2))
