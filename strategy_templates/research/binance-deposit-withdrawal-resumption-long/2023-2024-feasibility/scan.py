import urllib.request, json, datetime, time, re
from pathlib import Path
UTC=datetime.timezone.utc
UA='Mozilla/5.0'
U='https://www.binance.com/bapi/composite/v1/public/cms/article/list/query'
ROOT=Path('strategy_templates/research/binance-deposit-withdrawal-resumption-long/2023-2024-feasibility')
rows=[]
seen=False
for page in range(1,100):
    try:
        req=urllib.request.Request(f'{U}?type=1&pageNo={page}&pageSize=50&catalogId=157',headers={'User-Agent':UA,'Accept':'application/json'})
        o=json.loads(urllib.request.urlopen(req,timeout=20).read())
    except Exception:
        time.sleep(.7); continue
    cats=((o.get('data') or {}).get('catalogs') or [])
    arts=(cats[0].get('articles') if cats else []) or []
    if not arts: break
    dates=[datetime.datetime.fromtimestamp(a['releaseDate']/1000,UTC) for a in arts]
    for a,dt in zip(arts,dates):
        title=a.get('title','')
        if dt.year in (2023,2024) and re.search(r'(resum|reopen|restor)',title,re.I) and re.search(r'(deposit|withdraw|network)',title,re.I):
            rows.append({'release_time':a['releaseDate'],'code':a['code'],'title':title})
    if any(dt.year in (2023,2024) for dt in dates): seen=True
    if seen and min(dates).year<2023: break
    time.sleep(.03)
(ROOT/'inputs/candidates.json').write_text(json.dumps(rows,ensure_ascii=False,indent=2))
print('COUNT',len(rows))
for x in rows: print(x['release_time'],x['code'],x['title'])
