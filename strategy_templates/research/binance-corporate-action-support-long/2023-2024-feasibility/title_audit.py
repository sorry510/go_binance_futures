import urllib.request,json,datetime,time,re
from pathlib import Path
UTC=datetime.timezone.utc
ROOT=Path('strategy_templates/research/binance-corporate-action-support-long/2023-2024-feasibility')
LIST='https://www.binance.com/bapi/composite/v1/public/cms/article/list/query'
UA='Mozilla/5.0'
rows=[]; matches=[]; seen=False
pat=re.compile(r'(?i)(token swap|migration|rebranding|redenomination)')
for page in range(1,110):
    req=urllib.request.Request(f'{LIST}?type=1&pageNo={page}&pageSize=50&catalogId=49',headers={'User-Agent':UA,'Accept':'application/json'})
    o=json.loads(urllib.request.urlopen(req,timeout=20).read())
    aa=((((o.get('data') or {}).get('catalogs') or [{}])[0]).get('articles') or [])
    if not aa: break
    ds=[datetime.datetime.fromtimestamp(a['releaseDate']/1000,UTC) for a in aa]
    for a,dt in zip(aa,ds):
        if dt.year in (2023,2024):
            rec={'release_time':a['releaseDate'],'code':a['code'],'title':a.get('title','')}
            rows.append(rec)
            if pat.search(rec['title']): matches.append(rec)
    if any(dt.year in (2023,2024) for dt in ds): seen=True
    if seen and min(ds).year<2023: break
    time.sleep(.03)
(ROOT/'inputs/all_titles_2023_2024.json').write_text(json.dumps(rows,ensure_ascii=False,indent=2))
(ROOT/'inputs/title_candidates.json').write_text(json.dumps(matches,ensure_ascii=False,indent=2))
print('ALL_TITLES',len(rows),'CANDIDATES',len(matches))
for x in matches: print(datetime.datetime.fromtimestamp(x['release_time']/1000,UTC).date(),x['code'],x['title'],sep=' | ')
