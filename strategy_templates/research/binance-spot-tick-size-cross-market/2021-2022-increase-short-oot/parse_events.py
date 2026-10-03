import urllib.request, json, re, datetime, time, decimal
from pathlib import Path

ROOT=Path("strategy_templates/research/binance-spot-tick-size-cross-market/2021-2022-increase-short-oot")
ARTICLES=json.loads((ROOT/"inputs/articles.json").read_text())
H={"User-Agent":"Mozilla/5.0","Accept":"application/json"}
DETAIL="https://www.binance.com/bapi/composite/v1/public/cms/article/detail/query"

def get_json(url):
    last=None
    for k in range(7):
        try:
            req=urllib.request.Request(url,headers=H)
            with urllib.request.urlopen(req,timeout=25) as r: return json.loads(r.read())
        except Exception as e:
            last=e; time.sleep(.75*(k+1))
    raise last

def flat(n):
    if isinstance(n,dict):
        if n.get("node")=="text": return n.get("text","")
        s="".join(flat(c) for c in n.get("child",[]) or [])
        return (" "+s+" ") if n.get("tag") in {"p","div","td","th","tr","table","li","ul","ol","br"} else s
    if isinstance(n,list): return " ".join(flat(x) for x in n)
    return ""

def norm_hm(hm,ampm):
    h,m=map(int,hm.split(":"))
    if ampm:
        a=ampm.upper()
        if a=="AM" and h==12: h=0
        if a=="PM" and h!=12: h+=12
    return f"{h:02d}:{m:02d}"

def effective_segments(txt):
    # 2022 uses 24h times; 2021 uses forms like "06:00 AM (UTC)" and "4:00 AM (UTC)".
    marks=list(re.finditer(r"Batch\s+\d+\s*:\s*(20\d\d-\d\d-\d\d)\s+(\d{1,2}:\d{2})\s*(AM|PM)?\s*\(UTC\)",txt,re.I))
    if marks:
        return [(m.group(1),norm_hm(m.group(2),m.group(3)),txt[m.end():marks[i+1].start() if i+1<len(marks) else len(txt)])
                for i,m in enumerate(marks)]
    marks=list(re.finditer(r"By\s+(20\d\d-\d\d-\d\d)\s+(\d{1,2}:\d{2})\s*(AM|PM)?\s*\(UTC\)\s*:",txt,re.I))
    if marks:
        return [(m.group(1),norm_hm(m.group(2),m.group(3)),txt[m.end():marks[i+1].start() if i+1<len(marks) else len(txt)])
                for i,m in enumerate(marks)]
    m=re.search(r"\b(?:at|by|on)\s+(20\d\d-\d\d-\d\d)\s+(\d{1,2}:\d{2})\s*(AM|PM)?\s*\(UTC\)",txt,re.I)
    return [(m.group(1),norm_hm(m.group(2),m.group(3)),txt)] if m else []

events=[]; audits=[]; errors=[]; texts=[]
for i,a in enumerate(ARTICLES,1):
    code=a["code"]
    try:
        o=get_json(f"{DETAIL}?articleCode={code}"); d=o.get("data") or {}
        body=d.get("body","{}")
        try: root=json.loads(body) if isinstance(body,str) else body
        except Exception: root={}
        txt=re.sub(r"\s+"," ",flat(root).replace("\xa0"," ").replace("&nbsp;"," ")).strip()
        texts.append({"code":code,"title":d.get("title",a.get("title","")),"text":txt})
        segs=effective_segments(txt); n=0
        for day,hm,seg in segs:
            et=int(datetime.datetime.fromisoformat(day+"T"+hm+":00+00:00").timestamp()*1000)
            # The 2021-08 article also contains a separate Step Size table.
            # Only parse the Tick Size section to preserve event semantics.
            if "Trading Pair Step Size" in seg:
                seg=seg.split("Trading Pair Step Size",1)[0]
            # 2021-2022 table cells are flattened as e.g. DOTUSDT 0.01 0.001.
            for sym,old,new in re.findall(r"\b([A-Z0-9]{2,24}USDT)\s+([0-9]+(?:\.[0-9]+)?)\s+([0-9]+(?:\.[0-9]+)?)\b",seg):
                aa,bb=decimal.Decimal(old),decimal.Decimal(new)
                if aa==bb: continue
                base=sym[:-4]
                if base.endswith("UP") or base.endswith("DOWN"): continue
                direction="SHORT" if bb>aa else "LONG"
                events.append({
                    "article_code":code,"article_title":d.get("title",a.get("title","")),
                    "publish":a.get("publish"),"effective_ms":et,"effective_utc":day+"T"+hm+":00Z",
                    "spot_pair":base+"/USDT","base":base,"symbol":sym,
                    "old_tick":old,"new_tick":new,"direction":direction
                }); n+=1
        audits.append({"code":code,"title":d.get("title",a.get("title","")),"segments":len(segs),"usdt_events":n})
        print("ARTICLE",i,len(ARTICLES),code,"segments",len(segs),"USDT_EVENTS",n,flush=True)
    except Exception as e:
        errors.append({"code":code,"title":a.get("title"),"error":str(e)})
        print("ERROR",code,e,flush=True)

short=[x for x in events if x["direction"]=="SHORT"]
(ROOT/"inputs/article_texts.json").write_text(json.dumps(texts,ensure_ascii=False,indent=2))
(ROOT/"inputs/articles_audit.json").write_text(json.dumps(audits,ensure_ascii=False,indent=2))
(ROOT/"inputs/all_usdt_tick_events.json").write_text(json.dumps(events,ensure_ascii=False,indent=2))
(ROOT/"inputs/increase_short_events_raw.json").write_text(json.dumps(short,ensure_ascii=False,indent=2))
(ROOT/"inputs/parse_errors.json").write_text(json.dumps(errors,ensure_ascii=False,indent=2))
summary={"articles":len(ARTICLES),"parsed_articles":len(audits),"parse_errors":len(errors),
         "all_usdt_events":len(events),"increase_short_events":len(short),
         "increase_unique_symbols":len({x["symbol"] for x in short}),
         "increase_batches":len({x["effective_utc"] for x in short}),"returns_read":False}
(ROOT/"results/parse_summary.json").write_text(json.dumps(summary,indent=2))
print(json.dumps(summary,indent=2))
for x in short: print("SHORT_EVENT",x["effective_utc"],x["spot_pair"],x["old_tick"],"->",x["new_tick"])
