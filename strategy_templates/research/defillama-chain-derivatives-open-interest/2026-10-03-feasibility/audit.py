import concurrent.futures,datetime,json,time,urllib.request
from collections import defaultdict
from pathlib import Path

ROOT=Path("strategy_templates/research/defillama-chain-derivatives-open-interest/2026-10-03-feasibility")
MAP=json.loads((ROOT/"inputs/chain_native_map.json").read_text())
UA={"User-Agent":"Mozilla/5.0","Accept":"application/json"}

def get_json(url):
    last=None
    for i in range(5):
        try:
            req=urllib.request.Request(url,headers=UA)
            with urllib.request.urlopen(req,timeout=35) as r:return json.loads(r.read())
        except Exception as e:
            last=e; time.sleep(.5*(i+1))
    raise last

overview=get_json("https://api.llama.fi/overview/open-interest?excludeTotalDataChart=true&excludeTotalDataChartBreakdown=true")
protocols=overview.get("protocols") or []
targets=[]
for p in protocols:
    chains=p.get("chains") or []
    if any(c in MAP for c in chains) and p.get("slug"):
        targets.append({"slug":p["slug"],"chains":chains,"name":p.get("name")})
print("overview",len(protocols),"mapped-chain protocols",len(targets),flush=True)
(ROOT/"inputs/target_protocols.json").write_text(json.dumps(targets,ensure_ascii=False,indent=2))

def fetch(t):
    slug=t["slug"]
    try:
        o=get_json(f"https://api.llama.fi/summary/open-interest/{slug}?dataType=openInterestAtEnd&excludeTotalDataChartBreakdown=false")
        return {"slug":slug,"name":o.get("name"),"breakdown":o.get("totalDataChartBreakdown") or []}
    except Exception as e:
        return {"slug":slug,"name":t.get("name"),"error":repr(e),"breakdown":[]}

rows=[]
with concurrent.futures.ThreadPoolExecutor(max_workers=8) as ex:
    futs=[ex.submit(fetch,t) for t in targets]
    for i,f in enumerate(concurrent.futures.as_completed(futs),1):
        rows.append(f.result())
        if i%10==0:print("FETCH",i,"/",len(futs),flush=True)
(ROOT/"inputs/protocol_breakdowns.json").write_text(json.dumps(rows,ensure_ascii=False))

chain_day=defaultdict(lambda:defaultdict(float))
errors=[]
for r in rows:
    if r.get("error"):
        errors.append(r);continue
    for item in r.get("breakdown") or []:
        try:
            ts=int(item[0]); dt=datetime.datetime.fromtimestamp(ts,datetime.timezone.utc).date().isoformat()
            bd=item[1]
        except:continue
        if not isinstance(bd,dict):continue
        for chain,sub in bd.items():
            if chain not in MAP:continue
            total=0.0
            if isinstance(sub,dict):
                for v in sub.values():
                    try:total+=float(v)
                    except:pass
            else:
                try:total=float(sub)
                except:continue
            if total>=0: chain_day[chain][dt]+=total

def head_ok(sym,m):
    u=f"https://data.binance.vision/data/futures/um/monthly/klines/{sym}USDT/1h/{sym}USDT-1h-{m}.zip"
    try:
        req=urllib.request.Request(u,method="HEAD",headers={"User-Agent":"Mozilla/5.0"})
        with urllib.request.urlopen(req,timeout=15) as r:return r.status==200
    except:return False

audit=[]
for chain,sym in MAP.items():
    ds=chain_day.get(chain,{})
    nz=[d for d,v in ds.items() if v>0 and (d.startswith("2023-") or d.startswith("2024-"))]
    b22=head_ok(sym,"2022-12")
    b24=head_ok(sym,"2024-12")
    rec={"chain":chain,"symbol":sym,"oi_days_2023_2024":len(nz),
         "first_oi_date":min(ds) if ds else None,"last_oi_date":max(ds) if ds else None,
         "binance_2022_12":b22,"binance_2024_12":b24,
         "passes_oi_history":len(nz)>=30,
         "passes_binance_proxy":b22 and b24,
         "eligible":len(nz)>=30 and b22 and b24}
    audit.append(rec);print("CHAIN",rec,flush=True)

eligible=[x for x in audit if x["eligible"]]
summary={"overview_protocols":len(protocols),"mapped_chain_protocols":len(targets),
         "summary_errors":len(errors),"mapped_chains":len(MAP),
         "eligible_chains":len(eligible),"eligible_tokens":[x["symbol"] for x in eligible],
         "coverage_gate_pass":len(eligible)>=8,"post_event_returns_read":False}
(ROOT/"results/chain_audit.json").write_text(json.dumps(audit,indent=2))
(ROOT/"results/summary.json").write_text(json.dumps(summary,indent=2))
print(json.dumps(summary,indent=2),flush=True)
