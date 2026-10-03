import csv, datetime, io, json, math, subprocess, zipfile
from concurrent.futures import ThreadPoolExecutor, as_completed
from pathlib import Path

UTC=datetime.timezone.utc
ROOT=Path(__file__).resolve().parent
SYMS=['SOLUSDT','DOGEUSDT','LTCUSDT','AVAXUSDT','UNIUSDT','ZECUSDT']
HOUR=3600000
START=int(datetime.datetime(2026,1,1,tzinfo=UTC).timestamp()*1000)
END=int(datetime.datetime(2026,9,1,tzinfo=UTC).timestamp()*1000)
FLAT=0.0001
FUND=Path('/tmp/v154-oos2-funding'); FUND.mkdir(exist_ok=True)
KLINE=Path('/tmp/v154-oos2-kline'); KLINE.mkdir(exist_ok=True)

def months():
    d=datetime.date(2025,12,1); end=datetime.date(2026,8,1); out=[]
    while d<=end:
        out.append(d.strftime('%Y-%m'))
        d=(d.replace(day=28)+datetime.timedelta(days=4)).replace(day=1)
    return out

def fetch(sym,mo,kind):
    if kind=='funding':
        cache=FUND/f'{sym}-{mo}.zip'
        url=f'https://data.binance.vision/data/futures/um/monthly/fundingRate/{sym}/{sym}-fundingRate-{mo}.zip'
    else:
        cache=KLINE/f'{sym}-{mo}.zip'
        url=f'https://data.binance.vision/data/futures/um/monthly/klines/{sym}/1h/{sym}-1h-{mo}.zip'
    if cache.exists(): return cache
    tmp=str(cache)+'.part'
    q=subprocess.run([
        'curl','--http1.1','-L','--retry','4','--retry-all-errors',
        '--connect-timeout','8','--max-time','40','-sS','-w','%{http_code}',
        '-o',tmp,url
    ],capture_output=True,text=True)
    code=(q.stdout or '')[-3:]
    if q.returncode==0 and code=='200':
        Path(tmp).replace(cache); return cache
    Path(tmp).unlink(missing_ok=True)
    raise RuntimeError((sym,mo,kind,q.returncode,code))

def ensure():
    jobs=[(s,m,k) for s in SYMS for m in months() for k in ('funding','kline')]
    with ThreadPoolExecutor(max_workers=16) as ex:
        fs=[ex.submit(fetch,*j) for j in jobs]
        for i,f in enumerate(as_completed(fs),1):
            f.result()
            if i%30==0 or i==len(fs): print('DOWNLOAD',i,'/',len(fs),flush=True)
def load_funding(sym):
    out=[]
    for mo in months():
        with zipfile.ZipFile(FUND/f'{sym}-{mo}.zip') as z:
            with z.open(z.namelist()[0]) as fh:
                for r in csv.DictReader(io.TextIOWrapper(fh)):
                    try:
                        out.append((int(r['calc_time']),int(r['funding_interval_hours']),float(r['last_funding_rate'])))
                    except Exception: pass
    out.sort(); return out

def load_bars(sym):
    out={}
    for mo in months():
        with zipfile.ZipFile(KLINE/f'{sym}-{mo}.zip') as z:
            with z.open(z.namelist()[0]) as fh:
                for r in csv.reader(io.TextIOWrapper(fh)):
                    if not r or not r[0].isdigit() or len(r)<5: continue
                    t=int(r[0]); t=t//1000 if t>10**15 else t
                    out[t]=(float(r[1]),float(r[4]))
    return out

def mean(xs): return sum(xs)/len(xs) if xs else 0.0

def summarize(rows):
    if not rows: return {'n':0}
    syms=sorted(set(x['symbol'] for x in rows)); by={}; pos=0
    for s in syms:
        q=[x for x in rows if x['symbol']==s]; m=mean([x['r12'] for x in q])
        by[s]={'n':len(q),'mean1':mean([x['r1'] for x in q]),
               'mean4':mean([x['r4'] for x in q]),'mean12':m,
               'win12':mean([1.0 if x['r12']>0 else 0.0 for x in q])}
        if m>0: pos+=1
    return {'n':len(rows),'symbols':len(syms),
            'mean1':mean([x['r1'] for x in rows]),
            'mean4':mean([x['r4'] for x in rows]),
            'mean12':mean([x['r12'] for x in rows]),
            'win12':mean([1.0 if x['r12']>0 else 0.0 for x in rows]),
            'positive_symbols12':pos,'by_symbol':by}

ensure()
events=[]; meta={}
for sym in SYMS:
    funding=load_funding(sym); bars=load_bars(sym); n=0
    for i in range(1,len(funding)):
        pt,pi,pr=funding[i-1]; t,interval,rate=funding[i]
        if pi!=8 or interval!=8: continue
        if not (7.5*HOUR <= t-pt <= 8.5*HOUR): continue
        if t<START or t>=END: continue
        if abs(pr-FLAT)>1e-15 or abs(rate-FLAT)<=1e-15: continue
        direction=-1 if rate>FLAT else 1
        entry_t=(t//HOUR+1)*HOUR
        if entry_t not in bars: continue
        if datetime.datetime.fromtimestamp((entry_t+12*HOUR)/1000,UTC).year!=2026: continue
        needed=[entry_t+k*HOUR for k in range(12)]
        if any(x not in bars for x in needed): continue
        entry=bars[entry_t][0]
        if entry<=0: continue
        events.append({
            'symbol':sym,'signal_time':t,'side':'LONG' if direction>0 else 'SHORT',
            'prev_funding':pr,'current_funding':rate,
            'r1':direction*math.log(bars[entry_t][1]/entry),
            'r4':direction*math.log(bars[entry_t+3*HOUR][1]/entry),
            'r12':direction*math.log(bars[entry_t+11*HOUR][1]/entry)
        }); n+=1
    meta[sym]={'funding_rows':len(funding),'kline_rows':len(bars),'events':n}
    print('SYMBOL',sym,'events',n,flush=True)

allx=summarize(events)
longs=summarize([x for x in events if x['side']=='LONG'])
shorts=summarize([x for x in events if x['side']=='SHORT'])
weeks=((datetime.date(2026,9,1)-datetime.date(2026,1,1)).days/7.0)*len(SYMS)
freq=len(events)/weeks
gate=(allx.get('mean12',0)>=0.002 and allx.get('positive_symbols12',0)>=4 and freq>=0.30)
summary={
  'version':'v154','stage':'OOS2-2026-jan-aug',
  'status':'ready_exact_validation' if gate else 'frozen_failed_oos2',
  'gate_pass':gate,'oos_2026_jan_aug':allx,
  'long':longs,'short':shorts,
  'frequency_per_symbol_week':freq,
  'source_meta':meta,
  
  'strict_engine_run':False
}
(ROOT/'results/events.json').write_text(json.dumps(events,indent=2))
(ROOT/'results/summary.json').write_text(json.dumps(summary,indent=2))
print('SUMMARY',json.dumps({
  'events':len(events),'mean1':allx.get('mean1',0),'mean4':allx.get('mean4',0),
  'mean12':allx.get('mean12',0),'positive_symbols':allx.get('positive_symbols12',0),
  'frequency_per_symbol_week':freq,
  'long_n':longs.get('n',0),'long_mean12':longs.get('mean12',0),
  'short_n':shorts.get('n',0),'short_mean12':shorts.get('mean12',0),
  'gate_pass':gate
},indent=2),flush=True)
