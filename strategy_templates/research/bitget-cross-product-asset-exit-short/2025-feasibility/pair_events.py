import json,re,html,datetime
from pathlib import Path
# spot token events from official cached bodies
spots=json.load(open('/tmp/bitget_2025_spot_articles.json'))
sroot=Path('/tmp/bitget_spot2025_cache')
spot_events=[]
for a in spots:
    aid=a['href'].split('/')[-1];p=sroot/(aid+'.html')
    if not p.exists():continue
    body=html.unescape(p.read_text(errors='ignore')).upper()
    bases=[]
    for b,q in re.findall(r'(?<![A-Z0-9])([A-Z0-9]{1,30})\s*/\s*(USDT|USDC)(?![A-Z0-9])',body):
        if b not in ('FUTURES','CONVERT') and b not in bases:bases.append(b)
    if not a.get('datePublished'):continue
    dt=datetime.datetime.fromisoformat(a['datePublished'])
    for b in bases:
        spot_events.append({'base':b,'spot_ts':dt.timestamp(),'spot_iso':dt.astimezone(datetime.timezone.utc).isoformat(),
                            'spot_url':'https://www.bitget.com'+a['href'],'spot_title':a['headline']})
# futures from official section titles; only explicit USDT contracts
sec=json.load(open('/tmp/bitget_2025_section_items.json'))['futures']
fut_events=[]
for a in sec:
    t=a['title'].upper().replace(' ','')
    bases=[]
    for b in re.findall(r'(?<![A-Z0-9])([A-Z0-9]{1,30})USDT(?![A-Z0-9])',t):
        if b not in bases:bases.append(b)
    # section time is official display, UTC+8
    dt=datetime.datetime.strptime(a['section_date'],'%Y-%m-%d %H:%M').replace(tzinfo=datetime.timezone(datetime.timedelta(hours=8)))
    for b in bases:
        fut_events.append({'base':b,'futures_ts_approx':dt.timestamp(),'futures_section_iso':dt.astimezone(datetime.timezone.utc).isoformat(),
                           'futures_url':'https://www.bitget.com'+a['href'],'futures_title':a['title']})
# pair within 7d, one best nearest spot for each future base
pairs=[]
for f in fut_events:
    q=[s for s in spot_events if s['base']==f['base'] and abs(s['spot_ts']-f['futures_ts_approx'])<=7*86400]
    if not q:continue
    s=min(q,key=lambda z:abs(z['spot_ts']-f['futures_ts_approx']))
    pairs.append({**f,**s,'gap_hours':abs(s['spot_ts']-f['futures_ts_approx'])/3600})
# de-dupe same base earliest matched futures
best={}
for x in sorted(pairs,key=lambda z:z['futures_ts_approx']):
    best.setdefault(x['base'],x)
pairs=list(best.values())
Path('/tmp/bitget_cross_product_pairs_2025_prelim.json').write_text(json.dumps(pairs,ensure_ascii=False,indent=2))
print('SPOT_TOKEN_EVENTS',len(spot_events),'FUT_TOKEN_EVENTS',len(fut_events),'MATCHED',len(pairs))
for x in pairs:print(x['base'],'gap_h',round(x['gap_hours'],1),'| F',x['futures_title'],'| S',x['spot_title'])
