import json,urllib.request,time,collections
assets=["btc","eth","xrp","ada","doge","ltc","link","bch","etc","trx","xlm","uni","aave","zec","mana","algo","snx","comp","sol","avax"]
sets={}
errors={}
for i,a in enumerate(assets,1):
    u=f"https://community-api.coinmetrics.io/v4/catalog/assets?assets={a}"
    try:
        req=urllib.request.Request(u,headers={"User-Agent":"Mozilla/5.0"})
        o=json.loads(urllib.request.urlopen(req,timeout=30).read())
        data=o.get("data") or []
        mets=set()
        if data:
            for m in data[0].get("metrics",[]):
                if any(f.get("community") and f.get("frequency")=="1d" for f in m.get("frequencies",[])):
                    mets.add(m["metric"])
        sets[a]=mets
        print("ASSET",i,len(assets),a,"metrics",len(mets),flush=True)
    except Exception as e:
        errors[a]=repr(e); sets[a]=set(); print("ERR",a,e,flush=True)
    time.sleep(.2)
cnt=collections.defaultdict(list)
for a,ms in sets.items():
    for m in ms: cnt[m].append(a)
rows=[{"metric":m,"count":len(v),"assets":sorted(v)} for m,v in cnt.items() if len(v)>=8]
rows.sort(key=lambda x:(-x["count"],x["metric"]))
out={"assets":assets,"errors":errors,"metrics_ge8":rows}
open("/tmp/coinmetrics_metric_inventory.json","w").write(json.dumps(out,indent=2))
print("\nMETRICS_GE8",len(rows))
for r in rows:
    print(f"{r['count']:2d} {r['metric']}: {','.join(r['assets'])}")
