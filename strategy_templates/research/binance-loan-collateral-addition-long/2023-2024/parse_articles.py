import urllib.request,urllib.error,json,datetime,time,re
from pathlib import Path
UTC=datetime.timezone.utc;UA='Mozilla/5.0'
LIST='https://www.binance.com/bapi/composite/v1/public/cms/article/list/query'
DETAIL='https://www.binance.com/bapi/composite/v1/public/cms/article/detail/query'
def get_json(u):
    last=None
    for k in range(6):
        try:
            req=urllib.request.Request(u,headers={'User-Agent':UA,'Accept':'application/json'})
            return json.loads(urllib.request.urlopen(req,timeout=20).read())
        except Exception as e:last=e;time.sleep(.8*(k+1))
    raise last
def flat(n):
    if isinstance(n,dict):
        if n.get('node')=='text':return n.get('text','')
        s=''.join(flat(x) for x in n.get('child',[]) or [])
        if n.get('tag') in {'p','div','td','th','tr','table','li','br'}:return ' '+s+' '
        return s
    if isinstance(n,list):return ' '.join(flat(x) for x in n)
    return ''
def table_rows(n,out):
    if isinstance(n,dict):
        if n.get('tag')=='tr':
            cells=[flat(x).replace('\xa0',' ').strip() for x in n.get('child',[]) or [] if isinstance(x,dict) and x.get('tag') in ('td','th')]
            if cells:out.append(cells)
        for x in n.get('child',[]) or []:table_rows(x,out)
    elif isinstance(n,list):
        for x in n:table_rows(x,out)
def toks_from_segment(s):
    par=[x for x in re.findall(r'\(([A-Z0-9]{2,20})\)',s) if x not in {'USDT','USDC','BUSD'}]
    if par:return list(dict.fromkeys(par))
    xs=re.findall(r'(?<![A-Z0-9])([A-Z][A-Z0-9]{1,19})(?![A-Z0-9])',s)
    bad={'NEW','VIP','LOAN','FLEXIBLE','ASSET','ASSETS','AND','THE','USD','USDT','USDC','BUSD','ETH','BTC'}
    return list(dict.fromkeys(x for x in xs if x not in bad))
def parse_collateral(root):
    out=[]
    rows=[];table_rows(root,rows)
    if rows:
        hdr=None
        for r in rows:
            for i,c in enumerate(r):
                if 'New Collateral Assets' in c:
                    hdr=i;break
            if hdr is not None:break
        if hdr is not None:
            for r in rows[1:]:
                if len(r)>hdr:out.extend(toks_from_segment(r[hdr]))
    txt=' '.join(flat(root).replace('\xa0',' ').split())
    pats=[
      r'New collateral assets:\s*(.*?)(?=Please Note|Please refer|To place|What Is|For More Information|Notes:|$)',
      r'as new loanable assets, and (.*?) as new collateral assets',
      r'new loanable assets, and (.*?) as new collateral assets',
      r'has added (.*?) as new collateral assets',
      r'adds? (.*?) as new collateral assets',
      r'and (.*?) as new collateral assets'
    ]
    for pat in pats:
        for m in re.finditer(pat,txt,re.I):
            out.extend(toks_from_segment(m.group(1)))
    return list(dict.fromkeys(out)),txt
arts=[];seen=False
for page in range(1,110):
    o=get_json(f'{LIST}?type=1&pageNo={page}&pageSize=50&catalogId=49')
    aa=((((o.get('data') or {}).get('catalogs') or [{}])[0]).get('articles') or [])
    if not aa:break
    ds=[datetime.datetime.fromtimestamp(a['releaseDate']/1000,UTC) for a in aa]
    for a,d in zip(aa,ds):
        if d.year in (2023,2024):
            seen=True
            if re.search(r'collateral asset',a.get('title',''),re.I):arts.append(a)
    if seen and min(ds).year<2023:break
    time.sleep(.04)
print('ARTICLES',len(arts),flush=True)
events=[];parsed=[]
for i,a in enumerate(arts,1):
    o=get_json(f'{DETAIL}?articleCode={a["code"]}');d=(o.get('data') or {})
    root=json.loads(d.get('body','{}')) if isinstance(d.get('body'),str) else (d.get('body') or {})
    toks,txt=parse_collateral(root)
    rec={'code':a['code'],'title':d.get('title',a.get('title','')),'release_time':int(d.get('publishDate') or a['releaseDate']),'tokens':toks,'opening':txt[:3000]}
    parsed.append(rec)
    for t in toks:events.append({'article_code':rec['code'],'article_title':rec['title'],'release_time':rec['release_time'],'ticker':t,'symbol':t+'USDT'})
    print('PARSE',i,'/',len(arts),'TOKENS',len(toks),toks,flush=True)
    time.sleep(.08)
Path('/tmp/collateral_articles.json').write_text(json.dumps(parsed,ensure_ascii=False,indent=2))
Path('/tmp/collateral_events.json').write_text(json.dumps(events,ensure_ascii=False,indent=2))
print('TOKEN_EVENTS',len(events),'UNIQUE',len(set(x['symbol'] for x in events)),'ZERO_ARTICLES',sum(not x['tokens'] for x in parsed),flush=True)
