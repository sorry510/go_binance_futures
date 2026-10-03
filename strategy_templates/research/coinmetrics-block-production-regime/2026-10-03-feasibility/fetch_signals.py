import json,urllib.request,urllib.parse,time,datetime
from pathlib import Path
from collections import defaultdict,Counter

ROOT=Path("strategy_templates/research/coinmetrics-block-production-regime/2026-10-03-feasibility")
ASSETS=["ada","algo","bch","btc","doge","etc","eth","ltc","trx","xlm","xrp","zec"]
SYMBOL={a:a.upper()+"USDT" for a in ASSETS}
BASE="https://community-api.coinmetrics.io/v4/timeseries/asset-metrics"

params={
 "assets":",".join(ASSETS),"metrics":"BlkCnt","frequency":"1d",
 "start_time":"2022-11-25","end_time":"2024-12-31","page_size":10000
}
url=BASE+"?"+urllib.parse.urlencode(params)
rows=[]
while url:
    req=urllib.request.Request(url,headers={"User-Agent":"Mozilla/5.0","Accept":"application/json"})
    o=json.loads(urllib.request.urlopen(req,timeout=30).read())
    rows.extend(o.get("data") or [])
    url=o.get("next_page_url")
    if url and url.startswith("https://api.coinmetrics.io"):
        url=url.replace("https://api.coinmetrics.io","https://community-api.coinmetrics.io",1)
    if url: time.sleep(.4)

(ROOT/"inputs").mkdir(exist_ok=True)
(ROOT/"results").mkdir(exist_ok=True)
(ROOT/"inputs/raw_blkcnt.json").write_text(json.dumps(rows,indent=2))

by=defaultdict(dict)
for r in rows:
    a=r.get("asset")
    try:v=float(r.get("BlkCnt"))
    except:continue
    if a in ASSETS and v>0:
        by[a][r["time"][:10]]=v

def d(s): return datetime.date.fromisoformat(s)
events=[]
coverage={}
for a in ASSETS:
    m=by[a]
    dates=sorted(m)
    coverage[a]={
      "rows":len(dates),
      "first":dates[0] if dates else None,
      "last":dates[-1] if dates else None,
      "rows_2023":sum(x.startswith("2023-") for x in dates),
      "rows_2024":sum(x.startswith("2024-") for x in dates),
    }
    scores={}
    for ds in dates:
        day=d(ds)
        prev=[(day-datetime.timedelta(days=k)).isoformat() for k in range(1,31)]
        if any(x not in m for x in prev): continue
        base=sum(m[x] for x in prev)/30
        if base<=0: continue
        import math
        scores[ds]=math.log(m[ds]/base)
    for ds in sorted(scores):
        day=d(ds)
        if not (datetime.date(2023,1,1)<=day<=datetime.date(2024,12,31)): continue
        pds=(day-datetime.timedelta(days=1)).isoformat()
        if pds not in scores: continue
        prev=scores[pds]; now=scores[ds]
        side=None
        if prev<=0<now: side="LONG"
        elif prev>=0>now: side="SHORT"
        if side:
            events.append({
              "asset":a,"symbol":SYMBOL[a],"signal_date":ds,
              "side":side,"blkcnt":m[ds],
              "baseline30":sum(m[(day-datetime.timedelta(days=k)).isoformat()] for k in range(1,31))/30,
              "score_prev":prev,"score_now":now
            })

summary={
 "candidate_assets":len(ASSETS),
 "coverage":coverage,
 "raw_signals":len(events),
 "raw_symbols":len({e["symbol"] for e in events}),
 "signals_by_symbol":dict(Counter(e["symbol"] for e in events)),
 "long":sum(e["side"]=="LONG" for e in events),
 "short":sum(e["side"]=="SHORT" for e in events),
 "post_signal_returns_read":False
}
(ROOT/"inputs/raw_signals.json").write_text(json.dumps(events,indent=2))
(ROOT/"results/source_summary.json").write_text(json.dumps(summary,indent=2))
print(json.dumps(summary,indent=2))
