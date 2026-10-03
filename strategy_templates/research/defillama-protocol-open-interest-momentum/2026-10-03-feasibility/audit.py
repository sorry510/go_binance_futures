import concurrent.futures,datetime,json,time,urllib.request,urllib.error
from collections import defaultdict
from pathlib import Path

ROOT=Path("strategy_templates/research/defillama-protocol-open-interest-momentum/2026-10-03-feasibility")
UA={"User-Agent":"Mozilla/5.0","Accept":"application/json"}
STABLE={"USDT","USDC","BUSD","DAI","TUSD","FDUSD","USDP","USDS","FRAX","LUSD","USD","USDE","SUSDE","DOLA"}

def get_json(url):
    last=None
    for i in range(5):
        try:
            req=urllib.request.Request(url,headers=UA)
            with urllib.request.urlopen(req,timeout=30) as r:
                return json.loads(r.read())
        except Exception as e:
            last=e; time.sleep(.4*(i+1))
    raise last

overview=get_json("https://api.llama.fi/overview/open-interest?excludeTotalDataChart=true&excludeTotalDataChartBreakdown=true")
protocols=overview.get("protocols") or []
slugs=sorted({p.get("slug") for p in protocols if p.get("slug")})
print("overview_protocols",len(slugs),flush=True)

def fetch(slug):
    try:
        o=get_json(f"https://api.llama.fi/summary/open-interest/{slug}?dataType=openInterestAtEnd&excludeTotalDataChartBreakdown=true")
        chart=o.get("totalDataChart") or []
        vals=[]
        for row in chart:
            try:
                ts=int(row[0]); v=float(row[1])
                dt=datetime.datetime.fromtimestamp(ts,datetime.timezone.utc).date()
            except: continue
            if v>=0: vals.append((dt.isoformat(),v))
        return {
          "slug":slug,"name":o.get("name"),"displayName":o.get("displayName"),
          "symbol":(o.get("symbol") or "").strip().upper(),
          "gecko_id":o.get("gecko_id"),"chains":o.get("chains") or [],
          "chart":vals
        }
    except Exception as e:
        return {"slug":slug,"error":repr(e)}

rows=[]
with concurrent.futures.ThreadPoolExecutor(max_workers=8) as ex:
    futs={ex.submit(fetch,s):s for s in slugs}
    for i,f in enumerate(concurrent.futures.as_completed(futs),1):
        r=f.result(); rows.append(r)
        if i%20==0: print("FETCH",i,"/",len(slugs),flush=True)

(ROOT/"inputs/protocol_summaries.json").write_text(json.dumps(rows,ensure_ascii=False))
errs=[r for r in rows if r.get("error")]
print("errors",len(errs),flush=True)

by=defaultdict(list)
for r in rows:
    s=r.get("symbol","")
    if not s or s in STABLE or "/" in s or " " in s or len(s)>15: continue
    chart=r.get("chart") or []
    d23=sum(1 for d,v in chart if d.startswith("2023-"))
    d24=sum(1 for d,v in chart if d.startswith("2024-"))
    if d23+d24<30: continue
    by[s].append(r)

def head_ok(symbol,month):
    u=f"https://data.binance.vision/data/futures/um/monthly/klines/{symbol}USDT/1h/{symbol}USDT-1h-{month}.zip"
    try:
        req=urllib.request.Request(u,method="HEAD",headers={"User-Agent":"Mozilla/5.0"})
        with urllib.request.urlopen(req,timeout=15) as r:
            return r.status==200
    except: return False

candidates=[]
for sym,rs in sorted(by.items()):
    first=min(d for r in rs for d,v in r["chart"])
    last=max(d for r in rs for d,v in r["chart"])
    d23=sum(1 for d in {d for r in rs for d,v in r["chart"]} if d.startswith("2023-"))
    d24=sum(1 for d in {d for r in rs for d,v in r["chart"]} if d.startswith("2024-"))
    has_2022_12=head_ok(sym,"2022-12")
    has_2024_12=head_ok(sym,"2024-12")
    rec={"symbol":sym,"protocols":[r["slug"] for r in rs],"protocol_names":[r.get("name") for r in rs],
         "first_oi_date":first,"last_oi_date":last,"days_2023":d23,"days_2024":d24,
         "binance_2022_12":has_2022_12,"binance_2024_12":has_2024_12,
         "plausible_2y_by_end_2024":has_2022_12 and has_2024_12}
    candidates.append(rec)
    print("CAND",rec,flush=True)

eligible=[x for x in candidates if x["plausible_2y_by_end_2024"]]
summary={"overview_protocols":len(slugs),"summary_errors":len(errs),
         "symbols_with_2023_2024_oi":len(candidates),
         "plausible_same_name_2y_tokens":len(eligible),
         "coverage_gate_pass":len(eligible)>=8,
         "tokens":[x["symbol"] for x in eligible],
         "post_event_returns_read":False}
(ROOT/"results/candidates.json").write_text(json.dumps(candidates,ensure_ascii=False,indent=2))
(ROOT/"results/summary.json").write_text(json.dumps(summary,ensure_ascii=False,indent=2))
print(json.dumps(summary,indent=2),flush=True)
