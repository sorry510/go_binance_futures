import json,re,time,urllib.request,urllib.error
from pathlib import Path

ROOT=Path("strategy_templates/research/binance-liquid-swap-pool-removal-short/2023-2024-feasibility")
DETAIL="https://www.binance.com/bapi/composite/v1/public/cms/article/detail/query"
UA="Mozilla/5.0"
STABLE_FIAT={
 "USDT","USDC","BUSD","TUSD","FDUSD","DAI","USDP","USDS","USD","EUR","TRY","GBP","AUD",
 "BRL","BIDR","IDRT","RUB","UAH","NGN","PLN","RON","ARS","ZAR","CZK","JPY"
}

def get_json(url):
    last=None
    for i in range(5):
        try:
            req=urllib.request.Request(url,headers={"User-Agent":UA,"Accept":"application/json"})
            with urllib.request.urlopen(req,timeout=25) as r:
                return json.loads(r.read())
        except Exception as e:
            last=e; time.sleep(.6*(i+1))
    raise last

def flatten(n):
    if isinstance(n,dict):
        if n.get("node")=="text": return n.get("text","")
        s="".join(flatten(x) for x in n.get("child",[]) or [])
        return (" "+s+" ") if n.get("tag") in {"p","div","td","th","tr","table","li","ul","ol","br"} else s
    if isinstance(n,list): return " ".join(flatten(x) for x in n)
    return ""
titles=json.loads((ROOT/"inputs/title_candidates.json").read_text())
articles=[]
events=[]
errors=[]
for i,a in enumerate(titles,1):
    try:
        o=get_json(f"{DETAIL}?articleCode={a['code']}")
        data=(o or {}).get("data") or {}
        body=json.loads(data.get("body","{}")) if isinstance(data.get("body"),str) else (data.get("body") or {})
        text=" ".join(flatten(body).replace("\xa0"," ").replace("&nbsp;"," ").split())
        # Limit extraction to the removal-list area before user-instruction boilerplate.
        core=text
        marker="Users who hold positions"
        if marker in core: core=core.split(marker,1)[0]
        pairs=sorted(set(re.findall(r"\b([A-Z0-9]{2,20})/([A-Z0-9]{2,20})\b",core)))
        assets=sorted({x for pair in pairs for x in pair if x not in STABLE_FIAT})
        rec={
          "code":a["code"],"title":data.get("title",a.get("title","")),
          "publish_ms":int(data.get("publishDate") or a["release_time"]),
          "pairs":["/".join(x) for x in pairs],"assets":assets,
          "body_prefix":core[:5000],
        }
        articles.append(rec)
        for asset in assets:
            events.append({"article_code":rec["code"],"title":rec["title"],"publish_ms":rec["publish_ms"],
                           "asset":asset,"symbol":asset+"USDT","side":"SHORT","pairs":rec["pairs"]})
        print("ARTICLE",i,len(titles),a["code"],"pairs",len(pairs),"assets",assets,flush=True)
    except Exception as e:
        errors.append({"code":a["code"],"title":a.get("title"),"error":str(e)})
        print("ERROR",a["code"],e,flush=True)

(ROOT/"inputs/articles.json").write_text(json.dumps(articles,ensure_ascii=False,indent=2))
(ROOT/"inputs/events_raw.json").write_text(json.dumps(events,ensure_ascii=False,indent=2))
(ROOT/"inputs/parse_errors.json").write_text(json.dumps(errors,ensure_ascii=False,indent=2))
summary={"articles":len(articles),"parse_errors":len(errors),"raw_events":len(events),
         "raw_unique_tokens":len({e["asset"] for e in events}),
         "independent_batches":len({e["article_code"] for e in events}),
         "post_event_returns_read":False}
(ROOT/"results/audit_summary.json").write_text(json.dumps(summary,indent=2))
print(json.dumps(summary,indent=2))
