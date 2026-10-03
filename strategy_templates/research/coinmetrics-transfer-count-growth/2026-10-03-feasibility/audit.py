import json,urllib.request,urllib.parse,time,datetime
from collections import defaultdict
from pathlib import Path

ROOT=Path("strategy_templates/research/coinmetrics-transfer-count-growth/2026-10-03-feasibility")
ASSETS=['btc','eth','bnb','xrp','ada','doge','ltc','link','bch','etc','trx','xlm','dot','uni','aave','zec','eos','mana','algo','xtz','snx','comp','mkr']
SYMBOL={a:a.upper()+'USDT' for a in ASSETS}
UA={'User-Agent':'Mozilla/5.0'}

def get_json(u):
    return json.loads(urllib.request.urlopen(urllib.request.Request(u,headers=UA),timeout=30).read())

# Current USD-M trading set.
ex=get_json('https://fapi.binance.com/fapi/v1/exchangeInfo')
trading={x['symbol'] for x in ex.get('symbols',[]) if x.get('contractType')=='PERPETUAL' and x.get('status')=='TRADING' and x.get('quoteAsset')=='USDT'}

# Earliest same-name USD-M daily kline.
earliest={}
for a in ASSETS:
    s=SYMBOL[a]
    if s not in trading:
        earliest[a]=None
        continue
    try:
        o=get_json(f'https://fapi.binance.com/fapi/v1/klines?symbol={s}&interval=1d&startTime=0&limit=1')
        earliest[a]=int(o[0][0]) if isinstance(o,list) and o else None
    except Exception:
        earliest[a]=None
    time.sleep(.03)

# Fetch Community TxTfrCnt history in one paginated request.
params={
 'assets':','.join(ASSETS),'metrics':'TxTfrCnt',
 'start_time':'2023-01-01','end_time':'2024-12-31',
 'frequency':'1d','page_size':10000
}
u='https://community-api.coinmetrics.io/v4/timeseries/asset-metrics?'+urllib.parse.urlencode(params)
rows=[]
while u:
    o=get_json(u); rows.extend(o.get('data',[]))
    u=o.get('next_page_url')
    if u and u.startswith('https://api.coinmetrics.io'):
        u=u.replace('https://api.coinmetrics.io','https://community-api.coinmetrics.io',1)
    if u: time.sleep(.3)
(ROOT/'inputs/coinmetrics_rows.json').write_text(json.dumps(rows))

by=defaultdict(list)
for r in rows:
    try:v=float(r['TxTfrCnt'])
    except:continue
    if v>0: by[r['asset']].append((r['time'][:10],v))

audit=[]
for a in ASSETS:
    vals=by.get(a,[])
    first=vals[0][0] if vals else None; last=vals[-1][0] if vals else None
    e=earliest[a]
    current=symbol=SYMBOL[a] in trading
    # Count data days whose UTC date is at least 730d after USD-M first daily bar.
    eligible_days=0
    if e:
        ed=datetime.datetime.fromtimestamp(e/1000,datetime.timezone.utc).date()
        for ds,_ in vals:
            day=datetime.date.fromisoformat(ds)
            if (day-ed).days>=730: eligible_days+=1
    audit.append({
      'asset':a,'symbol':SYMBOL[a],'current_trading_usdm':SYMBOL[a] in trading,
      'earliest_usdm_ms':e,'tx_tfr_nonzero_days_2023_2024':len(vals),
      'tx_tfr_first':first,'tx_tfr_last':last,'days_meeting_730d_history':eligible_days
    })
passing=[x for x in audit if x['current_trading_usdm'] and x['tx_tfr_nonzero_days_2023_2024']>=600 and x['days_meeting_730d_history']>=180]
summary={
 'candidate_assets':len(ASSETS),'rows':len(rows),'passing_assets':len(passing),
 'passing_symbols':[x['symbol'] for x in passing],
 'coverage_gate_min_assets':8,'coverage_gate_pass':len(passing)>=8,
 'post_signal_returns_read':False
}
(ROOT/'inputs/coverage_audit.json').write_text(json.dumps(audit,indent=2))
(ROOT/'results/summary.json').write_text(json.dumps(summary,indent=2))
print(json.dumps(summary,indent=2))
for x in passing: print('PASS',x)
