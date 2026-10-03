import json,time,urllib.parse,urllib.request,urllib.error
from pathlib import Path

ROOT=Path("strategy_templates/research/github-core-issue-resolution-balance/2026-10-03-feasibility")
MAP=json.loads((ROOT/"source_map.json").read_text())
API="https://api.github.com/search/issues"
UA="Mozilla/5.0"

def request_total(repo):
    q=f"repo:{repo} is:issue closed:2023-01-01..2024-12-31"
    url=API+"?"+urllib.parse.urlencode({"q":q,"per_page":1})
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
                return int(obj.get("total_count",0))
        except urllib.error.HTTPError as e:
            if e.code in (403,429):
                reset=int(e.headers.get("X-RateLimit-Reset","0") or 0)
                wait=max(2,reset-int(time.time())+1)
                print("HTTP_RATE_WAIT",wait,flush=True); time.sleep(wait); continue
            raise
        except (urllib.error.URLError,TimeoutError,ConnectionError,OSError) as e:
            print("NET_RETRY",type(e).__name__,str(e)[:160],flush=True); time.sleep(3)

rows={}
for asset,repo in MAP.items():
    n=request_total(repo)
    rows[asset]={"repo":repo,"closed_2023_2024":n}
    print(asset,n,flush=True)
passed=sum(v["closed_2023_2024"]>=100 for v in rows.values())
summary={
  "version":"v177",
  "repos":rows,
  "repos_ge_100_closures_2023_2024":passed,
  "required_repos":8,
  "coverage_gate_pass":passed>=8,
  "post_token_returns_read":False
}
(ROOT/"results/summary.json").write_text(json.dumps(summary,indent=2))
print(json.dumps(summary,indent=2),flush=True)
