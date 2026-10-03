import json,re,time,urllib.request
from pathlib import Path
ROOT=Path('strategy_templates/research/binance-range-bound-product-underlying/2023-feasibility')
SRC=Path('strategy_templates/research/binance-corporate-action-support-long/2023-2024-feasibility/inputs/all_titles_2023_2024.json')
U='https://www.binance.com/bapi/composite/v1/public/cms/article/detail/query'
UA='Mozilla/5.0'
titles=[r for r in json.load(open(SRC)) if 'Range Bound' in r.get('title','')]

def get(code):
  last=None
  for i in range(5):
    try:
      req=urllib.request.Request(f'{U}?articleCode={code}',headers={'User-Agent':UA,'Accept':'application/json'})
      return json.loads(urllib.request.urlopen(req,timeout=25).read())['data']
    except Exception as e:
      last=e; time.sleep(.5*(i+1))
  raise last

def flat(n):
  if isinstance(n,dict):
    if n.get('node')=='text': return n.get('text','')
    s=''.join(flat(x) for x in n.get('child',[]) or [])
    return (' '+s+' ') if n.get('tag') in {'p','div','td','th','tr','table','li','ul','ol','br'} else s
  if isinstance(n,list): return ' '.join(flat(x) for x in n)
  return ''

rows=[]; all_assets=set(); errors=[]
for a in titles:
  try:
    x=get(a['code']); b=json.loads(x.get('body','{}')) if isinstance(x.get('body'),str) else x.get('body',{})
    t=' '.join(flat(b).replace('&nbsp;',' ').replace('\xa0',' ').split())
    # Explicit product table / explanatory sentence in every weekly batch.
    assets=set()
    # Limit to body sections describing Range Bound underlying assets / choose-product examples.
    for m in re.finditer(r'(?i)(Range Bound Underlying Asset|Choose the Range Bound product)[^.!]{0,900}',t):
      seg=m.group(0)
      assets.update(re.findall(r'\b(BTC|ETH|BNB|XRP|ADA|SOL|DOGE|LTC|BCH|DOT|AVAX|LINK)\b',seg))
    # Launch article has no fixed table; it is product introduction, not an underlying-addition event.
    rows.append({'code':a['code'],'title':x.get('title',a['title']),'publish_ms':x.get('publishDate',a['release_time']),'assets':sorted(assets)})
    all_assets.update(assets)
    print(a['code'],sorted(assets),flush=True)
  except Exception as e:
    errors.append({'code':a['code'],'error':str(e)})
(ROOT/'inputs/articles.json').write_text(json.dumps(rows,ensure_ascii=False,indent=2))
summary={'articles':len(rows),'parse_errors':len(errors),'unique_explicit_underlyings':sorted(all_assets),
         'unique_underlying_count':len(all_assets),'coverage_gate_min_unique_tokens':8,
         'coverage_gate_pass':len(all_assets)>=8,'post_event_returns_read':False}
(ROOT/'results/summary.json').write_text(json.dumps(summary,indent=2))
print(json.dumps(summary,indent=2))
