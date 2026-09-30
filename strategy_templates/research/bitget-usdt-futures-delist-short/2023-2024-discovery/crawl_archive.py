import subprocess,re,html,json,datetime,time
from pathlib import Path
BASE='https://www.bitget.com'
def get(u):
    p=subprocess.run(['curl','--http1.1','-L','--retry','3','--retry-all-errors',
                      '--connect-timeout','8','--max-time','30','-sS',u],
                     capture_output=True,text=True)
    if p.returncode: raise RuntimeError((u,p.returncode,p.stderr[-200:]))
    return p.stdout
items={}
pat=re.compile(r'<a[^>]+href="([^"]*/support/articles/\d+[^"]*)"[^>]*>(.*?)</a>',re.S|re.I)
for page in range(16,25):
    s=get(f'{BASE}/support/sections/12508313443290/{page}')
    for href,b in pat.findall(s):
        title=html.unescape(' '.join(re.sub('<[^>]+>',' ',b).split()))
        href=href.split('?')[0]
        items[href]=title
    print('PAGE',page,'total',len(items),flush=True)
cand=[]
for href,title in items.items():
    lo=title.lower()
    if 'delist' not in lo: continue
    if not any(k in lo for k in ['future','perpetual']): continue
    if any(k in lo for k in ['postpone','postponement','delay the delisting']): continue
    cand.append((href,title))
print('CANDIDATE_LINKS',len(cand),flush=True)
out=[]
for i,(href,title) in enumerate(cand,1):
    s=get(BASE+href)
    raws=re.findall(r'<script[^>]+type="application/ld\+json"[^>]*>(.*?)</script>',s,re.S|re.I)
    art=None
    for raw in raws:
        try:o=json.loads(html.unescape(raw))
        except:continue
        for x in o.get('@graph',[]) if isinstance(o,dict) else []:
            if isinstance(x,dict) and x.get('@type')=='Article': art=x;break
        if art:break
    if not art: continue
    dt=art.get('datePublished'); desc=art.get('description','')
    out.append({'href':href,'title':title,'datePublished':dt,'description':desc})
    print(i,dt,title,flush=True)
Path('/tmp/bitget_futures_delist_candidates.json').write_text(json.dumps(out,ensure_ascii=False,indent=2))
print('DONE',len(out))
