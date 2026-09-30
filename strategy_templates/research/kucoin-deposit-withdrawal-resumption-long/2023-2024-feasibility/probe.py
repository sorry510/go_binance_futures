import json, re, urllib.parse, urllib.request, datetime

H={"User-Agent":"Mozilla/5.0","Accept":"application/json"}
U="https://www.kucoin.com/_api/cms/articles"
items={}
for page in range(1,14):
    url=U+"?"+urllib.parse.urlencode({"page":page,"pageSize":100,"keyword":"Deposit and Withdrawal Services"})
    obj=json.loads(urllib.request.urlopen(urllib.request.Request(url,headers=H),timeout=20).read())
    for x in obj.get("items") or []:
        items[x["id"]]=x
for x in items.values():
    ts=x.get("first_publish_at") or x.get("publish_ts")
    if not ts: continue
    d=datetime.datetime.fromtimestamp(int(ts),datetime.timezone.utc)
    t=(x.get("title") or "").strip()
    resume=re.search(r"resum|reopen|re-open|now open|restor|available again|recovery",t,re.I)
    if d.year in (2023,2024) and resume and re.search(r"deposit|withdraw",t,re.I):
        print(d.isoformat(),x["id"],t)
