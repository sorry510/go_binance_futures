import csv,datetime,io,json,time,urllib.request,urllib.error,zipfile
from functools import lru_cache
from pathlib import Path
from collections import Counter

UTC=datetime.timezone.utc
UA='Mozilla/5.0'
BASE='https://data.binance.vision/data/futures/um/monthly/klines'
ROOT=Path('strategy_templates/research/binance-spot-tick-size-cross-market/2021-2022-increase-short-oot')
EVENTS=json.loads((ROOT/'inputs/increase_short_events_raw.json').read_text())
CACHE=Path('/tmp/spot_tick_oot_eligibility'); CACHE.mkdir(exist_ok=True)

def mon(x): return x.strftime('%Y-%m')
def prevmon(x):
    return f'{x.year-1}-12' if x.month==1 else f'{x.year:04d}-{x.month-1:02d}'
def sub_days(x,days): return x-datetime.timedelta(days=days)

@lru_cache(None)
def rows(sym,m):
    zp=CACHE/f'{sym}-1h-{m}.zip'; miss=CACHE/f'{sym}-1h-{m}.missing'
    if miss.exists(): return []
    if not zp.exists():
        u=f'{BASE}/{sym}/1h/{sym}-1h-{m}.zip'
        last=None
        for i in range(5):
            try:
                req=urllib.request.Request(u,headers={'User-Agent':UA})
                b=urllib.request.urlopen(req,timeout=25).read()
                zipfile.ZipFile(io.BytesIO(b))
                zp.write_bytes(b); last=None; break
            except urllib.error.HTTPError as e:
                if e.code==404:
                    miss.write_text('404'); return []
                last=e
            except Exception as e:
                last=e
            time.sleep(.5*(i+1))
        if last is not None: raise last
    z=zipfile.ZipFile(zp); out=[]
    with z.open(z.namelist()[0]) as f:
        for r in csv.reader(io.TextIOWrapper(f)):
            if not r or not r[0].isdigit() or len(r)<8: continue
            t=int(r[0]); t=t//1000 if t>10**15 else t
            out.append((t,float(r[7])))
    return out

res=[]
for i,e in enumerate(EVENTS,1):
    evt=datetime.datetime.fromtimestamp(e['effective_ms']/1000,UTC)
    sym=e['symbol']; rec=dict(e)
    erows=rows(sym,mon(evt))
    if not erows:
        rec.update(eligible=False,reason='no_event_month'); res.append(rec); continue
    cutoff=sub_days(evt,730); hrows=rows(sym,mon(cutoff)); cut=int(cutoff.timestamp()*1000)
    if not hrows or hrows[0][0]>cut:
        rec.update(eligible=False,reason='history_lt_730d'); res.append(rec); continue
    hour=(e['effective_ms']//3600000)*3600000
    data=rows(sym,prevmon(evt))+erows
    prior=[x for x in data if hour-24*3600000<=x[0]<hour]
    if len(prior)<24:
        rec.update(eligible=False,reason='missing_prior_24h',prior_bars=len(prior)); res.append(rec); continue
    qv=sum(x[1] for x in prior); rec['qv24']=qv
    if qv<5_000_000:
        rec.update(eligible=False,reason='qv_lt_5m'); res.append(rec); continue
    rec.update(eligible=True,reason='ok'); res.append(rec)
    if i%25==0: print('PROGRESS',i,'/',len(EVENTS),flush=True)

eligible=[x for x in res if x['eligible']]
summary={
 'raw_increase_events':len(EVENTS),
 'raw_unique_symbols':len({x['symbol'] for x in EVENTS}),
 'raw_batches':len({x['effective_utc'] for x in EVENTS}),
 'eligible_events':len(eligible),
 'eligible_unique_symbols':len({x['symbol'] for x in eligible}),
 'eligible_batches':len({x['effective_utc'] for x in eligible}),
 'excluded_by_reason':dict(Counter(x['reason'] for x in res if not x['eligible'])),
 'returns_read':False
}
(ROOT/'inputs/eligibility.json').write_text(json.dumps(res,ensure_ascii=False,indent=2))
(ROOT/'inputs/eligible_short_events.json').write_text(json.dumps(eligible,ensure_ascii=False,indent=2))
(ROOT/'results/eligibility_summary.json').write_text(json.dumps(summary,indent=2))
print(json.dumps(summary,indent=2))
for x in eligible:
    print('ELIGIBLE',x['effective_utc'],x['symbol'],'qv24',round(x['qv24'],2))
