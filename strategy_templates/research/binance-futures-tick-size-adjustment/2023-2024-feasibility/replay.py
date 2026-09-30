import csv,io,zipfile,urllib.request,urllib.error,time,datetime,math
from functools import lru_cache
UTC=datetime.timezone.utc
BASE='https://data.binance.vision/data/futures/um/monthly/klines'
UA='Mozilla/5.0'
E=[]
def add(ts,side,syms,tag):
    for s in syms:E.append((s,datetime.datetime.fromisoformat(ts),side,tag))
add('2023-08-29T06:30:00+00:00',1,['AMBUSDT','HOOKUSDT','API3USDT','TRBUSDT','DODOXUSDT'],'2023-08-29 finer')
add('2024-03-19T06:30:00+00:00',1,['STEEMUSDT','ONGUSDT','AUCTIONUSDT','RADUSDT','RDNTUSDT','VETUSDT','CFXUSDT','RONINUSDT','AEVOUSDT'],'2024-03-19 finer')
add('2024-04-11T06:30:00+00:00',1,['WUSDT','IDUSDT','SSVUSDT','FXSUSDT','GLMRUSDT','GMXUSDT','NMRUSDT'],'2024-04-11 finer')
add('2024-05-08T06:30:00+00:00',1,['ENAUSDT','TNSRUSDT','AEVOUSDT','OMNIUSDT'],'2024-05-08 finer')
add('2024-10-14T06:30:00+00:00',-1,['SOLUSDT'],'2024-10-14 coarser')
add('2024-11-28T06:30:00+00:00',1,['IOUSDT','PYTHUSDT','XAIUSDT','WUSDT','PIXELUSDT','FIDAUSDT','BBUSDT'],'2024-11-28 finer')
def mon(d):return d.strftime('%Y-%m')
def prevmon(d):return f'{d.year-1}-12' if d.month==1 else f'{d.year:04d}-{d.month-1:02d}'
def sub2(d):
    try:return d.replace(year=d.year-2)
    except:return d.replace(year=d.year-2,day=28)
@lru_cache(None)
def rows(sym,m):
    u=f'{BASE}/{sym}/1h/{sym}-1h-{m}.zip'
    for k in range(4):
        try:
            bb=urllib.request.urlopen(urllib.request.Request(u,headers={'User-Agent':UA}),timeout=20).read()
            z=zipfile.ZipFile(io.BytesIO(bb));fn=z.namelist()[0];out=[]
            for r in csv.reader(io.TextIOWrapper(z.open(fn))):
                if not r or not r[0].isdigit():continue
                t=int(r[0]);t=t//1000 if t>10**15 else t
                out.append((t,float(r[1]),float(r[4]),float(r[7]) if len(r)>7 else 0))
            return out
        except urllib.error.HTTPError as e:
            if e.code==404:return []
            time.sleep(.4*(k+1))
        except Exception:time.sleep(.4*(k+1))
    return []
good=[]
for sym,e,d,tag in E:
    h=rows(sym,mon(sub2(e))); cutoff=int(sub2(e).timestamp()*1000)
    if not h or h[0][0]>cutoff:print('DROP_AGE',sym);continue
    ems=int(e.timestamp()*1000);hour=(ems//3600000)*3600000
    data=rows(sym,prevmon(e))+rows(sym,mon(e));prior=[r for r in data if hour-86400000<=r[0]<hour]
    q=sum(r[3] for r in prior)
    if len(prior)<24 or q<5_000_000:print('DROP_LIQ',sym,len(prior),round(q));continue
    fut=[r for r in rows(sym,mon(e)) if r[0]>=ems]
    if len(fut)<12:print('DROP_FWD',sym);continue
    en=fut[0][1]; r1=d*math.log(fut[0][2]/en);r4=d*math.log(fut[3][2]/en);r12=d*math.log(fut[11][2]/en)
    good.append((sym,tag,r1,r4,r12,q));print('OK',sym,tag,'r12%',round(r12*100,3),flush=True)
print('N',len(good))
for name,i in [('R1',2),('R4',3),('R12',4)]:
    v=[x[i] for x in good]
    print(name,'mean%',sum(v)/len(v)*100 if v else None,'win',sum(x>0 for x in v)/len(v) if v else None)
from collections import defaultdict
g=defaultdict(list)
for x in good:g[x[1]].append(x[4])
print('BATCHES')
for k,v in g.items():print(k,len(v),round(sum(v)/len(v)*100,3),sum(x>0 for x in v))
