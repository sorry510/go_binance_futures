import json,subprocess,re,html
from pathlib import Path
from concurrent.futures import ThreadPoolExecutor,as_completed
x=json.load(open('/tmp/bitget_2025_section_items.json'))
spot=x['spot'];CACHE=Path('/tmp/bitget_spot2025_cache');CACHE.mkdir(exist_ok=True)
def fetch(a):
    aid=a['href'].split('/')[-1];p=CACHE/(aid+'.html')
    if not p.exists() or p.stat().st_size<1000:
        u='https://www.bitget.com'+a['href']
        q=subprocess.run(['curl','--http1.1','-L','--retry','2','--retry-all-errors','--connect-timeout','8','--max-time','25','-sS','-o',str(p),u])
        if q.returncode:raise RuntimeError((a['href'],q.returncode))
    s=p.read_text(errors='ignore');art=None
    for raw in re.findall(r'<script[^>]+type="application/ld\+json"[^>]*>(.*?)</script>',s,re.S|re.I):
        try:o=json.loads(html.unescape(raw))
        except:continue
        for z in o.get('@graph',[]) if isinstance(o,dict) else []:
            if isinstance(z,dict) and z.get('@type')=='Article':art=z;break
        if art:break
    if not art:return {**a,'parse_error':'no_article'}
    return {**a,'headline':art.get('headline',a['title']),'description':art.get('description',''),'datePublished':art.get('datePublished')}
out=[]
with ThreadPoolExecutor(max_workers=4) as ex:
    fs={ex.submit(fetch,a):a for a in spot}
    for i,f in enumerate(as_completed(fs),1):
        try:out.append(f.result())
        except Exception as e:out.append({**fs[f],'parse_error':str(e)})
        if i%10==0 or i==len(fs):print('FETCH',i,'/',len(fs),flush=True)
out.sort(key=lambda z:z.get('datePublished') or z.get('section_date') or '')
Path('/tmp/bitget_2025_spot_articles.json').write_text(json.dumps(out,ensure_ascii=False,indent=2))
print('errors',sum('parse_error' in a for a in out))
for a in out[:8]:print(a.get('datePublished'),a['headline'],'|',a.get('description','')[:300])
