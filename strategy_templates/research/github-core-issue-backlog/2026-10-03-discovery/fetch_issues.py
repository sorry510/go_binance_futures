import json,time,urllib.parse,urllib.request,urllib.error,math
from pathlib import Path

ROOT=Path("strategy_templates/research/github-core-issue-backlog/2026-10-03-discovery")
OUT=ROOT/"inputs"
MAP=json.loads((OUT/"source_map.json").read_text())
UA="Mozilla/5.0"
API="https://api.github.com/search/issues"

def req_json(url):
    while True:
        try:
            req=urllib.request.Request(url,headers={"User-Agent":UA,"Accept":"application/vnd.github+json"})
            with urllib.request.urlopen(req,timeout=30) as r:
                obj=json.loads(r.read())
                rem=int(r.headers.get("X-RateLimit-Remaining","1"))
                reset=int(r.headers.get("X-RateLimit-Reset","0"))
                if rem<=1:
                    wait=max(1,reset-int(time.time())+1)
                    print("RATE_WAIT",wait,flush=True); time.sleep(wait)
                return obj
        except urllib.error.HTTPError as e:
            if e.code in (403,429):
                reset=int(e.headers.get("X-RateLimit-Reset","0") or 0)
                wait=max(2,reset-int(time.time())+1)
                print("HTTP_RATE_WAIT",e.code,wait,flush=True); time.sleep(wait); continue
            raise
        except (urllib.error.URLError, TimeoutError, ConnectionError, OSError) as e:
            print("NET_RETRY",type(e).__name__,str(e)[:180],flush=True)
            time.sleep(3)
            continue

def fetch_year(asset,repo,year):
    fp=OUT/f"{asset}_{year}_issues.json"
    if fp.exists():
        x=json.loads(fp.read_text())
        print("REUSE",asset,year,len(x),flush=True)
        return x
    q=f"repo:{repo} is:issue created:{year}-01-01..{year}-12-31"
    params={"q":q,"per_page":100,"page":1,"sort":"created","order":"asc"}
    first=req_json(API+"?"+urllib.parse.urlencode(params))
    total=int(first.get("total_count",0))
    if total>1000:
        raise RuntimeError(f"{asset} {year} total {total} >1000; partition required")
    items=first.get("items",[])
    pages=math.ceil(total/100) if total else 0
    for page in range(2,pages+1):
        params["page"]=page
        o=req_json(API+"?"+urllib.parse.urlencode(params))
        items.extend(o.get("items",[]))
    rows=[{"number":int(x["number"]),"created_at":x["created_at"]} for x in items if not x.get("pull_request")]
    rows.sort(key=lambda x:(x["created_at"],x["number"]))
    if len(rows)!=total:
        raise RuntimeError(f"{asset} {year} expected {total}, got {len(rows)}")
    fp.write_text(json.dumps(rows,indent=2))
    print("DONE",asset,year,len(rows),flush=True)
    return rows

all_summary={}
for asset,repo in MAP.items():
    combined=[]
    # Reuse already-complete BTC combined file.
    if asset=="BTC" and (OUT/"BTC_issues.json").exists():
        x=json.loads((OUT/"BTC_issues.json").read_text())
        by={y:[r for r in x if r["created_at"].startswith(str(y))] for y in (2022,2023,2024)}
        if all(len(by[y])>0 for y in by):
            for y in (2022,2023,2024):
                fp=OUT/f"BTC_{y}_issues.json"
                if not fp.exists(): fp.write_text(json.dumps(by[y],indent=2))
                combined.extend(by[y])
                print("REUSE",asset,y,len(by[y]),flush=True)
        else:
            combined=[]
    if not combined:
        for year in (2022,2023,2024):
            combined.extend(fetch_year(asset,repo,year))
    combined.sort(key=lambda x:(x["created_at"],x["number"]))
    (OUT/f"{asset}_issues.json").write_text(json.dumps(combined,indent=2))
    all_summary[asset]={"repo":repo,"rows":len(combined),
                        "y2022":sum(r["created_at"].startswith("2022") for r in combined),
                        "y2023":sum(r["created_at"].startswith("2023") for r in combined),
                        "y2024":sum(r["created_at"].startswith("2024") for r in combined)}
    print("ASSET_DONE",asset,all_summary[asset],flush=True)

(OUT/"issue_fetch_summary.json").write_text(json.dumps(all_summary,indent=2))
print(json.dumps(all_summary,indent=2),flush=True)
