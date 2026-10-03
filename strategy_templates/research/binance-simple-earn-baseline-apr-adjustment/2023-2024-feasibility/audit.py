import json,re,time,urllib.request
from pathlib import Path
ROOT=Path("strategy_templates/research/binance-simple-earn-baseline-apr-adjustment/2023-2024-feasibility")
CANDS=json.loads((ROOT/"inputs/title_candidates.json").read_text())
DETAIL="https://www.binance.com/bapi/composite/v1/public/cms/article/detail/query"
H={"User-Agent":"Mozilla/5.0","Accept":"application/json"}

def get_json(url):
    last=None
    for i in range(6):
        try:
            req=urllib.request.Request(url,headers=H)
            with urllib.request.urlopen(req,timeout=25) as r:return json.loads(r.read())
        except Exception as e:
            last=e;time.sleep(.6*(i+1))
    raise last
def flat(n):
    if isinstance(n,dict):
        if n.get("node")=="text":return n.get("text","")
        s="".join(flat(x) for x in n.get("child",[]) or [])
        return (" "+s+" ") if n.get("tag") in {"p","div","td","th","tr","table","li","ul","ol","br"} else s
    if isinstance(n,list):return " ".join(flat(x) for x in n)
    return ""

out=[];errs=[]
for i,a in enumerate(CANDS,1):
    try:
        o=get_json(f"{DETAIL}?articleCode={a['code']}")
        d=o.get("data") or {}
        body=d.get("body","{}")
        try:root=json.loads(body) if isinstance(body,str) else body
        except:root={}
        txt=re.sub(r"\s+"," ",flat(root).replace("\xa0"," ").replace("&nbsp;"," ")).strip()
        out.append({"code":a["code"],"release_time":a["release_time"],"title":d.get("title",a["title"]),
                    "publishDate":d.get("publishDate"),"text":txt})
        print("ARTICLE",i,len(CANDS),a["code"],"chars",len(txt),flush=True)
    except Exception as e:
        errs.append({"code":a["code"],"title":a["title"],"error":str(e)})
        print("ERROR",a["code"],e,flush=True)
(ROOT/"inputs/articles.json").write_text(json.dumps(out,ensure_ascii=False,indent=2))
(ROOT/"inputs/fetch_errors.json").write_text(json.dumps(errs,ensure_ascii=False,indent=2))
print("FETCHED",len(out),"ERRORS",len(errs))
