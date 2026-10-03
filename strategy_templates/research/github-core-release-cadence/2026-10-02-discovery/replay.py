import csv,datetime,io,json,math,time,urllib.request,urllib.error,zipfile
from collections import defaultdict
from functools import lru_cache
from pathlib import Path

UTC=datetime.timezone.utc
UA='Mozilla/5.0'
BASE='https://data.binance.vision/data/futures/um/monthly/klines'
ROOT=Path('strategy_templates/research/github-core-release-cadence/2026-10-02-discovery')
EVENTS=json.loads((ROOT/'inputs/eligible_signals.json').read_text())
CACHE=Path('/tmp/v162-github-release-cadence'); CACHE.mkdir(exist_ok=True)

def month(dt): return dt.strftime('%Y-%m')
def next_month(dt):
    if dt.month==12: return dt.replace(year=dt.year+1,month=1,day=1)
    return dt.replace(month=dt.month+1,day=1)

@lru_cache(None)
def bars(sym,m):
    zp=CACHE/f'{sym}-1h-{m}.zip'; miss=CACHE/f'{sym}-1h-{m}.missing'
    if miss.exists(): return {}
    if not zp.exists():
        url=f'{BASE}/{sym}/1h/{sym}-1h-{m}.zip'
        last=None
        for i in range(5):
            try:
                req=urllib.request.Request(url,headers={'User-Agent':UA})
                data=urllib.request.urlopen(req,timeout=25).read()
                zipfile.ZipFile(io.BytesIO(data))
                zp.write_bytes(data); last=None; break
            except urllib.error.HTTPError as e:
                if e.code==404:
                    miss.write_text('404'); return {}
                last=e
            except Exception as e:
                last=e
            time.sleep(.5*(i+1))
        if last is not None: raise last
    z=zipfile.ZipFile(zp); out={}
    with z.open(z.namelist()[0]) as f:
        for r in csv.reader(io.TextIOWrapper(f)):
            if not r or not r[0].isdigit() or len(r)<5: continue
            t=int(r[0]); t=t//1000 if t>10**15 else t
            out[t]=(float(r[1]),float(r[4]))
    return out

def mean(xs): return sum(xs)/len(xs) if xs else 0.0
def sm(rows):
    return {
      'n':len(rows),
      'mean_r1':mean([x['r1'] for x in rows]),
      'mean_r4':mean([x['r4'] for x in rows]),
      'mean_r12':mean([x['r12'] for x in rows]),
      'win_r12':mean([1.0 if x['r12']>0 else 0.0 for x in rows]),
      'long':sum(1 for x in rows if x['side']=='LONG'),
      'short':sum(1 for x in rows if x['side']=='SHORT')
    }

out=[]; errors=[]
for i,e in enumerate(EVENTS,1):
    pub=datetime.datetime.fromisoformat(e['published_at'].replace('Z','+00:00'))
    pubms=int(pub.timestamp()*1000)
    entry=((pubms//3600000)+1)*3600000
    edt=datetime.datetime.fromtimestamp(entry/1000,UTC)
    endt=datetime.datetime.fromtimestamp((entry+11*3600000)/1000,UTC)
    if endt.year!=pub.year:
        errors.append({**e,'reason':'12h_endpoint_crosses_signal_year'}); continue
    data={}
    data.update(bars(e['symbol'],month(edt)))
    if (edt.year,edt.month)!=(endt.year,endt.month):
        data.update(bars(e['symbol'],month(endt)))
    if entry not in data:
        errors.append({**e,'reason':'missing_entry_bar','entry_time':entry}); continue
    px=data[entry][0]
    vals={}; ok=True
    for h in (1,4,12):
        t=entry+(h-1)*3600000
        if t not in data:
            errors.append({**e,'reason':f'missing_{h}h_bar','entry_time':entry}); ok=False; break
        side=1.0 if e['side']=='LONG' else -1.0
        vals[h]=side*math.log(data[t][1]/px)
    if not ok: continue
    out.append({**e,'entry_time':entry,'entry_price':px,'r1':vals[1],'r4':vals[4],'r12':vals[12]})
    if i%25==0: print('PROGRESS',i,'/',len(EVENTS),flush=True)

by_symbol=defaultdict(list); by_year=defaultdict(list); by_side=defaultdict(list); by_repo=defaultdict(list)
for x in out:
    by_symbol[x['asset']].append(x)
    by_year[x['published_at'][:4]].append(x)
    by_side[x['side']].append(x)
    by_repo[x['repo']].append(x)

overall=sm(out)
sym_stats={k:sm(v) for k,v in sorted(by_symbol.items())}
year_stats={k:sm(v) for k,v in sorted(by_year.items())}
side_stats={k:sm(v) for k,v in sorted(by_side.items())}
repo_stats={k:sm(v) for k,v in sorted(by_repo.items())}
positive=sum(1 for v in sym_stats.values() if v['mean_r12']>0)
gate=(len(out)>=80 and len(sym_stats)>=8 and overall['mean_r12']>=0.0025 and
      positive/max(1,len(sym_stats))>=0.60 and
      year_stats.get('2023',{}).get('mean_r12',0)>0 and
      year_stats.get('2024',{}).get('mean_r12',0)>0)

summary={
  'version':'v162',
  'status':'promote_oos' if gate else 'frozen_failed_discovery',
  'gate_pass':gate,
  'events':len(out),
  'symbols':len(sym_stats),
  'positive_symbols':f'{positive}/{len(sym_stats)}',
  'positive_symbol_fraction':positive/max(1,len(sym_stats)),
  'overall':overall,
  'by_year':year_stats,
  'by_side':side_stats,
  'by_symbol':sym_stats,
  'by_repo':repo_stats,
  'replay_errors':errors,
  'oos_2025_plus_evaluated':False,
  'strict_engine_run':False
}
(ROOT/'results/events.json').write_text(json.dumps(out,ensure_ascii=False,indent=2))
(ROOT/'results/replay_errors.json').write_text(json.dumps(errors,ensure_ascii=False,indent=2))
(ROOT/'results/summary.json').write_text(json.dumps(summary,ensure_ascii=False,indent=2))
print(json.dumps({
 'events':len(out),'symbols':len(sym_stats),'positive_symbols':summary['positive_symbols'],
 'mean_r1':overall['mean_r1'],'mean_r4':overall['mean_r4'],'mean_r12':overall['mean_r12'],
 '2023_r12':year_stats.get('2023',{}).get('mean_r12',0),
 '2024_r12':year_stats.get('2024',{}).get('mean_r12',0),
 'long_r12':side_stats.get('LONG',{}).get('mean_r12',0),
 'short_r12':side_stats.get('SHORT',{}).get('mean_r12',0),
 'gate_pass':gate,'errors':len(errors)
},indent=2))
