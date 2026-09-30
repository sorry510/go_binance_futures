import json,urllib.request,urllib.error,zipfile,io,csv,datetime,math
from functools import lru_cache
UTC=datetime.timezone.utc;UA='Mozilla/5.0'
src=json.load(open('/tmp/usdc_perp_launch_eligibility.json'))['eligible']
VISION='https://data.binance.vision/data/futures/um/monthly/klines'
def month(ms):return datetime.datetime.fromtimestamp(ms/1000,UTC).strftime('%Y-%m')
def nextmonth(ms):
    d=datetime.datetime.fromtimestamp(ms/1000,UTC)
    return f'{d.year+1}-01' if d.month==12 else f'{d.year:04d}-{d.month+1:02d}'
@lru_cache(None)
def rows(sym,mo):
    u=f'{VISION}/{sym}/1h/{sym}-1h-{mo}.zip'
    try:b=urllib.request.urlopen(urllib.request.Request(u,headers={'User-Agent':UA}),timeout=20).read()
    except Exception:return []
    z=zipfile.ZipFile(io.BytesIO(b));fn=z.namelist()[0];out=[]
    for r in csv.reader(io.TextIOWrapper(z.open(fn))):
        if not r or not r[0].isdigit():continue
        t=int(r[0]);t=t//1000 if t>10**15 else t
        out.append((t,float(r[1]),float(r[4])))
    return out
res=[]
for e in src:
    sym=e['usdt_symbol'];evt=e['launch_ms'];data=rows(sym,month(evt))+rows(sym,nextmonth(evt))
    f=[r for r in data if r[0]>=evt][:13]
    if len(f)<12 or f[0][0]!=evt:
        print('DROP_FWD',e['base'],len(f),f[0][0] if f else None);continue
    ent=f[0][1]
    x=dict(e);x['r1']=-math.log(f[0][2]/ent);x['r4']=-math.log(f[3][2]/ent);x['r12']=-math.log(f[11][2]/ent)
    res.append(x)
    print('EV',e['base'],'r12%',round(x['r12']*100,3))
def stats(a,k):
    v=[x[k] for x in a];return {'n':len(v),'mean':sum(v)/len(v) if v else None,'win':sum(z>0 for z in v)/len(v) if v else None}
print('SYMBOL_LEVEL')
for k in ('r1','r4','r12'):print(k,stats(res,k))
batches={}
for x in res:batches.setdefault(x['launch_ms'],[]).append(x)
br=[]
for t,q in batches.items():
    br.append({'t':t,'r1':sum(x['r1'] for x in q)/len(q),'r4':sum(x['r4'] for x in q)/len(q),'r12':sum(x['r12'] for x in q)/len(q),'n':len(q)})
print('BATCHES',len(br))
for k in ('r1','r4','r12'):print('BATCH',k,stats(br,k))
for x in br:
    print('BATCHROW',datetime.datetime.fromtimestamp(x['t']/1000,UTC).isoformat(),'n',x['n'],'r12%',round(x['r12']*100,3))
out={'symbol_level':{k:stats(res,k) for k in ('r1','r4','r12')},'batch_level':{k:stats(br,k) for k in ('r1','r4','r12')},'batches':br,'events':res}
open('/tmp/usdc_perp_launch_diagnostic.json','w').write(json.dumps(out,indent=2))
