import csv,io,zipfile,urllib.request,urllib.error,datetime,json
from functools import lru_cache
from pathlib import Path

UTC=datetime.timezone.utc
UA='Mozilla/5.0'
ROOT=Path('strategy_templates/research/binance-token-trading-fee-change/2023-2024-feasibility')
BATCHES=json.loads((ROOT/'inputs/qualifying_batches.json').read_text())
BASE='https://data.binance.vision/data/futures/um/monthly/klines'
MAP={'PEPE':'1000PEPEUSDT','BONK':'1000BONKUSDT'}

def sym(t): return MAP.get(t,t+'USDT')
def dt(ms): return datetime.datetime.fromtimestamp(ms/1000,UTC)
def mon(x): return x.strftime('%Y-%m')
def sub2(x):
    try:return x.replace(year=x.year-2)
    except ValueError:return x.replace(year=x.year-2,day=28)
def prevmon(x):
    return f'{x.year-1}-12' if x.month==1 else f'{x.year:04d}-{x.month-1:02d}'

@lru_cache(None)
def fetch(symbol,m):
    u=f'{BASE}/{symbol}/1h/{symbol}-1h-{m}.zip'
    try:
        return urllib.request.urlopen(urllib.request.Request(u,headers={'User-Agent':UA}),timeout=15).read()
    except urllib.error.HTTPError as e:
        if e.code==404:return b''
        raise
    except Exception:
        return b''

@lru_cache(None)
def rows(symbol,m):
    b=fetch(symbol,m)
    if not b:return []
    z=zipfile.ZipFile(io.BytesIO(b)); fn=z.namelist()[0]; out=[]
    for r in csv.reader(io.TextIOWrapper(z.open(fn))):
        if not r or not r[0].isdigit():continue
        t=int(r[0]); t=t//1000 if t>10**15 else t
        out.append((t,float(r[7]) if len(r)>7 else 0.0))
    return out

details=[]
batch_results=[]
for b in BATCHES:
    e=dt(b['release_time']); hour=(b['release_time']//3600000)*3600000
    eligible=[]
    for t in b['tokens']:
        s=sym(t); cutoff=sub2(e); hist=rows(s,mon(cutoff))
        rec={'code':b['code'],'ticker':t,'symbol':s,'side':b['side'],'release_time':b['release_time']}
        if not hist or hist[0][0]>int(cutoff.timestamp()*1000):
            rec.update(eligible=False,reason='history_lt_2y_or_missing');details.append(rec);continue
        prior=[x for x in rows(s,prevmon(e))+rows(s,mon(e)) if hour-86400000<=x[0]<hour]
        if len(prior)<24:
            rec.update(eligible=False,reason='missing_24h_bars',prior_bars=len(prior));details.append(rec);continue
        qv=sum(x[1] for x in prior); rec['quote_volume_24h']=qv
        if qv<5_000_000:
            rec.update(eligible=False,reason='qv_lt_5m');details.append(rec);continue
        rec.update(eligible=True,reason='ok');details.append(rec);eligible.append(t)
    batch_results.append({'code':b['code'],'title':b['title'],'side':b['side'],'eligible_tokens':eligible,'eligible_batch':bool(eligible)})

summary={
 'raw_qualifying_batches':len(BATCHES),
 'production_eligible_batches':sum(x['eligible_batch'] for x in batch_results),
 'coverage_gate_pass':sum(x['eligible_batch'] for x in batch_results)>=8,
 'post_event_returns_read':False,
 'batch_results':batch_results
}
(ROOT/'results/eligibility.json').write_text(json.dumps(details,ensure_ascii=False,indent=2))
(ROOT/'results/summary.json').write_text(json.dumps(summary,ensure_ascii=False,indent=2))
print(json.dumps(summary,ensure_ascii=False,indent=2))
