import json,re,datetime
from pathlib import Path
SRC=Path('/tmp/bitget_futures_delist_candidates.json')
a=json.load(open(SRC))
CORP={
 'FET':'merge_to_ASI','OCEAN':'merge_to_ASI','AGIX':'merge_to_ASI',
 'RNDR':'rebrand_to_RENDER','GAL':'rebrand_to_G','FRONT':'rebrand_to_SLF',
 'KLAY':'rebrand_to_KAIA','TOMO':'rebrand_to_VIC','COCOS':'rebrand_to_COMBO',
 '10000AIDOGE':'contract_denomination_lifecycle'
}
rows=[];excluded=[]
for x in a:
 y=int(x['datePublished'][:4])
 if y not in (2023,2024): continue
 text=(x['title']+' '+x.get('description','')).upper().replace('10000 LADYS','10000LADYS')
 syms=[]
 for m in re.finditer(r'(?<![A-Z0-9])([A-Z0-9]{2,30})\s*USDT(?![A-Z0-9])',text):
  s=m.group(1)+'USDT'
  if s not in syms: syms.append(s)
 # WAVES article omits USDT in title/desc phrasing
 if 'DELISTING OF WAVES FUTURES' in text and 'WAVESUSDT' not in syms: syms.append('WAVESUSDT')
 # only direct USDT-margined contracts
 for s in syms:
  base=s[:-4]
  reason=None
  if base in CORP: reason=CORP[base]
  if reason:
   excluded.append({'symbol':s,'reason':reason,**x}); continue
  dt=datetime.datetime.fromisoformat(x['datePublished'])
  rows.append({'bitget_symbol':s,'event_ts':dt.timestamp(),'event_iso_utc':dt.astimezone(datetime.timezone.utc).isoformat(),
               'article_url':'https://www.bitget.com'+x['href'],'title':x['title'],'description':x.get('description','')})
# de-dupe same ticker: earliest independent announcement
best={}
dups=[]
for x in sorted(rows,key=lambda z:z['event_ts']):
 s=x['bitget_symbol']
 if s in best:
  dups.append({'symbol':s,'reason':'duplicate_later_announcement','kept_event_iso_utc':best[s]['event_iso_utc'],**x})
 else: best[s]=x
rows=list(best.values())
Path('/tmp/bitget_delist_events_normalized.json').write_text(json.dumps(rows,ensure_ascii=False,indent=2))
Path('/tmp/bitget_delist_exclusions.json').write_text(json.dumps(excluded+dups,ensure_ascii=False,indent=2))
print('RAW_ARTICLES_2023_24',sum(x['datePublished'][:4] in ('2023','2024') for x in a))
print('EVENTS',len(rows),'UNIQUE',len({x['bitget_symbol'] for x in rows}))
print('EXCLUDED_CORP',len(excluded),'DUPS',len(dups))
for x in rows: print(x['event_iso_utc'],x['bitget_symbol'],'|',x['title'])
print('EXCLUSIONS')
for x in excluded+dups: print(x['symbol'],x['reason'],'|',x['title'])
