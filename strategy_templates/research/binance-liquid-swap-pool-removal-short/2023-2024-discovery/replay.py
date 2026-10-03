import csv,datetime,io,json,math,time,urllib.request,urllib.error,zipfile
from functools import lru_cache
from pathlib import Path
from collections import defaultdict

UTC=datetime.timezone.utc
UA='Mozilla/5.0'
BASE='https://data.binance.vision/data/futures/um/monthly/klines'
ROOT=Path('strategy_templates/research/binance-liquid-swap-pool-removal-short/2023-2024-discovery')
EVENTS=json.loads((ROOT/'inputs/eligible.json').read_text())
CACHE=Path('/tmp/liquid_swap_removal_discovery'); CACHE.mkdir(exist_ok=True)

def month(dt): return dt.strftime('%Y-%m')
def next_month(dt):
    if dt.month==12: return dt.replace(year=dt.year+1,month=1,day=1)
    return dt.replace(month=dt.month+1,day=1)

@lru_cache(None)
def bars(sym,m):
    zpath=CACHE/f'{sym}-1h-{m}.zip'
    miss=CACHE/f'{sym}-1h-{m}.missing'
    if miss.exists(): return {}
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
                    miss.write_text('404'); return {}
                last=e
            except Exception as e: last=e
            time.sleep(.5*(i+1))
        if last is not None: raise last
    z=zipfile.ZipFile(zpath); out={}
    with z.open(z.namelist()[0]) as f:
        for r in csv.reader(io.TextIOWrapper(f)):
            if not r or not r[0].isdigit() or len(r)<5: continue
            t=int(r[0]); t=t//1000 if t>10**15 else t
            out[t]=(float(r[1]),float(r[4]))
    return out
def mean(xs): return sum(xs)/len(xs) if xs else 0.0

out=[]
errors=[]
for i,e in enumerate(EVENTS,1):
    pub=int(e['publish_ms'])
    entry=((pub//3600000)+1)*3600000
    edt=datetime.datetime.fromtimestamp(entry/1000,UTC)
    # Audit confirmed every 12h endpoint stays in the entry calendar month.
    # Load only the event month; no outcome semantics change.
    data=bars(e['symbol'],month(edt))
    if entry not in data:
        errors.append({**e,'reason':'missing_entry_bar','entry_time':entry}); continue
    entry_px=data[entry][0]
    vals={}
    ok=True
    for h in (1,4,12):
        t=entry+(h-1)*3600000
        if t not in data:
            ok=False; errors.append({**e,'reason':f'missing_{h}h_bar','entry_time':entry}); break
        close=data[t][1]
        vals[h]=-math.log(close/entry_px)
    if not ok: continue
    out.append({
      **e,'entry_time':entry,'entry_price':entry_px,
      'r1':vals[1],'r4':vals[4],'r12':vals[12]
    })
    if i%25==0: print('PROGRESS',i,'/',len(EVENTS),flush=True)

by_token=defaultdict(list); by_batch=defaultdict(list); by_month=defaultdict(list)
for x in out:
    by_token[x['asset']].append(x)
    by_batch[x['article_code']].append(x)
    dt=datetime.datetime.fromtimestamp(x['publish_ms']/1000,UTC)
    by_month[dt.strftime('%Y-%m')].append(x)

def sm(rows):
    return {'n':len(rows),'mean_r1':mean([x['r1'] for x in rows]),'mean_r4':mean([x['r4'] for x in rows]),
            'mean_r12':mean([x['r12'] for x in rows]),'win_r12':mean([1.0 if x['r12']>0 else 0.0 for x in rows])}
overall=sm(out)
token_stats={k:sm(v) for k,v in sorted(by_token.items())}
batch_stats={k:sm(v) for k,v in sorted(by_batch.items())}
month_stats={k:sm(v) for k,v in sorted(by_month.items())}
batch_equal=mean([v['mean_r12'] for v in batch_stats.values()])
positive_tokens=sum(1 for v in token_stats.values() if v['mean_r12']>0)
positive_batches=sum(1 for v in batch_stats.values() if v['mean_r12']>0)
gate=(overall['mean_r12']>=0.005 and batch_equal>=0.005 and
      positive_tokens/max(1,len(token_stats))>=0.60 and
      positive_batches/max(1,len(batch_stats))>=0.60)

summary={
 'status':'promote_exact' if gate else 'frozen_failed_discovery',
 'gate_pass':gate,
 'events':len(out),
 'unique_tokens':len(token_stats),
 'independent_batches':len(batch_stats),
 'overall':overall,
 'batch_equal_mean_r12':batch_equal,
 'positive_tokens':f'{positive_tokens}/{len(token_stats)}',
 'positive_token_fraction':positive_tokens/max(1,len(token_stats)),
 'positive_batches':f'{positive_batches}/{len(batch_stats)}',
 'positive_batch_fraction':positive_batches/max(1,len(batch_stats)),
 'by_token':token_stats,
 'by_batch':batch_stats,
 'by_signal_month':month_stats,
 'replay_errors':errors,
 'exact_tp_sl_run':False,
 'later_oos_evaluated':False
}
(ROOT/'results/events.json').write_text(json.dumps(out,ensure_ascii=False,indent=2))
(ROOT/'results/replay_errors.json').write_text(json.dumps(errors,ensure_ascii=False,indent=2))
(ROOT/'results/summary.json').write_text(json.dumps(summary,ensure_ascii=False,indent=2))
print(json.dumps({
 'events':len(out),'tokens':len(token_stats),'batches':len(batch_stats),
 'mean_r1':overall['mean_r1'],'mean_r4':overall['mean_r4'],'mean_r12':overall['mean_r12'],
 'batch_equal_mean_r12':batch_equal,
 'positive_tokens':summary['positive_tokens'],'positive_batches':summary['positive_batches'],
 'gate_pass':gate,'errors':len(errors)
},indent=2))
for k,v in month_stats.items():
    print('MONTH',k,v)
