import datetime, json, re, time, urllib.request, urllib.error
from pathlib import Path

UTC = datetime.timezone.utc
ROOT = Path("strategy_templates/research/binance-trading-bots-support-long/2023-2024-feasibility")
LIST = "https://www.binance.com/bapi/composite/v1/public/cms/article/list/query"
DETAIL = "https://www.binance.com/bapi/composite/v1/public/cms/article/detail/query"
UA = "Mozilla/5.0"
EXCLUDED_BASES = {
    "USDT","USDC","FDUSD","TUSD","USDP","BUSD","DAI","EUR","TRY","BRL","ARS",
    "COP","MXN","CZK","PLN","RON","UAH","NGN","ZAR","GBP","AUD","JPY","RUB"
}

def get_json(url, tries=5):
    last = None
    for k in range(tries):
        try:
            req = urllib.request.Request(url, headers={"User-Agent":UA,"Accept":"application/json"})
            with urllib.request.urlopen(req, timeout=25) as r:
                return json.loads(r.read())
        except Exception as e:
            last = e
            time.sleep(0.5 * (k+1))
    raise last
def flatten(node):
    if isinstance(node, dict):
        if node.get("node") == "text":
            return node.get("text","")
        s = "".join(flatten(x) for x in node.get("child",[]) or [])
        if node.get("tag") in {"p","div","li","td","th","tr","table","br"}:
            return " " + s + " "
        return s
    if isinstance(node, list):
        return " ".join(flatten(x) for x in node)
    return ""

def fetch_titles():
    out = []
    seen = False
    for page in range(1,120):
        obj = get_json(f"{LIST}?type=1&pageNo={page}&pageSize=50&catalogId=48")
        arts = (((obj.get("data") or {}).get("catalogs") or [{}])[0].get("articles") or [])
        if not arts:
            break
        ds = [datetime.datetime.fromtimestamp(a["releaseDate"]/1000,UTC) for a in arts]
        for a,dt in zip(arts,ds):
            if dt.year in (2023,2024) and "Trading Bots" in a.get("title",""):
                out.append(a)
        if any(dt.year in (2023,2024) for dt in ds):
            seen = True
        if seen and min(ds).year < 2023:
            break
        time.sleep(0.03)
    return out
def parse_article(a):
    obj = get_json(f"{DETAIL}?articleCode={a['code']}")
    data = (obj or {}).get("data") or {}
    pub_ms = int(data["publishDate"])
    body = data.get("body","{}")
    root = json.loads(body) if isinstance(body,str) else body
    text = " ".join(flatten(root).replace("\xa0"," ").split())

    m_bot = re.search(r"(?i)(?:enable|enabled)\s+Trading Bots services|Trading Bots services for|enable\s+Spot Grid", text)
    if not m_bot:
        return None, {"code":a["code"],"title":data.get("title",a.get("title","")),"error":"bot_section_not_found"}

    bot_start = m_bot.start()
    tail = text[bot_start:]
    stop = len(tail)
    for marker in ["Start Trading", "Notes:", "Notes :", "Thank you for your support"]:
        i = tail.find(marker)
        if i >= 0:
            stop = min(stop,i)
    bot_text = tail[:stop]
    bot_pairs = re.findall(r"\b([A-Z0-9]{2,24})/([A-Z0-9]{2,24})\b", bot_text)

    new_text = text[:bot_start]
    m_new = re.search(r"(?i)open trading for", new_text)
    if m_new:
        new_text = new_text[m_new.start():]
    else:
        new_text = ""
    new_pairs = re.findall(r"\b([A-Z0-9]{2,24})/([A-Z0-9]{2,24})\b", new_text)
    bot_bases = sorted({b for b,q in bot_pairs if b not in EXCLUDED_BASES})
    new_bases = {b for b,q in new_pairs}
    clean = [b for b in bot_bases if b not in new_bases]
    rec = {
        "code":a["code"],
        "title":data.get("title",a.get("title","")),
        "publish_ms":pub_ms,
        "publish":datetime.datetime.fromtimestamp(pub_ms/1000,UTC).isoformat(),
        "bot_pairs":sorted(set("/".join(x) for x in bot_pairs)),
        "new_pairs":sorted(set("/".join(x) for x in new_pairs)),
        "bot_bases":bot_bases,
        "new_pair_bases":sorted(new_bases),
        "clean_existing_bot_bases":clean,
        "bot_excerpt":bot_text[:2500],
    }
    return rec, None

titles = fetch_titles()
articles = []
errors = []
events = []
for i,a in enumerate(titles,1):
    try:
        rec,err = parse_article(a)
        if err:
            errors.append(err)
            continue
        articles.append(rec)
        for base in rec["clean_existing_bot_bases"]:
            events.append({
                "article_code":rec["code"],"article_title":rec["title"],
                "publish_ms":rec["publish_ms"],"publish":rec["publish"],
                "asset":base,"symbol":base+"USDT","side":"LONG"
            })
        print("ARTICLE",i,len(titles),rec["publish"][:10],len(rec["clean_existing_bot_bases"]),rec["clean_existing_bot_bases"],flush=True)
    except Exception as e:
        errors.append({"code":a.get("code"),"title":a.get("title"),"error":repr(e)})
    time.sleep(0.03)
(ROOT/"inputs/titles.json").write_text(json.dumps(titles,ensure_ascii=False,indent=2))
(ROOT/"inputs/articles.json").write_text(json.dumps(articles,ensure_ascii=False,indent=2))
(ROOT/"inputs/raw_events.json").write_text(json.dumps(events,ensure_ascii=False,indent=2))
(ROOT/"inputs/parse_errors.json").write_text(json.dumps(errors,ensure_ascii=False,indent=2))

summary = {
    "title_count":len(titles),
    "parsed_articles":len(articles),
    "parse_errors":len(errors),
    "raw_events":len(events),
    "raw_unique_tokens":len({e["asset"] for e in events}),
    "independent_batches":len({e["article_code"] for e in events}),
    "post_event_returns_read":False
}
(ROOT/"results/audit_summary.json").write_text(json.dumps(summary,ensure_ascii=False,indent=2))
print(json.dumps(summary,ensure_ascii=False,indent=2))
print("TOKENS",sorted({e["asset"] for e in events}))
