import urllib.request, urllib.parse, json, time
from pathlib import Path

ROOT=Path("strategy_templates/research/coinmetrics-net-supply-growth/2026-10-02-feasibility")
ASSETS=["btc","eth","xrp","ada","link","bch","ltc","doge","uni","zec"]
BASE="https://community-api.coinmetrics.io/v4/timeseries/asset-metrics"
params={"assets":",".join(ASSETS),"metrics":"SplyCur","frequency":"1d",
        "start_time":"2023-01-01","end_time":"2024-12-31","page_size":10000}
u=BASE+"?"+urllib.parse.urlencode(params)
rows=[]
while u:
    req=urllib.request.Request(u,headers={"User-Agent":"Mozilla/5.0","Accept":"application/json"})
    o=json.loads(urllib.request.urlopen(req,timeout=30).read())
    rows.extend(o.get("data",[]))
    u=o.get("next_page_url")
    if u and u.startswith("https://api.coinmetrics.io"):
        u=u.replace("https://api.coinmetrics.io","https://community-api.coinmetrics.io",1)
    if u: time.sleep(.5)

by={a:[] for a in ASSETS}
for r in rows:
    a=r.get("asset")
    try: v=float(r["SplyCur"])
    except: continue
    if a in by and v>0: by[a].append((r["time"][:10],v))

stats={}
variable=[]
for a in ASSETS:
    q=sorted(by[a])
    nz=sum(1 for i in range(1,len(q)) if q[i][1]!=q[i-1][1])
    distinct=len({v for _,v in q})
    rec={"days":len(q),"first":q[0][0] if q else None,"last":q[-1][0] if q else None,
         "nonzero_daily_changes":nz,"distinct_values":distinct,
         "variable":len(q)>=700 and nz>=30}
    stats[a]=rec
    if rec["variable"]: variable.append(a)
    print(a,rec,flush=True)

summary={"metric":"SplyCur","assets":len(ASSETS),"variable_assets":len(variable),
         "variable_asset_list":variable,"coverage_gate_pass":len(variable)>=8,
         "post_price_returns_read":False,"by_asset":stats}
(ROOT/"results/summary.json").write_text(json.dumps(summary,indent=2))
(ROOT/"results/raw_metric_rows.json").write_text(json.dumps(rows))
print(json.dumps({k:v for k,v in summary.items() if k!="by_asset"},indent=2))
