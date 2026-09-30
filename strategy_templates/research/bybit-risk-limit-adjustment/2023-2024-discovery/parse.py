import urllib.request,re,json,time,datetime
from pathlib import Path
UTC=datetime.timezone.utc;UA='Mozilla/5.0'
ART=json.load(open('/tmp/bybit_risk_limit_articles.json'))
def fetch(path):
    if path.startswith('/article/'):path='/en'+path
    u='https://announcements.bybit.com'+path
    for k in range(4):
        try:
            h=urllib.request.urlopen(urllib.request.Request(u,headers={'User-Agent':UA}),timeout=20).read().decode('utf-8','ignore')
            m=re.search(r'<script id="__NEXT_DATA__" type="application/json">(.*?)</script>',h,re.S)
            return json.loads(m.group(1))['props']['pageProps']['articleDetail']
        except Exception:
            time.sleep(.5*(k+1))
    return None
def text(n):
    if isinstance(n,dict):
        s=n.get('text','') if isinstance(n.get('text'),str) else ''
        return s+''.join(text(x) for x in n.get('children',[]) or [])
    if isinstance(n,list):return ''.join(text(x) for x in n)
    return ''
def find_type(n,typ,out):
    if isinstance(n,dict):
        if n.get('type')==typ:out.append(n)
        for v in n.values():find_type(v,typ,out)
    elif isinstance(n,list):
        for x in n:find_type(x,typ,out)
def num(s):
    s=str(s).replace(',','').replace('%','').strip()
    m=re.search(r'-?\d+(?:\.\d+)?',s)
    return float(m.group()) if m else None
events=[];article_stats=[]
for i,a in enumerate(ART,1):
    d=fetch(a.get('url',''))
    if not d:
        article_stats.append({'title':a.get('title'),'status':'fetch_failed'});continue
    root=((d.get('content') or {}).get('json') or {})
    tabs=[];find_type(root,'table',tabs)
    parsed=0
    for t in tabs:
        rows=[];find_type(t,'tr',rows)
        grid=[]
        for r in rows:
            cells=[re.sub(r'\s+',' ',text(c)).strip() for c in (r.get('children') or [])]
            grid.append(cells)
        if len(grid)<4:continue
        syms=re.findall(r'([0-9A-Z]+USDT)',grid[0][0] if grid[0] else '')
        data=None
        for rr in grid[3:8]:
            if len(rr)>=8 and num(rr[0]) is not None and num(rr[3]) is not None and num(rr[4]) is not None and num(rr[7]) is not None:
                data=rr;break
        if not syms or not data:continue
        oldcap,oldlev,newcap,newlev=num(data[0]),num(data[3]),num(data[4]),num(data[7])
        if None in (oldcap,oldlev,newcap,newlev):continue
        if newlev>oldlev:cls='expansion';basis='leverage'
        elif newlev<oldlev:cls='contraction';basis='leverage'
        elif newcap>oldcap:cls='expansion';basis='notional_cap'
        elif newcap<oldcap:cls='contraction';basis='notional_cap'
        else:cls='unchanged';basis='none'
        for s in syms:
            events.append({'article_objectID':a.get('objectID'),'article_title':a.get('title'),
                           'event_ts':a.get('date_timestamp') or a.get('publish_time'),
                           'bybit_symbol':s,'classification':cls,'classification_basis':basis,
                           'old_first_cap':oldcap,'new_first_cap':newcap,
                           'old_first_leverage':oldlev,'new_first_leverage':newlev,
                           'url':a.get('url')})
            parsed+=1
    article_stats.append({'title':a.get('title'),'tables':len(tabs),'parsed_symbol_events':parsed,
                          'description':d.get('description','')})
    print('ARTICLE',i,'/',len(ART),'tables',len(tabs),'parsed',parsed,'|',a.get('title'),flush=True)
uniq={}
for x in events:
    uniq[(x['article_objectID'],x['bybit_symbol'])]=x
events=sorted(uniq.values(),key=lambda x:(x['event_ts'] or 0,x['bybit_symbol']))
from collections import Counter
print('PARSED_EVENTS',len(events),'ARTICLES_PARSED',len(set(x['article_objectID'] for x in events)),flush=True)
print('CLASS',Counter(x['classification'] for x in events),'BASIS',Counter(x['classification_basis'] for x in events),flush=True)
print('UNPARSED_ARTICLES',sum(x.get('parsed_symbol_events',0)==0 for x in article_stats),flush=True)
for x in article_stats:
    if x.get('parsed_symbol_events',0)==0:print('UNPARSED',x['title'],'|',x.get('description',''),flush=True)
Path('/tmp/bybit_risk_limit_parsed_events.json').write_text(json.dumps(events,ensure_ascii=False,indent=2))
Path('/tmp/bybit_risk_limit_article_stats.json').write_text(json.dumps(article_stats,ensure_ascii=False,indent=2))
