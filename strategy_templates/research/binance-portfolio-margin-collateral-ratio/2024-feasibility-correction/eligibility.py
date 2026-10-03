import csv,datetime,io,json,time,urllib.request,urllib.error,zipfile
from pathlib import Path
from functools import lru_cache
from concurrent.futures import ThreadPoolExecutor,as_completed
from collections import Counter
UTC=datetime.timezone.utc;UA='Mozilla/5.0'
BASE='https://data.binance.vision/data/futures/um/monthly/klines'
ROOT=Path('strategy_templates/research/binance-portfolio-margin-collateral-ratio/2024-feasibility-correction')
EVENTS=json.loads((ROOT/'inputs/events_raw.json').read_text())
CACHE=Path('/tmp/collateral_ratio_corrected');CACHE.mkdir(exist_ok=True)
def mon(d):return d.strftime('%Y-%m')
def prevmon(d):return f'{d.year-1}-12' if d.month==1 else f'{d.year:04d}-{d.month-1:02d}'
def sub2(d):
    try:return d.replace(year=d.year-2)
    except ValueError:return d.replace(year=d.year-2,day=28)
@lru_cache(None)
def rows(sym,m):
    p=CACHE/f'{sym}-1h-{m}.zip';miss=CACHE/f'{sym}-1h-{m}.missing'
    if miss.exists():return []
    if not p.exists():
        u=f'{BASE}/{sym}/1h/{sym}-1h-{m}.zip';last=None
        for i in range(5):
            try:
                req=urllib.request.Request(u,headers={'User-Agent':UA})
                b=urllib.request.urlopen(req,timeout=25).read();zipfile.ZipFile(io.BytesIO(b))
                p.write_bytes(b);last=None;break
            except urllib.error.HTTPError as e:
                if e.code==404:miss.write_text('404');return []
                last=e
            except Exception as e:last=e
            time.sleep(.5*(i+1))
        if last is not None:raise last
    z=zipfile.ZipFile(p);out=[]
    with z.open(z.namelist()[0]) as f:
        for r in csv.reader(io.TextIOWrapper(f)):
            if not r or not r[0].isdigit() or len(r)<8:continue
            t=int(r[0]);t=t//1000 if t>10**15 else t
            out.append((t,float(r[7])))
    return out
def check(e):
    dt=datetime.datetime.fromtimestamp(e['effective_ms']/1000,UTC);rec=dict(e)
    ev=rows(e['symbol'],mon(dt))
    if not ev:return rec|{'eligible':False,'reason':'no_event_month'}
    cutdt=sub2(dt);hist=rows(e['symbol'],mon(cutdt));cut=int(cutdt.timestamp()*1000)
    if not hist or hist[0][0]>cut:return rec|{'eligible':False,'reason':'history_lt_2y'}
    hour=(e['effective_ms']//3600000)*3600000
    prior=[x for x in rows(e['symbol'],prevmon(dt))+ev if hour-24*3600000<=x[0]<hour]
    if len(prior)<24:return rec|{'eligible':False,'reason':'missing_prior_24h','prior_bars':len(prior)}
    qv=sum(x[1] for x in prior)
    if qv<5_000_000:return rec|{'eligible':False,'reason':'qv_lt_5m','quote_volume_24h':qv}
    return rec|{'eligible':True,'reason':'ok','quote_volume_24h':qv}
res=[]
with ThreadPoolExecutor(max_workers=8) as ex:
    fs=[ex.submit(check,e) for e in EVENTS]
    for i,f in enumerate(as_completed(fs),1):
        res.append(f.result())
        if i%10==0 or i==len(fs):print('CHECK',i,'/',len(fs),flush=True)
res.sort(key=lambda x:(x['effective_ms'],x['symbol']))
good=[x for x in res if x['eligible']]
summary={'raw_events':len(EVENTS),'eligible_events':len(good),
         'eligible_unique_assets':len({x['asset'] for x in good}),
         'eligible_articles':len({x['article_code'] for x in good}),
         'eligible_long':sum(x['direction']=='LONG' for x in good),
         'eligible_short':sum(x['direction']=='SHORT' for x in good),
         'excluded_by_reason':dict(Counter(x['reason'] for x in res if not x['eligible'])),
         'coverage_gate_pass':len(good)>=8,'post_event_returns_read':False}
(ROOT/'results/eligibility.json').write_text(json.dumps(res,ensure_ascii=False,indent=2))
(ROOT/'results/eligible.json').write_text(json.dumps(good,ensure_ascii=False,indent=2))
(ROOT/'results/summary.json').write_text(json.dumps(summary,indent=2))
print(json.dumps(summary,indent=2))
for x in good:print('ELIGIBLE',x['effective_utc'],x['asset'],x['before'],'->',x['after'],x['direction'],round(x['quote_volume_24h']))
