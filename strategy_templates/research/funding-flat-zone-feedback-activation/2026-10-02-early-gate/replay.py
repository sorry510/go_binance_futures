import csv, datetime, io, json, math, zipfile
from pathlib import Path

UTC=datetime.timezone.utc
ROOT=Path(__file__).resolve().parent
SYMS=['SOLUSDT','DOGEUSDT','LTCUSDT','AVAXUSDT','UNIUSDT','ZECUSDT']
HOUR=3600000
START=int(datetime.datetime(2023,1,1,tzinfo=UTC).timestamp()*1000)
END=int(datetime.datetime(2025,1,1,tzinfo=UTC).timestamp()*1000)
FLAT=0.0001
FUND_CACHE=Path('/tmp/v152-funding')
KLINE_CACHE=Path('/tmp/v149_spot_perp_qv')

def months():
    d=datetime.date(2022,12,1); end=datetime.date(2024,12,1); out=[]
    while d<=end:
        out.append(d.strftime('%Y-%m'))
        d=(d.replace(day=28)+datetime.timedelta(days=4)).replace(day=1)
    return out

def load_funding(sym):
    out=[]
    for mo in months():
        p=FUND_CACHE/f'{sym}-{mo}.zip'
        if not p.exists() or p.stat().st_size==0: continue
        with zipfile.ZipFile(p) as z:
            with z.open(z.namelist()[0]) as fh:
                for r in csv.DictReader(io.TextIOWrapper(fh)):
                    try:
                        out.append((int(r['calc_time']),int(r['funding_interval_hours']),float(r['last_funding_rate'])))
                    except Exception:
                        pass
    out.sort()
    return out

def load_bars(sym):
    out={}
    for mo in months():
        p=KLINE_CACHE/f'perp-{sym}-{mo}.zip'
        if not p.exists() or p.stat().st_size==0:
            raise RuntimeError(f'missing cached perp kline {sym} {mo}')
        with zipfile.ZipFile(p) as z:
            with z.open(z.namelist()[0]) as fh:
                for r in csv.reader(io.TextIOWrapper(fh)):
                    if not r or not r[0].isdigit() or len(r)<8: continue
                    t=int(r[0]); t=t//1000 if t>10**15 else t
                    out[t]=(float(r[1]),float(r[4]))
    return out

def mean(xs):
    return sum(xs)/len(xs) if xs else 0.0
def summarize(rows):
    if not rows: return {'n':0}
    syms=sorted(set(x['symbol'] for x in rows))
    by={}; pos=0
    for s in syms:
        q=[x for x in rows if x['symbol']==s]
        m12=mean([x['r12'] for x in q])
        by[s]={
            'n':len(q),'mean1':mean([x['r1'] for x in q]),
            'mean4':mean([x['r4'] for x in q]),'mean12':m12,
            'win12':mean([1.0 if x['r12']>0 else 0.0 for x in q])
        }
        if m12>0: pos+=1
    return {
        'n':len(rows),'symbols':len(syms),
        'mean1':mean([x['r1'] for x in rows]),
        'mean4':mean([x['r4'] for x in rows]),
        'mean12':mean([x['r12'] for x in rows]),
        'win12':mean([1.0 if x['r12']>0 else 0.0 for x in rows]),
        'positive_symbols12':pos,'by_symbol':by
    }

events=[]; meta={}
for sym in SYMS:
    funding=load_funding(sym); bars=load_bars(sym); n=0; flat_rows=0
    for i in range(1,len(funding)):
        pt,pi,pr=funding[i-1]
        t,interval,rate=funding[i]
        if pi!=8 or interval!=8: continue
        gap=t-pt
        if not (7.5*HOUR <= gap <= 8.5*HOUR): continue
        if abs(pr-FLAT) <= 1e-15: flat_rows+=1
        if t<START or t>=END: continue
        if abs(pr-FLAT)>1e-15 or abs(rate-FLAT)<=1e-15: continue
        direction=-1 if rate>FLAT else 1
        entry_t=(t//HOUR+1)*HOUR
        if entry_t not in bars: continue
        signal_year=datetime.datetime.fromtimestamp(t/1000,UTC).year
        if datetime.datetime.fromtimestamp((entry_t+12*HOUR)/1000,UTC).year!=signal_year:
            continue
        needed=[entry_t+k*HOUR for k in range(12)]
        if any(x not in bars for x in needed): continue
        entry=bars[entry_t][0]
        if entry<=0: continue
        e={
            'symbol':sym,'signal_time':t,'year':signal_year,
            'side':'LONG' if direction>0 else 'SHORT',
            'prev_funding':pr,'current_funding':rate,
            'r1':direction*math.log(bars[entry_t][1]/entry),
            'r4':direction*math.log(bars[entry_t+3*HOUR][1]/entry),
            'r12':direction*math.log(bars[entry_t+11*HOUR][1]/entry)
        }
        events.append(e); n+=1
    meta[sym]={'funding_rows':len(funding),'flat_prev_rows':flat_rows,'events':n}
    print('SYMBOL',sym,'events',n,'funding',len(funding),'flat_prev_rows',flat_rows,flush=True)
disc=summarize(events)
y23=summarize([x for x in events if x['year']==2023])
y24=summarize([x for x in events if x['year']==2024])
longs=summarize([x for x in events if x['side']=='LONG'])
shorts=summarize([x for x in events if x['side']=='SHORT'])
weeks=((datetime.date(2025,1,1)-datetime.date(2023,1,1)).days/7.0)*len(SYMS)
freq=len(events)/weeks

gate=(
    disc.get('mean12',0)>=0.002 and
    disc.get('positive_symbols12',0)>=4 and
    freq>=0.30 and
    y23.get('mean12',0)>0 and
    y24.get('mean12',0)>0
)
summary={
    'version':'v154',
    'status':'promote_oos' if gate else 'frozen_failed_early_gate',
    'gate_pass':gate,
    'discovery_2023_2024':disc,
    '2023':y23,'2024':y24,
    'long':longs,'short':shorts,
    'frequency_per_symbol_week':freq,
    'source_meta':meta,
    'oos_evaluated':False,
    'strict_engine_run':False
}
(ROOT/'results'/'events.json').write_text(json.dumps(events,indent=2))
(ROOT/'results'/'summary.json').write_text(json.dumps(summary,indent=2))
print('SUMMARY',json.dumps({
    'events':len(events),
    'mean1':disc.get('mean1',0),
    'mean4':disc.get('mean4',0),
    'mean12':disc.get('mean12',0),
    'positive_symbols':disc.get('positive_symbols12',0),
    'frequency_per_symbol_week':freq,
    '2023_mean12':y23.get('mean12',0),
    '2024_mean12':y24.get('mean12',0),
    'long_n':longs.get('n',0),'long_mean12':longs.get('mean12',0),
    'short_n':shorts.get('n',0),'short_mean12':shorts.get('mean12',0),
    'gate_pass':gate
},indent=2),flush=True)
