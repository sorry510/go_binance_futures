import bisect, csv, datetime, io, json, math, subprocess, zipfile
from concurrent.futures import ThreadPoolExecutor, as_completed
from pathlib import Path

UTC=datetime.timezone.utc
ROOT=Path(__file__).resolve().parent
SYMS=['SOLUSDT','DOGEUSDT','LTCUSDT','AVAXUSDT','UNIUSDT','ZECUSDT']
HOUR=3600000
DISC_START=int(datetime.datetime(2023,1,1,tzinfo=UTC).timestamp()*1000)
DISC_END=int(datetime.datetime(2025,1,1,tzinfo=UTC).timestamp()*1000)
KLINE_CACHE=Path('/tmp/v149_spot_perp_qv')
METRIC_CACHE=Path('/tmp/v147-binance-metrics')
FUND_CACHE=Path('/tmp/v152-funding')
FUND_CACHE.mkdir(exist_ok=True)

def months():
    d=datetime.date(2022,12,1); end=datetime.date(2024,12,1); out=[]
    while d<=end:
        out.append(d.strftime('%Y-%m'))
        d=(d.replace(day=28)+datetime.timedelta(days=4)).replace(day=1)
    return out
def fetch_funding(sym,mo):
    p=FUND_CACHE/f'{sym}-{mo}.zip'
    if p.exists():
        return p
    u=f'https://data.binance.vision/data/futures/um/monthly/fundingRate/{sym}/{sym}-fundingRate-{mo}.zip'
    tmp=str(p)+'.part'
    q=subprocess.run([
        'curl','--http1.1','-L','--retry','4','--retry-all-errors',
        '--connect-timeout','8','--max-time','30','-sS','-w','%{http_code}',
        '-o',tmp,u
    ],capture_output=True,text=True)
    code=(q.stdout or '')[-3:]
    if q.returncode==0 and code=='200':
        Path(tmp).replace(p)
    elif code=='404':
        Path(tmp).unlink(missing_ok=True); p.write_bytes(b'')
    else:
        Path(tmp).unlink(missing_ok=True)
        raise RuntimeError((sym,mo,q.returncode,code))
    return p

def ensure_funding():
    jobs=[(s,m) for s in SYMS for m in months()]
    with ThreadPoolExecutor(max_workers=12) as ex:
        fs=[ex.submit(fetch_funding,*j) for j in jobs]
        for i,f in enumerate(as_completed(fs),1):
            f.result()
            if i%30==0 or i==len(fs):
                print('FUND_DOWNLOAD',i,'/',len(fs),flush=True)
def load_klines(sym):
    out={}
    for mo in months():
        p=KLINE_CACHE/f'perp-{sym}-{mo}.zip'
        if not p.exists() or p.stat().st_size==0:
            raise RuntimeError(f'missing cached perp kline {sym} {mo}')
        z=zipfile.ZipFile(p)
        with z.open(z.namelist()[0]) as fh:
            for r in csv.reader(io.TextIOWrapper(fh)):
                if not r or not r[0].isdigit():
                    continue
                t=int(r[0]); t=t//1000 if t>10**15 else t
                out[t]=(float(r[1]),float(r[4]),float(r[7]))
    return out

def load_funding(sym):
    out=[]
    for mo in months():
        p=FUND_CACHE/f'{sym}-{mo}.zip'
        if not p.exists() or p.stat().st_size==0:
            continue
        z=zipfile.ZipFile(p)
        with z.open(z.namelist()[0]) as fh:
            cr=csv.DictReader(io.TextIOWrapper(fh))
            for r in cr:
                try:
                    out.append((int(r['calc_time']),int(r['funding_interval_hours']),float(r['last_funding_rate'])))
                except Exception:
                    pass
    out.sort()
    return out
def load_metrics(sym):
    pts=[]
    d=datetime.date(2022,12,28); end=datetime.date(2024,12,31)
    while d<=end:
        p=METRIC_CACHE/f'{sym}-{d.isoformat()}.zip'
        if not p.exists():
            d+=datetime.timedelta(days=1); continue
        z=zipfile.ZipFile(p)
        with z.open(z.namelist()[0]) as fh:
            cr=csv.DictReader(io.TextIOWrapper(fh))
            for r in cr:
                try:
                    t=int(datetime.datetime.strptime(r['create_time'],'%Y-%m-%d %H:%M:%S').replace(tzinfo=UTC).timestamp()*1000)
                    oi=float(r['sum_open_interest_value'])
                    if oi>0: pts.append((t,oi))
                except Exception:
                    pass
        d+=datetime.timedelta(days=1)
    pts.sort()
    return pts

def mean(xs):
    return sum(xs)/len(xs) if xs else 0.0
def summarize(es):
    if not es: return {'n':0}
    sy=sorted(set(x['symbol'] for x in es)); by={}; pos=0
    for s in sy:
        q=[x for x in es if x['symbol']==s]
        m=mean([x['r12'] for x in q])
        by[s]={
            'n':len(q),'mean1':mean([x['r1'] for x in q]),
            'mean4':mean([x['r4'] for x in q]),'mean12':m,
            'win12':mean([1.0 if x['r12']>0 else 0.0 for x in q])
        }
        if m>0: pos+=1
    return {
        'n':len(es),'symbols':len(sy),
        'mean1':mean([x['r1'] for x in es]),
        'mean4':mean([x['r4'] for x in es]),
        'mean12':mean([x['r12'] for x in es]),
        'win12':mean([1.0 if x['r12']>0 else 0.0 for x in es]),
        'positive_symbols12':pos,'by_symbol':by
    }

def latest_oi(points,times,t):
    i=bisect.bisect_right(times,t)-1
    return points[i][1] if i>=0 else None
def oi_rows(funding, metrics):
    rows=[]
    metric_times=[x[0] for x in metrics]
    for t,interval,rate in funding:
        oi=latest_oi(metrics,metric_times,t) if interval==8 else None
        rows.append({'t':t,'interval':interval,'rate':rate,'oi':oi})
    return rows

def normal_chain(rows,i):
    if i<2: return False
    q=rows[i-2:i+1]
    if any(x['interval']!=8 or x['oi'] is None or x['oi']<=0 for x in q):
        return False
    for a,b in zip(q,q[1:]):
        gap=b['t']-a['t']
        if not (7.5*HOUR <= gap <= 8.5*HOUR):
            return False
    return True

ensure_funding()
events=[]
meta={}
for sym in SYMS:
    bars=load_klines(sym)
    funding=load_funding(sym)
    metrics=load_metrics(sym)
    rows=oi_rows(funding,metrics)
    n=0
    for i in range(2,len(rows)):
        if not normal_chain(rows,i):
            continue
        r0,r1,cur=rows[i-2],rows[i-1],rows[i]
        if cur['t']<DISC_START or cur['t']>=DISC_END:
            continue
        prev_growth=math.log(r1['oi']/r0['oi'])
        cur_growth=math.log(cur['oi']/r1['oi'])
        if not (prev_growth<=0 and cur_growth>0):
            continue
        if cur['rate']==0:
            continue
        direction=-1 if cur['rate']>0 else 1
        entry_t=(cur['t']//HOUR+1)*HOUR
        if entry_t not in bars:
            continue
        signal_year=datetime.datetime.fromtimestamp(cur['t']/1000,UTC).year
        end_close=entry_t+12*HOUR
        if datetime.datetime.fromtimestamp(end_close/1000,UTC).year!=signal_year:
            continue
        needed=[entry_t+k*HOUR for k in range(12)]
        if any(x not in bars for x in needed):
            continue
        entry=bars[entry_t][0]
        if entry<=0:
            continue
        e={
            'symbol':sym,'signal_time':cur['t'],'year':signal_year,
            'direction':'LONG' if direction>0 else 'SHORT',
            'funding_rate':cur['rate'],'oi_notional':cur['oi'],
            'oi_growth_prev':prev_growth,'oi_growth_now':cur_growth,
            'r1':direction*math.log(bars[entry_t][1]/entry),
            'r4':direction*math.log(bars[entry_t+3*HOUR][1]/entry),
            'r12':direction*math.log(bars[entry_t+11*HOUR][1]/entry)
        }
        events.append(e); n+=1
    meta[sym]={
        'funding_rows':len(funding),'metric_rows':len(metrics),
        'kline_rows':len(bars),'events':n,
        'valid_oi_rows':sum(1 for x in rows if x['oi'] is not None)
    }
    print('SYMBOL',sym,'events',n,'fund',len(funding),'metrics',len(metrics),flush=True)
disc=summarize(events)
y23=summarize([x for x in events if x['year']==2023])
y24=summarize([x for x in events if x['year']==2024])
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
    'version':'v153',
    'status':'promote_oos' if gate else 'frozen_failed_early_gate',
    'gate_pass':gate,
    'discovery_2023_2024':disc,
    '2023':y23,'2024':y24,
    'frequency_per_symbol_week':freq,
    'source_meta':meta,
    'oos_evaluated':False,
    'strict_engine_run':False
}
(ROOT/'results'/'events.json').write_text(json.dumps(events,indent=2))
(ROOT/'results'/'summary.json').write_text(json.dumps(summary,indent=2))
print('SUMMARY',json.dumps(summary,indent=2),flush=True)
