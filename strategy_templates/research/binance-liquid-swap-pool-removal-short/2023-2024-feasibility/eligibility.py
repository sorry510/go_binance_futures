import csv,datetime,io,json,time,urllib.request,urllib.error,zipfile
from functools import lru_cache
from pathlib import Path
from collections import Counter
UTC=datetime.timezone.utc
UA='Mozilla/5.0'
BASE='https://data.binance.vision/data/futures/um/monthly/klines'
ROOT=Path('strategy_templates/research/binance-liquid-swap-pool-removal-short/2023-2024-feasibility')
EVENTS=json.loads((ROOT/'inputs/events_raw.json').read_text())
CACHE=Path('/tmp/liquid_swap_removal_eligibility'); CACHE.mkdir(exist_ok=True)

def mon(x): return x.strftime('%Y-%m')
def prevmon(x):
    return f'{x.year-1}-12' if x.month==1 else f'{x.year:04d}-{x.month-1:02d}'
def sub2(x):
    try: return x.replace(year=x.year-2)
    except ValueError: return x.replace(year=x.year-2,day=28)

@lru_cache(None)
def rows(sym,m):
    zpath=CACHE/f'{sym}-1h-{m}.zip'
    miss=CACHE/f'{sym}-1h-{m}.missing'
    if miss.exists(): return []
    if not zpath.exists():
        u=f'{BASE}/{sym}/1h/{sym}-1h-{m}.zip'
        last=None
        for i in range(5):
            try:
                req=urllib.request.Request(u,headers={'User-Agent':UA})
                b=urllib.request.urlopen(req,timeout=25).read()
                zipfile.ZipFile(io.BytesIO(b))
                zpath.write_bytes(b); last=None; break
            except urllib.error.HTTPError as e:
                if e.code==404:
                    miss.write_text('404'); return []
                last=e
            except Exception as e: last=e
            time.sleep(.5*(i+1))
        if last is not None: raise last
    z=zipfile.ZipFile(zpath)
    out=[]
    with z.open(z.namelist()[0]) as f:
        for r in csv.reader(io.TextIOWrapper(f)):
            if not r or not r[0].isdigit() or len(r)<8: continue
            t=int(r[0]); t=t//1000 if t>10**15 else t
            out.append((t,float(r[7])))
    return out
res=[]
for i,e in enumerate(EVENTS,1):
    evt=datetime.datetime.fromtimestamp(e['publish_ms']/1000,UTC)
    sym=e['symbol']; rec=dict(e)
    rec['event_utc']=evt.isoformat()
    event_rows=rows(sym,mon(evt))
    if not event_rows:
        rec.update(eligible=False,reason='no_event_month'); res.append(rec); continue
    cutoff=sub2(evt); hist=rows(sym,mon(cutoff)); cut=int(cutoff.timestamp()*1000)
    if not hist or hist[0][0]>cut:
        rec.update(eligible=False,reason='history_lt_2y'); res.append(rec); continue
    hour=(e['publish_ms']//3600000)*3600000
    data=rows(sym,prevmon(evt))+event_rows
    prior=[x for x in data if hour-24*3600000<=x[0]<hour]
    if len(prior)<24:
        rec.update(eligible=False,reason='missing_prior_24h',prior_bars=len(prior)); res.append(rec); continue
    qv=sum(x[1] for x in prior); rec['quote_volume_24h']=qv
    if qv<5_000_000:
        rec.update(eligible=False,reason='qv_lt_5m'); res.append(rec); continue
    rec.update(eligible=True,reason='ok'); res.append(rec)
    if i%20==0: print('PROGRESS',i,'/',len(EVENTS),flush=True)

eligible=[x for x in res if x['eligible']]
summary={
 'raw_events':len(EVENTS),
 'raw_unique_tokens':len({x['asset'] for x in EVENTS}),
 'raw_batches':len({x['article_code'] for x in EVENTS}),
 'eligible_events':len(eligible),
 'eligible_unique_tokens':len({x['asset'] for x in eligible}),
 'eligible_batches':len({x['article_code'] for x in eligible}),
 'excluded_by_reason':dict(Counter(x['reason'] for x in res if not x['eligible'])),
 'coverage_gate_pass':(
   len(eligible)>=8 and len({x['asset'] for x in eligible})>=8 and len({x['article_code'] for x in eligible})>=5
 ),
 'post_event_returns_read':False,
}
(ROOT/'results/eligibility.json').write_text(json.dumps(res,ensure_ascii=False,indent=2))
(ROOT/'results/eligible.json').write_text(json.dumps(eligible,ensure_ascii=False,indent=2))
(ROOT/'results/eligibility_summary.json').write_text(json.dumps(summary,indent=2))
print(json.dumps(summary,indent=2))
print('TOKENS',sorted({x['asset'] for x in eligible}))
print('BATCHES',len({x['article_code'] for x in eligible}))
