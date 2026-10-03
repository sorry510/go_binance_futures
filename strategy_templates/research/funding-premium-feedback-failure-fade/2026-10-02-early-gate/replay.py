import csv,datetime,io,json,math,zipfile
from pathlib import Path

UTC=datetime.timezone.utc
ROOT=Path(__file__).resolve().parent
SYMS=['SOLUSDT','DOGEUSDT','LTCUSDT','AVAXUSDT','UNIUSDT','ZECUSDT']
HOUR=3600000
START=int(datetime.datetime(2023,1,1,tzinfo=UTC).timestamp()*1000)
END=int(datetime.datetime(2025,1,1,tzinfo=UTC).timestamp()*1000)
FUND_CACHE=Path('/tmp/v152-funding')
PREM_CACHE=Path('/tmp/v148_premium_zero_cross')

def months():
    out=[]
    d=datetime.date(2023,1,1)
    end=datetime.date(2024,12,1)
    while d<=end:
        out.append(d.strftime('%Y-%m'))
        d=(d.replace(day=28)+datetime.timedelta(days=4)).replace(day=1)
    return out

def load_funding(sym):
    out=[]
    for mo in months():
        p=FUND_CACHE/f'{sym}-{mo}.zip'
        if not p.exists() or p.stat().st_size==0:
            raise RuntimeError(f'missing funding cache {p}')
        with zipfile.ZipFile(p) as z:
            with z.open(z.namelist()[0]) as fh:
                for r in csv.DictReader(io.TextIOWrapper(fh)):
                    try:
                        out.append((int(r['calc_time']),int(r['funding_interval_hours']),float(r['last_funding_rate'])))
                    except Exception:
                        pass
    out.sort()
    return out
def load_hourly(sym,kind):
    out={}
    for mo in months():
        p=PREM_CACHE/f'{kind}-{sym}-1h-{mo}.zip'
        if not p.exists() or p.stat().st_size==0:
            raise RuntimeError(f'missing cache {p}')
        with zipfile.ZipFile(p) as z:
            with z.open(z.namelist()[0]) as fh:
                for r in csv.reader(io.TextIOWrapper(fh)):
                    if not r or not r[0].isdigit() or len(r)<5: continue
                    t=int(r[0]); t=t//1000 if t>10**15 else t
                    if kind=='premiumIndexKlines':
                        out[t]=float(r[4])
                    else:
                        out[t]=(float(r[1]),float(r[4]))
    return out

def mean(xs): return sum(xs)/len(xs) if xs else 0.0

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
    funding=load_funding(sym)
    premium=load_hourly(sym,'premiumIndexKlines')
    price=load_hourly(sym,'klines')
    states=[]
    for t,interval,rate in funding:
        if interval!=8 or t<START or t>=END: continue
        prem_t=(t//HOUR)*HOUR
        if prem_t not in premium: continue
        pclose=premium[prem_t]
        if rate==0 or pclose==0:
            state=False
        else:
            state=(rate>0 and pclose>0) or (rate<0 and pclose<0)
        states.append((t,rate,pclose,state))
    n=0
    for i in range(1,len(states)):
        pt,pr,pp,ps=states[i-1]
        t,rate,pclose,state=states[i]
        if not (7.5*HOUR <= t-pt <= 8.5*HOUR): continue
        if ps or not state: continue
        side=-1 if pclose>0 else 1
        entry_t=(t//HOUR+1)*HOUR
        if entry_t not in price: continue
        year=datetime.datetime.fromtimestamp(t/1000,UTC).year
        end_t=entry_t+11*HOUR
        if datetime.datetime.fromtimestamp(end_t/1000,UTC).year!=year: continue
        needed=[entry_t,entry_t+3*HOUR,end_t]
        if any(x not in price for x in needed): continue
        entry=price[entry_t][0]
        if entry<=0: continue
        events.append({
          'symbol':sym,'signal_time':t,'year':year,
          'side':'LONG' if side>0 else 'SHORT',
          'funding_rate':rate,'post1h_premium_close':pclose,
          'prev_funding_rate':pr,'prev_post1h_premium_close':pp,
          'r1':side*math.log(price[entry_t][1]/entry),
          'r4':side*math.log(price[entry_t+3*HOUR][1]/entry),
          'r12':side*math.log(price[end_t][1]/entry)
        })
        n+=1
    meta[sym]={'funding_rows':len(funding),'states':len(states),'events':n}
    print('SYMBOL',sym,'events',n,'states',len(states),flush=True)
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
  'version':'v155',
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
 'events':len(events),'mean1':disc.get('mean1',0),'mean4':disc.get('mean4',0),
 'mean12':disc.get('mean12',0),'positive_symbols':disc.get('positive_symbols12',0),
 'frequency_per_symbol_week':freq,'2023_mean12':y23.get('mean12',0),
 '2024_mean12':y24.get('mean12',0),'long_n':longs.get('n',0),
 'long_mean12':longs.get('mean12',0),'short_n':shorts.get('n',0),
 'short_mean12':shorts.get('mean12',0),'gate_pass':gate
},indent=2),flush=True)
