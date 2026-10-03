import json,re,time,urllib.request,datetime,decimal
from pathlib import Path

ROOT=Path("strategy_templates/research/binance-futures-tick-size-adjustment/2023-2024-missing-batches-untouched")
ARTICLES=json.loads((ROOT/"inputs/articles.json").read_text())
H={"User-Agent":"Mozilla/5.0","Accept":"application/json"}
DETAIL="https://www.binance.com/bapi/composite/v1/public/cms/article/detail/query"

def get_json(url):
    last=None
    for i in range(6):
        try:
            req=urllib.request.Request(url,headers=H)
            with urllib.request.urlopen(req,timeout=25) as r:return json.loads(r.read())
        except Exception as e:last=e;time.sleep(.6*(i+1))
    raise last

def flat(n):
    if isinstance(n,dict):
        if n.get("node")=="text":return n.get("text","")
        s="".join(flat(x) for x in n.get("child",[]) or [])
        return (" "+s+" ") if n.get("tag") in {"p","div","td","th","tr","table","li","ul","ol","br"} else s
    if isinstance(n,list):return " ".join(flat(x) for x in n)
    return ""

events=[];audits=[];errors=[]
for i,a in enumerate(ARTICLES,1):
    try:
        o=get_json(f"{DETAIL}?articleCode={a['code']}");d=o.get("data") or {}
        body=d.get("body","{}");root=json.loads(body) if isinstance(body,str) else body
        txt=" ".join(flat(root).replace("\xa0"," ").replace("&nbsp;"," ").split())
        # Futures tick-size notices in this corpus use a single explicit effective timestamp.
        m=re.search(r"\b(?:at|by)\s+(20\d\d-\d\d-\d\d)\s+(\d{1,2}:\d{2})\s*\(UTC\)",txt,re.I)
        if not m:
            raise RuntimeError("no_effective_time")
        day,hm=m.group(1),m.group(2)
        if len(hm.split(":")[0])==1:hm="0"+hm
        et=int(datetime.datetime.fromisoformat(day+"T"+hm+":00+00:00").timestamp()*1000)
        found=[]
        # Match USD-M USDT perpetual rows only; COIN-M symbols end in USD, not USDT.
        for sym,old,new in re.findall(r"\b([A-Z0-9]{2,28}USDT)\s+([0-9]+(?:\.[0-9]+)?)\s+([0-9]+(?:\.[0-9]+)?)\b",txt):
            aa,bb=decimal.Decimal(old),decimal.Decimal(new)
            if aa==bb:continue
            rec={"article_code":a["code"],"article_title":d.get("title",a.get("title","")),
                 "publish_ms":int(d.get("publishDate") or a["release_time"]),
                 "effective_ms":et,"effective_utc":day+"T"+hm+":00Z",
                 "symbol":sym,"base":sym[:-4],"old_tick":old,"new_tick":new,
                 "direction":"LONG" if bb<aa else "SHORT"}
            found.append(rec)
        # Deduplicate identical symbol rows within the article.
        uniq={}
        for r in found:uniq[(r["symbol"],r["effective_ms"])]=r
        found=list(uniq.values());events.extend(found)
        audits.append({"code":a["code"],"title":d.get("title",a.get("title","")),
                       "effective_utc":day+"T"+hm+":00Z","usdt_events":len(found),"body_prefix":txt[:3500]})
        print("ARTICLE",i,len(ARTICLES),a["code"],"events",len(found),flush=True)
    except Exception as e:
        errors.append({"code":a["code"],"title":a.get("title"),"error":str(e)})
        print("ERROR",a["code"],e,flush=True)

(ROOT/"inputs/articles_audit.json").write_text(json.dumps(audits,ensure_ascii=False,indent=2))
(ROOT/"inputs/events_raw.json").write_text(json.dumps(events,ensure_ascii=False,indent=2))
(ROOT/"inputs/parse_errors.json").write_text(json.dumps(errors,ensure_ascii=False,indent=2))
summary={"missing_articles":len(ARTICLES),"parsed_articles":len(audits),"parse_errors":len(errors),
         "raw_events":len(events),"raw_unique_symbols":len({x["symbol"] for x in events}),
         "independent_articles":len({x["article_code"] for x in events}),
         "long":sum(x["direction"]=="LONG" for x in events),
         "short":sum(x["direction"]=="SHORT" for x in events),
         "returns_read":False}
(ROOT/"results/audit_summary.json").write_text(json.dumps(summary,indent=2))
print(json.dumps(summary,indent=2))
for x in events:print("EVENT",x["effective_utc"],x["symbol"],x["old_tick"],"->",x["new_tick"],x["direction"])
