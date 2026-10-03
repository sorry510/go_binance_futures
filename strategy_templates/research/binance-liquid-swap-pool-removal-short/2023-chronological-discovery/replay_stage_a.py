import csv,datetime,io,json,math,time,urllib.request,urllib.error,zipfile
from functools import lru_cache
from pathlib import Path
from collections import defaultdict

UTC=datetime.timezone.utc
UA='Mozilla/5.0'
ROOT=Path('strategy_templates/research/binance-liquid-swap-pool-removal-short/2023-chronological-discovery')
EVENTS=json.loads((ROOT/'inputs/stage_a_events.json').read_text())
VISION='https://data.binance.vision/data/futures/um/monthly/klines'
CACHE=Path('/tmp/liquid_swap_removal_eligibility')
HOUR=3600000

def month_from_ms(ms):
    return datetime.datetime.fromtimestamp(ms/1000,UTC).strftime('%Y-%m')

@lru_cache(None)
def rows(sym,month):
    path=CACHE/f'{sym}-1h-{month}.zip'
    if not path.exists():
        u=f'{VISION}/{sym}/1h/{sym}-1h-{month}.zip'
        last=None
        for i in range(5):
            try:
                req=urllib.request.Request(u,headers={'User-Agent':UA})
                b=urllib.request.urlopen(req,timeout=25).read()
                zipfile.ZipFile(io.BytesIO(b))
                path.write_bytes(b); last=None; break
            except Exception as e:
                last=e; time.sleep(.5*(i+1))
        if last is not None: raise last
    z=zipfile.ZipFile(path); out={}
    with z.open(z.namelist()[0]) as f:
        for r in csv.reader(io.TextIOWrapper(f)):
            if not r or not r[0].isdigit() or len(r)<5: continue
            t=int(r[0]); t=t//1000 if t>10**15 else t
            out[t]=(float(r[1]),float(r[4]))
    return out

def mean(xs):
    return sum(xs)/len(xs) if xs else 0.0
out=[]
for e in EVENTS:
    entry=((int(e['publish_ms'])//HOUR)+1)*HOUR
    m=month_from_ms(entry)
    data=rows(e['symbol'],m)
    if entry not in data:
        raise RuntimeError(f"missing entry bar {e['symbol']} {entry}")
    ep=data[entry][0]
    rs={}
    for h in (1,4,12):
        t=entry+(h-1)*HOUR
        if t not in data:
            # month-boundary fallback
            data2=rows(e['symbol'],month_from_ms(t))
            if t not in data2: raise RuntimeError(f"missing endpoint {e['symbol']} {t}")
            cp=data2[t][1]
        else:
            cp=data[t][1]
        # preregistered SHORT
        rs[h]=-math.log(cp/ep)
    out.append({
      'article_code':e['article_code'],'title':e['title'],'asset':e['asset'],'symbol':e['symbol'],
      'publish_ms':e['publish_ms'],'entry_ms':entry,'side':'SHORT',
      'r1':rs[1],'r4':rs[4],'r12':rs[12],
    })

out.sort(key=lambda x:(x['publish_ms'],x['asset']))
with (ROOT/'results/stage_a/events.csv').open('w',newline='') as f:
    cols=['article_code','title','asset','symbol','publish_ms','entry_ms','side','r1','r4','r12']
    w=csv.DictWriter(f,fieldnames=cols); w.writeheader(); w.writerows(out)
by_token=defaultdict(list)
by_batch=defaultdict(list)
for e in out:
    by_token[e['asset']].append(e)
    by_batch[e['article_code']].append(e)

def summ(rows):
    return {
      'n':len(rows),
      'mean_r1':mean([x['r1'] for x in rows]),
      'mean_r4':mean([x['r4'] for x in rows]),
      'mean_r12':mean([x['r12'] for x in rows]),
      'win_r12':mean([1.0 if x['r12']>0 else 0.0 for x in rows])
    }

overall=summ(out)
token_stats={k:summ(v) for k,v in sorted(by_token.items())}
batch_stats={k:summ(v) for k,v in sorted(by_batch.items())}
pos_tokens=sum(1 for v in token_stats.values() if v['mean_r12']>0)
pos_batches=sum(1 for v in batch_stats.values() if v['mean_r12']>0)
batch_equal=mean([v['mean_r12'] for v in batch_stats.values()])
gate=(
    overall['mean_r12']>=0.005 and
    batch_equal>=0.005 and
    pos_tokens/len(token_stats)>=0.60 and
    pos_batches/len(batch_stats)>=0.60
)
summary={
  'stage':'A',
  'status':'stage_a_pass' if gate else 'frozen_failed_stage_a',
  'gate_pass':gate,
  'events':len(out),
  'unique_tokens':len(token_stats),
  'independent_batches':len(batch_stats),
  'overall':overall,
  'batch_equal_mean_r12':batch_equal,
  'positive_tokens':f'{pos_tokens}/{len(token_stats)}',
  'positive_token_fraction':pos_tokens/len(token_stats),
  'positive_batches':f'{pos_batches}/{len(batch_stats)}',
  'positive_batch_fraction':pos_batches/len(batch_stats),
  'by_token':token_stats,
  'by_batch':batch_stats,
  'stage_b_post_event_returns_read':False,
}
(ROOT/'results/stage_a/summary.json').write_text(json.dumps(summary,ensure_ascii=False,indent=2))
print(json.dumps({
 'events':summary['events'],'tokens':summary['unique_tokens'],'batches':summary['independent_batches'],
 'mean_r1':overall['mean_r1'],'mean_r4':overall['mean_r4'],'mean_r12':overall['mean_r12'],
 'batch_equal_r12':batch_equal,'positive_tokens':summary['positive_tokens'],
 'positive_batches':summary['positive_batches'],'gate_pass':gate
},indent=2))
