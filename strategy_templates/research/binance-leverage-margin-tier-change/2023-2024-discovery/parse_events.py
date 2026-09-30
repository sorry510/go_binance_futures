import json,re,urllib.request,urllib.error,time,datetime,html
from concurrent.futures import ThreadPoolExecutor,as_completed
from pathlib import Path
UA='Mozilla/5.0'
arts=json.load(open('/tmp/binance_leverage_tier_articles.json'))
cache=Path('/tmp/binance_leverage_details');cache.mkdir(exist_ok=True)

def fetch(a):
    f=cache/(a['code']+'.json')
    if f.exists(): return json.loads(f.read_text())
    u='https://www.binance.com/bapi/composite/v1/public/cms/article/detail/query?articleCode='+a['code']
    for k in range(5):
        try:
            req=urllib.request.Request(u,headers={'User-Agent':UA,'Accept':'application/json','clienttype':'web','lang':'en'})
            o=json.loads(urllib.request.urlopen(req,timeout=20).read())
            f.write_text(json.dumps(o,ensure_ascii=False))
            return o
        except Exception as e:
            if k==4: raise
            time.sleep(.5*(k+1))

def txt(n):
    if not isinstance(n,dict): return ''
    if n.get('node')=='text': return html.unescape(str(n.get('text','')).replace('&nbsp;',' '))
    return ''.join(txt(x) for x in n.get('child',[]) or [])

def table_rows(n):
    out=[]
    def walk(x):
        if not isinstance(x,dict):return
        if x.get('tag')=='tr':
            cells=[]
            for c in x.get('child',[]) or []:
                if isinstance(c,dict) and c.get('tag') in ('td','th'):
                    cells.append(' '.join(txt(c).split()))
            if cells:out.append(cells)
        else:
            for c in x.get('child',[]) or []:walk(c)
    walk(n);return out

def leverage_max(s):
    vals=[int(x) for x in re.findall(r'(?<![\d.])(\d{1,3})(?=\s*x|\s*-)',s,re.I)]
    if not vals:
        m=re.search(r'(\d{1,3})\s*x',s,re.I)
        if m:vals=[int(m.group(1))]
    return max(vals) if vals else None
def parse(a,o):
    d=(o.get('data') or {})
    body=d.get('body') or ''
    try:root=json.loads(body)
    except Exception:return [],'body_not_json'
    full=' '.join(txt(root).split())
    # Exact effective timestamp in UTC. Prefer explicit update time in body, not releaseDate.
    tm=None
    for pat in [r'at\s+(20\d\d-\d\d-\d\d)\s+(\d\d:\d\d)(?::\d\d)?\s*\(UTC\)',
                r'on\s+(20\d\d-\d\d-\d\d)\s+at\s+(\d\d:\d\d)(?::\d\d)?\s*\(UTC\)']:
        m=re.search(pat,full,re.I)
        if m:
            tm=int(datetime.datetime.fromisoformat(m.group(1)+'T'+m.group(2)+':00+00:00').timestamp()*1000);break
    if tm is None:return [],'no_effective_time'
    out=[];children=root.get('child',[]) or [];current=None
    for node in children:
        t=' '.join(txt(node).split())
        # A list item or paragraph immediately preceding a table commonly names the contract.
        syms=re.findall(r'(?<![A-Z0-9])(1000)?([A-Z0-9]{2,20})USDT\s*\(USD[^)]*M\s+Perpetual Contract\)',t)
        if syms:
            raw=(syms[-1][0] or '')+syms[-1][1]+'USDT';current=raw
        if isinstance(node,dict) and node.get('tag')=='table' and current:
            rows=table_rows(node)
            data=None
            for r in rows:
                if len(r)>=6 and any('x' in z.lower() for z in (r[0],r[3])):
                    old=leverage_max(r[0]);new=leverage_max(r[3])
                    if old is not None and new is not None:
                        data=(old,new,r);break
            if data:
                old,new,row=data
                cls='loosening' if new>old else 'tightening' if new<old else 'unchanged'
                out.append({'article_code':a['code'],'article_title':a['title'],'release_time':a['releaseDate'],
                            'effective_time':tm,'symbol':current,'old_first_max_leverage':old,
                            'new_first_max_leverage':new,'classification':cls,'first_row':row})
            current=None
    return out,None

with ThreadPoolExecutor(max_workers=6) as ex:
    fs={ex.submit(fetch,a):a for a in arts}
    details={}
    for i,f in enumerate(as_completed(fs),1):
        a=fs[f]
        try:details[a['code']]=f.result()
        except Exception as e:print('FETCH_ERR',a['code'],e)
        if i%10==0:print('FETCH',i,'/',len(fs),flush=True)

events=[];errors={}
for a in arts:
    if a['code'] not in details:continue
    ev,err=parse(a,details[a['code']]);events.extend(ev)
    if err:errors[a['code']]=err

Path('/tmp/binance_leverage_events_parsed.json').write_text(json.dumps(events,ensure_ascii=False,indent=2))
from collections import Counter
print('ARTICLES',len(arts),'EVENTS',len(events),'UNIQUE_SYMBOLS',len(set(x['symbol'] for x in events)))
print('CLASS',Counter(x['classification'] for x in events),'PARSE_ERRORS',Counter(errors.values()))
for x in events[:40]:
    dt=datetime.datetime.fromtimestamp(x['effective_time']/1000,datetime.timezone.utc)
    print(dt.isoformat(),x['symbol'],x['old_first_max_leverage'],'->',x['new_first_max_leverage'],x['classification'])
