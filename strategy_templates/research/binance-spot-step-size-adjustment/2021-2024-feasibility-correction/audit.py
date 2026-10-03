import json,re,time,urllib.request,datetime,decimal
from pathlib import Path

ROOT=Path("strategy_templates/research/binance-spot-step-size-adjustment/2021-2024-feasibility-correction")
H={"User-Agent":"Mozilla/5.0","Accept":"application/json"}
DETAIL="https://www.binance.com/bapi/composite/v1/public/cms/article/detail/query"
STABLE_FIAT={"USDT","USDC","BUSD","TUSD","FDUSD","DAI","USDP","USDS","USD","EUR","TRY","GBP","AUD","BRL","BIDR","IDRT","RUB","UAH","NGN","PLN","RON","ARS","ZAR","CZK","JPY","PAX"}

articles=[
 {"code":"6925d618ab6b47e2936cc4614eaad64b","title":"Updates on Tick Size and Step Size for Spot Trading Pairs","publish":"2021-08-12T15:57:03.751000+00:00"},
 {"code":"d2a7ae9f33d44bf085782269afd4ddfc","title":"Updates on Step Size for Spot Trading Pairs (2024-04-29)","publish_ms":1713765618078},
 {"code":"99bb1634643e454cae9246dc43236785","title":"Updates on Step Size for Spot Trading Pairs (2024-06-19)","publish_ms":1718175610311},
]

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

events=[]; audit=[]
for a in articles:
 o=get_json(f"{DETAIL}?articleCode={a['code']}");data=o.get("data") or {}
 body=data.get("body","{}");root=json.loads(body) if isinstance(body,str) else body
 txt=" ".join(flat(root).replace("\xa0"," ").replace("&nbsp;"," ").split())
 if a["code"]=="6925d618ab6b47e2936cc4614eaad64b":
  m=re.search(r"at\s+(2021-08-26)\s+(\d{1,2}:\d{2})\s+AM\s*\(UTC\)",txt,re.I)
  seg=txt.split("Trading Pair Step Size",1)[1] if "Trading Pair Step Size" in txt else ""
  et=int(datetime.datetime.fromisoformat(m.group(1)+"T"+m.group(2)+":00+00:00").timestamp()*1000) if m else 0
  segments=[(et,seg)]
 else:
  marks=list(re.finditer(r"By\s+(20\d\d-\d\d-\d\d)\s+(\d\d:\d\d)\s*\(UTC\)\s*:",txt,re.I))
  segments=[]
  for i,m in enumerate(marks):
   et=int(datetime.datetime.fromisoformat(m.group(1)+"T"+m.group(2)+":00+00:00").timestamp()*1000)
   segments.append((et,txt[m.end():marks[i+1].start() if i+1<len(marks) else len(txt)]))
 n=0
 for et,seg in segments:
  for pair,old,new in re.findall(r"\b([A-Z0-9]{2,24})/USDT\s+([0-9]+(?:\.[0-9]+)?)\s+([0-9]+(?:\.[0-9]+)?)\b",seg):
   pass
  # 2021 uses concatenated symbols, 2024 slash pairs.
  found=[]
  found += [(m[0],m[1],m[2]) for m in re.findall(r"\b([A-Z0-9]{2,24})/USDT\s+([0-9]+(?:\.[0-9]+)?)\s+([0-9]+(?:\.[0-9]+)?)\b",seg)]
  found += [(m[0][:-4],m[1],m[2]) for m in re.findall(r"\b([A-Z0-9]{2,24}USDT)\s+([0-9]+(?:\.[0-9]+)?)\s+([0-9]+(?:\.[0-9]+)?)\b",seg)]
  seen=set()
  for base,old,new in found:
   if base in seen or base in STABLE_FIAT or base.endswith("UP") or base.endswith("DOWN"):continue
   seen.add(base)
   aa,bb=decimal.Decimal(old),decimal.Decimal(new)
   if aa==bb:continue
   events.append({"article_code":a["code"],"article_title":data.get("title",a["title"]),
                  "effective_ms":et,"effective_utc":datetime.datetime.fromtimestamp(et/1000,datetime.timezone.utc).isoformat(),
                  "base":base,"symbol":base+"USDT","old_step":old,"new_step":new,
                  "direction":"LONG" if bb<aa else "SHORT"})
   n+=1
 audit.append({"code":a["code"],"title":data.get("title",a["title"]),"events":n,"text_prefix":txt[:3000]})
 print("ARTICLE",a["code"],"events",n,flush=True)

(ROOT/"inputs/articles_audit.json").write_text(json.dumps(audit,ensure_ascii=False,indent=2))
(ROOT/"inputs/events_raw.json").write_text(json.dumps(events,ensure_ascii=False,indent=2))
summary={"articles":len(articles),"raw_events":len(events),"raw_unique_tokens":len({x["base"] for x in events}),
         "raw_independent_articles":len({x["article_code"] for x in events}),"returns_read":False}
(ROOT/"results/audit_summary.json").write_text(json.dumps(summary,indent=2))
print(json.dumps(summary,indent=2))
for x in events:print("EVENT",x["effective_utc"],x["base"],x["old_step"],"->",x["new_step"],x["direction"])
