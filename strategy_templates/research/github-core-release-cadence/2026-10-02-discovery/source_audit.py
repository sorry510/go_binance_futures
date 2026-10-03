import json,math,time,urllib.request,urllib.error,datetime
from pathlib import Path
ROOT=Path("strategy_templates/research/github-core-release-cadence/2026-10-02-discovery")
MAP=json.loads((ROOT/"inputs/source_map.json").read_text())
UA="Mozilla/5.0"
END=datetime.datetime(2025,1,1,tzinfo=datetime.timezone.utc)

def get(url):
    last=None
    for k in range(6):
        try:
            req=urllib.request.Request(url,headers={"User-Agent":UA,"Accept":"application/vnd.github+json"})
            with urllib.request.urlopen(req,timeout=25) as r:
                return json.loads(r.read())
        except Exception as e:
            last=e; time.sleep(.7*(k+1))
    raise last

history={}; signals=[]; audit={}
for asset,repo in MAP.items():
    objs=[]
    for page in range(1,11):
        arr=get(f"https://api.github.com/repos/{repo}/releases?per_page=100&page={page}")
        if not arr: break
        objs.extend(arr)
        if len(arr)<100: break
        time.sleep(.1)
    stable=[]
    for r in objs:
        if r.get("draft") or r.get("prerelease") or not r.get("published_at"): continue
        dt=datetime.datetime.fromisoformat(r["published_at"].replace("Z","+00:00"))
        if dt>=END: continue
        stable.append({"release_id":r["id"],"tag_name":r.get("tag_name"),"name":r.get("name"),
                       "published_at":dt.isoformat().replace("+00:00","Z"),"utc_date":dt.date().isoformat(),
                       "html_url":r.get("html_url")})
    stable.sort(key=lambda x:x["published_at"])
    # dedupe same UTC day to earliest, matching parent family
    ded=[]
    seen=set()
    for r in stable:
        if r["utc_date"] in seen: continue
        seen.add(r["utc_date"]); ded.append(r)
    history[asset]={"repo":repo,"releases":ded}
    scores=[]
    for i in range(2,len(ded)):
        t0=datetime.datetime.fromisoformat(ded[i-2]["published_at"].replace("Z","+00:00"))
        t1=datetime.datetime.fromisoformat(ded[i-1]["published_at"].replace("Z","+00:00"))
        t2=datetime.datetime.fromisoformat(ded[i]["published_at"].replace("Z","+00:00"))
        a=(t1-t0).total_seconds(); b=(t2-t1).total_seconds()
        scores.append(None if a<=0 or b<=0 else math.log(a/b))
    n=0
    for i in range(3,len(ded)):
        prev=scores[i-3]; now=scores[i-2]
        if prev is None or now is None: continue
        side="LONG" if prev<=0<now else ("SHORT" if prev>=0>now else "")
        if not side: continue
        pub=datetime.datetime.fromisoformat(ded[i]["published_at"].replace("Z","+00:00"))
        if pub.year not in (2023,2024): continue
        signals.append({"asset":asset,"symbol":asset+"USDT","repo":repo,**ded[i],
                        "score_prev":prev,"score_now":now,"side":side})
        n+=1
    audit[asset]={"repo":repo,"objects_fetched":len(objs),"stable_days_before_2025":len(ded),
                  "first":ded[0]["published_at"] if ded else None,"last":ded[-1]["published_at"] if ded else None,
                  "discovery_signals":n}
    print(asset,repo,"days",len(ded),"signals",n,flush=True)

signals.sort(key=lambda x:(x["published_at"],x["asset"]))
(ROOT/"inputs/release_history_through_2024.json").write_text(json.dumps(history,indent=2))
(ROOT/"inputs/cadence_signals_raw.json").write_text(json.dumps(signals,indent=2))
summary={"raw_signals":len(signals),"raw_symbols":len({x["symbol"] for x in signals}),
         "long":sum(x["side"]=="LONG" for x in signals),"short":sum(x["side"]=="SHORT" for x in signals),
         "by_asset":audit,"post_event_returns_read":False}
(ROOT/"results/source_summary.json").write_text(json.dumps(summary,indent=2))
print(json.dumps({"raw_signals":summary["raw_signals"],"raw_symbols":summary["raw_symbols"],
                  "long":summary["long"],"short":summary["short"]},indent=2))
