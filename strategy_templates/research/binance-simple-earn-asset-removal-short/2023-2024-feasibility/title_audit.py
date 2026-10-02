import urllib.request, json, datetime, time, re
from pathlib import Path

UTC=datetime.timezone.utc
UA='Mozilla/5.0'
LIST='https://www.binance.com/bapi/composite/v1/public/cms/article/list/query'
DETAIL='https://www.binance.com/bapi/composite/v1/public/cms/article/detail/query'
ROOT=Path('strategy_templates/research/binance-simple-earn-asset-removal-short/2023-2024-feasibility')

def get(url):
    req=urllib.request.Request(url,headers={'User-Agent':UA,'Accept':'application/json'})
    return json.loads(urllib.request.urlopen(req,timeout=20).read())

def flat(n):
    if isinstance(n,dict):
        if n.get('node')=='text': return n.get('text','')
        s=''.join(flat(x) for x in n.get('child',[]) or [])
        return (' '+s+' ') if n.get('tag') in {'p','div','td','th','tr','table','li','br'} else s
    if isinstance(n,list): return ' '.join(flat(x) for x in n)
    return ''
rows=[]
seen=False
for page in range(1,150):
    try:
        o=get(f'{LIST}?type=1&pageNo={page}&pageSize=50&catalogId=49')
    except Exception:
        time.sleep(.7); continue
    cats=((o.get('data') or {}).get('catalogs') or [])
    arts=(cats[0].get('articles') if cats else []) or []
    if not arts: break
    dates=[datetime.datetime.fromtimestamp(a['releaseDate']/1000,UTC) for a in arts]
    for a,dt in zip(arts,dates):
        if dt.year in (2023,2024) and 'Simple Earn' in a.get('title',''):
            rows.append({'release_time':a['releaseDate'],'code':a['code'],'title':a.get('title','')})
    if any(dt.year in (2023,2024) for dt in dates): seen=True
    if seen and min(dates).year<2023: break
    time.sleep(.03)

direct=re.compile(r'(remove|removal|delist|cease|discontinu|close|suspend|redeem)',re.I)
broad=re.compile(r'(update|notice|support|migration|redeem|redemption|cease|close|suspend|remove|delist|terminate|discontinue)',re.I)
direct_rows=[x for x in rows if direct.search(x['title'])]
candidate_rows=[x for x in rows if broad.search(x['title'])]
details=[]
for x in candidate_rows:
    d=get(f'{DETAIL}?articleCode={x["code"]}')
    data=d.get('data') or {}
    body=data.get('body','{}')
    root=json.loads(body) if isinstance(body,str) else (body or {})
    txt=' '.join(flat(root).replace('\xa0',' ').split())
    details.append({**x,'body_text':txt})
    time.sleep(.05)

(ROOT/'inputs/simple_earn_titles_2023_2024.json').write_text(
    json.dumps(rows,ensure_ascii=False,indent=2))
(ROOT/'inputs/direct_title_candidates.json').write_text(
    json.dumps(direct_rows,ensure_ascii=False,indent=2))
(ROOT/'inputs/broad_review_candidates.json').write_text(
    json.dumps(details,ensure_ascii=False,indent=2))

print('TOTAL_SIMPLE_EARN_TITLES',len(rows))
print('DIRECT_TITLE_CANDIDATES',len(direct_rows))
print('BROAD_REVIEW_CANDIDATES',len(details))
for x in details:
    print(x['release_time'],x['code'],x['title'])
