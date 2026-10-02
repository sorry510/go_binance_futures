import urllib.request, json, datetime, time, re
from pathlib import Path

UTC=datetime.timezone.utc
LIST='https://www.binance.com/bapi/composite/v1/public/cms/article/list/query'
ROOT=Path('strategy_templates/research/binance-loan-asset-removal-short/2023-2024-feasibility')
rows=[]
seen=False

for page in range(1,110):
    req=urllib.request.Request(
        f'{LIST}?type=1&pageNo={page}&pageSize=50&catalogId=49',
        headers={'User-Agent':'Mozilla/5.0','Accept':'application/json'})
    o=json.loads(urllib.request.urlopen(req,timeout=20).read())
    articles=((((o.get('data') or {}).get('catalogs') or [{}])[0]).get('articles') or [])
    if not articles:
        break
    dates=[datetime.datetime.fromtimestamp(a['releaseDate']/1000,UTC) for a in articles]
    for a,dt in zip(articles,dates):
        if dt.year in (2023,2024) and re.search(r'loan',a.get('title',''),re.I):
            rows.append({
                'release_time':a['releaseDate'],
                'code':a['code'],
                'title':a.get('title','')
            })
    if any(dt.year in (2023,2024) for dt in dates):
        seen=True
    if seen and min(dates).year < 2023:
        break
    time.sleep(.03)

(ROOT/'inputs/loan_titles_2023_2024.json').write_text(
    json.dumps(rows,ensure_ascii=False,indent=2))
print('saved',len(rows))
