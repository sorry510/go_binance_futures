import calendar,json,math,time,urllib.parse,urllib.request,urllib.error
from pathlib import Path

ROOT=Path("strategy_templates/research/github-core-issue-resolution-balance/2026-10-03-discovery")
OUT=ROOT/"inputs"
MAP=json.loads((OUT/"source_map.json").read_text())
API="https://api.github.com/search/issues"
UA="Mozilla/5.0"

def req_json(url):
    while True:
        try:
            req=urllib.request.Request(url,headers={"User-Agent":UA,"Accept":"application/vnd.github+json"})
            with urllib.request.urlopen(req,timeout=30) as r:
                obj=json.loads(r.read())
                rem=int(r.headers.get("X-RateLimit-Remaining","1"))
                reset=int(r.headers.get("X-RateLimit-Reset","0") or 0)
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
        except (urllib.error.URLError,TimeoutError,ConnectionError,OSError) as e:
            print("NET_RETRY",type(e).__name__,str(e)[:180],flush=True); time.sleep(3)

def search_range(repo,start,end):
    q=f"repo:{repo} is:issue closed:{start}..{end}"
    params={"q":q,"per_page":100,"page":1,"sort":"created","order":"asc"}
    first=req_json(API+"?"+urllib.parse.urlencode(params))
    total=int(first.get("total_count",0))
    if total>1000:
        return None,total
    items=first.get("items",[])
    for page in range(2,math.ceil(total/100)+1):
        params["page"]=page
        o=req_json(API+"?"+urllib.parse.urlencode(params))
        items.extend(o.get("items",[]))
    rows=[{"number":int(x["number"]),"closed_at":x.get("closed_at")} for x in items
          if not x.get("pull_request") and x.get("closed_at")]
    # Search result total should already be issue-only.
    rows.sort(key=lambda x:(x["closed_at"],x["number"]))
    if len(rows)!=total:
        raise RuntimeError(f"{repo} {start}..{end}: expected {total}, got {len(rows)}")
    return rows,total

def fetch_year(asset,repo,year):
    fp=OUT/f"{asset}_{year}_closures.json"
    if fp.exists():
        x=json.loads(fp.read_text())
        print("REUSE",asset,year,len(x),flush=True)
        return x
    rows,total=search_range(repo,f"{year}-01-01",f"{year}-12-31")
    if rows is None:
        rows=[]
        print("MONTH_SPLIT",asset,year,total,flush=True)
        for month in range(1,13):
            last=calendar.monthrange(year,month)[1]
            part,n=search_range(repo,f"{year}-{month:02d}-01",f"{year}-{month:02d}-{last:02d}")
            if part is None:
                raise RuntimeError(f"{asset} {year}-{month:02d} still >1000")
            rows.extend(part)
        # An issue can only close once, so unique by issue number is safe.
        rows=list({x["number"]:x for x in rows}.values())
        rows.sort(key=lambda x:(x["closed_at"],x["number"]))
        if len(rows)!=total:
            raise RuntimeError(f"{asset} {year} month split expected {total}, got {len(rows)}")
    fp.write_text(json.dumps(rows,indent=2))
    print("DONE",asset,year,len(rows),flush=True)
    return rows

summary={}
for asset,repo in MAP.items():
    combined=[]
    for year in (2022,2023,2024):
        combined.extend(fetch_year(asset,repo,year))
    combined.sort(key=lambda x:(x["closed_at"],x["number"]))
    (OUT/f"{asset}_closures.json").write_text(json.dumps(combined,indent=2))
    summary[asset]={"repo":repo,"rows":len(combined),
                    "y2022":sum(x["closed_at"].startswith("2022") for x in combined),
                    "y2023":sum(x["closed_at"].startswith("2023") for x in combined),
                    "y2024":sum(x["closed_at"].startswith("2024") for x in combined)}
    print("ASSET_DONE",asset,summary[asset],flush=True)

(OUT/"closure_fetch_summary.json").write_text(json.dumps(summary,indent=2))
print(json.dumps(summary,indent=2),flush=True)
