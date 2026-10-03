import csv, io, zipfile, subprocess, datetime, json, math
from pathlib import Path

UTC=datetime.timezone.utc
ROOT=Path(__file__).resolve().parent
CACHE=Path('/tmp/v149_spot_perp_qv')
CACHE.mkdir(exist_ok=True)
SYMS=['SOLUSDT','DOGEUSDT','LTCUSDT','AVAXUSDT','UNIUSDT','ZECUSDT']
START_MONTH='2022-12'
END_MONTH='2024-12'
HOUR=3600000
DISC_START=int(datetime.datetime(2023,1,1,tzinfo=UTC).timestamp()*1000)
DISC_END=int(datetime.datetime(2025,1,1,tzinfo=UTC).timestamp()*1000)

def months(start_ym,end_ym):
    d=datetime.date.fromisoformat(start_ym+'-01')
    end=datetime.date.fromisoformat(end_ym+'-01')
    out=[]
    while d<=end:
        out.append(d.strftime('%Y-%m'))
        d=(d.replace(day=28)+datetime.timedelta(days=4)).replace(day=1)
    return out
def fetch(market,sym,mo):
    p=CACHE/f'{market}-{sym}-{mo}.zip'
    if p.exists():
        return p
    if market=='spot':
        url=f'https://data.binance.vision/data/spot/monthly/klines/{sym}/1h/{sym}-1h-{mo}.zip'
    else:
        url=f'https://data.binance.vision/data/futures/um/monthly/klines/{sym}/1h/{sym}-1h-{mo}.zip'
    tmp=str(p)+'.part'
    q=subprocess.run([
        'curl','--http1.1','-L','--retry','4','--retry-all-errors',
        '--connect-timeout','8','--max-time','40','-sS','-w','%{http_code}',
        '-o',tmp,url
    ],capture_output=True,text=True)
    code=(q.stdout or '')[-3:]
    if q.returncode==0 and code=='200':
        Path(tmp).replace(p)
    elif code=='404':
        Path(tmp).unlink(missing_ok=True)
        p.write_bytes(b'')
    else:
        Path(tmp).unlink(missing_ok=True)
        raise RuntimeError((market,sym,mo,q.returncode,code))
    return p
def load_market(market,sym):
    out={}
    missing=[]
    for mo in months(START_MONTH,END_MONTH):
        p=fetch(market,sym,mo)
        if not p.exists() or p.stat().st_size==0:
            missing.append(mo)
            continue
        z=zipfile.ZipFile(p)
        with z.open(z.namelist()[0]) as fh:
            for r in csv.reader(io.TextIOWrapper(fh)):
                if not r or not r[0].isdigit():
                    continue
                t=int(r[0])
                if t>10**15:
                    t//=1000
                try:
                    o=float(r[1]); c=float(r[4]); qv=float(r[7])
                except Exception:
                    continue
                if o>0 and c>0 and qv>=0:
                    out[t]=(o,c,qv)
    return out,missing

def mean(xs):
    return sum(xs)/len(xs) if xs else 0.0

def summarize(events):
    if not events:
        return {'n':0}
    syms=sorted(set(x['symbol'] for x in events))
    by={}
    pos=0
    for s in syms:
        q=[x for x in events if x['symbol']==s]
        m=mean([x['r12'] for x in q])
        by[s]={
            'n':len(q),
            'mean1':mean([x['r1'] for x in q]),
            'mean4':mean([x['r4'] for x in q]),
            'mean12':m,
            'win12':mean([1.0 if x['r12']>0 else 0.0 for x in q])
        }
        if m>0:
            pos+=1
    return {
        'n':len(events),
        'symbols':len(syms),
        'mean1':mean([x['r1'] for x in events]),
        'mean4':mean([x['r4'] for x in events]),
        'mean12':mean([x['r12'] for x in events]),
        'win12':mean([1.0 if x['r12']>0 else 0.0 for x in events]),
        'positive_symbols12':pos,
        'positive_symbol_fraction12':pos/len(syms),
        'by_symbol':by
    }

events=[]
meta={}
for sym in SYMS:
    spot,smiss=load_market('spot',sym)
    perp,fmiss=load_market('perp',sym)
    common=sorted(set(spot)&set(perp))
    common_set=set(common)
    n=0
    for t in common:
        signal_t=t+HOUR
        if signal_t<DISC_START or signal_t>=DISC_END:
            continue
        history=[t-k*HOUR for k in range(24,-1,-1)]
        if any(x not in common_set for x in history):
            continue
        cur_hours=[t-k*HOUR for k in range(23,-1,-1)]
        prev_hours=[t-k*HOUR for k in range(24,0,-1)]
        spot_cur=sum(spot[x][2] for x in cur_hours)
        perp_cur=sum(perp[x][2] for x in cur_hours)
        spot_prev=sum(spot[x][2] for x in prev_hours)
        perp_prev=sum(perp[x][2] for x in prev_hours)
        if min(spot_cur,perp_cur,spot_prev,perp_prev)<=0:
            continue
        ratio_prev=spot_prev/perp_prev
        ratio_now=spot_cur/perp_cur
        if not (ratio_prev<=1.0 and ratio_now>1.0):
            continue
        first=cur_hours[0]
        spot_ret=math.log(spot[t][1]/spot[first][0])
        if spot_ret==0:
            continue
        direction=1 if spot_ret>0 else -1
        entry_t=signal_t
        forward=[entry_t+k*HOUR for k in range(12)]
        if any(x not in perp for x in forward):
            continue
        year=datetime.datetime.fromtimestamp(signal_t/1000,UTC).year
        if datetime.datetime.fromtimestamp((entry_t+12*HOUR-1)/1000,UTC).year!=year:
            continue
        entry=perp[entry_t][0]
        if entry<=0:
            continue
        e={
            'symbol':sym,
            'signal_time':signal_t,
            'year':year,
            'direction':'LONG' if direction>0 else 'SHORT',
            'ratio_prev':ratio_prev,
            'ratio_now':ratio_now,
            'spot_return24':spot_ret,
            'r1':direction*math.log(perp[entry_t][1]/entry),
            'r4':direction*math.log(perp[entry_t+3*HOUR][1]/entry),
            'r12':direction*math.log(perp[entry_t+11*HOUR][1]/entry)
        }
        events.append(e); n+=1
    meta[sym]={
        'spot_rows':len(spot),'perp_rows':len(perp),
        'spot_missing_months':smiss,'perp_missing_months':fmiss,
        'events':n
    }
    print('SYMBOL',sym,'events',n,'spot',len(spot),'perp',len(perp),flush=True)
disc=summarize(events)
y23=summarize([x for x in events if x['year']==2023])
y24=summarize([x for x in events if x['year']==2024])
triggered=disc.get('symbols',0)
weeks=((datetime.date(2025,1,1)-datetime.date(2023,1,1)).days/7.0)*triggered if triggered else 0
freq=len(events)/weeks if weeks else 0
gate=(
    disc.get('mean12',0)>=0.002 and
    disc.get('positive_symbols12',0)>=4 and
    freq>=0.30 and
    y23.get('mean12',0)>0 and
    y24.get('mean12',0)>0
)
summary={
    'version':'v149',
    'status':'promote_oos' if gate else 'frozen_failed_early_gate',
    'gate_pass':gate,
    'discovery_2023_2024':disc,
    '2023':y23,
    '2024':y24,
    'frequency_per_symbol_week':freq,
    'source_meta':meta,
    'oos_evaluated':False,
    'strict_engine_run':False
}
(ROOT/'results'/'events.json').write_text(json.dumps(events,indent=2))
(ROOT/'results'/'summary.json').write_text(json.dumps(summary,indent=2))
print('SUMMARY',json.dumps(summary,indent=2),flush=True)
