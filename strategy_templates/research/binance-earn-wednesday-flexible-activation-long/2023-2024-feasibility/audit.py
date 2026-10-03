import datetime, json, re, time, urllib.request, urllib.error
from pathlib import Path

UTC=datetime.timezone.utc
ROOT=Path("strategy_templates/research/binance-earn-wednesday-flexible-activation-long/2023-2024-feasibility")
LIST="https://www.binance.com/bapi/composite/v1/public/cms/article/list/query"
DETAIL="https://www.binance.com/bapi/composite/v1/public/cms/article/detail/query"
UA="Mozilla/5.0"
EXCLUDED={
 "USDT","USDC","FDUSD","TUSD","USDP","BUSD","DAI","AEUR","EURI","EURT","EUR","TRY","BRL","ARS","COP",
 "MXN","CZK","PLN","RON","UAH","NGN","ZAR","GBP","AUD","JPY","RUB"
}
STOP_PREFIXES=(
 "Locked Products","Liquidity Farming","ETH Staking","SOL Staking","Auto-Invest",
 "Dual Investment","BNB Vault","DeFi Staking","Range Bound","Liquidity Swap"
)

def get_json(url,tries=6):
    last=None
    for k in range(tries):
        try:
            req=urllib.request.Request(url,headers={"User-Agent":UA,"Accept":"application/json"})
            with urllib.request.urlopen(req,timeout=25) as r:
                return json.loads(r.read())
        except Exception as e:
            last=e; time.sleep(1.5*(k+1))
    raise last
def flatten(node):
    if isinstance(node,dict):
        if node.get("node")=="text": return node.get("text","")
        return " ".join(flatten(x) for x in node.get("child",[]) or [])
    if isinstance(node,list): return " ".join(flatten(x) for x in node)
    return ""

def table_rows(node,out):
    if isinstance(node,dict):
        if node.get("tag")=="tr":
            cells=[]
            for c in node.get("child",[]) or []:
                if isinstance(c,dict) and c.get("tag") in ("td","th"):
                    cells.append(" ".join(flatten(c).replace("\xa0"," ").split()))
            if cells: out.append(cells)
        for c in node.get("child",[]) or []: table_rows(c,out)
    elif isinstance(node,list):
        for c in node: table_rows(c,out)

def token_like(s):
    s=s.replace("&nbsp;"," ").strip()
    return bool(re.fullmatch(r"[A-Z0-9]{2,20}",s)) and s not in EXCLUDED

def parse_flexible(body):
    root=json.loads(body) if isinstance(body,str) else body
    rows=[]; table_rows(root,rows)
    active=False; tokens=[]; trace=[]
    for cells in rows:
        first=cells[0].replace("&nbsp;"," ").strip()
        if first.startswith("Flexible Products"):
            active=True
            if len(cells)>1 and token_like(cells[1]): tokens.append(cells[1])
            trace.append(cells)
            continue
        if active and any(first.startswith(x) for x in STOP_PREFIXES):
            break
        if active:
            trace.append(cells)
            if token_like(first): tokens.append(first)
    return sorted(set(tokens)), trace
def find_warmup():
    best=None
    seen_2022=False
    for page in range(1,140):
        obj=get_json(f"{LIST}?type=1&pageNo={page}&pageSize=50&catalogId=49")
        arts=(((obj.get("data") or {}).get("catalogs") or [{}])[0].get("articles") or [])
        if not arts: break
        dates=[datetime.datetime.fromtimestamp(a["releaseDate"]/1000,UTC) for a in arts]
        for a,dt in zip(arts,dates):
            if dt.year==2022 and "Earn Wednesday" in a.get("title",""):
                if best is None or a["releaseDate"]>best["releaseDate"]: best=a
                seen_2022=True
        if seen_2022 and min(dates).year<2022: break
        time.sleep(.05)
    if best is None: raise RuntimeError("no pre-2023 Earn Wednesday warmup found")
    return best

def detail(a):
    obj=get_json(f"{DETAIL}?articleCode={a['code']}")
    d=(obj or {}).get("data") or {}
    tokens,trace=parse_flexible(d.get("body","{}"))
    return {
      "code":a["code"],"title":d.get("title",a.get("title","")),
      "publish_ms":int(d.get("publishDate",a.get("releaseDate",a.get("release_time")))),
      "tokens":tokens,"flexible_rows":trace,
    }

corpus=json.loads(Path(
 "strategy_templates/research/binance-corporate-action-support-long/2023-2024-feasibility/inputs/all_titles_2023_2024.json"
).read_text())
current=[a for a in corpus if a.get("title","").startswith("Earn Wednesday")]
current.sort(key=lambda x:x.get("releaseDate",x.get("release_time")))
warm=find_warmup()
print("WARMUP_TITLE",datetime.datetime.fromtimestamp(warm["releaseDate"]/1000,UTC),warm["title"],flush=True)
articles=[]
errors=[]
all_source=[warm]+current
for i,a in enumerate(all_source):
    try:
        rec=detail(a)
        rec["publish"]=datetime.datetime.fromtimestamp(rec["publish_ms"]/1000,UTC).isoformat()
        rec["is_warmup"]=(i==0)
        articles.append(rec)
        print("ARTICLE",i,len(all_source)-1,rec["publish"][:10],rec["tokens"],flush=True)
    except Exception as e:
        errors.append({"code":a.get("code"),"title":a.get("title"),"error":repr(e),"is_warmup":i==0})
        print("ERROR",a.get("code"),repr(e),flush=True)
    time.sleep(.08)

if errors:
    (ROOT/"inputs/parse_errors.json").write_text(json.dumps(errors,ensure_ascii=False,indent=2))
    raise RuntimeError(f"parse/fetch errors={len(errors)}; no coverage decision")

events=[]
prev=None
for rec in articles:
    cur=set(rec["tokens"])
    if rec["is_warmup"]:
        prev=cur
        continue
    entered=sorted(cur-(prev or set()))
    rec["entered_tokens"]=entered
    rec["previous_tokens"]=sorted(prev or set())
    for token in entered:
        events.append({
          "article_code":rec["code"],"article_title":rec["title"],
          "publish_ms":rec["publish_ms"],"publish":rec["publish"],
          "asset":token,"symbol":token+"USDT","side":"LONG"
        })
    prev=cur
(ROOT/"inputs/articles.json").write_text(json.dumps(articles,ensure_ascii=False,indent=2))
(ROOT/"inputs/raw_events.json").write_text(json.dumps(events,ensure_ascii=False,indent=2))
(ROOT/"inputs/parse_errors.json").write_text("[]")

summary={
 "earn_wednesday_articles_2023_2024":len(current),
 "warmup_article_code":warm["code"],
 "parsed_articles_including_warmup":len(articles),
 "raw_activation_events":len(events),
 "raw_unique_tokens":len({e["asset"] for e in events}),
 "independent_batches":len({e["article_code"] for e in events}),
 "post_event_returns_read":False
}
(ROOT/"results/audit_summary.json").write_text(json.dumps(summary,ensure_ascii=False,indent=2))
print(json.dumps(summary,ensure_ascii=False,indent=2))
print("TOKENS",sorted({e["asset"] for e in events}))
