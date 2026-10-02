import csv, io, zipfile, urllib.request, urllib.error, time, datetime, json
from pathlib import Path

UTC=datetime.timezone.utc
UA='Mozilla/5.0'
ROOT=Path('strategy_templates/research/binance-unplanned-transfer-suspension-short/2023-2024-feasibility')
EV=json.loads((ROOT/'inputs/events_raw.json').read_text())
VISION='https://data.binance.vision/data/futures/um/monthly/klines'
CACHE=Path('/tmp/transfer_suspension_coverage_cache')
CACHE.mkdir(exist_ok=True)

def mon(ms):
    return datetime.datetime.fromtimestamp(ms/1000,UTC).strftime('%Y-%m')

def prevm(ms):
    d=datetime.datetime.fromtimestamp(ms/1000,UTC)
    return f'{d.year-1}-12' if d.month==1 else f'{d.year:04d}-{d.month-1:02d}'

def sub2(ms):
    d=datetime.datetime.fromtimestamp(ms/1000,UTC)
    try: d=d.replace(year=d.year-2)
    except ValueError: d=d.replace(year=d.year-2,day=28)
    return int(d.timestamp()*1000)
def fetch(sym,mo):
    p=CACHE/f'{sym}-{mo}.zip'
    if p.exists(): return p
    u=f'{VISION}/{sym}/1h/{sym}-1h-{mo}.zip'
    try:
        req=urllib.request.Request(u,headers={'User-Agent':UA})
        p.write_bytes(urllib.request.urlopen(req,timeout=20).read())
    except urllib.error.HTTPError as e:
        if e.code==404: p.write_bytes(b'')
        else: raise
    return p

def rows(sym,mo):
    p=fetch(sym,mo)
    if not p.exists() or p.stat().st_size==0: return []
    z=zipfile.ZipFile(p)
    out=[]
    for r in csv.reader(io.TextIOWrapper(z.open(z.namelist()[0]))):
        if not r or not r[0].isdigit(): continue
        t=int(r[0]); t=t//1000 if t>10**15 else t
        out.append((t,float(r[7]) if len(r)>7 else 0.0))
    return out
eligible=[]
excluded=[]
for e in EV:
    sym=e['symbol']; evt=e['release_time']; cut=sub2(evt)
    hist=rows(sym,mon(cut))
    if not hist or hist[0][0]>cut:
        excluded.append({**e,'reason':'history_lt_2y_or_missing'})
        continue
    hour=(evt//3600000)*3600000
    prior=[r for r in rows(sym,prevm(evt))+rows(sym,mon(evt)) if hour-86400000<=r[0]<=hour-1]
    if len(prior)<24:
        excluded.append({**e,'reason':'missing_24h_bars','prior_bars':len(prior)})
        continue
    qv=sum(r[1] for r in prior)
    if qv<5_000_000:
        excluded.append({**e,'reason':'quote_volume_lt_5m','quote_volume_24h':qv})
        continue
    eligible.append({**e,'quote_volume_24h':qv})

summary={
    'raw_events':len(EV),
    'raw_unique_tokens':len({e['ticker'] for e in EV}),
    'eligible_events':len(eligible),
    'eligible_unique_tokens':len({e['ticker'] for e in eligible}),
    'coverage_gate_pass':len(eligible)>=8 and len({e['ticker'] for e in eligible})>=8,
}
reasons={}
for x in excluded:
    reasons[x['reason']]=reasons.get(x['reason'],0)+1
summary['excluded_by_reason']=reasons
summary['post_event_returns_read']=False
(ROOT/'results/eligible.json').write_text(json.dumps(eligible,ensure_ascii=False,indent=2))
(ROOT/'results/excluded.json').write_text(json.dumps(excluded,ensure_ascii=False,indent=2))
(ROOT/'results/summary.json').write_text(json.dumps(summary,ensure_ascii=False,indent=2))
print(json.dumps(summary,ensure_ascii=False,indent=2))
for x in eligible:
    print('ELIGIBLE',x['ticker'],x['quote_volume_24h'])
for x in excluded:
    print('EXCLUDED',x['ticker'],x['reason'],x.get('quote_volume_24h',''))
