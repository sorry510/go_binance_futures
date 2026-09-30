import urllib.request,urllib.error,json,time,re,datetime
from pathlib import Path
SRC='/Users/zhz/work/binance/go_binance_futures/strategy_templates/research/binance-loan-collateral-addition-long/2023-2024/inputs/articles.json'
ART=json.load(open(SRC)); UA='Mozilla/5.0'
DETAIL='https://www.binance.com/bapi/composite/v1/public/cms/article/detail/query'
def get_json(u):
    last=None
    for k in range(5):
        try:
            req=urllib.request.Request(u,headers={'User-Agent':UA,'Accept':'application/json'})
            return json.loads(urllib.request.urlopen(req,timeout=20).read())
        except Exception as e:last=e;time.sleep(.7*(k+1))
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
def toks(seg):
    par=[x for x in re.findall(r'\(([A-Z0-9]{2,20})\)',seg) if x not in {'USDT','USDC','BUSD','USD'}]
    if par:return list(dict.fromkeys(par))
    xs=re.findall(r'(?<![A-Z0-9])([A-Z][A-Z0-9]{1,19})(?![A-Z0-9])',seg)
    bad={'NEW','VIP','LOAN','LOANS','FLEXIBLE','RATE','ASSET','ASSETS','COLLATERAL','LOANABLE','AND','THE','USD','USDT','USDC','BUSD','ETH','BTC','BINANCE'}
    return list(dict.fromkeys(x for x in xs if x not in bad))
def parse_loanable(root):
    out=[]; rows=[]; table_rows(root,rows)
    if rows:
        # Header may place New Loanable Assets in one column.
        for hi,r in enumerate(rows):
            idx=[i for i,c in enumerate(r) if 'New Loanable Assets' in c]
            if not idx: continue
            j=idx[0]
            for rr in rows[hi+1:]:
                # Stop at a new independent header table.
                if any('New Loanable Assets' in c for c in rr):break
                if len(rr)>j:out.extend(toks(rr[j]))
            break
    txt=' '.join(flat(root).replace('\xa0',' ').split())
    pats=[
      r'has added the following new loanable assets:\s*(.*?)(?=Binance VIP Loan|New collateral assets|Please Note|Please refer|To place|What Is|For More Information|Notes:|$)',
      r'New loanable assets:\s*(.*?)(?=New collateral assets|Please Note|Please refer|To place|What Is|For More Information|Notes:|$)',
      r'as new loanable assets(?:,|;| and)\s*(.*?)(?=as new collateral assets|Please Note|Please refer|What Is|Notes:|$)',
    ]
    for pat in pats:
        for m in re.finditer(pat,txt,re.I):out.extend(toks(m.group(1)))
    return list(dict.fromkeys(out)),rows,txt
events=[];parsed=[]
for i,a in enumerate(ART,1):
    o=get_json(f'{DETAIL}?articleCode={a["code"]}');d=(o.get('data') or {})
    root=json.loads(d.get('body','{}')) if isinstance(d.get('body'),str) else (d.get('body') or {})
    ts,rows,txt=parse_loanable(root)
    rec={'code':a['code'],'title':d.get('title',a['title']),'release_time':int(d.get('publishDate') or a['release_time']),'tokens':ts,'rows':rows[:12]}
    parsed.append(rec)
    for t in ts:events.append({'article_code':rec['code'],'article_title':rec['title'],'release_time':rec['release_time'],'ticker':t,'symbol':t+'USDT'})
    print('PARSE',i,'/',len(ART),'TOKENS',len(ts),ts,flush=True)
    time.sleep(.08)
Path('/tmp/binance_loanable_articles.json').write_text(json.dumps(parsed,ensure_ascii=False,indent=2))
Path('/tmp/binance_loanable_events.json').write_text(json.dumps(events,ensure_ascii=False,indent=2))
from collections import Counter
print('EVENTS',len(events),'UNIQUE',len(set(x['symbol'] for x in events)),'ZERO',sum(not x['tokens'] for x in parsed))
print('YEARS',Counter(datetime.datetime.fromtimestamp(x['release_time']/1000,datetime.timezone.utc).year for x in events))
