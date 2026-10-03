import ast, datetime, json, re, time, urllib.request
from pathlib import Path

UTC=datetime.timezone.utc
ROOT=Path("strategy_templates/research/binance-earn-wednesday-flexible-activation-long/2023-2024-feasibility")
DETAIL="https://www.binance.com/bapi/composite/v1/public/cms/article/detail/query"
UA="Mozilla/5.0"
EXCLUDED={
 "USDT","USDC","FDUSD","TUSD","USDP","BUSD","DAI","AEUR","EURI","EURT",
 "EUR","TRY","BRL","ARS","COP","MXN","CZK","PLN","RON","UAH","NGN","ZAR",
 "GBP","AUD","JPY","RUB"
}
STOP_PREFIXES=(
 "Locked Products","Liquidity Farming","ETH Staking","SOL Staking","Auto-Invest",
 "Dual Investment","BNB Vault","DeFi Staking","Range Bound","Liquidity Swap"
)

def get_json(url,tries=8):
    last=None
    for k in range(tries):
        try:
            req=urllib.request.Request(url,headers={"User-Agent":UA,"Accept":"application/json"})
            with urllib.request.urlopen(req,timeout=25) as r: return json.loads(r.read())
        except Exception as e:
            last=e; time.sleep(5+5*k)
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
    active=False; tokens=[]
    for cells in rows:
        first=cells[0].replace("&nbsp;"," ").strip()
        if first.startswith("Flexible Products"):
            active=True
            if len(cells)>1 and token_like(cells[1]): tokens.append(cells[1])
            continue
        if active and any(first.startswith(x) for x in STOP_PREFIXES): break
        if active and token_like(first): tokens.append(first)
    return sorted(set(tokens))
corpus=json.loads(Path(
 "strategy_templates/research/binance-corporate-action-support-long/2023-2024-feasibility/inputs/all_titles_2023_2024.json"
).read_text())
current=[a for a in corpus if a.get("title","").startswith("Earn Wednesday")]
current.sort(key=lambda x:x.get("releaseDate",x.get("release_time")))

sets={0:{"tokens":["ENS","MTL","RLC"],"date":"2022-12-28","title":"Earn Wednesday: New Limited-Time Offers Available Now! (2022-12-28)"}}
log=(ROOT/"results/audit.log").read_text()
for line in log.splitlines():
    m=re.match(r"ARTICLE (\d+) 88 (\d{4}-\d{2}-\d{2}) (\[.*\])$",line)
    if not m: continue
    idx=int(m.group(1))
    toks=[t for t in ast.literal_eval(m.group(3)) if t not in EXCLUDED]
    title=current[idx-1]["title"] if idx>0 else sets[0]["title"]
    sets[idx]={"tokens":sorted(set(toks)),"date":m.group(2),"title":title}

missing=[i for i in range(1,len(current)+1) if i not in sets]
print("CHECKPOINT",len(sets)-1,"of",len(current),"missing",missing,flush=True)

for idx in missing:
    a=current[idx-1]
    obj=get_json(f"{DETAIL}?articleCode={a['code']}")
    data=(obj or {}).get("data") or {}
    toks=parse_flexible(data.get("body","{}"))
    pub=int(data.get("publishDate",a.get("release_time")))
    dt=datetime.datetime.fromtimestamp(pub/1000,UTC).date().isoformat()
    sets[idx]={"tokens":toks,"date":dt,"title":data.get("title",a["title"]),"publish_ms":pub}
    print("RESUMED",idx,len(current),dt,toks,flush=True)
    time.sleep(3)
articles=[]
events=[]
prev=set(sets[0]["tokens"])
for idx,a in enumerate(current,1):
    rec=sets[idx]
    cur=set(rec["tokens"])
    publish_ms=int(rec.get("publish_ms",a.get("release_time")))
    entered=sorted(cur-prev)
    art={
      "index":idx,"code":a["code"],"title":rec["title"],"publish_ms":publish_ms,
      "publish":datetime.datetime.fromtimestamp(publish_ms/1000,UTC).isoformat(),
      "tokens":sorted(cur),"previous_tokens":sorted(prev),"entered_tokens":entered
    }
    articles.append(art)
    for token in entered:
        events.append({
          "article_code":a["code"],"article_title":art["title"],
          "publish_ms":publish_ms,"publish":art["publish"],
          "asset":token,"symbol":token+"USDT","side":"LONG"
        })
    prev=cur

(ROOT/"inputs/articles.json").write_text(json.dumps(articles,ensure_ascii=False,indent=2))
(ROOT/"inputs/raw_events.json").write_text(json.dumps(events,ensure_ascii=False,indent=2))
(ROOT/"inputs/parse_errors.json").write_text("[]")
summary={
 "earn_wednesday_articles_2023_2024":len(current),
 "warmup_date":"2022-12-28",
 "raw_activation_events":len(events),
 "raw_unique_tokens":len({e["asset"] for e in events}),
 "independent_batches":len({e["article_code"] for e in events}),
 "post_event_returns_read":False
}
(ROOT/"results/audit_summary.json").write_text(json.dumps(summary,ensure_ascii=False,indent=2))
print(json.dumps(summary,ensure_ascii=False,indent=2))
print("TOKENS",sorted({e["asset"] for e in events}))
