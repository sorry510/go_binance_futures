import json,re,html,datetime,urllib.request,urllib.error,time,csv,io,zipfile
from pathlib import Path
UTC=datetime.timezone.utc;UA='Mozilla/5.0'
posts=json.load(open('/tmp/kraken_asset_listing_posts.json'))
EXCLUDE=re.compile(r'(Expanded margin|OTC|deposits and withdrawals|deposit[s]? and withdrawal|available on .* network|available for funding|how to get listed|accelerating new listings|\bUSA\b|\bCanada\b|United Kingdom)',re.I)
INCLUDE=re.compile(r'(Trading for .*starts|available for trading|open for trading|live and available for trading|trading starts today|trading starts now|trading available now|tokens available for trading|now available for trading)',re.I)
STOP={'USD','EUR','USDT','USDC','CAD','GBP','BTC','ETH','PAIR','ASSET','TOKEN','TRADING','KRAKEN','PRO','APP','OTC','RFQ','USA'}
def title_text(p):return html.unescape(re.sub('<[^>]+>','',p['title']['rendered']))
def strip_html(x):
    x=re.sub(r'<script.*?</script>|<style.*?</style>',' ',x,flags=re.I|re.S)
    return html.unescape(re.sub(r'<[^>]+>',' ',x))
def assets_from_post(p):
    title=title_text(p); body=p['content']['rendered']; out=set()
    # Only title-level parenthesized tickers are authoritative. Do not mine
    # arbitrary body parentheses: project/network abbreviations can collide
    # with real Binance symbols.
    for t in re.findall(r'\(([A-Z0-9]{2,15})\)',title):
        if t not in STOP:out.add(t)
    for t in re.findall(r'\b[A-Z][A-Z0-9]{1,14}\b',title):
        if t not in STOP:out.add(t)
    for tr in re.findall(r'<tr\b[^>]*>(.*?)</tr>',body,flags=re.I|re.S):
        m=re.search(r'<t[dh]\b[^>]*>(.*?)</t[dh]>',tr,flags=re.I|re.S)
        if not m:continue
        txt=re.sub(r'\s+',' ',strip_html(m.group(1))).strip().replace('$','')
        if re.fullmatch(r'[A-Z0-9]{2,15}',txt) and txt not in STOP:out.add(txt)
    return sorted(out)
universe=[]
for p in posts:
    title=title_text(p)
    if EXCLUDE.search(title) or not INCLUDE.search(title):continue
    assets=assets_from_post(p)
    if not assets:continue
    pub=datetime.datetime.fromisoformat(p['date_gmt']).replace(tzinfo=UTC)
    for a in assets:
        universe.append({'post_id':p['id'],'url':p['link'],'title':title,'publish_ms':int(pub.timestamp()*1000),'publish_utc':pub.isoformat(),'asset':a})
print('POSTS_INCLUDED',len(set(x['post_id'] for x in universe)),'ASSET_EVENTS_RAW',len(universe),flush=True)
for x in universe:print('RAW',x['publish_utc'],x['asset'],x['title'],flush=True)
Path('/tmp/kraken_listing_universe.json').write_text(json.dumps(universe,ensure_ascii=False,indent=2))
VISION='https://data.binance.vision/data/futures/um/monthly/klines'
CACHE=Path('/tmp/kraken_listing_cache');CACHE.mkdir(exist_ok=True)
def dt(ms):return datetime.datetime.fromtimestamp(ms/1000,UTC)
def mon(ms):return dt(ms).strftime('%Y-%m')
def prevmon(ms):
 d=dt(ms);return f'{d.year-1}-12' if d.month==1 else f'{d.year:04d}-{d.month-1:02d}'
def sub2(ms):
 d=dt(ms)
 try:d=d.replace(year=d.year-2)
 except ValueError:d=d.replace(year=d.year-2,day=28)
 return int(d.timestamp()*1000)
def fetch(sym,mo):
 p=CACHE/f'{sym}-{mo}.zip'
 if p.exists():return p
 u=f'{VISION}/{sym}/1h/{sym}-1h-{mo}.zip'
 for k in range(4):
  try:b=urllib.request.urlopen(urllib.request.Request(u,headers={'User-Agent':UA}),timeout=20).read();p.write_bytes(b);return p
  except urllib.error.HTTPError as e:
   if e.code==404:p.write_bytes(b'');return p
   time.sleep(.4*(k+1))
  except Exception:time.sleep(.4*(k+1))
 p.write_bytes(b'');return p
def rows(sym,mo):
 p=fetch(sym,mo)
 if not p.exists() or p.stat().st_size==0:return []
 z=zipfile.ZipFile(p);fn=z.namelist()[0];out=[]
 for r in csv.reader(io.TextIOWrapper(z.open(fn))):
  if not r or not r[0].isdigit():continue
  t=int(r[0]);t=t//1000 if t>10**15 else t
  out.append((t,float(r[7]) if len(r)>7 else 0.0))
 return out
elig=[]
for i,e in enumerate(universe,1):
    # Kraken ticker -> Binance conventional USDT symbol; 1000-prefix not guessed.
    sym=e['asset']+'USDT';evt=e['publish_ms'];cut=sub2(evt)
    hist=rows(sym,mon(cut))
    x=dict(e);x['symbol']=sym
    if not hist or hist[0][0]>cut:
        x.update(eligible=False,reason='history_lt_2y_or_missing');elig.append(x);continue
    hour=(evt//3600000)*3600000;start=hour-24*3600000;end=hour-1
    prior=rows(sym,prevmon(evt))+rows(sym,mon(evt));prior=[r for r in prior if start<=r[0]<=end];qv=sum(r[1] for r in prior)
    x['bars_24h']=len(prior);x['quote_volume_24h']=qv
    if len(prior)<24:x.update(eligible=False,reason='missing_24h_bars')
    elif qv<5_000_000:x.update(eligible=False,reason='quote_volume_lt_5m')
    else:x.update(eligible=True,reason='ok')
    elig.append(x)
from collections import Counter
good=[x for x in elig if x['eligible']]
print('ELIGIBLE',len(good),'POSTS',len(set(x['post_id'] for x in good)),'SYMS',len(set(x['symbol'] for x in good)),flush=True)
print('REASONS',Counter(x['reason'] for x in elig),flush=True)
for y in [2023,2024]:
 q=[x for x in good if dt(x['publish_ms']).year==y];print('YEAR',y,len(q),[x['symbol'] for x in q],flush=True)
for x in good:print('OK',x['publish_utc'],x['symbol'],round(x['quote_volume_24h']),x['title'],flush=True)
Path('/tmp/kraken_listing_eligibility.json').write_text(json.dumps(elig,ensure_ascii=False,indent=2))
