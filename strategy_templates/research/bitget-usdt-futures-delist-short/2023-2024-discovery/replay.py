import json,datetime,math,csv,io,zipfile,subprocess
from pathlib import Path
from collections import defaultdict
from functools import lru_cache
ROOT=Path(__file__).resolve().parent
EV=[x for x in json.load(open(ROOT/'inputs'/'eligibility.json')) if x.get('eligible')]
CACHE=Path('/tmp/bitget_delist_binance_cache')
BASE='https://data.binance.vision/data/futures/um/monthly/klines'
H=3600000
def dtsec(x):return datetime.datetime.fromtimestamp(float(x),datetime.timezone.utc)
def month(d):return d.strftime('%Y-%m')
def nextmonth(d):
    x=(d.replace(day=28)+datetime.timedelta(days=4)).replace(day=1)
    return x.strftime('%Y-%m')
def fetch(sym,mo):
    p=CACHE/f'{sym}-{mo}.zip'
    if p.exists():return p
    u=f'{BASE}/{sym}/1h/{sym}-1h-{mo}.zip';tmp=str(p)+'.part'
    q=subprocess.run(['curl','--http1.1','-L','--retry','4','--retry-all-errors',
                      '--connect-timeout','8','--max-time','40','-sS','-w','%{http_code}',
                      '-o',tmp,u],capture_output=True,text=True)
    code=(q.stdout or '')[-3:]
    if q.returncode==0 and code=='200':Path(tmp).replace(p)
    elif code=='404':Path(tmp).unlink(missing_ok=True);p.write_bytes(b'')
    else:
        Path(tmp).unlink(missing_ok=True)
        raise RuntimeError(f'fetch {sym} {mo} rc={q.returncode} http={code}')
    return p
# ensure event and next month
for e in EV:
    d=dtsec(e['event_ts'])
    fetch(e['symbol'],month(d));fetch(e['symbol'],nextmonth(d))
@lru_cache(None)
def bars(sym,mo):
    p=CACHE/f'{sym}-{mo}.zip'
    if not p.exists() or p.stat().st_size==0:return {}
    try:z=zipfile.ZipFile(p)
    except:return {}
    out={}
    for r in csv.reader(io.TextIOWrapper(z.open(z.namelist()[0]))):
        if not r or not r[0].isdigit():continue
        t=int(r[0]);t=t//1000 if t>10**15 else t
        out[t]=(float(r[1]),float(r[4]))
    return out
events=[];skipped=[]
for e in EV:
    evt=int(float(e['event_ts'])*1000)
    ent=((evt//H)+1)*H
    d=dtsec(e['event_ts'])
    bm={}
    bm.update(bars(e['symbol'],month(d)));bm.update(bars(e['symbol'],nextmonth(d)))
    need=[ent,ent+3*H,ent+11*H]
    if any(t not in bm for t in need):
        skipped.append({**e,'reason':'missing_endpoint_bars','entry_ms':ent});continue
    op=bm[ent][0]
    if op<=0:
        skipped.append({**e,'reason':'bad_entry','entry_ms':ent});continue
    r1=-math.log(bm[ent][1]/op)
    r4=-math.log(bm[ent+3*H][1]/op)
    r12=-math.log(bm[ent+11*H][1]/op)
    events.append({**e,'entry_ms':ent,'entry_utc':datetime.datetime.fromtimestamp(ent/1000,datetime.timezone.utc).isoformat(),
                   'entry_open':op,'r1':r1,'r4':r4,'r12':r12})
def mean(a):return sum(a)/len(a) if a else 0.0
def summarize(es):
    bysym=defaultdict(list);bybatch=defaultdict(list);byyear=defaultdict(list)
    for x in es:
        bysym[x['symbol']].append(x['r12']);bybatch[x['article_url']].append(x['r12'])
        byyear[datetime.datetime.fromtimestamp(float(x['event_ts']),datetime.timezone.utc).year].append(x)
    sm={s:mean(v) for s,v in bysym.items()};bm={b:mean(v) for b,v in bybatch.items()}
    yrs={}
    for y,q in byyear.items():
        yrs[str(y)]={'n':len(q),'mean1':mean([x['r1'] for x in q]),'mean4':mean([x['r4'] for x in q]),
                      'mean12':mean([x['r12'] for x in q]),'win12':mean([1 if x['r12']>0 else 0 for x in q])}
    return {'n':len(es),'symbols':len(bysym),'batches':len(bybatch),
            'mean1':mean([x['r1'] for x in es]),'mean4':mean([x['r4'] for x in es]),'mean12':mean([x['r12'] for x in es]),
            'win12':mean([1 if x['r12']>0 else 0 for x in es]),
            'symbol_positive':sum(v>0 for v in sm.values()),'symbol_positive_fraction':mean([1 if v>0 else 0 for v in sm.values()]),
            'batch_equal_mean12':mean(list(bm.values())),'batch_positive':sum(v>0 for v in bm.values()),
            'batch_positive_fraction':mean([1 if v>0 else 0 for v in bm.values()]),
            'by_symbol':sm,'by_batch':bm,'by_year':yrs}
s=summarize(events)
s['gate_pass']=(s['n']>=8 and s['symbols']>=8 and s['batches']>=8 and
                s['mean12']>=0.005 and s['batch_equal_mean12']>=0.005 and
                s['symbol_positive_fraction']>=0.60 and s['batch_positive_fraction']>=0.60)
s['gate']='n>=8, symbols>=8, batches>=8, event mean12>=0.50%, batch-equal mean12>=0.50%, symbol breadth>=60%, batch breadth>=60%'
(ROOT/'results'/'events.json').write_text(json.dumps(events,ensure_ascii=False,indent=2))
(ROOT/'results'/'skipped.json').write_text(json.dumps(skipped,ensure_ascii=False,indent=2))
(ROOT/'results'/'summary.json').write_text(json.dumps(s,ensure_ascii=False,indent=2))
print(json.dumps(s,ensure_ascii=False,indent=2))
